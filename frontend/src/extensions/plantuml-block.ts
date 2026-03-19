import { Node, mergeAttributes, InputRule } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import { encode } from "plantuml-encoder";
import type { LocaleRef } from "@/i18n";

// Types for tiptap-markdown serialization
interface MarkdownSerializerState {
  write(text: string): void;
  text(text: string, escape?: boolean): void;
  ensureNewLine(): void;
  closeBlock(node: ProseMirrorNode): void;
}

interface MarkdownItInstance {
  renderer: {
    rules: Record<string, (tokens: Array<{ info: string; content: string }>, idx: number) => string | undefined>;
  };
  utils: {
    escapeHtml(str: string): string;
  };
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    plantumlBlock: {
      setPlantUMLBlock: (attributes: { source: string }) => ReturnType;
    };
  }
}

const PLANTUML_SERVER = "https://www.plantuml.com/plantuml/svg";

interface PlantUMLBlockOptions {
  localeRef: LocaleRef | null;
}

export const PlantUMLBlock = Node.create<PlantUMLBlockOptions>({
  name: "plantumlBlock",
  group: "block",
  atom: true,

  addOptions() {
    return {
      localeRef: null,
    };
  },

  addAttributes() {
    return {
      source: {
        default: "@startuml\n\n@enduml",
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: 'div[data-type="plantuml-block"]',
        getAttrs: (element) => {
          const el = element as HTMLElement;
          return {
            source: el.getAttribute("source") || el.getAttribute("data-source") || "@startuml\n\n@enduml",
          };
        },
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      "div",
      mergeAttributes(HTMLAttributes, { "data-type": "plantuml-block" }),
    ];
  },

  addStorage() {
    return {
      markdown: {
        serialize(state: MarkdownSerializerState, node: ProseMirrorNode) {
          state.write("```plantuml\n");
          state.text(node.attrs.source as string, false);
          state.ensureNewLine();
          state.write("```");
          state.closeBlock(node);
        },
        parse: {
          setup(markdownit: MarkdownItInstance) {
            // Override fence renderer to handle plantuml specially
            markdownit.renderer.rules.fence = (tokens: Array<{ info: string; content: string }>, idx: number) => {
              const token = tokens[idx];
              if (!token) return "";

              const info = token.info.trim();
              if (info === "plantuml") {
                const source = token.content;
                // Return HTML that will be parsed as plantumlBlock
                return `<div data-type="plantuml-block" data-source="${markdownit.utils.escapeHtml(source)}"></div>\n`;
              }

              // Fall back to default fence rendering for other languages
              const escapedContent = markdownit.utils.escapeHtml(token.content);
              return `<pre><code class="language-${info}">${escapedContent}</code></pre>\n`;
            };
          },
          updateDOM(element: HTMLElement) {
            // Transfer data-source to source attribute for TipTap parsing
            const source = element.getAttribute("data-source");
            if (source) {
              element.setAttribute("source", source);
            }
          },
        },
      },
    };
  },

  addNodeView() {
    return ({ node, editor, getPos }) => {
      const localeRef = this.options.localeRef;
      const dom = document.createElement("div");
      dom.classList.add("plantuml-block");
      dom.setAttribute("data-type", "plantuml-block");

      // Create textarea (hidden by default in collapsed state)
      const textarea = document.createElement("textarea");
      textarea.classList.add("plantuml-block-input");
      textarea.value = node.attrs.source as string;
      textarea.placeholder = "@startuml\nAlice -> Bob: Hello\n@enduml";
      textarea.rows = 6;

      // Create preview container (visible by default in collapsed state)
      const preview = document.createElement("div");
      preview.classList.add("plantuml-block-preview");

      const img = document.createElement("img");
      img.classList.add("plantuml-block-img");
      preview.appendChild(img);

      const loadingDiv = document.createElement("div");
      loadingDiv.classList.add("plantuml-block-loading");
      loadingDiv.textContent = localeRef?.current.plantumlBlock.rendering ?? "Rendering...";
      loadingDiv.style.display = "none";
      preview.appendChild(loadingDiv);

      const errorDiv = document.createElement("div");
      errorDiv.classList.add("plantuml-error");
      errorDiv.style.display = "none";
      preview.appendChild(errorDiv);

      // State management
      let isEditing = false;
      let currentNode = node;

      const renderDiagram = (source: string) => {
        const trimmed = source.trim();
        const isDefault = !trimmed || trimmed === "@startuml\n\n@enduml" || trimmed === "@startuml\n@enduml";

        if (isDefault) {
          img.style.display = "none";
          loadingDiv.style.display = "none";
          errorDiv.style.display = "none";
          // Show placeholder for empty diagrams
          const placeholder = preview.querySelector(".plantuml-placeholder");
          if (!placeholder) {
            const p = document.createElement("span");
            p.classList.add("plantuml-placeholder");
            p.textContent = localeRef?.current.plantumlBlock.clickToAdd ?? "Click to add diagram";
            preview.appendChild(p);
          }
          return;
        }

        // Remove placeholder if exists
        const placeholder = preview.querySelector(".plantuml-placeholder");
        if (placeholder) placeholder.remove();

        try {
          const encoded = encode(source);
          const url = `${PLANTUML_SERVER}/${encoded}`;

          // Show loading indicator
          loadingDiv.style.display = "block";
          img.style.display = "none";
          errorDiv.style.display = "none";

          img.src = url;
          img.alt = "PlantUML Diagram";

          img.onload = () => {
            loadingDiv.style.display = "none";
            img.style.display = "block";
            errorDiv.style.display = "none";
          };

          img.onerror = () => {
            loadingDiv.style.display = "none";
            img.style.display = "none";
            errorDiv.textContent = localeRef?.current.plantumlBlock.failedToRender ?? "Failed to render diagram";
            errorDiv.style.display = "block";
          };
        } catch {
          loadingDiv.style.display = "none";
          img.style.display = "none";
          errorDiv.textContent = localeRef?.current.plantumlBlock.invalidSyntax ?? "Invalid PlantUML syntax";
          errorDiv.style.display = "block";
        }
      };

      const autoResizeTextarea = () => {
        textarea.style.height = "auto";
        textarea.style.height = Math.max(80, textarea.scrollHeight) + "px";
      };

      const enterEditMode = () => {
        if (isEditing) return;
        isEditing = true;

        // Hide preview, show textarea
        preview.style.display = "none";
        textarea.style.display = "block";

        // Sync textarea value from current node attrs
        textarea.value = currentNode.attrs.source as string;
        autoResizeTextarea();

        // Focus the textarea
        textarea.focus();
      };

      const exitEditMode = () => {
        if (!isEditing) return;

        // Save changes if value differs
        const newSource = textarea.value;
        if (newSource !== currentNode.attrs.source) {
          if (typeof getPos === "function") {
            const pos = getPos();
            if (pos !== undefined) {
              editor.view.dispatch(
                editor.view.state.tr.setNodeMarkup(pos, undefined, { source: newSource })
              );
            }
          }
        }

        isEditing = false;

        // Hide textarea, show preview
        textarea.style.display = "none";
        preview.style.display = "flex";

        // Re-render diagram with current value
        renderDiagram(textarea.value);
      };

      // Initial render
      renderDiagram(node.attrs.source as string);

      // Check if content is empty or default
      const isEmptyOrDefault = (source: string) => {
        const trimmed = source.trim();
        return !trimmed || trimmed === "@startuml\n\n@enduml" || trimmed === "@startuml\n@enduml";
      };

      // If content is empty or default, start in expanded mode
      if (isEmptyOrDefault(node.attrs.source as string)) {
        // Start expanded
        preview.style.display = "none";
        textarea.style.display = "block";
        isEditing = true;
        // Focus will be called by selectNode
      } else {
        // Start collapsed
        preview.style.display = "flex";
        textarea.style.display = "none";
      }

      // Click on preview enters edit mode
      preview.addEventListener("click", () => {
        enterEditMode();
      });

      // Textarea input - auto-resize
      textarea.addEventListener("input", () => {
        autoResizeTextarea();
      });

      // Textarea blur exits edit mode
      textarea.addEventListener("blur", () => {
        // Small delay to allow selectNode/deselectNode to fire first
        setTimeout(() => {
          if (isEditing && document.activeElement !== textarea) {
            exitEditMode();
          }
        }, 0);
      });

      // Keyboard navigation at boundaries
      textarea.addEventListener("keydown", (e) => {
        const pos = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const len = textarea.value.length;
        const hasSelection = pos !== end;

        // Backspace at position 0 with no selection: delete the entire node
        if (e.key === "Backspace" && pos === 0 && !hasSelection) {
          e.preventDefault();
          if (typeof getPos !== "function") return;
          const nodePos = getPos();
          if (nodePos === undefined) return;

          // Exit edit mode first, then delete the node
          exitEditMode();

          editor
            .chain()
            .focus()
            .command(({ tr }) => {
              tr.delete(nodePos, nodePos + currentNode.nodeSize);
              return true;
            })
            .run();
          return;
        }

        // Arrow Up at first line: move cursor to block before this node
        if (e.key === "ArrowUp" && !hasSelection) {
          const textBeforeCursor = textarea.value.slice(0, pos);
          const isFirstLine = !textBeforeCursor.includes("\n");

          if (isFirstLine) {
            e.preventDefault();
            exitEditMode();

            if (typeof getPos !== "function") return;
            const nodePos = getPos();
            if (nodePos === undefined) return;

            editor.chain().focus().setTextSelection(nodePos).run();
            return;
          }
        }

        // Arrow Down at last line: move cursor to block after this node
        if (e.key === "ArrowDown" && !hasSelection) {
          const textAfterCursor = textarea.value.slice(pos);
          const isLastLine = !textAfterCursor.includes("\n");

          if (isLastLine) {
            e.preventDefault();
            exitEditMode();

            if (typeof getPos !== "function") return;
            const nodePos = getPos();
            if (nodePos === undefined) return;

            const afterPos = nodePos + currentNode.nodeSize;
            editor.chain().focus().setTextSelection(afterPos).run();
            return;
          }
        }

        // Arrow Left at position 0: move to block before
        if (e.key === "ArrowLeft" && pos === 0 && !hasSelection) {
          e.preventDefault();
          exitEditMode();

          if (typeof getPos !== "function") return;
          const nodePos = getPos();
          if (nodePos === undefined) return;

          editor.chain().focus().setTextSelection(nodePos).run();
          return;
        }

        // Arrow Right at end: move to block after
        if (e.key === "ArrowRight" && pos === len && !hasSelection) {
          e.preventDefault();
          exitEditMode();

          if (typeof getPos !== "function") return;
          const nodePos = getPos();
          if (nodePos === undefined) return;

          const afterPos = nodePos + currentNode.nodeSize;
          editor.chain().focus().setTextSelection(afterPos).run();
          return;
        }

        // Escape: exit edit mode
        if (e.key === "Escape") {
          e.preventDefault();
          exitEditMode();
          editor.commands.focus();
          return;
        }
      });

      dom.appendChild(preview);
      dom.appendChild(textarea);

      return {
        dom,
        selectNode: () => {
          enterEditMode();
        },
        deselectNode: () => {
          exitEditMode();
        },
        update: (updatedNode) => {
          if (updatedNode.type.name !== "plantumlBlock") return false;
          currentNode = updatedNode;
          if (!isEditing) {
            textarea.value = updatedNode.attrs.source as string;
            renderDiagram(updatedNode.attrs.source as string);
          }
          return true;
        },
        stopEvent: (event) => {
          return event.target === textarea;
        },
        ignoreMutation: () => true,
      };
    };
  },

  addInputRules() {
    return [
      new InputRule({
        find: /^```plantuml$/,
        handler: ({ state, range }) => {
          const { tr } = state;
          tr.replaceWith(
            range.from,
            range.to,
            this.type.create({ source: "@startuml\n\n@enduml" })
          );
        },
      }),
    ];
  },

  addCommands() {
    return {
      setPlantUMLBlock:
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
