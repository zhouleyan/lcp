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
	// +openapi:description=端点描述
	Description string `json:"description,omitempty"`
	// +openapi:description=是否公开（公开端点对所有工作空间可见）
	IsPublic *bool `json:"isPublic,omitempty"`
	// +openapi:description=Metrics 查询地址（VictoriaMetrics）
	// +openapi:required
	MetricsURL string `json:"metricsUrl,omitempty"`
	// +openapi:description=Logs 查询地址
	LogsURL string `json:"logsUrl,omitempty"`
	// +openapi:description=Traces 查询地址
	TracesURL string `json:"tracesUrl,omitempty"`
	// +openapi:description=APM 查询地址
	ApmURL string `json:"apmUrl,omitempty"`
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

// ProbeResult 端点连通性检测结果。
// +openapi:description=端点连通性检测结果：对端点所有已配置 URL 的连通性检测。
type ProbeResult struct {
	runtime.TypeMeta `json:",inline"`
	// +openapi:description=各 URL 的检测结果
	Results []ProbeResultItem `json:"results"`
}

func (p *ProbeResult) GetTypeMeta() *runtime.TypeMeta { return &p.TypeMeta }

// ProbeResultItem 单个 URL 的检测结果。
// +openapi:description=单个 URL 的连通性检测结果。
type ProbeResultItem struct {
	// +openapi:description=字段名称（metricsUrl/logsUrl/tracesUrl/apmUrl）
	Field string `json:"field"`
	// +openapi:description=URL 地址
	URL string `json:"url"`
	// +openapi:description=是否连通
	Success bool `json:"success"`
	// +openapi:description=HTTP 状态码
	StatusCode int `json:"statusCode,omitempty"`
	// +openapi:description=失败阶段（dns/tcp/tls/http）
	Phase string `json:"phase,omitempty"`
	// +openapi:description=失败原因
	Message string `json:"message,omitempty"`
	// +openapi:description=检测耗时
	Duration string `json:"duration"`
}

// DB type aliases
type DBEndpoint = generated.O11yEndpoint
