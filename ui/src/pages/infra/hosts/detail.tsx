import { useCallback, useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { Pencil, Trash2, Cpu, HardDrive, MemoryStick } from "lucide-react"
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
  getHost, getWorkspaceHost, getNamespaceHost,
  updateHost, updateWorkspaceHost, updateNamespaceHost,
  deleteHost, deleteWorkspaceHost, deleteNamespaceHost,
  getHostAssignments,
} from "@/api/infra/hosts"
import { ApiError, translateApiError } from "@/api/client"
import type { Host, HostAssignment } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useScopeStore } from "@/stores/scope-store"

export default function HostDetailPage() {
  const { hostId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()
  const scopeWorkspaceId = useScopeStore((s) => s.workspaceId)
  const scopeNamespaceId = useScopeStore((s) => s.namespaceId)

  const [host, setHost] = useState<Host | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  // Assignments (only for platform-level hosts)
  const [assignments, setAssignments] = useState<HostAssignment[]>([])
  const [assignmentsLoading, setAssignmentsLoading] = useState(false)

  const isPlatformScope = !scopeWorkspaceId

  const permPrefix = scopeWorkspaceId && scopeNamespaceId
    ? "infra:workspaces:namespaces:hosts"
    : scopeWorkspaceId
      ? "infra:workspaces:hosts"
      : "infra:hosts"

  const permScope = scopeNamespaceId
    ? { workspaceId: scopeWorkspaceId!, namespaceId: scopeNamespaceId }
    : scopeWorkspaceId
      ? { workspaceId: scopeWorkspaceId }
      : undefined

  const fetchHost = useCallback(async () => {
    if (!hostId) return
    try {
      let h
      if (scopeWorkspaceId && scopeNamespaceId) {
        h = await getNamespaceHost(scopeWorkspaceId, scopeNamespaceId, hostId)
      } else if (scopeWorkspaceId) {
        h = await getWorkspaceHost(scopeWorkspaceId, hostId)
      } else {
        h = await getHost(hostId)
      }
      setHost(h)
    } catch {
      setHost(null)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hostId, scopeWorkspaceId, scopeNamespaceId])

  const fetchAssignments = useCallback(async () => {
    if (!hostId || !isPlatformScope) return
    setAssignmentsLoading(true)
    try {
      const data = await getHostAssignments(hostId)
      setAssignments(data.items ?? [])
    } catch {
      setAssignments([])
    } finally {
      setAssignmentsLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hostId, isPlatformScope])

  useEffect(() => { fetchHost() }, [fetchHost])
  useEffect(() => { fetchAssignments() }, [fetchAssignments])

  const handleDelete = async () => {
    if (!host) return
    try {
      if (scopeWorkspaceId && scopeNamespaceId) {
        await deleteNamespaceHost(scopeWorkspaceId, scopeNamespaceId, host.metadata.id)
      } else if (scopeWorkspaceId) {
        await deleteWorkspaceHost(scopeWorkspaceId, host.metadata.id)
      } else {
        await deleteHost(host.metadata.id)
      }
      toast.success(t("action.deleteSuccess"))
      navigate("..")
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("host.title") }) : err.message)
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

  if (!host) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("host.noData")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{host.metadata.name}</h1>
          <Badge variant={host.spec.status === "active" ? "default" : "secondary"}>
            {host.spec.status === "active" ? t("common.active") : t("common.inactive")}
          </Badge>
          {host.spec.environmentName && (
            <Badge variant="outline">{host.spec.environmentName}</Badge>
          )}
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
        {/* Hardware overview */}
        <div className="grid grid-cols-3 gap-4">
          <Card>
            <CardContent className="flex items-center gap-4 p-4">
              <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
                <Cpu className="text-primary h-5 w-5" />
              </div>
              <div>
                <p className="text-2xl font-bold">{host.spec.cpuCores || 0}</p>
                <p className="text-muted-foreground text-sm">{t("host.cpuCores")}</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-4 p-4">
              <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
                <MemoryStick className="text-primary h-5 w-5" />
              </div>
              <div>
                <p className="text-2xl font-bold">{host.spec.memoryMb || 0}</p>
                <p className="text-muted-foreground text-sm">{t("host.memoryMb")}</p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="flex items-center gap-4 p-4">
              <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
                <HardDrive className="text-primary h-5 w-5" />
              </div>
              <div>
                <p className="text-2xl font-bold">{host.spec.diskGb || 0}</p>
                <p className="text-muted-foreground text-sm">{t("host.diskGb")}</p>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Basic info */}
        <Card>
          <CardHeader>
            <CardTitle>{t("host.basicInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{host.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.displayName")}</span>
                <p className="font-medium">{host.spec.displayName || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("host.hostname")}</span>
                <p className="font-medium">{host.spec.hostname || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("host.ipAddress")}</span>
                <p className="font-medium">{host.spec.ipAddress || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("host.os")}</span>
                <p className="font-medium">{host.spec.os || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("host.arch")}</span>
                <p className="font-medium">{host.spec.arch || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("host.environment")}</span>
                <p className="font-medium">{host.spec.environmentName || t("host.environmentNone")}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.status")}</span>
                <p>
                  <Badge variant={host.spec.status === "active" ? "default" : "secondary"}>
                    {host.spec.status === "active" ? t("common.active") : t("common.inactive")}
                  </Badge>
                </p>
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("common.description")}</span>
                <p className="font-medium">{host.spec.description || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(host.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(host.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Assignments table (platform level only) */}
        {isPlatformScope && (
          <Card>
            <CardHeader>
              <CardTitle>{t("host.assignments")}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("host.selectWorkspace")}</TableHead>
                      <TableHead>{t("host.selectNamespace")}</TableHead>
                      <TableHead>{t("common.created")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {assignmentsLoading ? (
                      Array.from({ length: 2 }).map((_, i) => (
                        <TableRow key={i}>
                          {Array.from({ length: 3 }).map((_, j) => (
                            <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                          ))}
                        </TableRow>
                      ))
                    ) : assignments.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={3} className="text-muted-foreground py-8 text-center">
                          {t("host.assignmentsEmpty")}
                        </TableCell>
                      </TableRow>
                    ) : (
                      assignments.map((a) => (
                        <TableRow key={a.metadata.id}>
                          <TableCell>{a.spec.workspaceName || a.spec.workspaceId || "-"}</TableCell>
                          <TableCell>{a.spec.namespaceName || a.spec.namespaceId || "-"}</TableCell>
                          <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                            {new Date(a.metadata.createdAt).toLocaleString()}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Edit dialog */}
      <EditHostDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        host={host}
        onSuccess={fetchHost}
        scopeWorkspaceId={scopeWorkspaceId}
        scopeNamespaceId={scopeNamespaceId}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("host.deleteConfirm", { name: host.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Edit Host Dialog =====

function EditHostDialog({
  open, onOpenChange, host, onSuccess, scopeWorkspaceId, scopeNamespaceId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  host: Host
  onSuccess: () => void
  scopeWorkspaceId: string | null
  scopeNamespaceId: string | null
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    displayName: z.string().optional(),
    description: z.string().optional(),
    hostname: z.string().optional(),
    ipAddress: z.string().optional(),
    os: z.string().optional(),
    arch: z.string().optional(),
    cpuCores: z.string().optional(),
    memoryMb: z.string().optional(),
    diskGb: z.string().optional(),
    status: z.enum(["active", "inactive"]),
  })

  type FormValues = z.infer<typeof schema>

  const form = useForm<FormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      displayName: host.spec.displayName ?? "",
      description: host.spec.description ?? "",
      hostname: host.spec.hostname ?? "",
      ipAddress: host.spec.ipAddress ?? "",
      os: host.spec.os ?? "",
      arch: host.spec.arch ?? "",
      cpuCores: host.spec.cpuCores ? String(host.spec.cpuCores) : "",
      memoryMb: host.spec.memoryMb ? String(host.spec.memoryMb) : "",
      diskGb: host.spec.diskGb ? String(host.spec.diskGb) : "",
      status: (host.spec.status as "active" | "inactive") ?? "active",
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        displayName: host.spec.displayName ?? "",
        description: host.spec.description ?? "",
        hostname: host.spec.hostname ?? "",
        ipAddress: host.spec.ipAddress ?? "",
        os: host.spec.os ?? "",
        arch: host.spec.arch ?? "",
        cpuCores: host.spec.cpuCores ? String(host.spec.cpuCores) : "",
        memoryMb: host.spec.memoryMb ? String(host.spec.memoryMb) : "",
        diskGb: host.spec.diskGb ? String(host.spec.diskGb) : "",
        status: (host.spec.status as "active" | "inactive") ?? "active",
      })
    }
  }, [open, host, form])

  const onSubmit = async (values: FormValues) => {
    setLoading(true)
    try {
      const spec: Host["spec"] = {
        ...host.spec,
        displayName: values.displayName,
        description: values.description,
        hostname: values.hostname,
        ipAddress: values.ipAddress,
        os: values.os,
        arch: values.arch,
        cpuCores: values.cpuCores ? Number(values.cpuCores) : undefined,
        memoryMb: values.memoryMb ? Number(values.memoryMb) : undefined,
        diskGb: values.diskGb ? Number(values.diskGb) : undefined,
        status: values.status,
      }

      const payload = { metadata: host.metadata, spec }

      if (scopeWorkspaceId && scopeNamespaceId) {
        await updateNamespaceHost(scopeWorkspaceId, scopeNamespaceId, host.metadata.id, payload)
      } else if (scopeWorkspaceId) {
        await updateWorkspaceHost(scopeWorkspaceId, host.metadata.id, payload)
      } else {
        await updateHost(host.metadata.id, payload)
      }
      toast.success(t("action.updateSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError) {
        form.setError("root", {
          message: translateApiError(err) !== err.message
            ? t(translateApiError(err), { resource: t("host.title") })
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
      <DialogContent className="max-w-lg" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("host.edit")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div>
              <label className="text-sm font-medium">{t("host.name")}</label>
              <Input value={host.metadata.name} disabled className="mt-1" />
            </div>
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem><FormLabel>{t("host.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem><FormLabel>{t("host.description")}</FormLabel><FormControl><Textarea rows={2} {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <div className="grid grid-cols-2 gap-4">
              <FormField control={form.control} name="hostname" render={({ field }) => (
                <FormItem><FormLabel>{t("host.hostname")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
              )} />
              <FormField control={form.control} name="ipAddress" render={({ field }) => (
                <FormItem><FormLabel>{t("host.ipAddress")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
              )} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <FormField control={form.control} name="os" render={({ field }) => (
                <FormItem><FormLabel>{t("host.os")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
              )} />
              <FormField control={form.control} name="arch" render={({ field }) => (
                <FormItem><FormLabel>{t("host.arch")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
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
