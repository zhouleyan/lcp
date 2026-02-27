package validation

import (
	"net/mail"
	"regexp"

	"lcp.io/lcp/lib/api/types"
)

var (
	usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	phoneRegexp    = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
)

// ValidateUserCreate validates a UserSpec for creation.
func ValidateUserCreate(spec *types.UserSpec) ErrorList {
	var errs ErrorList

	if spec.Username == "" {
		errs = append(errs, FieldError{Field: "spec.username", Message: "is required"})
	} else if !usernameRegexp.MatchString(spec.Username) {
		errs = append(errs, FieldError{Field: "spec.username", Message: "must be 3-50 alphanumeric characters or underscores"})
	}

	if spec.Email == "" {
		errs = append(errs, FieldError{Field: "spec.email", Message: "is required"})
	} else if _, err := mail.ParseAddress(spec.Email); err != nil {
		errs = append(errs, FieldError{Field: "spec.email", Message: "is not a valid email address"})
	}

	if spec.Phone != "" && !phoneRegexp.MatchString(spec.Phone) {
		errs = append(errs, FieldError{Field: "spec.phone", Message: "must be E.164 format (e.g. +8613800138000)"})
	}

	if spec.Status != "" && spec.Status != "active" && spec.Status != "inactive" {
		errs = append(errs, FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}

	return errs
}
