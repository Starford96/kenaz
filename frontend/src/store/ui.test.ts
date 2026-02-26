import { describe, it, expect, beforeEach } from "vitest";
import { useUIStore } from "./ui";

describe("useUIStore", () => {
  beforeEach(() => {
    useUIStore.setState({
      tabs: [],
      activeTab: null,
      sidebarCollapsed: false,
      contextPanelOpen: true,
      searchOpen: false,
      mobileDrawer: null,
    });
  });

  it("opens a new tab and sets it active", () => {
    useUIStore.getState().openTab("hello.md", "Hello");
    const state = useUIStore.getState();
    expect(state.tabs).toHaveLength(1);
    expect(state.tabs[0]).toEqual({ path: "hello.md", title: "Hello" });
    expect(state.activeTab).toBe("hello.md");
  });

  it("does not duplicate tabs on re-open", () => {
    useUIStore.getState().openTab("a.md", "A");
    useUIStore.getState().openTab("b.md", "B");
    useUIStore.getState().openTab("a.md", "A");
    expect(useUIStore.getState().tabs).toHaveLength(2);
    expect(useUIStore.getState().activeTab).toBe("a.md");
  });

  it("closes a tab and activates the last remaining", () => {
    useUIStore.getState().openTab("a.md", "A");
    useUIStore.getState().openTab("b.md", "B");
    useUIStore.getState().closeTab("b.md");
    const state = useUIStore.getState();
    expect(state.tabs).toHaveLength(1);
    expect(state.activeTab).toBe("a.md");
  });

  it("toggles sidebar collapsed", () => {
    expect(useUIStore.getState().sidebarCollapsed).toBe(false);
    useUIStore.getState().toggleSidebar();
    expect(useUIStore.getState().sidebarCollapsed).toBe(true);
    useUIStore.getState().toggleSidebar();
    expect(useUIStore.getState().sidebarCollapsed).toBe(false);
  });

  it("sets search open state", () => {
    useUIStore.getState().setSearchOpen(true);
    expect(useUIStore.getState().searchOpen).toBe(true);
    useUIStore.getState().setSearchOpen(false);
    expect(useUIStore.getState().searchOpen).toBe(false);
  });
});
