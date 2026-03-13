import { useCallback, useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { Pencil, Trash2, FolderKanban, Users, ShieldCheck } from "lucide-react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useScopeStore } from "@/stores/scope-store"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
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
import { getWorkspace, updateWorkspace, deleteWorkspace } from "@/api/iam/workspaces"
import { ApiError, translateApiError } from "@/api/client"
import type { Workspace } from "@/api/types"
import { OverviewCard } from "@/components/overview-card"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useWorkspaceStore } from "@/stores/workspace-store"

export default function WorkspaceDetailPage() {
  const { workspaceId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()
  const setCurrentWorkspace = useWorkspaceStore((s) => s.setCurrentWorkspace)
  const [workspace, setWorkspace] = useState<Workspace | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const fetchWorkspace = useCallback(async () => {
    if (!workspaceId) return
    try {
      const ws = await getWorkspace(workspaceId)
      setWorkspace(ws)
      setCurrentWorkspace(workspaceId, ws.metadata.name)
    } catch {
      setWorkspace(null)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspaceId])

  useEffect(() => { fetchWorkspace() }, [fetchWorkspace])

  const handleDelete = async () => {
    if (!workspace) return
    try {
      await deleteWorkspace(workspace.metadata.id)
      useScopeStore.getState().invalidate()
      toast.success(t("action.deleteSuccess"))
      navigate("/iam/workspaces")
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("workspace.title") }) : err.message)
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

  if (!workspace) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("workspace.notFound")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{workspace.metadata.name}</h1>
          <Badge variant={workspace.spec.status === "active" ? "default" : "secondary"}>
            {workspace.spec.status === "active" ? t("common.active") : t("common.inactive")}
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          {hasPermission("iam:workspaces:update", { workspaceId }) && (
            <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              {t("common.edit")}
            </Button>
          )}
          {hasPermission("iam:workspaces:delete", { workspaceId }) && (
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("common.delete")}
            </Button>
          )}
        </div>
      </div>

      {/* Overview content */}
      <div className="space-y-6">
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
          <OverviewCard label={t("workspace.namespaces")} icon={FolderKanban} value={workspace.spec.namespaceCount ?? 0} onClick={() => navigate("namespaces")} />
          <OverviewCard label={t("workspace.members")} icon={Users} value={workspace.spec.memberCount ?? 0} onClick={() => navigate("users")} />
          <OverviewCard label={t("rolebinding.title")} icon={ShieldCheck} value={workspace.spec.roleBindingCount ?? 0} onClick={() => navigate("rolebindings")} />
        </div>
        <Card>
          <CardHeader>
            <CardTitle>{t("workspace.details")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{workspace.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.displayName")}</span>
                <p className="font-medium">{workspace.spec.displayName || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("workspace.owner")}</span>
                <p className="font-medium">{workspace.spec.ownerName || workspace.spec.ownerId}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.status")}</span>
                <p>
                  <Badge variant={workspace.spec.status === "active" ? "default" : "secondary"}>
                    {workspace.spec.status === "active" ? t("common.active") : t("common.inactive")}
                  </Badge>
                </p>
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("common.description")}</span>
                <p className="font-medium">{workspace.spec.description || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(workspace.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(workspace.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* edit dialog */}
      <EditWorkspaceDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        workspace={workspace}
        onSuccess={fetchWorkspace}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("workspace.deleteConfirm", { name: workspace.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Edit Workspace Dialog =====

function EditWorkspaceDialog({
  open, onOpenChange, workspace, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  workspace: Workspace
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    displayName: z.string().optional(),
    description: z.string().optional(),
    status: z.enum(["active", "inactive"]),
  })

  type FormValues = z.infer<typeof schema>

  const form = useForm<FormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      displayName: workspace.spec.displayName ?? "",
      description: workspace.spec.description ?? "",
      status: workspace.spec.status ?? "active",
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        displayName: workspace.spec.displayName ?? "",
        description: workspace.spec.description ?? "",
        status: workspace.spec.status ?? "active",
      })
    }
  }, [open, workspace, form])

  const onSubmit = async (values: FormValues) => {
    setLoading(true)
    try {
      await updateWorkspace(workspace.metadata.id, {
        metadata: workspace.metadata,
        spec: { ...workspace.spec, displayName: values.displayName, description: values.description, status: values.status },
      })
      useScopeStore.getState().invalidate()
      toast.success(t("action.updateSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError) {
        form.setError("root", { message: translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("workspace.title") }) : err.message })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] flex flex-col overflow-hidden" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("workspace.edit")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex min-h-0 flex-1 flex-col overflow-hidden">
            <div className="space-y-4 overflow-y-auto flex-1 min-h-0">
              {form.formState.errors.root && (
                <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {form.formState.errors.root.message}
                </div>
              )}
              <div>
                <label className="text-sm font-medium">{t("common.name")}</label>
                <Input value={workspace.metadata.name} disabled className="mt-1" />
              </div>
              <FormField control={form.control} name="displayName" render={({ field }) => (
                <FormItem><FormLabel>{t("common.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
              )} />
              <FormField control={form.control} name="description" render={({ field }) => (
                <FormItem><FormLabel>{t("common.description")}</FormLabel><FormControl><Textarea rows={3} {...field} /></FormControl><FormMessage /></FormItem>
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
