import { ChevronLeft, ChevronRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { useTranslation } from "@/i18n"
import { PAGE_SIZE_OPTIONS } from "@/hooks/use-list-state"

interface PaginationProps {
  totalCount: number
  page: number
  pageSize: number
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
}

export function Pagination({ totalCount, page, pageSize, onPageChange, onPageSizeChange }: PaginationProps) {
  const { t } = useTranslation()
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize))

  if (totalCount === 0) return null

  return (
    <div className="mt-4 flex items-center justify-between">
      <div className="flex items-center gap-4">
        <p className="text-muted-foreground text-sm">{t("common.total", { count: totalCount })}</p>
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground text-sm">{t("common.pageSize")}</span>
          <Select value={String(pageSize)} onValueChange={(v) => onPageSizeChange(Number(v))}>
            <SelectTrigger className="h-8 w-[70px]"><SelectValue /></SelectTrigger>
            <SelectContent>{PAGE_SIZE_OPTIONS.map((s) => <SelectItem key={s} value={String(s)}>{s}</SelectItem>)}</SelectContent>
          </Select>
        </div>
      </div>
      <div className="flex items-center gap-1">
        <Button variant="outline" size="icon" className="h-8 w-8" disabled={page <= 1} onClick={() => onPageChange(page - 1)}><ChevronLeft className="h-4 w-4" /></Button>
        <span className="text-sm px-2">{t("common.page", { page, total: totalPages })}</span>
        <Button variant="outline" size="icon" className="h-8 w-8" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}><ChevronRight className="h-4 w-4" /></Button>
      </div>
    </div>
  )
}
