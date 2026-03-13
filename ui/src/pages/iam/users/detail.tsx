import { useCallback, useEffect, useState } from "react"
import { useParams, useNavigate, Link } from "react-router"
import { Pencil, Trash2, Search, Filter } from "lucide-react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
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
import { getUser, updateUser, deleteUser, listUsers, listUserWorkspaces, listUserNamespaces } from "@/api/iam/users"
import { listUserRoleBindings } from "@/api/iam/rbac"
import { ApiError, showApiError, translateApiError, translateDetailMessage } from "@/api/client"
import type { User, Workspace, Namespace, RoleBinding, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { usePermission } from "@/hooks/use-permission"
import { ConfirmDialog } from "@/components/confirm-dialog"

export default function UserDetailPage() {
  const { userId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const { hasPermission } = usePermission()
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const fetchUser = useCallback(async () => {
    if (!userId) return
    try {
      const u = await getUser(userId)
      setUser(u)
    } catch {
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [userId])

  useEffect(() => { fetchUser() }, [fetchUser])

  const handleDelete = async () => {
    if (!user) return
    try {
      await deleteUser(user.metadata.id)
      toast.success(t("action.deleteSuccess"))
      navigate("/iam/users")
    } catch (err) {
      showApiError(err, t, "user.title")
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

  if (!user) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("user.notFound")}</p>
      </div>
    )
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{user.spec.username}</h1>
          <Badge variant={user.spec.status === "active" ? "default" : "secondary"}>
            {user.spec.status === "active" ? t("common.active") : t("common.inactive")}
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          {hasPermission("iam:users:update") && (
            <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              {t("common.edit")}
            </Button>
          )}
          {hasPermission("iam:users:delete") && (
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("common.delete")}
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-6">
      {/* user info card */}
      <Card>
        <CardHeader>
          <CardTitle>{t("user.details")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
            <div>
              <span className="text-muted-foreground">{t("user.username")}</span>
              <p className="font-medium">{user.spec.username}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.displayName")}</span>
              <p className="font-medium">{user.spec.displayName || "-"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("user.email")}</span>
              <p className="font-medium">{user.spec.email}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.phone")}</span>
              <p className="font-medium">{user.spec.phone || "-"}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.status")}</span>
              <p>
                <Badge variant={user.spec.status === "active" ? "default" : "secondary"}>
                  {user.spec.status === "active" ? t("common.active") : t("common.inactive")}
                </Badge>
              </p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.created")}</span>
              <p className="font-medium">{new Date(user.metadata.createdAt).toLocaleString()}</p>
            </div>
            <div>
              <span className="text-muted-foreground">{t("common.updated")}</span>
              <p className="font-medium">{new Date(user.metadata.updatedAt).toLocaleString()}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* associated workspaces */}
      <UserWorkspacesCard userId={user.metadata.id} />

      {/* associated namespaces */}
      <UserNamespacesCard userId={user.metadata.id} />

      {/* role bindings */}
      <UserRoleBindingsCard userId={user.metadata.id} />
      </div>

      {/* edit dialog */}
      <EditUserDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        user={user}
        onSuccess={fetchUser}
      />

      {/* delete confirm */}
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("user.deleteConfirm", { name: user.spec.username })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Joined Workspaces Card =====

function UserWorkspacesCard({ userId }: { userId: string }) {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
  } = useListState({ defaultSortBy: "joined_at", defaultSortOrder: "desc", defaultPageSize: 10 })
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [statusFilter, setStatusFilter] = useState("all")

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter
      const data = await listUserWorkspaces(userId, params)
      setWorkspaces(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t, "workspace.title")
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userId, page, pageSize, sortBy, sortOrder, search, statusFilter])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, statusFilter, pageSize])

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("user.workspaces")}</CardTitle>
      </CardHeader>
      <CardContent>
        {/* toolbar */}
        <div className="mb-4 flex items-center gap-3">
          <div className="relative max-w-xs">
            <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
            <Input
              placeholder={t("common.search")}
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>

        {/* table */}
        <div className="border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                  {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("display_name")}>
                  {t("common.displayName")}<SortIcon field="display_name" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead>{t("workspace.owner")}</TableHead>
                <TableHead className="cursor-pointer select-none text-center" onClick={() => handleSort("namespace_count")}>
                  {t("workspace.namespaceCount")}<SortIcon field="namespace_count" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none text-center" onClick={() => handleSort("member_count")}>
                  {t("workspace.memberCount")}<SortIcon field="member_count" sortBy={sortBy} sortOrder={sortOrder} />
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
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("role_name")}>
                  {t("user.role")}<SortIcon field="role_name" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("joined_at")}>
                  {t("user.joinedAt")}<SortIcon field="joined_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                  {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("updated_at")}>
                  {t("common.updated")}<SortIcon field="updated_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 3 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 10 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : workspaces.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={10} className="py-10 text-center text-muted-foreground">
                    {t("user.noWorkspaces")}
                  </TableCell>
                </TableRow>
              ) : (
                workspaces.map((ws) => (
                  <TableRow key={ws.metadata.id}>
                    <TableCell className="font-medium">
                      <Link to={`/iam/workspaces/${ws.metadata.id}`} className="text-primary hover:underline">
                        {ws.metadata.name}
                      </Link>
                    </TableCell>
                    <TableCell>{ws.spec.displayName || "-"}</TableCell>
                    <TableCell>{ws.spec.ownerName || "-"}</TableCell>
                    <TableCell className="text-center">{ws.spec.namespaceCount ?? 0}</TableCell>
                    <TableCell className="text-center">{ws.spec.memberCount ?? 0}</TableCell>
                    <TableCell>
                      <Badge variant={ws.spec.status === "active" ? "default" : "secondary"}>
                        {ws.spec.status === "active" ? t("common.active") : t("common.inactive")}
                      </Badge>
                    </TableCell>
                    <TableCell><Badge variant="outline">{t(`role.${ws.spec.role}`, { defaultValue: ws.spec.roleDisplayName || ws.spec.role || "" })}</Badge></TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {ws.spec.joinedAt ? new Date(ws.spec.joinedAt).toLocaleString() : "-"}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {new Date(ws.metadata.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {new Date(ws.metadata.updatedAt).toLocaleString()}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {/* pagination */}
        {totalCount > 0 && (
          <Pagination
            page={page}
            pageSize={pageSize}
            totalCount={totalCount}
            onPageChange={setPage}
            onPageSizeChange={setPageSize}
          />
        )}
      </CardContent>
    </Card>
  )
}

// ===== Joined Namespaces Card =====

function UserNamespacesCard({ userId }: { userId: string }) {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
  } = useListState({ defaultSortBy: "joined_at", defaultSortOrder: "desc", defaultPageSize: 10 })
  const [namespaces, setNamespaces] = useState<Namespace[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [statusFilter, setStatusFilter] = useState("all")

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (statusFilter !== "all") params.status = statusFilter
      const data = await listUserNamespaces(userId, params)
      setNamespaces(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t, "namespace.title")
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userId, page, pageSize, sortBy, sortOrder, search, statusFilter])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, statusFilter, pageSize])

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("user.namespaceRefs")}</CardTitle>
      </CardHeader>
      <CardContent>
        {/* toolbar */}
        <div className="mb-4 flex items-center gap-3">
          <div className="relative max-w-xs">
            <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
            <Input
              placeholder={t("common.search")}
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>

        {/* table */}
        <div className="border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                  {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("display_name")}>
                  {t("common.displayName")}<SortIcon field="display_name" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead>{t("namespace.workspaceName")}</TableHead>
                <TableHead>{t("namespace.owner")}</TableHead>
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
                <TableHead className="cursor-pointer select-none text-center" onClick={() => handleSort("member_count")}>
                  {t("namespace.memberCount")}<SortIcon field="member_count" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("role_name")}>
                  {t("user.role")}<SortIcon field="role_name" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("joined_at")}>
                  {t("user.joinedAt")}<SortIcon field="joined_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                  {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("updated_at")}>
                  {t("common.updated")}<SortIcon field="updated_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 3 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 10 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : namespaces.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={10} className="py-10 text-center text-muted-foreground">
                    {t("user.noNamespaceRefs")}
                  </TableCell>
                </TableRow>
              ) : (
                namespaces.map((ns) => (
                  <TableRow key={ns.metadata.id}>
                    <TableCell className="font-medium">
                      <Link to={`/iam/namespaces/${ns.metadata.id}`} className="text-primary hover:underline">
                        {ns.metadata.name}
                      </Link>
                    </TableCell>
                    <TableCell>{ns.spec.displayName || "-"}</TableCell>
                    <TableCell>
                      {ns.spec.workspaceName ? (
                        <Link to={`/iam/workspaces/${ns.spec.workspaceId}`} className="text-primary hover:underline">
                          {ns.spec.workspaceName}
                        </Link>
                      ) : "-"}
                    </TableCell>
                    <TableCell>{ns.spec.ownerName || "-"}</TableCell>
                    <TableCell>
                      <Badge variant={ns.spec.status === "active" ? "default" : "secondary"}>
                        {ns.spec.status === "active" ? t("common.active") : t("common.inactive")}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-center">{ns.spec.memberCount ?? 0}</TableCell>
                    <TableCell><Badge variant="outline">{t(`role.${ns.spec.role}`, { defaultValue: ns.spec.roleDisplayName || ns.spec.role || "" })}</Badge></TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {ns.spec.joinedAt ? new Date(ns.spec.joinedAt).toLocaleString() : "-"}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {new Date(ns.metadata.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {new Date(ns.metadata.updatedAt).toLocaleString()}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {/* pagination */}
        {totalCount > 0 && (
          <Pagination
            page={page}
            pageSize={pageSize}
            totalCount={totalCount}
            onPageChange={setPage}
            onPageSizeChange={setPageSize}
          />
        )}
      </CardContent>
    </Card>
  )
}

// ===== User Role Bindings Card =====

function UserRoleBindingsCard({ userId }: { userId: string }) {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
  } = useListState({ defaultSortBy: "created_at", defaultSortOrder: "desc", defaultPageSize: 10 })
  const [bindings, setBindings] = useState<RoleBinding[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)
  const [scopeFilter, setScopeFilter] = useState("all")

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search
      if (scopeFilter !== "all") params.scope = scopeFilter
      const data = await listUserRoleBindings(userId, params)
      setBindings(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t, "rolebinding.title")
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userId, page, pageSize, sortBy, sortOrder, search, scopeFilter])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, scopeFilter, pageSize])

  const scopeLabel = (scope: string) => {
    if (scope === "platform") return t("rolebinding.scope.platform")
    if (scope === "workspace") return t("rolebinding.scope.workspace")
    return t("rolebinding.scope.namespace")
  }

  const scopeVariant = (scope: string): "default" | "secondary" | "outline" => {
    if (scope === "platform") return "default"
    if (scope === "workspace") return "secondary"
    return "outline"
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("user.rolebindings")}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="mb-4 flex items-center gap-3">
          <div className="relative max-w-xs">
            <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
            <Input
              placeholder={t("common.search")}
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>
        <div className="border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("role_name")}>
                  {t("role.title")}<SortIcon field="role_name" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button className="inline-flex items-center gap-1 select-none">
                        {t("rolebinding.scope")}
                        <Filter className={`h-3 w-3 ${scopeFilter !== "all" ? "text-primary" : "opacity-40"}`} />
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="start">
                      <DropdownMenuItem onClick={() => setScopeFilter("all")}>{t("common.all")}</DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setScopeFilter("platform")}>{t("rolebinding.scope.platform")}</DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setScopeFilter("workspace")}>{t("rolebinding.scope.workspace")}</DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setScopeFilter("namespace")}>{t("rolebinding.scope.namespace")}</DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("scope_target")}>
                  {t("rolebinding.scopeTarget")}<SortIcon field="scope_target" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                  {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 3 }).map((_, i) => (
                  <TableRow key={i}>{Array.from({ length: 4 }).map((_, j) => (<TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>))}</TableRow>
                ))
              ) : bindings.length === 0 ? (
                <TableRow><TableCell colSpan={4} className="text-muted-foreground py-8 text-center">{t("user.noRolebindings")}</TableCell></TableRow>
              ) : (
                bindings.map((b) => (
                  <TableRow key={b.metadata.id}>
                    <TableCell>
                      <Badge variant="secondary">
                        {t(`role.${b.spec.roleName}`, { defaultValue: b.spec.roleDisplayName || b.spec.roleName || "" })}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={scopeVariant(b.spec.scope)}>{scopeLabel(b.spec.scope)}</Badge>
                    </TableCell>
                    <TableCell className="text-sm">
                      {b.spec.scope === "platform" ? (
                        t("rolebinding.scope.platform")
                      ) : b.spec.scope === "namespace" ? (
                        <span>
                          {b.spec.workspaceName && (
                            <Link to={`/iam/workspaces/${b.spec.workspaceId}`} className="text-primary hover:underline">{b.spec.workspaceName}</Link>
                          )}
                          {b.spec.workspaceName && b.spec.namespaceName && " / "}
                          {b.spec.namespaceName && (
                            <Link to={`/iam/workspaces/${b.spec.workspaceId}/namespaces/${b.spec.namespaceId}`} className="text-primary hover:underline">{b.spec.namespaceName}</Link>
                          )}
                        </span>
                      ) : (
                        b.spec.workspaceName ? (
                          <Link to={`/iam/workspaces/${b.spec.workspaceId}`} className="text-primary hover:underline">{b.spec.workspaceName}</Link>
                        ) : "-"
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {new Date(b.metadata.createdAt).toLocaleString()}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
        {totalCount > 0 && (
          <Pagination page={page} pageSize={pageSize} totalCount={totalCount} onPageChange={setPage} onPageSizeChange={setPageSize} />
        )}
      </CardContent>
    </Card>
  )
}

// ===== Edit User Dialog =====

function EditUserDialog({
  open, onOpenChange, user, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: User
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    email: z.email(t("api.validation.email.format")),
    displayName: z.string().optional(),
    phone: z.string()
      .min(1, t("api.validation.required", { field: t("common.phone") }))
      .regex(/^1[3-9]\d{9}$/, t("api.validation.phone.format")),
    status: z.enum(["active", "inactive"]),
  })

  type FormValues = z.infer<typeof schema>

  const form = useForm<FormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: {
      email: user.spec.email,
      displayName: user.spec.displayName ?? "",
      phone: user.spec.phone ?? "",
      status: user.spec.status ?? "active",
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        email: user.spec.email,
        displayName: user.spec.displayName ?? "",
        phone: user.spec.phone ?? "",
        status: user.spec.status ?? "active",
      })
    }
  }, [open, user, form])

  const checkUniqueness = async (field: "email" | "phone", value: string) => {
    if (!value) return
    try {
      const data = await listUsers({ page: 1, pageSize: 1, [field]: value })
      const exists = data.items?.some((u) => {
        if (u.metadata.id === user.metadata.id) return false
        return u.spec[field]?.toLowerCase() === value.toLowerCase()
      })
      if (exists) form.setError(field, { message: t(`api.validation.${field}.taken`) })
    } catch { /* backend will enforce */ }
  }

  const onSubmit = async (values: FormValues) => {
    setLoading(true)
    try {
      await updateUser(user.metadata.id, {
        metadata: user.metadata,
        spec: {
          ...user.spec,
          email: values.email,
          displayName: values.displayName,
          phone: values.phone,
          status: values.status,
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
          form.setError(field, { message: i18nKey !== d.message ? t(i18nKey, { field: t(`user.${field}`) || field }) : d.message })
        }
      } else if (err instanceof ApiError) {
        form.setError("root", { message: translateApiError(err) !== err.message ? t(translateApiError(err), { resource: t("user.title") }) : err.message })
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
          <DialogTitle>{t("user.edit")}</DialogTitle>
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
              <label className="text-sm font-medium">{t("user.username")}</label>
              <Input value={user.spec.username} disabled className="mt-1" />
            </div>
            <FormField control={form.control} name="email" render={({ field }) => (
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
            )} />
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem>
                <FormLabel>{t("common.displayName")}</FormLabel>
                <FormControl><Input {...field} /></FormControl>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="phone" render={({ field }) => (
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
