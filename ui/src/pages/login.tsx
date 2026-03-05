import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useTranslation } from "@/i18n"

export default function LoginPage() {
  const { t } = useTranslation()

  return (
    <div className="flex min-h-screen items-center justify-center">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-center text-2xl">{t("login.title")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="username">{t("login.username")}</Label>
            <Input id="username" placeholder={t("login.usernamePlaceholder")} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">{t("login.password")}</Label>
            <Input id="password" type="password" placeholder={t("login.passwordPlaceholder")} />
          </div>
          <Button className="w-full">{t("login.signIn")}</Button>
        </CardContent>
      </Card>
    </div>
  )
}
