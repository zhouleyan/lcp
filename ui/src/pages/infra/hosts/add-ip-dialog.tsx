import { useEffect, useState } from "react"
import { toast } from "sonner"
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import {
  Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import {
  addHostIP, addWorkspaceHostIP, addNamespaceHostIP,
} from "@/api/infra/hosts"
import {
  listInfraNetworks, listWorkspaceInfraNetworks, listNamespaceInfraNetworks,
} from "@/api/infra/networks"
import { showApiError } from "@/api/client"
import type { AvailableNetwork } from "@/api/types"
import { useTranslation } from "@/i18n"
import { scopedApiCall } from "@/lib/nav-config"
import { isIPInCIDR } from "@/lib/ip-utils"

export function AddIPDialog({
  open, onOpenChange, hostId, onSuccess, scopeWorkspaceId, scopeNamespaceId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  hostId: string
  onSuccess: () => void
  scopeWorkspaceId: string | undefined
  scopeNamespaceId: string | undefined
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [networks, setNetworks] = useState<AvailableNetwork[]>([])
  const [subnetId, setSubnetId] = useState("")
  const [ip, setIp] = useState("")
  const [ipError, setIpError] = useState("")

  useEffect(() => {
    if (!open) return
    setSubnetId("")
    setIp("")
    setIpError("")
    const fetchNetworks = async () => {
      try {
        const data = await scopedApiCall(
          scopeWorkspaceId, scopeNamespaceId,
          () => listInfraNetworks(),
          (wsId) => listWorkspaceInfraNetworks(wsId),
          (wsId, nsId) => listNamespaceInfraNetworks(wsId, nsId),
        )
        const nets = data.items ?? []
        setNetworks(nets)
        const first = nets.flatMap((n) => n.spec.subnets).find((s) => s.freeIPs > 0)
        if (first) setSubnetId(first.id)
      } catch {
        setNetworks([])
      }
    }
    fetchNetworks()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const handleSubmit = async () => {
    if (!subnetId) return
    if (ip) {
      if (!/^(\d{1,3}\.){3}\d{1,3}$/.test(ip)) {
        setIpError(t("api.validation.ip.format"))
        return
      }
      const subnet = networks.flatMap((n) => n.spec.subnets).find((s) => s.id === subnetId)
      if (subnet && !isIPInCIDR(ip, subnet.cidr)) {
        setIpError(t("host.ips.ip.outOfRange"))
        return
      }
    }
    setIpError("")
    setLoading(true)
    try {
      const data = { subnetId, ...(ip ? { ip } : {}) }
      await scopedApiCall(
        scopeWorkspaceId, scopeNamespaceId,
        () => addHostIP(hostId, data),
        (wsId) => addWorkspaceHostIP(wsId, hostId, data),
        (wsId, nsId) => addNamespaceHostIP(wsId, nsId, hostId, data),
      )
      toast.success(t("action.createSuccess"))
      onOpenChange(false)
      onSuccess()
    } catch (err) {
      const apiErr = err as { reason?: string }
      if (apiErr.reason === "Conflict") {
        setIpError(t("host.ips.ip.conflict"))
      } else {
        showApiError(err, t, "host.ips.ip")
      }
    } finally {
      setLoading(false)
    }
  }

  const selectedSubnet = subnetId
    ? networks.flatMap((n) => n.spec.subnets).find((s) => s.id === subnetId)
    : null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md" onOpenAutoFocus={(e) => e.preventDefault()} aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("host.ips.add")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          {networks.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("host.ips.noNetworks")}</p>
          ) : (
            <>
              <div>
                <label className="text-sm font-medium">{t("host.ips.subnetId")}</label>
                <Select value={subnetId} onValueChange={setSubnetId}>
                  <SelectTrigger className="mt-1 w-full">
                    <SelectValue placeholder={t("host.ips.subnet.select")} />
                  </SelectTrigger>
                  <SelectContent>
                    {networks.map((net) => (
                      <SelectGroup key={net.metadata.id}>
                        <SelectLabel className="text-sm text-muted-foreground">
                          {net.spec.displayName || net.metadata.name}{net.spec.cidr ? ` (${net.spec.cidr})` : ""}
                        </SelectLabel>
                        {net.spec.subnets.map((sub) => (
                          <SelectItem key={sub.id} value={sub.id} disabled={sub.freeIPs === 0}>
                            {sub.displayName || sub.name} {sub.cidr}
                            <span className="text-muted-foreground ml-2 text-xs">
                              ({t("host.ips.subnet.free", { free: sub.freeIPs, total: sub.totalIPs })})
                            </span>
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    ))}
                  </SelectContent>
                </Select>
                {selectedSubnet && (
                  <p className="text-muted-foreground mt-1 text-xs">
                    {selectedSubnet.cidr} · {t("host.ips.subnet.free", { free: selectedSubnet.freeIPs, total: selectedSubnet.totalIPs })}
                  </p>
                )}
              </div>
              <div>
                <label className="text-sm font-medium">{t("host.ips.ip")}</label>
                <Input className="mt-1" value={ip} onChange={(e) => { setIp(e.target.value); setIpError("") }} placeholder={t("host.ips.ip.auto")} />
                <p className="text-destructive mt-1 min-h-[20px] text-sm">{ipError || "\u00A0"}</p>
              </div>
            </>
          )}
        </div>
        <DialogFooter className="mt-4">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
          <Button onClick={handleSubmit} disabled={loading || !subnetId}>{loading ? "..." : t("common.save")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
