import { useCallback, useEffect, useRef, useState } from "react"
import { Link } from "react-router"
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
  Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import { listRoles, createRole, updateRole, deleteRole, listPermissions } from "@/api/rbac"
import { ApiError, translateDetailMessage, translateApiError } from "@/api/client"
import type { Role, Permission, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { PermissionSelector } from "@/components/permission-selector"

const SCOPE_VARIANT: Record<string, "default" | "secondary" | "outline"> = {
  platform: "default",
  workspace: "secondary",
  namespace: "outline",
}

export default function RoleListPage() {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [scopeFilter, setScopeFilter] = useState<string>("all")
  const [builtinFilter, setBuiltinFilter] = useState<string>("all")
  const [permissions, setPermissions] = useState<Permission[]>([])

  // dialogs
  const [createOpen, setCreateOpen] = useState(false)
  const [editRole, setEditRole] = useState<Role | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Role | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const fetchRoles = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (scopeFilter !== "all") params.scope = scopeFilter
      if (builtinFilter !== "all") params.builtin = builtinFilter
      const data = await listRoles(params)
      setRoles(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch {
      toast.error(t("api.error.internalError"))
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search, scopeFilter, builtinFilter])

  useEffect(() => { fetchRoles() }, [fetchRoles])
  useEffect(() => { setPage(1) }, [search, scopeFilter, builtinFilter, pageSize])
  useEffect(() => { clearSelection() }, [roles])

  // fetch permissions once for form dialogs
  useEffect(() => {
    listPermissions({ pageSize: 1000 })
      .then((data) => setPermissions(data.items ?? []))
      .catch(() => {})
  }, [])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteRole(deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchRoles()
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("role.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const handleBatchDelete = async () => {
    try {
      await Promise.all(Array.from(selected).map((id) => deleteRole(id)))
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchRoles()
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("role.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  // only non-builtin roles can be selected for batch delete
  const selectableRoles = roles.filter((r) => !r.spec.builtin)
  const selectableIds = selectableRoles.map((r) => r.metadata.id)

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("role.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("role.manage", { count: totalCount })}
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          {t("role.create")}
        </Button>
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("role.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("role.batchDelete")} ({selected.size})
          </Button>
        )}
      </div>

      {/* table */}
      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                <Checkbox
                  checked={selectableIds.length > 0 && selected.size === selectableIds.length}
                  onCheckedChange={() => toggleAll(selectableIds)}
                />
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("name")}
              >
                {t("role.name")}
                <SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("display_name")}
              >
                {t("common.displayName")}
                <SortIcon field="display_name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("role.scope")}
                      <Filter className={`h-3 w-3 ${scopeFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setScopeFilter("all")}>
                      {t("common.all")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setScopeFilter("platform")}>
                      {t("role.scope.platform")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setScopeFilter("workspace")}>
                      {t("role.scope.workspace")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setScopeFilter("namespace")}>
                      {t("role.scope.namespace")}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="inline-flex items-center gap-1 select-none">
                      {t("role.builtin")}
                      <Filter className={`h-3 w-3 ${builtinFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuItem onClick={() => setBuiltinFilter("all")}>
                      {t("common.all")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setBuiltinFilter("true")}>
                      {t("role.builtin")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => setBuiltinFilter("false")}>
                      {t("role.custom")}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableHead>
              <TableHead>{t("common.description")}</TableHead>
              <TableHead>{t("role.rules")}</TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("created_at")}
              >
                {t("common.created")}
                <SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="w-24">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 9 }).map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-4 w-20" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : roles.length === 0 ? (
              <TableRow>
                <TableCell colSpan={9} className="text-muted-foreground py-8 text-center">
                  {t("role.noData")}
                </TableCell>
              </TableRow>
            ) : (
              roles.map((role) => (
                <TableRow key={role.metadata.id}>
                  <TableCell>
                    <Checkbox
                      checked={selected.has(role.metadata.id)}
                      onCheckedChange={() => toggleOne(role.metadata.id)}
                      disabled={!!role.spec.builtin}
                    />
                  </TableCell>
                  <TableCell>
                    <Link to={`/roles/${role.metadata.id}`} className="font-medium hover:underline">
                      {role.spec.name}
                    </Link>
                  </TableCell>
                  <TableCell>{t(`role.${role.spec.name}`, { defaultValue: role.spec.displayName || "-" })}</TableCell>
                  <TableCell>
                    <Badge variant={SCOPE_VARIANT[role.spec.scope] ?? "outline"}>
                      {t(`role.scope.${role.spec.scope}`)}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={role.spec.builtin ? "secondary" : "outline"}>
                      {role.spec.builtin ? t("role.builtin") : t("role.custom")}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-48 truncate text-sm">
                    {t(`role.desc.${role.spec.name}`, { defaultValue: role.spec.description || "-" })}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {t("role.rulesCount", { count: role.spec.ruleCount ?? role.spec.rules?.length ?? 0 })}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {new Date(role.metadata.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => setEditRole(role)}
                        disabled={!!role.spec.builtin}
                        title={role.spec.builtin ? t("role.builtinCannotEdit") : t("common.edit")}
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-destructive hover:text-destructive"
                        onClick={() => setDeleteTarget(role)}
                        disabled={!!role.spec.builtin}
                        title={role.spec.builtin ? t("role.builtinCannotDelete") : t("common.delete")}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      {/* create dialog */}
      <RoleFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        permissions={permissions}
        onSuccess={fetchRoles}
      />

      {/* edit dialog */}
      <RoleFormDialog
        open={!!editRole}
        onOpenChange={(v) => { if (!v) setEditRole(null) }}
        role={editRole ?? undefined}
        permissions={permissions}
        onSuccess={fetchRoles}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("role.deleteConfirm", { name: deleteTarget?.spec.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("role.batchDelete")}
        description={t("role.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// --- Role Create/Edit Form Dialog ---

interface RoleFormValues {
  name: string
  displayName: string
  description: string
  scope: "platform" | "workspace" | "namespace"
  rules: string[]
}

function RoleFormDialog({
  open,
  onOpenChange,
  role,
  permissions,
  onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  role?: Role
  permissions: Permission[]
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const isEdit = !!role
  const [loading, setLoading] = useState(false)

  const roleFormSchema = z.object({
    name: isEdit
      ? z.string()
      : z.string()
          .min(3, t("role.validation.name.format"))
          .max(50, t("role.validation.name.format"))
          .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("role.validation.name.format")),
    displayName: z.string().optional(),
    description: z.string().optional(),
    scope: z.enum(["platform", "workspace", "namespace"]),
    rules: z.array(z.string()).min(1, t("role.validation.rules.required")),
  })

  const form = useForm<RoleFormValues>({
    resolver: zodResolver(roleFormSchema) as never,
    mode: "onBlur",
    defaultValues: {
      name: "",
      displayName: "",
      description: "",
      scope: "platform",
      rules: [],
    },
  })

  useEffect(() => {
    if (open) {
      if (role) {
        form.reset({
          name: role.spec.name,
          displayName: role.spec.displayName ?? "",
          description: role.spec.description ?? "",
          scope: role.spec.scope,
          rules: role.spec.rules ?? [],
        })
      } else {
        form.reset({
          name: "",
          displayName: "",
          description: "",
          scope: "platform",
          rules: [],
        })
      }
    }
  }, [open, role, form])

  const checkUniqueness = async (value: string) => {
    if (!value) return
    try {
      const data = await listRoles({ page: 1, pageSize: 1, search: value })
      const exists = data.items?.some((r) => {
        if (isEdit && r.metadata.id === role?.metadata.id) return false
        return r.spec.name === value
      })
      if (exists) form.setError("name", { message: t("role.validation.name.taken") })
    } catch { /* backend will enforce */ }
  }

  const onSubmit = async (values: RoleFormValues) => {
    setLoading(true)
    try {
      if (isEdit) {
        await updateRole(role.metadata.id, {
          metadata: role.metadata,
          spec: {
            ...role.spec,
            displayName: values.displayName || undefined,
            description: values.description || undefined,
            rules: values.rules,
          },
        })
        toast.success(t("action.updateSuccess"))
      } else {
        await createRole({
          metadata: {} as Role["metadata"],
          spec: {
            name: values.name,
            displayName: values.displayName || undefined,
            description: values.description || undefined,
            scope: values.scope,
            rules: values.rules,
          } as Role["spec"],
        })
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError && err.details?.length) {
        for (const d of err.details) {
          const field = d.field.replace(/^spec\./, "") as keyof RoleFormValues
          const i18nKey = translateDetailMessage(d.message)
          form.setError(field, { message: i18nKey !== d.message ? t(i18nKey, { field: t(`role.${field}`) || field }) : d.message })
        }
      } else if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        form.setError("root", { message: i18nKey !== err.message ? t(i18nKey, { resource: t("role.title") }) : err.message })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  const selectedRules = form.watch("rules")
  const watchedScope = form.watch("scope")
  const isFirstScopeRender = useRef(true)

  useEffect(() => {
    if (isFirstScopeRender.current) {
      isFirstScopeRender.current = false
      return
    }
    if (!watchedScope || watchedScope === "platform") return
    const currentRules = form.getValues("rules")
    if (currentRules.length > 0) {
      form.setValue("rules", [], { shouldValidate: true })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [watchedScope])

  // Reset the ref when dialog opens/closes so it works correctly on reopening
  useEffect(() => {
    if (open) {
      isFirstScopeRender.current = true
    }
  }, [open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:!max-w-none !w-auto min-w-[800px] max-h-[85vh] flex flex-col overflow-hidden" onOpenAutoFocus={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("role.edit") : t("role.create")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex min-h-0 flex-col">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive mb-4">
                {form.formState.errors.root.message}
              </div>
            )}
            <div className="grid grid-cols-3 gap-6 min-h-0 flex-1">
              {/* Left: basic fields */}
              <div className="col-span-1 space-y-4 overflow-y-auto">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("role.name")}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          disabled={isEdit}
                          onBlur={async (e) => {
                            field.onBlur()
                            if (isEdit || !e.target.value) return
                            const valid = await form.trigger("name")
                            if (valid) checkUniqueness(e.target.value)
                          }}
                        />
                      </FormControl>
                      {!isEdit && <FormDescription>{t("role.validation.name.hint")}</FormDescription>}
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="scope"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("role.scope")}</FormLabel>
                      <Select value={field.value} onValueChange={field.onChange} disabled={isEdit}>
                        <FormControl>
                          <SelectTrigger className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="platform">{t("role.scope.platform")}</SelectItem>
                          <SelectItem value="workspace">{t("role.scope.workspace")}</SelectItem>
                          <SelectItem value="namespace">{t("role.scope.namespace")}</SelectItem>
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="displayName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("common.displayName")}</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="description"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("common.description")}</FormLabel>
                      <FormControl>
                        <Textarea {...field} rows={3} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              {/* Right: permission selector */}
              <FormField
                control={form.control}
                name="rules"
                render={() => (
                  <FormItem className="col-span-2 flex flex-col min-h-0">
                    <FormLabel>
                      {t("role.rules")}
                      {selectedRules.length > 0 && (
                        <span className="text-muted-foreground ml-2 font-normal">
                          ({t("role.rulesCount", { count: selectedRules.length })})
                        </span>
                      )}
                    </FormLabel>
                    <PermissionSelector
                      permissions={permissions}
                      value={selectedRules}
                      onChange={(rules) => form.setValue("rules", rules, { shouldValidate: true })}
                      scope={form.watch("scope")}
                    />
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <DialogFooter className="mt-6 pt-4 border-t shrink-0">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" disabled={loading}>
                {loading ? "..." : t("common.save")}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
