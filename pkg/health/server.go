package health

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/routing"
)

type Server struct {
	server    *http.Server
	mu        sync.RWMutex
	ready     bool
	checks    map[string]Check
	startTime time.Time
}

type Check struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type StatusResponse struct {
	Status string           `json:"status"`
	Uptime string           `json:"uptime"`
	Checks map[string]Check `json:"checks,omitempty"`
}

func NewServer(host string, port int) *Server {
	mux := http.NewServeMux()
	s := &Server{
		ready:     false,
		checks:    make(map[string]Check),
		startTime: time.Now(),
	}

	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.HandleFunc("/routing/decisions", s.routingDecisionsHandler)
	mux.HandleFunc("/routing/stats", s.routingStatsHandler)

	addr := fmt.Sprintf("%s:%d", host, port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()
	return s.server.ListenAndServe()
}

func (s *Server) StartContext(ctx context.Context) error {
	s.mu.Lock()
	s.ready = true
	s.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.server.Shutdown(context.Background())
	}
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	s.ready = false
	s.mu.Unlock()
	return s.server.Shutdown(ctx)
}

func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	s.ready = ready
	s.mu.Unlock()
}

func (s *Server) RegisterCheck(name string, checkFn func() (bool, string)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, msg := checkFn()
	s.checks[name] = Check{
		Name:      name,
		Status:    statusString(status),
		Message:   msg,
		Timestamp: time.Now(),
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	uptime := time.Since(s.startTime)
	resp := StatusResponse{
		Status: "ok",
		Uptime: uptime.String(),
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	ready := s.ready
	checks := make(map[string]Check)
	maps.Copy(checks, s.checks)
	s.mu.RUnlock()

	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(StatusResponse{
			Status: "not ready",
			Checks: checks,
		})
		return
	}

	for _, check := range checks {
		if check.Status == "fail" {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(StatusResponse{
				Status: "not ready",
				Checks: checks,
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	uptime := time.Since(s.startTime)
	json.NewEncoder(w).Encode(StatusResponse{
		Status: "ready",
		Uptime: uptime.String(),
		Checks: checks,
	})
}

// RegisterOnMux registers /health and /ready handlers onto the given mux.
// This allows the health endpoints to be served by a shared HTTP server.
func (s *Server) RegisterOnMux(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.HandleFunc("/routing/decisions", s.routingDecisionsHandler)
	mux.HandleFunc("/routing/stats", s.routingStatsHandler)
}

// metricsHandler returns routing metrics in JSON format.
// Privacy: Only returns aggregated statistics, no user content.
func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metrics := routing.GetGlobalMetrics()
	snapshot := metrics.GetSnapshot()

	// Convert to JSON-friendly format
	response := map[string]any{
		"timestamp":               snapshot.Timestamp,
		"total_classifications":   snapshot.TotalClassifications,
		"avg_score":               snapshot.AvgScore,
		"avg_confidence":          snapshot.AvgConfidence,
		"tier_counts":             snapshot.TierCounts,
		"signal_counts":           snapshot.SignalCounts,
		"score_distribution":      snapshot.ScoreDistribution,
		"confidence_distribution": snapshot.ConfidenceDistribution,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// routingDecisionsHandler returns recent routing decisions for visualization.
// Privacy: Returns only statistical data, no user message content.
func (s *Server) routingDecisionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	recorder := routing.GetGlobalRecorder()

	// Parse query parameters
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	tier := r.URL.Query().Get("tier")
	var decisions []routing.DecisionRecord

	if tier != "" {
		parsed, ok := routing.TryParseTier(tier)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid tier (use simple, moderate, complex, or reasoning)",
			})
			return
		}
		decisions = recorder.GetByTier(parsed, limit)
	} else {
		decisions = recorder.GetRecent(limit)
	}

	response := map[string]any{
		"total":     recorder.Count(),
		"limit":     limit,
		"decisions": decisions,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// routingStatsHandler returns aggregated statistics about routing decisions.
func (s *Server) routingStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	recorder := routing.GetGlobalRecorder()
	stats := recorder.GetStats()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

func parseIntParam(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func statusString(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}
