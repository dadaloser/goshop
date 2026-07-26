package authz

import "strings"

const scopeBindingSeparator = "\x1f"

type ResourceScope struct {
	Domain       string
	StoreID      string
	TeamID       string
	ResourceType string
	ResourceID   string
}

func NormalizeResourceScope(scope ResourceScope) ResourceScope {
	scope.Domain = strings.ToLower(strings.TrimSpace(scope.Domain))
	scope.StoreID = strings.TrimSpace(scope.StoreID)
	scope.TeamID = strings.TrimSpace(scope.TeamID)
	scope.ResourceType = strings.ToLower(strings.TrimSpace(scope.ResourceType))
	scope.ResourceID = strings.TrimSpace(scope.ResourceID)
	return scope
}

func ResourceScopeKey(scope ResourceScope) string {
	scope = NormalizeResourceScope(scope)
	return strings.Join([]string{scope.Domain, scope.StoreID, scope.TeamID, scope.ResourceType, scope.ResourceID}, scopeBindingSeparator)
}

func EncodeResourceScope(scope ResourceScope) string {
	return ResourceScopeKey(scope)
}

func DecodeResourceScope(value string) ResourceScope {
	parts := strings.Split(value, scopeBindingSeparator)
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	return NormalizeResourceScope(ResourceScope{
		Domain:       parts[0],
		StoreID:      parts[1],
		TeamID:       parts[2],
		ResourceType: parts[3],
		ResourceID:   parts[4],
	})
}

func ParseResourceScopes(raw any) []ResourceScope {
	values := []ResourceScope{}
	switch typed := raw.(type) {
	case []string:
		for _, value := range typed {
			values = append(values, DecodeResourceScope(value))
		}
	case []any:
		for _, value := range typed {
			text, ok := value.(string)
			if !ok {
				continue
			}
			values = append(values, DecodeResourceScope(text))
		}
	}
	return values
}

func ResourceScopeAllows(granted []ResourceScope, requested ResourceScope) bool {
	requested = NormalizeResourceScope(requested)
	for _, item := range granted {
		item = NormalizeResourceScope(item)
		if item.Domain != requested.Domain || item.StoreID != requested.StoreID || item.TeamID != requested.TeamID {
			continue
		}
		if item.ResourceType == "" && item.ResourceID == "" {
			return true
		}
		if item.ResourceType == requested.ResourceType && item.ResourceID == requested.ResourceID {
			return true
		}
	}
	return false
}
