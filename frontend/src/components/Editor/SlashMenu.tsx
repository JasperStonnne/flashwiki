import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useState,
  useCallback,
  useMemo,
  useRef,
} from "react";
import type { SlashCommandItem, SlashCommandGroup } from "@/extensions/slash-command";
import { useLocale } from "@/i18n";
import styles from "./SlashMenu.module.css";

export interface SlashMenuRef {
  onKeyDown: (props: { event: KeyboardEvent }) => boolean;
}

interface SlashMenuProps {
  items: SlashCommandItem[];
  command: (item: SlashCommandItem) => void;
}

const GROUP_ORDER: SlashCommandGroup[] = ["text", "list", "block", "media", "advanced"];

export const SlashMenu = forwardRef<SlashMenuRef, SlashMenuProps>(
  ({ items, command }, ref) => {
    const [selectedIndex, setSelectedIndex] = useState(0);
    const containerRef = useRef<HTMLDivElement>(null);
    const { t } = useLocale();

    const groupLabels = t.slashMenu.groups;

    // Group items while preserving order
    const groupedItems = useMemo(() => {
      const groups: { group: SlashCommandGroup; items: SlashCommandItem[] }[] = [];
      const seenGroups = new Set<SlashCommandGroup>();

      for (const item of items) {
        if (!seenGroups.has(item.group)) {
          seenGroups.add(item.group);
          groups.push({ group: item.group, items: [] });
        }
        const groupEntry = groups.find((g) => g.group === item.group);
        groupEntry?.items.push(item);
      }

      // Sort groups by predefined order
      groups.sort(
        (a, b) => GROUP_ORDER.indexOf(a.group) - GROUP_ORDER.indexOf(b.group)
      );

      return groups;
    }, [items]);

    const selectItem = useCallback(
      (index: number) => {
        const item = items[index];
        if (item) {
          command(item);
        }
      },
      [items, command]
    );

    const upHandler = useCallback(() => {
      setSelectedIndex((prevIndex) =>
        prevIndex <= 0 ? items.length - 1 : prevIndex - 1
      );
    }, [items.length]);

    const downHandler = useCallback(() => {
      setSelectedIndex((prevIndex) =>
        prevIndex >= items.length - 1 ? 0 : prevIndex + 1
      );
    }, [items.length]);

    const enterHandler = useCallback(() => {
      selectItem(selectedIndex);
    }, [selectItem, selectedIndex]);

    useEffect(() => {
      setSelectedIndex(0);
    }, [items]);

    // Auto-scroll selected item into view
    useEffect(() => {
      const selected = containerRef.current?.querySelector('[data-selected="true"]');
      selected?.scrollIntoView({ block: "nearest" });
    }, [selectedIndex]);

    useImperativeHandle(ref, () => ({
      onKeyDown: ({ event }) => {
        if (event.key === "ArrowUp") {
          upHandler();
          return true;
        }

        if (event.key === "ArrowDown") {
          downHandler();
          return true;
        }

        if (event.key === "Enter") {
          enterHandler();
          return true;
        }

        if (event.key === "Tab") {
          if (event.shiftKey) {
            upHandler();
          } else {
            downHandler();
          }
          return true;
        }

        return false;
      },
    }));

    if (items.length === 0) {
      return (
        <div className={styles.container}>
          <div className={styles.empty}>{t.slashMenu.noResults}</div>
        </div>
      );
    }

    // Create a flat index map for keyboard navigation
    let flatIndex = 0;

    return (
      <div ref={containerRef} className={styles.container}>
        {groupedItems.map((group, groupIndex) => (
          <div key={group.group}>
            {groupIndex > 0 && <div className={styles.groupDivider} />}
            <div className={styles.groupHeader}>{groupLabels[group.group]}</div>
            {group.items.map((item) => {
              const currentIndex = flatIndex++;
              return (
                <button
                  key={item.id}
                  type="button"
                  className={styles.item}
                  data-selected={currentIndex === selectedIndex}
                  onClick={() => selectItem(currentIndex)}
                  onMouseEnter={() => setSelectedIndex(currentIndex)}
                >
                  <span className={styles.icon}>{item.icon}</span>
                  <div className={styles.content}>
                    <span className={styles.title}>{item.title}</span>
                    <span className={styles.description}>{item.description}</span>
                  </div>
                </button>
              );
            })}
          </div>
        ))}
      </div>
    );
  }
);

SlashMenu.displayName = "SlashMenu";
