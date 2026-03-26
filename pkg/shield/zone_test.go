package shield

import (
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/zones"
)

func TestZoneAccessDecision(t *testing.T) {
	tests := []struct {
		name       string
		zone       zones.Zone
		rel        zones.Relationship
		access     zones.AccessLevel
		wantAction ShieldAction
	}{
		{"owner read privacy", zones.ZonePrivacy, zones.RelationshipOwner, zones.AccessRead, ActionLog},
		{"friend read privacy", zones.ZonePrivacy, zones.RelationshipFriend, zones.AccessRead, ActionBlock},
		{"friend read shared", zones.ZoneShared, zones.RelationshipFriend, zones.AccessRead, ActionLog},
		{"friend write shared", zones.ZoneShared, zones.RelationshipFriend, zones.AccessWrite, ActionBlock},
		{"none read public", zones.ZonePublic, zones.RelationshipNone, zones.AccessRead, ActionLog},
		{"none write public", zones.ZonePublic, zones.RelationshipNone, zones.AccessWrite, ActionBlock},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := EvaluateZoneAccess(tt.zone, tt.rel, tt.access, "test-resource")
			if decision.Action != tt.wantAction {
				t.Errorf("EvaluateZoneAccess() action = %s, want %s", decision.Action, tt.wantAction)
			}
		})
	}
}
