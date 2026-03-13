import type { Messages } from "../../types"

const o11y: Messages = {
  "endpoint.title": "监控端点",
  "endpoint.create": "创建端点",
  "endpoint.edit": "编辑端点",
  "endpoint.noData": "暂无监控端点。",
  "endpoint.deleteConfirm": "确定要删除端点 \"{name}\" 吗？此操作不可撤销。",
  "endpoint.deleteSelected": "删除选中",
  "endpoint.batchDeleteConfirm": "确定要删除选中的 {count} 个端点吗？此操作不可撤销。",
  "endpoint.visibility": "可见性",
  "endpoint.public": "公开",
  "endpoint.private": "私有",
  "endpoint.endpoints": "端点",
  "endpoint.metricsUrl": "Metrics URL",
  "endpoint.logsUrl": "Logs URL",
  "endpoint.tracesUrl": "Traces URL",
  "endpoint.apmUrl": "APM URL",
  "endpoint.metricsUrlPlaceholder": "例如 http://victoria-metrics:8428",
  "endpoint.urlPlaceholder": "例如 http://host:port",
  "endpoint.urlInvalid": "请输入有效的 URL（如 http://host:port）",
  "endpoint.metricsLabel": "Metrics",
  "endpoint.logsLabel": "Logs",
  "endpoint.tracesLabel": "Traces",
  "endpoint.apmLabel": "APM",
  "endpoint.searchPlaceholder": "搜索名称、描述、Metrics URL...",
  "perm.group.o11y": "可观测性",
  "perm.group.o11y.endpoints": "监控端点",
}

export default o11y
