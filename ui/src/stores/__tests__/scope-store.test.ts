import { describe, it, expect, beforeEach } from "vitest"
import { useScopeStore } from "../scope-store"

describe("useScopeStore", () => {
  beforeEach(() => {
    useScopeStore.setState({ workspaceId: null, namespaceId: null })
  })

  it("has correct initial state", () => {
    const state = useScopeStore.getState()
    expect(state.workspaceId).toBeNull()
    expect(state.namespaceId).toBeNull()
  })

  it("setWorkspace updates workspaceId", () => {
    useScopeStore.getState().setWorkspace("ws-1")
    expect(useScopeStore.getState().workspaceId).toBe("ws-1")
  })

  it("setWorkspace resets namespaceId to null", () => {
    useScopeStore.setState({ workspaceId: "ws-1", namespaceId: "ns-1" })
    useScopeStore.getState().setWorkspace("ws-2")
    expect(useScopeStore.getState().workspaceId).toBe("ws-2")
    expect(useScopeStore.getState().namespaceId).toBeNull()
  })

  it("setWorkspace to null resets both", () => {
    useScopeStore.setState({ workspaceId: "ws-1", namespaceId: "ns-1" })
    useScopeStore.getState().setWorkspace(null)
    expect(useScopeStore.getState().workspaceId).toBeNull()
    expect(useScopeStore.getState().namespaceId).toBeNull()
  })

  it("setNamespace updates namespaceId", () => {
    useScopeStore.setState({ workspaceId: "ws-1" })
    useScopeStore.getState().setNamespace("ns-1")
    expect(useScopeStore.getState().namespaceId).toBe("ns-1")
  })

  it("setNamespace to null clears namespaceId", () => {
    useScopeStore.setState({ workspaceId: "ws-1", namespaceId: "ns-1" })
    useScopeStore.getState().setNamespace(null)
    expect(useScopeStore.getState().namespaceId).toBeNull()
    expect(useScopeStore.getState().workspaceId).toBe("ws-1")
  })
})
