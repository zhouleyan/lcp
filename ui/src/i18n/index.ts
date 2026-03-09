import { create } from "zustand"
import { persist } from "zustand/middleware"
import type { Locale, Messages } from "./types"
import zhCN from "./locales/zh-CN"
import enUS from "./locales/en-US"

const messages: Record<Locale, Messages> = {
  "zh-CN": zhCN,
  "en-US": enUS,
}

interface I18nState {
  locale: Locale
  setLocale: (locale: Locale) => void
}

export const useI18nStore = create<I18nState>()(
  persist(
    (set) => ({
      locale: "zh-CN",
      setLocale: (locale) => set({ locale }),
    }),
    { name: "lcp-locale" },
  ),
)

export function useTranslation() {
  const { locale, setLocale } = useI18nStore()

  const t = (key: string, vars?: Record<string, string | number>): string => {
    const raw = messages[locale]?.[key]
    let msg = raw ?? (vars?.defaultValue != null ? String(vars.defaultValue) : key)
    if (vars) {
      for (const [k, v] of Object.entries(vars)) {
        if (k === "defaultValue") continue
        msg = msg.replace(`{${k}}`, String(v))
      }
    }
    return msg
  }

  return { t, locale, setLocale }
}

export type { Locale } from "./types"
