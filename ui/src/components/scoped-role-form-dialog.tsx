import { useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  createWorkspaceRole, updateWorkspaceRole, listWorkspaceRoles,
  createNamespaceRole, updateNamespaceRole, listNamespaceRoles,
  getWorkspaceRole, getNamespaceRole,
} from "@/api/iam/rbac"
import { ApiError, translateApiError, translateDetailMessage } from "@/api/client"
import type { Role, Permission } from "@/api/types"
import { useTranslation } from "@/i18n"
import { PermissionSelector } from "@/components/permission-selector"

interface RoleFormValues {
  name: string
  displayName: string
  description: string
  rules: string[]
}

export function ScopedRoleFormDialog({
  open,
  onOpenChange,
  scope,
  scopeId,
  workspaceId,
  role,
  permissions,
  onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  scope: "workspace" | "namespace"
  scopeId: string
  workspaceId?: string
  role?: Role
  permissions: Permission[]
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const isEdit = !!role
  const [loading, setLoading] = useState(false)
  const [fullRole, setFullRole] = useState<Role | null>(null)

  const roleFormSchema = z.object({
    name: isEdit
      ? z.string()
      : z.string()
          .min(3, t("role.validation.name.format"))
          .max(50, t("role.validation.name.format"))
          .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, t("role.validation.name.format")),
    displayName: z.string().optional(),
    description: z.string().optional(),
    rules: z.array(z.string()).min(1, t("role.validation.rules.required")),
  })

  const form = useForm<RoleFormValues>({
    resolver: zodResolver(roleFormSchema) as never,
    mode: "onBlur",
    defaultValues: {
      name: "",
      displayName: "",
      description: "",
      rules: [],
    },
  })

  // Fetch full role data (with rules) when editing — list API only returns ruleCount
  useEffect(() => {
    if (open && role) {
      const fetchFull = async () => {
        try {
          const r = scope === "workspace"
            ? await getWorkspaceRole(scopeId, role.metadata.id)
            : await getNamespaceRole(workspaceId!, scopeId, role.metadata.id)
          setFullRole(r)
          form.reset({
            name: r.spec.name,
            displayName: r.spec.displayName ?? "",
            description: r.spec.description ?? "",
            rules: r.spec.rules ?? [],
          })
        } catch { /* fall back to list data */ }
      }
      fetchFull()
    } else if (open) {
      setFullRole(null)
      form.reset({ name: "", displayName: "", description: "", rules: [] })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, role])

  const checkUniqueness = async (value: string) => {
    if (!value) return
    try {
      const data = scope === "workspace"
        ? await listWorkspaceRoles(scopeId, { page: 1, pageSize: 1, search: value })
        : await listNamespaceRoles(workspaceId!, scopeId, { page: 1, pageSize: 1, search: value })
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
        const editRole = fullRole ?? role
        if (scope === "workspace") {
          await updateWorkspaceRole(scopeId, editRole.metadata.id, {
            metadata: editRole.metadata,
            spec: { ...editRole.spec, displayName: values.displayName || undefined, description: values.description || undefined, rules: values.rules },
          })
        } else {
          await updateNamespaceRole(workspaceId!, scopeId, editRole.metadata.id, {
            metadata: editRole.metadata,
            spec: { ...editRole.spec, displayName: values.displayName || undefined, description: values.description || undefined, rules: values.rules },
          })
        }
        toast.success(t("action.updateSuccess"))
      } else {
        const spec = {
          name: values.name,
          displayName: values.displayName || undefined,
          description: values.description || undefined,
          scope,
          rules: values.rules,
        } as Role["spec"]
        if (scope === "workspace") {
          await createWorkspaceRole(scopeId, { metadata: {} as Role["metadata"], spec })
        } else {
          await createNamespaceRole(workspaceId!, scopeId, { metadata: {} as Role["metadata"], spec })
        }
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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:!max-w-none !w-auto min-w-[800px] max-h-[85vh] flex flex-col overflow-hidden" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
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
              <div className="col-span-1 space-y-4 overflow-y-auto">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel required>{t("role.name")}</FormLabel>
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
                  name="displayName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("common.displayName")}</FormLabel>
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
                      <FormLabel>{t("common.description")}</FormLabel>
                      <FormControl><Textarea {...field} rows={3} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name="rules"
                render={() => (
                  <FormItem className="col-span-2 flex flex-col min-h-0">
                    <FormLabel required>
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
                      scope={scope}
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
