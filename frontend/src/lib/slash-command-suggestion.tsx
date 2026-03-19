import { ReactRenderer } from "@tiptap/react";
import tippy, { Instance as TippyInstance } from "tippy.js";
import type { SuggestionOptions, SuggestionProps } from "@tiptap/suggestion";
import {
  SlashMenu,
  type SlashMenuRef,
} from "@/components/Editor/SlashMenu";
import {
  getSlashCommandItems,
  type SlashCommandItem,
} from "@/extensions/slash-command";
import type { LocaleRef } from "@/i18n";

function fuzzyMatch(query: string, text: string): boolean {
  let qi = 0;
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) qi++;
  }
  return qi === q.length;
}

export function createSlashCommandSuggestion(
  localeRef: LocaleRef,
): Partial<SuggestionOptions<SlashCommandItem>> {
  return {
    items: ({ query }) => {
      const items = getSlashCommandItems(localeRef.current);
      if (!query) return items;

      return items.filter((item) => {
        if (fuzzyMatch(query, item.title)) return true;
        return item.searchTerms.some((term) => fuzzyMatch(query, term));
      });
    },

    render: () => {
      let component: ReactRenderer<SlashMenuRef> | null = null;
      let popup: TippyInstance[] | null = null;

      return {
        onStart: (props: SuggestionProps<SlashCommandItem>) => {
          component = new ReactRenderer(SlashMenu, {
            props,
            editor: props.editor,
          });

          if (!props.clientRect) {
            return;
          }

          popup = tippy("body", {
            getReferenceClientRect: props.clientRect as () => DOMRect,
            appendTo: () => document.body,
            content: component.element,
            showOnCreate: true,
            interactive: true,
            trigger: "manual",
            placement: "bottom-start",
          });
        },

        onUpdate(props: SuggestionProps<SlashCommandItem>) {
          component?.updateProps(props);

          if (!props.clientRect) {
            return;
          }

          popup?.[0]?.setProps({
            getReferenceClientRect: props.clientRect as () => DOMRect,
          });
        },

        onKeyDown(props: { event: KeyboardEvent }) {
          if (props.event.key === "Escape") {
            popup?.[0]?.hide();
            return true;
          }

          return component?.ref?.onKeyDown(props) ?? false;
        },

        onExit() {
          popup?.[0]?.destroy();
          component?.destroy();
        },
      };
    },
  };
}
