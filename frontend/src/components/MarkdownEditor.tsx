import { useRef, useEffect, useCallback } from "react";
import { EditorView, keymap, placeholder } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { markdown } from "@codemirror/lang-markdown";
import { languages } from "@codemirror/language-data";
import { oneDark } from "@codemirror/theme-one-dark";
import { defaultKeymap, indentWithTab } from "@codemirror/commands";

interface Props {
  value: string;
  onChange: (value: string) => void;
  onSave?: () => void;
  autoFocus?: boolean;
}

/**
 * CodeMirror 6 Markdown editor wrapped as a React component.
 * Calls onChange on every keystroke and onSave on Cmd/Ctrl+S.
 */
export default function MarkdownEditor({
  value,
  onChange,
  onSave,
  autoFocus,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  // Keep callback refs stable to avoid recreating the editor.
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onSaveRef = useRef(onSave);
  onSaveRef.current = onSave;

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

    const state = EditorState.create({
      doc: value,
      extensions: [
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
        }),
      ],
    });

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
  }, []); // mount once

  // Sync external value changes (e.g. after save returns updated content).
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

  return (
    <div
      ref={containerRef}
      style={{ height: "100%", overflow: "auto" }}
    />
  );
}
