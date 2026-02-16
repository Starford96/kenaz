import { useEffect } from "react";
import { ConfigProvider, Layout } from "antd";
import { darkTheme } from "./styles/theme";
import { useUIStore } from "./store/ui";
import { useSSE } from "./hooks/useSSE";
import Sidebar from "./components/Sidebar";
import TabBar from "./components/TabBar";
import ContextPanel from "./components/ContextPanel";
import SearchModal from "./components/SearchModal";

const { Sider, Content } = Layout;

export default function App() {
  const { sidebarCollapsed, contextPanelOpen, setSearchOpen } = useUIStore();

  // Connect to SSE for real-time updates.
  useSSE();

  // Global keyboard shortcut: Cmd/Ctrl+K → quick search.
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

  return (
    <ConfigProvider theme={darkTheme}>
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
          }}
        >
          <TabBar />
        </Content>

        {/* Right context panel: backlinks + outline. */}
        {contextPanelOpen && (
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
    </ConfigProvider>
  );
}
