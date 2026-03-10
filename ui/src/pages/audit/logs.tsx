import { useCallback, useEffect, useState } from "react"
import { Eye, Filter, Search } from "lucide-react"
import { toast } from "sonner"
import type { DateRange } from "react-day-picker"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import { Separator } from "@/components/ui/separator"
import { DateRangePicker } from "@/components/ui/date-range-picker"
import { listAuditLogs } from "@/api/audit/logs"
import { ApiError, translateApiError } from "@/api/client"
import type { AuditLog, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"

function formatJsonDetail(detail: Record<string, unknown>): string {
  return JSON.stringify(detail, null, 2)
}

export default function AuditLogListPage() {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
  } = useListState({ defaultSortBy: "created_at", defaultSortOrder: "desc" })
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)

  // filters
  const [eventTypeFilter, setEventTypeFilter] = useState<string>("all")
  const [actionFilter, setActionFilter] = useState<string>("all")
  const [moduleFilter, setModuleFilter] = useState<string>("all")
  const [successFilter, setSuccessFilter] = useState<string>("all")
  const [dateRange, setDateRange] = useState<DateRange | undefined>()

  // detail sheet
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null)

  const fetchLogs = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (eventTypeFilter !== "all") params.eventType = eventTypeFilter
      if (actionFilter !== "all") params.action = actionFilter
      if (moduleFilter !== "all") params.module = moduleFilter
      if (successFilter !== "all") params.success = successFilter
      if (dateRange?.from) params.startTime = dateRange.from.toISOString()
      if (dateRange?.to) params.endTime = dateRange.to.toISOString()
      const data = await listAuditLogs(params)
      setLogs(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search, eventTypeFilter, actionFilter, moduleFilter, successFilter, dateRange])

  useEffect(() => { fetchLogs() }, [fetchLogs])
  useEffect(() => { setPage(1) }, [search, eventTypeFilter, actionFilter, moduleFilter, successFilter, dateRange, pageSize])

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t("audit.title")}</h1>
        <p className="text-muted-foreground text-sm">
          {t("audit.manage", { count: totalCount })}
        </p>
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("audit.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        <DateRangePicker
          value={dateRange}
          onChange={setDateRange}
          placeholder={t("audit.filter.dateRange")}
          resetLabel={t("common.reset")}
          className="h-9 w-auto"
        />
      </div>

      {/* table */}
      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("username")}
              >
                {t("audit.username")}
                <SortIcon field="username" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("audit.eventType")}
                      <Filter className={`h-3 w-3 ${eventTypeFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setEventTypeFilter("all")}>
                      {t("common.all")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setEventTypeFilter("api_operation")}>
                      {t("audit.eventType.api_operation")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setEventTypeFilter("authentication")}>
                      {t("audit.eventType.authentication")}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("audit.action")}
                      <Filter className={`h-3 w-3 ${actionFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setActionFilter("all")}>
                      {t("common.all")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setActionFilter("create")}>
                      {t("audit.action.create")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setActionFilter("update")}>
                      {t("audit.action.update")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setActionFilter("patch")}>
                      {t("audit.action.patch")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setActionFilter("delete")}>
                      {t("audit.action.delete")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setActionFilter("deleteCollection")}>
                      {t("audit.action.deleteCollection")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setActionFilter("login")}>
                      {t("audit.action.login")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setActionFilter("login_failed")}>
                      {t("audit.action.login_failed")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setActionFilter("token_refresh")}>
                      {t("audit.action.token_refresh")}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("resource_type")}
              >
                {t("audit.resourceType")}
                <SortIcon field="resource_type" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>
                <div className="flex items-center gap-0.5">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button className="inline-flex items-center gap-1 select-none">
                        {t("audit.module")}
                        <Filter className={`h-3 w-3 ${moduleFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="start">
                      <DropdownMenuItem onClick={() => setModuleFilter("all")}>
                        {t("common.all")}
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setModuleFilter("iam")}>
                        iam
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setModuleFilter("dashboard")}>
                        dashboard
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setModuleFilter("audit")}>
                        audit
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                  <button className="cursor-pointer select-none" onClick={() => handleSort("module")}>
                    <SortIcon field="module" sortBy={sortBy} sortOrder={sortOrder} />
                  </button>
                </div>
              </TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("audit.success")}
                      <Filter className={`h-3 w-3 ${successFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setSuccessFilter("all")}>
                      {t("common.all")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setSuccessFilter("true")}>
                      {t("audit.success.true")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setSuccessFilter("false")}>
                      {t("audit.success.false")}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("status_code")}
              >
                {t("audit.statusCode")}
                <SortIcon field="status_code" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("duration_ms")}
              >
                {t("audit.duration")}
                <SortIcon field="duration_ms" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("created_at")}
              >
                {t("audit.createdAt")}
                <SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="w-16">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 10 }).map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-4 w-20" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : logs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={10} className="text-muted-foreground py-8 text-center">
                  {t("audit.noData")}
                </TableCell>
              </TableRow>
            ) : (
              logs.map((log) => (
                <TableRow key={log.spec.id}>
                  <TableCell className="font-medium">{log.spec.username || "-"}</TableCell>
                  <TableCell>{t(`audit.eventType.${log.spec.eventType}`)}</TableCell>
                  <TableCell>{t(`audit.action.${log.spec.action}`)}</TableCell>
                  <TableCell>
                    {log.spec.resourceType
                      ? `${log.spec.resourceType}${log.spec.resourceId ? `/${log.spec.resourceId}` : ""}`
                      : "-"}
                  </TableCell>
                  <TableCell>{log.spec.module || "-"}</TableCell>
                  <TableCell>
                    <Badge variant={log.spec.success ? "default" : "destructive"}>
                      {log.spec.success ? t("audit.success.true") : t("audit.success.false")}
                    </Badge>
                  </TableCell>
                  <TableCell>{log.spec.statusCode ?? "-"}</TableCell>
                  <TableCell>{log.spec.durationMs != null ? `${log.spec.durationMs}ms` : "-"}</TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(log.spec.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => setSelectedLog(log)}
                      title={t("audit.viewDetail")}
                    >
                      <Eye className="h-3.5 w-3.5" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      {/* detail dialog */}
      <Dialog open={!!selectedLog} onOpenChange={(v) => { if (!v) setSelectedLog(null) }}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>{t("audit.detail")}</DialogTitle>
            <DialogDescription>ID: {selectedLog?.spec.id}</DialogDescription>
          </DialogHeader>
          {selectedLog && (
            <div className="space-y-5">
              {/* Two-column: Basic + Resource */}
              <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                {/* Basic */}
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold">{t("audit.detail")}</h3>
                  <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
                    <dt className="text-muted-foreground">{t("audit.username")}</dt>
                    <dd className="font-medium">{selectedLog.spec.username || "-"}</dd>
                    <dt className="text-muted-foreground">{t("audit.userId")}</dt>
                    <dd>{selectedLog.spec.userId || "-"}</dd>
                    <dt className="text-muted-foreground">{t("audit.eventType")}</dt>
                    <dd>{t(`audit.eventType.${selectedLog.spec.eventType}`)}</dd>
                    <dt className="text-muted-foreground">{t("audit.action")}</dt>
                    <dd>{t(`audit.action.${selectedLog.spec.action}`)}</dd>
                    <dt className="text-muted-foreground">{t("audit.success")}</dt>
                    <dd>
                      <Badge variant={selectedLog.spec.success ? "default" : "destructive"}>
                        {selectedLog.spec.success ? t("audit.success.true") : t("audit.success.false")}
                      </Badge>
                    </dd>
                    <dt className="text-muted-foreground">{t("audit.createdAt")}</dt>
                    <dd>{new Date(selectedLog.spec.createdAt).toLocaleString()}</dd>
                  </dl>
                </div>

                {/* Resource */}
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold">{t("audit.resourceType")}</h3>
                  <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
                    <dt className="text-muted-foreground">{t("audit.resourceType")}</dt>
                    <dd>{selectedLog.spec.resourceType || "-"}</dd>
                    <dt className="text-muted-foreground">{t("audit.resourceId")}</dt>
                    <dd>{selectedLog.spec.resourceId || "-"}</dd>
                    <dt className="text-muted-foreground">{t("audit.module")}</dt>
                    <dd>{selectedLog.spec.module || "-"}</dd>
                    <dt className="text-muted-foreground">{t("audit.scope")}</dt>
                    <dd>{selectedLog.spec.scope}</dd>
                    <dt className="text-muted-foreground">{t("audit.workspaceId")}</dt>
                    <dd>{selectedLog.spec.workspaceId || "-"}</dd>
                    <dt className="text-muted-foreground">{t("audit.namespaceId")}</dt>
                    <dd>{selectedLog.spec.namespaceId || "-"}</dd>
                  </dl>
                </div>
              </div>

              <Separator />

              {/* HTTP section - full width, two-column grid */}
              <div className="space-y-3">
                <h3 className="text-sm font-semibold">HTTP</h3>
                <dl className="grid grid-cols-1 gap-x-8 gap-y-2 text-sm md:grid-cols-2">
                  <div className="grid grid-cols-[auto_1fr] gap-x-4">
                    <dt className="text-muted-foreground">{t("audit.httpMethod")}</dt>
                    <dd className="font-mono">{selectedLog.spec.httpMethod || "-"}</dd>
                  </div>
                  <div className="grid grid-cols-[auto_1fr] gap-x-4">
                    <dt className="text-muted-foreground">{t("audit.statusCode")}</dt>
                    <dd>
                      <span className={
                        (selectedLog.spec.statusCode ?? 0) < 400 ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"
                      }>
                        {selectedLog.spec.statusCode ?? "-"}
                      </span>
                    </dd>
                  </div>
                  <div className="col-span-full grid grid-cols-[auto_1fr] gap-x-4">
                    <dt className="text-muted-foreground">{t("audit.httpPath")}</dt>
                    <dd className="break-all font-mono">{selectedLog.spec.httpPath || "-"}</dd>
                  </div>
                  <div className="grid grid-cols-[auto_1fr] gap-x-4">
                    <dt className="text-muted-foreground">{t("audit.duration")}</dt>
                    <dd>{selectedLog.spec.durationMs != null ? `${selectedLog.spec.durationMs}ms` : "-"}</dd>
                  </div>
                  <div className="grid grid-cols-[auto_1fr] gap-x-4">
                    <dt className="text-muted-foreground">{t("audit.clientIp")}</dt>
                    <dd className="font-mono">{selectedLog.spec.clientIp || "-"}</dd>
                  </div>
                  <div className="col-span-full grid grid-cols-[auto_1fr] gap-x-4">
                    <dt className="text-muted-foreground">{t("audit.userAgent")}</dt>
                    <dd className="break-all">{selectedLog.spec.userAgent || "-"}</dd>
                  </div>
                </dl>
              </div>

              {/* Request Body */}
              {selectedLog.spec.detail && (
                <>
                  <Separator />
                  <div className="space-y-3">
                    <h3 className="text-sm font-semibold">{t("audit.detail.field")}</h3>
                    <pre className="max-h-80 overflow-auto rounded-md border bg-muted/50 p-4 font-mono text-xs leading-relaxed">
                      {formatJsonDetail(selectedLog.spec.detail)}
                    </pre>
                  </div>
                </>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
