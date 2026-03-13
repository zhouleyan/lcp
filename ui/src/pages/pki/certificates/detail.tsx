import { useCallback, useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router"
import { Trash2, Download, ChevronDown, Copy, Check } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { getCertificate, deleteCertificate, exportCertificateFile } from "@/api/pki/certificates"
import { showApiError } from "@/api/client"
import type { Certificate } from "@/api/types"
import { useTranslation } from "@/i18n"
import { usePermission } from "@/hooks/use-permission"
import { ConfirmDialog } from "@/components/confirm-dialog"

// Certificate type badge colors
const certTypeBadgeVariant: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  ca: "default",
  server: "secondary",
  client: "outline",
  both: "secondary",
}

function getCertStatus(notAfter?: string): "valid" | "expiringSoon" | "expired" {
  if (!notAfter) return "valid"
  const expiry = new Date(notAfter)
  const now = new Date()
  if (expiry < now) return "expired"
  const thirtyDays = 30 * 24 * 60 * 60 * 1000
  if (expiry.getTime() - now.getTime() < thirtyDays) return "expiringSoon"
  return "valid"
}

const statusBadgeVariant: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  valid: "default",
  expiringSoon: "outline",
  expired: "destructive",
}

export default function CertificateDetailPage() {
  const { certificateId } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [cert, setCert] = useState<Certificate | null>(null)
  const [loading, setLoading] = useState(true)
  const { hasPermission } = usePermission()
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [copiedCert, setCopiedCert] = useState(false)
  const [copiedKey, setCopiedKey] = useState(false)

  const permPrefix = "pki:certificates"

  const fetchCert = useCallback(async () => {
    if (!certificateId) return
    try {
      const c = await getCertificate(certificateId)
      setCert(c)
    } catch {
      setCert(null)
    } finally {
      setLoading(false)
    }
  }, [certificateId])

  useEffect(() => { fetchCert() }, [fetchCert])

  const handleDelete = async () => {
    if (!cert) return
    try {
      await deleteCertificate(cert.metadata.id)
      toast.success(t("action.deleteSuccess"))
      navigate("/pki/certificates")
    } catch (err) {
      showApiError(err, t, "certificate.title")
    }
  }

  const handleExport = async (file: string) => {
    if (!cert) return
    try {
      await exportCertificateFile(cert.metadata.id, file)
    } catch (err) {
      showApiError(err, t, "certificate.title")
    }
  }

  const handleCopy = async (text: string, type: "cert" | "key") => {
    await navigator.clipboard.writeText(text)
    if (type === "cert") {
      setCopiedCert(true)
      setTimeout(() => setCopiedCert(false), 2000)
    } else {
      setCopiedKey(true)
      setTimeout(() => setCopiedKey(false), 2000)
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

  if (!cert) {
    return (
      <div className="p-6">
        <p className="text-muted-foreground">{t("certificate.notFound")}</p>
      </div>
    )
  }

  const status = getCertStatus(cert.status?.notAfter)
  const isCA = cert.spec.certType === "ca"

  return (
    <div className="p-6">
      {/* header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{cert.metadata.name}</h1>
          <Badge variant={certTypeBadgeVariant[cert.spec.certType] ?? "secondary"}>
            {t(`certificate.certType.${cert.spec.certType}`)}
          </Badge>
          <Badge variant={statusBadgeVariant[status]}>
            {t(`certificate.${status}`)}
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          {hasPermission(`${permPrefix}:export`) && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm">
                  <Download className="mr-2 h-4 w-4" />
                  {t("certificate.export")}
                  <ChevronDown className="ml-1 h-3 w-3" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleExport("cert.pem")}>
                  {t("certificate.exportCert")}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleExport("key.pem")}>
                  {t("certificate.exportKey")}
                </DropdownMenuItem>
                {!isCA && (
                  <DropdownMenuItem onClick={() => handleExport("ca.pem")}>
                    {t("certificate.exportCA")}
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
          {hasPermission(`${permPrefix}:delete`) && (
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="mr-2 h-4 w-4" />
              {t("common.delete")}
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-6">
        {/* info card */}
        <Card>
          <CardHeader>
            <CardTitle>{t("certificate.details")}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm">
              <div>
                <span className="text-muted-foreground">{t("common.name")}</span>
                <p className="font-medium">{cert.metadata.name}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("certificate.certType")}</span>
                <p className="font-medium">{t(`certificate.certType.${cert.spec.certType}`)}</p>
              </div>
              {cert.spec.commonName && (
                <div>
                  <span className="text-muted-foreground">{t("certificate.commonName")}</span>
                  <p className="font-medium">{cert.spec.commonName}</p>
                </div>
              )}
              {cert.spec.caName && (
                <div>
                  <span className="text-muted-foreground">{t("certificate.caName")}</span>
                  <p className="font-medium">{cert.spec.caName}</p>
                </div>
              )}
              {cert.spec.dnsNames && cert.spec.dnsNames.length > 0 && (
                <div className="col-span-2">
                  <span className="text-muted-foreground">{t("certificate.dnsNames")}</span>
                  <div className="mt-1 flex flex-wrap gap-1.5">
                    {cert.spec.dnsNames.map((dns) => (
                      <Badge key={dns} variant="outline">{dns}</Badge>
                    ))}
                  </div>
                </div>
              )}
              {cert.spec.ipAddresses && cert.spec.ipAddresses.length > 0 && (
                <div className="col-span-2">
                  <span className="text-muted-foreground">{t("certificate.ipAddresses")}</span>
                  <div className="mt-1 flex flex-wrap gap-1.5">
                    {cert.spec.ipAddresses.map((ip) => (
                      <Badge key={ip} variant="outline">{ip}</Badge>
                    ))}
                  </div>
                </div>
              )}
              <div>
                <span className="text-muted-foreground">{t("certificate.serialNumber")}</span>
                <p className="font-mono text-xs">{cert.status?.serialNumber || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("certificate.validityDays")}</span>
                <p className="font-medium">{cert.spec.validityDays || "-"}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("certificate.notBefore")}</span>
                <p className="font-medium">
                  {cert.status?.notBefore ? new Date(cert.status.notBefore).toLocaleString() : "-"}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("certificate.notAfter")}</span>
                <p className="font-medium">
                  {cert.status?.notAfter ? new Date(cert.status.notAfter).toLocaleString() : "-"}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.created")}</span>
                <p className="font-medium">{new Date(cert.metadata.createdAt).toLocaleString()}</p>
              </div>
              <div>
                <span className="text-muted-foreground">{t("common.updated")}</span>
                <p className="font-medium">{new Date(cert.metadata.updatedAt).toLocaleString()}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Certificate PEM */}
        {cert.status?.certificate && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>{t("certificate.pemCert")}</CardTitle>
                <Button variant="ghost" size="sm" onClick={() => handleCopy(cert.status!.certificate, "cert")}>
                  {copiedCert ? <Check className="mr-1 h-4 w-4" /> : <Copy className="mr-1 h-4 w-4" />}
                  {copiedCert ? t("common.copied") : t("common.copy")}
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <pre className="bg-muted overflow-x-auto rounded-md p-4 text-xs leading-relaxed font-mono">
                {cert.status.certificate}
              </pre>
            </CardContent>
          </Card>
        )}

        {/* Private Key PEM */}
        {cert.status?.privateKey && (
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>{t("certificate.pemKey")}</CardTitle>
                <Button variant="ghost" size="sm" onClick={() => handleCopy(cert.status!.privateKey!, "key")}>
                  {copiedKey ? <Check className="mr-1 h-4 w-4" /> : <Copy className="mr-1 h-4 w-4" />}
                  {copiedKey ? t("common.copied") : t("common.copy")}
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <pre className="bg-muted overflow-x-auto rounded-md p-4 text-xs leading-relaxed font-mono">
                {cert.status.privateKey}
              </pre>
            </CardContent>
          </Card>
        )}
      </div>

      {/* delete confirm */}
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("common.delete")}
        description={t("certificate.deleteConfirm", { name: cert.metadata.name })}
        onConfirm={handleDelete}
        confirmText={t("common.delete")}
      />
    </div>
  )
}
