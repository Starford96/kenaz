import { create } from "zustand";

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

  // Tabs.
  tabs: Tab[];
  activeTab: string | null;
  openTab: (path: string, title?: string) => void;
  closeTab: (path: string) => void;
  setActiveTab: (path: string) => void;

  // Quick search modal.
  searchOpen: boolean;
  setSearchOpen: (open: boolean) => void;
}

export const useUIStore = create<UIState>((set) => ({
  sidebarCollapsed: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),

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
}));
