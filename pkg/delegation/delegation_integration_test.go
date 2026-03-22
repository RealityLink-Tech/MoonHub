package delegation

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/providers"
	"github.com/RealityLink-Tech/MoonHub/pkg/tools"
)

// routingTestProvider routes Chat by message shape: sub-agent system prompts contain
// "specialized sub-agent" (see LLMExecutor.Execute).
type routingTestProvider struct {
	mu            sync.Mutex
	TotalChats    int
	SubAgentChats int
	SubResponse   string
}

func (p *routingTestProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	_ []providers.ToolDefinition,
	model string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	p.TotalChats++
	sub := false
	for _, m := range messages {
		if m.Role == "system" && strings.Contains(m.Content, "specialized sub-agent") {
			sub = true
			break
		}
	}
	if sub {
		p.SubAgentChats++
	}
	p.mu.Unlock()

	content := "main-model"
	if sub {
		content = p.SubResponse
		if content == "" {
			content = "SUB_AGENT_OK"
		}
	}
	return &providers.LLMResponse{Content: content}, nil
}

func (p *routingTestProvider) GetDefaultModel() string {
	return "test-model"
}

func testDelegationConfig() DelegationConfig {
	return DelegationConfig{
		Enabled:              true,
		DefaultSubAgentTools: []string{"read_file"},
		MaxActivePerUser:     10,
		MaxConcurrentTasks:   3,
		RetentionDays:        7,
		ReuseThreshold:       0.6,
	}
}

func TestFunctional_delegate_task_runsSubAgentLLM(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	db := filepath.Join(tmp, "delegation.db")
	p := &routingTestProvider{SubResponse: "delegated-result-xyz"}
	ds, err := NewDelegationSystem(db, testDelegationConfig(), p, "m", tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer ds.Close()

	reg := tools.NewToolRegistry()
	ds.RegisterTools(reg)

	ctx := context.Background()
	ctx = tools.WithToolContext(ctx, "telegram", "room-1")
	ctx = tools.WithDelegationUserID(ctx, "deleg:session-a")

	res := reg.ExecuteWithContext(ctx, "delegate_task", map[string]any{
		"task":           "analyze the caching strategy in the codebase",
		"reuse_existing": false,
	}, "telegram", "room-1", "deleg:session-a", nil)

	if res == nil || res.IsError {
		t.Fatalf("delegate_task: %#v", res)
	}
	if !strings.Contains(res.ForLLM, "delegated-result-xyz") {
		t.Fatalf("expected sub-agent output in ForLLM, got: %s", res.ForLLM)
	}
	p.mu.Lock()
	tc, sc := p.TotalChats, p.SubAgentChats
	p.mu.Unlock()
	if sc != 1 {
		t.Fatalf("expected 1 sub-agent Chat call, got sub=%d total=%d", sc, tc)
	}
}

func TestFunctional_delegate_background_then_undelivered(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	db := filepath.Join(tmp, "delegation.db")
	p := &routingTestProvider{SubResponse: "bg-done-42"}
	ds, err := NewDelegationSystem(db, testDelegationConfig(), p, "m", tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer ds.Close()

	reg := tools.NewToolRegistry()
	ds.RegisterTools(reg)

	ctx := context.Background()
	ctx = tools.WithToolContext(ctx, "pico", "c1")
	ctx = tools.WithDelegationUserID(ctx, "deleg:session-b")

	sub := reg.ExecuteWithContext(ctx, "delegate_background", map[string]any{
		"task": "long analysis of dependencies",
	}, "pico", "c1", "deleg:session-b", nil)
	if sub == nil || sub.IsError {
		t.Fatalf("delegate_background: %#v", sub)
	}

	deadline := time.Now().Add(5 * time.Second)
	var records []*BackgroundTaskRecord
	for time.Now().Before(deadline) {
		records, err = ds.GetUndeliveredResults(ctx, "deleg:session-b")
		if err != nil {
			t.Fatal(err)
		}
		if len(records) == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 undelivered record, got %d", len(records))
	}
	if records[0].Status != "completed" && records[0].Status != "failed" {
		t.Fatalf("unexpected status %q", records[0].Status)
	}
	if records[0].Status == "completed" && !strings.Contains(records[0].Result, "bg-done-42") {
		t.Fatalf("expected result to contain sub-agent output, got %q", records[0].Result)
	}

	p.mu.Lock()
	sc := p.SubAgentChats
	p.mu.Unlock()
	if sc < 1 {
		t.Fatalf("expected background worker to invoke sub-agent Chat, sub calls=%d", sc)
	}
}

func TestFunctional_intercom_agentCreated_and_taskQueued(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	db := filepath.Join(tmp, "delegation.db")
	p := &routingTestProvider{SubResponse: "ok"}
	ds, err := NewDelegationSystem(db, testDelegationConfig(), p, "m", tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer ds.Close()

	var agentCreated, taskQueued atomic.Int32
	un1 := ds.OnAgentCreated(func(_ context.Context, e IntercomEvent) {
		if e.Topic == TopicAgentCreated {
			agentCreated.Add(1)
		}
	})
	un2 := ds.OnTaskQueued(func(_ context.Context, e IntercomEvent) {
		if e.Topic == TopicTaskQueued {
			taskQueued.Add(1)
		}
	})
	defer un1()
	defer un2()

	reg := tools.NewToolRegistry()
	ds.RegisterTools(reg)
	ctx := tools.WithDelegationUserID(context.Background(), "deleg:session-c")

	reg.ExecuteWithContext(ctx, "delegate_background", map[string]any{
		"task": "quick check",
	}, "", "", "deleg:session-c", nil)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if agentCreated.Load() >= 1 && taskQueued.Load() >= 1 {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if agentCreated.Load() < 1 {
		t.Fatal("expected agent:created from intercom")
	}
	if taskQueued.Load() < 1 {
		t.Fatal("expected task:queued from intercom")
	}
}

func TestFunctional_blackboard_intercomUserID(t *testing.T) {
	t.Parallel()
	ic := NewIntercom(10)
	b := NewBlackboard(5)
	b.SetIntercom(ic)
	var saw atomic.Bool
	un := ic.On(TopicBlackboardProposal, func(_ context.Context, e IntercomEvent) {
		if e.UserID == "partition-u" && e.Data["sub_agent_id"] == "sa-1" {
			saw.Store(true)
		}
	})
	defer un()
	b.Post("partition-u", "sa-1", "proposal-body")
	time.Sleep(50 * time.Millisecond)
	if !saw.Load() {
		t.Fatal("expected blackboard proposal on intercom with UserID=partition-u")
	}
}

func TestFunctional_DelegationSystem_blackboardFacade(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	ds, err := NewDelegationSystem(filepath.Join(tmp, "d.db"), testDelegationConfig(), &routingTestProvider{}, "m", tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer ds.Close()
	var resolved atomic.Bool
	un := ds.OnBlackboardResolved(func(_ context.Context, e IntercomEvent) {
		if e.Data["resolution"] == "approved" {
			resolved.Store(true)
		}
	})
	defer un()
	ds.BlackboardPost("u", "agent-x", "q?")
	ds.BlackboardResolve("u", "agent-x", "approved")
	time.Sleep(50 * time.Millisecond)
	if !resolved.Load() {
		t.Fatal("expected blackboard:resolved via facade")
	}
}

func TestFunctional_delegate_to_existing_sync(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	db := filepath.Join(tmp, "delegation.db")
	p := &routingTestProvider{SubResponse: "existing-agent-reply"}
	ds, err := NewDelegationSystem(db, testDelegationConfig(), p, "m", tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer ds.Close()

	reg := tools.NewToolRegistry()
	ds.RegisterTools(reg)
	ctx := tools.WithDelegationUserID(context.Background(), "deleg:session-d")

	// Create a sub-agent via delegate_task (reuse_existing false for deterministic new agent)
	r1 := reg.ExecuteWithContext(ctx, "delegate_task", map[string]any{
		"task":           "seed task for label",
		"label":          "worker-alpha",
		"reuse_existing": false,
	}, "x", "y", "deleg:session-d", nil)
	if r1 == nil || r1.IsError {
		t.Fatalf("seed delegate_task: %#v", r1)
	}

	// list to read agent id
	list := reg.ExecuteWithContext(ctx, "list_sub_agents", map[string]any{}, "x", "y", "deleg:session-d", nil)
	if list == nil || list.IsError || !strings.Contains(list.ForLLM, "worker-alpha") {
		t.Fatalf("list_sub_agents: %#v", list)
	}
	// Parse ID line "ID: agent-..." from list output
	idLine := ""
	for _, line := range strings.Split(list.ForLLM, "\n") {
		if strings.Contains(line, "ID:") {
			idLine = strings.TrimSpace(line)
			break
		}
	}
	const prefix = "ID: "
	idx := strings.Index(idLine, prefix)
	if idx < 0 {
		t.Fatalf("could not find agent id in list: %s", list.ForLLM)
	}
	agentID := strings.TrimSpace(idLine[idx+len(prefix):])

	r2 := reg.ExecuteWithContext(ctx, "delegate_to_existing", map[string]any{
		"sub_agent_id": agentID,
		"task":         "follow-up question",
		"background":   false,
	}, "x", "y", "deleg:session-d", nil)
	if r2 == nil || r2.IsError {
		t.Fatalf("delegate_to_existing: %#v", r2)
	}
	if !strings.Contains(r2.ForLLM, "existing-agent-reply") {
		t.Fatalf("expected follow-up result, got %s", r2.ForLLM)
	}
	p.mu.Lock()
	sc := p.SubAgentChats
	p.mu.Unlock()
	if sc < 2 {
		t.Fatalf("expected at least 2 sub-agent chats (seed + follow-up), got %d", sc)
	}
}
