import { Node, mergeAttributes, InputRule } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import katex from "katex";
import type { LocaleRef } from "@/i18n";

// Types for tiptap-markdown serialization
interface MarkdownSerializerState {
  write(text: string): void;
  text(text: string, escape?: boolean): void;
  ensureNewLine(): void;
  closeBlock(node: ProseMirrorNode): void;
}

interface MarkdownItState {
  src: string;
  line: number;
  lineMax: number;
  bMarks: number[]; // byte offset of each line start
  eMarks: number[]; // byte offset of each line end
  tShift: number[]; // indent of first non-space char
  sCount: number[]; // indent level
  push(type: string, tag: string, nesting: number): {
    content: string;
    map: [number, number] | null;
    attrSet(name: string, value: string): void;
  };
  getLines(begin: number, end: number, indent: number, keepLastLF: boolean): string;
}

interface MarkdownItInstance {
  block: {
    ruler: {
      before(name: string, ruleName: string, fn: (state: MarkdownItState, startLine: number, endLine: number, silent: boolean) => boolean): void;
    };
  };
  renderer: {
    rules: Record<string, (tokens: Array<{ content: string }>, idx: number) => string>;
  };
  utils: {
    escapeHtml(str: string): string;
  };
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    mathBlock: {
      setMathBlock: (attributes: { latex: string }) => ReturnType;
    };
  }
}

interface MathBlockOptions {
  localeRef: LocaleRef | null;
}

export const MathBlock = Node.create<MathBlockOptions>({
  name: "mathBlock",
  group: "block",
  atom: true,

  addOptions() {
    return {
      localeRef: null,
    };
  },

  addAttributes() {
    return {
      latex: {
        default: "",
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: 'div[data-type="math-block"]',
        getAttrs: (element) => {
          const el = element as HTMLElement;
          return {
            latex: el.getAttribute("latex") || el.getAttribute("data-latex") || "",
          };
        },
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      "div",
      mergeAttributes(HTMLAttributes, { "data-type": "math-block" }),
    ];
  },

  addStorage() {
    return {
      markdown: {
        serialize(state: MarkdownSerializerState, node: ProseMirrorNode) {
          state.write("$$\n");
          state.text(node.attrs.latex as string, false);
          state.ensureNewLine();
          state.write("$$");
          state.closeBlock(node);
        },
        parse: {
          setup(markdownit: MarkdownItInstance) {
            // Add block rule for $$...$$ using correct markdown-it block rule API
            markdownit.block.ruler.before(
              "fence",
              "math_block",
              (
                state: MarkdownItState,
                startLine: number,
                endLine: number,
                silent: boolean
              ) => {
                // 1. Get the start position of the current line
                const bMarkStart = state.bMarks[startLine];
                const tShiftStart = state.tShift[startLine];
                const eMarkStart = state.eMarks[startLine];
                if (
                  bMarkStart === undefined ||
                  tShiftStart === undefined ||
                  eMarkStart === undefined
                ) {
                  return false;
                }

                const lineStart = bMarkStart + tShiftStart;
                const lineEnd = eMarkStart;
                const lineText = state.src.slice(lineStart, lineEnd).trim();

                // 2. Must start with exactly "$$" (optionally with trailing whitespace)
                if (lineText !== "$$") return false;

                // 3. Find closing "$$" on a subsequent line
                let closingLine = startLine + 1;
                let found = false;

                while (closingLine < endLine) {
                  const cBMark = state.bMarks[closingLine];
                  const cTShift = state.tShift[closingLine];
                  const cEMark = state.eMarks[closingLine];
                  if (
                    cBMark === undefined ||
                    cTShift === undefined ||
                    cEMark === undefined
                  ) {
                    closingLine++;
                    continue;
                  }

                  const cLineStart = cBMark + cTShift;
                  const cLineEnd = cEMark;
                  const cLineText = state.src.slice(cLineStart, cLineEnd).trim();

                  if (cLineText === "$$") {
                    found = true;
                    break;
                  }
                  closingLine++;
                }

                if (!found) return false;
                if (silent) return true;

                // 4. Extract content between opening and closing lines
                const closingBMark = state.bMarks[closingLine];
                if (closingBMark === undefined) return false;

                const contentStart = eMarkStart + 1; // after opening $$ newline
                const contentEnd = closingBMark; // before closing $$ line
                const content = state.src.slice(contentStart, contentEnd).trim();

                // 5. Create token
                const token = state.push("math_block", "div", 0);
                token.content = content;
                token.map = [startLine, closingLine + 1];
                token.attrSet("data-type", "math-block");
                token.attrSet("data-latex", content);

                // 6. Advance past closing line
                state.line = closingLine + 1;

                return true;
              }
            );

            // Add renderer for math_block token
            markdownit.renderer.rules.math_block = (
              tokens: Array<{ content: string }>,
              idx: number
            ) => {
              const token = tokens[idx];
              if (!token) return "";
              const latex = token.content;
              return `<div data-type="math-block" data-latex="${markdownit.utils.escapeHtml(latex)}"></div>\n`;
            };
          },
          updateDOM(element: HTMLElement) {
            // Transfer data-latex to latex attribute for TipTap parsing
            const latex = element.getAttribute("data-latex");
            if (latex) {
              element.setAttribute("latex", latex);
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
      dom.classList.add("math-block");
      dom.setAttribute("data-type", "math-block");

      // Create textarea (hidden by default in collapsed state)
      const textarea = document.createElement("textarea");
      textarea.classList.add("math-block-input");
      textarea.value = node.attrs.latex as string;
      textarea.placeholder = localeRef?.current.mathBlock.enterLatex ?? "Enter LaTeX...";
      textarea.rows = 3;

      // Create preview container (visible by default in collapsed state)
      const preview = document.createElement("div");
      preview.classList.add("math-block-preview");

      // State management
      let isEditing = false;
      let currentNode = node;

      const renderMath = (latex: string) => {
        if (!latex.trim()) {
          preview.innerHTML = `<span class="math-placeholder">${localeRef?.current.mathBlock.clickToAdd ?? "Click to add formula"}</span>`;
          preview.classList.remove("math-error");
          return;
        }
        try {
          preview.innerHTML = katex.renderToString(latex, {
            throwOnError: false,
            displayMode: true,
          });
          preview.classList.remove("math-error");
        } catch {
          preview.innerHTML = `<span class="math-error">${localeRef?.current.mathBlock.invalidLatex ?? "Invalid LaTeX:"} ${latex}</span>`;
          preview.classList.add("math-error");
        }
      };

      const autoResizeTextarea = () => {
        textarea.style.height = "auto";
        textarea.style.height = Math.max(60, textarea.scrollHeight) + "px";
      };

      const enterEditMode = () => {
        if (isEditing) return;
        isEditing = true;

        // Hide preview, show textarea
        preview.style.display = "none";
        textarea.style.display = "block";

        // Sync textarea value from current node attrs
        textarea.value = currentNode.attrs.latex as string;
        autoResizeTextarea();

        // Focus the textarea
        textarea.focus();
      };

      const exitEditMode = () => {
        if (!isEditing) return;

        // Save changes if value differs
        const newLatex = textarea.value;
        if (newLatex !== currentNode.attrs.latex) {
          if (typeof getPos === "function") {
            const pos = getPos();
            if (pos !== undefined) {
              editor.view.dispatch(
                editor.view.state.tr.setNodeMarkup(pos, undefined, { latex: newLatex })
              );
            }
          }
        }

        isEditing = false;

        // Hide textarea, show preview
        textarea.style.display = "none";
        preview.style.display = "block";

        // Re-render preview with current value
        renderMath(textarea.value);
      };

      // Initial render
      renderMath(node.attrs.latex as string);

      // If content is empty, start in expanded mode
      if (!node.attrs.latex || !(node.attrs.latex as string).trim()) {
        // Start expanded
        preview.style.display = "none";
        textarea.style.display = "block";
        isEditing = true;
        // Focus will be called by selectNode
      } else {
        // Start collapsed
        preview.style.display = "block";
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
          if (updatedNode.type.name !== "mathBlock") return false;
          currentNode = updatedNode;
          if (!isEditing) {
            textarea.value = updatedNode.attrs.latex as string;
            renderMath(updatedNode.attrs.latex as string);
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
        find: /^\$\$$/,
        handler: ({ state, range }) => {
          const { tr } = state;
          tr.replaceWith(range.from, range.to, this.type.create({ latex: "" }));
        },
      }),
    ];
  },

  addCommands() {
    return {
      setMathBlock:
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
