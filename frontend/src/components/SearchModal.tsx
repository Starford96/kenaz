import { useState, useCallback, useEffect, useRef, useMemo } from "react";
import { Modal, Input, Typography, Empty } from "antd";
import {
  SearchOutlined,
  FileMarkdownOutlined,
  PlusOutlined,
  ApartmentOutlined,
  MenuOutlined,
  AppstoreOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { searchNotes, listNotes, type SearchResult } from "../api/notes";
import { useUIStore } from "../store/ui";
import { useIsMobile } from "../hooks/useIsMobile";

const { Text } = Typography;

interface PaletteItem {
  key: string;
  label: string;
  description?: string;
  icon: React.ReactNode;
  kind: "file" | "command";
  action: () => void;
}

/** Global command palette / quick search, triggered by Cmd/Ctrl+K. */
export default function SearchModal() {
  const {
    searchOpen,
    setSearchOpen,
    openTab,
    toggleSidebar,
    toggleContextPanel,
  } = useUIStore();
  const isMobile = useIsMobile();
  const [query, setQuery] = useState("");
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  // Preload notes list for empty-query file browsing.
  const { data: notesList } = useQuery({
    queryKey: ["notes"],
    queryFn: () => listNotes({ limit: 500 }),
  });

  // Static commands.
  const commands: PaletteItem[] = useMemo(
    () => [
      {
        key: "cmd:new-note",
        label: "New Note",
        description: "Create a new markdown note",
        icon: <PlusOutlined />,
        kind: "command",
        action: () => {
          // Close palette, let sidebar handle creation.
          // We use a tiny hack: dispatch a custom event the sidebar listens to.
          window.dispatchEvent(new CustomEvent("kenaz:create-note"));
        },
      },
      {
        key: "cmd:graph",
        label: "Open Graph View",
        description: "Interactive knowledge graph",
        icon: <ApartmentOutlined />,
        kind: "command",
        action: () => openTab("__graph__", "Graph"),
      },
      {
        key: "cmd:toggle-sidebar",
        label: "Toggle Sidebar",
        icon: <MenuOutlined />,
        kind: "command",
        action: toggleSidebar,
      },
      {
        key: "cmd:toggle-context",
        label: "Toggle Context Panel",
        icon: <AppstoreOutlined />,
        kind: "command",
        action: toggleContextPanel,
      },
    ],
    [openTab, toggleSidebar, toggleContextPanel],
  );

  // Debounced FTS search.
  useEffect(() => {
    if (!query.trim()) {
      setSearchResults([]);
      return;
    }
    setLoading(true);
    const timer = setTimeout(async () => {
      try {
        const r = await searchNotes(query);
        setSearchResults(r ?? []);
        setSelected(0);
      } catch {
        setSearchResults([]);
      } finally {
        setLoading(false);
      }
    }, 200);
    return () => clearTimeout(timer);
  }, [query]);

  // Build combined items list.
  const items: PaletteItem[] = useMemo(() => {
    const q = query.trim().toLowerCase();

    if (q) {
      // Search results as file items.
      const fileItems: PaletteItem[] = searchResults.map((r) => ({
        key: `file:${r.path}`,
        label: r.title || r.path,
        description: r.snippet,
        icon: <FileMarkdownOutlined />,
        kind: "file",
        action: () => openTab(r.path, r.title || r.path),
      }));

      // Filter commands by query.
      const cmdItems = commands.filter((c) =>
        c.label.toLowerCase().includes(q),
      );

      return [...fileItems, ...cmdItems];
    }

    // No query: show recent files + all commands.
    const recentFiles: PaletteItem[] = (notesList?.notes ?? [])
      .slice(0, 8)
      .map((n) => ({
        key: `file:${n.path}`,
        label: n.title || n.path,
        description: n.path,
        icon: <FileMarkdownOutlined />,
        kind: "file",
        action: () => openTab(n.path, n.title || n.path),
      }));

    return [...commands, ...recentFiles];
  }, [query, searchResults, commands, notesList, openTab]);

  const pick = useCallback(
    (item: PaletteItem) => {
      item.action();
      setSearchOpen(false);
      setQuery("");
    },
    [setSearchOpen],
  );

  // Keyboard navigation.
  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelected((s) => Math.min(s + 1, items.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelected((s) => Math.max(s - 1, 0));
      } else if (e.key === "Enter" && items[selected]) {
        pick(items[selected]);
      }
    },
    [items, selected, pick],
  );

  // Auto-focus input on open.
  useEffect(() => {
    if (searchOpen) {
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [searchOpen]);

  return (
    <Modal
      open={searchOpen}
      onCancel={() => {
        setSearchOpen(false);
        setQuery("");
      }}
      footer={null}
      closable={false}
      width={isMobile ? "100vw" : 560}
      styles={{
        body: { padding: 0 },
        wrapper: {},
      }}
      className={`kenaz-palette${isMobile ? " kenaz-palette--mobile" : ""}`}
      style={isMobile ? { top: 0, margin: 0, maxWidth: "100vw", padding: 0 } : { top: 80 }}
    >
      {/* Search input */}
      <div
        style={{
          padding: "12px 16px",
          borderBottom: "1px solid #3a3a4e",
        }}
      >
        <Input
          ref={inputRef as never}
          prefix={
            <SearchOutlined style={{ color: "#6c7086", fontSize: 16 }} />
          }
          placeholder="Search notes or run a command…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={onKeyDown}
          allowClear
          size="large"
          variant="borderless"
          autoFocus
          style={{ fontSize: 15 }}
        />
      </div>

      {/* Results list */}
      <div style={{ maxHeight: isMobile ? "calc(100dvh - 120px)" : 380, overflow: "auto" }}>
        {items.length > 0 ? (
          <div style={{ padding: "4px 0" }}>
            {items.map((item, idx) => (
              <div
                key={item.key}
                onClick={() => pick(item)}
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 10,
                  padding: isMobile ? "12px 16px" : "8px 16px",
                  cursor: "pointer",
                  background:
                    idx === selected ? "#3a3a5e" : "transparent",
                  transition: "background 0.1s",
                }}
                onMouseEnter={() => setSelected(idx)}
              >
                <span
                  style={{
                    color: item.kind === "command" ? "#7c3aed" : "#6c7086",
                    fontSize: 16,
                    flexShrink: 0,
                    width: 20,
                    textAlign: "center",
                  }}
                >
                  {item.icon}
                </span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <Text
                    strong
                    style={{
                      color: "#cdd6f4",
                      fontSize: 14,
                      display: "block",
                    }}
                    ellipsis
                  >
                    {item.label}
                  </Text>
                  {item.description && (
                    <Text
                      type="secondary"
                      style={{ fontSize: 12, lineHeight: "16px" }}
                      ellipsis
                    >
                      {item.description}
                    </Text>
                  )}
                </div>
                {item.kind === "command" && (
                  <Text
                    type="secondary"
                    style={{
                      fontSize: 11,
                      flexShrink: 0,
                      background: "#1e1e2e",
                      padding: "1px 6px",
                      borderRadius: 4,
                    }}
                  >
                    command
                  </Text>
                )}
              </div>
            ))}
          </div>
        ) : query.trim() && !loading ? (
          <Empty
            description="No results"
            style={{ padding: 24 }}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        ) : null}
      </div>

      {/* Footer hint (desktop only) */}
      {!isMobile && (
        <div
          style={{
            padding: "6px 16px",
            borderTop: "1px solid #3a3a4e",
            display: "flex",
            gap: 16,
            fontSize: 11,
            color: "#6c7086",
          }}
        >
          <span>↑↓ navigate</span>
          <span>↵ select</span>
          <span>esc close</span>
        </div>
      )}
    </Modal>
  );
}
