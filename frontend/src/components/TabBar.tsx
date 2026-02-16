import { lazy, Suspense } from "react";
import { Tabs, Spin } from "antd";
import { useUIStore } from "../store/ui";
import NoteView from "./NoteView";

const GraphView = lazy(() => import("./GraphView"));

/** Tabbed note viewer pane. */
export default function TabBar() {
  const { tabs, activeTab, setActiveTab, closeTab } = useUIStore();

  if (tabs.length === 0) {
    return (
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          height: "100%",
          color: "#6c7086",
          fontSize: 16,
        }}
      >
        Open a note from the sidebar or press ⌘K to search
      </div>
    );
  }

  return (
    <Tabs
      type="editable-card"
      activeKey={activeTab ?? undefined}
      onChange={setActiveTab}
      onEdit={(key, action) => {
        if (action === "remove" && typeof key === "string") closeTab(key);
      }}
      hideAdd
      size="small"
      style={{ height: "100%" }}
      items={tabs.map((t) => ({
        key: t.path,
        label: t.title,
        children:
          t.path === "__graph__" ? (
            <Suspense fallback={<Spin style={{ marginTop: 48 }} />}>
              <GraphView />
            </Suspense>
          ) : (
            <NoteView path={t.path} />
          ),
        closable: true,
      }))}
    />
  );
}
