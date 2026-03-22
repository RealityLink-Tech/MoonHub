package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/shield"
)

type panicShieldEvaluator struct{}

func (panicShieldEvaluator) IsActive() bool { return true }

func (panicShieldEvaluator) Evaluate(ShieldEvent) ShieldDecision {
	panic("shield Evaluate should not run when execution was pre-approved")
}

func TestWebFetchTool_SkipsShieldWhenPreApproved(t *testing.T) {
	allowPrivateWebFetchHosts.Store(true)
	t.Cleanup(func() { allowPrivateWebFetchHosts.Store(false) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body><p>ok</p></body></html>"))
	}))
	t.Cleanup(srv.Close)

	tool, err := NewWebFetchTool(50000, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	tool = tool.WithShield(panicShieldEvaluator{})

	ctx := shield.ContextWithApprovedToolExecution(context.Background())
	res := tool.Execute(ctx, map[string]any{"url": srv.URL})
	if res == nil || res.IsError {
		t.Fatalf("expected success, got %#v", res)
	}
	if res.ForLLM == "" {
		t.Fatal("expected non-empty ForLLM")
	}
}
