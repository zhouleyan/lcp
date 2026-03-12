import { useCallback, useEffect, useState } from "react"
import { Link, useParams } from "react-router"
import { Plus, Pencil, Trash2, Search, Filter } from "lucide-react"
import { useForm } from "react-hook-form"
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
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  listEnvironments, listWorkspaceEnvironments, listNamespaceEnvironments,
  createEnvironment, createWorkspaceEnvironment, createNamespaceEnvironment,
  updateEnvironment, updateWorkspaceEnvironment, updateNamespaceEnvironment,
  deleteEnvironment, deleteEnvironments,
  deleteWorkspaceEnvironment, deleteWorkspaceEnvironments,
  deleteNamespaceEnvironment, deleteNamespaceEnvironments,
} from "@/api/infra/environments"
import { showApiError } from "@/api/client"
import type { Environment, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { buildPermScope, scopedApiCall } from "@/lib/nav-config"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"

import { ENV_TYPES } from "@/pages/infra/constants"

export default function EnvironmentListPage() {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const { hasPermission, hasAnyPermission } = usePermission()
  const { workspaceId: scopeWorkspaceId, namespaceId: scopeNamespaceId } = useParams()

  const [environments, setEnvironments] = useState<Environment[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [statusFilter, setStatusFilter] = useState("all")
  const [envTypeFilter, setEnvTypeFilter] = useState("all")

  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Environment | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Environment | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  // Determine permission prefix based on scope
  const permPrefix = "infra:environments"

  const permScope = buildPermScope(scopeWorkspaceId, scopeNamespaceId)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter
      if (envTypeFilter !== "all") params.envType = envTypeFilter

      const data = await scopedApiCall(
        scopeWorkspaceId, scopeNamespaceId,
        () => listEnvironments(params),
        (wsId) => listWorkspaceEnvironments(wsId, params),
        (wsId, nsId) => listNamespaceEnvironments(wsId, nsId, params),
      )
      setEnvironments(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search, statusFilter, envTypeFilter, scopeWorkspaceId, scopeNamespaceId])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, statusFilter, envTypeFilter, pageSize])
  useEffect(() => { clearSelection() }, [environments])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await scopedApiCall(
        scopeWorkspaceId, scopeNamespaceId,
        () => deleteEnvironment(deleteTarget.metadata.id),
        (wsId) => deleteWorkspaceEnvironment(wsId, deleteTarget.metadata.id),
        (wsId, nsId) => deleteNamespaceEnvironment(wsId, nsId, deleteTarget.metadata.id),
      )
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      showApiError(err, t, "environment.title")
    }
  }

  const handleBatchDelete = async () => {
    try {
      const ids = Array.from(selected)
      await scopedApiCall(
        scopeWorkspaceId, scopeNamespaceId,
        () => deleteEnvironments(ids),
        (wsId) => deleteWorkspaceEnvironments(wsId, ids),
        (wsId, nsId) => deleteNamespaceEnvironments(wsId, nsId, ids),
      )
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchData()
    } catch (err) {
      showApiError(err, t, "environment.title")
    }
  }

  const envTypeBadge = (envType?: string) => {
    const key = `env.type.${envType ?? "custom"}` as const
    return t(key)
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("env.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("env.manage", { count: totalCount })}
          </p>
        </div>
        {(scopeWorkspaceId ? hasPermission(`${permPrefix}:create`, permScope) : hasAnyPermission("infra:environments:create")) && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("env.create")}
          </Button>
        )}
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("env.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && hasPermission(`${permPrefix}:deleteCollection`, permScope) && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("env.deleteSelected")} ({selected.size})
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
                    checked={environments.length > 0 && selected.size === environments.length}
                    onCheckedChange={() => toggleAll(environments.map((e) => e.metadata.id))}
                  />
                )}
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("display_name")}>
                {t("common.displayName")}<SortIcon field="display_name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>{t("common.description")}</TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("env.envType")}
                      <Filter className={`h-3 w-3 ${envTypeFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setEnvTypeFilter("all")}>{t("env.filter.envTypeAll")}</DropdownMenuItem>
                    {ENV_TYPES.map((et) => (
                      <DropdownMenuItem key={et} onClick={() => setEnvTypeFilter(et)}>
                        {t(`env.type.${et}` as const)}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead className="cursor-pointer select-none text-center" onClick={() => handleSort("host_count")}>
                {t("env.hostCount")}<SortIcon field="host_count" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
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
              <TableHead className="w-24">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 9 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : environments.length === 0 ? (
              <TableRow>
                <TableCell colSpan={9} className="text-muted-foreground py-8 text-center">
                  {t("env.noData")}
                </TableCell>
              </TableRow>
            ) : (
              environments.map((env) => (
                <TableRow key={env.metadata.id}>
                  <TableCell>
                    {hasPermission(`${permPrefix}:deleteCollection`, permScope) && (
                      <Checkbox
                        checked={selected.has(env.metadata.id)}
                        onCheckedChange={() => toggleOne(env.metadata.id)}
                      />
                    )}
                  </TableCell>
                  <TableCell>
                    <Link to={`${env.metadata.id}`} className="font-medium hover:underline">
                      {env.metadata.name}
                    </Link>
                  </TableCell>
                  <TableCell>{env.spec.displayName || "-"}</TableCell>
                  <TableCell className="max-w-[200px] truncate text-muted-foreground text-sm" title={env.spec.description}>
                    {env.spec.description || "-"}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{envTypeBadge(env.spec.envType)}</Badge>
                  </TableCell>
                  <TableCell className="text-center">{env.spec.hostCount ?? 0}</TableCell>
                  <TableCell>
                    <Badge variant={env.spec.status === "active" ? "default" : "secondary"}>
                      {env.spec.status === "active" ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(env.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {hasPermission(`${permPrefix}:update`, permScope) && (
                        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setEditTarget(env)} title={t("common.edit")}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      {hasPermission(`${permPrefix}:delete`, permScope) && (
                        <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive" onClick={() => setDeleteTarget(env)} title={t("common.delete")}>
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      <EnvironmentFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchData}
        scopeWorkspaceId={scopeWorkspaceId}
        scopeNamespaceId={scopeNamespaceId}
      />

      <EnvironmentFormDialog
        open={!!editTarget}
        onOpenChange={(v) => { if (!v) setEditTarget(null) }}
        environment={editTarget ?? undefined}
        onSuccess={fetchData}
        scopeWorkspaceId={scopeWorkspaceId}
        scopeNamespaceId={scopeNamespaceId}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("env.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("env.deleteSelected")}
        description={t("env.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Environment Form Dialog =====

interface EnvironmentFormValues {
  name: string
  displayName: string
  description: string
  envType: string
  status: "active" | "inactive"
}

function EnvironmentFormDialog({
  open, onOpenChange, environment, onSuccess, scopeWorkspaceId, scopeNamespaceId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  environment?: Environment
  onSuccess: () => void
  scopeWorkspaceId?: string
  scopeNamespaceId?: string
}) {
  const { t } = useTranslation()
  const isEdit = !!environment
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    name: z.string()
      .min(3, t("api.validation.name.format"))
      .max(50, t("api.validation.name.format"))
      .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("api.validation.name.format")),
    displayName: z.string().optional(),
    description: z.string().optional(),
    envType: z.string().min(1),
    status: z.enum(["active", "inactive"]),
  })

  const form = useForm<EnvironmentFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: { name: "", displayName: "", description: "", envType: "custom", status: "active" },
  })

  useEffect(() => {
    if (open) {
      if (environment) {
        form.reset({
          name: environment.metadata.name,
          displayName: environment.spec.displayName ?? "",
          description: environment.spec.description ?? "",
          envType: environment.spec.envType ?? "custom",
          status: (environment.spec.status as "active" | "inactive") ?? "active",
        })
      } else {
        form.reset({ name: "", displayName: "", description: "", envType: "custom", status: "active" })
      }
    }
  }, [open, environment, form])

  const onSubmit = async (values: EnvironmentFormValues) => {
    setLoading(true)
    try {
      const payload = {
        metadata: { name: values.name } as Environment["metadata"],
        spec: {
          displayName: values.displayName,
          description: values.description,
          envType: values.envType,
          status: values.status,
        } as Environment["spec"],
      }

      if (isEdit) {
        payload.metadata = environment.metadata
        await scopedApiCall(
          scopeWorkspaceId, scopeNamespaceId,
          () => updateEnvironment(environment.metadata.id, payload),
          (wsId) => updateWorkspaceEnvironment(wsId, environment.metadata.id, payload),
          (wsId, nsId) => updateNamespaceEnvironment(wsId, nsId, environment.metadata.id, payload),
        )
        toast.success(t("action.updateSuccess"))
      } else {
        await scopedApiCall(
          scopeWorkspaceId, scopeNamespaceId,
          () => createEnvironment(payload),
          (wsId) => createWorkspaceEnvironment(wsId, payload),
          (wsId, nsId) => createNamespaceEnvironment(wsId, nsId, payload),
        )
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      showApiError(err, t, "environment.title")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] flex flex-col overflow-hidden" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("env.edit") : t("env.create")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex min-h-0 flex-col flex-1 overflow-hidden">
            {form.formState.errors.root && (
              <div className="shrink-0 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div className="space-y-4 overflow-y-auto flex-1 min-h-0">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel required>{t("env.name")}</FormLabel>
                  <FormControl>
                    <Input {...field} disabled={isEdit} placeholder="my-environment" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="displayName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("env.displayName")}</FormLabel>
                  <FormControl><Input {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("env.description")}</FormLabel>
                  <FormControl><Textarea rows={3} {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="envType"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("env.envType")}</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {ENV_TYPES.map((et) => (
                        <SelectItem key={et} value={et}>{t(`env.type.${et}` as const)}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="status"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("common.status")}</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="active">{t("common.active")}</SelectItem>
                      <SelectItem value="inactive">{t("common.inactive")}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
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
