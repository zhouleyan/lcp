import { useEffect, useState } from "react"
import { useNavigate, useSearchParams } from "react-router"
import { exchangeCodeForTokens } from "@/lib/auth"

export default function AuthCallbackPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const code = searchParams.get("code")
    if (!code) {
      setError("Missing authorization code")
      return
    }

    exchangeCodeForTokens(code)
      .then(() => navigate("/", { replace: true }))
      .catch((err) => setError(err.message))
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
      <p className="text-muted-foreground">Authenticating...</p>
    </div>
  )
}
