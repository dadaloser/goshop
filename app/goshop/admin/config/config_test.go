package config

import (
	"testing"

	"goshop/pkg/log"
)

func TestAdminAuthOptionsHasPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		permission  string
		want        bool
	}{
		{
			name:        "empty required permission passes",
			permissions: nil,
			want:        true,
		},
		{
			name:        "exact permission passes",
			permissions: []string{" user:list ", "user:list"},
			permission:  "user:list",
			want:        true,
		},
		{
			name:        "resource wildcard passes",
			permissions: []string{"user:*"},
			permission:  "user:list",
			want:        true,
		},
		{
			name:        "global wildcard passes",
			permissions: []string{"*"},
			permission:  "order:refund",
			want:        true,
		},
		{
			name:        "unmatched permission rejects",
			permissions: []string{"goods:list"},
			permission:  "user:list",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &AdminAuthOptions{Permissions: tt.permissions}
			if got := opts.HasPermission(tt.permission); got != tt.want {
				t.Fatalf("HasPermission(%q) = %t, want %t", tt.permission, got, tt.want)
			}
		})
	}
}

func TestAdminAuthOptionsEffectivePermissionsFromEnv(t *testing.T) {
	t.Setenv("GOSHOP_ADMIN_PERMISSIONS", "user:list, goods:* , user:list,,")

	opts := &AdminAuthOptions{}
	got := opts.EffectivePermissions()
	want := []string{"user:list", "goods:*"}
	if len(got) != len(want) {
		t.Fatalf("len(EffectivePermissions()) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("EffectivePermissions()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestAdminAuthOptionsValidateStartup(t *testing.T) {
	t.Setenv("GOSHOP_ADMIN_TOKEN", "")
	t.Setenv("GOSHOP_ADMIN_PERMISSIONS", "")

	tests := []struct {
		name    string
		opts    *AdminAuthOptions
		wantErr bool
	}{
		{
			name:    "missing token rejects",
			opts:    &AdminAuthOptions{Permissions: []string{"user:list"}},
			wantErr: true,
		},
		{
			name:    "missing permissions rejects",
			opts:    &AdminAuthOptions{Token: "secret"},
			wantErr: true,
		},
		{
			name: "complete config passes",
			opts: &AdminAuthOptions{
				Token:       "secret",
				Permissions: []string{"user:list"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.ValidateStartup()
			if tt.wantErr && err == nil {
				t.Fatal("ValidateStartup() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateStartup() error = %v", err)
			}
		})
	}
}

func TestConfigValidateStartupRequiresAdminAuth(t *testing.T) {
	cfg := &Config{Log: &log.Options{}}

	if err := cfg.ValidateStartup(); err == nil {
		t.Fatal("ValidateStartup() error = nil, want missing admin-auth error")
	}
}

func TestConfigValidateStartupDoesNotBypassAuthInDevelopment(t *testing.T) {
	cfg := &Config{Log: &log.Options{Development: true}}

	if err := cfg.ValidateStartup(); err == nil {
		t.Fatal("ValidateStartup() error = nil, want missing admin-auth error")
	}
}
