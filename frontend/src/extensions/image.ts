import { Node, mergeAttributes, InputRule, PasteRule } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import { Plugin, PluginKey } from "@tiptap/pm/state";

// Types for tiptap-markdown serialization
interface MarkdownSerializerState {
  write(text: string): void;
  text(text: string, escape?: boolean): void;
  ensureNewLine(): void;
  closeBlock(node: ProseMirrorNode): void;
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    resizableImage: {
      setImage: (options: { src: string; alt?: string; title?: string; width?: number }) => ReturnType;
    };
  }
}

export const Image = Node.create({
  name: "image",
  group: "block",
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      src: {
        default: null,
      },
      alt: {
        default: null,
      },
      title: {
        default: null,
      },
      width: {
        default: 100,
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: "img[src]",
        getAttrs: (element) => {
          const el = element as HTMLImageElement;
          const widthAttr = el.getAttribute("width");
          let width = 100;
          if (widthAttr) {
            // Handle both "50%" and "50" formats
            const parsed = parseInt(widthAttr.replace("%", ""), 10);
            if (!isNaN(parsed)) width = parsed;
          }
          return {
            src: el.getAttribute("src"),
            alt: el.getAttribute("alt"),
            title: el.getAttribute("title"),
            width,
          };
        },
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      "img",
      mergeAttributes(HTMLAttributes, {
        style: `width: ${HTMLAttributes.width}%`,
      }),
    ];
  },

  addStorage() {
    return {
      markdown: {
        // Serialize as raw HTML to preserve width attribute
        // Standard Markdown ![alt](src) would lose width info
        serialize(state: MarkdownSerializerState, node: ProseMirrorNode) {
          const src = node.attrs.src as string || "";
          const alt = node.attrs.alt as string || "";
          const title = node.attrs.title as string || "";
          const width = node.attrs.width as number;

          // Escape quotes in attributes
          const escapedSrc = src.replace(/"/g, "&quot;");
          const escapedAlt = alt.replace(/"/g, "&quot;");
          const escapedTitle = title.replace(/"/g, "&quot;");

          state.write(`<img src="${escapedSrc}" alt="${escapedAlt}" title="${escapedTitle}" width="${width}%">`);
          state.closeBlock(node);
        },
        parse: {
          // Parsing is handled by parseHTML() since markdown-it with html: true
          // will pass through <img> tags which TipTap then parses
        },
      },
    };
  },

  addNodeView() {
    return ({ node, editor, getPos }) => {
      const container = document.createElement("div");
      container.classList.add("resizable-image");
      container.style.width = `${node.attrs.width}%`;

      const img = document.createElement("img");
      img.src = node.attrs.src as string;
      img.alt = (node.attrs.alt as string) || "";
      img.title = (node.attrs.title as string) || "";
      img.style.width = "100%";
      img.draggable = false;

      // Handle load failure with fallback link display
      img.onerror = () => {
        // Replace img element with a fallback link display
        container.innerHTML = ""; // clear resize handles and img
        container.classList.remove("resizable-image");
        container.classList.add("image-fallback");

        const prefix = document.createElement("span");
        prefix.textContent = "!";
        prefix.className = "image-fallback-prefix";

        const linkEl = document.createElement("a");
        linkEl.href = node.attrs.src as string;
        linkEl.textContent = (node.attrs.alt as string) || (node.attrs.src as string);
        linkEl.className = "image-fallback-link";
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
      container.appendChild(img);
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
          if (updatedNode.type.name !== "image") return false;
          img.src = updatedNode.attrs.src as string;
          img.alt = (updatedNode.attrs.alt as string) || "";
          img.title = (updatedNode.attrs.title as string) || "";
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

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: new PluginKey("imageDrop"),
        props: {
          handleDrop(view, event, _slice, moved) {
            // Only handle file drops, not internal node moves
            if (moved || !event.dataTransfer?.files?.length) {
              return false;
            }

            const files = Array.from(event.dataTransfer.files);
            const imageFiles = files.filter((f) => f.type.startsWith("image/"));

            if (imageFiles.length === 0) return false;

            event.preventDefault();

            // Get drop position
            const coordinates = view.posAtCoords({
              left: event.clientX,
              top: event.clientY,
            });

            if (!coordinates) return false;

            // Insert Markdown image placeholder for each file
            const { tr } = view.state;
            let insertPos = coordinates.pos;

            for (const file of imageFiles) {
              const filename = file.name;
              const text = `![${filename}]()`;
              tr.insertText(text, insertPos);
              insertPos += text.length;
            }

            view.dispatch(tr);
            return true;
          },
        },
      }),
    ];
  },

  addInputRules() {
    return [
      new InputRule({
        // Match ![alt](url) when the closing ) is typed
        find: /!\[([^\]]*)\]\(([^)]+)\)$/,
        handler: ({ state, range, match }) => {
          const alt = match[1] ?? "";
          const src = match[2] ?? "";
          const { tr } = state;

          // Replace the text with an image node
          tr.replaceWith(range.from, range.to, this.type.create({ src, alt }));
        },
      }),
    ];
  },

  addPasteRules() {
    return [
      new PasteRule({
        find: /!\[([^\]]*)\]\(([^)]+)\)/g,
        handler: ({ state, range, match }) => {
          const alt = match[1] ?? "";
          const src = match[2] ?? "";
          const { tr } = state;
          tr.replaceWith(range.from, range.to, this.type.create({ src, alt }));
        },
      }),
    ];
  },

  addCommands() {
    return {
      setImage:
        (options) =>
        ({ commands }) => {
          return commands.insertContent({
            type: this.name,
            attrs: options,
          });
        },
    };
  },
});
