import { BubbleMenu } from "@tiptap/react/menus";
import type { Editor } from "@tiptap/core";
import {
  RowsPlusTop,
  RowsPlusBottom,
  ColumnsPlusLeft,
  ColumnsPlusRight,
  Trash,
} from "@phosphor-icons/react";
import { useLocale } from "@/i18n";
import styles from "./Editor.module.css";

const TABLE_ICON_SIZE = 14;

interface TableMenuProps {
  editor: Editor;
}

export function TableMenu({ editor }: TableMenuProps) {
  const { t } = useLocale();

  return (
    <BubbleMenu
      editor={editor}
      shouldShow={({ editor: e }: { editor: Editor }) => e.isActive("table")}
      className={styles.tableMenu}
    >
      <button
        type="button"
        className={styles.tableMenuBtn}
        onClick={() => editor.chain().focus().addRowBefore().run()}
        title={t.tableMenu.addRowBefore}
      >
        <RowsPlusTop size={TABLE_ICON_SIZE} /> {t.tableMenu.row}
      </button>
      <button
        type="button"
        className={styles.tableMenuBtn}
        onClick={() => editor.chain().focus().addRowAfter().run()}
        title={t.tableMenu.addRowAfter}
      >
        <RowsPlusBottom size={TABLE_ICON_SIZE} /> {t.tableMenu.row}
      </button>
      <button
        type="button"
        className={styles.tableMenuBtn}
        onClick={() => editor.chain().focus().deleteRow().run()}
        title={t.tableMenu.deleteRow}
      >
        <Trash size={TABLE_ICON_SIZE} /> {t.tableMenu.row}
      </button>
      <span className={styles.tableMenuDivider} />
      <button
        type="button"
        className={styles.tableMenuBtn}
        onClick={() => editor.chain().focus().addColumnBefore().run()}
        title={t.tableMenu.addColBefore}
      >
        <ColumnsPlusLeft size={TABLE_ICON_SIZE} /> {t.tableMenu.col}
      </button>
      <button
        type="button"
        className={styles.tableMenuBtn}
        onClick={() => editor.chain().focus().addColumnAfter().run()}
        title={t.tableMenu.addColAfter}
      >
        <ColumnsPlusRight size={TABLE_ICON_SIZE} /> {t.tableMenu.col}
      </button>
      <button
        type="button"
        className={styles.tableMenuBtn}
        onClick={() => editor.chain().focus().deleteColumn().run()}
        title={t.tableMenu.deleteCol}
      >
        <Trash size={TABLE_ICON_SIZE} /> {t.tableMenu.col}
      </button>
      <span className={styles.tableMenuDivider} />
      <button
        type="button"
        className={styles.tableMenuBtn}
        onClick={() => editor.chain().focus().deleteTable().run()}
        title={t.tableMenu.deleteTable}
      >
        <Trash size={TABLE_ICON_SIZE} /> {t.tableMenu.table}
      </button>
    </BubbleMenu>
  );
}
