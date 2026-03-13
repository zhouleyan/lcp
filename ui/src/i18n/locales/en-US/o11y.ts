import type { Messages } from "../../types"

const o11y: Messages = {
  "endpoint.title": "Monitoring Endpoints",
  "endpoint.create": "Create Endpoint",
  "endpoint.edit": "Edit Endpoint",
  "endpoint.noData": "No monitoring endpoints yet.",
  "endpoint.deleteConfirm": "Are you sure you want to delete endpoint \"{name}\"? This action cannot be undone.",
  "endpoint.deleteSelected": "Delete Selected",
  "endpoint.batchDeleteConfirm": "Are you sure you want to delete {count} selected endpoints? This action cannot be undone.",
  "endpoint.visibility": "Visibility",
  "endpoint.public": "Public",
  "endpoint.private": "Private",
  "endpoint.endpoints": "Endpoints",
  "endpoint.metricsUrl": "Metrics URL",
  "endpoint.logsUrl": "Logs URL",
  "endpoint.tracesUrl": "Traces URL",
  "endpoint.apmUrl": "APM URL",
  "endpoint.metricsUrlPlaceholder": "e.g. http://victoria-metrics:8428",
  "endpoint.urlPlaceholder": "e.g. http://host:port",
  "endpoint.urlInvalid": "Please enter a valid URL (e.g. http://host:port)",
  "endpoint.metricsLabel": "Metrics",
  "endpoint.logsLabel": "Logs",
  "endpoint.tracesLabel": "Traces",
  "endpoint.apmLabel": "APM",
  "endpoint.searchPlaceholder": "Search name, description, Metrics URL...",
  "perm.group.o11y": "Observability",
  "perm.group.o11y.endpoints": "Monitoring Endpoints",
}

export default o11y
