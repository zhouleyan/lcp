package iam

import (
	"net/mail"
	"regexp"

	"lcp.io/lcp/lib/api/validation"
)

var (
	usernameRegexp      = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	phoneRegexp         = regexp.MustCompile(`^1[3-9]\d{9}$`)
	workspaceNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)
	namespaceNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)
	passwordUpperRegexp = regexp.MustCompile(`[A-Z]`)
	passwordLowerRegexp = regexp.MustCompile(`[a-z]`)
	passwordDigitRegexp = regexp.MustCompile(`[0-9]`)
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

	if spec.Phone == "" {
		errs = append(errs, validation.FieldError{Field: "spec.phone", Message: "is required"})
	} else if !phoneRegexp.MatchString(spec.Phone) {
		errs = append(errs, validation.FieldError{Field: "spec.phone", Message: "must be a valid Chinese mobile number (e.g. 13800138000)"})
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

// ValidateWorkspaceCreate validates workspace creation.
func ValidateWorkspaceCreate(name string, spec *WorkspaceSpec) validation.ErrorList {
	var errs validation.ErrorList

	if name == "" {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
	} else if !workspaceNameRegexp.MatchString(name) {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens"})
	}

	if spec.OwnerID == "" {
		errs = append(errs, validation.FieldError{Field: "spec.ownerId", Message: "is required"})
	}

	if spec.Status != "" && spec.Status != "active" && spec.Status != "inactive" {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}

	return errs
}

// ValidateNamespaceCreate validates namespace creation.
func ValidateNamespaceCreate(name string, spec *NamespaceSpec) validation.ErrorList {
	var errs validation.ErrorList

	if name == "" {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
	} else if !namespaceNameRegexp.MatchString(name) {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens"})
	}

	if spec.WorkspaceID == "" {
		errs = append(errs, validation.FieldError{Field: "spec.workspaceId", Message: "is required"})
	}

	if spec.OwnerID == "" {
		errs = append(errs, validation.FieldError{Field: "spec.ownerId", Message: "is required"})
	}

	if spec.Visibility != "" && spec.Visibility != "public" && spec.Visibility != "private" {
		errs = append(errs, validation.FieldError{Field: "spec.visibility", Message: "must be 'public' or 'private'"})
	}

	if spec.MaxMembers < 0 {
		errs = append(errs, validation.FieldError{Field: "spec.maxMembers", Message: "must be >= 0"})
	}

	return errs
}

// ValidatePassword validates a password string.
func ValidatePassword(password string) validation.ErrorList {
	var errs validation.ErrorList
	if len(password) < 8 || len(password) > 128 {
		errs = append(errs, validation.FieldError{Field: "spec.password", Message: "must be 8-128 characters"})
		return errs
	}
	if !passwordUpperRegexp.MatchString(password) {
		errs = append(errs, validation.FieldError{Field: "spec.password", Message: "must contain at least one uppercase letter"})
	}
	if !passwordLowerRegexp.MatchString(password) {
		errs = append(errs, validation.FieldError{Field: "spec.password", Message: "must contain at least one lowercase letter"})
	}
	if !passwordDigitRegexp.MatchString(password) {
		errs = append(errs, validation.FieldError{Field: "spec.password", Message: "must contain at least one digit"})
	}
	return errs
}
