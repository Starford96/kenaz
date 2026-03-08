import { useState, useEffect, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { Input, Tree, Typography, Spin, Space, Button, App } from "antd";
import {
  FileMarkdownOutlined,
  FolderOutlined,
  SearchOutlined,
  PlusOutlined,
  ApartmentOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import type { DataNode } from "antd/es/tree";
import { listNotes, getNote, type NoteListItem } from "../api/notes";
import { downloadDirectory } from "../utils/downloadNote";
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
  const { message } = App.useApp();
  const isMobile = useIsMobile();
  const [createOpen, setCreateOpen] = useState(false);
  const [createFolderPath, setCreateFolderPath] = useState<string>("");
  const [downloadingFolder, setDownloadingFolder] = useState<string | null>(null);

  const openCreateModal = useCallback((folderPath?: string) => {
    setCreateFolderPath(folderPath ?? "");
    setCreateOpen(true);
  }, []);

  // Listen for create-note command from palette.
  useEffect(() => {
    const handler = () => openCreateModal();
    window.addEventListener("kenaz:create-note", handler);
    return () => window.removeEventListener("kenaz:create-note", handler);
  }, [openCreateModal]);

  const { data, isLoading } = useQuery({
    queryKey: ["notes"],
    queryFn: () => listNotes({ limit: 1000 }),
  });

  const handleDownloadFolder = useCallback(
    async (folderKey: string) => {
      const notes = data?.notes ?? [];
      const prefix = folderKey.endsWith("/") ? folderKey : `${folderKey}/`;
      const notesInFolder = notes.filter((n) => n.path.startsWith(prefix));
      if (notesInFolder.length === 0) {
        message.warning("No notes in this folder");
        return;
      }
      setDownloadingFolder(folderKey);
      try {
        await downloadDirectory(
          folderKey,
          notesInFolder,
          async (path) => (await getNote(path)).content,
        );
      } catch (err) {
        message.error("Download failed");
        console.error(err);
      } finally {
        setDownloadingFolder(null);
      }
    },
    [data?.notes, message],
  );

  const treeData = data ? buildTree(data.notes ?? []) : [];

  return (
    <div
      style={{
        padding: isMobile ? 14 : 12,
        height: "100%",
        overflow: "auto",
        minWidth: 0,
      }}
    >
      <Space direction="vertical" style={{ width: "100%", minWidth: 0 }} size="small">
        <div
          style={{
            display: "flex",
            gap: 4,
            alignItems: "center",
            minWidth: 0,
          }}
        >
          <Search
            placeholder="Search notes…"
            prefix={<SearchOutlined />}
            onFocus={() => setSearchOpen(true)}
            allowClear
            size="small"
            style={{ flex: 1, minWidth: 0 }}
          />
          <Button
            type="text"
            size="small"
            icon={<ApartmentOutlined />}
            onClick={() => {
              openTab("__graph__", "Graph");
              if (isMobile) setMobileDrawer(null);
            }}
            title="Knowledge graph"
            style={{ flexShrink: 0 }}
          />
          <Button
            type="text"
            size="small"
            icon={<PlusOutlined />}
            onClick={() => openCreateModal()}
            title="New note"
            style={{ flexShrink: 0 }}
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
            titleRender={(node) => {
              const isFolder = !node.isLeaf;
              const key = node.key as string;
              const isDownloading = downloadingFolder === key;
              return (
                <span
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 2,
                    width: "100%",
                  }}
                >
                  <span
                    style={{
                      flex: 1,
                      minWidth: 0,
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {typeof node.title === "function"
                      ? node.title(node)
                      : node.title}
                  </span>
                  {isFolder && (
                    <>
                      <Button
                        type="text"
                        size="small"
                        icon={<PlusOutlined />}
                        onClick={(e) => {
                          e.stopPropagation();
                          openCreateModal(`${key}/`);
                        }}
                        title="Create note in folder"
                        style={{ paddingInline: 4, minWidth: 0 }}
                      />
                      <Button
                        type="text"
                        size="small"
                        icon={<DownloadOutlined />}
                        loading={isDownloading}
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDownloadFolder(key);
                        }}
                        title="Download folder"
                        style={{ paddingInline: 4, minWidth: 0 }}
                      />
                    </>
                  )}
                </span>
              );
            }}
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
      <CreateNoteModal
        open={createOpen}
        onClose={() => {
          setCreateOpen(false);
          setCreateFolderPath("");
        }}
        initialPath={createFolderPath}
      />
    </div>
  );
}
