# Extensions

All extensions are in `src/extensions/` and exported via `index.ts`.

## Contents

- [Underline](#underline)
- [Link](#link)
- [Image](#image)
- [VideoBlock](#videoblock)
- [Table](#table)
- [TaskList / TaskItem](#tasklist--taskitem)
- [CodeBlockLowlight](#codeblocklowlight)
- [MathInline](#mathinline)
- [MathBlock](#mathblock)
- [PlantUMLBlock](#plantumlblock)
- [SlashCommand](#slashcommand)
- [CustomKeymap](#customkeymap)
- [Highlight](#highlight)
- [TyporaMode](#typoramode)
- [TabHandler](#tabhandler)

---

## Underline

**File**: `underline.ts`

**Type**: Mark

**Source**: Re-exports `@tiptap/extension-underline`

**Shortcut**: `Ctrl+U` (via CustomKeymap)

**Usage**:
```ts
editor.chain().focus().toggleUnderline().run()
```

---

## Link

**File**: `link.ts`

**Type**: Mark

**Source**: Configured `@tiptap/extension-link`

**Configuration**:
| Option | Value | Description |
|--------|-------|-------------|
| openOnClick | false | Don't navigate on click in edit mode |
| autolink | true | Auto-detect URLs |
| linkOnPaste | true | Convert pasted URLs to links |

**Usage**:
```ts
editor.chain().focus().setLink({ href: 'https://...' }).run()
editor.chain().focus().unsetLink().run()
```

---

## Image

**File**: `image.ts`

**Type**: Node

**Attributes**:
| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| src | string | null | Image URL |
| alt | string | null | Alt text |
| title | string | null | Title |
| width | number | 100 | Width percentage (10-100) |

**NodeView**: Custom DOM structure
```html
<div class="resizable-image" style="width: {width}%">
  <div class="resize-handle resize-handle-left"></div>
  <img src="..." />
  <div class="resize-handle resize-handle-right"></div>
</div>
```

**Load Failure Fallback**: When image fails to load, displays `!` prefix with link:
```html
<div class="image-fallback">
  <span class="image-fallback-prefix">!</span>
  <a class="image-fallback-link" href="...">alt text or URL</a>
</div>
```

**InputRule**: Typing `![alt](url)` auto-converts to Image node when `)` is typed

**PasteRule**: Pasting text containing `![alt](url)` auto-converts to Image nodes

**Drag-and-Drop**: Dropping image files inserts `![filename]()` placeholder at drop position

**Markdown Serialization**: Raw HTML
```html
<img src="..." alt="..." title="..." width="50%">
```

**Usage**:
```ts
editor.chain().focus().setImage({ src: '...' }).run()
```

---

## VideoBlock

**File**: `video-block.ts`

**Type**: Node

**Markdown Syntax**: `@[title](url)`

**Attributes**:
| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| src | string | "" | Video URL |
| width | number | 100 | Width percentage |
| title | string | "" | Title |

**NodeView**: Custom DOM structure
```html
<div class="resizable-video" style="width: {width}%">
  <div class="resize-handle resize-handle-left"></div>
  <video class="video-block-player" controls src="..."></video>
  <div class="resize-handle resize-handle-right"></div>
</div>
```

**Load Failure Fallback**: When video fails to load, displays `@` prefix with link:
```html
<div class="video-fallback">
  <span class="video-fallback-prefix">@</span>
  <a class="video-fallback-link" href="...">title or URL</a>
</div>
```

**InputRule**: Typing `@[title](url)` auto-converts to VideoBlock node when `)` is typed

**PasteRule**: Pasting text containing `@[title](url)` auto-converts to VideoBlock nodes

**Markdown Serialization**:
```
@[My Video](https://example.com/video.mp4)
```

**Usage**:
```ts
editor.commands.insertContent({
  type: 'videoBlock',
  attrs: { src: '...' }
})
```

---

## Table

**File**: `table.ts`

**Type**: Node (Table, TableRow, TableHeader, TableCell)

**Source**: Configured `@tiptap/extension-table` family

**Configuration**:
| Option | Value | Description |
|--------|-------|-------------|
| resizable | false | Column resize disabled |

**Exports**:
- `Table` - Table container
- `TableRow` - Row
- `TableHeader` - Header cell
- `TableCell` - Body cell

**Usage**:
```ts
editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
editor.chain().focus().addColumnAfter().run()
editor.chain().focus().addRowBefore().run()
editor.chain().focus().deleteRow().run()
editor.chain().focus().deleteColumn().run()
editor.chain().focus().deleteTable().run()
```

---

## TaskList / TaskItem

**File**: `task-list.ts`

**Type**: Node

**Source**: Configured `@tiptap/extension-task-list` and `@tiptap/extension-task-item`

**Configuration**:
- TaskItem: `nested: true` - supports nesting

**Usage**:
```ts
editor.chain().focus().toggleTaskList().run()
```

---

## CodeBlockLowlight

**File**: `code-block.ts`

**Type**: Node

**Source**: Configured `@tiptap/extension-code-block-lowlight`

**Dependency**: lowlight with `common` language pack

**Usage**:
```ts
editor.chain().focus().toggleCodeBlock().run()
```

---

## MathInline

**File**: `math-inline.ts`

**Type**: Node (inline)

**Markdown Syntax**: `$LaTeX$`

**Attributes**:
| Attribute | Type | Description |
|-----------|------|-------------|
| latex | string | LaTeX source |

**NodeView**:
- KaTeX rendered display
- Click to edit: converts node back to `$latex$` text for inline editing
- When user finishes editing, InputRule re-renders to math node

**InputRule**: Typing `$...$` auto-converts (regex from `math-utils.ts`)

**PasteRule**: Pasting text containing `$...$` auto-converts to math nodes

**Markdown Serialization**:
```
$E = mc^2$
```

---

## MathBlock

**File**: `math-block.ts`

**Type**: Node (block)

**Markdown Syntax**:
```
$$
LaTeX
$$
```

**Attributes**:
| Attribute | Type | Description |
|-----------|------|-------------|
| latex | string | LaTeX source |

**Options**:
| Option | Type | Description |
|--------|------|-------------|
| localeRef | `LocaleRef \| null` | Mutable ref to current locale dictionary for UI strings |

**Localized UI strings** (read from `localeRef.current.mathBlock.*`):
- Textarea placeholder (e.g. "Enter LaTeX..." / "输入 LaTeX...")
- Empty state text (e.g. "Click to add formula" / "点击添加公式")
- Error message prefix (e.g. "Invalid LaTeX:" / "无效的 LaTeX：")

**NodeView** (Collapsed/Expanded pattern):
- **Collapsed state (default)**: Shows only KaTeX preview, textarea hidden
- **Expanded state (on select)**: Shows only editable textarea, preview hidden
- Uses `selectNode`/`deselectNode` callbacks for atom nodes
- Click on preview enters edit mode
- Escape key or blur exits edit mode
- Empty content starts in expanded mode

**Keyboard navigation at boundaries**:
- Backspace at position 0: deletes entire node
- Arrow Up on first line: exits to block before
- Arrow Down on last line: exits to block after
- Arrow Left at position 0: exits to block before
- Arrow Right at end: exits to block after

**InputRule**: Typing `$$` auto-creates

---

## PlantUMLBlock

**File**: `plantuml-block.ts`

**Type**: Node

**Attributes**:
| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| source | string | `@startuml\n\n@enduml` | PlantUML source |

**Options**:
| Option | Type | Description |
|--------|------|-------------|
| localeRef | `LocaleRef \| null` | Mutable ref to current locale dictionary for UI strings |

**Localized UI strings** (read from `localeRef.current.plantumlBlock.*`):
- Empty state text (e.g. "Click to add diagram" / "点击添加图表")
- Loading indicator (e.g. "Rendering..." / "渲染中...")
- Render error (e.g. "Failed to render diagram" / "图表渲染失败")
- Syntax error (e.g. "Invalid PlantUML syntax" / "无效的 PlantUML 语法")

**Rendering**:
- Uses plantuml-encoder to encode
- Fetches `https://www.plantuml.com/plantuml/svg/{encoded}`
- Loading indicator shown during network request

**NodeView** (Collapsed/Expanded pattern):
- **Collapsed state (default)**: Shows only SVG preview, textarea hidden
- **Expanded state (on select)**: Shows only editable textarea, preview hidden
- Uses `selectNode`/`deselectNode` callbacks for atom nodes
- Click on preview enters edit mode
- Escape key or blur exits edit mode
- Empty/default content starts in expanded mode
- Placeholder text for empty diagrams

**Keyboard navigation at boundaries**:
- Backspace at position 0: deletes entire node
- Arrow Up on first line: exits to block before
- Arrow Down on last line: exits to block after
- Arrow Left at position 0: exits to block before
- Arrow Right at end: exits to block after

**InputRule**: Typing ` ```plantuml ` auto-creates

**Markdown Serialization**:
````
```plantuml
@startuml
Alice -> Bob: Hello
@enduml
```
````

---

## SlashCommand

**File**: `slash-command.tsx`

**Type**: Extension

**Trigger**: Type `/`

**Items**: Generated by `getSlashCommandItems(t: Dictionary)` — returns localized titles/descriptions based on current locale.

**Command Groups**:

| Group | Commands |
|-------|----------|
| text | Heading 1, Heading 2, Heading 3 |
| list | Bullet List, Numbered List, Task List |
| block | Blockquote, Code Block, Horizontal Rule |
| media | Table, Image, Video |
| advanced | Math Block, PlantUML |

**Search**: Each item has a `searchTerms` array containing both English and Chinese terms. Fuzzy subsequence matching is used on both `title` and `searchTerms`.

**Icons**: Phosphor Icons (20px)

**Exported Types**:
```ts
export type SlashCommandGroup = "text" | "list" | "block" | "media" | "advanced";

export interface SlashCommandItem {
  id: string;
  title: string;
  description: string;
  searchTerms: string[];
  icon: ReactNode;
  group: SlashCommandGroup;
  command: (props: { editor: Editor; range: Range }) => void;
}
```

---

## CustomKeymap

**File**: `custom-keymap.ts`

**Type**: Extension

**Shortcuts**:
| Shortcut | Action |
|----------|--------|
| Mod-Alt-1 | Heading 1 |
| Mod-Alt-2 | Heading 2 |
| Mod-Alt-3 | Heading 3 |
| Mod-Alt-4 | Heading 4 |
| Mod-Alt-5 | Heading 5 |
| Mod-Alt-6 | Heading 6 |
| Mod-Alt-0 | Paragraph |
| Mod-u | Underline |
| Mod-Shift-h | Highlight |
| Mod-k | Insert Markdown link `[]()` |

Note: `Mod` = Ctrl (Windows/Linux) or Cmd (Mac)

---

## Highlight

**File**: `highlight.ts`

**Type**: Mark

**Source**: Extended `@tiptap/extension-highlight`

**Markdown Syntax**: `==highlighted text==`

**Shortcut**: `Ctrl+Shift+H` (via CustomKeymap)

**Storage**: Defines `markdown.serialize` and `markdown.parse` for `==...==` syntax

**Usage**:
```ts
editor.chain().focus().toggleHighlight().run()
```

---

## TyporaMode

**File**: `typora-mode.ts`

**Type**: Extension

**Purpose**: Typora-style live heading markers

**Behavior**:
- Shows Markdown heading markers (`#`, `##`, etc.) when cursor is inside a heading
- Hides markers when cursor moves elsewhere
- Uses ProseMirror `Decoration.widget` for markers
- Decorations callback wrapped in try-catch, returns `DecorationSet.empty` on error

**Supported Markers**:
| Level | Marker |
|-------|--------|
| 1 | `# ` |
| 2 | `## ` |
| 3 | `### ` |
| 4 | `#### ` |
| 5 | `##### ` |
| 6 | `###### ` |

**CSS Classes**:
- `.live-md-heading-marker` - Heading marker styling

---

## TabHandler

**File**: `tab-handler.ts`

**Type**: Extension

**Purpose**: Handles Tab and Shift+Tab key behavior across different block contexts

**Options**:
| Option | Type | Description |
|--------|------|-------------|
| localeRef | `LocaleRef \| null` | Mutable ref to current locale dictionary for indent hint message |

**Behavior by context**:

| Context | Tab | Shift+Tab |
|---------|-----|-----------|
| Table | Defers to TipTap table extension (next cell) | Defers to TipTap table extension (previous cell) |
| Code block | Insert 4 spaces | Remove 4 leading spaces or 1 leading tab from current line |
| List (bullet, ordered, task) | Sink list item (indent) | Lift list item (outdent) |
| Normal text (paragraph, heading, etc.) | Insert 4 spaces | Remove 4 leading spaces or 1 leading tab from block start |

**Indent limit hint**: When a list item cannot be indented further, displays a tooltip near the cursor (text from `localeRef.current.tabHandler.cannotIndent`) that fades out after 1.5 seconds.

**Implementation**: Uses a ProseMirror plugin (`handleKeyDown`) rather than TipTap `addKeyboardShortcuts` to always `preventDefault` on Tab before routing to the appropriate handler.
