package pki

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/lib/rest"
	"lcp.io/lcp/lib/runtime"
)

// NewExportHandler creates a GET handler for downloading certificate files.
// Supports: cert.pem, key.pem, ca.pem
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

		// Validate file name
		base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		ext := filepath.Ext(fileName)
		if !isValidFilePrefix(base) || !isValidFileExt(ext) {
			return nil, apierrors.NewBadRequest(
				fmt.Sprintf("unsupported file: %s (use cert.pem, key.pem, or ca.pem)", fileName), nil)
		}

		row, err := store.GetByID(ctx, id)
		if err != nil {
			return nil, err
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
				// CA cert itself: ca file is just itself
				data = row.Certificate
			}
		}

		// Build download filename: cert → {name}.pem, key → {name}-key.pem, ca → {caName}.pem
		var downloadName string
		switch base {
		case "cert":
			downloadName = row.Name + ext
		case "key":
			downloadName = row.Name + "-key" + ext
		case "ca":
			if row.CaName != nil {
				downloadName = *row.CaName + ext
			} else {
				downloadName = row.Name + ext
			}
		}

		return &rest.FileResponse{
			FileName:    downloadName,
			ContentType: "application/x-pem-file",
			Data:        data,
		}, nil
	}
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
