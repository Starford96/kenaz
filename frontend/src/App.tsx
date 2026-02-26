import { useEffect } from "react";
import { ConfigProvider, Layout, App as AntApp, Drawer } from "antd";
import { darkTheme } from "./styles/theme";
import { useUIStore } from "./store/ui";
import {
  useContainerWidth,
  MOBILE_BREAKPOINT,
  CONTEXT_PANEL_BREAKPOINT,
} from "./hooks/useContainerWidth";
import { useSSE } from "./hooks/useSSE";
import { useUrlSync } from "./hooks/useUrlSync";
import Sidebar from "./components/Sidebar";
import TabBar from "./components/TabBar";
import ContextPanel from "./components/ContextPanel";
import SearchModal from "./components/SearchModal";
import MobileHeader from "./components/MobileHeader";
import BottomActionBar from "./components/BottomActionBar";

const { Sider, Content } = Layout;

export default function App() {
  const {
    sidebarCollapsed,
    contextPanelOpen,
    setSearchOpen,
    mobileDrawer,
    setMobileDrawer,
  } = useUIStore();

  const containerWidth = useContainerWidth();
  const isMobile = containerWidth < MOBILE_BREAKPOINT;
  const canShowContextSider = containerWidth >= CONTEXT_PANEL_BREAKPOINT;

  useSSE();
  useUrlSync();

  useEffect(() => {
    if (isMobile) {
      const { contextPanelOpen, toggleContextPanel } = useUIStore.getState();
      if (contextPanelOpen) toggleContextPanel();
    }
  }, [isMobile]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setSearchOpen(true);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [setSearchOpen]);

  if (isMobile) {
    return (
      <ConfigProvider theme={darkTheme}>
        <AntApp>
          <div className="mobile-shell">
            <MobileHeader />

            <main className="mobile-content">
              <TabBar />
            </main>

            <BottomActionBar />

            <Drawer
              open={mobileDrawer === "sidebar"}
              onClose={() => setMobileDrawer(null)}
              placement="left"
              width="82vw"
              closable={false}
              className="kenaz-drawer"
              styles={{ body: { padding: 0, background: "#181825" } }}
            >
              <div
                style={{
                  padding: "14px 16px 8px",
                  fontSize: 15,
                  fontWeight: 600,
                  color: "#cdd6f4",
                  letterSpacing: 0.5,
                }}
              >
                ᚲ Kenaz
              </div>
              <Sidebar />
            </Drawer>

            <Drawer
              open={mobileDrawer === "context"}
              onClose={() => setMobileDrawer(null)}
              placement="right"
              width="82vw"
              closable={false}
              className="kenaz-drawer"
              styles={{ body: { padding: 0, background: "#181825" } }}
            >
              <ContextPanel />
            </Drawer>
          </div>

          <SearchModal />
        </AntApp>
      </ConfigProvider>
    );
  }

  return (
    <ConfigProvider theme={darkTheme}>
      <AntApp>
        <Layout style={{ height: "100vh" }}>
          {/* Left sidebar: file tree + search trigger. */}
          <Sider
            width={260}
            collapsedWidth={0}
            collapsed={sidebarCollapsed}
            style={{
              borderRight: "1px solid #3a3a4e",
              overflow: "auto",
            }}
          >
            <div
              style={{
                padding: "12px 16px 8px",
                fontSize: 14,
                fontWeight: 600,
                color: "#cdd6f4",
                letterSpacing: 0.5,
              }}
            >
              ᚲ Kenaz
            </div>
            <Sidebar />
          </Sider>

          {/* Center: tabbed note viewer. */}
          <Content
            style={{
              overflow: "auto",
              display: "flex",
              flexDirection: "column",
              minWidth: 0,
            }}
          >
            <TabBar />
          </Content>

          {/* Right context panel: backlinks + outline (hidden on narrow screens). */}
          {contextPanelOpen && canShowContextSider && (
            <Sider
              width={240}
              style={{
                borderLeft: "1px solid #3a3a4e",
                overflow: "auto",
              }}
            >
              <ContextPanel />
            </Sider>
          )}
        </Layout>

        {/* Quick search modal (Cmd/Ctrl+K). */}
        <SearchModal />
      </AntApp>
    </ConfigProvider>
  );
}
