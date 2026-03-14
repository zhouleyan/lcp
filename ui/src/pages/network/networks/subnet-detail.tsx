import React, { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { Plus, Pencil, Trash2, Filter } from "lucide-react"
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
import { Checkbox } from "@/components/ui/checkbox"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Form, FormControl, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { getSubnet, updateSubnet, deleteSubnet } from "@/api/network/subnets"
import { listAllocations, createAllocation, deleteAllocation } from "@/api/network/allocations"
import { ApiError, showApiError, translateApiError, translateDetailMessage } from "@/api/client"
import type { Subnet, IPAllocation, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { cidrUsableRange } from "./utils"

export default function SubnetDetailPage() {
  const { networkId, subnetId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const { hasPermission } = usePermission()

  const [subnet, setSubnet] = useState<Subnet | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const fetchSubnet = useCallback(async () => {
    if (!networkId || !subnetId) return
    try {
      const s = await getSubnet(networkId, subnetId)
      setSubnet(s)
    } catch {
      setSubnet(null)
    } finally {
      setLoading(false)
    }
  }, [networkId, subnetId])

  useEffect(() => { fetchSubnet() }, [fetchSubnet])

  const handleDelete = async () => {
    if (!subnet || !networkId) return
    try {
      await deleteSubnet(networkId, subnet.metadata.id)
      toast.success(t("action.deleteSuccess"))
      navigate("..")
    } catch (err) {
      showApiError(err, t, "subnet.title")
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

  if (!subnet) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("subnet.noData")}</p>
      </div>
    )
  }

  const usedIPs = subnet.spec.usedIPs ?? 0
  const totalIPs = subnet.spec.totalIPs ?? 0
  const freeIPs = subnet.spec.freeIPs ?? 0
  const usagePercent = totalIPs > 0 ? Math.round((usedIPs / totalIPs) * 100) : 0

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{subnet.metadata.name}</h1>
          <Badge variant="outline" className="font-mono">{subnet.spec.cidr}</Badge>
        </div>
        <div className="flex items-center gap-2">
          {hasPermission("network:subnets:update") && (
            <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              {t("common.edit")}
            </Button>
          )}
          {hasPermission("network:subnets:delete") && (
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("common.delete")}
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-4">
        {/* IP Usage */}
        <Card>
          <CardHeader>
            <CardTitle>{t("subnet.ipUsage")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="h-3 rounded-full bg-muted">
                <div
                  className={`h-3 rounded-full transition-all ${usagePercent > 90 ? "bg-primary" : usagePercent > 60 ? "bg-primary/50" : "bg-primary/20"}`}
                  style={{ width: `${usagePercent}%` }}
                />
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">
                  {usedIPs} / {totalIPs} {t("subnet.usedIPs")} ({freeIPs} {t("subnet.freeIPs")})
                </span>
                <span className="font-medium">{usagePercent}%</span>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Basic info */}
        <Card>
          <CardHeader>
            <CardTitle>{t("subnet.basicInfo")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{subnet.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.displayName")}</span>
                <p className="font-medium">{subnet.spec.displayName || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("subnet.cidr")}</span>
                <p className="font-medium font-mono">
                  {subnet.spec.cidr}
                  {cidrUsableRange(subnet.spec.cidr) && (
                    <span className="ml-2 text-xs font-normal text-muted-foreground">{cidrUsableRange(subnet.spec.cidr)}</span>
                  )}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("subnet.gateway")}</span>
                <p className="font-medium font-mono">{subnet.spec.gateway || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("subnet.broadcast")}</span>
                <p className="font-medium font-mono">{computeBroadcastIP(subnet.spec.cidr) || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("subnet.freeIPs")}</span>
                <p className="font-medium">{freeIPs}</p>
              </div>
              <div className="col-span-2">
                <span className="text-muted-foreground">{t("common.description")}</span>
                <p className="font-medium">{subnet.spec.description || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(subnet.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(subnet.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Allocations */}
        {networkId && subnetId && (
          <AllocationsSection networkId={networkId} subnetId={subnetId} cidr={subnet.spec.cidr} gateway={subnet.spec.gateway} nextFreeIP={subnet.spec.nextFreeIP} onSubnetChange={fetchSubnet} />
        )}
      </div>

      {/* Edit dialog */}
      {networkId && (
        <EditSubnetDialog
          open={editOpen}
          onOpenChange={setEditOpen}
          networkId={networkId}
          subnet={subnet}
          onSuccess={fetchSubnet}
        />
      )}

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("subnet.deleteConfirm", { name: subnet.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Allocations Section =====

function AllocationsSection({
  networkId, subnetId, cidr, gateway, nextFreeIP, onSubnetChange,
}: {
  networkId: string
  subnetId: string
  cidr: string
  gateway?: string
  nextFreeIP?: string
  onSubnetChange: () => void
}) {
  const { t } = useTranslation()
  const { hasPermission } = usePermission()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
  } = useListState()

  const [allocations, setAllocations] = useState<IPAllocation[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)

  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<IPAllocation | null>(null)
  const [bindFilter, setBindFilter] = useState("all")

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams & Record<string, string> = { page, pageSize, sortBy, sortOrder }
      if (bindFilter === "bound") params.hostBound = "true"
      else if (bindFilter === "unbound") params.hostBound = "false"
      const data = await listAllocations(networkId, subnetId, params)
      setAllocations(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [networkId, subnetId, page, pageSize, sortBy, sortOrder, bindFilter])

  useEffect(() => { fetchData() }, [fetchData])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteAllocation(networkId, subnetId, deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
      onSubnetChange()
    } catch (err) {
      showApiError(err, t, "allocation.title")
    }
  }

  const handleCreateSuccess = () => {
    fetchData()
    onSubnetChange()
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>{t("allocation.title")}</CardTitle>
          {hasPermission("network:allocations:create") && (
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              {t("allocation.create")}
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        <div className="border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("ip")}>
                  {t("allocation.ip")}<SortIcon field="ip" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead>{t("allocation.host")}</TableHead>
                <TableHead>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button className="inline-flex items-center gap-1 select-none">
                        {t("allocation.status")}<Filter className="h-3 w-3" />
                      </button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="start">
                      <DropdownMenuItem onClick={() => setBindFilter("all")}>{t("common.all")}</DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setBindFilter("bound")}>{t("allocation.bound")}</DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setBindFilter("unbound")}>{t("allocation.unbound")}</DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableHead>
                <TableHead>{t("allocation.description")}</TableHead>
                <TableHead>{t("allocation.isGateway")}</TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                  {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
                </TableHead>
                <TableHead className="w-20">{t("common.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 3 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 7 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : allocations.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-muted-foreground py-8 text-center">
                    {t("allocation.noData")}
                  </TableCell>
                </TableRow>
              ) : (
                allocations.map((alloc) => (
                  <TableRow key={alloc.metadata.id}>
                    <TableCell className="font-mono text-sm">{alloc.spec.ip}</TableCell>
                    <TableCell className="text-sm">{alloc.spec.hostName || "-"}</TableCell>
                    <TableCell>
                      {alloc.spec.hostId
                        ? <Badge variant="default">{t("allocation.bound")}</Badge>
                        : <Badge variant="secondary">{t("allocation.unbound")}</Badge>}
                    </TableCell>
                    <TableCell className="text-sm">{alloc.spec.description || "-"}</TableCell>
                    <TableCell>
                      {alloc.spec.isGateway
                        ? <Badge variant="outline">{t("allocation.isGateway")}</Badge>
                        : <span className="text-muted-foreground">-</span>}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {new Date(alloc.metadata.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      {hasPermission("network:allocations:delete") && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2 text-destructive hover:text-destructive"
                          onClick={() => setDeleteTarget(alloc)}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

        {/* Create allocation dialog */}
        <AllocationFormDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          networkId={networkId}
          subnetId={subnetId}
          cidr={cidr}
          gateway={gateway}
          nextFreeIP={nextFreeIP}
          onSuccess={handleCreateSuccess}
        />

        <ConfirmDialog
          open={!!deleteTarget}
          onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
          title={t("common.delete")}
          description={t("allocation.deleteConfirm", { ip: deleteTarget?.spec.ip ?? "" })}
          onConfirm={handleDelete}
          confirmText={t("common.delete")}
        />
      </CardContent>
    </Card>
  )
}

// ===== Allocation Form Dialog =====

/** Compute broadcast IP from CIDR string. */
function computeBroadcastIP(cidr: string): string | null {
  const { networkOctets, prefix } = parseCIDR(cidr)
  if (prefix === 0) return null
  const ranges = computeMaskRanges(networkOctets, prefix)
  return ranges.map((r) => r.max).join(".")
}

/** Parse CIDR into network octets and the count of fully-fixed octets. */
function parseCIDR(cidr: string) {
  const m = cidr.match(/^(\d+)\.(\d+)\.(\d+)\.(\d+)\/(\d+)$/)
  if (!m) return { networkOctets: [0, 0, 0, 0], fixedCount: 0, prefix: 0 }
  return {
    networkOctets: [+m[1], +m[2], +m[3], +m[4]],
    fixedCount: Math.floor(+m[5] / 8),
    prefix: +m[5],
  }
}

/** Compute mask-based range [min, max] for each octet. */
function computeMaskRanges(networkOctets: number[], prefix: number): { min: number; max: number }[] {
  return networkOctets.map((octet, i) => {
    const bitStart = i * 8
    const bitEnd = bitStart + 8
    if (prefix >= bitEnd) return { min: octet, max: octet }
    if (prefix <= bitStart) return { min: 0, max: 255 }
    const hostBits = bitEnd - prefix
    const min = octet & (256 - (1 << hostBits))
    return { min, max: min + ((1 << hostBits) - 1) }
  })
}

/** Compute the usable range [min, max] for each octet, excluding network/broadcast.
 *  For prefix >= 24 && <= 30 (single variable octet), tightens last octet by ±1.
 *  For wider prefixes, per-octet hints can't express the cross-octet constraint. */
function computeOctetRanges(networkOctets: number[], prefix: number): { min: number; max: number }[] {
  const ranges = computeMaskRanges(networkOctets, prefix)
  if (prefix >= 24 && prefix <= 30) {
    ranges[3] = { min: ranges[3].min + 1, max: ranges[3].max - 1 }
  }
  return ranges
}

/** Compute the full usable IP range for a CIDR (excludes network and broadcast for prefix <= 30). */
function computeUsableRange(networkOctets: number[], prefix: number): { first: string; last: string } | null {
  if (prefix > 30) return null
  const ranges = computeMaskRanges(networkOctets, prefix)
  const firstIP = ranges.map((r) => r.min)
  const lastIP = ranges.map((r) => r.max)
  firstIP[3] += 1
  lastIP[3] -= 1
  return { first: firstIP.join("."), last: lastIP.join(".") }
}

function AllocationFormDialog({
  open, onOpenChange, networkId, subnetId, cidr, gateway, nextFreeIP, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  networkId: string
  subnetId: string
  cidr: string
  gateway?: string
  nextFreeIP?: string
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [ipError, setIpError] = useState("")
  const [description, setDescription] = useState("")
  const [isGateway, setIsGateway] = useState(false)
  const [formError, setFormError] = useState("")
  const inputRefs = useRef<(HTMLInputElement | null)[]>([null, null, null, null])

  const { networkOctets, fixedCount, prefix } = useMemo(() => parseCIDR(cidr), [cidr])
  const octetRanges = useMemo(() => computeOctetRanges(networkOctets, prefix), [networkOctets, prefix])
  const usableRange = useMemo(() => computeUsableRange(networkOctets, prefix), [networkOctets, prefix])
  const reservedIPs = useMemo(() => {
    if (prefix > 30) return new Set<string>()
    const ranges = computeMaskRanges(networkOctets, prefix)
    return new Set([
      ranges.map((r) => r.min).join("."), // network address
      ranges.map((r) => r.max).join("."), // broadcast address
    ])
  }, [networkOctets, prefix])

  const defaultOctets = useMemo(() => {
    // Use backend-provided nextFreeIP if available
    if (nextFreeIP) {
      const parts = nextFreeIP.split(".")
      if (parts.length === 4) return parts
    }
    // Fallback: network address +1
    const d = networkOctets.map(String)
    d[3] = String(networkOctets[3] + 1)
    return d
  }, [networkOctets, nextFreeIP])

  const [octets, setOctets] = useState<string[]>(defaultOctets)

  useEffect(() => {
    if (open) {
      setOctets([...defaultOctets])
      setIpError("")
      setDescription("")
      setIsGateway(false)
      setFormError("")
      setLoading(false)
    }
  }, [open, defaultOctets])

  const updateOctet = (index: number, value: string) => {
    if (!/^\d{0,3}$/.test(value)) return
    if (value !== "" && +value > 255) return
    const next = [...octets]
    next[index] = value
    setOctets(next)
    setIpError("")
    // Auto-advance to next input when octet is complete
    if (value.length === 3 || (value.length === 2 && +value > 25)) {
      const nextEditable = [index + 1, index + 2, index + 3].find((i) => i < 4 && i >= fixedCount)
      if (nextEditable !== undefined) inputRefs.current[nextEditable]?.focus()
    }
  }

  const handleOctetKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "." || e.key === "Tab") {
      if (e.key === ".") e.preventDefault()
      const nextEditable = [index + 1, index + 2, index + 3].find((i) => i < 4 && i >= fixedCount)
      if (nextEditable !== undefined) inputRefs.current[nextEditable]?.focus()
    }
    // Backspace on empty → go to previous editable
    if (e.key === "Backspace" && octets[index] === "") {
      const prevEditable = [index - 1, index - 2, index - 3].find((i) => i >= 0 && i >= fixedCount)
      if (prevEditable !== undefined) inputRefs.current[prevEditable]?.focus()
    }
  }

  const handleOctetBlur = async (index: number) => {
    // Reset empty editable octet to default
    if (index >= fixedCount && octets[index] === "") {
      const next = [...octets]
      next[index] = defaultOctets[index]
      setOctets(next)
      setIpError("")
      return
    }
    // Check when all octets are filled
    if (octets.some((o) => o === "")) return
    const ip = octets.join(".")
    // Reject network/broadcast addresses
    if (reservedIPs.has(ip)) {
      setIpError(t("allocation.reservedIP"))
      return
    }
    // Check occupied
    try {
      const data = await listAllocations(networkId, subnetId, { page: 1, pageSize: 1, search: ip })
      if (data.items?.some((a) => a.spec.ip === ip)) {
        setIpError(t("api.error.ipAlreadyAllocated"))
      }
    } catch { /* backend will validate on submit */ }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const emptyEditable = octets.some((o, i) => i >= fixedCount && o === "")
    if (emptyEditable) {
      setIpError(t("api.validation.required", { field: t("allocation.ip") }))
      return
    }
    if (ipError) return

    const ip = octets.join(".")
    if (reservedIPs.has(ip)) {
      setIpError(t("allocation.reservedIP"))
      return
    }
    setLoading(true)
    try {
      await createAllocation(networkId, subnetId, {
        spec: { ip, description: description || undefined, isGateway: isGateway || undefined },
      })
      toast.success(t("action.createSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      if (err instanceof ApiError && err.details?.length) {
        const ipDetail = err.details.find((d) => d.field === "spec.ip")
        if (ipDetail) {
          const i18nKey = translateDetailMessage(ipDetail.message)
          setIpError(i18nKey !== ipDetail.message ? t(i18nKey, { field: t("allocation.ip") }) : ipDetail.message)
        }
        const descDetail = err.details.find((d) => d.field === "spec.description")
        if (descDetail) {
          const i18nKey = translateDetailMessage(descDetail.message)
          setFormError(i18nKey !== descDetail.message ? t(i18nKey, { field: t("allocation.description") }) : descDetail.message)
        }
      } else if (err instanceof ApiError) {
        const i18nKey = translateApiError(err)
        const msg = i18nKey !== err.message ? t(i18nKey, { resource: t("allocation.title") }) : err.message
        if (err.status === 409 || err.message.startsWith("IP ")) {
          setIpError(msg)
        } else {
          setFormError(msg)
        }
      } else {
        setFormError(t("api.error.internalError"))
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("allocation.create")}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          {formError && (
            <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {formError}
            </div>
          )}
          <div className="space-y-2">
            <label className="text-sm font-medium">{t("allocation.ip")}</label>
            <div className="flex items-center gap-1">
              {octets.map((octet, i) => (
                <React.Fragment key={i}>
                  {i > 0 && <span className="text-muted-foreground font-mono text-lg select-none pb-5">.</span>}
                  <div className="flex flex-col items-center">
                    <Input
                      ref={(el) => { inputRefs.current[i] = el }}
                      value={octet}
                      onChange={(e) => updateOctet(i, e.target.value)}
                      onKeyDown={(e) => handleOctetKeyDown(i, e)}
                      onBlur={() => handleOctetBlur(i)}
                      disabled={i < fixedCount}
                      className="w-16 text-center font-mono tabular-nums"
                      maxLength={3}
                      inputMode="numeric"
                    />
                    <span className="text-muted-foreground text-[11px] mt-1 font-mono">
                      {octetRanges[i].min === octetRanges[i].max
                        ? String(octetRanges[i].min)
                        : `${octetRanges[i].min}-${octetRanges[i].max}`}
                    </span>
                  </div>
                </React.Fragment>
              ))}
            </div>
            {usableRange && (
              <p className="text-muted-foreground text-xs font-mono">
                {t("allocation.usableRange", { first: usableRange.first, last: usableRange.last })}
              </p>
            )}
            {ipError && <p className="text-sm text-destructive">{ipError}</p>}
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">{t("allocation.description")}</label>
            <Textarea rows={2} value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>
          {!gateway && (
            <div className="flex items-center gap-2">
              <Checkbox id="isGateway" checked={isGateway} onCheckedChange={(v) => setIsGateway(v === true)} />
              <label htmlFor="isGateway" className="text-sm cursor-pointer select-none">{t("allocation.setAsGateway")}</label>
            </div>
          )}
          <DialogFooter className="mt-6 pt-4 border-t">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
            <Button type="submit" disabled={loading}>{loading ? "..." : t("common.save")}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// ===== Edit Subnet Dialog =====

function EditSubnetDialog({
  open, onOpenChange, networkId, subnet, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  networkId: string
  subnet: Subnet
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
      displayName: subnet.spec.displayName ?? "",
      description: subnet.spec.description ?? "",
    },
  })

  useEffect(() => {
    if (open) {
      form.reset({
        displayName: subnet.spec.displayName ?? "",
        description: subnet.spec.description ?? "",
      })
    }
  }, [open, subnet, form])

  const onSubmit = async (values: FormValues) => {
    setLoading(true)
    try {
      const spec: Subnet["spec"] = {
        ...subnet.spec,
        displayName: values.displayName,
        description: values.description,
      }

      await updateSubnet(networkId, subnet.metadata.id, { metadata: subnet.metadata, spec })
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
          <DialogTitle>{t("subnet.edit")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            {form.formState.errors.root && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div>
              <label className="text-sm font-medium">{t("subnet.name")}</label>
              <Input value={subnet.metadata.name} disabled className="mt-1" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-sm font-medium">{t("subnet.cidr")}</label>
                <Input value={subnet.spec.cidr} disabled className="mt-1 font-mono" />
              </div>
              <div>
                <label className="text-sm font-medium">{t("subnet.gateway")}</label>
                <Input value={subnet.spec.gateway || "-"} disabled className="mt-1 font-mono" />
              </div>
            </div>
            <FormField control={form.control} name="displayName" render={({ field }) => (
              <FormItem><FormLabel>{t("subnet.displayName")}</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="description" render={({ field }) => (
              <FormItem><FormLabel>{t("subnet.description")}</FormLabel><FormControl><Textarea rows={2} {...field} /></FormControl><FormMessage /></FormItem>
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
