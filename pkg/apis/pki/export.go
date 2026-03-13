package pki

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
)

// NewExportHandler creates a GET handler for downloading certificate files.
// Supports: cert.pem, key.pem, ca.pem, all.zip
//
// +openapi:action=export
// +openapi:resource=Certificate
// +openapi:summary=下载证书文件
func NewExportHandler(store CertificateStore, encryptionKey []byte) rest.HandlerFunc {
	return func(ctx context.Context, params map[string]string, body []byte) (runtime.Object, error) {
		idStr := params["certificateId"]
		id, err := rest.ParseID(idStr)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid certificate ID: %s", idStr), nil)
		}

		fileName := params["file"]
		if fileName == "" {
			return nil, apierrors.NewBadRequest("missing 'file' query parameter", nil)
		}

		row, err := store.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}

		// all.zip: bundle all files into a zip archive
		if fileName == "all.zip" {
			return buildZipExport(ctx, store, row, encryptionKey)
		}

		// Validate single-file name
		base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		ext := filepath.Ext(fileName)
		if !isValidFilePrefix(base) || !isValidFileExt(ext) {
			return nil, apierrors.NewBadRequest(
				fmt.Sprintf("unsupported file: %s (use cert.pem, key.pem, ca.pem, or all.zip)", fileName), nil)
		}

		var data []byte
		switch base {
		case "cert":
			data = row.Certificate
		case "key":
			data, err = Decrypt(row.PrivateKey, encryptionKey)
			if err != nil {
				return nil, apierrors.NewInternalError(fmt.Errorf("decrypt private key: %w", err))
			}
		case "ca":
			if row.CaName != nil {
				caCert, caErr := store.GetByName(ctx, *row.CaName)
				if caErr != nil {
					return nil, apierrors.NewInternalError(fmt.Errorf("load CA certificate: %w", caErr))
				}
				data = caCert.Certificate
			} else {
				data = row.Certificate
			}
		}

		return &rest.FileResponse{
			FileName:    buildDownloadName(row.Name, row.CertType, base, ext, row.CaName),
			ContentType: "application/x-pem-file",
			Data:        data,
		}, nil
	}
}

// buildDownloadName constructs the download filename with type suffix.
// CA:     cert → {name}.pem, key → {name}-key.pem
// Others: cert → {name}-{type}.pem, key → {name}-{type}-key.pem, ca → {caName}.pem
func buildDownloadName(name, certType, base, ext string, caName *string) string {
	suffix := certTypeSuffix(certType)
	switch base {
	case "cert":
		if suffix != "" {
			return name + "-" + suffix + ext
		}
		return name + ext
	case "key":
		if suffix != "" {
			return name + "-" + suffix + "-key" + ext
		}
		return name + "-key" + ext
	case "ca":
		if caName != nil {
			return *caName + ext
		}
		return name + ext
	}
	return name + ext
}

// buildZipExport creates a zip archive containing all certificate files.
func buildZipExport(ctx context.Context, store CertificateStore, row *DBCertificate, encryptionKey []byte) (*rest.FileResponse, error) {
	suffix := certTypeSuffix(row.CertType)
	files := make(map[string][]byte)

	// cert
	certName := row.Name + ".pem"
	if suffix != "" {
		certName = row.Name + "-" + suffix + ".pem"
	}
	files[certName] = row.Certificate

	// key
	keyPEM, err := Decrypt(row.PrivateKey, encryptionKey)
	if err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("decrypt private key: %w", err))
	}
	keyName := row.Name + "-key.pem"
	if suffix != "" {
		keyName = row.Name + "-" + suffix + "-key.pem"
	}
	files[keyName] = keyPEM

	// ca (only for non-CA certs)
	if row.CaName != nil {
		caCert, err := store.GetByName(ctx, *row.CaName)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("load CA certificate: %w", err))
		}
		files[*row.CaName+".pem"] = caCert.Certificate
	}

	// Build zip
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		fw, err := zw.Create(name)
		if err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("create zip entry: %w", err))
		}
		if _, err := fw.Write(data); err != nil {
			return nil, apierrors.NewInternalError(fmt.Errorf("write zip entry: %w", err))
		}
	}
	if err := zw.Close(); err != nil {
		return nil, apierrors.NewInternalError(fmt.Errorf("close zip: %w", err))
	}

	return &rest.FileResponse{
		FileName:    row.Name + ".zip",
		ContentType: "application/zip",
		Data:        buf.Bytes(),
	}, nil
}

func isValidFilePrefix(prefix string) bool {
	switch prefix {
	case "cert", "key", "ca":
		return true
	}
	return false
}

func isValidFileExt(ext string) bool {
	return ext == ".pem"
}

// certTypeSuffix returns the download filename suffix for a given cert type.
func certTypeSuffix(certType string) string {
	switch certType {
	case CertTypeServer:
		return "server"
	case CertTypeClient:
		return "client"
	case CertTypeBoth:
		return "peer"
	default:
		return ""
	}
}
