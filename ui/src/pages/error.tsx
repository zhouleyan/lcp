import { useSearchParams } from "react-router"
import { useEffect } from "react"
import { Button } from "@/components/ui/button"
import { useTranslation } from "@/i18n"
import { startAuthFlow } from "@/lib/auth"

const statusConfig: Record<string, { icon: string; titleKey: string; descKey: string }> = {
  "400": { icon: "⚠️", titleKey: "error.400.title", descKey: "error.400.desc" },
  "401": { icon: "🔒", titleKey: "error.401.title", descKey: "error.401.desc" },
  "403": { icon: "🚫", titleKey: "error.403.title", descKey: "error.403.desc" },
  "404": { icon: "📄", titleKey: "error.404.title", descKey: "error.404.desc" },
  "500": { icon: "⚙️", titleKey: "error.500.title", descKey: "error.500.desc" },
}

export default function ErrorPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const status = searchParams.get("status") || "404"

  useEffect(() => {
    if (status === "401") {
      startAuthFlow()
    }
  }, [status])

  if (status === "401") {
    return null
  }

  const config = statusConfig[status] || statusConfig["500"]

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4">
      <span className="text-6xl">{config.icon}</span>
      <h1 className="text-2xl font-semibold">{t(config.titleKey)}</h1>
      <p className="text-muted-foreground">{t(config.descKey)}</p>
      <Button variant="outline" onClick={() => (window.location.href = "/")}>
        {t("error.backHome")}
      </Button>
    </div>
  )
}
