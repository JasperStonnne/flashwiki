# Components

All components are in `src/components/Editor/`.

## Contents

- [Editor](#editor)
- [ExportButton](#exportbutton)
- [ContextMenu](#contextmenu)
- [TableMenu](#tablemenu)
- [SlashMenu](#slashmenu)
- [SourceEditor](#sourceeditor)
- [ViewToggle](#viewtoggle)
- [LanguageToggle](#languagetoggle)
- [ResizeHandle](#resizehandle)

---

## Editor

**File**: `Editor.tsx`

**Purpose**: Main editor container, manages view mode toggle, wraps content with `LocaleProvider`

**Props**:
```ts
interface EditorProps extends UseMarkdownEditorOptions {
  className?: string;
  locale?: Locale;
  onLocaleChange?: (locale: Locale) => void;
}

// UseMarkdownEditorOptions:
interface UseMarkdownEditorOptions {
  content?: string;       // Initial Markdown content
  placeholder?: string;   // Placeholder text
  onUpdate?: (markdown: string) => void;  // Content update callback
  locale?: Locale;        // Current locale ("en" | "zh")
}
```

**Locale control**: If `locale` is provided, the Editor operates in controlled mode. If omitted, locale state is managed internally by `LocaleProvider` with a toggle button.

**State**:
| State | Type | Description |
|-------|------|-------------|
| viewMode | "richtext" \| "source" | Current view mode |
| markdownSource | string | Markdown source |

**Shortcut**: `Ctrl+M` / `Cmd+M` toggles view

**Structure**:
```
Editor (wraps with LocaleProvider)
└── EditorInner
    ├── toolbarRow
    │   ├── LanguageToggle
    │   ├── ExportButton
    │   └── ViewToggle
    └── editorArea
        ├── EditorContent (richtext mode)
        │   ├── TableMenu
        │   └── DragHandle (data-tooltip from t.dragHandle.dragToMove)
        └── SourceEditor (source mode)
    └── ContextMenu (rendered via portal)
```

**Dependencies**:
- `@tiptap/react` - EditorContent
- `@tiptap/extension-drag-handle-react` - DragHandle
- `@phosphor-icons/react` - DotsSixVertical icon
- `@/i18n` - LocaleProvider, useLocale

---

## ExportButton

**File**: `ExportButton.tsx`

**Purpose**: PDF export button in the toolbar; sends markdown + locale to the backend PDF service and triggers a file download

**Props**:
```ts
interface ExportButtonProps {
  getMarkdown: (() => string) | null;
  disabled?: boolean;
}
```

**Behavior**:
- Calls `getMarkdown()` to serialize the current TipTap document to Markdown
- Calls `exportToPdf(markdown, locale)` — POSTs to the backend `/api/pdf` endpoint
- Downloads the returned PDF blob via a temporary `<a download>` element
- Shows loading state (`exporting`) while the request is in-flight
- Disabled when: `getMarkdown` is null, `disabled` prop is true (source mode), or currently exporting

**i18n**: Button `title` attribute reads from `t.exportButton.title` / `t.exportButton.exporting` via `useLocale()`

**Icon**: `FilePdf` (Phosphor Icons, 16px)

**Dependencies**:
- `@/lib/export-pdf` - `exportToPdf`
- `@/i18n` - `useLocale`

---

## ContextMenu

**File**: `ContextMenu/ContextMenu.tsx`

**Purpose**: Right-click context menu with nested submenus for formatting, insertion, and clipboard operations

**Props**:
```ts
interface ContextMenuProps {
  editor: Editor | null;
}
```

**Rendering**: Uses `createPortal` to render to `document.body`, avoiding React/ProseMirror DOM conflicts

**i18n**: All menu labels read from `t.contextMenu.*` via `useLocale()`. Keyboard shortcuts remain hardcoded (not language-dependent).

**Menu Structure** (nested submenus):

| Item | Type | Description |
|------|------|-------------|
| Add Link | action | Insert `[]()` link syntax (Ctrl+K) |
| Search for "..." | action | Google search for selected text (only with selection) |
| Format | submenu | Bold, Italic, Strikethrough, Highlight, Code, Math, Clear formatting |
| Paragraph | submenu | Lists, Heading 1-6, Body, Quote |
| Insert | submenu | Image, Video, Table, HR, Code block, Math block, PlantUML |
| Cut | action | Cut to clipboard (disabled without selection) |
| Copy | action | Copy to clipboard (disabled without selection) |
| Paste | action | Paste from clipboard |
| Paste as plain text | action | Paste without formatting |
| Select all | action | Select all content |

**Format Submenu**:
- Bold (Ctrl+B), Italic (Ctrl+I), Strikethrough (Ctrl+Shift+S), Highlight (Ctrl+Shift+H)
- Code (Ctrl+E), Math
- Clear formatting
- Shows checkmark for active formats

**Paragraph Submenu**:
- Bullet list, Numbered list, Task list
- Heading 1-6 (Ctrl+Alt+1-6), Body (Ctrl+Alt+0)
- Quote
- Mutually exclusive checkmarks for heading/body

**Insert Submenu**:
- Image, Video
- Table, Horizontal rule
- Code block, Math block, PlantUML block

**Features**:
- Submenus open on hover with 150ms delay
- Auto-flip positioning when near viewport edges (horizontal and vertical)
- Keyboard navigation: Arrow keys, Enter, Escape

**State**:
| State | Type | Description |
|-------|------|-------------|
| isOpen | boolean | Menu visibility |
| clickPosition | {x, y} | Original click position |
| hasSelection | boolean | Whether text is selected |
| selectedText | string | Selected text content |

---

## TableMenu

**File**: `TableMenu.tsx`

**Purpose**: Table operations floating menu (BubbleMenu)

**Props**:
```ts
interface TableMenuProps {
  editor: Editor;
}
```

**Visibility**: Shows when cursor is in table (`editor.isActive("table")`)

**i18n**: Button titles and visible labels read from `t.tableMenu.*` via `useLocale()`.

**Operations**:
| Button | Icon | Action |
|--------|------|--------|
| Row | RowsPlusTop | Add row above |
| Row | RowsPlusBottom | Add row below |
| Row | Trash | Delete current row |
| Col | ColumnsPlusLeft | Add column left |
| Col | ColumnsPlusRight | Add column right |
| Col | Trash | Delete current column |
| Table | Trash | Delete entire table |

**Icon Size**: 14px

**Style**: Dark glass effect

---

## SlashMenu

**File**: `SlashMenu.tsx`

**Purpose**: Slash command dropdown menu

**Props**:
```ts
interface SlashMenuProps {
  items: SlashCommandItem[];
  command: (item: SlashCommandItem) => void;
}
```

**Ref Methods** (for Suggestion API):
```ts
interface SlashMenuRef {
  onKeyDown: (props: { event: KeyboardEvent }) => boolean;
}
```

**i18n**: Group labels read from `t.slashMenu.groups` and "No results" text from `t.slashMenu.noResults` via `useLocale()`.

**Keyboard Navigation**:
| Key | Action |
|-----|--------|
| ArrowUp | Previous item |
| ArrowDown | Next item |
| Tab | Next item |
| Shift+Tab | Previous item |
| Enter | Execute command |
| Escape | Close menu |

**Features**:
- Auto-scroll: selected item scrolls into view on keyboard navigation
- Custom scrollbar: thin dark scrollbar matching glass theme (Firefox + WebKit)

**Style**: Dark glass effect with group headers and dividers

---

## SourceEditor

**File**: `SourceEditor.tsx`

**Purpose**: CodeMirror 6 Markdown source editor

**Props**:
```ts
interface SourceEditorProps {
  value: string;
  onChange: (value: string) => void;
  className?: string;
}
```

**CodeMirror Configuration**:
- `lineNumbers()` - Line numbers
- `highlightActiveLine()` - Highlight current line
- `history()` - Undo/redo
- `markdown()` - Markdown language support
- `syntaxHighlighting()` - Syntax highlighting

**Style**:
- Font: `var(--font-mono)`
- Font size: 14px
- Line height: 1.6
- Background: `var(--color-bg-subtle)`

---

## ViewToggle

**File**: `ViewToggle.tsx`

**Purpose**: View mode toggle buttons

**Props**:
```ts
interface ViewToggleProps {
  mode: "richtext" | "source";
  onModeChange: (mode: ViewMode) => void;
}
```

**Exported Type**:
```ts
export type ViewMode = "richtext" | "source";
```

**i18n**: Title attributes read from `t.viewToggle.*` via `useLocale()`.

**Buttons**:
| Icon | Mode | Title (en) |
|------|------|-------|
| TextAlignLeft | richtext | Rich Text View |
| BracketsAngle | source | Markdown Source |

**Icon Size**: 16px

---

## LanguageToggle

**File**: `LanguageToggle.tsx`

**Purpose**: Button to toggle between English and Chinese locale

**i18n**: Uses `useLocale()` to read current locale and call `setLocale()`. Displays `t.langToggle.label` ("EN" or "中").

**Style**: Small button placed next to ViewToggle in the toolbar row.

---

## ResizeHandle

**File**: `ResizeHandle.tsx`

**Purpose**: Reusable resize wrapper component

**Props**:
```ts
interface ResizeHandleProps {
  onResize: (widthPercent: number) => void;
  children: React.ReactNode;
  initialWidth?: number;  // Default: 100
}
```

**Range**: 10% - 100%

**Interaction**:
- Left and right drag handles
- Changes body cursor to `ew-resize` during drag
- Multiplier factor 2 (bidirectional adjustment)
