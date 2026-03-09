package errors

import "testing"

func TestNewBadRequest(t *testing.T) {
	err := NewBadRequest("invalid input", nil)
	if err.Status != 400 || err.Reason != "BadRequest" {
		t.Errorf("unexpected: %+v", err)
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestNewNotFound(t *testing.T) {
	err := NewNotFound("User", "alice")
	if err.Status != 404 || err.Reason != "NotFound" {
		t.Errorf("unexpected: %+v", err)
	}
}

func TestNewConflict(t *testing.T) {
	err := NewConflict("User", "alice")
	if err.Status != 409 || err.Reason != "Conflict" {
		t.Errorf("unexpected: %+v", err)
	}
}

func TestNewInternalError(t *testing.T) {
	err := NewInternalError(nil)
	if err.Status != 500 {
		t.Errorf("unexpected: %+v", err)
	}
}

func TestNewForbidden(t *testing.T) {
	err := NewForbidden("access denied")
	if err.Status != 403 || err.Reason != "Forbidden" {
		t.Errorf("unexpected: %+v", err)
	}
	if err.Message != "access denied" {
		t.Errorf("unexpected message: %s", err.Message)
	}
}

func TestIsForbidden(t *testing.T) {
	err := NewForbidden("no access")
	if !IsForbidden(err) {
		t.Error("expected IsForbidden to be true")
	}
	if IsForbidden(NewNotFound("User", "alice")) {
		t.Error("expected IsForbidden to be false for NotFound")
	}
	if IsForbidden(NewBadRequest("x", nil)) {
		t.Error("expected IsForbidden to be false for BadRequest")
	}
}

func TestIsNotFound(t *testing.T) {
	err := NewNotFound("User", "alice")
	if !IsNotFound(err) {
		t.Error("expected IsNotFound to be true")
	}
	if IsNotFound(NewBadRequest("x", nil)) {
		t.Error("expected IsNotFound to be false for BadRequest")
	}
}
