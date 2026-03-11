import { useCallback, useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { Pencil, Trash2, Server } from "lucide-react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import { ConfirmDialog } from "@/components/confirm-dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  getEnvironment, getWorkspaceEnvironment, getNamespaceEnvironment,
  updateEnvironment, updateWorkspaceEnvironment, updateNamespaceEnvironment,
  deleteEnvironment, deleteWorkspaceEnvironment, deleteNamespaceEnvironment,
  getEnvironmentHosts, getWorkspaceEnvironmentHosts, getNamespaceEnvironmentHosts,
} from "@/api/infra/environments"
import { ApiError, translateApiError } from "@/api/client"
import type { Environment, Host, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useScopeStore } from "@/stores/scope-store"
import { Pagination } from "@/components/pagination"

const ENV_TYPES = ["development", "testing", "staging", "production", "custom"]

export default function EnvironmentDetailPage() {
  const { environmentId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()
  const scopeWorkspaceId = useScopeStore((s) => s.workspaceId)
  const scopeNamespaceId = useScopeStore((s) => s.namespaceId)

  const [environment, setEnvironment] = useState<Environment | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  // Hosts in this environment
  const [hosts, setHosts] = useState<Host[]>([])
  const [hostsLoading, setHostsLoading] = useState(true)
  const [hostsTotal, setHostsTotal] = useState(0)
  const [hostsPage, setHostsPage] = useState(1)
  const [hostsPageSize, setHostsPageSize] = useState(10)

  const permPrefix = "infra:environments"

  const permScope = scopeNamespaceId
    ? { workspaceId: scopeWorkspaceId!, namespaceId: scopeNamespaceId }
    : scopeWorkspaceId
      ? { workspaceId: scopeWorkspaceId }
      : undefined

  const fetchEnvironment = useCallback(async () => {
    if (!environmentId) return
    try {
      let env
      if (scopeWorkspaceId && scopeNamespaceId) {
        env = await getNamespaceEnvironment(scopeWorkspaceId, scopeNamespaceId, environmentId)
      } else if (scopeWorkspaceId) {
        env = await getWorkspaceEnvironment(scopeWorkspaceId, environmentId)
      } else {
        env = await getEnvironment(environmentId)
      }
      setEnvironment(env)
    } catch {
      setEnvironment(null)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [environmentId, scopeWorkspaceId, scopeNamespaceId])

  const fetchHosts = useCallback(async () => {
    if (!environmentId) return
    setHostsLoading(true)
    try {
      const params: ListParams = { page: hostsPage, pageSize: hostsPageSize }
      let data
      if (scopeWorkspaceId && scopeNamespaceId) {
        data = await getNamespaceEnvironmentHosts(scopeWorkspaceId, scopeNamespaceId, environmentId, params)
      } else if (scopeWorkspaceId) {
        data = await getWorkspaceEnvironmentHosts(scopeWorkspaceId, environmentId, params)
      } else {
        data = await getEnvironmentHosts(environmentId, params)
      }
      setHosts(data.items ?? [])
      setHostsTotal(data.totalCount)
    } catch {
      setHosts([])
    } finally {
      setHostsLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [environmentId, hostsPage, hostsPageSize, scopeWorkspaceId, scopeNamespaceId])

  useEffect(() => { fetchEnvironment() }, [fetchEnvironment])
  useEffect(() => { fetchHosts() }, [fetchHosts])

  const handleDelete = async () => {
    if (!environment) return
    try {
      if (scopeWorkspaceId && scopeNamespaceId) {
        await deleteNamespaceEnvironment(scopeWorkspaceId, scopeNamespaceId, environment.metadata.id)
      } else if (scopeWorkspaceId) {
        await deleteWorkspaceEnvironment(scopeWorkspaceId, environment.metadata.id)
      } else {
        await deleteEnvironment(environment.metadata.id)
      }
      toast.success(t("action.deleteSuccess"))
      navigate("..")
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("env.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
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

  if (!environment) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("env.noData")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{environment.metadata.name}</h1>
          <Badge variant={environment.spec.status === "active" ? "default" : "secondary"}>
            {environment.spec.status === "active" ? t("common.active") : t("common.inactive")}
          </Badge>
          <Badge variant="outline">{t(`env.type.${environment.spec.envType ?? "custom"}` as const)}</Badge>
        </div>
        <div className="flex items-center gap-2">
          {hasPermission(`${permPrefix}:update`, permScope) && (
            <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              {t("common.edit")}
            </Button>
          )}
          {hasPermission(`${permPrefix}:delete`, permScope) && (
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("common.delete")}
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-4">
        {/* Overview cards */}
        <div className="grid grid-cols-3 gap-4">
          <Card>
            <CardContent className="flex items-center gap-4 p-4">
              <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
                <Server className="text-primary h-5 w-5" />
              </div>
              <div>
                <p className="text-2xl font-bold">{environment.spec.hostCount ?? 0}</p>
                <p className="text-muted-foreground text-sm">{t("env.hostCount")}</p>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Details card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("env.basicInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{environment.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.displayName")}</span>
                <p className="font-medium">{environment.spec.displayName || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("env.envType")}</span>
                <p className="font-medium">{t(`env.type.${environment.spec.envType ?? "custom"}` as const)}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.status")}</span>
                <p>
                  <Badge variant={environment.spec.status === "active" ? "default" : "secondary"}>
                    {environment.spec.status === "active" ? t("common.active") : t("common.inactive")}
                  </Badge>
                </p>
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("common.description")}</span>
                <p className="font-medium">{environment.spec.description || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(environment.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(environment.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Hosts in this environment */}
        <Card>
          <CardHeader>
            <CardTitle>{t("env.hosts")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("common.name")}</TableHead>
                    <TableHead>{t("host.hostname")}</TableHead>
                    <TableHead>{t("host.ipAddress")}</TableHead>
                    <TableHead>{t("host.os")}</TableHead>
                    <TableHead>{t("common.status")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {hostsLoading ? (
                    Array.from({ length: 3 }).map((_, i) => (
                      <TableRow key={i}>
                        {Array.from({ length: 5 }).map((_, j) => (
                          <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                        ))}
                      </TableRow>
                    ))
                  ) : hosts.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-muted-foreground py-8 text-center">
                        {t("env.hostsEmpty")}
                      </TableCell>
                    </TableRow>
                  ) : (
                    hosts.map((host) => (
                      <TableRow key={host.metadata.id}>
                        <TableCell className="font-medium">{host.metadata.name}</TableCell>
                        <TableCell>{host.spec.hostname || "-"}</TableCell>
                        <TableCell>{host.spec.ipAddress || "-"}</TableCell>
                        <TableCell>{host.spec.os || "-"}</TableCell>
                        <TableCell>
                          <Badge variant={host.spec.status === "active" ? "default" : "secondary"}>
                            {host.spec.status === "active" ? t("common.active") : t("common.inactive")}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
            <Pagination
              totalCount={hostsTotal}
              page={hostsPage}
              pageSize={hostsPageSize}
              onPageChange={setHostsPage}
              onPageSizeChange={setHostsPageSize}
            />
          </CardContent>
        </Card>
      </div>

      {/* Edit dialog */}
      <EditEnvironmentDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        environment={environment}
        onSuccess={fetchEnvironment}
        scopeWorkspaceId={scopeWorkspaceId}
        scopeNamespaceId={scopeNamespaceId}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("env.deleteConfirm", { name: environment.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Edit Environment Dialog =====

function EditEnvironmentDialog({
  open, onOpenChange, environment, onSuccess, scopeWorkspaceId, scopeNamespaceId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  environment: Environment
  onSuccess: () => void
  scopeWorkspaceId: string | null
  scopeNamespaceId: string | null
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    displayName: z.string().optional(),
    description: z.string().optional(),
    envType: z.string().min(1),
    status: z.enum(["active", "inactive"]),
  })

  type FormValues = z.infer<typeof schema>

  const form = useForm<FormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      displayName: environment.spec.displayName ?? "",
      description: environment.spec.description ?? "",
      envType: environment.spec.envType ?? "custom",
      status: (environment.spec.status as "active" | "inactive") ?? "active",
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        displayName: environment.spec.displayName ?? "",
        description: environment.spec.description ?? "",
        envType: environment.spec.envType ?? "custom",
        status: (environment.spec.status as "active" | "inactive") ?? "active",
      })
    }
  }, [open, environment, form])

  const onSubmit = async (values: FormValues) => {
    setLoading(true)
    try {
      const payload = {
        metadata: environment.metadata,
        spec: { ...environment.spec, ...values },
      }
      if (scopeWorkspaceId && scopeNamespaceId) {
        await updateNamespaceEnvironment(scopeWorkspaceId, scopeNamespaceId, environment.metadata.id, payload)
      } else if (scopeWorkspaceId) {
        await updateWorkspaceEnvironment(scopeWorkspaceId, environment.metadata.id, payload)
      } else {
        await updateEnvironment(environment.metadata.id, payload)
      }
      toast.success(t("action.updateSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError) {
        form.setError("root", {
          message: translateApiError(err) !== err.message
            ? t(translateApiError(err), { resource: t("env.title") })
            : err.message,
        })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("env.edit")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div>
              <label className="text-sm font-medium">{t("env.name")}</label>
              <Input value={environment.metadata.name} disabled className="mt-1" />
            </div>
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem><FormLabel>{t("env.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem><FormLabel>{t("env.description")}</FormLabel><FormControl><Textarea rows={3} {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="envType" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("env.envType")}</FormLabel>
                <Select value={field.value} onValueChange={field.onChange}>
                  <FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl>
                  <SelectContent>
                    {ENV_TYPES.map((et) => (
                      <SelectItem key={et} value={et}>{t(`env.type.${et}` as const)}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )} />
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
