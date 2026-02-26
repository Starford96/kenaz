import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { Input, Tree, Typography, Spin, Space, Button } from "antd";
import {
  FileMarkdownOutlined,
  FolderOutlined,
  SearchOutlined,
  PlusOutlined,
  ApartmentOutlined,
} from "@ant-design/icons";
import type { DataNode } from "antd/es/tree";
import { listNotes, type NoteListItem } from "../api/notes";
import { useUIStore } from "../store/ui";
import { useIsMobile } from "../hooks/useIsMobile";
import CreateNoteModal from "./CreateNoteModal";

const { Search } = Input;
const { Text } = Typography;

/** Build a tree structure from flat note paths. */
function buildTree(notes: NoteListItem[]): DataNode[] {
  const root: Record<string, DataNode> = {};

  for (const note of notes) {
    const parts = note.path.split("/");
    let current = root;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const isLast = i === parts.length - 1;
      const key = parts.slice(0, i + 1).join("/");

      if (!current[key]) {
        current[key] = {
          key,
          title: part,
          icon: isLast ? <FileMarkdownOutlined /> : <FolderOutlined />,
          children: isLast ? undefined : [],
          isLeaf: isLast,
        };
        // Attach to parent.
        if (i > 0) {
          const parentKey = parts.slice(0, i).join("/");
          const parent = current[parentKey];
          if (parent && parent.children) {
            parent.children.push(current[key]);
          }
        }
      }
    }
  }

  // Return only top-level nodes.
  const topLevel = new Set(
    (Object.keys(root) as string[]).filter((k) => !k.includes("/")),
  );
  return Array.from(topLevel).map((k) => root[k]);
}

export default function Sidebar() {
  const { openTab, setSearchOpen, setMobileDrawer } = useUIStore();
  const isMobile = useIsMobile();
  const [createOpen, setCreateOpen] = useState(false);

  // Listen for create-note command from palette.
  useEffect(() => {
    const handler = () => setCreateOpen(true);
    window.addEventListener("kenaz:create-note", handler);
    return () => window.removeEventListener("kenaz:create-note", handler);
  }, []);

  const { data, isLoading } = useQuery({
    queryKey: ["notes"],
    queryFn: () => listNotes({ limit: 1000 }),
  });

  const treeData = data ? buildTree(data.notes ?? []) : [];

  return (
    <div style={{ padding: 12, height: "100%", overflow: "auto" }}>
      <Space direction="vertical" style={{ width: "100%" }} size="small">
        <div style={{ display: "flex", gap: 4 }}>
          <Search
            placeholder="Search notes…"
            prefix={<SearchOutlined />}
            onFocus={() => setSearchOpen(true)}
            allowClear
            size="small"
            style={{ flex: 1 }}
          />
          <Button
            size="small"
            icon={<ApartmentOutlined />}
            onClick={() => {
              openTab("__graph__", "Graph");
              if (isMobile) setMobileDrawer(null);
            }}
            title="Knowledge graph"
          />
          <Button
            size="small"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
            title="New note"
          />
        </div>

        {isLoading ? (
          <Spin size="small" style={{ display: "block", marginTop: 24 }} />
        ) : treeData.length === 0 ? (
          <Text type="secondary" style={{ display: "block", marginTop: 16 }}>
            No notes yet
          </Text>
        ) : (
          <Tree
            showIcon
            treeData={treeData}
            onSelect={(_, info) => {
              const node = info.node;
              if (node.isLeaf) {
                openTab(node.key as string, node.title as string);
                if (isMobile) setMobileDrawer(null);
              }
            }}
            style={{ background: "transparent" }}
          />
        )}
      </Space>
      <CreateNoteModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </div>
  );
}
