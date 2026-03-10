import { useCallback } from "react"
import { format, setHours, setMinutes, setSeconds } from "date-fns"
import { CalendarIcon } from "lucide-react"
import type { DateRange } from "react-day-picker"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import { Input } from "@/components/ui/input"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"

interface DateRangePickerProps {
  value: DateRange | undefined
  onChange: (range: DateRange | undefined) => void
  placeholder?: string
  resetLabel?: string
  className?: string
}

function applyTime(date: Date, timeStr: string): Date {
  const [h, m, s] = timeStr.split(":").map(Number)
  return setSeconds(setMinutes(setHours(date, h || 0), m || 0), s || 0)
}

function extractTime(date: Date | undefined): string {
  if (!date) return "00:00:00"
  return format(date, "HH:mm:ss")
}

function DateRangePicker({
  value,
  onChange,
  placeholder = "Pick a date range",
  resetLabel = "Reset",
  className,
}: DateRangePickerProps) {
  const handleCalendarSelect = useCallback(
    (range: DateRange | undefined) => {
      if (!range) {
        onChange(undefined)
        return
      }
      // Preserve existing time when date changes
      const from = range.from
        ? applyTime(range.from, extractTime(value?.from))
        : undefined
      const to = range.to
        ? applyTime(range.to, extractTime(value?.to))
        : undefined
      onChange({ from, to })
    },
    [onChange, value?.from, value?.to]
  )

  const handleFromTime = useCallback(
    (timeStr: string) => {
      if (!value?.from) return
      onChange({ ...value, from: applyTime(value.from, timeStr) })
    },
    [onChange, value]
  )

  const handleToTime = useCallback(
    (timeStr: string) => {
      if (!value?.to) return
      onChange({ ...value, to: applyTime(value.to, timeStr) })
    },
    [onChange, value]
  )

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn(
            "justify-start rounded-none shadow-none text-left font-normal",
            !value?.from && "text-muted-foreground",
            className
          )}
        >
          <CalendarIcon className="size-4" />
          {value?.from ? (
            value.to ? (
              <>
                {format(value.from, "yyyy-MM-dd HH:mm:ss")} –{" "}
                {format(value.to, "yyyy-MM-dd HH:mm:ss")}
              </>
            ) : (
              format(value.from, "yyyy-MM-dd HH:mm:ss")
            )
          ) : (
            <span>{placeholder}</span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="range"
          selected={value}
          onSelect={handleCalendarSelect}
          numberOfMonths={2}
        />
        <div className="border-t px-3 py-2 flex items-center gap-4 text-sm">
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground whitespace-nowrap">
              {value?.from ? format(value.from, "MM-dd") : "--"}
            </span>
            <Input
              type="time"
              step="1"
              value={extractTime(value?.from)}
              onChange={(e) => handleFromTime(e.target.value)}
              disabled={!value?.from}
              className="h-7 w-[110px] rounded-none text-xs"
            />
          </div>
          <span className="text-muted-foreground">–</span>
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground whitespace-nowrap">
              {value?.to ? format(value.to, "MM-dd") : "--"}
            </span>
            <Input
              type="time"
              step="1"
              value={extractTime(value?.to)}
              onChange={(e) => handleToTime(e.target.value)}
              disabled={!value?.to}
              className="h-7 w-[110px] rounded-none text-xs"
            />
          </div>
          <div className="ml-auto">
            <Button
              variant="ghost"
              size="sm"
              disabled={!value?.from}
              onClick={() => onChange(undefined)}
            >
              {resetLabel}
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}

export { DateRangePicker }
export type { DateRangePickerProps }
