package validation

import (
	"testing"

	"lcp.io/lcp/lib/api/types"
)

func TestValidateUserCreate_Valid(t *testing.T) {
	spec := &types.UserSpec{
		Username: "alice",
		Email:    "alice@example.com",
		Phone:    "+8613800138000",
	}
	errs := ValidateUserCreate(spec)
	if errs.HasErrors() {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateUserCreate_EmptyUsername(t *testing.T) {
	spec := &types.UserSpec{Email: "a@b.com"}
	errs := ValidateUserCreate(spec)
	if !errs.HasErrors() {
		t.Error("expected error for empty username")
	}
}

func TestValidateUserCreate_InvalidUsername(t *testing.T) {
	spec := &types.UserSpec{Username: "a!", Email: "a@b.com"}
	errs := ValidateUserCreate(spec)
	if !errs.HasErrors() {
		t.Error("expected error for invalid username")
	}
}

func TestValidateUserCreate_ShortUsername(t *testing.T) {
	spec := &types.UserSpec{Username: "ab", Email: "a@b.com"}
	errs := ValidateUserCreate(spec)
	if !errs.HasErrors() {
		t.Error("expected error for short username")
	}
}

func TestValidateUserCreate_EmptyEmail(t *testing.T) {
	spec := &types.UserSpec{Username: "alice"}
	errs := ValidateUserCreate(spec)
	if !errs.HasErrors() {
		t.Error("expected error for empty email")
	}
}

func TestValidateUserCreate_InvalidEmail(t *testing.T) {
	spec := &types.UserSpec{Username: "alice", Email: "not-an-email"}
	errs := ValidateUserCreate(spec)
	if !errs.HasErrors() {
		t.Error("expected error for invalid email")
	}
}

func TestValidateUserCreate_InvalidPhone(t *testing.T) {
	spec := &types.UserSpec{Username: "alice", Email: "a@b.com", Phone: "12345"}
	errs := ValidateUserCreate(spec)
	if !errs.HasErrors() {
		t.Error("expected error for invalid phone")
	}
}

func TestValidateUserCreate_ValidPhoneOptional(t *testing.T) {
	spec := &types.UserSpec{Username: "alice", Email: "a@b.com"}
	errs := ValidateUserCreate(spec)
	if errs.HasErrors() {
		t.Errorf("phone is optional, got errors: %v", errs)
	}
}
