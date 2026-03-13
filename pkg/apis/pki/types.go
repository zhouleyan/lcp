package pki

import (
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db/generated"
)

// Certificate type constants.
const (
	CertTypeCA     = "ca"
	CertTypeServer = "server"
	CertTypeClient = "client"
	CertTypeBoth   = "both"
)

// DBCertificate is the database row type for certificates.
type DBCertificate = generated.Certificate

// Certificate
// +openapi:description=证书资源，支持 CA、服务端、客户端等类型
type Certificate struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             CertificateSpec   `json:"spec"`
	Status           CertificateStatus `json:"status,omitempty"`
}

func (c *Certificate) GetTypeMeta() *runtime.TypeMeta { return &c.TypeMeta }

// CertificateSpec holds the user-provided certificate parameters.
type CertificateSpec struct {
	// +openapi:required
	// +openapi:enum=ca,server,client,both
	// +openapi:description=证书类型
	CertType string `json:"certType"`

	// +openapi:description=通用名称（CA 类型必填）
	CommonName string `json:"commonName,omitempty"`

	// +openapi:description=SAN 域名列表（server/both 类型必填）
	DNSNames []string `json:"dnsNames,omitempty"`

	// +openapi:description=签发 CA 名称（非 CA 类型必填）
	CAName string `json:"caName,omitempty"`

	// +openapi:description=有效期天数（0 或省略表示使用默认值：CA 为 3650 天，其他为 365 天）
	ValidityDays int `json:"validityDays,omitempty"`
}

// CertificateStatus holds read-only certificate metadata filled by the system.
type CertificateStatus struct {
	// +openapi:description=证书序列号
	SerialNumber string `json:"serialNumber"`
	// +openapi:description=证书生效时间
	NotBefore string `json:"notBefore"`
	// +openapi:description=证书过期时间
	NotAfter string `json:"notAfter"`
	// +openapi:description=证书 PEM（公开，不含私钥）
	Certificate string `json:"certificate"`
}

// CertificateList is a list of certificates.
type CertificateList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Certificate `json:"items"`
	TotalCount       int64         `json:"totalCount"`
}

func (c *CertificateList) GetTypeMeta() *runtime.TypeMeta { return &c.TypeMeta }
