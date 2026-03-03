package namespace

import (
	"regexp"

	"lcp.io/lcp/lib/api/validation"
)

var namespaceNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

// ValidateNamespaceCreate validates namespace creation.
func ValidateNamespaceCreate(name string, spec *NamespaceSpec) validation.ErrorList {
	var errs validation.ErrorList

	if name == "" {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
	} else if !namespaceNameRegexp.MatchString(name) {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens"})
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

// ValidateNamespaceMember validates adding a member to a namespace.
func ValidateNamespaceMember(spec *NamespaceMemberSpec) validation.ErrorList {
	var errs validation.ErrorList

	if spec.UserID == "" {
		errs = append(errs, validation.FieldError{Field: "spec.userId", Message: "is required"})
	}

	if spec.Role == "" {
		errs = append(errs, validation.FieldError{Field: "spec.role", Message: "is required"})
	} else if spec.Role != "admin" && spec.Role != "member" && spec.Role != "viewer" {
		errs = append(errs, validation.FieldError{Field: "spec.role", Message: "must be 'admin', 'member', or 'viewer'"})
	}

	return errs
}
