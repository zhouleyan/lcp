import { useCallback, useEffect, useMemo, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { Pencil, Trash2, ArrowLeft } from "lucide-react"
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
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogDescription,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import { getRole, updateRole, deleteRole, listPermissions } from "@/api/rbac"
import { ApiError, translateApiError, translateDetailMessage } from "@/api/client"
import type { Role, Permission } from "@/api/types"
import { useTranslation } from "@/i18n"
import { PermissionSelector, patternCovers } from "@/components/permission-selector"

const SCOPE_VARIANT: Record<string, "default" | "secondary" | "outline"> = {
  platform: "default",
  workspace: "secondary",
  namespace: "outline",
}

export default function RoleDetailPage() {
  const { roleId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [role, setRole] = useState<Role | null>(null)
  const [loading, setLoading] = useState(true)
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const fetchRole = useCallback(async () => {
    if (!roleId) return
    try {
      const r = await getRole(roleId)
      setRole(r)
    } catch {
      setRole(null)
    } finally {
      setLoading(false)
    }
  }, [roleId])

  useEffect(() => { fetchRole() }, [fetchRole])

  useEffect(() => {
    listPermissions({ pageSize: 1000 })
      .then((data) => setPermissions(data.items ?? []))
      .catch(() => {})
  }, [])

  const handleDelete = async () => {
    if (!role) return
    try {
      await deleteRole(role.metadata.id)
      toast.success(t("action.deleteSuccess"))
      navigate("/roles")
    } catch {
      toast.error(t("api.error.internalError"))
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

  if (!role) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("role.notFound")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => navigate("/roles")}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-2xl font-bold">{role.spec.name}</h1>
          <Badge variant={SCOPE_VARIANT[role.spec.scope] ?? "outline"}>
            {t(`role.scope.${role.spec.scope}`)}
          </Badge>
          <Badge variant={role.spec.builtin ? "secondary" : "outline"}>
            {role.spec.builtin ? t("role.builtin") : t("role.custom")}
          </Badge>
        </div>
        {!role.spec.builtin && (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              {t("common.edit")}
            </Button>
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("common.delete")}
            </Button>
          </div>
        )}
      </div>

      {/* role info card */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>{t("role.details")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
            <div>
              <span className="text-muted-foreground">{t("role.name")}</span>
              <p className="font-medium">{role.spec.name}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.displayName")}</span>
              <p className="font-medium">{t(`role.${role.spec.name}`, { defaultValue: role.spec.displayName || "-" })}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("role.scope")}</span>
              <p>
                <Badge variant={SCOPE_VARIANT[role.spec.scope] ?? "outline"}>
                  {t(`role.scope.${role.spec.scope}`)}
                </Badge>
              </p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("role.builtin")}</span>
              <p>
                <Badge variant={role.spec.builtin ? "secondary" : "outline"}>
                  {role.spec.builtin ? t("role.builtin") : t("role.custom")}
                </Badge>
              </p>
            </div>
            <div className="col-span-2">
              <span className="text-muted-foreground">{t("common.description")}</span>
              <p className="font-medium">{t(`role.desc.${role.spec.name}`, { defaultValue: role.spec.description || "-" })}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.created")}</span>
              <p className="font-medium">{new Date(role.metadata.createdAt).toLocaleString()}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.updated")}</span>
              <p className="font-medium">{new Date(role.metadata.updatedAt).toLocaleString()}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* permission rules card */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>
            {t("role.rules")}
            <span className="text-muted-foreground ml-2 text-sm font-normal">
              ({t("role.rulesCount", { count: role.spec.rules?.length ?? 0 })})
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {(!role.spec.rules || role.spec.rules.length === 0) ? (
            <p className="text-muted-foreground text-sm">{t("role.noPermissions")}</p>
          ) : (
            <div className="grid grid-cols-2 gap-6" style={{ height: "min(600px, 60vh)" }}>
              <div className="overflow-y-auto rounded-md border">
                <PermissionSelector
                  permissions={permissions}
                  value={role.spec.rules}
                  readOnly
                />
              </div>
              <div className="overflow-y-auto rounded-md border">
                <MatchedRulesList
                  rules={role.spec.rules}
                  permissions={permissions}
                />
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* edit dialog */}
      {!role.spec.builtin && (
        <EditRoleDialog
          open={editOpen}
          onOpenChange={setEditOpen}
          role={role}
          permissions={permissions}
          onSuccess={fetchRole}
        />
      )}

      {/* delete confirm */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("common.delete")}</DialogTitle>
            <DialogDescription>
              {t("role.deleteConfirm", { name: role.spec.name })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>{t("common.cancel")}</Button>
            <Button variant="destructive" onClick={handleDelete}>{t("common.delete")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// ===== Matched Rules List =====

function MatchedRulesList({
  rules,
  permissions,
}: {
  rules: string[]
  permissions: Permission[]
}) {
  const { t } = useTranslation()

  const ruleDetails = useMemo(() => {
    return rules.map((rule) => {
      const matched = rule.includes("*")
        ? permissions.filter((p) => patternCovers(rule, p.spec.code))
        : permissions.filter((p) => p.spec.code === rule)
      return { rule, matched }
    })
  }, [rules, permissions])

  return (
    <div>
      {ruleDetails.map(({ rule, matched }) => (
        <div key={rule} className="border-b last:border-b-0">
          <div className="flex items-center justify-between bg-muted/30 px-3 py-1.5">
            <code className="text-xs font-medium">{rule}</code>
            <span className="text-muted-foreground text-xs">
              {t("role.matchCount", { count: matched.length })}
            </span>
          </div>
          {matched.length > 0 && (
            <div className="px-3 py-1">
              {matched.map((p) => (
                <p key={p.spec.code} className="text-muted-foreground truncate py-0.5 text-xs">
                  {t(`perm.${p.spec.code}`, { defaultValue: p.spec.description || p.spec.code })}
                </p>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

// ===== Edit Role Dialog =====

function EditRoleDialog({
  open, onOpenChange, role, permissions, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  role: Role
  permissions: Permission[]
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    displayName: z.string().optional(),
    description: z.string().optional(),
    rules: z.array(z.string()).min(1, t("role.validation.rules.required")),
  })

  type FormValues = z.infer<typeof schema>

  const form = useForm<FormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      displayName: role.spec.displayName ?? "",
      description: role.spec.description ?? "",
      rules: role.spec.rules ?? [],
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        displayName: role.spec.displayName ?? "",
        description: role.spec.description ?? "",
        rules: role.spec.rules ?? [],
      })
    }
  }, [open, role, form])

  const selectedRules = form.watch("rules")

  const onSubmit = async (values: FormValues) => {
    setLoading(true)
    try {
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
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError && err.details?.length) {
        for (const d of err.details) {
          const field = d.field.replace(/^spec\./, "") as keyof FormValues
          const i18nKey = translateDetailMessage(d.message)
          form.setError(field, { message: i18nKey !== d.message ? t(i18nKey, { field }) : d.message })
        }
      } else if (err instanceof ApiError) {
        form.setError("root", { message: translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("role.title") }) : err.message })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:!max-w-none !w-auto min-w-[800px] max-h-[85vh] flex flex-col overflow-hidden" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("role.edit")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex min-h-0 flex-col">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive mb-4">
                {form.formState.errors.root.message}
              </div>
            )}
            <div className="grid grid-cols-3 gap-6">
              {/* Left: basic fields */}
              <div className="col-span-1 space-y-4">
                <div>
                  <label className="text-sm font-medium">{t("role.name")}</label>
                  <Input value={role.spec.name} disabled className="mt-1" />
                </div>
                <div>
                  <label className="text-sm font-medium">{t("role.scope")}</label>
                  <Input value={t(`role.scope.${role.spec.scope}`)} disabled className="mt-1" />
                </div>
                <FormField control={form.control} name="displayName" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("common.displayName")}</FormLabel>
                    <FormControl><Input {...field} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={form.control} name="description" render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("common.description")}</FormLabel>
                    <FormControl><Textarea {...field} rows={3} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
              </div>
              {/* Right: permission selector */}
              <FormField
                control={form.control}
                name="rules"
                render={() => (
                  <FormItem className="col-span-2 flex flex-col">
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
                    />
                    <FormMessage />
                  </FormItem>
                )}
              />
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
