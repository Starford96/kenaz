import { useState, useEffect } from "react";
import { Modal, Input, App } from "antd";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createNote } from "../api/notes";
import { useUIStore } from "../store/ui";

interface Props {
  open: boolean;
  onClose: () => void;
  /** Pre-fill path (e.g. folder prefix like "research/"). */
  initialPath?: string;
  /** Pre-fill content (e.g. pasted text). */
  initialContent?: string;
}

/** Modal for creating a new note with a path and optional content. */
export default function CreateNoteModal({
  open,
  onClose,
  initialPath = "",
  initialContent = "",
}: Props) {
  const [path, setPath] = useState("");
  const [content, setContent] = useState("");
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { openTab } = useUIStore();

  useEffect(() => {
    if (open) {
      setPath(initialPath);
      setContent(initialContent);
    }
  }, [open, initialPath, initialContent]);

  const mutation = useMutation({
    mutationFn: () => {
      const body =
        content.trim() || `# ${titleFromPath(path)}\n`;
      return createNote(ensureMd(path), body);
    },
    onSuccess: (note) => {
      queryClient.invalidateQueries({ queryKey: ["notes"] });
      openTab(note.path, note.title || note.path);
      message.success(`Created ${note.path}`);
      setPath("");
      setContent("");
      onClose();
    },
    onError: (err: Error) => {
      message.error(err.message || "Failed to create note");
    },
  });

  const handleOk = () => {
    const trimmed = path.trim();
    if (!trimmed) return;
    if (trimmed.endsWith("/")) {
      message.warning("Add a filename (e.g. my-note)");
      return;
    }
    mutation.mutate();
  };

  const handleCancel = () => {
    setPath("");
    setContent("");
    onClose();
  };

  return (
    <Modal
      title="New Note"
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText="Create"
      confirmLoading={mutation.isPending}
      destroyOnClose
      width={480}
    >
      <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        <Input
          placeholder="e.g. projects/my-idea or research/new-note"
          value={path}
          onChange={(e) => setPath(e.target.value)}
          onPressEnter={(e) => {
            if (!e.shiftKey) handleOk();
          }}
          autoFocus
        />
        <Input.TextArea
          placeholder="Paste or type content… (optional)"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          autoSize={{ minRows: 4, maxRows: 12 }}
          style={{ fontFamily: "inherit" }}
        />
      </div>
    </Modal>
  );
}

function ensureMd(p: string): string {
  const trimmed = p.trim();
  return trimmed.endsWith(".md") ? trimmed : `${trimmed}.md`;
}

function titleFromPath(p: string): string {
  const base = p.split("/").pop() ?? p;
  return base.replace(/\.md$/, "").replace(/[-_]/g, " ");
}
