import { useState, useEffect, useCallback, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Input, Tree, Typography, Spin, Space, Button, App } from "antd";
import {
  FileMarkdownOutlined,
  FolderOutlined,
  SearchOutlined,
  PlusOutlined,
  ApartmentOutlined,
  DownloadOutlined,
  EditOutlined,
} from "@ant-design/icons";
import type { DataNode } from "antd/es/tree";
import { listNotes, getNote, renameNote, type NoteListItem } from "../api/notes";
import { downloadDirectory } from "../utils/downloadNote";
import { useUIStore } from "../store/ui";
import { useIsMobile } from "../hooks/useIsMobile";
import CreateNoteModal from "./CreateNoteModal";

const { Search } = Input;
const { Text } = Typography;

/** Sort tree nodes: folders first, then alphabetical (case-insensitive). */
function sortTree(nodes: DataNode[]): DataNode[] {
  return nodes
    .sort((a, b) => {
      const aIsFolder = !a.isLeaf;
      const bIsFolder = !b.isLeaf;
      if (aIsFolder !== bIsFolder) return aIsFolder ? -1 : 1;
      return String(a.title).localeCompare(String(b.title), undefined, {
        sensitivity: "base",
      });
    })
    .map((node) =>
      node.children ? { ...node, children: sortTree(node.children) } : node,
    );
}

/** Build a tree structure from flat note paths. */
function buildTree(notes: NoteListItem[]): DataNode[] {
  const root: Record<string, DataNode> = {};

  for (const note of notes) {
    const parts = note.path.split("/");
    const current = root;

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
  return sortTree(Array.from(topLevel).map((k) => root[k]));
}

export default function Sidebar() {
  const { openTab, renameTab, setSearchOpen, setMobileDrawer } = useUIStore();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();
  const [createOpen, setCreateOpen] = useState(false);
  const [createFolderPath, setCreateFolderPath] = useState<string>("");
  const [downloadingFolder, setDownloadingFolder] = useState<string | null>(null);
  const [renamingKey, setRenamingKey] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const renameInputRef = useRef<HTMLInputElement>(null);

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

  const startRename = useCallback((key: string, isFolder: boolean) => {
    setRenamingKey(key);
    // Extract just the name part (last segment, without .md for files).
    const parts = key.split("/");
    const name = parts[parts.length - 1];
    setRenameValue(isFolder ? name : name.replace(/\.md$/, ""));
    // Focus input after render.
    setTimeout(() => {
      renameInputRef.current?.focus();
      renameInputRef.current?.select();
    }, 0);
  }, []);

  const confirmRename = useCallback(async () => {
    if (!renamingKey || !renameValue.trim()) {
      setRenamingKey(null);
      return;
    }

    const key = renamingKey;
    const isFolder = !(key.endsWith(".md"));
    const parts = key.split("/");
    const parentDir = parts.slice(0, -1).join("/");
    const prefix = parentDir ? `${parentDir}/` : "";

    let oldPath: string;
    let newPath: string;

    if (isFolder) {
      oldPath = `${key}/`;
      newPath = `${prefix}${renameValue.trim()}/`;
    } else {
      oldPath = key;
      const newName = renameValue.trim().endsWith(".md")
        ? renameValue.trim()
        : `${renameValue.trim()}.md`;
      newPath = `${prefix}${newName}`;
    }

    if (oldPath === newPath) {
      setRenamingKey(null);
      return;
    }

    try {
      await renameNote(oldPath, newPath);
      message.success("Renamed");
      queryClient.invalidateQueries({ queryKey: ["notes"] });
      if (!isFolder) {
        renameTab(oldPath, newPath);
        queryClient.invalidateQueries({ queryKey: ["note", oldPath] });
      } else {
        // For directory renames, update all open tabs with old prefix.
        const tabs = useUIStore.getState().tabs;
        for (const tab of tabs) {
          if (tab.path.startsWith(oldPath)) {
            const newTabPath = newPath.slice(0, -1) + "/" + tab.path.slice(oldPath.length);
            renameTab(tab.path, newTabPath);
            queryClient.invalidateQueries({ queryKey: ["note", tab.path] });
          }
        }
      }
    } catch (err) {
      message.error(err instanceof Error ? err.message : "Rename failed");
    } finally {
      setRenamingKey(null);
    }
  }, [renamingKey, renameValue, message, queryClient, renameTab]);

  const cancelRename = useCallback(() => {
    setRenamingKey(null);
  }, []);

  const treeData = data ? buildTree(data.notes ?? []) : [];

  return (
    <div
      className="kenaz-sidebar"
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
            gap: 8,
            alignItems: "center",
            minWidth: 0,
          }}
        >
          <Search
            placeholder="Search notes…"
            prefix={<SearchOutlined />}
            onFocus={() => setSearchOpen(true)}
            allowClear
            size="middle"
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
            blockNode
            className="kenaz-sidebar-tree"
            titleRender={(node) => {
              const isFolder = !node.isLeaf;
              const key = node.key as string;
              const isDownloading = downloadingFolder === key;
              const isRenaming = renamingKey === key;

              if (isRenaming) {
                return (
                  <input
                    ref={renameInputRef}
                    value={renameValue}
                    onChange={(e) => setRenameValue(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        e.preventDefault();
                        confirmRename();
                      } else if (e.key === "Escape") {
                        e.preventDefault();
                        cancelRename();
                      }
                      e.stopPropagation();
                    }}
                    onBlur={() => confirmRename()}
                    onClick={(e) => e.stopPropagation()}
                    style={{
                      width: "100%",
                      border: "1px solid var(--ant-color-primary, #1677ff)",
                      borderRadius: 4,
                      padding: "1px 6px",
                      fontSize: "inherit",
                      lineHeight: "inherit",
                      outline: "none",
                      background: "var(--ant-color-bg-container, #fff)",
                      color: "var(--ant-color-text, #000)",
                    }}
                  />
                );
              }

              return (
                <span
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 6,
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
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={(e) => {
                      e.stopPropagation();
                      startRename(key, isFolder);
                    }}
                    title="Rename"
                    className="kenaz-tree-action"
                    style={{ paddingInline: 4, minWidth: 0 }}
                  />
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
                        className="kenaz-tree-action"
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
                        className="kenaz-tree-action"
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
