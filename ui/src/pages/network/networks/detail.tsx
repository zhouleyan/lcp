import { useCallback, useEffect, useState } from "react"
import { useParams, useNavigate, Link } from "react-router"
import { Plus, Pencil, Trash2, Search } from "lucide-react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { getNetwork, updateNetwork, deleteNetwork } from "@/api/network/networks"
import {
  listSubnets, createSubnet, updateSubnet, deleteSubnet, deleteSubnets,
} from "@/api/network/subnets"
import { ApiError, showApiError, translateApiError, translateDetailMessage } from "@/api/client"
import type { Network, Subnet, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { cidrUsableRange } from "./utils"

export default function NetworkDetailPage() {
  const { networkId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()

  const [network, setNetwork] = useState<Network | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const fetchNetwork = useCallback(async () => {
    if (!networkId) return
    try {
      const n = await getNetwork(networkId)
      setNetwork(n)
    } catch {
      setNetwork(null)
    } finally {
      setLoading(false)
    }
  }, [networkId])

  useEffect(() => { fetchNetwork() }, [fetchNetwork])

  const handleDelete = async () => {
    if (!network) return
    try {
      await deleteNetwork(network.metadata.id)
      toast.success(t("action.deleteSuccess"))
      navigate("..")
    } catch (err) {
      showApiError(err, t, "network.title")
    }
  }

  if (loading) {
    return (
      <div className="space-y-4 p-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  if (!network) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("network.noData")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{network.metadata.name}</h1>
        </div>
        <div className="flex items-center gap-2">
          {hasPermission("network:networks:update") && (
            <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              {t("common.edit")}
            </Button>
          )}
          {hasPermission("network:networks:delete") && (
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("common.delete")}
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-4">
        {/* Basic info */}
        <Card>
          <CardHeader>
            <CardTitle>{t("network.basicInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{network.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.displayName")}</span>
                <p className="font-medium">{network.spec.displayName || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("network.cidr")}</span>
                <p className="font-medium font-mono">
                  {network.spec.cidr || "-"}
                  {network.spec.cidr && cidrUsableRange(network.spec.cidr) && (
                    <span className="ml-2 text-xs font-normal text-muted-foreground">{cidrUsableRange(network.spec.cidr)}</span>
                  )}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("network.isPublic")}</span>
                <p className="mt-1">
                  <Badge variant={network.spec.isPublic !== false ? "default" : "secondary"}>
                    {network.spec.isPublic !== false ? t("network.public") : t("network.private")}
                  </Badge>
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("network.subnetCount")}</span>
                {(() => {
                  const used = network.spec.subnetCount ?? 0
                  const total = network.spec.maxSubnets ?? 10
                  const pct = total > 0 ? Math.round((used / total) * 100) : 0
                  const barColor = pct > 90 ? "bg-primary" : pct > 60 ? "bg-primary/50" : "bg-primary/20"
                  return (
                    <div className="mt-1 space-y-1">
                      <div className="h-2 rounded-full bg-muted">
                        <div className={`h-2 rounded-full transition-all ${barColor}`} style={{ width: `${pct}%` }} />
                      </div>
                      <div className="flex justify-between text-xs text-muted-foreground">
                        <span className="font-medium">{used} / {total}</span>
                        <span>{pct}%</span>
                      </div>
                    </div>
                  )
                })()}
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("common.description")}</span>
                <p className="font-medium">{network.spec.description || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(network.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(network.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Subnets */}
        {networkId && <SubnetsSection networkId={networkId} networkCIDR={network.spec.cidr} />}
      </div>

      {/* Edit dialog */}
      <EditNetworkDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        network={network}
        onSuccess={fetchNetwork}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("network.deleteConfirm", { name: network.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Subnets Section =====

function SubnetsSection({ networkId, networkCIDR }: { networkId: string; networkCIDR?: string }) {
  const { t } = useTranslation()
  const { hasPermission } = usePermission()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()

  const [subnets, setSubnets] = useState<Subnet[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)

  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Subnet | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Subnet | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search

      const data = await listSubnets(networkId, params)
      setSubnets(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [networkId, page, pageSize, sortBy, sortOrder, search])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, pageSize])
  useEffect(() => { clearSelection() }, [subnets])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteSubnet(networkId, deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      showApiError(err, t, "subnet.title")
    }
  }

  const handleBatchDelete = async () => {
    try {
      const ids = Array.from(selected)
      await deleteSubnets(networkId, ids)
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchData()
    } catch (err) {
      showApiError(err, t, "subnet.title")
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>{t("subnet.title")}</CardTitle>
          {hasPermission("network:subnets:create") && (
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              {t("subnet.create")}
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {/* filters */}
        <div className="mb-4 flex items-center gap-3">
          <div className="relative max-w-xs flex-1">
            <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
            <Input
              placeholder={t("subnet.searchPlaceholder")}
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              className="pl-9"
            />
          </div>
          {selected.size > 0 && hasPermission("network:subnets:deleteCollection") && (
            <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("subnet.deleteSelected")} ({selected.size})
            </Button>
          )}
        </div>

        {/* table */}
        <div className="border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  {hasPermission("network:subnets:deleteCollection") && (
                    <Checkbox
                      checked={subnets.length > 0 && selected.size === subnets.length}
                      onCheckedChange={() => toggleAll(subnets.map((s) => s.metadata.id))}
                    />
                  )}
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                  {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("cidr")}>
                  {t("subnet.cidr")}<SortIcon field="cidr" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead>{t("subnet.gateway")}</TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("usage")}>
                  {t("subnet.ipUsage")}<SortIcon field="usage" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                  {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("updated_at")}>
                  {t("common.updated")}<SortIcon field="updated_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="w-28">{t("common.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 3 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 8 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : subnets.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-muted-foreground py-8 text-center">
                    {t("subnet.noData")}
                  </TableCell>
                </TableRow>
              ) : (
                subnets.map((subnet) => (
                  <TableRow key={subnet.metadata.id}>
                    <TableCell>
                      {hasPermission("network:subnets:deleteCollection") && (
                        <Checkbox
                          checked={selected.has(subnet.metadata.id)}
                          onCheckedChange={() => toggleOne(subnet.metadata.id)}
                        />
                      )}
                    </TableCell>
                    <TableCell>
                      <Link to={`subnets/${subnet.metadata.id}`} className="font-medium hover:underline">
                        {subnet.metadata.name}
                      </Link>
                    </TableCell>
                    <TableCell className="text-sm font-mono">
                      <div>{subnet.spec.cidr}</div>
                      {cidrUsableRange(subnet.spec.cidr) && (
                        <div className="text-xs text-muted-foreground">{cidrUsableRange(subnet.spec.cidr)}</div>
                      )}
                    </TableCell>
                    <TableCell className="text-sm font-mono">{subnet.spec.gateway || "-"}</TableCell>
                    <TableCell>
                      {(() => {
                        const used = subnet.spec.usedIPs ?? 0
                        const total = subnet.spec.totalIPs ?? 0
                        const free = subnet.spec.freeIPs ?? 0
                        const pct = total > 0 ? Math.round((used / total) * 100) : 0
                        const barColor = pct > 90 ? "bg-primary" : pct > 60 ? "bg-primary/50" : "bg-primary/20"
                        return (
                          <div className="pr-8 space-y-1">
                            <div className="h-2 rounded-full bg-muted">
                              <div className={`h-2 rounded-full transition-all ${barColor}`} style={{ width: `${pct}%` }} />
                            </div>
                            <div className="flex justify-between text-xs text-muted-foreground">
                              <span>{used}/{total} ({free} {t("subnet.freeIPs")})</span>
                              <span>{pct}%</span>
                            </div>
                          </div>
                        )
                      })()}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {new Date(subnet.metadata.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {subnet.metadata.updatedAt ? new Date(subnet.metadata.updatedAt).toLocaleString() : "-"}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm" className="h-8 px-2">
                            {t("common.actions")}
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {hasPermission("network:subnets:update") && (
                            <DropdownMenuItem onClick={() => setEditTarget(subnet)}>
                              <Pencil className="mr-2 h-3.5 w-3.5" />
                              {t("common.edit")}
                            </DropdownMenuItem>
                          )}
                          {hasPermission("network:subnets:delete") && (
                            <>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem className="text-destructive" onClick={() => setDeleteTarget(subnet)}>
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

        {/* Create subnet dialog */}
        <SubnetFormDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          networkId={networkId}
          networkCIDR={networkCIDR}
          onSuccess={fetchData}
        />

        {/* Edit subnet dialog */}
        <SubnetFormDialog
          open={!!editTarget}
          onOpenChange={(v) => { if (!v) setEditTarget(null) }}
          networkId={networkId}
          networkCIDR={networkCIDR}
          subnet={editTarget ?? undefined}
          onSuccess={fetchData}
        />

        <ConfirmDialog
          open={!!deleteTarget}
          onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
          title={t("common.delete")}
          description={t("subnet.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
          onConfirm={handleDelete}
          confirmText={t("common.delete")}
        />

        <ConfirmDialog
          open={batchDeleteOpen}
          onOpenChange={setBatchDeleteOpen}
          title={t("subnet.deleteSelected")}
          description={t("subnet.batchDeleteConfirm", { count: selected.size })}
          onConfirm={handleBatchDelete}
          confirmText={t("common.delete")}
        />
      </CardContent>
    </Card>
  )
}

// ===== Edit Network Dialog =====

function EditNetworkDialog({
  open, onOpenChange, network, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  network: Network
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    displayName: z.string().optional(),
    description: z.string().optional(),
  })

  type FormValues = z.infer<typeof schema>

  const form = useForm<FormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      displayName: network.spec.displayName ?? "",
      description: network.spec.description ?? "",
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        displayName: network.spec.displayName ?? "",
        description: network.spec.description ?? "",
      })
    }
  }, [open, network, form])

  const onSubmit = async (values: FormValues) => {
    setLoading(true)
    try {
      const spec: Network["spec"] = {
        ...network.spec,
        displayName: values.displayName,
        description: values.description,
      }

      await updateNetwork(network.metadata.id, { metadata: network.metadata, spec })
      toast.success(t("action.updateSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError && err.details?.length) {
        for (const d of err.details) {
          const fieldName = d.field.replace(/^(spec|metadata)\./, "") as keyof FormValues
          const i18nKey = translateDetailMessage(d.message)
          form.setError(fieldName, { message: i18nKey !== d.message ? t(i18nKey, { field: fieldName }) : d.message })
        }
      } else if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        form.setError("root", { message: i18nKey !== err.message ? t(i18nKey, { resource: t("network.title") }) : err.message })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("network.edit")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div>
              <label className="text-sm font-medium">{t("network.name")}</label>
              <Input value={network.metadata.name} disabled className="mt-1" />
            </div>
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem><FormLabel>{t("network.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem><FormLabel>{t("network.description")}</FormLabel><FormControl><Textarea rows={2} {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <DialogFooter className="mt-6 pt-4 border-t">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
              <Button type="submit" disabled={loading}>{loading ? "..." : t("common.save")}</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

// ===== Subnet Form Dialog =====

interface SubnetFormValues {
  name: string
  displayName: string
  description: string
  cidr: string
  gateway: string
}

// Mirrors backend ValidateSubnetCreate CIDR containment check for instant UX feedback.
function isCIDRWithinNetwork(subnetCIDR: string, networkCIDR: string): boolean {
  const parse = (cidr: string) => {
    const [ip, prefixStr] = cidr.split("/")
    const prefix = parseInt(prefixStr, 10)
    const octets = ip.split(".").map(Number)
    const ipNum = ((octets[0] << 24) | (octets[1] << 16) | (octets[2] << 8) | octets[3]) >>> 0
    return { ipNum, prefix }
  }
  const net = parse(networkCIDR)
  const sub = parse(subnetCIDR)
  if (sub.prefix < net.prefix) return false
  const mask = net.prefix === 0 ? 0 : (~0 << (32 - net.prefix)) >>> 0
  return (sub.ipNum & mask) === (net.ipNum & mask)
}

function SubnetFormDialog({
  open, onOpenChange, networkId, networkCIDR, subnet, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  networkId: string
  networkCIDR?: string
  subnet?: Subnet
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const isEdit = !!subnet
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    name: z.string()
      .min(3, t("api.validation.name.networkFormat"))
      .max(50, t("api.validation.name.networkFormat"))
      .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("api.validation.name.networkFormat")),
    displayName: z.string().optional(),
    description: z.string().optional(),
    cidr: z.string()
      .min(1, t("api.validation.required", { field: t("subnet.cidr") }))
      .regex(/^\d{1,3}(\.\d{1,3}){3}\/\d{1,2}$/, t("api.validation.cidr.format")),
    gateway: z.string()
      .refine((v) => !v || /^\d{1,3}(\.\d{1,3}){3}$/.test(v), t("api.validation.ip.format"))
      .optional(),
  })

  const form = useForm<SubnetFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: { name: "", displayName: "", description: "", cidr: "", gateway: "" },
  })

  useEffect(() => {
    if (open) {
      if (subnet) {
        form.reset({
          name: subnet.metadata.name,
          displayName: subnet.spec.displayName ?? "",
          description: subnet.spec.description ?? "",
          cidr: subnet.spec.cidr,
          gateway: subnet.spec.gateway ?? "",
        })
      } else {
        form.reset({ name: "", displayName: "", description: "", cidr: "", gateway: "" })
      }
    }
  }, [open, subnet, form])

  const onSubmit = async (values: SubnetFormValues) => {
    setLoading(true)
    try {
      const spec: Subnet["spec"] = {
        displayName: values.displayName,
        description: values.description,
        cidr: values.cidr,
        gateway: values.gateway || undefined,
      }

      const payload = {
        metadata: isEdit ? subnet.metadata : { name: values.name } as Subnet["metadata"],
        spec,
      }

      if (isEdit) {
        await updateSubnet(networkId, subnet.metadata.id, payload)
        toast.success(t("action.updateSuccess"))
      } else {
        await createSubnet(networkId, payload)
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError && err.details?.length) {
        for (const d of err.details) {
          const fieldName = d.field.replace(/^(spec|metadata)\./, "") as keyof SubnetFormValues
          const i18nKey = translateDetailMessage(d.message)
          form.setError(fieldName, { message: i18nKey !== d.message ? t(i18nKey, { field: t(`subnet.${fieldName}`) || fieldName }) : d.message })
        }
      } else if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        form.setError("root", { message: i18nKey !== err.message ? t(i18nKey, { resource: t("subnet.title") }) : err.message })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("subnet.edit") : t("subnet.create")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <FormField control={form.control} name="name" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("subnet.name")}</FormLabel>
                <FormControl><Input {...field} disabled={isEdit} placeholder="my-subnet" /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem><FormLabel>{t("subnet.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem><FormLabel>{t("subnet.description")}</FormLabel><FormControl><Textarea rows={2} {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <div className="grid grid-cols-2 gap-4">
              <FormField control={form.control} name="cidr" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("subnet.cidr")}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      disabled={isEdit}
                      placeholder={t("subnet.cidrPlaceholder")}
                      onBlur={async (e) => {
                        field.onBlur()
                        const val = e.target.value
                        if (!val || isEdit) return
                        // Wait for zod format validation
                        const valid = await form.trigger("cidr")
                        if (!valid) return
                        // Check CIDR within network CIDR
                        if (networkCIDR && !isCIDRWithinNetwork(val, networkCIDR)) {
                          form.setError("cidr", { message: t("api.validation.cidr.notWithinNetwork") })
                          return
                        }
                        // Auto-fill gateway
                        if (!form.getValues("gateway")) {
                          const match = val.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\/\d{1,2}$/)
                          if (match) {
                            const octets = [+match[1], +match[2], +match[3], +match[4]]
                            octets[3] += 1
                            if (octets[3] <= 255) form.setValue("gateway", octets.join("."))
                          }
                        }
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="gateway" render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("subnet.gateway")}</FormLabel>
                  <FormControl><Input {...field} disabled={isEdit} placeholder={t("subnet.gatewayPlaceholder")} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
            </div>
            <DialogFooter className="mt-6 pt-4 border-t">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
              <Button type="submit" disabled={loading}>{loading ? "..." : t("common.save")}</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
