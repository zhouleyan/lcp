import { useCallback, useEffect, useRef, useState } from "react"

export const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

export interface UseListStateOptions {
  defaultSortBy?: string
  defaultSortOrder?: "asc" | "desc"
  defaultPageSize?: number
}

export function useListState(options: UseListStateOptions = {}) {
  const {
    defaultSortBy = "created_at",
    defaultSortOrder = "desc",
    defaultPageSize = 20,
  } = options

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(defaultPageSize)
  const [sortBy, setSortBy] = useState(defaultSortBy)
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">(defaultSortOrder)
  const [searchInput, setSearchInput] = useState("")
  const [search, setSearch] = useState("")
  const [selected, setSelected] = useState<Set<string>>(new Set())

  // Debounce search
  const searchTimer = useRef<ReturnType<typeof setTimeout>>(null)
  useEffect(() => {
    searchTimer.current = setTimeout(() => setSearch(searchInput), 300)
    return () => { if (searchTimer.current) clearTimeout(searchTimer.current) }
  }, [searchInput])

  const sortByRef = useRef(defaultSortBy)
  const handleSort = useCallback((field: string) => {
    if (sortByRef.current === field) {
      setSortOrder((o) => (o === "asc" ? "desc" : "asc"))
    } else {
      sortByRef.current = field
      setSortBy(field)
      setSortOrder("asc")
    }
  }, [])

  const toggleAll = useCallback((ids: string[]) => {
    setSelected((prev) => prev.size === ids.length ? new Set() : new Set(ids))
  }, [])

  const toggleOne = useCallback((id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }, [])

  const clearSelection = useCallback(() => setSelected(new Set()), [])

  return {
    page, setPage,
    pageSize, setPageSize,
    sortBy, sortOrder, handleSort,
    searchInput, setSearchInput, search,
    selected, toggleAll, toggleOne, clearSelection,
  }
}
