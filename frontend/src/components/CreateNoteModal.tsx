import { useState } from "react";
import { Modal, Input, App } from "antd";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createNote } from "../api/notes";
import { useUIStore } from "../store/ui";

interface Props {
  open: boolean;
  onClose: () => void;
}

/** Modal for creating a new note with a path and initial content. */
export default function CreateNoteModal({ open, onClose }: Props) {
  const [path, setPath] = useState("");
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { openTab } = useUIStore();

  const mutation = useMutation({
    mutationFn: () => createNote(ensureMd(path), `# ${titleFromPath(path)}\n`),
    onSuccess: (note) => {
      queryClient.invalidateQueries({ queryKey: ["notes"] });
      openTab(note.path, note.title || note.path);
      message.success(`Created ${note.path}`);
      setPath("");
      onClose();
    },
    onError: (err: Error) => {
      message.error(err.message || "Failed to create note");
    },
  });

  const handleOk = () => {
    if (!path.trim()) return;
    mutation.mutate();
  };

  return (
    <Modal
      title="New Note"
      open={open}
      onOk={handleOk}
      onCancel={() => {
        setPath("");
        onClose();
      }}
      okText="Create"
      confirmLoading={mutation.isPending}
      destroyOnClose
    >
      <Input
        placeholder="e.g. projects/my-idea.md"
        value={path}
        onChange={(e) => setPath(e.target.value)}
        onPressEnter={handleOk}
        autoFocus
      />
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
