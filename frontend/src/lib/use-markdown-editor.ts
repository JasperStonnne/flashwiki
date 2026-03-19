import { useRef, useEffect, useMemo } from "react";
import { useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import { Markdown, type MarkdownStorage } from "tiptap-markdown";
import {
  CustomKeymap,
  Underline,
  TaskList,
  TaskItem,
  Link,
  Image,
  Table,
  TableRow,
  TableHeader,
  TableCell,
  CodeBlockLowlight,
  MathInline,
  MathBlock,
  PlantUMLBlock,
  VideoBlock,
  SlashCommand,
  Highlight,
  TyporaMode,
  TabHandler,
} from "@/extensions";
import { createSlashCommandSuggestion } from "./slash-command-suggestion";
import { dictionaries } from "@/i18n";
import type { Locale, LocaleRef } from "@/i18n";

export interface UseMarkdownEditorOptions {
  /** Initial content (Markdown string) */
  content?: string;
  /** Placeholder hint text */
  placeholder?: string;
  /** Content change callback */
  onUpdate?: (markdown: string) => void;
  /** Current locale */
  locale?: Locale;
}

export function useMarkdownEditor(options: UseMarkdownEditorOptions = {}) {
  const {
    content,
    placeholder,
    onUpdate,
    locale = "zh",
  } = options;

  const localeRef = useRef<LocaleRef["current"]>(dictionaries[locale]);

  // Keep localeRef in sync with current locale
  useEffect(() => {
    localeRef.current = dictionaries[locale];
  }, [locale]);

  const localeRefObj = useMemo<LocaleRef>(() => ({ get current() { return localeRef.current; }, set current(v) { localeRef.current = v; } }), []);

  const slashSuggestion = useMemo(
    () => createSlashCommandSuggestion(localeRefObj),
    [localeRefObj]
  );

  // Compute initial content and placeholder once (don't react to locale changes for these)
  const initialContent = useMemo(
    () => content ?? dictionaries[locale].editor.defaultContent,
    // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  );

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [1, 2, 3, 4, 5, 6] },
        codeBlock: false,
      }),
      Placeholder.configure({
        showOnlyWhenEditable: true,
        showOnlyCurrent: true,
        placeholder: ({ node }) => {
          if (node.type.name === "heading") {
            const t = localeRef.current;
            return `${t.placeholder.heading} ${node.attrs.level as number}`;
          }
          return placeholder ?? localeRef.current.placeholder.default;
        },
      }),
      // tiptap-markdown: enables Markdown serialization/parsing directly on ProseMirror doc
      // html: true allows raw HTML in Markdown (needed for video/image with attributes)
      Markdown.configure({
        html: true,
        transformPastedText: true,
        transformCopiedText: true,
      }),
      CustomKeymap,
      Underline,
      TaskList,
      TaskItem,
      Link,
      Image,
      Table,
      TableRow,
      TableHeader,
      TableCell,
      CodeBlockLowlight,
      MathInline,
      MathBlock.configure({
        localeRef: localeRefObj,
      }),
      PlantUMLBlock.configure({
        localeRef: localeRefObj,
      }),
      VideoBlock,
      Highlight,
      TyporaMode,
      TabHandler.configure({
        localeRef: localeRefObj,
      }),
      SlashCommand.configure({
        suggestion: slashSuggestion,
      }),
    ],
    content: initialContent,
    onUpdate: ({ editor: e }) => {
      // tiptap-markdown adds `markdown` to storage at runtime
      const storage = e.storage as unknown as { markdown: MarkdownStorage };
      onUpdate?.(storage.markdown.getMarkdown());
    },
    editorProps: {
      attributes: {
        spellcheck: "false",
      },
    },
  });

  return editor;
}
