// Package authclaims defines the claims carried by GoShop access tokens.
package authclaims

import "github.com/golang-jwt/jwt/v5"

// Claims contains GoShop identity, role, permission, and resource-scope data.
// The framework treats it only as jwt.Claims and does not depend on these fields.
type Claims struct {
	ID              uint     `json:"user_id"`
	NickName        string   `json:"nick_name,omitempty"`
	AuthorityID     uint     `json:"authority_id,omitempty"`
	Roles           []string `json:"roles,omitempty"`
	PrincipalType   string   `json:"principal_type,omitempty"`
	AccountStatus   string   `json:"status,omitempty"`
	Scope           []string `json:"scope,omitempty"`
	TokenVersion    uint64   `json:"token_version"`
	SessionID       string   `json:"session_id,omitempty"`
	ResourceDomains []string `json:"resource_domains,omitempty"`
	ResourceStores  []string `json:"resource_stores,omitempty"`
	ResourceTeams   []string `json:"resource_teams,omitempty"`
	ResourceScopes  []string `json:"resource_scopes,omitempty"`
	jwt.RegisteredClaims
}
