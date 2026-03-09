package errors

import (
	"errors"
	"fmt"
	"net/http"

	"lcp.io/lcp/lib/runtime"
)

// StatusError represents an API error with HTTP status code.
type StatusError struct {
	runtime.TypeMeta `json:",inline"`
	Status           int    `json:"status"`
	Reason           string `json:"reason"`
	Message          string `json:"message"`
	Details          any    `json:"details,omitempty"`
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%s: %s", e.Reason, e.Message)
}

func (e *StatusError) GetTypeMeta() *runtime.TypeMeta {
	return &e.TypeMeta
}

// GetStatus returns the HTTP status code.
func (e *StatusError) GetStatus() int {
	return e.Status
}

func newStatusError(status int, reason, message string, details any) *StatusError {
	return &StatusError{
		TypeMeta: runtime.TypeMeta{APIVersion: "v1", Kind: "Status"},
		Status:   status,
		Reason:   reason,
		Message:  message,
		Details:  details,
	}
}

func NewBadRequest(message string, details any) *StatusError {
	return newStatusError(http.StatusBadRequest, "BadRequest", message, details)
}

func NewNotFound(resource, name string) *StatusError {
	return newStatusError(http.StatusNotFound, "NotFound",
		fmt.Sprintf("%s %q not found", resource, name), nil)
}

func NewConflict(resource, name string) *StatusError {
	return newStatusError(http.StatusConflict, "Conflict",
		fmt.Sprintf("%s %q already exists", resource, name), nil)
}

func NewConflictMessage(message string) *StatusError {
	return newStatusError(http.StatusConflict, "Conflict", message, nil)
}

func NewForbidden(message string) *StatusError {
	return newStatusError(http.StatusForbidden, "Forbidden", message, nil)
}

func NewInternalError(err error) *StatusError {
	msg := "internal server error"
	if err != nil {
		msg = err.Error()
	}
	return newStatusError(http.StatusInternalServerError, "InternalError", msg, nil)
}

func IsNotFound(err error) bool {
	if se, ok := errors.AsType[*StatusError](err); ok {
		return se.Status == http.StatusNotFound
	}
	return false
}

func IsConflict(err error) bool {
	if se, ok := errors.AsType[*StatusError](err); ok {
		return se.Status == http.StatusConflict
	}
	return false
}

func IsForbidden(err error) bool {
	if se, ok := errors.AsType[*StatusError](err); ok {
		return se.Status == http.StatusForbidden
	}
	return false
}
