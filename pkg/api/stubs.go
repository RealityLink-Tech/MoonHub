package api

import (
	"net/http"

	"github.com/RealityLink-Tech/MoonHub/pkg/agent"
)

// Minimal stub -- full implementation in Task 2
type ChatHub struct{}

func newChatHub(agent *agent.AgentLoop) *ChatHub {
	return &ChatHub{}
}

func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handlePatchConfig(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) handleListModels(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) handleListTools(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) handleChannelCatalog(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *Handler) handleChatWS(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
