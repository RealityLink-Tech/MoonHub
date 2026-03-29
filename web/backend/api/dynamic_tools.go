package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/RealityLink-Tech/MoonHub/pkg/config"
	"github.com/RealityLink-Tech/MoonHub/pkg/dynamictools"
	openai_compat "github.com/RealityLink-Tech/MoonHub/pkg/providers/openai_compat"
	"github.com/RealityLink-Tech/MoonHub/pkg/providers/protocoltypes"
)

// DynamicToolsHandler handles dynamic tools API requests.
type DynamicToolsHandler struct {
	toolManager  *dynamictools.ToolManager
	schemaEngine *dynamictools.SchemaEngine
	configPath   string
}

// NewDynamicToolsHandler creates a DynamicToolsHandler with a SQLite-backed
// ToolManager and a SchemaEngine. The database is stored at
// <moonhub-home>/dynamic_tools.db.
func NewDynamicToolsHandler(configPath string) (*DynamicToolsHandler, error) {
	homeDir := moonhubHomeDir()
	dbPath := filepath.Join(homeDir, "dynamic_tools.db")

	tm, err := dynamictools.NewToolManager(dbPath)
	if err != nil {
		return nil, fmt.Errorf("create tool manager: %w", err)
	}

	hostFn := dynamictools.NewHostFunctions()
	se := dynamictools.NewSchemaEngine(hostFn)

	return &DynamicToolsHandler{
		toolManager:  tm,
		schemaEngine: se,
		configPath:   configPath,
	}, nil
}

// Close releases the underlying database connection.
func (h *DynamicToolsHandler) Close() {
	if h.toolManager != nil {
		_ = h.toolManager.Close()
	}
}

// RegisterRoutes registers dynamic tools routes on the given ServeMux.
func (h *DynamicToolsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dynamic-tools", h.handleList)
	mux.HandleFunc("POST /api/dynamic-tools/generate", h.handleGenerate)
	mux.HandleFunc("POST /api/dynamic-tools/{id}/execute", h.handleExecute)
	mux.HandleFunc("GET /api/dynamic-tools/{id}/schema", h.handleGetSchema)
	mux.HandleFunc("DELETE /api/dynamic-tools/{id}", h.handleDelete)
	mux.HandleFunc("PATCH /api/dynamic-tools/{id}/home", h.handleSetOnHome)
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// handleList handles GET /api/dynamic-tools?source=ai
func (h *DynamicToolsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	source := r.URL.Query().Get("source")
	tools, err := h.toolManager.List(r.Context(), source)
	if err != nil {
		writeDynamicError(w, http.StatusInternalServerError, "failed to list tools: "+err.Error())
		return
	}

	if tools == nil {
		tools = []*dynamictools.DynamicTool{}
	}
	writeDynamicSuccess(w, tools)
}

// handleGenerate handles POST /api/dynamic-tools/generate
func (h *DynamicToolsHandler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	var req dynamictools.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeDynamicError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Prompt == "" {
		writeDynamicError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	category := req.Context
	if category == "" {
		category = inferCategory(req.Prompt)
	}
	if category != "chat" && category != "space" {
		category = "chat"
	}

	hash := dynamictools.ComputeContentHash(category, req.Prompt)

	// Dedup: if a tool with the same hash exists, return it directly.
	existing, err := h.toolManager.FindByHash(r.Context(), hash)
	if err != nil {
		writeDynamicError(w, http.StatusInternalServerError, "failed to check existing tool: "+err.Error())
		return
	}
	if existing != nil {
		writeDynamicSuccess(w, dynamictools.GenerateResult{
			ToolID:      existing.ID,
			Name:        existing.Name,
			IsNew:       false,
			ChatSchema:  existing.ChatSchema,
			SpaceSchema: existing.SpaceSchema,
		})
		return
	}

	// Generate with LLM.
	tool, err := h.generateWithLLM(r.Context(), req.Prompt, category, hash)
	if err != nil {
		writeDynamicError(w, http.StatusInternalServerError, "LLM generation failed: "+err.Error())
		return
	}

	if err := h.toolManager.Insert(r.Context(), tool); err != nil {
		writeDynamicError(w, http.StatusInternalServerError, "failed to save tool: "+err.Error())
		return
	}

	writeDynamicSuccess(w, dynamictools.GenerateResult{
		ToolID:      tool.ID,
		Name:        tool.Name,
		IsNew:       true,
		ChatSchema:  tool.ChatSchema,
		SpaceSchema: tool.SpaceSchema,
	})
}

// handleExecute handles POST /api/dynamic-tools/{id}/execute
func (h *DynamicToolsHandler) handleExecute(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeDynamicError(w, http.StatusBadRequest, "id is required")
		return
	}

	tool, err := h.toolManager.GetByID(r.Context(), id)
	if err != nil {
		writeDynamicError(w, http.StatusInternalServerError, "failed to get tool: "+err.Error())
		return
	}
	if tool == nil {
		writeDynamicError(w, http.StatusNotFound, "tool not found")
		return
	}

	var params map[string]any
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			writeDynamicError(w, http.StatusBadRequest, "invalid params: "+err.Error())
			return
		}
	}
	if params == nil {
		params = make(map[string]any)
	}

	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "chat"
	}

	var result *dynamictools.ExecutionResult
	switch mode {
	case "space":
		result, err = h.schemaEngine.ExecuteSpace(r.Context(), tool, params)
	default:
		result, err = h.schemaEngine.Execute(r.Context(), tool, params)
	}
	if err != nil {
		writeDynamicError(w, http.StatusInternalServerError, "execution failed: "+err.Error())
		return
	}

	writeDynamicSuccess(w, result)
}

// handleGetSchema handles GET /api/dynamic-tools/{id}/schema?mode=chat|space
func (h *DynamicToolsHandler) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeDynamicError(w, http.StatusBadRequest, "id is required")
		return
	}

	tool, err := h.toolManager.GetByID(r.Context(), id)
	if err != nil {
		writeDynamicError(w, http.StatusInternalServerError, "failed to get tool: "+err.Error())
		return
	}
	if tool == nil {
		writeDynamicError(w, http.StatusNotFound, "tool not found")
		return
	}

	mode := r.URL.Query().Get("mode")
	switch mode {
	case "space":
		writeDynamicSuccess(w, tool.SpaceSchema)
	default:
		writeDynamicSuccess(w, tool.ChatSchema)
	}
}

// handleDelete handles DELETE /api/dynamic-tools/{id}
func (h *DynamicToolsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeDynamicError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.toolManager.Delete(r.Context(), id); err != nil {
		writeDynamicError(w, http.StatusInternalServerError, "failed to delete tool: "+err.Error())
		return
	}

	writeDynamicSuccess(w, map[string]string{"id": id})
}

// handleSetOnHome handles PATCH /api/dynamic-tools/{id}/home
func (h *DynamicToolsHandler) handleSetOnHome(w http.ResponseWriter, r *http.Request) {
	if !requireLANClient(w, r) {
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeDynamicError(w, http.StatusBadRequest, "id is required")
		return
	}

	var body struct {
		OnHome bool `json:"on_home"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeDynamicError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	if err := h.toolManager.SetOnHome(r.Context(), id, body.OnHome); err != nil {
		writeDynamicError(w, http.StatusInternalServerError, "failed to update: "+err.Error())
		return
	}

	writeDynamicSuccess(w, map[string]any{"id": id, "is_on_home": body.OnHome})
}

// ---------------------------------------------------------------------------
// LLM generation
// ---------------------------------------------------------------------------

type providerCreds struct {
	apiKey  string
	apiBase string
	model   string
	proxy   string
}

// generateWithLLM calls the configured LLM to generate a DynamicTool from a
// user prompt.
func (h *DynamicToolsHandler) generateWithLLM(ctx context.Context, prompt, category, hash string) (*dynamictools.DynamicTool, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	creds := resolveProvider(cfg)
	if creds.apiKey == "" || creds.apiBase == "" {
		return nil, fmt.Errorf("no LLM provider configured (set api_key and api_base in config)")
	}

	provider := openai_compat.NewProvider(creds.apiKey, creds.apiBase, creds.proxy)

	messages := []protocoltypes.Message{
		{Role: "system", Content: buildGenerateSystemPrompt()},
		{Role: "user", Content: prompt},
	}

	resp, err := provider.Chat(ctx, messages, nil, creds.model, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM chat: %w", err)
	}

	return parseGenerateResponse(resp.Content, category, hash)
}

// resolveProvider extracts LLM credentials from the config.
// It first checks model_list for the default model, then falls back to the
// legacy providers section.
func resolveProvider(cfg *config.Config) providerCreds {
	// Try model_list using the default model name.
	modelName := cfg.Agents.Defaults.GetModelName()
	if modelName != "" {
		if mc, err := cfg.GetModelConfig(modelName); err == nil {
			if mc.APIKey != "" && mc.APIBase != "" {
				return providerCreds{
					apiKey:  mc.APIKey,
					apiBase: mc.APIBase,
					model:   mc.Model,
					proxy:   mc.Proxy,
				}
			}
		}
	}

	// Fallback: use the first model in model_list that has an API key.
	for _, mc := range cfg.ModelList {
		if mc.APIKey != "" && mc.APIBase != "" {
			return providerCreds{
				apiKey:  mc.APIKey,
				apiBase: mc.APIBase,
				model:   mc.Model,
				proxy:   mc.Proxy,
			}
		}
	}

	// Legacy fallback: use the first provider with a non-empty API key.
	apiKey := cfg.GetAPIKey()
	apiBase := cfg.GetAPIBase()
	if apiKey != "" && apiBase != "" {
		providerName := cfg.Agents.Defaults.Provider
		return providerCreds{
			apiKey:  apiKey,
			apiBase: apiBase,
			model:   providerName + "/" + modelName,
		}
	}

	return providerCreds{}
}

// buildGenerateSystemPrompt returns the system prompt that instructs the LLM
// how to generate dynamic tool schemas.
func buildGenerateSystemPrompt() string {
	return `You are a UI component generator for MoonHub. Given a user description, generate TWO JSON schemas:

1. chat_schema: A compact component tree for chat message display.
2. space_schema: A full-width component tree for dashboard/space display.

Available component types:
CHAT: text, heading, badge, icon, image, progress, stat_card, action_button, divider
SPACE: stat_card, chart_bar, chart_line, chart_pie, list, grid, card, timeline, counter, status_indicator, markdown_block, iframe

IMPORTANT RULES:
- Return ONLY valid JSON with no explanation or markdown wrappers.
- Each schema must have: {"id":"root","type":"<component_type>","props":{...},"children":[...]}
- Use "props.title", "props.value", "props.subtitle" for data display.
- For charts, include "props.data" as an array of objects.
- Use real, realistic placeholder data so the component is immediately useful.
- chat_schema should be compact (1-2 components).
- space_schema can be richer (3-5 components with nested children).

RESPONSE FORMAT (JSON):
{
  "name": "<short english name>",
  "description": "<english description>",
  "chat_schema": { "id": "root", "type": "stat_card", "props": {...}, "children": [] },
  "space_schema": { "id": "root", "type": "grid", "props": {...}, "children": [...] }
}`
}

// parseGenerateResponse parses the LLM JSON output into a DynamicTool.
func parseGenerateResponse(content, category, hash string) (*dynamictools.DynamicTool, error) {
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in LLM response")
	}

	var raw struct {
		Name        string                    `json:"name"`
		Description string                    `json:"description"`
		ChatSchema  dynamictools.GeneratedComponent `json:"chat_schema"`
		SpaceSchema dynamictools.GeneratedComponent `json:"space_schema"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w", err)
	}

	if raw.ChatSchema.Type == "" {
		raw.ChatSchema = dynamictools.GeneratedComponent{ID: "root", Type: "text", Props: map[string]any{"text": content}}
	}
	if raw.SpaceSchema.Type == "" {
		raw.SpaceSchema = dynamictools.GeneratedComponent{ID: "root", Type: "card", Props: map[string]any{"title": raw.Name}, Children: []dynamictools.GeneratedComponent{}}
	}
	if raw.Name == "" {
		raw.Name = "untitled"
	}

	return &dynamictools.DynamicTool{
		ID:           dynamictools.GenerateID(category),
		Name:         raw.Name,
		Description:  raw.Description,
		Category:     category,
		ChatSchema:   raw.ChatSchema,
		SpaceSchema:  raw.SpaceSchema,
		Engine:       "schema",
		ContentHash:  hash,
		IsAIGenerated: true,
	}, nil
}

// extractJSON attempts to extract a JSON object from a string that may be
// wrapped in markdown code blocks (```json ... ```).
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Try to extract from ```json ... ``` or ``` ... ```
	if idx := strings.Index(s, "```"); idx != -1 {
		after := s[idx+3:]
		if idx2 := strings.Index(after, "\n"); idx2 != -1 {
			after = after[idx2+1:]
		}
		if end := strings.Index(after, "```"); end != -1 {
			return strings.TrimSpace(after[:end])
		}
	}

	// Try to find the first { and last } to extract the JSON object.
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(s, "}")
	if end <= start {
		return ""
	}
	return s[start : end+1]
}

// inferCategory detects whether the prompt is more suited for a "space" or
// "chat" context based on keyword matching.
func inferCategory(prompt string) string {
	lower := strings.ToLower(prompt)
	spaceKeywords := []string{
		"dashboard", "chart", "graph", "table", "analytics", "statistics",
		"monitor", "overview", "report", "panel", "widget", "仪表盘",
		"图表", "统计", "监控", "面板", "看板",
	}
	for _, kw := range spaceKeywords {
		if strings.Contains(lower, kw) {
			return "space"
		}
	}
	return "chat"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// moonhubHomeDir returns the MoonHub home directory, respecting MOONHUB_HOME
// and falling back to ~/.moonhub.
func moonhubHomeDir() string {
	if dir := os.Getenv("MOONHUB_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".moonhub"
	}
	return filepath.Join(home, ".moonhub")
}

func writeDynamicSuccess(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
}

func writeDynamicError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"success": false, "error": msg})
}
