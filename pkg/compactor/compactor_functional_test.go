// Functional tests for the compaction pipeline (no agent loop).

package compactor

import (
	"context"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
)

type stubSummarizeProvider struct {
	summary string
	model   string
}

func (s *stubSummarizeProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{Content: s.summary}, nil
}

func (s *stubSummarizeProvider) GetDefaultModel() string {
	if s.model != "" {
		return s.model
	}
	return "stub-model"
}

type panicChatProvider struct{}

func (p *panicChatProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	opts map[string]any,
) (*providers.LLMResponse, error) {
	panic("Chat must not be called")
}

func (p *panicChatProvider) GetDefaultModel() string { return "panic-model" }

func TestCompactIfNeeded_DisabledDoesNotCallLLM(t *testing.T) {
	t.Parallel()
	db := t.TempDir() + "/c.db"
	cfg := DefaultConfig()
	cfg.Enabled = false
	cfg.DBPath = db
	cfg.ParallelProcessing = false
	cfg.IncrementalCompaction = false

	c, err := New(cfg, &panicChatProvider{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	msgs := repeatRoleMessages("user", "hello world\n", 50)
	ctx := context.Background()
	res, err := c.CompactIfNeeded(ctx, "s1", msgs, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if res != nil {
		t.Fatalf("CompactIfNeeded with Enabled=false: got result, want nil")
	}
}

func TestCompactIfNeeded_EnabledBelowThresholdDoesNotCallLLM(t *testing.T) {
	t.Parallel()
	db := t.TempDir() + "/c.db"
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.DBPath = db
	cfg.TriggerTokenPercent = 95
	cfg.ParallelProcessing = false
	cfg.IncrementalCompaction = false

	c, err := New(cfg, &panicChatProvider{})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	msgs := []providers.Message{{Role: "user", Content: "short"}}
	ctx := context.Background()
	res, err := c.CompactIfNeeded(ctx, "s1", msgs, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if res != nil {
		t.Fatalf("below threshold: want nil, got %+v", res)
	}
}

func TestCompactIfNeeded_EnabledAboveThresholdRunsPipeline(t *testing.T) {
	t.Parallel()
	db := t.TempDir() + "/c.db"
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.DBPath = db
	cfg.TriggerTokenPercent = 1
	cfg.KeepRecent = 2
	cfg.ParallelProcessing = false
	cfg.IncrementalCompaction = false
	cfg.TierBudgets = TierBudgets{L0: 80, L1: 200, L2: 400}

	llmOut := `## User Identity
Name: Bob.

## Decisions
Chose SQLite.

## Tasks
Ship v1.

## Preferences
Concise answers.

## Context
Testing compactor end-to-end.
`
	prov := &stubSummarizeProvider{summary: llmOut}

	c, err := New(cfg, prov)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	msgs := repeatRoleMessages("user", "This is a test line for token estimate. ", 80)
	ctx := context.Background()
	res, err := c.CompactIfNeeded(ctx, "session-a", msgs, 500)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected compaction result")
	}
	if res.Summary.L2 == "" {
		t.Fatal("expected non-empty L2")
	}
	if res.MessagesKept != cfg.GetKeepRecentCount() {
		t.Fatalf("MessagesKept=%d want %d", res.MessagesKept, cfg.GetKeepRecentCount())
	}

	st, err := c.store.GetState(ctx, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	if st == nil {
		t.Fatal("expected compaction state")
	}
	if st.LastMessageIndex != 0 {
		t.Fatalf("LastMessageIndex=%d want 0 (tail-only session contract)", st.LastMessageIndex)
	}

	got, err := c.GetTieredSummary(ctx, "session-a")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.L2 == "" {
		t.Fatal("GetTieredSummary: expected stored L2")
	}
	if got.GetTier(2) == "" {
		t.Fatal("GetTier(2) empty")
	}
}

func TestForceCompact_IncrementalSecondRunUsesSliceFromZero(t *testing.T) {
	t.Parallel()
	db := t.TempDir() + "/c.db"
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.DBPath = db
	cfg.KeepRecent = 3
	cfg.ParallelProcessing = false
	cfg.IncrementalCompaction = true
	cfg.TierBudgets = TierBudgets{L0: 60, L1: 120, L2: 250}

	prov := &stubSummarizeProvider{summary: "## User Identity\nU.\n## Decisions\nD.\n## Tasks\nT.\n## Preferences\nP.\n## Context\nC.\n"}
	c, err := New(cfg, prov)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	first := repeatRoleMessages("user", "line x ", 20)
	_, err = c.ForceCompact(ctx, "inc", first)
	if err != nil {
		t.Fatal(err)
	}

	second := repeatRoleMessages("assistant", "reply y ", 25)
	_, err = c.ForceCompact(ctx, "inc", second)
	if err != nil {
		t.Fatal(err)
	}
	st, err := c.store.GetState(ctx, "inc")
	if err != nil || st == nil {
		t.Fatalf("state: err=%v st=%v", err, st)
	}
	if st.LastMessageIndex != 0 {
		t.Fatalf("after second compact LastMessageIndex=%d want 0", st.LastMessageIndex)
	}
}

func repeatRoleMessages(role, line string, n int) []providers.Message {
	out := make([]providers.Message, n)
	for i := range out {
		out[i] = providers.Message{Role: role, Content: line}
	}
	return out
}
