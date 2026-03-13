package o11y

import (
	"lcp.io/lcp/lib/api/types"
	"lcp.io/lcp/lib/runtime"
	"lcp.io/lcp/pkg/db/generated"
)

// Endpoint
// +openapi:description=可观测性端点：配置 VictoriaMetrics 等监控系统的连接信息。
type Endpoint struct {
	runtime.TypeMeta `json:",inline"`
	types.ObjectMeta `json:"metadata"`
	Spec             EndpointSpec `json:"spec"`
}

func (e *Endpoint) GetTypeMeta() *runtime.TypeMeta { return &e.TypeMeta }

// EndpointSpec
// +openapi:description=端点属性：包含监控系统连接 URL 和状态信息。
type EndpointSpec struct {
	// +openapi:description=Metrics 查询地址（VictoriaMetrics）
	// +openapi:required
	MetricsURL string `json:"metricsUrl,omitempty"`
	// +openapi:description=端点状态
	// +openapi:enum=active,inactive
	Status string `json:"status,omitempty"`
}

// EndpointList
// +openapi:description=监控端点列表：分页返回的端点集合。
type EndpointList struct {
	runtime.TypeMeta `json:",inline"`
	Items            []Endpoint `json:"items"`
	TotalCount       int64      `json:"totalCount"`
}

func (e *EndpointList) GetTypeMeta() *runtime.TypeMeta { return &e.TypeMeta }

// DB type aliases
type DBEndpoint = generated.O11yEndpoint
