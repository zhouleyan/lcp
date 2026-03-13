package o11y

import (
	"net/url"
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

// validateURL checks that a URL is valid http/https. If required is true, empty is an error.
func validateURL(errs *validation.ErrorList, field string, rawURL string, required bool) {
	if rawURL == "" {
		if required {
			*errs = append(*errs, validation.FieldError{Field: field, Message: "is required"})
		}
		return
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		*errs = append(*errs, validation.FieldError{Field: field, Message: "must be a valid URL (e.g. http://host:port)"})
		return
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		*errs = append(*errs, validation.FieldError{Field: field, Message: "must use http or https scheme"})
	}
}

func validateEndpointURLs(errs *validation.ErrorList, spec *EndpointSpec) {
	validateURL(errs, "spec.metricsUrl", spec.MetricsURL, true)
	validateURL(errs, "spec.logsUrl", spec.LogsURL, false)
	validateURL(errs, "spec.tracesUrl", spec.TracesURL, false)
	validateURL(errs, "spec.apmUrl", spec.ApmURL, false)
}

// ValidateEndpointCreate validates an EndpointSpec for creation.
func ValidateEndpointCreate(name string, spec *EndpointSpec) validation.ErrorList {
	var errs validation.ErrorList
	validateName(&errs, name)
	validateEndpointURLs(&errs, spec)
	validateStatus(&errs, spec.Status)
	return errs
}

// ValidateEndpointUpdate validates name and EndpointSpec for full update.
func ValidateEndpointUpdate(name string, spec *EndpointSpec) validation.ErrorList {
	var errs validation.ErrorList
	validateName(&errs, name)
	validateEndpointURLs(&errs, spec)
	validateStatus(&errs, spec.Status)
	return errs
}
