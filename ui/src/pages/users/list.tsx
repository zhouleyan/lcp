import { useCallback, useEffect, useRef, useState } from "react"
import { Plus, Pencil, Trash2, ArrowUpDown, ArrowUp, ArrowDown, Search } from "lucide-react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { listUsers, createUser, updateUser, deleteUser, deleteUsers } from "@/api/users"
import { ApiError, translateDetailMessage, translateApiError } from "@/api/client"
import type { User, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"

const PAGE_SIZE = 20

type SortField = "username" | "email" | "created_at"

export default function UserListPage() {
  const { t } = useTranslation()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [page, setPage] = useState(1)
  const [sortBy, setSortBy] = useState<SortField>("created_at")
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc")
  const [searchInput, setSearchInput] = useState("")
  const [search, setSearch] = useState("")
  const [statusFilter, setStatusFilter] = useState<string>("all")
  const [selected, setSelected] = useState<Set<string>>(new Set())

  // dialogs
  const [createOpen, setCreateOpen] = useState(false)
  const [editUser, setEditUser] = useState<User | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const totalPages = Math.max(1, Math.ceil(totalCount / PAGE_SIZE))

  // Debounce search input
  const searchTimer = useRef<ReturnType<typeof setTimeout>>(null)
  useEffect(() => {
    searchTimer.current = setTimeout(() => setSearch(searchInput), 300)
    return () => { if (searchTimer.current) clearTimeout(searchTimer.current) }
  }, [searchInput])

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = {
        page,
        pageSize: PAGE_SIZE,
        sortBy,
        sortOrder,
      }
      if (search) params.username = search
      if (statusFilter !== "all") params.status = statusFilter
      const data = await listUsers(params)
      setUsers(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch {
      toast.error(t("api.error.internalError"))
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, sortBy, sortOrder, search, statusFilter])

  useEffect(() => {
    fetchUsers()
  }, [fetchUsers])

  // reset page when filters change
  useEffect(() => {
    setPage(1)
  }, [search, statusFilter])

  // clear selection on data change
  useEffect(() => {
    setSelected(new Set())
  }, [users])

  const handleSort = (field: SortField) => {
    if (sortBy === field) {
      setSortOrder((o) => (o === "asc" ? "desc" : "asc"))
    } else {
      setSortBy(field)
      setSortOrder("asc")
    }
  }

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortBy !== field) return <ArrowUpDown className="ml-1 inline h-3 w-3 opacity-40" />
    return sortOrder === "asc" ? (
      <ArrowUp className="ml-1 inline h-3 w-3" />
    ) : (
      <ArrowDown className="ml-1 inline h-3 w-3" />
    )
  }

  const toggleAll = () => {
    if (selected.size === users.length) {
      setSelected(new Set())
    } else {
      setSelected(new Set(users.map((u) => u.metadata.id)))
    }
  }

  const toggleOne = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteUser(deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchUsers()
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("user.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  const handleBatchDelete = async () => {
    try {
      await deleteUsers(Array.from(selected))
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      setSelected(new Set())
      fetchUsers()
    } catch (err) {
      if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        toast.error(i18nKey !== err.message ? t(i18nKey, { resource: t("user.title") }) : err.message)
      } else {
        toast.error(t("api.error.internalError"))
      }
    }
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("user.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("user.manage", { count: totalCount })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {selected.size > 0 && (
            <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("user.batchDelete")} ({selected.size})
            </Button>
          )}
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("user.create")}
          </Button>
        </div>
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("user.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("common.all")}</SelectItem>
            <SelectItem value="active">{t("common.active")}</SelectItem>
            <SelectItem value="inactive">{t("common.inactive")}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                <input
                  type="checkbox"
                  className="accent-primary h-4 w-4 rounded"
                  checked={users.length > 0 && selected.size === users.length}
                  onChange={toggleAll}
                />
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("username")}
              >
                {t("user.username")}
                <SortIcon field="username" />
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("email")}
              >
                {t("user.email")}
                <SortIcon field="email" />
              </TableHead>
              <TableHead>{t("common.displayName")}</TableHead>
              <TableHead>{t("common.status")}</TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => handleSort("created_at")}
              >
                {t("common.created")}
                <SortIcon field="created_at" />
              </TableHead>
              <TableHead className="w-24">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 7 }).map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-4 w-20" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : users.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="text-muted-foreground py-8 text-center">
                  {t("user.noData")}
                </TableCell>
              </TableRow>
            ) : (
              users.map((user) => (
                <TableRow key={user.metadata.id}>
                  <TableCell>
                    <input
                      type="checkbox"
                      className="accent-primary h-4 w-4 rounded"
                      checked={selected.has(user.metadata.id)}
                      onChange={() => toggleOne(user.metadata.id)}
                    />
                  </TableCell>
                  <TableCell className="font-medium">{user.spec.username}</TableCell>
                  <TableCell>{user.spec.email}</TableCell>
                  <TableCell>{user.spec.displayName || "-"}</TableCell>
                  <TableCell>
                    <Badge variant={user.spec.status === "active" ? "default" : "secondary"}>
                      {user.spec.status === "active" ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {new Date(user.metadata.createdAt).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => setEditUser(user)}
                        title={t("common.edit")}
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-destructive hover:text-destructive"
                        onClick={() => setDeleteTarget(user)}
                        title={t("common.delete")}
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

      {/* pagination */}
      {totalCount > 0 && (
        <div className="mt-4 flex items-center justify-between">
          <p className="text-muted-foreground text-sm">
            {t("common.total", { count: totalCount })}
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              {t("common.previous")}
            </Button>
            <span className="text-sm">
              {t("common.page", { page, total: totalPages })}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              {t("common.next")}
            </Button>
          </div>
        </div>
      )}

      {/* create dialog */}
      <UserFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchUsers}
      />

      {/* edit dialog */}
      <UserFormDialog
        open={!!editUser}
        onOpenChange={(v) => { if (!v) setEditUser(null) }}
        user={editUser ?? undefined}
        onSuccess={fetchUsers}
      />

      {/* delete confirm */}
      <Dialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("common.delete")}</DialogTitle>
            <DialogDescription>
              {t("user.deleteConfirm", { name: deleteTarget?.spec.username ?? "" })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              {t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* batch delete confirm */}
      <Dialog open={batchDeleteOpen} onOpenChange={setBatchDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("user.batchDelete")}</DialogTitle>
            <DialogDescription>
              {t("user.batchDeleteConfirm", { count: selected.size })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBatchDeleteOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleBatchDelete}>
              {t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// --- User Create/Edit Form Dialog ---

interface UserFormValues {
  username: string
  email: string
  displayName?: string
  phone: string
  password?: string
  status: "active" | "inactive"
}

function UserFormDialog({
  open,
  onOpenChange,
  user,
  onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  user?: User
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const isEdit = !!user
  const [loading, setLoading] = useState(false)

  const userFormSchema = z.object({
    username: z
      .string()
      .min(3, t("api.validation.username.format"))
      .max(50, t("api.validation.username.format"))
      .regex(/^[a-zA-Z0-9_]+$/, t("api.validation.username.format")),
    email: z.email(t("api.validation.email.format")),
    displayName: z.string().optional(),
    phone: z
      .string()
      .min(1, t("api.validation.required", { field: t("common.phone") }))
      .regex(/^1[3-9]\d{9}$/, t("api.validation.phone.format")),
    password: isEdit
      ? z.string().optional()
      : z
          .string()
          .min(8, t("api.validation.password.length"))
          .max(128, t("api.validation.password.length"))
          .regex(/[A-Z]/, t("api.validation.password.uppercase"))
          .regex(/[a-z]/, t("api.validation.password.lowercase"))
          .regex(/[0-9]/, t("api.validation.password.digit")),
    status: z.enum(["active", "inactive"]),
  })

  const form = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema) as never,
    mode: "onBlur",
    defaultValues: {
      username: "",
      email: "",
      displayName: "",
      phone: "",
      password: "",
      status: "active",
    },
  })

  const checkUniqueness = async (field: "username" | "email" | "phone", value: string) => {
    if (!value) return
    try {
      const params: ListParams = { page: 1, pageSize: 1, [field]: value }
      const data = await listUsers(params)
      const exists = data.items?.some((u) => {
        if (isEdit && u.metadata.id === user?.metadata.id) return false
        return u.spec[field]?.toLowerCase() === value.toLowerCase()
      })
      if (exists) {
        form.setError(field, { message: t(`api.validation.${field}.taken`) })
      }
    } catch {
      // uniqueness will be enforced on submit by backend
    }
  }

  // reset form when dialog opens with user data
  useEffect(() => {
    if (open) {
      if (user) {
        form.reset({
          username: user.spec.username,
          email: user.spec.email,
          displayName: user.spec.displayName ?? "",
          phone: user.spec.phone ?? "",
          password: "",
          status: user.spec.status ?? "active",
        })
      } else {
        form.reset({
          username: "",
          email: "",
          displayName: "",
          phone: "",
          password: "",
          status: "active",
        })
      }
    }
  }, [open, user, form])

  const onSubmit = async (values: UserFormValues) => {
    setLoading(true)
    try {
      const spec = {
        username: values.username,
        email: values.email,
        displayName: values.displayName || undefined,
        phone: values.phone,
        status: values.status,
      } as User["spec"]

      if (isEdit) {
        await updateUser(user.metadata.id, {
          metadata: user.metadata,
          spec,
        })
        toast.success(t("action.updateSuccess"))
      } else {
        // include password for creation
        const createSpec: Record<string, unknown> = { ...spec }
        if (values.password) createSpec.password = values.password
        await createUser({
          metadata: {} as User["metadata"],
          spec: createSpec as unknown as User["spec"],
        })
        toast.success(t("action.createSuccess"))
      }
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError && err.details?.length) {
        for (const d of err.details) {
          const field = d.field.replace(/^spec\./, "") as keyof UserFormValues
          const i18nKey = translateDetailMessage(d.message)
          form.setError(field, { message: i18nKey !== d.message ? t(i18nKey, { field: t(`user.${field}`) || field }) : d.message })
        }
      } else if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        form.setError("root", { message: i18nKey !== err.message ? t(i18nKey, { resource: t("user.title") }) : err.message })
      } else {
        form.setError("root", { message: t("api.error.internalError") })
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onOpenAutoFocus={(e) => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>{isEdit ? t("user.edit") : t("user.create")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <FormField
              control={form.control}
              name="username"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("user.username")}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      disabled={isEdit}
                      onBlur={async (e) => {
                        field.onBlur()
                        if (isEdit || !e.target.value) return
                        const valid = await form.trigger("username")
                        if (valid) checkUniqueness("username", e.target.value)
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("user.email")}</FormLabel>
                  <FormControl>
                    <Input
                      type="email"
                      {...field}
                      onBlur={async (e) => {
                        field.onBlur()
                        if (!e.target.value) return
                        const valid = await form.trigger("email")
                        if (valid) checkUniqueness("email", e.target.value)
                      }}
                    />
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
              name="phone"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("common.phone")}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      onBlur={async (e) => {
                        field.onBlur()
                        if (!e.target.value) return
                        const valid = await form.trigger("phone")
                        if (valid) checkUniqueness("phone", e.target.value)
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {!isEdit && (
              <FormField
                control={form.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("common.password")}</FormLabel>
                    <FormControl>
                      <Input type="password" {...field} />
                    </FormControl>
                    <FormDescription>{t("api.validation.password.hint")}</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            <FormField
              control={form.control}
              name="status"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t("common.status")}</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
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
            <DialogFooter>
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
