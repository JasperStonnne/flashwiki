import { useRef, useEffect, useCallback } from "react";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers, highlightActiveLine } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { markdown } from "@codemirror/lang-markdown";
import { syntaxHighlighting, defaultHighlightStyle } from "@codemirror/language";
import styles from "./SourceEditor.module.css";

interface SourceEditorProps {
  value: string;
  onChange: (value: string) => void;
  className?: string;
}

export function SourceEditor({ value, onChange, className }: SourceEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const isExternalUpdate = useRef(false);

  // Store onChange in a ref to avoid recreating the editor when it changes
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  // Initialize CodeMirror
  useEffect(() => {
    if (!containerRef.current) return;

    const state = EditorState.create({
      doc: value,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        markdown(),
        syntaxHighlighting(defaultHighlightStyle),
        EditorView.updateListener.of((update) => {
          if (update.docChanged && !isExternalUpdate.current) {
            onChangeRef.current(update.state.doc.toString());
          }
        }),
        EditorView.theme({
          "&": {
            height: "100%",
            fontSize: "14px",
          },
          ".cm-scroller": {
            fontFamily: "var(--font-mono)",
            lineHeight: "1.6",
          },
          ".cm-content": {
            padding: "var(--space-4)",
          },
          ".cm-gutters": {
            backgroundColor: "var(--color-bg-subtle)",
            borderRight: "1px solid var(--color-border)",
            color: "var(--color-text-secondary)",
          },
          ".cm-activeLineGutter": {
            backgroundColor: "var(--color-bg)",
          },
        }),
      ],
    });

    const view = new EditorView({
      state,
      parent: containerRef.current,
    });

    viewRef.current = view;

    return () => {
      view.destroy();
      viewRef.current = null;
    };
  // Only run on mount - value is handled separately
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync external value changes into the editor
  const syncValue = useCallback((newValue: string) => {
    const view = viewRef.current;
    if (!view) return;

    const currentValue = view.state.doc.toString();
    if (currentValue === newValue) return;

    isExternalUpdate.current = true;
    view.dispatch({
      changes: {
        from: 0,
        to: currentValue.length,
        insert: newValue,
      },
    });
    isExternalUpdate.current = false;
  }, []);

  // When value prop changes (e.g., switching from rich text to source), sync it
  useEffect(() => {
    syncValue(value);
  }, [value, syncValue]);

  return (
    <div
      ref={containerRef}
      className={`${styles.container} ${className ?? ""}`}
    />
  );
}
