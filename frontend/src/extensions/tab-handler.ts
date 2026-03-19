import { Extension } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import type { EditorView } from "@tiptap/pm/view";
import type { LocaleRef } from "@/i18n";

const tabHandlerKey = new PluginKey("tabHandler");

interface TabHandlerOptions {
  localeRef: LocaleRef | null;
}

export const TabHandler = Extension.create<TabHandlerOptions>({
  name: "tabHandler",

  addOptions() {
    return {
      localeRef: null,
    };
  },

  addProseMirrorPlugins() {
    const editor = this.editor;
    const localeRef = this.options.localeRef;

    return [
      new Plugin({
        key: tabHandlerKey,
        props: {
          handleKeyDown(view, event) {
            if (event.key !== "Tab") return false;

            // If the slash command popup is open, let the suggestion plugin handle Tab.
            const slashPopup = document.querySelector(".tippy-box");
            if (slashPopup) {
              event.preventDefault();
              return false;
            }

            event.preventDefault(); // ALWAYS prevent browser Tab behavior

            const { state } = view;
            const { selection } = state;

            // === CASE 1: Inside a table ===
            // Let TipTap's table extension handle Tab (next cell).
            // We already preventDefault'd, but we return false to let
            // the table extension's handler fire.
            if (editor.isActive("table")) {
              return false;
            }

            // === CASE 2: Inside a code block ===
            if (editor.isActive("codeBlock")) {
              if (event.shiftKey) {
                // Shift+Tab in code block: remove 4 spaces from line start
                // (or do nothing if no indent)
                const { $from } = selection;
                const textBefore = state.doc.textBetween(
                  $from.start(),
                  $from.pos
                );
                // Find the start of the current line
                const lastNewline = textBefore.lastIndexOf("\n");
                const lineStart = $from.start() + lastNewline + 1;
                const lineText = state.doc.textBetween(lineStart, $from.pos);

                if (lineText.startsWith("    ")) {
                  // Remove 4 spaces
                  const tr = state.tr.delete(lineStart, lineStart + 4);
                  view.dispatch(tr);
                } else if (lineText.startsWith("\t")) {
                  // Remove 1 tab
                  const tr = state.tr.delete(lineStart, lineStart + 1);
                  view.dispatch(tr);
                }
              } else {
                // Tab in code block: insert 4 spaces
                const tr = state.tr.insertText("    ", selection.from, selection.to);
                view.dispatch(tr);
              }
              return true;
            }

            // === CASE 3: Inside a list (bullet, ordered, task) ===
            if (
              editor.isActive("bulletList") ||
              editor.isActive("orderedList") ||
              editor.isActive("taskList")
            ) {
              if (event.shiftKey) {
                // Shift+Tab: lift list item
                const lifted = editor.chain().liftListItem("listItem").run();
                if (!lifted) {
                  // Try task item
                  editor.chain().liftListItem("taskItem").run();
                }
              } else {
                // Tab: sink list item
                const sunk = editor.chain().sinkListItem("listItem").run();
                if (!sunk) {
                  // Try task item
                  const sunkTask = editor.chain().sinkListItem("taskItem").run();
                  if (!sunkTask) {
                    // Cannot indent further — show a brief visual hint
                    showIndentLimitHint(view, localeRef?.current.tabHandler.cannotIndent ?? "Cannot indent further");
                  }
                }
              }
              return true;
            }

            // === CASE 4: Normal paragraph / heading / anything else ===
            if (event.shiftKey) {
              // Shift+Tab: remove 4 spaces from start of current line
              const { $from } = selection;
              const blockStart = $from.start();
              const blockText = state.doc.textBetween(blockStart, $from.end());

              if (blockText.startsWith("    ")) {
                const tr = state.tr.delete(blockStart, blockStart + 4);
                view.dispatch(tr);
              } else if (blockText.startsWith("\t")) {
                const tr = state.tr.delete(blockStart, blockStart + 1);
                view.dispatch(tr);
              }
            } else {
              // Tab: insert 4 spaces at cursor
              const tr = state.tr.insertText("    ", selection.from, selection.to);
              view.dispatch(tr);
            }
            return true;
          },
        },
      }),
    ];
  },
});

/**
 * Show a brief visual hint that the list item cannot be indented further.
 * Renders a small tooltip near the cursor that fades out after 1.5s.
 */
function showIndentLimitHint(view: EditorView, message: string): void {
  const { from } = view.state.selection;
  const coords = view.coordsAtPos(from);

  const hint = document.createElement("div");
  hint.textContent = message;
  hint.style.cssText = `
    position: fixed;
    left: ${coords.left}px;
    top: ${coords.top - 28}px;
    padding: 2px 8px;
    background: rgba(28, 28, 30, 0.9);
    color: rgba(255, 255, 255, 0.8);
    font-size: 11px;
    border-radius: 4px;
    pointer-events: none;
    z-index: 9999;
    opacity: 1;
    transition: opacity 0.3s ease;
  `;

  document.body.appendChild(hint);

  setTimeout(() => {
    hint.style.opacity = "0";
  }, 1000);

  setTimeout(() => {
    hint.remove();
  }, 1500);
}
