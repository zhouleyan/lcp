package validation

import (
	"regexp"

	"lcp.io/lcp/lib/api/types"
)

var namespaceNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

// ValidateNamespaceCreate validates namespace creation.
func ValidateNamespaceCreate(name string, spec *types.NamespaceSpec) ErrorList {
	var errs ErrorList

	if name == "" {
		errs = append(errs, FieldError{Field: "metadata.name", Message: "is required"})
	} else if !namespaceNameRegexp.MatchString(name) {
		errs = append(errs, FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens"})
	}

	if spec.OwnerID == "" {
		errs = append(errs, FieldError{Field: "spec.ownerId", Message: "is required"})
	}

	if spec.Visibility != "" && spec.Visibility != "public" && spec.Visibility != "private" {
		errs = append(errs, FieldError{Field: "spec.visibility", Message: "must be 'public' or 'private'"})
	}

	if spec.MaxMembers < 0 {
		errs = append(errs, FieldError{Field: "spec.maxMembers", Message: "must be >= 0"})
	}

	return errs
}

// ValidateNamespaceMember validates adding a member to a namespace.
func ValidateNamespaceMember(spec *types.NamespaceMemberSpec) ErrorList {
	var errs ErrorList

	if spec.UserID == "" {
		errs = append(errs, FieldError{Field: "spec.userId", Message: "is required"})
	}

	if spec.Role == "" {
		errs = append(errs, FieldError{Field: "spec.role", Message: "is required"})
	} else if spec.Role != "admin" && spec.Role != "member" && spec.Role != "viewer" {
		errs = append(errs, FieldError{Field: "spec.role", Message: "must be 'admin', 'member', or 'viewer'"})
	}

	return errs
}
