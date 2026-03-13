import { useCallback, useEffect, useState } from "react"
import { Link, useParams } from "react-router"
import { Plus, Pencil, Trash2, Search, Filter, Unlink, Link2, X } from "lucide-react"
import { useForm, useFieldArray } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  listHosts, listWorkspaceHosts, listNamespaceHosts,
  createHost, createWorkspaceHost, createNamespaceHost,
  updateHost, updateWorkspaceHost, updateNamespaceHost,
  deleteHost, deleteHosts,
  deleteWorkspaceHost, deleteWorkspaceHosts,
  deleteNamespaceHost, deleteNamespaceHosts,
  bindHostEnvironment, unbindHostEnvironment,
  bindWorkspaceHostEnvironment, unbindWorkspaceHostEnvironment,
  bindNamespaceHostEnvironment, unbindNamespaceHostEnvironment,
} from "@/api/infra/hosts"
import {
  listEnvironments, listWorkspaceEnvironments, listNamespaceEnvironments,
} from "@/api/infra/environments"
import {
  listInfraNetworks, listWorkspaceInfraNetworks, listNamespaceInfraNetworks,
} from "@/api/infra/networks"
import { showApiError } from "@/api/client"
import type { Host, Environment, AvailableNetwork, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { buildPermScope, scopedApiCall } from "@/lib/nav-config"
import { usePermission } from "@/hooks/use-permission"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"

export default function HostListPage() {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const { hasPermission, hasAnyPermission } = usePermission()
  const { workspaceId: scopeWorkspaceId, namespaceId: scopeNamespaceId } = useParams()

  const [hosts, setHosts] = useState<Host[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [statusFilter, setStatusFilter] = useState("all")

  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Host | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Host | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  // Action dialogs
  const [bindTarget, setBindTarget] = useState<Host | null>(null)
  const [unbindTarget, setUnbindTarget] = useState<Host | null>(null)

  const permPrefix = "infra:hosts"
  const isPlatformScope = !scopeWorkspaceId

  const permScope = buildPermScope(scopeWorkspaceId, scopeNamespaceId)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter

      const data = await scopedApiCall(
        scopeWorkspaceId, scopeNamespaceId,
        () => listHosts(params),
        (wsId) => listWorkspaceHosts(wsId, params),
        (wsId, nsId) => listNamespaceHosts(wsId, nsId, params),
      )
      setHosts(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search, statusFilter, scopeWorkspaceId, scopeNamespaceId])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, statusFilter, pageSize])
  useEffect(() => { clearSelection() }, [hosts])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await scopedApiCall(
        scopeWorkspaceId, scopeNamespaceId,
        () => deleteHost(deleteTarget.metadata.id),
        (wsId) => deleteWorkspaceHost(wsId, deleteTarget.metadata.id),
        (wsId, nsId) => deleteNamespaceHost(wsId, nsId, deleteTarget.metadata.id),
      )
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      showApiError(err, t, "host.title")
    }
  }

  const handleBatchDelete = async () => {
    try {
      const ids = Array.from(selected)
      await scopedApiCall(
        scopeWorkspaceId, scopeNamespaceId,
        () => deleteHosts(ids),
        (wsId) => deleteWorkspaceHosts(wsId, ids),
        (wsId, nsId) => deleteNamespaceHosts(wsId, nsId, ids),
      )
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchData()
    } catch (err) {
      showApiError(err, t, "host.title")
    }
  }

  const handleUnbind = async () => {
    if (!unbindTarget) return
    try {
      await scopedApiCall(
        scopeWorkspaceId, scopeNamespaceId,
        () => unbindHostEnvironment(unbindTarget.metadata.id),
        (wsId) => unbindWorkspaceHostEnvironment(wsId, unbindTarget.metadata.id),
        (wsId, nsId) => unbindNamespaceHostEnvironment(wsId, nsId, unbindTarget.metadata.id),
      )
      toast.success(t("action.updateSuccess"))
      setUnbindTarget(null)
      fetchData()
    } catch (err) {
      showApiError(err, t, "host.title")
    }
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("host.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("host.manage", { count: totalCount })}
          </p>
        </div>
        {(scopeWorkspaceId ? hasPermission(`${permPrefix}:create`, permScope) : hasAnyPermission("infra:hosts:create")) && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("host.create")}
          </Button>
        )}
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("host.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && hasPermission(`${permPrefix}:deleteCollection`, permScope) && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("host.deleteSelected")} ({selected.size})
          </Button>
        )}
      </div>

      {/* table */}
      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                {hasPermission(`${permPrefix}:deleteCollection`, permScope) && (
                  <Checkbox
                    checked={hosts.length > 0 && selected.size === hosts.length}
                    onCheckedChange={() => toggleAll(hosts.map((h) => h.metadata.id))}
                  />
                )}
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>{t("host.ipAddress")}</TableHead>
              <TableHead>{t("host.os")}</TableHead>
              <TableHead>{t("host.environment")}</TableHead>
              {isPlatformScope && (
                <>
                  <TableHead>{t("workspace.title")}</TableHead>
                  <TableHead>{t("namespace.title")}</TableHead>
                </>
              )}
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("common.status")}
                      <Filter className={`h-3 w-3 ${statusFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setStatusFilter("all")}>{t("common.all")}</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setStatusFilter("active")}>{t("common.active")}</DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setStatusFilter("inactive")}>{t("common.inactive")}</DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="w-28">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: isPlatformScope ? 10 : 8 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : hosts.length === 0 ? (
              <TableRow>
                <TableCell colSpan={isPlatformScope ? 10 : 8} className="text-muted-foreground py-8 text-center">
                  {t("host.noData")}
                </TableCell>
              </TableRow>
            ) : (
              hosts.map((host) => (
                <TableRow key={host.metadata.id}>
                  <TableCell>
                    {hasPermission(`${permPrefix}:deleteCollection`, permScope) && (
                      <Checkbox
                        checked={selected.has(host.metadata.id)}
                        onCheckedChange={() => toggleOne(host.metadata.id)}
                      />
                    )}
                  </TableCell>
                  <TableCell>
                    <Link to={`${host.metadata.id}`} className="font-medium hover:underline">
                      {host.metadata.name}
                    </Link>
                  </TableCell>
                  <TableCell className="text-sm">{host.spec.ipAddress || "-"}</TableCell>
                  <TableCell className="text-sm">{host.spec.os || "-"}</TableCell>
                  <TableCell>
                    {host.spec.environmentName ? (
                      <Badge variant="outline">{host.spec.environmentName}</Badge>
                    ) : (
                      <span className="text-muted-foreground text-sm">{t("host.environmentNone")}</span>
                    )}
                  </TableCell>
                  {isPlatformScope && (
                    <>
                      <TableCell className="text-sm">{host.spec.workspaceName || "-"}</TableCell>
                      <TableCell className="text-sm">{host.spec.namespaceName || "-"}</TableCell>
                    </>
                  )}
                  <TableCell>
                    <Badge variant={host.spec.status === "active" ? "default" : "secondary"}>
                      {host.spec.status === "active" ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(host.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm" className="h-8 px-2">
                          {t("common.actions")}
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        {hasPermission(`${permPrefix}:update`, permScope) && (
                          <DropdownMenuItem onClick={() => setEditTarget(host)}>
                            <Pencil className="mr-2 h-3.5 w-3.5" />
                            {t("common.edit")}
                          </DropdownMenuItem>
                        )}
                        {/* Bind/Unbind environment */}
                        {!host.spec.environmentId && hasPermission(`${permPrefix}:bind-environment`, permScope) && (
                          <DropdownMenuItem onClick={() => setBindTarget(host)}>
                            <Link2 className="mr-2 h-3.5 w-3.5" />
                            {t("host.bindEnv")}
                          </DropdownMenuItem>
                        )}
                        {host.spec.environmentId && hasPermission(`${permPrefix}:unbind-environment`, permScope) && (
                          <DropdownMenuItem onClick={() => setUnbindTarget(host)}>
                            <Unlink className="mr-2 h-3.5 w-3.5" />
                            {t("host.unbindEnv")}
                          </DropdownMenuItem>
                        )}
                        {hasPermission(`${permPrefix}:delete`, permScope) && (
                          <>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem className="text-destructive" onClick={() => setDeleteTarget(host)}>
                              <Trash2 className="mr-2 h-3.5 w-3.5" />
                              {t("common.delete")}
                            </DropdownMenuItem>
                          </>
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      {/* Create dialog */}
      <HostFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchData}
        scopeWorkspaceId={scopeWorkspaceId}
        scopeNamespaceId={scopeNamespaceId}
      />

      {/* Edit dialog */}
      <HostFormDialog
        open={!!editTarget}
        onOpenChange={(v) => { if (!v) setEditTarget(null) }}
        host={editTarget ?? undefined}
        onSuccess={fetchData}
        scopeWorkspaceId={scopeWorkspaceId}
        scopeNamespaceId={scopeNamespaceId}
      />

      {/* Bind environment dialog */}
      <BindEnvironmentDialog
        open={!!bindTarget}
        onOpenChange={(v) => { if (!v) setBindTarget(null) }}
        host={bindTarget}
        onSuccess={fetchData}
        scopeWorkspaceId={scopeWorkspaceId}
        scopeNamespaceId={scopeNamespaceId}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("host.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("host.deleteSelected")}
        description={t("host.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={!!unbindTarget}
        onOpenChange={(v) => { if (!v) setUnbindTarget(null) }}
        title={t("host.unbindEnv")}
        description={t("host.unbindEnvConfirm", { name: unbindTarget?.metadata.name ?? "" })}
        onConfirm={handleUnbind}
        confirmText={t("common.confirm")}
      />

    </div>
  )
}

// ===== Host Form Dialog =====

interface IPConfigFormValues {
  subnetId: string
  ip: string
}

interface HostFormValues {
  name: string
  displayName: string
  description: string
  hostname: string
  os: string
  arch: string
  cpuCores: string
  memoryMb: string
  diskGb: string
  status: "active" | "inactive"
  ips: IPConfigFormValues[]
}

function HostFormDialog({
  open, onOpenChange, host, onSuccess, scopeWorkspaceId, scopeNamespaceId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  host?: Host
  onSuccess: () => void
  scopeWorkspaceId: string | undefined
  scopeNamespaceId: string | undefined
}) {
  const { t } = useTranslation()
  const isEdit = !!host
  const [loading, setLoading] = useState(false)
  const [networks, setNetworks] = useState<AvailableNetwork[]>([])

  const ipConfigSchema = z.object({
    subnetId: z.string().min(1, t("api.validation.required", { field: t("host.ips.subnetId") })),
    ip: z.string().optional().refine(
      (v) => !v || /^(\d{1,3}\.){3}\d{1,3}$/.test(v),
      { message: t("api.validation.ip.format") },
    ),
  })

  const schema = z.object({
    name: z.string()
      .min(3, t("api.validation.name.format"))
      .max(50, t("api.validation.name.format"))
      .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("api.validation.name.format")),
    displayName: z.string().optional(),
    description: z.string().optional(),
    hostname: z.string().optional(),
    os: z.string().optional(),
    arch: z.string().optional(),
    cpuCores: z.string().optional(),
    memoryMb: z.string().optional(),
    diskGb: z.string().optional(),
    status: z.enum(["active", "inactive"]),
    ips: z.array(ipConfigSchema).optional().default([]),
  })

  const form = useForm<HostFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: { name: "", displayName: "", description: "", hostname: "", os: "", arch: "", cpuCores: "", memoryMb: "", diskGb: "", status: "active", ips: [] },
  })

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "ips",
  })

  useEffect(() => {
    if (open) {
      if (host) {
        form.reset({
          name: host.metadata.name,
          displayName: host.spec.displayName ?? "",
          description: host.spec.description ?? "",
          hostname: host.spec.hostname ?? "",
          os: host.spec.os ?? "",
          arch: host.spec.arch ?? "",
          cpuCores: host.spec.cpuCores ? String(host.spec.cpuCores) : "",
          memoryMb: host.spec.memoryMb ? String(host.spec.memoryMb) : "",
          diskGb: host.spec.diskGb ? String(host.spec.diskGb) : "",
          status: (host.spec.status as "active" | "inactive") ?? "active",
          ips: [],
        })
      } else {
        form.reset({ name: "", displayName: "", description: "", hostname: "", os: "", arch: "", cpuCores: "", memoryMb: "", diskGb: "", status: "active", ips: [] })
      }
      // Fetch available networks for IP configuration (create mode only)
      if (!host) {
        const fetchNetworks = async () => {
          try {
            const data = await scopedApiCall(
              scopeWorkspaceId, scopeNamespaceId,
              () => listInfraNetworks(),
              (wsId) => listWorkspaceInfraNetworks(wsId),
              (wsId, nsId) => listNamespaceInfraNetworks(wsId, nsId),
            )
            setNetworks(data.items ?? [])
          } catch {
            setNetworks([])
          }
        }
        fetchNetworks()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, host, form])

  const onSubmit = async (values: HostFormValues) => {
    setLoading(true)
    try {
      const ips = values.ips
        ?.filter((ip) => ip.subnetId)
        .map((ip) => ({ subnetId: ip.subnetId, ...(ip.ip ? { ip: ip.ip } : {}) }))

      const spec: Host["spec"] = {
        hostname: values.hostname,
        ...(ips && ips.length > 0 ? { ips } : {}),
        os: values.os,
        arch: values.arch,
        cpuCores: values.cpuCores ? Number(values.cpuCores) : undefined,
        memoryMb: values.memoryMb ? Number(values.memoryMb) : undefined,
        diskGb: values.diskGb ? Number(values.diskGb) : undefined,
        displayName: values.displayName,
        description: values.description,
        status: values.status,
      } as Host["spec"]

      const payload = {
        metadata: isEdit ? host.metadata : { name: values.name } as Host["metadata"],
        spec,
      }

      if (isEdit) {
        await scopedApiCall(
          scopeWorkspaceId, scopeNamespaceId,
          () => updateHost(host.metadata.id, payload),
          (wsId) => updateWorkspaceHost(wsId, host.metadata.id, payload),
          (wsId, nsId) => updateNamespaceHost(wsId, nsId, host.metadata.id, payload),
        )
        toast.success(t("action.updateSuccess"))
      } else {
        await scopedApiCall(
          scopeWorkspaceId, scopeNamespaceId,
          () => createHost(payload),
          (wsId) => createWorkspaceHost(wsId, payload),
          (wsId, nsId) => createNamespaceHost(wsId, nsId, payload),
        )
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      showApiError(err, t, "host.title")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] flex flex-col overflow-hidden" onOpenAutoFocus={(e) => e.preventDefault()} onCloseAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("host.edit") : t("host.create")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex min-h-0 flex-col flex-1 overflow-hidden">
            {form.formState.errors.root && (
              <div className="shrink-0 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div className="space-y-4 overflow-y-auto flex-1 min-h-0">
            <FormField control={form.control} name="name" render={({ field }) => (
              <FormItem>
                <FormLabel required>{t("host.name")}</FormLabel>
                <FormControl><Input {...field} disabled={isEdit} placeholder="my-host" /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem><FormLabel>{t("host.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem><FormLabel>{t("host.description")}</FormLabel><FormControl><Textarea rows={2} {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="hostname" render={({ field }) => (
              <FormItem><FormLabel>{t("host.hostname")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />

            {/* IP Configuration (create mode only) — right after hostname */}
            {!isEdit && (
              <div className="space-y-3">
                <div className="text-sm font-medium">{t("host.ips.section")}</div>
                {fields.map((field, index) => {
                  const selectedSubnetId = form.watch(`ips.${index}.subnetId`)
                  const selectedSubnet = selectedSubnetId
                    ? networks.flatMap((n) => n.spec.subnets).find((s) => s.id === selectedSubnetId)
                    : null
                  return (
                    <div key={field.id}>
                      <div className="flex items-start gap-2">
                        <div className="grid grid-cols-2 gap-2 flex-1 min-w-0">
                          <FormField control={form.control} name={`ips.${index}.subnetId`} render={({ field: f }) => (
                            <FormItem>
                              <Select value={f.value} onValueChange={f.onChange}>
                                <FormControl>
                                  <SelectTrigger className="w-full truncate">
                                    <SelectValue placeholder={t("host.ips.subnet.select")} />
                                  </SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                  {networks.map((net) => (
                                    <SelectGroup key={net.metadata.id}>
                                      <SelectLabel>{net.spec.displayName || net.metadata.name}{net.spec.cidr ? ` (${net.spec.cidr})` : ""}</SelectLabel>
                                      {net.spec.subnets.map((sub) => (
                                        <SelectItem key={sub.id} value={sub.id} disabled={sub.freeIPs === 0}>
                                          {sub.displayName || sub.name} — {sub.cidr}
                                          <span className="text-muted-foreground ml-2 text-xs">
                                            ({t("host.ips.subnet.free", { free: sub.freeIPs, total: sub.totalIPs })})
                                          </span>
                                        </SelectItem>
                                      ))}
                                    </SelectGroup>
                                  ))}
                                </SelectContent>
                              </Select>
                              <FormMessage />
                            </FormItem>
                          )} />
                          <FormField control={form.control} name={`ips.${index}.ip`} render={({ field: f }) => (
                            <FormItem>
                              <FormControl><Input {...f} placeholder={t("host.ips.ip")} /></FormControl>
                              <FormMessage />
                            </FormItem>
                          )} />
                        </div>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="h-9 w-9 shrink-0"
                          onClick={() => remove(index)}
                        >
                          <X className="h-4 w-4" />
                        </Button>
                      </div>
                      {selectedSubnet && (
                        <p className="text-muted-foreground mt-0.5 mb-0 text-xs pl-0.5">
                          {selectedSubnet.cidr} · {t("host.ips.subnet.free", { free: selectedSubnet.freeIPs, total: selectedSubnet.totalIPs })}
                        </p>
                      )}
                    </div>
                  )
                })}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => append({ subnetId: "", ip: "" })}
                >
                  <Plus className="mr-1 h-3.5 w-3.5" />
                  {t("host.ips.add")}
                </Button>
              </div>
            )}

            <div className="grid grid-cols-2 gap-4">
              <FormField control={form.control} name="os" render={({ field }) => (
                <FormItem><FormLabel>{t("host.os")}</FormLabel><FormControl><Input {...field} placeholder="Linux" /></FormControl><FormMessage /></FormItem>
              )} />
              <FormField control={form.control} name="arch" render={({ field }) => (
                <FormItem><FormLabel>{t("host.arch")}</FormLabel><FormControl><Input {...field} placeholder="amd64" /></FormControl><FormMessage /></FormItem>
              )} />
            </div>
            <div className="grid grid-cols-3 gap-4">
              <FormField control={form.control} name="cpuCores" render={({ field }) => (
                <FormItem><FormLabel>{t("host.cpuCores")}</FormLabel><FormControl><Input type="number" {...field} /></FormControl><FormMessage /></FormItem>
              )} />
              <FormField control={form.control} name="memoryMb" render={({ field }) => (
                <FormItem><FormLabel>{t("host.memoryMb")}</FormLabel><FormControl><Input type="number" {...field} /></FormControl><FormMessage /></FormItem>
              )} />
              <FormField control={form.control} name="diskGb" render={({ field }) => (
                <FormItem><FormLabel>{t("host.diskGb")}</FormLabel><FormControl><Input type="number" {...field} /></FormControl><FormMessage /></FormItem>
              )} />
            </div>
            <FormField control={form.control} name="status" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("common.status")}</FormLabel>
                <Select value={field.value} onValueChange={field.onChange}>
                  <FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl>
                  <SelectContent>
                    <SelectItem value="active">{t("common.active")}</SelectItem>
                    <SelectItem value="inactive">{t("common.inactive")}</SelectItem>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )} />

            </div>
            <DialogFooter className="mt-6 pt-4 border-t shrink-0">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
              <Button type="submit" disabled={loading}>{loading ? "..." : t("common.save")}</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

// ===== Bind Environment Dialog =====

function BindEnvironmentDialog({
  open, onOpenChange, host, onSuccess, scopeWorkspaceId, scopeNamespaceId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  host: Host | null
  onSuccess: () => void
  scopeWorkspaceId: string | undefined
  scopeNamespaceId: string | undefined
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [environments, setEnvironments] = useState<Environment[]>([])
  const [selectedEnvId, setSelectedEnvId] = useState("")

  useEffect(() => {
    if (open) {
      setSelectedEnvId("")
      const fetchEnvs = async () => {
        try {
          const data = await scopedApiCall(
            scopeWorkspaceId, scopeNamespaceId,
            () => listEnvironments({ pageSize: 100 }),
            (wsId) => listWorkspaceEnvironments(wsId, { pageSize: 100, inherit: "true" }),
            (wsId, nsId) => listNamespaceEnvironments(wsId, nsId, { pageSize: 100, inherit: "true" }),
          )
          setEnvironments(data.items ?? [])
        } catch {
          setEnvironments([])
        }
      }
      fetchEnvs()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, scopeWorkspaceId, scopeNamespaceId])

  const handleBind = async () => {
    if (!host || !selectedEnvId) return
    setLoading(true)
    try {
      const body = { environmentId: selectedEnvId }
      await scopedApiCall(
        scopeWorkspaceId, scopeNamespaceId,
        () => bindHostEnvironment(host.metadata.id, body),
        (wsId) => bindWorkspaceHostEnvironment(wsId, host.metadata.id, body),
        (wsId, nsId) => bindNamespaceHostEnvironment(wsId, nsId, host.metadata.id, body),
      )
      toast.success(t("action.updateSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      showApiError(err, t, "host.title")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onOpenAutoFocus={(e) => e.preventDefault()} onCloseAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("host.bindEnv")}: {host?.metadata.name}</DialogTitle>
        </DialogHeader>
        <div>
          <label className="text-sm font-medium">{t("host.selectEnvironment")}</label>
          <Select value={selectedEnvId} onValueChange={setSelectedEnvId}>
            <SelectTrigger className="w-full mt-1"><SelectValue placeholder={t("host.selectEnvironment")} /></SelectTrigger>
            <SelectContent>
              {environments.map((env) => (
                <SelectItem key={env.metadata.id} value={env.metadata.id}>
                  {env.spec.displayName || env.metadata.name}
                  {scopeWorkspaceId && env.spec.scope && (
                    <span className="text-muted-foreground ml-2 text-xs">
                      ({t(`scope.${env.spec.scope}`)})
                    </span>
                  )}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <DialogFooter className="mt-6 pt-4 border-t">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
          <Button onClick={handleBind} disabled={loading || !selectedEnvId}>
            {loading ? "..." : t("host.bindEnv")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
