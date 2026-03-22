package commands

import (
	"context"
	"testing"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/shield"
)

func TestApproveRejectCommands_WiredToApprovalManager(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		prefix string
		rt     func(*shield.ApprovalManager) *Runtime
	}{
		{
			name:   "approve",
			prefix: "/approve ",
			rt: func(mgr *shield.ApprovalManager) *Runtime {
				return &Runtime{
					ApproveAction: func(id string) bool { return mgr.Approve(id) },
				}
			},
		},
		{
			name:   "reject",
			prefix: "/reject ",
			rt: func(mgr *shield.ApprovalManager) *Runtime {
				return &Runtime{
					RejectAction: func(id string) bool { return mgr.Reject(id) },
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mgr := shield.NewApprovalManager(time.Minute)
			t.Cleanup(mgr.Close)

			req := mgr.CreateRequest(shield.ShieldEvent{
				Scope:    shield.ScopeToolCall,
				ToolName: "echo",
			}, shield.ShieldDecision{
				Action: shield.ActionRequireApproval,
				Reason: "test",
			})

			ex := NewExecutor(NewRegistry(BuiltinDefinitions()), tc.rt(mgr))

			var reply string
			res := ex.Execute(context.Background(), Request{
				Text: tc.prefix + req.ID,
				Reply: func(text string) error {
					reply = text
					return nil
				},
			})
			if res.Outcome != OutcomeHandled {
				t.Fatalf("outcome=%v", res.Outcome)
			}
			if reply == "" {
				t.Fatal("expected reply")
			}
		})
	}
}
