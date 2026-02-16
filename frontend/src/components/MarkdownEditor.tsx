import { useRef, useEffect, useCallback } from "react";
import { EditorView, keymap, placeholder } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { markdown } from "@codemirror/lang-markdown";
import { languages } from "@codemirror/language-data";
import { oneDark } from "@codemirror/theme-one-dark";
import { defaultKeymap, indentWithTab } from "@codemirror/commands";
import { wikilinkCompletion } from "./wikilinkComplete";
import { uploadAttachment } from "../api/notes";

interface Props {
  value: string;
  onChange: (value: string) => void;
  onSave?: () => void;
  autoFocus?: boolean;
  /** Called to get note list for wikilink autocomplete. */
  notePaths?: () => { path: string; title: string }[];
}

/**
 * CodeMirror 6 Markdown editor with wikilink autocomplete and attachment paste/drag.
 */
export default function MarkdownEditor({
  value,
  onChange,
  onSave,
  autoFocus,
  notePaths,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onSaveRef = useRef(onSave);
  onSaveRef.current = onSave;
  const notePathsRef = useRef(notePaths);
  notePathsRef.current = notePaths;

  const saveKeymap = useCallback(
    () =>
      keymap.of([
        {
          key: "Mod-s",
          run: () => {
            onSaveRef.current?.();
            return true;
          },
        },
      ]),
    [],
  );

  // Create editor once.
  useEffect(() => {
    if (!containerRef.current) return;

    const extensions = [
      markdown({ codeLanguages: languages }),
      oneDark,
      EditorView.lineWrapping,
      placeholder("Start writing…"),
      saveKeymap(),
      keymap.of([indentWithTab, ...defaultKeymap]),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          onChangeRef.current(update.state.doc.toString());
        }
      }),
      // Wikilink autocomplete.
      wikilinkCompletion(() => notePathsRef.current?.() ?? []),
      // Theme overrides for Obsidian-like look.
      EditorView.theme({
        "&": {
          fontSize: "15px",
          height: "100%",
        },
        ".cm-content": {
          fontFamily:
            "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
          padding: "16px 0",
        },
        ".cm-gutters": {
          background: "#1e1e2e",
          borderRight: "1px solid #3a3a4e",
        },
        ".cm-activeLineGutter": {
          background: "#252536",
        },
        ".cm-activeLine": {
          background: "#252536",
        },
        "&.cm-focused .cm-cursor": {
          borderLeftColor: "#7c3aed",
        },
        "&.cm-focused .cm-selectionBackground, ::selection": {
          background: "#3a3a5e !important",
        },
        // Autocomplete tooltip styling.
        ".cm-tooltip.cm-tooltip-autocomplete": {
          background: "#2a2a3c",
          border: "1px solid #3a3a4e",
        },
        ".cm-tooltip.cm-tooltip-autocomplete > ul > li": {
          color: "#cdd6f4",
        },
        ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]": {
          background: "#3a3a5e",
          color: "#cdd6f4",
        },
      }),
    ];

    const state = EditorState.create({ doc: value, extensions });
    const view = new EditorView({ state, parent: containerRef.current });
    viewRef.current = view;

    if (autoFocus) {
      requestAnimationFrame(() => view.focus());
    }

    return () => {
      view.destroy();
      viewRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync external value changes.
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const current = view.state.doc.toString();
    if (current !== value) {
      view.dispatch({
        changes: { from: 0, to: current.length, insert: value },
      });
    }
  }, [value]);

  // Attachment paste/drop handler.
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const insertAtCursor = (view: EditorView, text: string) => {
      const pos = view.state.selection.main.head;
      view.dispatch({
        changes: { from: pos, to: pos, insert: text },
        selection: { anchor: pos + text.length },
      });
    };

    const handleFiles = async (files: FileList) => {
      const view = viewRef.current;
      if (!view) return;

      for (const file of Array.from(files)) {
        if (!file.type.startsWith("image/")) continue;
        try {
          // Insert placeholder while uploading.
          const placeholder = `![Uploading ${file.name}…]()\n`;
          insertAtCursor(view, placeholder);

          const result = await uploadAttachment(file);
          // Replace placeholder with actual link.
          const doc = view.state.doc.toString();
          const idx = doc.indexOf(placeholder);
          if (idx >= 0) {
            const link = `![${file.name}](${result.url})\n`;
            view.dispatch({
              changes: { from: idx, to: idx + placeholder.length, insert: link },
            });
          }
        } catch {
          // Remove placeholder on failure.
          const doc = view.state.doc.toString();
          const ph = `![Uploading ${file.name}…]()\n`;
          const idx = doc.indexOf(ph);
          if (idx >= 0) {
            view.dispatch({
              changes: { from: idx, to: idx + ph.length, insert: "" },
            });
          }
        }
      }
    };

    const onPaste = (e: ClipboardEvent) => {
      if (e.clipboardData?.files?.length) {
        e.preventDefault();
        handleFiles(e.clipboardData.files);
      }
    };

    const onDrop = (e: DragEvent) => {
      if (e.dataTransfer?.files?.length) {
        e.preventDefault();
        handleFiles(e.dataTransfer.files);
      }
    };

    const onDragOver = (e: DragEvent) => e.preventDefault();

    container.addEventListener("paste", onPaste);
    container.addEventListener("drop", onDrop);
    container.addEventListener("dragover", onDragOver);
    return () => {
      container.removeEventListener("paste", onPaste);
      container.removeEventListener("drop", onDrop);
      container.removeEventListener("dragover", onDragOver);
    };
  }, []);

  return (
    <div ref={containerRef} style={{ height: "100%", overflow: "auto" }} />
  );
}
