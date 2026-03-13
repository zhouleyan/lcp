package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// IssueRequest holds the parameters for issuing a certificate.
type IssueRequest struct {
	CACertPEM    []byte
	CAKeyPEM     []byte
	DNSNames     []string
	CertType     string // server, client, both
	ValidityDays int
}

// GenerateCA creates a self-signed CA certificate.
// Returns PEM-encoded certificate and private key.
func GenerateCA(commonName string, validityDays int) (certPEM, keyPEM []byte, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	serialNumber, err := randomSerialNumber()
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(0, 0, validityDays),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("create certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal private key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, nil
}

// IssueCertificate creates a certificate signed by the given CA.
// Returns PEM-encoded certificate, private key, and serial number string.
func IssueCertificate(req IssueRequest) (certPEM, keyPEM []byte, serialNumberStr string, err error) {
	// Parse CA cert
	caBlock, _ := pem.Decode(req.CACertPEM)
	if caBlock == nil {
		return nil, nil, "", fmt.Errorf("invalid CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, "", fmt.Errorf("parse CA certificate: %w", err)
	}

	// Parse CA key
	caKeyBlock, _ := pem.Decode(req.CAKeyPEM)
	if caKeyBlock == nil {
		return nil, nil, "", fmt.Errorf("invalid CA private key PEM")
	}
	caKey, err := x509.ParseECPrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, nil, "", fmt.Errorf("parse CA private key: %w", err)
	}

	// Generate new key pair
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, "", fmt.Errorf("generate key: %w", err)
	}

	serialNumber, err := randomSerialNumber()
	if err != nil {
		return nil, nil, "", err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{},
		DNSNames:     req.DNSNames,
		NotBefore:    now,
		NotAfter:     now.AddDate(0, 0, req.ValidityDays),
	}

	// Set key usage based on cert type
	switch req.CertType {
	case "server":
		template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	case "client":
		template.KeyUsage = x509.KeyUsageDigitalSignature
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	case "both":
		template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	default:
		return nil, nil, "", fmt.Errorf("unsupported cert type: %s", req.CertType)
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &priv.PublicKey, caKey)
	if err != nil {
		return nil, nil, "", fmt.Errorf("create certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, "", fmt.Errorf("marshal private key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, serialNumber.Text(16), nil
}

func randomSerialNumber() (*big.Int, error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}
	return serialNumber, nil
}
