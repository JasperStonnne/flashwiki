import { Extension, Range } from "@tiptap/core";
import Suggestion, { SuggestionOptions } from "@tiptap/suggestion";
import type { Editor } from "@tiptap/core";
import type { ReactNode } from "react";
import {
  TextHOne,
  TextHTwo,
  TextHThree,
  ListBullets,
  ListNumbers,
  CheckSquare,
  Quotes,
  CodeBlock,
  Minus,
  Table,
  Image,
  VideoCamera,
  MathOperations,
  TreeStructure,
} from "@phosphor-icons/react";
import { insertMarkdownImage, insertMarkdownVideo } from "@/lib/link-utils";
import type { Dictionary } from "@/i18n";

const SLASH_ICON_SIZE = 20;

export type SlashCommandGroup = "text" | "list" | "block" | "media" | "advanced";

export interface SlashCommandItem {
  id: string;
  title: string;
  description: string;
  searchTerms: string[];
  icon: ReactNode;
  group: SlashCommandGroup;
  command: (props: { editor: Editor; range: Range }) => void;
}

export function getSlashCommandItems(t: Dictionary): SlashCommandItem[] {
  return [
    {
      id: "heading1",
      title: t.slash.heading1.title,
      description: t.slash.heading1.description,
      searchTerms: ["heading 1", "heading", "h1", "\u6807\u98981", "\u6807\u9898", "\u5927\u6807\u9898"],
      icon: <TextHOne size={SLASH_ICON_SIZE} />,
      group: "text",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).setHeading({ level: 1 }).run();
      },
    },
    {
      id: "heading2",
      title: t.slash.heading2.title,
      description: t.slash.heading2.description,
      searchTerms: ["heading 2", "heading", "h2", "\u6807\u98982", "\u6807\u9898", "\u4E2D\u6807\u9898"],
      icon: <TextHTwo size={SLASH_ICON_SIZE} />,
      group: "text",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).setHeading({ level: 2 }).run();
      },
    },
    {
      id: "heading3",
      title: t.slash.heading3.title,
      description: t.slash.heading3.description,
      searchTerms: ["heading 3", "heading", "h3", "\u6807\u98983", "\u6807\u9898", "\u5C0F\u6807\u9898"],
      icon: <TextHThree size={SLASH_ICON_SIZE} />,
      group: "text",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).setHeading({ level: 3 }).run();
      },
    },
    {
      id: "bulletList",
      title: t.slash.bulletList.title,
      description: t.slash.bulletList.description,
      searchTerms: ["bullet list", "bullet", "unordered", "ul", "\u65E0\u5E8F\u5217\u8868", "\u5217\u8868"],
      icon: <ListBullets size={SLASH_ICON_SIZE} />,
      group: "list",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).toggleBulletList().run();
      },
    },
    {
      id: "numberedList",
      title: t.slash.numberedList.title,
      description: t.slash.numberedList.description,
      searchTerms: ["numbered list", "numbered", "ordered", "ol", "\u6709\u5E8F\u5217\u8868", "\u5217\u8868"],
      icon: <ListNumbers size={SLASH_ICON_SIZE} />,
      group: "list",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).toggleOrderedList().run();
      },
    },
    {
      id: "taskList",
      title: t.slash.taskList.title,
      description: t.slash.taskList.description,
      searchTerms: ["task list", "task", "todo", "checkbox", "\u4EFB\u52A1\u5217\u8868", "\u4EFB\u52A1", "\u5F85\u529E"],
      icon: <CheckSquare size={SLASH_ICON_SIZE} />,
      group: "list",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).toggleTaskList().run();
      },
    },
    {
      id: "blockquote",
      title: t.slash.blockquote.title,
      description: t.slash.blockquote.description,
      searchTerms: ["blockquote", "quote", "\u5F15\u7528"],
      icon: <Quotes size={SLASH_ICON_SIZE} />,
      group: "block",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).toggleBlockquote().run();
      },
    },
    {
      id: "codeBlock",
      title: t.slash.codeBlock.title,
      description: t.slash.codeBlock.description,
      searchTerms: ["code block", "code", "codeblock", "\u4EE3\u7801\u5757", "\u4EE3\u7801"],
      icon: <CodeBlock size={SLASH_ICON_SIZE} />,
      group: "block",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).toggleCodeBlock().run();
      },
    },
    {
      id: "horizontalRule",
      title: t.slash.horizontalRule.title,
      description: t.slash.horizontalRule.description,
      searchTerms: ["horizontal rule", "divider", "hr", "\u5206\u5272\u7EBF", "\u6C34\u5E73\u7EBF"],
      icon: <Minus size={SLASH_ICON_SIZE} />,
      group: "block",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).setHorizontalRule().run();
      },
    },
    {
      id: "table",
      title: t.slash.table.title,
      description: t.slash.table.description,
      searchTerms: ["table", "\u8868\u683C"],
      icon: <Table size={SLASH_ICON_SIZE} />,
      group: "media",
      command: ({ editor, range }) => {
        editor
          .chain()
          .focus()
          .deleteRange(range)
          .insertTable({ rows: 3, cols: 3, withHeaderRow: true })
          .run();
      },
    },
    {
      id: "image",
      title: t.slash.image.title,
      description: t.slash.image.description,
      searchTerms: ["image", "picture", "img", "\u56FE\u7247", "\u63D2\u56FE"],
      icon: <Image size={SLASH_ICON_SIZE} />,
      group: "media",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).run();
        insertMarkdownImage(editor);
      },
    },
    {
      id: "video",
      title: t.slash.video.title,
      description: t.slash.video.description,
      searchTerms: ["video", "\u89C6\u9891"],
      icon: <VideoCamera size={SLASH_ICON_SIZE} />,
      group: "media",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).run();
        insertMarkdownVideo(editor);
      },
    },
    {
      id: "mathBlock",
      title: t.slash.mathBlock.title,
      description: t.slash.mathBlock.description,
      searchTerms: ["math block", "math", "latex", "formula", "\u6570\u5B66\u516C\u5F0F", "\u516C\u5F0F"],
      icon: <MathOperations size={SLASH_ICON_SIZE} />,
      group: "advanced",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).run();
        editor.commands.insertContent({
          type: "mathBlock",
          attrs: { latex: "" },
        });
      },
    },
    {
      id: "plantuml",
      title: t.slash.plantuml.title,
      description: t.slash.plantuml.description,
      searchTerms: ["plantuml", "diagram", "uml", "PlantUML", "\u56FE\u8868", "\u6D41\u7A0B\u56FE"],
      icon: <TreeStructure size={SLASH_ICON_SIZE} />,
      group: "advanced",
      command: ({ editor, range }) => {
        editor.chain().focus().deleteRange(range).run();
        editor.commands.insertContent({
          type: "plantumlBlock",
          attrs: { source: "@startuml\n\n@enduml" },
        });
      },
    },
  ];
}

export interface SlashCommandOptions {
  suggestion: Partial<SuggestionOptions<SlashCommandItem>>;
}

export const SlashCommand = Extension.create<SlashCommandOptions>({
  name: "slashCommand",

  addOptions() {
    return {
      suggestion: {
        char: "/",
        command: ({ editor, range, props }) => {
          props.command({ editor, range });
        },
      },
    };
  },

  addProseMirrorPlugins() {
    return [
      Suggestion({
        editor: this.editor,
        ...this.options.suggestion,
      }),
    ];
  },
});
