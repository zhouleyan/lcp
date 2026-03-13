package pki

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestGenerateCA(t *testing.T) {
	certPEM, keyPEM, err := GenerateCA("Test CA", 3650)
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}

	// Verify cert is valid PEM
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("expected PEM CERTIFICATE block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	if !cert.IsCA {
		t.Fatal("expected IsCA=true")
	}
	if cert.Subject.CommonName != "Test CA" {
		t.Fatalf("CN=%q, want %q", cert.Subject.CommonName, "Test CA")
	}

	// Verify key is valid PEM
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "EC PRIVATE KEY" && keyBlock.Type != "PRIVATE KEY" {
		t.Fatalf("expected PEM private key block, got %q", keyBlock.Type)
	}
}

func TestIssueCertificate(t *testing.T) {
	caCertPEM, caKeyPEM, err := GenerateCA("Test CA", 3650)
	if err != nil {
		t.Fatal(err)
	}

	certPEM, keyPEM, serialNumber, err := IssueCertificate(IssueRequest{
		CACertPEM:    caCertPEM,
		CAKeyPEM:     caKeyPEM,
		DNSNames:     []string{"example.com", "*.example.com"},
		CertType:     "server",
		ValidityDays: 365,
	})
	if err != nil {
		t.Fatalf("IssueCertificate: %v", err)
	}

	if serialNumber == "" {
		t.Fatal("expected non-empty serial number")
	}

	// Parse and verify
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	if cert.IsCA {
		t.Fatal("expected IsCA=false")
	}
	if len(cert.DNSNames) != 2 {
		t.Fatalf("expected 2 DNS names, got %d", len(cert.DNSNames))
	}
	if cert.ExtKeyUsage[0] != x509.ExtKeyUsageServerAuth {
		t.Fatal("expected ServerAuth ExtKeyUsage")
	}

	// Verify key is valid
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		t.Fatal("expected PEM private key block")
	}

	// Verify cert is signed by CA
	caBlock, _ := pem.Decode(caCertPEM)
	caCert, _ := x509.ParseCertificate(caBlock.Bytes)
	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Fatalf("certificate verification failed: %v", err)
	}
}

func TestIssueCertificateClientType(t *testing.T) {
	caCertPEM, caKeyPEM, _ := GenerateCA("Test CA", 3650)

	certPEM, _, _, err := IssueCertificate(IssueRequest{
		CACertPEM:    caCertPEM,
		CAKeyPEM:     caKeyPEM,
		DNSNames:     nil,
		CertType:     "client",
		ValidityDays: 365,
	})
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(certPEM)
	cert, _ := x509.ParseCertificate(block.Bytes)

	if cert.ExtKeyUsage[0] != x509.ExtKeyUsageClientAuth {
		t.Fatal("expected ClientAuth ExtKeyUsage")
	}
}

func TestIssueCertificateBothType(t *testing.T) {
	caCertPEM, caKeyPEM, _ := GenerateCA("Test CA", 3650)

	certPEM, _, _, err := IssueCertificate(IssueRequest{
		CACertPEM:    caCertPEM,
		CAKeyPEM:     caKeyPEM,
		DNSNames:     []string{"example.com"},
		CertType:     "both",
		ValidityDays: 365,
	})
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(certPEM)
	cert, _ := x509.ParseCertificate(block.Bytes)

	hasServer, hasClient := false, false
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			hasServer = true
		}
		if usage == x509.ExtKeyUsageClientAuth {
			hasClient = true
		}
	}
	if !hasServer || !hasClient {
		t.Fatal("expected both ServerAuth and ClientAuth")
	}
}
