package pki

import (
	"net"
	"regexp"

	"lcp.io/lcp/lib/api/validation"
)

var (
	nameRegexp     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)
	validCertTypes = map[string]bool{
		CertTypeCA: true, CertTypeServer: true,
		CertTypeClient: true, CertTypeBoth: true,
	}
)

// ValidateCertificateCreate validates a CertificateSpec for creation.
func ValidateCertificateCreate(name string, spec *CertificateSpec) validation.ErrorList {
	var errs validation.ErrorList

	// name
	if name == "" {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "is required"})
	} else if !nameRegexp.MatchString(name) {
		errs = append(errs, validation.FieldError{Field: "metadata.name", Message: "must be 3-50 lowercase alphanumeric characters or hyphens"})
	}

	// certType
	if !validCertTypes[spec.CertType] {
		errs = append(errs, validation.FieldError{Field: "spec.certType", Message: "must be ca, server, client, or both"})
		return errs // can't validate further without valid type
	}

	// validityDays
	if spec.ValidityDays < 0 {
		errs = append(errs, validation.FieldError{Field: "spec.validityDays", Message: "must be positive"})
	}
	if spec.ValidityDays > 36500 {
		errs = append(errs, validation.FieldError{Field: "spec.validityDays", Message: "must not exceed 36500 (100 years)"})
	}

	switch spec.CertType {
	case CertTypeCA:
		if spec.CommonName == "" {
			errs = append(errs, validation.FieldError{Field: "spec.commonName", Message: "is required for CA type"})
		}
		if spec.CAName != "" {
			errs = append(errs, validation.FieldError{Field: "spec.caName", Message: "must be empty for CA type"})
		}
	case CertTypeServer, CertTypeBoth:
		if spec.CAName == "" {
			errs = append(errs, validation.FieldError{Field: "spec.caName", Message: "is required for non-CA type"})
		}
		if len(spec.DNSNames) == 0 && len(spec.IPAddresses) == 0 {
			errs = append(errs, validation.FieldError{Field: "spec.dnsNames", Message: "dnsNames or ipAddresses is required for server/both type"})
		}
	case CertTypeClient:
		if spec.CAName == "" {
			errs = append(errs, validation.FieldError{Field: "spec.caName", Message: "is required for non-CA type"})
		}
	}

	// Validate IP address format
	for _, ip := range spec.IPAddresses {
		if net.ParseIP(ip) == nil {
			errs = append(errs, validation.FieldError{Field: "spec.ipAddresses", Message: "invalid IP address: " + ip})
		}
	}

	return errs
}
