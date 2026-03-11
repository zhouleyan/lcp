package infra

import (
	"regexp"

	"lcp.io/lcp/lib/api/validation"
)

var (
	nameRegexp      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)
	validScopes     = map[string]bool{ScopePlatform: true, ScopeWorkspace: true, ScopeNamespace: true}
	validStatuses   = map[string]bool{"active": true, "inactive": true}
	validEnvTypes   = map[string]bool{"development": true, "testing": true, "staging": true, "production": true, "custom": true}
)

// ValidateHostCreate validates a HostSpec for creation.
func ValidateHostCreate(name string, spec *HostSpec) validation.ErrorList {
	var errs validation.ErrorList

	if name == "" {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
	} else if !nameRegexp.MatchString(name) {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens"})
	}

	if spec.Status != "" && !validStatuses[spec.Status] {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}

	return errs
}

// ValidateHostUpdate validates a HostSpec for full update.
func ValidateHostUpdate(spec *HostSpec) validation.ErrorList {
	var errs validation.ErrorList

	if spec.Status != "" && !validStatuses[spec.Status] {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}

	return errs
}

// ValidateEnvironmentCreate validates an EnvironmentSpec for creation.
func ValidateEnvironmentCreate(name string, spec *EnvironmentSpec) validation.ErrorList {
	var errs validation.ErrorList

	if name == "" {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
	} else if !nameRegexp.MatchString(name) {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens"})
	}

	if spec.EnvType != "" && !validEnvTypes[spec.EnvType] {
		errs = append(errs, validation.FieldError{Field: "spec.envType", Message: "must be development, testing, staging, production, or custom"})
	}

	if spec.Status != "" && !validStatuses[spec.Status] {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}

	return errs
}

// ValidateEnvironmentUpdate validates an EnvironmentSpec for full update.
func ValidateEnvironmentUpdate(spec *EnvironmentSpec) validation.ErrorList {
	var errs validation.ErrorList

	if spec.EnvType != "" && !validEnvTypes[spec.EnvType] {
		errs = append(errs, validation.FieldError{Field: "spec.envType", Message: "must be development, testing, staging, production, or custom"})
	}

	if spec.Status != "" && !validStatuses[spec.Status] {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}

	return errs
}

// ValidateAssignRequest validates an assign/unassign request.
func ValidateAssignRequest(req *AssignRequest) validation.ErrorList {
	var errs validation.ErrorList

	hasWS := req.WorkspaceID != ""
	hasNS := req.NamespaceID != ""

	if !hasWS && !hasNS {
		errs = append(errs, validation.FieldError{Field: "workspaceId/namespaceId", Message: "one of workspaceId or namespaceId is required"})
	}
	if hasWS && hasNS {
		errs = append(errs, validation.FieldError{Field: "workspaceId/namespaceId", Message: "only one of workspaceId or namespaceId can be specified"})
	}

	return errs
}

// ValidateBindEnvironmentRequest validates a bind-environment request.
func ValidateBindEnvironmentRequest(req *BindEnvironmentRequest) validation.ErrorList {
	var errs validation.ErrorList

	if req.EnvironmentID == "" {
		errs = append(errs, validation.FieldError{Field: "environmentId", Message: "is required"})
	}

	return errs
}
