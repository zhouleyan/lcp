import { useState, useMemo, useCallback } from "react"
import { ChevronRight, ChevronDown, Search } from "lucide-react"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { useTranslation } from "@/i18n"
import type { Permission } from "@/api/types"

// --- Constants ---

const VERB_GROUPS = [
  { key: "read", verbs: ["list", "get"] },
  { key: "create", verbs: ["create"] },
  { key: "update", verbs: ["update", "patch"] },
  { key: "delete", verbs: ["delete", "deleteCollection"] },
] as const

const SCOPE_LEVELS: Record<string, number> = {
  platform: 0,
  workspace: 1,
  namespace: 2,
}

const STANDARD_VERBS = new Set(VERB_GROUPS.flatMap((g) => g.verbs))

function getVerb(code: string): string {
  return code.split(":").pop() || ""
}

// --- Types ---

interface GroupNode {
  key: string
  wildcardPattern: string
  i18nKey: string
  children: GroupNode[]
  permissions: PermNode[]
}

interface PermNode {
  code: string
  i18nKey: string
}

// --- Helpers ---

export function patternCovers(pattern: string, code: string): boolean {
  if (pattern === "*:*") return true
  const starIdx = pattern.indexOf("*")
  if (starIdx === -1) return pattern === code
  const prefix = pattern.slice(0, starIdx)
  const suffix = pattern.slice(starIdx + 1)
  return code.startsWith(prefix) && code.endsWith(suffix) && code.length > prefix.length + suffix.length
}

function isSelected(rules: string[], code: string): boolean {
  if (rules.includes(code)) return true
  return rules.some((r) => r.includes("*") && patternCovers(r, code))
}

function isLocked(rules: string[], code: string): boolean {
  return rules.some((r) => r !== code && r.includes("*") && patternCovers(r, code))
}

function isCoarserOrEqual(a: string, b: string): boolean {
  if (a === b) return true
  if (!a.includes("*")) return false
  return patternCovers(a, b)
}

function buildTree(perms: Permission[]): GroupNode {
  // Deduplicate by code — same code may appear at multiple scopes
  const seen = new Set<string>()
  const unique = perms.filter((p) => {
    if (seen.has(p.spec.code)) return false
    seen.add(p.spec.code)
    return true
  })

  const moduleMap = new Map<string, Map<string, Permission[]>>()

  for (const p of unique) {
    const parts = p.spec.code.split(":")
    const module = parts[0]
    const resourceKey = parts.slice(0, -1).join(":")

    if (!moduleMap.has(module)) moduleMap.set(module, new Map())
    const resourceMap = moduleMap.get(module)!
    if (!resourceMap.has(resourceKey)) resourceMap.set(resourceKey, [])
    resourceMap.get(resourceKey)!.push(p)
  }

  const root: GroupNode = {
    key: "root",
    wildcardPattern: "*:*",
    i18nKey: "perm.group.all",
    children: [],
    permissions: [],
  }

  for (const [module, resourceMap] of moduleMap) {
    const moduleNode: GroupNode = {
      key: module,
      wildcardPattern: `${module}:*`,
      i18nKey: `perm.group.${module}`,
      children: [],
      permissions: [],
    }

    const topResourceMap = new Map<string, { key: string; subResources: Map<string, Permission[]> }>()

    for (const [resourceKey, permsInGroup] of resourceMap) {
      const parts = resourceKey.split(":")
      const topResource = parts[1]
      if (!topResourceMap.has(topResource)) {
        topResourceMap.set(topResource, { key: topResource, subResources: new Map() })
      }
      topResourceMap.get(topResource)!.subResources.set(resourceKey, permsInGroup)
    }

    for (const [topResource, { subResources }] of topResourceMap) {
      const resourceNode: GroupNode = {
        key: `${module}:${topResource}`,
        wildcardPattern: `${module}:${topResource}:*`,
        i18nKey: `perm.group.${module}.${topResource}`,
        children: [],
        permissions: [],
      }

      for (const [resourceKey, permsInGroup] of subResources) {
        const parts = resourceKey.split(":")
        if (parts.length <= 2) {
          for (const p of permsInGroup) {
            resourceNode.permissions.push({
              code: p.spec.code,
              i18nKey: `perm.${p.spec.code}`,
            })
          }
        } else {
          const subResourceName = parts.slice(2).join(":")
          const subNode: GroupNode = {
            key: resourceKey,
            wildcardPattern: `${resourceKey}:*`,
            i18nKey: `perm.group.${module}.${topResource}.${subResourceName.replace(/:/g, ".")}`,
            children: [],
            permissions: permsInGroup.map((p) => ({
              code: p.spec.code,
              i18nKey: `perm.${p.spec.code}`,
            })),
          }
          resourceNode.children.push(subNode)
        }
      }

      moduleNode.children.push(resourceNode)
    }

    root.children.push(moduleNode)
  }

  return root
}

function getAllCodes(node: GroupNode): string[] {
  const codes: string[] = []
  for (const p of node.permissions) codes.push(p.code)
  for (const child of node.children) codes.push(...getAllCodes(child))
  return codes
}

// --- Components ---

export function PermissionSelector({
  permissions,
  value,
  onChange,
  readOnly,
  scope,
}: {
  permissions: Permission[]
  value: string[]
  onChange?: (rules: string[]) => void
  readOnly?: boolean
  scope?: "platform" | "workspace" | "namespace"
}) {
  const { t } = useTranslation()
  const [search, setSearch] = useState("")
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set(["root"]))

  const filteredPermissions = useMemo(() => {
    if (!scope || scope === "platform") return permissions
    const minLevel = SCOPE_LEVELS[scope] ?? 0
    return permissions.filter((p) => (SCOPE_LEVELS[p.spec.scope] ?? 0) >= minLevel)
  }, [permissions, scope])

  const tree = useMemo(() => buildTree(filteredPermissions), [filteredPermissions])

  const allGroupKeys = useMemo(() => {
    const keys = new Set<string>()
    const walk = (node: GroupNode) => {
      keys.add(node.key)
      node.children.forEach(walk)
    }
    walk(tree)
    return keys
  }, [tree])

  useState(() => {
    setExpanded(allGroupKeys)
  })

  const toggleExpand = useCallback((key: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }, [])

  const noop = useCallback(() => {}, [])

  const toggleWildcard = useCallback(
    (pattern: string, allCodes: string[]) => {
      if (!onChange) return
      if (value.includes(pattern)) {
        onChange(value.filter((r) => r !== pattern))
      } else {
        const cleaned = value.filter((r) => !allCodes.includes(r) && !(r.includes("*") && isCoarserOrEqual(pattern, r)))
        onChange([...cleaned, pattern])
      }
    },
    [value, onChange],
  )

  const togglePermission = useCallback(
    (code: string) => {
      if (!onChange) return
      if (value.includes(code)) {
        onChange(value.filter((r) => r !== code))
      } else {
        onChange([...value, code])
      }
    },
    [value, onChange],
  )

  const matchingCodes = useMemo(() => {
    if (!search) return null
    const lower = search.toLowerCase()
    const codes = new Set<string>()
    for (const p of filteredPermissions) {
      const desc = t(`perm.${p.spec.code}`, { defaultValue: p.spec.description || p.spec.code })
      if (
        p.spec.code.toLowerCase().includes(lower) ||
        desc.toLowerCase().includes(lower)
      ) {
        codes.add(p.spec.code)
      }
    }
    return codes
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, filteredPermissions])

  if (readOnly) {
    return (
      <div className="py-1">
        <TreeNode
          node={tree}
          value={value}
          onChange={noop}
          toggleWildcard={toggleWildcard}
          toggleExpand={toggleExpand}
          togglePermission={togglePermission}
          expanded={expanded}
          matchingCodes={null}
          depth={0}
          readOnly
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2 min-h-0 flex-1">
      <div className="relative">
        <Search className="text-muted-foreground absolute left-2.5 top-2.5 h-4 w-4" />
        <Input
          placeholder={t("common.search")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="h-9 pl-9"
        />
      </div>
      <div className="flex-1 min-h-0 overflow-y-auto rounded-md border py-1">
        <TreeNode
          node={tree}
          value={value}
          onChange={onChange ?? noop}
          toggleWildcard={toggleWildcard}
          toggleExpand={toggleExpand}
          togglePermission={togglePermission}
          expanded={expanded}
          matchingCodes={matchingCodes}
          depth={0}
          readOnly={readOnly}
        />
      </div>
    </div>
  )
}

function TreeNode({
  node,
  value,
  onChange,
  toggleWildcard,
  toggleExpand,
  togglePermission,
  expanded,
  matchingCodes,
  depth,
  readOnly,
}: {
  node: GroupNode
  value: string[]
  onChange: (rules: string[]) => void
  toggleWildcard: (pattern: string, allCodes: string[]) => void
  toggleExpand: (key: string) => void
  togglePermission: (code: string) => void
  expanded: Set<string>
  matchingCodes: Set<string> | null
  depth: number
  readOnly?: boolean
}) {
  const { t } = useTranslation()
  const allCodes = useMemo(() => getAllCodes(node), [node])
  const wildcardSelected = isSelected(value, node.wildcardPattern) || value.includes(node.wildcardPattern)
  const checked: boolean | "indeterminate" = useMemo(() => {
    if (wildcardSelected) return true
    const someSelected = allCodes.some((c) => isSelected(value, c))
    if (!someSelected) return false
    const allSelected = allCodes.every((c) => isSelected(value, c))
    return allSelected ? true : "indeterminate"
  }, [wildcardSelected, allCodes, value])
  const locked = isLocked(value, node.wildcardPattern)

  const hasMatch = matchingCodes === null || allCodes.some((c) => matchingCodes.has(c))
  if (!hasMatch) return null

  const isOpen = expanded.has(node.key)
  const isRoot = node.key === "root"

  // Compute verb groups from ALL codes recursively under this node
  const verbGroupData = useMemo(() => {
    return VERB_GROUPS.map((group) => {
      const codes = allCodes.filter((c) => (group.verbs as readonly string[]).includes(getVerb(c)))
      return { ...group, codes }
    }).filter((g) => g.codes.length > 0)
  }, [allCodes])

  // Custom action permissions (direct only, not recursive)
  const customPerms = useMemo(() => {
    return node.permissions.filter((p) => !(STANDARD_VERBS as Set<string>).has(getVerb(p.code)))
  }, [node.permissions])

  const hasChildren = node.children.length > 0 || verbGroupData.length > 0 || customPerms.length > 0

  // Toggle verb group
  const toggleVerbGroup = useCallback(
    (group: { verbs: readonly string[]; codes: string[] }) => {
      if (isRoot) {
        // Root: use *:verb patterns
        const patterns = group.verbs.map((v) => `*:${v}`)
        const allPatternsSelected = patterns.every((p) => value.includes(p))
        if (allPatternsSelected) {
          onChange(value.filter((r) => !patterns.includes(r)))
        } else {
          const newPatterns = patterns.filter((p) => !value.includes(p))
          const cleaned = value.filter((r) => !group.codes.includes(r))
          onChange([...cleaned, ...newPatterns])
        }
      } else {
        // Non-root: toggle individual codes
        const allSelected = group.codes.every((c) => isSelected(value, c))
        if (allSelected) {
          onChange(value.filter((r) => !group.codes.includes(r)))
        } else {
          const toAdd = group.codes.filter((c) => !isSelected(value, c))
          onChange([...value, ...toAdd])
        }
      }
    },
    [isRoot, value, onChange],
  )

  return (
    <div>
      {/* Group header */}
      <div
        className={`flex items-center gap-2 rounded px-2 py-1 ${readOnly ? "" : "hover:bg-accent cursor-pointer"}`}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
      >
        <button
          type="button"
          className="flex h-4 w-4 shrink-0 items-center justify-center"
          onClick={() => toggleExpand(node.key)}
        >
          {hasChildren ? (
            isOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />
          ) : (
            <span className="w-3.5" />
          )}
        </button>
        <Checkbox
          checked={checked}
          disabled={readOnly || locked}
          onCheckedChange={readOnly ? undefined : () => toggleWildcard(node.wildcardPattern, allCodes)}
        />
        <span className="text-sm font-medium">{t(node.i18nKey, { defaultValue: node.key })}</span>
        {!readOnly && <span className="text-muted-foreground font-mono text-xs">{node.wildcardPattern}</span>}
      </div>

      {/* Expanded content */}
      {isOpen && (
        <>
          {/* Verb group toggles + custom actions in one row */}
          {(verbGroupData.length > 0 || customPerms.length > 0) && (
            <div
              className="flex flex-wrap gap-x-1 gap-y-0.5 py-0.5"
              style={{ paddingLeft: `${(depth + 1) * 16 + 28}px`, paddingRight: 8 }}
            >
              {verbGroupData.map((group) => {
                let groupChecked: boolean | "indeterminate" = false
                let groupLocked = false

                if (isRoot) {
                  const patterns = group.verbs.map((v) => `*:${v}`)
                  const allPatternsSelected = patterns.every((p) => value.includes(p))
                  const somePatternsSelected = patterns.some((p) => value.includes(p))
                  groupChecked = allPatternsSelected || value.includes("*:*")
                  if (!groupChecked && (somePatternsSelected || group.codes.some((c) => isSelected(value, c)))) {
                    groupChecked = "indeterminate"
                  }
                  groupLocked = value.includes("*:*")
                } else {
                  const allSelected = group.codes.every((c) => isSelected(value, c))
                  const someSelected = group.codes.some((c) => isSelected(value, c))
                  groupChecked = allSelected
                  if (!allSelected && someSelected) groupChecked = "indeterminate"
                  groupLocked = group.codes.every((c) => isLocked(value, c))
                }

                return (
                  <label
                    key={group.key}
                    className={`flex items-center gap-1 rounded bg-muted/50 px-2 py-0.5 text-xs ${readOnly ? "" : "hover:bg-accent cursor-pointer"}`}
                    title={readOnly ? undefined : (isRoot ? group.verbs.map((v) => `*:${v}`).join(", ") : group.codes.join(", "))}
                  >
                    <Checkbox
                      className="h-3.5 w-3.5"
                      checked={groupChecked}
                      disabled={readOnly || groupLocked}
                      onCheckedChange={readOnly ? undefined : () => toggleVerbGroup(group)}
                    />
                    <span className="whitespace-nowrap font-medium">
                      {t(`perm.verbGroup.${group.key}`)}
                    </span>
                  </label>
                )
              })}

              {/* Custom action permissions */}
              {customPerms.map((perm) => {
                const show = matchingCodes === null || matchingCodes.has(perm.code)
                if (!show) return null
                const permChecked = isSelected(value, perm.code)
                const permLocked = isLocked(value, perm.code)
                const desc = t(perm.i18nKey, { defaultValue: perm.code })
                return (
                  <label
                    key={perm.code}
                    className={`flex items-center gap-1 rounded px-1.5 py-0.5 text-xs ${readOnly ? "" : "hover:bg-accent cursor-pointer"}`}
                    title={readOnly ? undefined : perm.code}
                  >
                    <Checkbox
                      className="h-3.5 w-3.5"
                      checked={permChecked}
                      disabled={readOnly || permLocked}
                      onCheckedChange={readOnly ? undefined : () => togglePermission(perm.code)}
                    />
                    <span className="text-muted-foreground whitespace-nowrap">{desc}</span>
                  </label>
                )
              })}
            </div>
          )}

          {/* Child groups */}
          {node.children.map((child) => (
            <TreeNode
              key={child.key}
              node={child}
              value={value}
              onChange={onChange}
              toggleWildcard={toggleWildcard}
              toggleExpand={toggleExpand}
              togglePermission={togglePermission}
              expanded={expanded}
              matchingCodes={matchingCodes}
              depth={depth + 1}
              readOnly={readOnly}
            />
          ))}
        </>
      )}
    </div>
  )
}
