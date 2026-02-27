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

func TestIsNotFound(t *testing.T) {
	err := NewNotFound("User", "alice")
	if !IsNotFound(err) {
		t.Error("expected IsNotFound to be true")
	}
	if IsNotFound(NewBadRequest("x", nil)) {
		t.Error("expected IsNotFound to be false for BadRequest")
	}
}
