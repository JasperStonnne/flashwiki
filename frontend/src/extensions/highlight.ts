import Highlight from "@tiptap/extension-highlight";

// Types for tiptap-markdown serialization
interface MarkdownItState {
  src: string;
  pos: number;
  posMax: number;
  push(type: string, tag: string, nesting: number): {
    content: string;
    markup: string;
  };
}

interface MarkdownItInstance {
  inline: {
    ruler: {
      before(
        name: string,
        ruleName: string,
        fn: (state: MarkdownItState, silent: boolean) => boolean
      ): void;
    };
  };
  renderer: {
    rules: Record<
      string,
      (tokens: Array<{ content: string; markup: string }>, idx: number) => string
    >;
  };
}

/**
 * Highlight extension with ==text== Markdown syntax support.
 *
 * Uses tiptap-markdown for serialization/parsing.
 */
const ConfiguredHighlight = Highlight.extend({
  addStorage() {
    return {
      markdown: {
        serialize: {
          open: "==",
          close: "==",
          mixable: true,
          expelEnclosingWhitespace: true,
        },
        parse: {
          setup(markdownit: MarkdownItInstance) {
            // Add inline rule for ==...== (highlight syntax)
            markdownit.inline.ruler.before(
              "emphasis",
              "highlight",
              (state: MarkdownItState, silent: boolean): boolean => {
                const start = state.pos;
                const max = state.posMax;

                // Check for opening ==
                if (
                  start + 1 >= max ||
                  state.src.charAt(start) !== "=" ||
                  state.src.charAt(start + 1) !== "="
                ) {
                  return false;
                }

                // Skip opening ==
                let pos = start + 2;

                // Find closing ==
                while (pos + 1 < max) {
                  if (
                    state.src.charAt(pos) === "=" &&
                    state.src.charAt(pos + 1) === "="
                  ) {
                    // Found closing ==
                    if (!silent) {
                      const content = state.src.slice(start + 2, pos);

                      // Don't match empty content
                      if (content.length === 0) {
                        return false;
                      }

                      const tokenOpen = state.push("highlight_open", "mark", 1);
                      tokenOpen.markup = "==";

                      const tokenText = state.push("text", "", 0);
                      tokenText.content = content;

                      const tokenClose = state.push("highlight_close", "mark", -1);
                      tokenClose.markup = "==";
                    }

                    state.pos = pos + 2;
                    return true;
                  }
                  pos++;
                }

                return false;
              }
            );

            // Add renderer rules
            markdownit.renderer.rules.highlight_open = () => "<mark>";
            markdownit.renderer.rules.highlight_close = () => "</mark>";
          },
        },
      },
    };
  },
});

export { ConfiguredHighlight as Highlight };
