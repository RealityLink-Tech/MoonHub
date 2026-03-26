package zones

import "testing"

func TestZoneString(t *testing.T) {
	tests := []struct {
		zone     Zone
		expected string
	}{
		{ZonePrivacy, "privacy"},
		{ZoneShared, "shared"},
		{ZonePublic, "public"},
	}
	for _, tt := range tests {
		if got := tt.zone.String(); got != tt.expected {
			t.Errorf("Zone(%d).String() = %s, want %s", tt.zone, got, tt.expected)
		}
	}
}

func TestZoneAccess(t *testing.T) {
	tests := []struct {
		zone      Zone
		rel       Relationship
		access    AccessLevel
		wantAllow bool
	}{
		{ZonePrivacy, RelationshipOwner, AccessRead, true},
		{ZonePrivacy, RelationshipOwner, AccessWrite, true},
		{ZonePrivacy, RelationshipFriend, AccessRead, false},
		{ZonePrivacy, RelationshipNone, AccessRead, false},

		{ZoneShared, RelationshipOwner, AccessWrite, true},
		{ZoneShared, RelationshipFriend, AccessRead, true},
		{ZoneShared, RelationshipFriend, AccessWrite, false},
		{ZoneShared, RelationshipNone, AccessRead, false},

		{ZonePublic, RelationshipOwner, AccessWrite, true},
		{ZonePublic, RelationshipFriend, AccessRead, true},
		{ZonePublic, RelationshipNone, AccessRead, true},
		{ZonePublic, RelationshipNone, AccessWrite, false},
	}
	for _, tt := range tests {
		if got := tt.zone.Allow(tt.rel, tt.access); got != tt.wantAllow {
			t.Errorf("Zone(%s).Allow(%s, %s) = %v, want %v",
				tt.zone, tt.rel, tt.access, got, tt.wantAllow)
		}
	}
}

func TestNewZoneManager(t *testing.T) {
	zm := NewZoneManager("/tmp/test-zones")
	if zm == nil {
		t.Fatal("expected non-nil ZoneManager")
	}
}

func TestZoneManager_SetAndGet(t *testing.T) {
	zm := NewZoneManager("")

	zm.SetResourceZone("project-code", ZoneShared)
	if got := zm.GetResourceZone("project-code"); got != ZoneShared {
		t.Errorf("expected shared, got %s", got)
	}
}

func TestZoneManager_DefaultZone(t *testing.T) {
	zm := NewZoneManager("")
	if got := zm.GetResourceZone("nonexistent"); got != ZonePrivacy {
		t.Errorf("expected privacy (default), got %s", got)
	}
}

func TestZoneManager_CheckAccess(t *testing.T) {
	zm := NewZoneManager("")
	zm.SetResourceZone("public-readme", ZonePublic)
	zm.SetResourceZone("shared-project", ZoneShared)
	zm.SetResourceZone("private-keys", ZonePrivacy)

	tests := []struct {
		resource  string
		rel       Relationship
		access    AccessLevel
		wantAllow bool
	}{
		{"public-readme", RelationshipNone, AccessRead, true},
		{"public-readme", RelationshipNone, AccessWrite, false},
		{"shared-project", RelationshipFriend, AccessRead, true},
		{"shared-project", RelationshipFriend, AccessWrite, false},
		{"shared-project", RelationshipNone, AccessRead, false},
		{"private-keys", RelationshipFriend, AccessRead, false},
		{"private-keys", RelationshipOwner, AccessRead, true},
	}
	for _, tt := range tests {
		if got := zm.CheckAccess(tt.resource, tt.rel, tt.access); got != tt.wantAllow {
			t.Errorf("CheckAccess(%s, %s, %s) = %v, want %v",
				tt.resource, tt.rel, tt.access, got, tt.wantAllow)
		}
	}
}

func TestZoneManager_ListResources(t *testing.T) {
	zm := NewZoneManager("")
	zm.SetResourceZone("a", ZonePublic)
	zm.SetResourceZone("b", ZoneShared)
	zm.SetResourceZone("c", ZonePrivacy)

	resources := zm.ListResources(ZoneShared)
	if len(resources) != 1 || resources[0] != "b" {
		t.Errorf("expected [b], got %v", resources)
	}
}
