package authz

import "testing"

func TestAuthorizationHelpersAndCollections(t *testing.T) {
	all := AllPermissions()
	if len(all) == 0 {
		t.Fatal("AllPermissions() returned empty slice")
	}
	all[0] = Permission("mutated")
	if AllPermissions()[0] == Permission("mutated") {
		t.Fatal("AllPermissions() returned shared backing slice")
	}

	if !IsValidPermission(string(PermissionGoodsReadAny)) {
		t.Fatalf("IsValidPermission(%q) = false, want true", PermissionGoodsReadAny)
	}
	if IsValidPermission(" not-a-real-permission ") {
		t.Fatal("IsValidPermission(invalid) = true, want false")
	}

	if got := NormalizeAccountStatus(" "); got != AccountStatusActive {
		t.Fatalf("NormalizeAccountStatus(blank) = %q, want %q", got, AccountStatusActive)
	}
	if got := NormalizeAccountStatus(" LOCKED "); got != AccountStatusLocked {
		t.Fatalf("NormalizeAccountStatus(LOCKED) = %q, want %q", got, AccountStatusLocked)
	}

	domains := AllBusinessDomains()
	if len(domains) == 0 {
		t.Fatal("AllBusinessDomains() returned empty slice")
	}
	domains[0] = BusinessDomain("mutated")
	if AllBusinessDomains()[0] == BusinessDomain("mutated") {
		t.Fatal("AllBusinessDomains() returned shared backing slice")
	}

	if !IsValidBusinessDomain(string(BusinessDomainCatalog)) {
		t.Fatal("IsValidBusinessDomain(catalog) = false, want true")
	}
	if IsValidBusinessDomain("unknown") {
		t.Fatal("IsValidBusinessDomain(unknown) = true, want false")
	}

	builtin := BuiltinRoleDefinitions()
	if len(builtin) == 0 {
		t.Fatal("BuiltinRoleDefinitions() returned empty slice")
	}
	builtin[0].Name = "mutated"
	if BuiltinRoleDefinitions()[0].Name == "mutated" {
		t.Fatal("BuiltinRoleDefinitions() returned shared backing slice")
	}

	if !IsReservedNonStaffRoleName("basic") {
		t.Fatal("IsReservedNonStaffRoleName(basic) = false, want true")
	}
	if IsReservedNonStaffRoleName(string(StaffRoleAdmin)) {
		t.Fatalf("IsReservedNonStaffRoleName(%q) = true, want false", StaffRoleAdmin)
	}
}

func TestResourceScopeEncodingParsingAndAuthorization(t *testing.T) {
	scope := ResourceScope{
		Domain:       " catalog ",
		StoreID:      "store-a",
		TeamID:       " team-a ",
		ResourceType: " GOODS ",
		ResourceID:   " sku-1 ",
	}
	normalized := NormalizeResourceScope(scope)
	if normalized.Domain != "catalog" || normalized.TeamID != "team-a" || normalized.ResourceType != "goods" || normalized.ResourceID != "sku-1" {
		t.Fatalf("NormalizeResourceScope() = %+v", normalized)
	}

	encoded := EncodeResourceScope(scope)
	decoded := DecodeResourceScope(encoded)
	if decoded != normalized {
		t.Fatalf("DecodeResourceScope(EncodeResourceScope()) = %+v, want %+v", decoded, normalized)
	}

	parsed := ParseResourceScopes([]any{encoded, 42, "finance\x1fstore-b"})
	if len(parsed) != 2 {
		t.Fatalf("ParseResourceScopes([]any) len = %d, want 2", len(parsed))
	}
	if parsed[1].Domain != "finance" || parsed[1].StoreID != "store-b" {
		t.Fatalf("ParseResourceScopes() second item = %+v", parsed[1])
	}

	granted := []ResourceScope{
		{Domain: "catalog", StoreID: "store-a"},
		{Domain: "catalog", StoreID: "store-b", ResourceType: "goods", ResourceID: "goods-2"},
	}
	if !ResourceScopeAllows(granted, ResourceScope{Domain: "catalog", StoreID: "store-a", ResourceType: "goods", ResourceID: "goods-1"}) {
		t.Fatal("ResourceScopeAllows() broad store grant = false, want true")
	}
	if !ResourceScopeAllows(granted, ResourceScope{Domain: "catalog", StoreID: "store-b", ResourceType: "goods", ResourceID: "goods-2"}) {
		t.Fatal("ResourceScopeAllows() resource grant = false, want true")
	}
	if ResourceScopeAllows(granted, ResourceScope{Domain: "catalog", StoreID: "store-b", ResourceType: "goods", ResourceID: "goods-3"}) {
		t.Fatal("ResourceScopeAllows() unrelated resource = true, want false")
	}
}
