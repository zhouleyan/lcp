import { describe, it, expect, beforeEach } from "vitest"
import { renderHook } from "@testing-library/react"
import { usePermission } from "../use-permission"
import { usePermissionStore } from "@/stores/permission-store"
import type { UserPermissionsSpec } from "@/api/types"

function setPermissions(perms: UserPermissionsSpec | null) {
  usePermissionStore.setState({ permissions: perms })
}

const basePerms: UserPermissionsSpec = {
  isPlatformAdmin: false,
  platform: [],
  workspaces: {},
  namespaces: {},
}

describe("usePermission", () => {
  beforeEach(() => {
    setPermissions(null)
  })

  // --- isPlatformAdmin ---

  describe("isPlatformAdmin", () => {
    it("returns false when permissions is null", () => {
      const { result } = renderHook(() => usePermission())
      expect(result.current.isPlatformAdmin).toBe(false)
    })

    it("returns true when isPlatformAdmin is true", () => {
      setPermissions({ ...basePerms, isPlatformAdmin: true })
      const { result } = renderHook(() => usePermission())
      expect(result.current.isPlatformAdmin).toBe(true)
    })

    it("returns false when isPlatformAdmin is false", () => {
      setPermissions({ ...basePerms, isPlatformAdmin: false })
      const { result } = renderHook(() => usePermission())
      expect(result.current.isPlatformAdmin).toBe(false)
    })
  })

  // --- hasPermission (no scope) ---

  describe("hasPermission without scope", () => {
    it("returns false when permissions is null", () => {
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("iam:users:list")).toBe(false)
    })

    it("returns true for any code when isPlatformAdmin", () => {
      setPermissions({ ...basePerms, isPlatformAdmin: true })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("anything")).toBe(true)
    })

    it("returns true when platform includes code", () => {
      setPermissions({ ...basePerms, platform: ["iam:users:list"] })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("iam:users:list")).toBe(true)
    })

    it("returns false when platform does not include code", () => {
      setPermissions({ ...basePerms, platform: ["iam:users:list"] })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("iam:users:delete")).toBe(false)
    })
  })

  // --- hasPermission (workspaceId scope) ---

  describe("hasPermission with workspaceId scope", () => {
    it("returns true when workspace permissions include code", () => {
      setPermissions({
        ...basePerms,
        workspaces: {
          "ws-1": { roleNames: ["admin"], permissions: ["iam:workspaces:update"] },
        },
      })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("iam:workspaces:update", { workspaceId: "ws-1" })).toBe(true)
    })

    it("returns false when workspace permissions do not include code", () => {
      setPermissions({
        ...basePerms,
        workspaces: {
          "ws-1": { roleNames: ["viewer"], permissions: ["iam:workspaces:get"] },
        },
      })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("iam:workspaces:delete", { workspaceId: "ws-1" })).toBe(
        false,
      )
    })

    it("returns true when isPlatformAdmin regardless of scope", () => {
      setPermissions({ ...basePerms, isPlatformAdmin: true })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("anything", { workspaceId: "ws-1" })).toBe(true)
    })

    it("returns true when platform permission covers workspace scope", () => {
      setPermissions({ ...basePerms, platform: ["iam:workspaces:update"] })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("iam:workspaces:update", { workspaceId: "ws-1" })).toBe(true)
    })
  })

  // --- hasPermission (namespaceId scope) ---

  describe("hasPermission with namespaceId scope", () => {
    it("returns true when namespace permissions include code", () => {
      setPermissions({
        ...basePerms,
        namespaces: {
          "ns-1": {
            roleNames: ["editor"],
            workspaceId: "ws-1",
            permissions: ["iam:namespaces:update"],
          },
        },
        workspaces: {
          "ws-1": { roleNames: ["viewer"], permissions: ["iam:workspaces:get"] },
        },
      })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("iam:namespaces:update", { namespaceId: "ns-1" })).toBe(true)
    })

    it("returns true when parent workspace permissions include code (inheritance)", () => {
      setPermissions({
        ...basePerms,
        namespaces: {
          "ns-1": {
            roleNames: ["viewer"],
            workspaceId: "ws-1",
            permissions: ["iam:namespaces:get"],
          },
        },
        workspaces: {
          "ws-1": { roleNames: ["admin"], permissions: ["iam:namespaces:update"] },
        },
      })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("iam:namespaces:update", { namespaceId: "ns-1" })).toBe(true)
    })

    it("returns false when neither namespace nor parent workspace include code", () => {
      setPermissions({
        ...basePerms,
        namespaces: {
          "ns-1": {
            roleNames: ["viewer"],
            workspaceId: "ws-1",
            permissions: ["iam:namespaces:get"],
          },
        },
        workspaces: {
          "ws-1": { roleNames: ["viewer"], permissions: ["iam:workspaces:get"] },
        },
      })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasPermission("iam:namespaces:delete", { namespaceId: "ns-1" })).toBe(
        false,
      )
    })
  })

  // --- hasAnyPermission ---

  describe("hasAnyPermission", () => {
    it("returns true when platform includes code", () => {
      setPermissions({ ...basePerms, platform: ["iam:users:list"] })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasAnyPermission("iam:users:list")).toBe(true)
    })

    it("returns true when any workspace includes code", () => {
      setPermissions({
        ...basePerms,
        workspaces: {
          "ws-1": { roleNames: ["admin"], permissions: ["iam:workspaces:update"] },
        },
      })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasAnyPermission("iam:workspaces:update")).toBe(true)
    })

    it("returns true when any namespace includes code", () => {
      setPermissions({
        ...basePerms,
        namespaces: {
          "ns-1": {
            roleNames: ["editor"],
            workspaceId: "ws-1",
            permissions: ["iam:namespaces:update"],
          },
        },
      })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasAnyPermission("iam:namespaces:update")).toBe(true)
    })

    it("returns false when no scope includes code", () => {
      setPermissions({
        ...basePerms,
        platform: ["iam:users:list"],
        workspaces: {
          "ws-1": { roleNames: ["viewer"], permissions: ["iam:workspaces:get"] },
        },
        namespaces: {
          "ns-1": {
            roleNames: ["viewer"],
            workspaceId: "ws-1",
            permissions: ["iam:namespaces:get"],
          },
        },
      })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasAnyPermission("iam:roles:delete")).toBe(false)
    })

    it("returns true when isPlatformAdmin", () => {
      setPermissions({ ...basePerms, isPlatformAdmin: true })
      const { result } = renderHook(() => usePermission())
      expect(result.current.hasAnyPermission("anything")).toBe(true)
    })
  })
})
