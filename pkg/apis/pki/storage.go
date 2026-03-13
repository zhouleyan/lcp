package pki

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"strconv"
	"time"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db"
)

var (
	pemDecode            = pem.Decode
	x509ParseCertificate = x509.ParseCertificate
)

type certificateStorage struct {
	store         CertificateStore
	encryptionKey []byte
}

// NewCertificateStorage creates a REST storage for certificates.
// Certificates are immutable — Update and Patch are not supported.
func NewCertificateStorage(store CertificateStore, encryptionKey []byte) rest.Storage {
	return &certificateStorage{store: store, encryptionKey: encryptionKey}
}

func (s *certificateStorage) NewObject() runtime.Object { return &Certificate{} }

// +openapi:summary=获取证书详情
func (s *certificateStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id, err := rest.ParseID(options.PathParams["certificateId"])
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid certificate ID: %s", options.PathParams["certificateId"]), nil)
	}

	row, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return dbToAPI(row), nil
}

// +openapi:summary=获取证书列表
func (s *certificateStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)

	result, err := s.store.List(ctx, query)
	if err != nil {
		return nil, err
	}

	items := make([]Certificate, len(result.Items))
	for i := range result.Items {
		items[i] = *dbToAPI(&result.Items[i])
	}

	return &CertificateList{
		TypeMeta:   runtime.TypeMeta{Kind: "CertificateList"},
		Items:      items,
		TotalCount: result.TotalCount,
	}, nil
}

// +openapi:summary=创建证书
func (s *certificateStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	cert, ok := obj.(*Certificate)
	if !ok {
		return nil, fmt.Errorf("expected *Certificate, got %T", obj)
	}

	if errs := ValidateCertificateCreate(cert.ObjectMeta.Name, &cert.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}

	if options.DryRun {
		return cert, nil
	}

	var certPEM, keyPEM []byte
	var serialNumber string
	var notBefore, notAfter time.Time

	validityDays := cert.Spec.ValidityDays

	switch cert.Spec.CertType {
	case CertTypeCA:
		if validityDays == 0 {
			validityDays = 3650
		}
		var err error
		certPEM, keyPEM, err = GenerateCA(cert.Spec.CommonName, validityDays)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("generate CA: %w", err))
		}
		serialNumber, notBefore, notAfter, err = parseCertMeta(certPEM)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("parse generated CA cert: %w", err))
		}

	default:
		if validityDays == 0 {
			validityDays = 365
		}
		// Load CA
		caCert, err := s.store.GetByName(ctx, cert.Spec.CAName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, apierrors.NewBadRequest(fmt.Sprintf("CA %q not found", cert.Spec.CAName), nil)
			}
			return nil, err
		}
		if caCert.CertType != CertTypeCA {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("%q is not a CA certificate", cert.Spec.CAName), nil)
		}

		// Decrypt CA private key
		caKeyPEM, err := Decrypt(caCert.PrivateKey, s.encryptionKey)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("decrypt CA key: %w", err))
		}

		// Parse IP strings to net.IP
		var ips []net.IP
		for _, s := range cert.Spec.IPAddresses {
			ips = append(ips, net.ParseIP(s))
		}

		certPEM, keyPEM, serialNumber, err = IssueCertificate(IssueRequest{
			CACertPEM:    caCert.Certificate,
			CAKeyPEM:     caKeyPEM,
			DNSNames:     cert.Spec.DNSNames,
			IPAddresses:  ips,
			CertType:     cert.Spec.CertType,
			ValidityDays: validityDays,
		})
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("issue certificate: %w", err))
		}
		_, notBefore, notAfter, err = parseCertMeta(certPEM)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("parse issued cert: %w", err))
		}
	}

	// Encrypt private key before storage
	encryptedKey, err := Encrypt(keyPEM, s.encryptionKey)
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("encrypt private key: %w", err))
	}

	caName := cert.Spec.CAName
	var caNamePtr *string
	if caName != "" {
		caNamePtr = &caName
	}

	dnsNames := cert.Spec.DNSNames
	if dnsNames == nil {
		dnsNames = []string{}
	}
	ipAddresses := cert.Spec.IPAddresses
	if ipAddresses == nil {
		ipAddresses = []string{}
	}

	row, err := s.store.Create(ctx, &DBCertificate{
		Name:         cert.ObjectMeta.Name,
		CertType:     cert.Spec.CertType,
		CommonName:   cert.Spec.CommonName,
		DnsNames:     dnsNames,
		IpAddresses:  ipAddresses,
		CaName:       caNamePtr,
		SerialNumber: serialNumber,
		Certificate:  certPEM,
		PrivateKey:   encryptedKey,
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	})
	if err != nil {
		return nil, err
	}

	return dbToAPI(row), nil
}

// +openapi:summary=删除证书
func (s *certificateStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	if options.DryRun {
		return nil
	}

	id, err := rest.ParseID(options.PathParams["certificateId"])
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid certificate ID: %s", options.PathParams["certificateId"]), nil)
	}

	// If it's a CA, check for dependent certificates
	row, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if row.CertType == CertTypeCA {
		count, err := s.store.CountByCAName(ctx, row.Name)
		if err != nil {
			return err
		}
		if count > 0 {
			return apierrors.NewBadRequest(
				fmt.Sprintf("cannot delete CA %q: %d certificate(s) depend on it", row.Name, count), nil)
		}
	}

	return s.store.Delete(ctx, id)
}

// +openapi:summary=批量删除证书
func (s *certificateStorage) DeleteCollection(ctx context.Context, ids []string, options *rest.DeleteOptions) (*rest.DeletionResult, error) {
	if options.DryRun {
		return &rest.DeletionResult{SuccessCount: len(ids)}, nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		parsed, err := rest.ParseID(id)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid ID: %s", id), nil)
		}
		int64IDs = append(int64IDs, parsed)
	}

	count, err := s.store.DeleteByIDs(ctx, int64IDs)
	if err != nil {
		return nil, err
	}

	return &rest.DeletionResult{
		SuccessCount: int(count),
		FailedCount:  len(ids) - int(count),
	}, nil
}

// --- helpers ---

func restOptionsToListQuery(options *rest.ListOptions) db.ListQuery {
	query := db.ListQuery{
		Filters: make(map[string]any),
		Pagination: db.Pagination{
			Page:     options.Pagination.Page,
			PageSize: options.Pagination.PageSize,
		},
	}
	for k, v := range options.Filters {
		query.Filters[k] = v
	}
	if options.SortBy != "" {
		query.SortBy = options.SortBy
	}
	if options.SortOrder != "" {
		query.SortOrder = string(options.SortOrder)
	}
	return query
}

func dbToAPI(row *DBCertificate) *Certificate {
	cert := &Certificate{
		TypeMeta: runtime.TypeMeta{Kind: "Certificate"},
		ObjectMeta: types.ObjectMeta{
			ID:        strconv.FormatInt(row.ID, 10),
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
		Spec: CertificateSpec{
			CertType:   row.CertType,
			CommonName: row.CommonName,
		},
		Status: CertificateStatus{
			SerialNumber: row.SerialNumber,
			NotBefore:    row.NotBefore.Format(time.RFC3339),
			NotAfter:     row.NotAfter.Format(time.RFC3339),
			Certificate:  string(row.Certificate),
		},
	}
	if len(row.DnsNames) > 0 {
		cert.Spec.DNSNames = row.DnsNames
	}
	if len(row.IpAddresses) > 0 {
		cert.Spec.IPAddresses = row.IpAddresses
	}
	if row.CaName != nil {
		cert.Spec.CAName = *row.CaName
	}
	return cert
}

func parseCertMeta(certPEM []byte) (serialNumber string, notBefore, notAfter time.Time, err error) {
	block, _ := pemDecode(certPEM)
	if block == nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("failed to decode PEM block")
	}
	cert, err := x509ParseCertificate(block.Bytes)
	if err != nil {
		return "", time.Time{}, time.Time{}, fmt.Errorf("parse certificate: %w", err)
	}
	return cert.SerialNumber.Text(16), cert.NotBefore, cert.NotAfter, nil
}
