import { Extension } from "@tiptap/core";
import { insertMarkdownLink } from "@/lib/link-utils";

/**
 * 自定义快捷键扩展。
 *
 * StarterKit 已经内置了 Bold(Ctrl+B), Italic(Ctrl+I) 等常用快捷键。
 * 这里补充额外快捷键：
 *   - Ctrl+Alt+1/2/3/4/5/6  → Heading 1/2/3/4/5/6
 *   - Ctrl+Alt+0            → Normal text (paragraph)
 *   - Ctrl+U                → Underline
 *   - Ctrl+Shift+H          → Highlight
 *   - Ctrl+K                → Insert Markdown link
 */
export const CustomKeymap = Extension.create({
  name: "customKeymap",

  addKeyboardShortcuts() {
    return {
      "Mod-Alt-1": () =>
        this.editor.chain().focus().toggleHeading({ level: 1 }).run(),
      "Mod-Alt-2": () =>
        this.editor.chain().focus().toggleHeading({ level: 2 }).run(),
      "Mod-Alt-3": () =>
        this.editor.chain().focus().toggleHeading({ level: 3 }).run(),
      "Mod-Alt-4": () =>
        this.editor.chain().focus().toggleHeading({ level: 4 }).run(),
      "Mod-Alt-5": () =>
        this.editor.chain().focus().toggleHeading({ level: 5 }).run(),
      "Mod-Alt-6": () =>
        this.editor.chain().focus().toggleHeading({ level: 6 }).run(),
      "Mod-Alt-0": () =>
        this.editor.chain().focus().setParagraph().run(),
      "Mod-u": () =>
        this.editor.chain().focus().toggleUnderline().run(),
      "Mod-Shift-h": () =>
        this.editor.chain().focus().toggleHighlight().run(),
      "Mod-k": () => {
        insertMarkdownLink(this.editor);
        return true;
      },
    };
  },
});
