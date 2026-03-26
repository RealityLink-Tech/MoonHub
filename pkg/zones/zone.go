// Package zones provides data partition management for MoonHub.
// Resources are assigned to zones (privacy, shared, public) and
// access is checked based on the requester's relationship.
package zones

// Zone represents a data partition level.
type Zone int

const (
	ZonePrivacy Zone = iota
	ZoneShared
	ZonePublic
)

func (z Zone) String() string {
	switch z {
	case ZonePrivacy:
		return "privacy"
	case ZoneShared:
		return "shared"
	case ZonePublic:
		return "public"
	default:
		return "unknown"
	}
}

// Relationship represents the relationship between two agents.
type Relationship int

const (
	RelationshipNone Relationship = iota
	RelationshipFriend
	RelationshipOwner
)

func (r Relationship) String() string {
	switch r {
	case RelationshipOwner:
		return "owner"
	case RelationshipFriend:
		return "friend"
	case RelationshipNone:
		return "none"
	default:
		return "unknown"
	}
}

// AccessLevel represents the type of access being requested.
type AccessLevel int

const (
	AccessRead AccessLevel = iota
	AccessWrite
)

func (a AccessLevel) String() string {
	switch a {
	case AccessRead:
		return "read"
	case AccessWrite:
		return "write"
	default:
		return "unknown"
	}
}

// Allow checks whether a given relationship has the requested access to this zone.
func (z Zone) Allow(rel Relationship, access AccessLevel) bool {
	switch z {
	case ZonePrivacy:
		return rel == RelationshipOwner
	case ZoneShared:
		switch rel {
		case RelationshipOwner:
			return true
		case RelationshipFriend:
			return access == AccessRead
		default:
			return false
		}
	case ZonePublic:
		return access == AccessRead || rel == RelationshipOwner
	default:
		return false
	}
}

// ZoneManager manages resource-to-zone mappings.
// Resources default to ZonePrivacy if not explicitly assigned.
type ZoneManager struct {
	resources map[string]Zone
}

// NewZoneManager creates a new zone manager.
// The storeDir parameter is reserved for future persistent storage.
func NewZoneManager(storeDir string) *ZoneManager {
	return &ZoneManager{
		resources: make(map[string]Zone),
	}
}

// SetResourceZone assigns a resource to a zone.
func (zm *ZoneManager) SetResourceZone(resource string, zone Zone) {
	zm.resources[resource] = zone
}

// GetResourceZone returns the zone for a resource, defaulting to ZonePrivacy.
func (zm *ZoneManager) GetResourceZone(resource string) Zone {
	if zone, ok := zm.resources[resource]; ok {
		return zone
	}
	return ZonePrivacy
}

// CheckAccess checks whether a requester with the given relationship
// has the requested access level to a specific resource.
func (zm *ZoneManager) CheckAccess(resource string, rel Relationship, access AccessLevel) bool {
	zone := zm.GetResourceZone(resource)
	return zone.Allow(rel, access)
}

// ListResources returns all resources assigned to a given zone.
func (zm *ZoneManager) ListResources(zone Zone) []string {
	var result []string
	for res, z := range zm.resources {
		if z == zone {
			result = append(result, res)
		}
	}
	return result
}
