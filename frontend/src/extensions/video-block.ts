import { Node, mergeAttributes, InputRule, PasteRule } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";

// Types for tiptap-markdown serialization
interface MarkdownSerializerState {
  write(text: string): void;
  text(text: string, escape?: boolean): void;
  ensureNewLine(): void;
  closeBlock(node: ProseMirrorNode): void;
}

interface MarkdownItBlockState {
  src: string;
  bMarks: number[];
  eMarks: number[];
  tShift: number[];
  line: number;
  push(type: string, tag: string, nesting: number): {
    content: string;
    map: [number, number];
    attrSet(name: string, value: string): void;
    attrGet?(name: string): string | null;
  };
}

interface MarkdownItInstance {
  block: {
    ruler: {
      before(name: string, ruleName: string, fn: (state: MarkdownItBlockState, startLine: number, endLine: number, silent: boolean) => boolean): void;
    };
  };
  renderer: {
    rules: Record<string, (tokens: Array<{ content: string; attrGet?(name: string): string | null }>, idx: number) => string>;
  };
  utils: {
    escapeHtml(str: string): string;
  };
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    videoBlock: {
      setVideoBlock: (attributes: {
        src: string;
        width?: number;
        title?: string;
      }) => ReturnType;
    };
  }
}

export const VideoBlock = Node.create({
  name: "videoBlock",
  group: "block",
  atom: true,

  addAttributes() {
    return {
      src: {
        default: "",
      },
      width: {
        default: 100,
      },
      title: {
        default: "",
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: 'div[data-type="video-block"]',
        getAttrs: (element) => {
          const el = element as HTMLElement;
          return {
            src: el.getAttribute("data-src") || el.getAttribute("src") || "",
            title: el.getAttribute("data-title") || el.getAttribute("title") || "",
            width: 100,
          };
        },
      },
      {
        // Parse raw <video> tags from Markdown HTML
        tag: "video[src]",
        getAttrs: (element) => {
          const el = element as HTMLVideoElement;
          const widthAttr = el.getAttribute("width");
          let width = 100;
          if (widthAttr) {
            // Handle both "50%" and "50" formats
            const parsed = parseInt(widthAttr.replace("%", ""), 10);
            if (!isNaN(parsed)) width = parsed;
          }
          return {
            src: el.getAttribute("src") || "",
            width,
            title: el.getAttribute("title") || "",
          };
        },
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      "div",
      mergeAttributes(HTMLAttributes, { "data-type": "video-block" }),
    ];
  },

  addStorage() {
    return {
      markdown: {
        // Serialize as @[title](url) custom syntax
        serialize(state: MarkdownSerializerState, node: ProseMirrorNode) {
          const src = node.attrs.src as string;
          const title = node.attrs.title as string;
          state.write(`@[${title}](${src})`);
          state.closeBlock(node);
        },
        parse: {
          setup(markdownit: MarkdownItInstance) {
            // Add block rule for @[title](url) syntax
            markdownit.block.ruler.before("paragraph", "video_block",
              (state: MarkdownItBlockState, startLine: number, _endLine: number, silent: boolean) => {
                const bMarkStart = state.bMarks[startLine];
                const tShiftStart = state.tShift[startLine];
                const eMarkStart = state.eMarks[startLine];

                if (bMarkStart === undefined || tShiftStart === undefined || eMarkStart === undefined) {
                  return false;
                }

                const lineStart = bMarkStart + tShiftStart;
                const lineEnd = eMarkStart;
                const lineText = state.src.slice(lineStart, lineEnd).trim();

                // Match @[title](url)
                const match = lineText.match(/^@\[([^\]]*)\]\(([^)]+)\)$/);
                if (!match) return false;

                if (silent) return true;

                const title = match[1] ?? "";
                const src = match[2] ?? "";

                const token = state.push("video_block", "div", 0);
                token.content = src;
                token.attrSet("data-type", "video-block");
                token.attrSet("data-src", src);
                token.attrSet("data-title", title);
                token.map = [startLine, startLine + 1];

                state.line = startLine + 1;
                return true;
              }
            );

            markdownit.renderer.rules.video_block = (tokens: Array<{ content: string; attrGet?(name: string): string | null }>, idx: number) => {
              const token = tokens[idx];
              if (!token) return "";
              const src = token.attrGet?.("data-src") ?? token.content ?? "";
              const title = token.attrGet?.("data-title") ?? "";
              const escapedSrc = markdownit.utils.escapeHtml(src);
              const escapedTitle = markdownit.utils.escapeHtml(title);
              return `<div data-type="video-block" data-src="${escapedSrc}" data-title="${escapedTitle}"></div>\n`;
            };
          },
          updateDOM(element: HTMLElement) {
            const src = element.getAttribute("data-src");
            const title = element.getAttribute("data-title");
            if (src) element.setAttribute("src", src);
            if (title) element.setAttribute("title", title);
          },
        },
      },
    };
  },

  addNodeView() {
    return ({ node, editor, getPos }) => {
      const container = document.createElement("div");
      container.classList.add("resizable-video");
      container.style.width = `${node.attrs.width}%`;

      const video = document.createElement("video");
      video.classList.add("video-block-player");
      video.controls = true;
      video.src = node.attrs.src as string;
      video.title = node.attrs.title as string;
      video.style.width = "100%";

      // Handle load failure with fallback link display
      video.onerror = () => {
        container.innerHTML = "";
        container.classList.remove("resizable-video");
        container.classList.add("video-fallback");

        const prefix = document.createElement("span");
        prefix.textContent = "@";
        prefix.className = "video-fallback-prefix";

        const linkEl = document.createElement("a");
        linkEl.href = node.attrs.src as string;
        linkEl.textContent = (node.attrs.title as string) || (node.attrs.src as string);
        linkEl.className = "video-fallback-link";
        linkEl.target = "_blank";
        linkEl.rel = "noopener noreferrer";
        linkEl.title = node.attrs.src as string;

        container.appendChild(prefix);
        container.appendChild(linkEl);
      };

      const leftHandle = document.createElement("div");
      leftHandle.classList.add("resize-handle", "resize-handle-left");

      const rightHandle = document.createElement("div");
      rightHandle.classList.add("resize-handle", "resize-handle-right");

      container.appendChild(leftHandle);
      container.appendChild(video);
      container.appendChild(rightHandle);

      let isDragging = false;
      let startX = 0;
      let startWidth = 0;
      let side: "left" | "right" = "right";

      const handleMouseDown = (e: MouseEvent, handleSide: "left" | "right") => {
        e.preventDefault();
        e.stopPropagation();
        isDragging = true;
        startX = e.clientX;
        startWidth = container.offsetWidth;
        side = handleSide;
        document.body.style.cursor = "ew-resize";
        document.body.style.userSelect = "none";
      };

      const handleMouseMove = (e: MouseEvent) => {
        if (!isDragging) return;

        const parent = container.parentElement;
        if (!parent) return;

        const parentWidth = parent.offsetWidth;
        const deltaX = e.clientX - startX;
        const multiplier = side === "right" ? 1 : -1;
        const newWidth = startWidth + deltaX * multiplier * 2;
        const newWidthPercent = Math.max(10, Math.min(100, (newWidth / parentWidth) * 100));

        container.style.width = `${newWidthPercent}%`;
      };

      const handleMouseUp = () => {
        if (!isDragging) return;
        isDragging = false;
        document.body.style.cursor = "";
        document.body.style.userSelect = "";

        if (typeof getPos !== "function") return;
        const pos = getPos();
        if (pos === undefined) return;

        const parent = container.parentElement;
        if (!parent) return;

        const parentWidth = parent.offsetWidth;
        const currentWidth = container.offsetWidth;
        const widthPercent = Math.round((currentWidth / parentWidth) * 100);

        editor
          .chain()
          .focus()
          .command(({ tr }) => {
            tr.setNodeMarkup(pos, undefined, {
              ...node.attrs,
              width: widthPercent,
            });
            return true;
          })
          .run();
      };

      leftHandle.addEventListener("mousedown", (e) => handleMouseDown(e, "left"));
      rightHandle.addEventListener("mousedown", (e) => handleMouseDown(e, "right"));
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);

      return {
        dom: container,
        update: (updatedNode) => {
          if (updatedNode.type.name !== "videoBlock") return false;
          video.src = updatedNode.attrs.src as string;
          video.title = updatedNode.attrs.title as string;
          container.style.width = `${updatedNode.attrs.width}%`;
          return true;
        },
        destroy: () => {
          document.removeEventListener("mousemove", handleMouseMove);
          document.removeEventListener("mouseup", handleMouseUp);
        },
      };
    };
  },

  addInputRules() {
    return [
      new InputRule({
        find: /@\[([^\]]*)\]\(([^)]+)\)$/,
        handler: ({ state, range, match }) => {
          const title = match[1] ?? "";
          const src = match[2] ?? "";
          const { tr } = state;
          tr.replaceWith(range.from, range.to,
            this.type.create({ src, title })
          );
        },
      }),
    ];
  },

  addPasteRules() {
    return [
      new PasteRule({
        find: /@\[([^\]]*)\]\(([^)]+)\)/g,
        handler: ({ state, range, match }) => {
          const title = match[1] ?? "";
          const src = match[2] ?? "";
          const { tr } = state;
          tr.replaceWith(range.from, range.to,
            this.type.create({ src, title })
          );
        },
      }),
    ];
  },

  addCommands() {
    return {
      setVideoBlock:
        (attributes) =>
        ({ commands }) => {
          return commands.insertContent({
            type: this.name,
            attrs: attributes,
          });
        },
    };
  },
});
