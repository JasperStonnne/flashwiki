import { Node, mergeAttributes, InputRule, PasteRule } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import { inputRegex } from "./math-utils";
import katex from "katex";

// Types for tiptap-markdown serialization
interface MarkdownSerializerState {
  write(text: string): void;
  text(text: string, escape?: boolean): void;
  ensureNewLine(): void;
  closeBlock(node: ProseMirrorNode): void;
}

interface MarkdownItState {
  src: string;
  pos: number;
  push(type: string, tag: string, nesting: number): {
    content: string;
    attrSet(name: string, value: string): void;
  };
}

interface MarkdownItInstance {
  inline: {
    ruler: {
      after(name: string, ruleName: string, fn: (state: MarkdownItState, silent: boolean) => boolean): void;
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
    mathInline: {
      setMathInline: (attributes: { latex: string }) => ReturnType;
    };
  }
}

export const MathInline = Node.create({
  name: "mathInline",
  group: "inline",
  inline: true,
  atom: true,

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
        tag: 'span[data-type="math-inline"]',
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
      "span",
      mergeAttributes(HTMLAttributes, { "data-type": "math-inline" }),
    ];
  },

  addStorage() {
    return {
      markdown: {
        serialize(state: MarkdownSerializerState, node: ProseMirrorNode) {
          state.write(`$${node.attrs.latex}$`);
        },
        parse: {
          setup(markdownit: MarkdownItInstance) {
            // Add inline rule for $...$
            markdownit.inline.ruler.after("emphasis", "math_inline", (state: MarkdownItState, silent: boolean) => {
              if (state.src[state.pos] !== "$") return false;

              // Don't match $$ (block math)
              if (state.src[state.pos + 1] === "$") return false;

              const start = state.pos + 1;
              let end = start;

              // Find closing $
              while (end < state.src.length) {
                if (state.src[end] === "$" && state.src[end - 1] !== "\\") {
                  break;
                }
                end++;
              }

              if (end >= state.src.length) return false;

              const latex = state.src.slice(start, end);
              if (!latex) return false;

              if (!silent) {
                const token = state.push("math_inline", "span", 0);
                token.content = latex;
                token.attrSet("data-type", "math-inline");
                token.attrSet("data-latex", latex);
              }

              state.pos = end + 1;
              return true;
            });

            // Add renderer for math_inline token
            markdownit.renderer.rules.math_inline = (tokens: Array<{ content: string }>, idx: number) => {
              const token = tokens[idx];
              if (!token) return "";
              const latex = token.content;
              return `<span data-type="math-inline" data-latex="${markdownit.utils.escapeHtml(latex)}"></span>`;
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
    return ({ node, editor }) => {
      const dom = document.createElement("span");
      dom.classList.add("math-inline");
      dom.setAttribute("data-type", "math-inline");

      const renderMath = () => {
        const latex = node.attrs.latex as string;
        try {
          dom.innerHTML = katex.renderToString(latex, {
            throwOnError: false,
            displayMode: false,
          });
        } catch (error) {
          dom.innerHTML = `<span class="math-error" title="Invalid LaTeX">${latex}</span>`;
        }
      };

      renderMath();

      // Click to edit: convert back to $latex$ text for editing
      dom.addEventListener("click", () => {
        if (!editor.isEditable) return;

        const latex = node.attrs.latex as string;
        const pos = editor.view.posAtDOM(dom, 0);

        // Replace the mathInline node with editable $latex$ text
        // Place cursor after the latex content so user can edit
        editor
          .chain()
          .focus()
          .setNodeSelection(pos)
          .command(({ tr }) => {
            const text = `$${latex}$`;
            tr.replaceSelectionWith(editor.state.schema.text(text));
            return true;
          })
          // Place cursor inside the $...$ before the closing $
          .setTextSelection(pos + 1 + latex.length)
          .run();
      });

      return {
        dom,
        update: (updatedNode) => {
          if (updatedNode.type.name !== "mathInline") return false;
          node = updatedNode;
          renderMath();
          return true;
        },
      };
    };
  },

  addInputRules() {
    return [
      new InputRule({
        find: inputRegex,
        handler: ({ state, range, match }) => {
          const latex = match[1];
          const { tr } = state;
          tr.replaceWith(
            range.from,
            range.to,
            this.type.create({ latex })
          );
        },
      }),
    ];
  },

  addPasteRules() {
    return [
      new PasteRule({
        // Match $...$ but not across line breaks (block math uses $$)
        // Use /g flag for paste rules to find all matches in pasted text
        find: /(?:^|[^$])\$([^$\n]+)\$/g,
        handler: ({ state, range, match }) => {
          const latex = match[1];
          if (!latex) return;

          const { tr } = state;
          tr.replaceWith(range.from, range.to, this.type.create({ latex }));
        },
      }),
    ];
  },

  addCommands() {
    return {
      setMathInline:
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
