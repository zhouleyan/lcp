package pki

import "testing"

func TestValidateCertificateCreate_CA(t *testing.T) {
	errs := ValidateCertificateCreate("my-ca", &CertificateSpec{
		CertType:     CertTypeCA,
		CommonName:   "My CA",
		ValidityDays: 3650,
	})
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateCertificateCreate_CA_MissingCommonName(t *testing.T) {
	errs := ValidateCertificateCreate("my-ca", &CertificateSpec{
		CertType: CertTypeCA,
	})
	if !errs.HasErrors() {
		t.Fatal("expected error for missing commonName")
	}
}

func TestValidateCertificateCreate_CA_WithCAName(t *testing.T) {
	errs := ValidateCertificateCreate("my-ca", &CertificateSpec{
		CertType:   CertTypeCA,
		CommonName: "My CA",
		CAName:     "other-ca",
	})
	if !errs.HasErrors() {
		t.Fatal("expected error: CA type should not have caName")
	}
}

func TestValidateCertificateCreate_Server(t *testing.T) {
	errs := ValidateCertificateCreate("my-cert", &CertificateSpec{
		CertType: CertTypeServer,
		CAName:   "my-ca",
		DNSNames: []string{"example.com"},
	})
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateCertificateCreate_Server_MissingCAName(t *testing.T) {
	errs := ValidateCertificateCreate("my-cert", &CertificateSpec{
		CertType: CertTypeServer,
		DNSNames: []string{"example.com"},
	})
	if !errs.HasErrors() {
		t.Fatal("expected error for missing caName")
	}
}

func TestValidateCertificateCreate_Server_MissingDNSNames(t *testing.T) {
	errs := ValidateCertificateCreate("my-cert", &CertificateSpec{
		CertType: CertTypeServer,
		CAName:   "my-ca",
	})
	if !errs.HasErrors() {
		t.Fatal("expected error for missing dnsNames")
	}
}

func TestValidateCertificateCreate_InvalidName(t *testing.T) {
	errs := ValidateCertificateCreate("A", &CertificateSpec{
		CertType:   CertTypeCA,
		CommonName: "My CA",
	})
	if !errs.HasErrors() {
		t.Fatal("expected error for invalid name")
	}
}

func TestValidateCertificateCreate_InvalidCertType(t *testing.T) {
	errs := ValidateCertificateCreate("my-cert", &CertificateSpec{
		CertType: "unknown",
	})
	if !errs.HasErrors() {
		t.Fatal("expected error for invalid certType")
	}
}

func TestValidateCertificateCreate_ValidityDaysTooHigh(t *testing.T) {
	errs := ValidateCertificateCreate("my-ca", &CertificateSpec{
		CertType:     CertTypeCA,
		CommonName:   "My CA",
		ValidityDays: 50000,
	})
	if !errs.HasErrors() {
		t.Fatal("expected error for validityDays too high")
	}
}
