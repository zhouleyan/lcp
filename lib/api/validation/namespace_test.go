package validation

import (
	"testing"

	"lcp.io/lcp/lib/api/types"
)

func TestValidateNamespaceCreate_Valid(t *testing.T) {
	name := "my-team"
	spec := &types.NamespaceSpec{OwnerID: "1", Visibility: "private"}
	errs := ValidateNamespaceCreate(name, spec)
	if errs.HasErrors() {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateNamespaceCreate_EmptyName(t *testing.T) {
	spec := &types.NamespaceSpec{OwnerID: "1"}
	errs := ValidateNamespaceCreate("", spec)
	if !errs.HasErrors() {
		t.Error("expected error for empty name")
	}
}

func TestValidateNamespaceCreate_InvalidName(t *testing.T) {
	spec := &types.NamespaceSpec{OwnerID: "1"}
	errs := ValidateNamespaceCreate("My Team!", spec)
	if !errs.HasErrors() {
		t.Error("expected error for invalid name")
	}
}

func TestValidateNamespaceCreate_EmptyOwnerID(t *testing.T) {
	spec := &types.NamespaceSpec{}
	errs := ValidateNamespaceCreate("my-team", spec)
	if !errs.HasErrors() {
		t.Error("expected error for empty ownerId")
	}
}

func TestValidateNamespaceCreate_InvalidVisibility(t *testing.T) {
	spec := &types.NamespaceSpec{OwnerID: "1", Visibility: "secret"}
	errs := ValidateNamespaceCreate("my-team", spec)
	if !errs.HasErrors() {
		t.Error("expected error for invalid visibility")
	}
}

func TestValidateNamespaceMember_Valid(t *testing.T) {
	spec := &types.NamespaceMemberSpec{UserID: "1", Role: "member"}
	errs := ValidateNamespaceMember(spec)
	if errs.HasErrors() {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateNamespaceMember_InvalidRole(t *testing.T) {
	spec := &types.NamespaceMemberSpec{UserID: "1", Role: "superadmin"}
	errs := ValidateNamespaceMember(spec)
	if !errs.HasErrors() {
		t.Error("expected error for invalid role")
	}
}

func TestValidateNamespaceMember_EmptyUserID(t *testing.T) {
	spec := &types.NamespaceMemberSpec{Role: "member"}
	errs := ValidateNamespaceMember(spec)
	if !errs.HasErrors() {
		t.Error("expected error for empty userId")
	}
}
