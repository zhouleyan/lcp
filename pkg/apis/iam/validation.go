package iam

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"lcp.io/lcp/lib/api/validation"
)

// seg matches a permission code segment: must start with lowercase, allows camelCase for verbs like "deleteCollection".
const seg = `[a-z][a-zA-Z0-9-]*`

var (
	usernameRegexp      = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	phoneRegexp         = regexp.MustCompile(`^1[3-9]\d{9}$`)
	workspaceNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)
	namespaceNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)
	passwordUpperRegexp = regexp.MustCompile(`[A-Z]`)
	passwordLowerRegexp = regexp.MustCompile(`[a-z]`)
	passwordDigitRegexp = regexp.MustCompile(`[0-9]`)

	// validPatternRegexp validates permission rule patterns:
	//   *:*                          → match all
	//   iam:*  iam:namespaces:*      → prefix wildcard
	//   *:list  *:deleteCollection   → suffix wildcard
	//   iam:users:list               → exact match (at least 2 segments)
	validPatternRegexp = regexp.MustCompile(
		`^(\*:\*` +
			`|(\*|` + seg + `)((:` + seg + `)*):\*` +
			`|\*:` + seg + `((:` + seg + `)*)` +
			`|` + seg + `(:` + seg + `)+)$`,
	)
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

// ValidateWorkspaceUpdate validates workspace fields for update.
func ValidateWorkspaceUpdate(spec *WorkspaceSpec) validation.ErrorList {
	var errs validation.ErrorList
	if spec.Status != "" && spec.Status != "active" && spec.Status != "inactive" {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}
	return errs
}

// ValidateNamespaceUpdate validates namespace fields for update.
func ValidateNamespaceUpdate(spec *NamespaceSpec) validation.ErrorList {
	var errs validation.ErrorList
	if spec.Visibility != "" && spec.Visibility != "public" && spec.Visibility != "private" {
		errs = append(errs, validation.FieldError{Field: "spec.visibility", Message: "must be 'public' or 'private'"})
	}
	if spec.MaxMembers < 0 {
		errs = append(errs, validation.FieldError{Field: "spec.maxMembers", Message: "must be >= 0"})
	}
	if spec.Status != "" && spec.Status != "active" && spec.Status != "inactive" {
		errs = append(errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}
	return errs
}

var (
	roleNameRegexp  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)
	validRoleScopes = map[string]bool{ScopePlatform: true, ScopeWorkspace: true, ScopeNamespace: true}
)

// ValidateRoleCreate validates a RoleSpec for creation.
func ValidateRoleCreate(spec *RoleSpec) validation.ErrorList {
	var errs validation.ErrorList
	if spec.Name == "" {
		errs = append(errs, validation.FieldError{Field: "spec.name", Message: "is required"})
	} else if !roleNameRegexp.MatchString(spec.Name) {
		errs = append(errs, validation.FieldError{Field: "spec.name", Message: "must match ^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$"})
	}
	if !validRoleScopes[spec.Scope] {
		errs = append(errs, validation.FieldError{Field: "spec.scope", Message: "must be platform, workspace, or namespace"})
	}
	if len(spec.Rules) == 0 {
		errs = append(errs, validation.FieldError{Field: "spec.rules", Message: "must not be empty"})
	}
	for i, rule := range spec.Rules {
		if ruleErrs := ValidatePermissionPattern(rule); ruleErrs.HasErrors() {
			for _, e := range ruleErrs {
				errs = append(errs, validation.FieldError{
					Field:   fmt.Sprintf("spec.rules[%d]", i),
					Message: e.Message,
				})
			}
		}
	}
	return errs
}

// ValidateRoleUpdate validates a RoleSpec for update.
func ValidateRoleUpdate(spec *RoleSpec) validation.ErrorList {
	var errs validation.ErrorList
	if len(spec.Rules) == 0 {
		errs = append(errs, validation.FieldError{Field: "spec.rules", Message: "must not be empty"})
	}
	for i, rule := range spec.Rules {
		if ruleErrs := ValidatePermissionPattern(rule); ruleErrs.HasErrors() {
			for _, e := range ruleErrs {
				errs = append(errs, validation.FieldError{
					Field:   fmt.Sprintf("spec.rules[%d]", i),
					Message: e.Message,
				})
			}
		}
	}
	return errs
}

// scopeLevel returns numeric level for scope comparison.
func scopeLevel(scope string) int {
	switch scope {
	case "platform":
		return 0
	case "workspace":
		return 1
	case "namespace":
		return 2
	default:
		return -1
	}
}

// ValidateRuleScopes checks that all permission rules are within the allowed scope.
func ValidateRuleScopes(roleScope string, rules []string, permissionsByCode map[string]string) validation.ErrorList {
	var errs validation.ErrorList
	minLevel := scopeLevel(roleScope)
	if minLevel <= 0 {
		return errs // platform roles can have any permissions
	}

	for i, rule := range rules {
		if rule == "*:*" {
			errs = append(errs, validation.FieldError{
				Field:   fmt.Sprintf("spec.rules[%d]", i),
				Message: fmt.Sprintf("pattern %s includes platform-level permissions, not allowed for %s role", rule, roleScope),
			})
			continue
		}

		if strings.Contains(rule, "*") {
			for code, permScope := range permissionsByCode {
				if MatchPermission(rule, code) {
					if scopeLevel(permScope) < minLevel {
						errs = append(errs, validation.FieldError{
							Field:   fmt.Sprintf("spec.rules[%d]", i),
							Message: fmt.Sprintf("pattern %s matches %s-scoped permission %s, not allowed for %s role", rule, permScope, code, roleScope),
						})
						break
					}
				}
			}
		} else {
			if permScope, ok := permissionsByCode[rule]; ok {
				if scopeLevel(permScope) < minLevel {
					errs = append(errs, validation.FieldError{
						Field:   fmt.Sprintf("spec.rules[%d]", i),
						Message: fmt.Sprintf("permission %s is %s-scoped, not allowed for %s role", rule, permScope, roleScope),
					})
				}
			}
		}
	}
	return errs
}

// ValidatePermissionPattern validates a permission rule pattern string.
func ValidatePermissionPattern(pattern string) validation.ErrorList {
	var errs validation.ErrorList
	if pattern == "" {
		errs = append(errs, validation.FieldError{Field: "pattern", Message: "cannot be empty"})
	} else if !validPatternRegexp.MatchString(pattern) {
		errs = append(errs, validation.FieldError{Field: "pattern", Message: "invalid pattern: " + pattern})
	}
	return errs
}

// ValidateRoleBindingCreate validates a RoleBindingSpec for creation.
func ValidateRoleBindingCreate(spec *RoleBindingSpec) validation.ErrorList {
	var errs validation.ErrorList
	if spec.UserID == "" {
		errs = append(errs, validation.FieldError{Field: "spec.userId", Message: "is required"})
	}
	if spec.RoleID == "" {
		errs = append(errs, validation.FieldError{Field: "spec.roleId", Message: "is required"})
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
