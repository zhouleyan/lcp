import { useCallback, useEffect, useRef, useState } from "react"
import { Plus, Trash2, Search, Download, ChevronDown } from "lucide-react"
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
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage,
} from "@/components/ui/form"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  listCertificates, createCertificate,
  deleteCertificate, deleteCertificates, exportCertificateFile,
} from "@/api/pki/certificates"
import { handleFormApiError, showApiError, SELECT_PAGE_SIZE } from "@/api/client"
import type { Certificate, ListParams } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { useListState } from "@/hooks/use-list-state"
import { SortIcon } from "@/components/sort-icon"
import { Pagination } from "@/components/pagination"
import { ConfirmDialog } from "@/components/confirm-dialog"

// Certificate type badge colors
const certTypeBadgeVariant: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  ca: "default",
  server: "secondary",
  client: "outline",
  both: "secondary",
}

// Certificate validity status helper
function getCertStatus(notAfter?: string): "valid" | "expiringSoon" | "expired" {
  if (!notAfter) return "valid"
  const expiry = new Date(notAfter)
  const now = new Date()
  if (expiry < now) return "expired"
  const thirtyDays = 30 * 24 * 60 * 60 * 1000
  if (expiry.getTime() - now.getTime() < thirtyDays) return "expiringSoon"
  return "valid"
}

export default function CertificateListPage() {
  const { t } = useTranslation()
  const {
    page, setPage, pageSize, setPageSize, sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  } = useListState()
  const { hasPermission } = usePermission()

  const [certificates, setCertificates] = useState<Certificate[]>([])
  const [loading, setLoading] = useState(true)
  const [totalCount, setTotalCount] = useState(0)

  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Certificate | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)

  const permPrefix = "pki:certificates"

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListParams = { page, pageSize, sortBy, sortOrder }
      if (search) params.search = search

      const data = await listCertificates(params)
      setCertificates(data.items ?? [])
      setTotalCount(data.totalCount)
    } catch (err) {
      showApiError(err, t)
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sortBy, sortOrder, search])

  useEffect(() => { fetchData() }, [fetchData])
  useEffect(() => { setPage(1) }, [search, pageSize])
  useEffect(() => { clearSelection() }, [certificates])

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteCertificate(deleteTarget.metadata.id)
      toast.success(t("action.deleteSuccess"))
      setDeleteTarget(null)
      fetchData()
    } catch (err) {
      showApiError(err, t, "certificate.title")
    }
  }

  const handleBatchDelete = async () => {
    try {
      const ids = Array.from(selected)
      await deleteCertificates(ids)
      toast.success(t("action.deleteSuccess"))
      setBatchDeleteOpen(false)
      clearSelection()
      fetchData()
    } catch (err) {
      showApiError(err, t, "certificate.title")
    }
  }

  const handleExport = async (id: string, file: string) => {
    try {
      await exportCertificateFile(id, file)
    } catch (err) {
      showApiError(err, t, "certificate.title")
    }
  }

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("certificate.title")}</h1>
          <p className="text-muted-foreground text-sm">
            {t("common.total", { count: totalCount })}
          </p>
        </div>
        {hasPermission(`${permPrefix}:create`) && (
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t("certificate.create")}
          </Button>
        )}
      </div>

      {/* filters */}
      <div className="mb-4 flex items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
          <Input
            placeholder={t("certificate.searchPlaceholder")}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {selected.size > 0 && hasPermission(`${permPrefix}:deleteCollection`) && (
          <Button variant="destructive" size="sm" onClick={() => setBatchDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            {t("certificate.deleteSelected")} ({selected.size})
          </Button>
        )}
      </div>

      {/* table */}
      <div className="border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-10">
                {hasPermission(`${permPrefix}:deleteCollection`) && (
                  <Checkbox
                    checked={certificates.length > 0 && selected.size === certificates.length}
                    onCheckedChange={() => toggleAll(certificates.map((c) => c.metadata.id))}
                  />
                )}
              </TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("name")}>
                {t("common.name")}<SortIcon field="name" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead>{t("certificate.certType")}</TableHead>
              <TableHead>{t("certificate.commonName")}</TableHead>
              <TableHead>{t("certificate.caName")}</TableHead>
              <TableHead>{t("certificate.notAfter")}</TableHead>
              <TableHead className="cursor-pointer select-none" onClick={() => handleSort("created_at")}>
                {t("common.created")}<SortIcon field="created_at" sortBy={sortBy} sortOrder={sortOrder} />
              </TableHead>
              <TableHead className="w-28">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 8 }).map((_, j) => (
                    <TableCell key={j}><Skeleton className="h-4 w-16" /></TableCell>
                  ))}
                </TableRow>
              ))
            ) : certificates.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="text-muted-foreground py-8 text-center">
                  {t("certificate.noData")}
                </TableCell>
              </TableRow>
            ) : (
              certificates.map((cert) => {
                const status = getCertStatus(cert.status?.notAfter)
                return (
                  <TableRow key={cert.metadata.id}>
                    <TableCell>
                      {hasPermission(`${permPrefix}:deleteCollection`) && (
                        <Checkbox
                          checked={selected.has(cert.metadata.id)}
                          onCheckedChange={() => toggleOne(cert.metadata.id)}
                        />
                      )}
                    </TableCell>
                    <TableCell className="font-medium">{cert.metadata.name}</TableCell>
                    <TableCell>
                      <Badge variant={certTypeBadgeVariant[cert.spec.certType] ?? "secondary"}>
                        {t(`certificate.certType.${cert.spec.certType}`)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {cert.spec.commonName || "-"}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {cert.spec.caName || "-"}
                    </TableCell>
                    <TableCell className="text-sm whitespace-nowrap">
                      {cert.status?.notAfter ? (
                        <span className={
                          status === "expired" ? "text-destructive" :
                          status === "expiringSoon" ? "text-yellow-600" :
                          "text-muted-foreground"
                        }>
                          {new Date(cert.status.notAfter).toLocaleDateString()}
                          {status !== "valid" && (
                            <Badge variant={status === "expired" ? "destructive" : "outline"} className="ml-2 text-xs">
                              {t(`certificate.${status}`)}
                            </Badge>
                          )}
                        </span>
                      ) : "-"}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {new Date(cert.metadata.createdAt).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        {hasPermission(`${permPrefix}:export`) && (
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="sm" className="h-8 px-2">
                                <Download className="mr-1 h-3.5 w-3.5" />
                                <ChevronDown className="h-3 w-3" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem onClick={() => handleExport(cert.metadata.id, "cert.pem")}>
                                {t("certificate.exportCert")}
                              </DropdownMenuItem>
                              <DropdownMenuItem onClick={() => handleExport(cert.metadata.id, "key.pem")}>
                                {t("certificate.exportKey")}
                              </DropdownMenuItem>
                              {cert.spec.certType !== "ca" && (
                                <DropdownMenuItem onClick={() => handleExport(cert.metadata.id, "ca.pem")}>
                                  {t("certificate.exportCA")}
                                </DropdownMenuItem>
                              )}
                            </DropdownMenuContent>
                          </DropdownMenu>
                        )}
                        {hasPermission(`${permPrefix}:delete`) && (
                          <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive hover:text-destructive"
                            onClick={() => setDeleteTarget(cert)} title={t("common.delete")}>
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </div>

      <Pagination totalCount={totalCount} page={page} pageSize={pageSize} onPageChange={setPage} onPageSizeChange={setPageSize} />

      <CertificateCreateDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={fetchData}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(v) => { if (!v) setDeleteTarget(null) }}
        title={t("common.delete")}
        description={t("certificate.deleteConfirm", { name: deleteTarget?.metadata.name ?? "" })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t("certificate.deleteSelected")}
        description={t("certificate.batchDeleteConfirm", { count: selected.size })}
        onConfirm={handleBatchDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}

// ===== Certificate Create Dialog =====

interface CertificateFormValues {
  name: string
  certType: "ca" | "server" | "client" | "both"
  commonName: string
  dnsNames: string
  ipAddresses: string
  caName: string
  validityDays: number
}

function CertificateCreateDialog({
  open, onOpenChange, onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [caList, setCaList] = useState<Certificate[]>([])

  const schema = z.object({
    name: z.string()
      .min(3, t("api.validation.name.format"))
      .max(50, t("api.validation.name.format"))
      .regex(/^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$/, t("api.validation.name.format")),
    certType: z.enum(["ca", "server", "client", "both"]),
    commonName: z.string().optional(),
    dnsNames: z.string().optional(),
    ipAddresses: z.string().optional(),
    caName: z.string().optional(),
    validityDays: z.coerce.number().int().min(1).max(36500),
  }).refine((data) => {
    if (data.certType === "ca" && !data.commonName) return false
    return true
  }, { message: t("api.validation.required", { field: t("certificate.commonName") }), path: ["commonName"] })
  .refine((data) => {
    if ((data.certType === "server" || data.certType === "both") && !data.dnsNames?.trim() && !data.ipAddresses?.trim()) return false
    return true
  }, { message: t("certificate.dnsOrIpRequired"), path: ["dnsNames"] })
  .refine((data) => {
    if (data.certType !== "ca" && !data.caName) return false
    return true
  }, { message: t("api.validation.required", { field: t("certificate.caName") }), path: ["caName"] })

  const form = useForm<CertificateFormValues>({
    resolver: zodResolver(schema) as never,
    mode: "onBlur",
    defaultValues: { name: "", certType: "ca", commonName: "", dnsNames: "", ipAddresses: "", caName: "", validityDays: 3650 },
  })

  const certType = form.watch("certType")
  const caName = form.watch("caName")
  const autoNameRef = useRef("")

  // Auto-generate name from CA name + cert type
  // e.g. CA "etcd-ca" + type "server" → "etcd-server"
  useEffect(() => {
    const currentName = form.getValues("name")
    if (certType === "ca") {
      // CA type: clear auto-name, let user fill manually
      if (currentName === autoNameRef.current) {
        form.setValue("name", "")
        autoNameRef.current = ""
      }
      return
    }
    if (!caName) return
    // Only auto-fill if name is empty or was previously auto-generated
    if (currentName && currentName !== autoNameRef.current) return

    const prefix = caName.replace(/-ca$/, "")
    const suffixMap: Record<string, string> = { server: "server", client: "client", both: "peer" }
    const suffix = suffixMap[certType] ?? certType
    const generated = `${prefix}-${suffix}`
    form.setValue("name", generated)
    autoNameRef.current = generated
  }, [certType, caName, form])

  // Update default validity when certType changes
  useEffect(() => {
    const current = form.getValues("validityDays")
    if (certType === "ca" && current === 365) {
      form.setValue("validityDays", 3650)
    } else if (certType !== "ca" && current === 3650) {
      form.setValue("validityDays", 365)
    }
  }, [certType, form])

  // Fetch CA list when dialog opens
  useEffect(() => {
    if (open) {
      listCertificates({ page: 1, pageSize: SELECT_PAGE_SIZE, certType: "ca" })
        .then((data) => setCaList(data.items ?? []))
        .catch(() => setCaList([]))
    }
  }, [open])

  useEffect(() => {
    if (open) {
      form.reset({ name: "", certType: "ca", commonName: "", dnsNames: "", ipAddresses: "", caName: "", validityDays: 3650 })
      autoNameRef.current = ""
    }
  }, [open, form])

  const onSubmit = async (values: CertificateFormValues) => {
    setLoading(true)
    try {
      const spec: Certificate["spec"] = {
        certType: values.certType,
        validityDays: values.validityDays,
      }
      if (values.certType === "ca") {
        spec.commonName = values.commonName
      } else {
        spec.caName = values.caName
        const dns = values.dnsNames.split(",").map((s) => s.trim()).filter(Boolean)
        if (dns.length > 0) spec.dnsNames = dns
        const ips = values.ipAddresses.split(",").map((s) => s.trim()).filter(Boolean)
        if (ips.length > 0) spec.ipAddresses = ips
        if (values.commonName) spec.commonName = values.commonName
      }

      await createCertificate({
        metadata: { name: values.name } as Certificate["metadata"],
        spec,
      })
      toast.success(t("action.createSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      handleFormApiError(err, form, t, "certificate", "certificate.title")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] flex flex-col overflow-hidden" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("certificate.create")}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex min-h-0 flex-col flex-1 overflow-hidden">
            {form.formState.errors.root && (
              <div className="shrink-0 rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {form.formState.errors.root.message}
              </div>
            )}
            <div className="space-y-4 overflow-y-auto flex-1 min-h-0">
              {/* Name */}
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel required>{t("common.name")}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={certType === "ca" ? "etcd-ca" : "etcd-server"} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Certificate Type */}
              <FormField
                control={form.control}
                name="certType"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel required>{t("certificate.certType")}</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="ca">{t("certificate.certType.ca")}</SelectItem>
                        <SelectItem value="server">{t("certificate.certType.server")}</SelectItem>
                        <SelectItem value="client">{t("certificate.certType.client")}</SelectItem>
                        <SelectItem value="both">{t("certificate.certType.both")}</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Common Name (required for CA, optional for others) */}
              <FormField
                control={form.control}
                name="commonName"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel required={certType === "ca"}>{t("certificate.commonName")}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder={t("certificate.commonNamePlaceholder")} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Signing CA (non-CA types only) */}
              {certType !== "ca" && (
                <FormField
                  control={form.control}
                  name="caName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel required>{t("certificate.caName")}</FormLabel>
                      <Select value={field.value} onValueChange={field.onChange}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder={t("certificate.caNamePlaceholder")} />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {caList.length === 0 ? (
                            <div className="text-muted-foreground px-2 py-4 text-center text-sm">
                              {t("certificate.noCaAvailable")}
                            </div>
                          ) : (
                            caList.map((ca) => (
                              <SelectItem key={ca.metadata.name} value={ca.metadata.name}>
                                {ca.metadata.name} ({ca.spec.commonName})
                              </SelectItem>
                            ))
                          )}
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {/* DNS Names (non-CA types) */}
              {certType !== "ca" && (
                <FormField
                  control={form.control}
                  name="dnsNames"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("certificate.dnsNames")}</FormLabel>
                      <FormControl>
                        <Input {...field} placeholder={t("certificate.dnsNamesPlaceholder")} />
                      </FormControl>
                      <FormDescription>{t("certificate.dnsNamesHint")}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {/* IP Addresses (non-CA types) */}
              {certType !== "ca" && (
                <FormField
                  control={form.control}
                  name="ipAddresses"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("certificate.ipAddresses")}</FormLabel>
                      <FormControl>
                        <Input {...field} placeholder={t("certificate.ipAddressesPlaceholder")} />
                      </FormControl>
                      <FormDescription>{t("certificate.ipAddressesHint")}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {/* Validity Days */}
              <FormField
                control={form.control}
                name="validityDays"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("certificate.validityDays")}</FormLabel>
                    <FormControl>
                      <Input type="number" {...field} min={1} max={36500} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
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
