import { useState, useCallback, useEffect, useRef } from "react";
import { Modal, Input, List, Typography, Empty } from "antd";
import { SearchOutlined } from "@ant-design/icons";
import { searchNotes, type SearchResult } from "../api/notes";
import { useUIStore } from "../store/ui";

const { Text } = Typography;

/** Global quick-search modal, triggered by Cmd/Ctrl+K. */
export default function SearchModal() {
  const { searchOpen, setSearchOpen, openTab } = useUIStore();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  // Debounced search.
  useEffect(() => {
    if (!query.trim()) {
      setResults([]);
      return;
    }
    setLoading(true);
    const timer = setTimeout(async () => {
      try {
        const r = await searchNotes(query);
        setResults(r ?? []);
        setSelected(0);
      } catch {
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 200);
    return () => clearTimeout(timer);
  }, [query]);

  const pick = useCallback(
    (r: SearchResult) => {
      openTab(r.path, r.title || r.path);
      setSearchOpen(false);
      setQuery("");
    },
    [openTab, setSearchOpen],
  );

  // Keyboard navigation.
  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelected((s) => Math.min(s + 1, results.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelected((s) => Math.max(s - 1, 0));
      } else if (e.key === "Enter" && results[selected]) {
        pick(results[selected]);
      }
    },
    [results, selected, pick],
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
      width={560}
      styles={{ body: { padding: 0 } }}
      style={{ top: 80 }}
    >
      <div style={{ padding: "12px 16px" }}>
        <Input
          ref={inputRef as never}
          prefix={<SearchOutlined />}
          placeholder="Search notes…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={onKeyDown}
          allowClear
          size="large"
          variant="borderless"
          autoFocus
        />
      </div>
      <div style={{ maxHeight: 360, overflow: "auto" }}>
        {results.length > 0 ? (
          <List
            size="small"
            loading={loading}
            dataSource={results}
            renderItem={(item, idx) => (
              <List.Item
                onClick={() => pick(item)}
                style={{
                  cursor: "pointer",
                  padding: "8px 16px",
                  background: idx === selected ? "#2a2a3c" : "transparent",
                }}
              >
                <List.Item.Meta
                  title={
                    <Text strong style={{ color: "#cdd6f4" }}>
                      {item.title || item.path}
                    </Text>
                  }
                  description={
                    <Text
                      type="secondary"
                      style={{ fontSize: 12 }}
                      ellipsis
                    >
                      {item.snippet}
                    </Text>
                  }
                />
              </List.Item>
            )}
          />
        ) : query.trim() && !loading ? (
          <Empty
            description="No results"
            style={{ padding: 24 }}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        ) : null}
      </div>
    </Modal>
  );
}
