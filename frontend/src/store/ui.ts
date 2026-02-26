import { create } from "zustand";
import { persist } from "zustand/middleware";

export interface Tab {
  path: string;
  title: string;
}

interface UIState {
  // Sidebar.
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;

  // Right panel.
  contextPanelOpen: boolean;
  toggleContextPanel: () => void;

  // Tabs (persisted).
  tabs: Tab[];
  activeTab: string | null;
  openTab: (path: string, title?: string) => void;
  closeTab: (path: string) => void;
  setActiveTab: (path: string) => void;

  // Quick search / command palette.
  searchOpen: boolean;
  setSearchOpen: (open: boolean) => void;

  // Mobile drawer (only one open at a time).
  mobileDrawer: "sidebar" | "context" | null;
  setMobileDrawer: (drawer: "sidebar" | "context" | null) => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      toggleSidebar: () =>
        set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),

      contextPanelOpen: true,
      toggleContextPanel: () =>
        set((s) => ({ contextPanelOpen: !s.contextPanelOpen })),

      tabs: [],
      activeTab: null,
      openTab: (path, title) =>
        set((s) => {
          const exists = s.tabs.find((t) => t.path === path);
          if (exists) return { activeTab: path };
          return {
            tabs: [...s.tabs, { path, title: title ?? path }],
            activeTab: path,
          };
        }),
      closeTab: (path) =>
        set((s) => {
          const tabs = s.tabs.filter((t) => t.path !== path);
          const activeTab =
            s.activeTab === path
              ? (tabs[tabs.length - 1]?.path ?? null)
              : s.activeTab;
          return { tabs, activeTab };
        }),
      setActiveTab: (path) => set({ activeTab: path }),

      searchOpen: false,
      setSearchOpen: (open) => set({ searchOpen: open }),

      mobileDrawer: null,
      setMobileDrawer: (drawer) => set({ mobileDrawer: drawer }),
    }),
    {
      name: "kenaz-ui",
      // Only persist tabs, activeTab, sidebar and panel state.
      partialize: (state) => ({
        tabs: state.tabs,
        activeTab: state.activeTab,
        sidebarCollapsed: state.sidebarCollapsed,
        contextPanelOpen: state.contextPanelOpen,
      }),
    },
  ),
);
