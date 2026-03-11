import { useEffect, useRef, useState } from "react"
import { useNavigate, useSearchParams } from "react-router"
import { exchangeCodeForTokens } from "@/lib/auth"
import { useTranslation } from "@/i18n"

export default function AuthCallbackPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)
  const exchanged = useRef(false)

  useEffect(() => {
    if (exchanged.current) return
    exchanged.current = true

    const code = searchParams.get("code")
    if (!code) {
      setError(t("auth.missingCode"))
      return
    }

    exchangeCodeForTokens(code)
      .then(() => navigate("/", { replace: true }))
      .catch(async (err) => {
        if (err instanceof Error && err.message === "Missing PKCE code verifier") {
          // PKCE verifier lost (new tab, page refresh, etc.) — restart auth flow
          const { startAuthFlow } = await import("@/lib/auth")
          await startAuthFlow()
          return
        }
        setError(err.message)
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams, navigate])

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-destructive">{error}</p>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p className="text-muted-foreground">{t("auth.authenticating")}</p>
    </div>
  )
}
