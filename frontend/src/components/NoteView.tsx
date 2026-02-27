import { useState, useEffect, useCallback, useMemo } from "react";
import type { MouseEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Spin, Typography, Tag, Divider, List, Button, Space, App } from "antd";
import { LinkOutlined, EditOutlined, EyeOutlined, DownloadOutlined } from "@ant-design/icons";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { getNote, updateNote, listNotes, type NoteDetail } from "../api/notes";
import { useUIStore } from "../store/ui";
import { useIsMobile } from "../hooks/useIsMobile";
import { slugify, extractText } from "../utils/slugify";
import { scrollToHeading } from "../utils/scrollToHeading";
import MarkdownEditor from "./MarkdownEditor";
import { c } from "../styles/colors";

const { Text } = Typography;

interface Props {
  path: string;
}

/** Renders a note in read mode or edit mode (CodeMirror). */
export default function NoteView({ path }: Props) {
  const { openTab } = useUIStore();
  const { message } = App.useApp();
  const isMobile = useIsMobile();
  const queryClient = useQueryClient();

  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");

  // Notes list for wikilink autocomplete.
  const { data: notesList } = useQuery({
    queryKey: ["notes"],
    queryFn: () => listNotes({ limit: 1000 }),
  });
  const getNotePaths = useCallback(
    () =>
      (notesList?.notes ?? []).map((n) => ({
        path: n.path,
        title: n.title,
      })),
    [notesList],
  );

  const {
    data: note,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["note", path],
    queryFn: () => getNote(path),
    enabled: !!path,
  });

  // Seed draft whenever note data arrives and we're not editing.
  useEffect(() => {
    if (note && !editing) {
      setDraft(note.content);
    }
  }, [note, editing]);

  // Save mutation with optimistic update.
  const saveMutation = useMutation({
    mutationFn: () => updateNote(path, draft, note?.checksum),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ["note", path] });
      const prev = queryClient.getQueryData<NoteDetail>(["note", path]);
      if (prev) {
        queryClient.setQueryData<NoteDetail>(["note", path], {
          ...prev,
          content: draft,
        });
      }
      return { prev };
    },
    onError: (_err, _vars, context) => {
      if (context?.prev) {
        queryClient.setQueryData(["note", path], context.prev);
      }
      message.error("Save failed — check for conflicts");
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(["note", path], saved);
      queryClient.invalidateQueries({ queryKey: ["notes"] });
      message.success("Saved");
    },
  });

  const handleSave = useCallback(() => {
    if (!saveMutation.isPending) {
      saveMutation.mutate();
    }
  }, [saveMutation]);

  const enterEdit = useCallback(() => {
    if (note) setDraft(note.content);
    setEditing(true);
  }, [note]);

  const exitEdit = useCallback(() => {
    setEditing(false);
  }, []);

  const handleDownload = useCallback(() => {
    if (!note) return;
    const blob = new Blob([note.content], { type: "text/markdown;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = path.split("/").pop() || "note.md";
    a.click();
    URL.revokeObjectURL(url);
  }, [note, path]);

  // Keyboard shortcuts: Cmd/Ctrl+E toggle edit, Cmd/Ctrl+S save.
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;
      if (e.key === "e") {
        e.preventDefault();
        if (editing) exitEdit();
        else enterEdit();
      }
      if (e.key === "s" && editing) {
        e.preventDefault();
        handleSave();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [editing, enterEdit, exitEdit, handleSave]);

  const markdownContent = useMemo(() => {
    const raw = note?.content ?? "";
    // Strip YAML frontmatter in preview mode.
    const withoutFrontmatter = raw.replace(/^---\n[\s\S]*?\n---\n?/, "");

    // Convert wikilinks to standard markdown links so the renderer creates anchors.
    // [[target]] -> [target](target)
    // [[target|label]] -> [label](target)
    return withoutFrontmatter.replace(/\[\[(.*?)\]\]/g, (_, inner: string) => {
      const pipeIdx = inner.indexOf("|");
      const target = (pipeIdx >= 0 ? inner.slice(0, pipeIdx) : inner).trim();
      const label = (pipeIdx >= 0 ? inner.slice(pipeIdx + 1) : inner).trim();
      if (!target) return "";
      return `[${label || target}](${encodeURI(target)})`;
    });
  }, [note?.content]);

  const handlePreviewClickCapture = useCallback(
    (e: MouseEvent<HTMLDivElement>) => {
      const targetEl = e.target as HTMLElement | null;
      const anchor = targetEl?.closest("a") as HTMLAnchorElement | null;
      if (!anchor) return;

      const rawHref = String(anchor.getAttribute("href") ?? "").trim();
      if (!rawHref) return;

      if (rawHref.startsWith("#")) {
        e.preventDefault();
        e.stopPropagation();
        const id = rawHref.slice(1);
        history.replaceState(null, "", rawHref);
        scrollToHeading(id);
        return;
      }

      const isExternal = /^(https?:|mailto:|tel:)/i.test(rawHref) || rawHref.startsWith("//");
      if (isExternal) return;

      e.preventDefault();
      e.stopPropagation();

      if (rawHref.startsWith("wikilink:")) {
        const target = decodeURIComponent(rawHref.replace("wikilink:", "")).trim();
        const pathCandidate = target.endsWith(".md") ? target : `${target}.md`;
        openTab(pathCandidate, target);
        return;
      }

      const cleaned = rawHref.replace(/^\//, "");
      const pathCandidate = cleaned.endsWith(".md") ? cleaned : `${cleaned}.md`;
      openTab(pathCandidate, cleaned);
    },
    [openTab],
  );

  if (isLoading) return <Spin style={{ marginTop: 48 }} />;
  if (error || !note)
    return (
      <div style={{ padding: 24, textAlign: "center", marginTop: 48 }}>
        <Text type="danger">Failed to load {path}</Text>
        <br />
        <Button
          size="small"
          style={{ marginTop: 8 }}
          onClick={() => useUIStore.getState().closeTab(path)}
        >
          Close tab
        </Button>
      </div>
    );


  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      {/* Toolbar */}
      <div
        style={{
          padding: isMobile ? "6px 12px" : "6px 24px",
          borderBottom: `1px solid ${c.border}`,
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          flexShrink: 0,
          gap: 8,
          flexWrap: "wrap",
        }}
      >
        <div style={{ flex: 1, minWidth: 0, display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" }}>
          <Text strong ellipsis style={{ fontSize: isMobile ? 13 : 14, maxWidth: isMobile ? "60vw" : undefined }}>
            {note.title || path}
          </Text>
          {note.tags.length > 0 &&
            note.tags.map((t) => (
              <Tag key={t} color="cyan" style={{ margin: 0, fontSize: 11 }}>
                {t}
              </Tag>
            ))}
        </div>
        <Space size="small" style={{ flexShrink: 0 }}>
          {editing && (
            <Button
              size="small"
              type="primary"
              loading={saveMutation.isPending}
              onClick={handleSave}
            >
              Save
            </Button>
          )}
          <Button
            size="small"
            icon={<DownloadOutlined />}
            onClick={handleDownload}
          />
          <Button
            size="small"
            icon={editing ? <EyeOutlined /> : <EditOutlined />}
            onClick={editing ? exitEdit : enterEdit}
          >
            {editing ? "Read" : "Edit"}
          </Button>
        </Space>
      </div>

      {/* Content area */}
      <div style={{ flex: 1, overflow: "auto" }}>
        {editing ? (
          <MarkdownEditor
            value={draft}
            onChange={setDraft}
            onSave={handleSave}
            autoFocus
            notePaths={getNotePaths}
          />
        ) : (
          <div
            className="md-preview"
            style={{
              padding: isMobile ? "12px 16px" : "16px 24px",
              maxWidth: isMobile ? undefined : 800,
            }}
            onClickCapture={handlePreviewClickCapture}
          >
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                h1: ({ children }) => <h1 id={slugify(extractText(children))}>{children}</h1>,
                h2: ({ children }) => <h2 id={slugify(extractText(children))}>{children}</h2>,
                h3: ({ children }) => <h3 id={slugify(extractText(children))}>{children}</h3>,
                h4: ({ children }) => <h4 id={slugify(extractText(children))}>{children}</h4>,
                h5: ({ children }) => <h5 id={slugify(extractText(children))}>{children}</h5>,
                h6: ({ children }) => <h6 id={slugify(extractText(children))}>{children}</h6>,
                a: ({ href, children }) => {
                  const rawHref = String(href ?? "").trim();

                  const renderInternal = (targetRaw: string, labelRaw?: string) => {
                    const target = targetRaw.trim();
                    if (!target) return <>{children}</>;
                    const pathCandidate = target.endsWith(".md") ? target : `${target}.md`;
                    const title = (labelRaw ?? target).trim();
                    return (
                      <span
                        role="link"
                        tabIndex={0}
                        className="wikilink"
                        onClick={() => openTab(pathCandidate, title)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter" || e.key === " ") {
                            e.preventDefault();
                            openTab(pathCandidate, title);
                          }
                        }}
                      >
                        {children}
                      </span>
                    );
                  };

                  if (rawHref.startsWith("wikilink:")) {
                    const target = decodeURIComponent(rawHref.replace("wikilink:", ""));
                    return renderInternal(target);
                  }

                  if (rawHref.startsWith("#")) {
                    const anchorId = rawHref.slice(1);
                    return (
                      <a
                        href={rawHref}
                        onClick={(e) => {
                          e.preventDefault();
                          history.replaceState(null, "", rawHref);
                          scrollToHeading(anchorId);
                        }}
                      >
                        {children}
                      </a>
                    );
                  }

                  try {
                    const resolved = new URL(rawHref, window.location.origin);
                    const isHttp = resolved.protocol === "http:" || resolved.protocol === "https:";
                    if (isHttp && resolved.origin === window.location.origin) {
                      const path = resolved.pathname.replace(/^\/+/, "");
                      const isKnownExternalPath =
                        path.startsWith("api/") || path.startsWith("attachments/") || path === "";
                      if (!isKnownExternalPath) {
                        return renderInternal(path);
                      }
                    }
                  } catch {
                    // Non-URL-ish href; fallback below.
                  }

                  if (!/^(https?:|mailto:|tel:)/i.test(rawHref) && !rawHref.startsWith("//")) {
                    return renderInternal(rawHref.replace(/^\//, ""));
                  }

                  return (
                    <a href={rawHref} target="_blank" rel="noreferrer">
                      {children}
                    </a>
                  );
                },
              }}
            >
              {markdownContent}
            </ReactMarkdown>

            {note.backlinks.length > 0 && (
              <>
                <Divider orientationMargin={0} plain>
                  <LinkOutlined /> Backlinks
                </Divider>
                <List
                  size="small"
                  dataSource={note.backlinks}
                  renderItem={(bl) => (
                    <List.Item
                      style={{ cursor: "pointer", padding: "4px 0" }}
                      onClick={() => openTab(bl, bl)}
                    >
                      <Text style={{ color: c.accent }}>{bl}</Text>
                    </List.Item>
                  )}
                />
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
