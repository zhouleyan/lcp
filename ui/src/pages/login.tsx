import { useState } from "react"
import { useSearchParams } from "react-router"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { LanguageSwitcher } from "@/components/language-switcher"
import { useTranslation } from "@/i18n"
import { loginWithCredentials, startAuthFlow } from "@/lib/auth"

const loginErrorMap: Record<string, string> = {
  "invalid credentials": "login.error.invalidCredentials",
  "account is not active": "login.error.accountInactive",
  "invalid or expired request_id": "login.error.sessionExpired",
}

export default function LoginPage() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const requestId = searchParams.get("request_id")
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (!requestId) {
      // No request_id means user navigated to /login directly — start OIDC flow
      await startAuthFlow()
      return
    }

    setLoading(true)
    try {
      const redirectUri = await loginWithCredentials(username, password, requestId)
      // Navigate using relative path to stay on the same origin,
      // preserving sessionStorage (PKCE code_verifier) across the redirect.
      const url = new URL(redirectUri)
      window.location.href = url.pathname + url.search
    } catch (err) {
      const msg = err instanceof Error ? err.message : ""
      const key = loginErrorMap[msg.toLowerCase()]
      if (msg.toLowerCase() === "invalid or expired request_id") {
        setError(t("login.error.sessionExpired"))
        setTimeout(() => startAuthFlow(), 1500)
        return
      }
      setError(key ? t(key) : t("login.error.failed"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="flex min-h-screen items-center justify-center bg-cover bg-center bg-no-repeat"
      style={{ backgroundImage: "url('/login-bg.svg')" }}
    >
      <Card className="w-full max-w-sm">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-2xl">{t("login.title")}</CardTitle>
            <LanguageSwitcher />
          </div>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">{t("login.username")}</Label>
              <Input
                id="username"
                placeholder={t("login.usernamePlaceholder")}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">{t("login.password")}</Label>
              <Input
                id="password"
                type="password"
                placeholder={t("login.passwordPlaceholder")}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            {error && <p className="text-destructive text-sm">{error}</p>}
            <Button className="w-full" type="submit" disabled={loading}>
              {loading ? "..." : t("login.signIn")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
