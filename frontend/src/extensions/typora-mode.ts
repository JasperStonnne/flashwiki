import { Extension } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import type { EditorState } from "@tiptap/pm/state";
import { Decoration, DecorationSet } from "@tiptap/pm/view";

const typoraModeKey = new PluginKey("typoraMode");

/**
 * Heading markers for block-level decorations.
 */
const HEADING_MARKERS: Record<number, string> = {
  1: "# ",
  2: "## ",
  3: "### ",
  4: "#### ",
  5: "##### ",
  6: "###### ",
};

/**
 * Compute decorations for block-level elements (headings).
 * Shows the Markdown heading marker (e.g., "## ") when cursor is inside a heading.
 */
function computeBlockDecorations(state: EditorState): DecorationSet {
  const decorations: Decoration[] = [];
  const { selection } = state;
  const $from = selection.$from;

  // Check if we're in a heading
  for (let depth = $from.depth; depth >= 0; depth--) {
    const node = $from.node(depth);

    if (node.type.name === "heading") {
      const level = node.attrs.level as number;
      const marker = HEADING_MARKERS[level];

      if (marker) {
        const nodeStart = $from.start(depth);

        // Add decoration to show the heading marker
        decorations.push(
          Decoration.widget(nodeStart, () => {
            const span = document.createElement("span");
            span.className = "live-md-heading-marker";
            span.textContent = marker;
            return span;
          })
        );
      }
      break;
    }
  }

  return DecorationSet.create(state.doc, decorations);
}

/**
 * Typora Mode Extension
 *
 * Shows Markdown heading markers (##) as decorations when the cursor
 * is inside a heading block.
 */
export const TyporaMode = Extension.create({
  name: "typoraMode",

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: typoraModeKey,

        props: {
          decorations(state) {
            try {
              return computeBlockDecorations(state);
            } catch {
              return DecorationSet.empty;
            }
          },
        },
      }),
    ];
  },
});
