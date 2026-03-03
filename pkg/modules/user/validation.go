package user

import (
	"net/mail"
	"regexp"

	"lcp.io/lcp/lib/api/validation"
)

var (
	usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	phoneRegexp    = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
)

// ValidateUserCreate validates a UserSpec for creation.
func ValidateUserCreate(spec *UserSpec) validation.ErrorList {
	var errs validation.ErrorList

	if spec.Username == "" {
		errs = append(errs, validation.FieldError{Field: "spec.username", Message: "is required"})
	} else if !usernameRegexp.MatchString(spec.Username) {
		errs = append(errs, validation.FieldError{Field: "spec.username", Message: "must be 3-50 alphanumeric characters or underscores"})
	}

	if spec.Email == "" {
		errs = append(errs, validation.FieldError{Field: "spec.email", Message: "is required"})
	} else if _, err := mail.ParseAddress(spec.Email); err != nil {
		errs = append(errs, validation.FieldError{Field: "spec.email", Message: "is not a valid email address"})
	}

	if spec.Phone != "" && !phoneRegexp.MatchString(spec.Phone) {
		errs = append(errs, validation.FieldError{Field: "spec.phone", Message: "must be E.164 format (e.g. +8613800138000)"})
	}

	if spec.Status != "" && spec.Status != "active" && spec.Status != "inactive" {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}

	return errs
}

// ValidateUserUpdate validates a UserSpec for full update.
func ValidateUserUpdate(spec *UserSpec) validation.ErrorList {
	return ValidateUserCreate(spec)
}
