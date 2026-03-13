package o11y

import (
	"regexp"

	"lcp.io/lcp/lib/api/validation"
)

var (
	nameRegexp    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)
	validStatuses = map[string]bool{"active": true, "inactive": true}
)

func validateName(errs *validation.ErrorList, name string) {
	if name == "" {
		*errs = append(*errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
	} else if !nameRegexp.MatchString(name) {
		*errs = append(*errs, validation.FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens"})
	}
}

func validateStatus(errs *validation.ErrorList, status string) {
	if status != "" && !validStatuses[status] {
		*errs = append(*errs, validation.FieldError{Field: "spec.status", Message: "must be 'active' or 'inactive'"})
	}
}

func validateMetricsURL(errs *validation.ErrorList, url string) {
	if url == "" {
		*errs = append(*errs, validation.FieldError{Field: "spec.metricsUrl", Message: "is required"})
	}
}

// ValidateEndpointCreate validates an EndpointSpec for creation.
func ValidateEndpointCreate(name string, spec *EndpointSpec) validation.ErrorList {
	var errs validation.ErrorList
	validateName(&errs, name)
	validateMetricsURL(&errs, spec.MetricsURL)
	validateStatus(&errs, spec.Status)
	return errs
}

// ValidateEndpointUpdate validates an EndpointSpec for full update.
func ValidateEndpointUpdate(spec *EndpointSpec) validation.ErrorList {
	var errs validation.ErrorList
	validateMetricsURL(&errs, spec.MetricsURL)
	validateStatus(&errs, spec.Status)
	return errs
}
