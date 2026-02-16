import { useState, useEffect, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Spin, Typography, Tag, Divider, List, Button, Space, App } from "antd";
import { LinkOutlined, EditOutlined, EyeOutlined } from "@ant-design/icons";
import { getNote, updateNote, listNotes, type NoteDetail } from "../api/notes";
import { useUIStore } from "../store/ui";
import MarkdownEditor from "./MarkdownEditor";

const { Text, Paragraph } = Typography;

interface Props {
  path: string;
}

/** Renders a note in read mode or edit mode (CodeMirror). */
export default function NoteView({ path }: Props) {
  const { openTab } = useUIStore();
  const { message } = App.useApp();
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

  if (isLoading) return <Spin style={{ marginTop: 48 }} />;
  if (error || !note)
    return <Text type="danger">Failed to load {path}</Text>;

  // Simple wikilink rendering: replace [[target]] and [[target|alias]].
  const renderContent = (raw: string) => {
    const parts = raw.split(/(\[\[.*?\]\])/g);
    return parts.map((part, i) => {
      const m = part.match(/^\[\[(.*?)\]\]$/);
      if (!m) return <span key={i}>{part}</span>;
      const inner = m[1];
      const pipeIdx = inner.indexOf("|");
      const target = pipeIdx >= 0 ? inner.slice(0, pipeIdx) : inner;
      const label = pipeIdx >= 0 ? inner.slice(pipeIdx + 1) : inner;
      return (
        <span
          key={i}
          className="wikilink"
          onClick={() => openTab(target.trim(), label.trim())}
        >
          {label}
        </span>
      );
    });
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      {/* Toolbar */}
      <div
        style={{
          padding: "6px 24px",
          borderBottom: "1px solid #3a3a4e",
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          flexShrink: 0,
        }}
      >
        <Space size="small">
          <Text strong style={{ fontSize: 14 }}>
            {note.title || path}
          </Text>
          {note.tags.length > 0 &&
            note.tags.map((t) => (
              <Tag key={t} color="purple" style={{ margin: 0 }}>
                {t}
              </Tag>
            ))}
        </Space>
        <Space size="small">
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
          <div style={{ padding: "16px 24px", maxWidth: 800 }}>
            <Paragraph style={{ whiteSpace: "pre-wrap", lineHeight: 1.8 }}>
              {renderContent(note.content)}
            </Paragraph>

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
                      <Text style={{ color: "#7c3aed" }}>{bl}</Text>
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
