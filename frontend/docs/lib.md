# Library Utilities

All utility functions are in `src/lib/`.

## Contents

- [useMarkdownEditor](#usemarkdowneditor)
- [createSlashCommandSuggestion](#createslashcommandsuggestion)
- [exportToPdf](#exporttopdf)
- [pdfConfig](#pdfconfig)
- [linkUtils](#linkutils)
- [wordSegmentation](#wordsegmentation)

---

## useMarkdownEditor

**File**: `use-markdown-editor.ts`

**Purpose**: React hook that initializes the TipTap editor with all extensions and i18n support

**Signature**:
```ts
function useMarkdownEditor(options?: UseMarkdownEditorOptions): Editor | null
```

**Options**:
```ts
interface UseMarkdownEditorOptions {
  /** Initial content (Markdown string) */
  content?: string;
  /** Placeholder text */
  placeholder?: string;
  /** Content update callback */
  onUpdate?: (markdown: string) => void;
  /** Current locale */
  locale?: Locale;
}
```

**Default Content**: Locale-aware. If `content` is not provided, uses `dictionaries[locale].editor.defaultContent`.

**Default Placeholder**: Locale-aware. If `placeholder` is not provided, reads from `localeRef.current.placeholder.default`.

**Locale Ref**: Creates a `localeRef` (mutable ref to the current `Dictionary`) and syncs it via `useEffect` when `locale` changes. Passes `localeRef` to MathBlock, PlantUMLBlock, and TabHandler via `.configure()`.

**Registered Extensions**:

| Extension | Configuration |
|-----------|---------------|
| StarterKit | heading: { levels: [1,2,3,4,5,6] }, codeBlock: false |
| Placeholder | Per-node-type placeholder function reading from localeRef |
| Markdown | html: true, transformPastedText: true, transformCopiedText: true |
| CustomKeymap | - |
| Underline | - |
| TaskList, TaskItem | - |
| Link | - |
| Image | - |
| Table, TableRow, TableHeader, TableCell | - |
| CodeBlockLowlight | - |
| MathInline | - |
| MathBlock | localeRef |
| PlantUMLBlock | localeRef |
| VideoBlock | - |
| Highlight | - |
| TyporaMode | - |
| TabHandler | localeRef |
| SlashCommand | suggestion: createSlashCommandSuggestion(localeRef) |

**Placeholder Configuration**:
```ts
Placeholder.configure({
  showOnlyWhenEditable: true,
  showOnlyCurrent: true,
  placeholder: ({ node }) => {
    if (node.type.name === "heading") {
      const t = localeRef.current;
      return `${t.placeholder.heading} ${node.attrs.level}`;
    }
    return placeholder ?? localeRef.current.placeholder.default;
  },
})
```

**editorProps**:
```ts
{
  attributes: {
    spellcheck: "false"
  }
}
```

**Usage**:
```tsx
import { useMarkdownEditor } from "@/lib/use-markdown-editor";

function MyEditor() {
  const editor = useMarkdownEditor({
    content: "# Hello World",
    locale: "en",
    onUpdate: (markdown) => console.log(markdown),
  });

  if (!editor) return null;

  return <EditorContent editor={editor} />;
}
```

---

## createSlashCommandSuggestion

**File**: `slash-command-suggestion.tsx`

**Purpose**: Factory function that creates Suggestion configuration for slash commands with locale-aware items and fuzzy matching

**Signature**:
```ts
function createSlashCommandSuggestion(
  localeRef: LocaleRef,
): Partial<SuggestionOptions<SlashCommandItem>>
```

### Items

Generates command list from `getSlashCommandItems(localeRef.current)` on each query, then filters with fuzzy matching:

```ts
items: ({ query }) => {
  const items = getSlashCommandItems(localeRef.current);
  if (!query) return items;
  return items.filter((item) => {
    if (fuzzyMatch(query, item.title)) return true;
    return item.searchTerms.some((term) => fuzzyMatch(query, term));
  });
}
```

### Fuzzy Match

Subsequence matching (case-insensitive):
```ts
function fuzzyMatch(query: string, text: string): boolean {
  let qi = 0;
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) qi++;
  }
  return qi === q.length;
}
```

### Render

Renders menu using ReactRenderer + Tippy.js.

**Tippy Configuration**:
| Option | Value |
|--------|-------|
| placement | bottom-start |
| trigger | manual |
| interactive | true |
| showOnCreate | true |
| appendTo | document.body |

**Rendering Flow**:
1. User types `/`
2. Suggestion API triggers `onStart`
3. Creates ReactRenderer to render SlashMenu component
4. Creates Tippy instance, uses `clientRect` for positioning
5. `onUpdate` updates filtered results and position on user input
6. Keyboard events passed to SlashMenu via `onKeyDown`
7. `onExit` cleans up when user selects command or presses Escape

**Dependencies**:
- `@tiptap/react` - ReactRenderer
- `tippy.js` - Popup positioning
- `@/components/Editor/SlashMenu` - Menu component
- `@/extensions/slash-command` - `getSlashCommandItems`, `SlashCommandItem`
- `@/i18n` - `LocaleRef`

---

## exportToPdf

**File**: `export-pdf.ts`

**Purpose**: Sends markdown content to the backend PDF rendering service and triggers a browser file download

**Signature**:
```ts
async function exportToPdf(markdown: string, locale: Locale): Promise<void>
```

**Flow**:
1. POSTs `{ markdown, locale }` as JSON to `${PDF_API_URL}/api/pdf`
2. On non-200 response, parses `{ error }` JSON and throws
3. Reads the response body as a `Blob`
4. Extracts filename from `Content-Disposition` header (falls back to `"Markdown Export.pdf"`)
5. Creates a temporary object URL, clicks a hidden `<a download>` element, revokes the URL

**Error handling**: Throws `Error` with the backend `error` message on HTTP error responses.

**Dependencies**:
- `@/lib/pdf-config` - `PDF_API_URL`
- `@/i18n` - `Locale` type

---

## pdfConfig

**File**: `pdf-config.ts`

**Purpose**: Exports `PDF_API_URL` derived from the `VITE_PDF_API_URL` Vite environment variable

```ts
export const PDF_API_URL = import.meta.env.VITE_PDF_API_URL ?? "";
```

**Behavior**:
- Empty string (`""`) → requests go to the same origin (works with Vite dev proxy and reverse proxy in prod)
- Set `VITE_PDF_API_URL=https://pdf.example.com` for cross-origin deployment

**Dev proxy**: `vite.config.ts` proxies `/api` → `http://localhost:3001` so that local development against a local backend requires no CORS configuration and no env var change.

---

**File**: `link-utils.ts`

**Purpose**: Plain Markdown link, image, and video insertion helpers

### insertMarkdownLink

**Signature**:
```ts
function insertMarkdownLink(editor: Editor): void
```

**Behavior**:
| Selection State | Action |
|-----------------|--------|
| No selection | Insert `[]()`, cursor inside `[]` |
| Has selection | Wrap as `[selectedText]()`, cursor inside `()` |

### insertMarkdownImage

**Signature**:
```ts
function insertMarkdownImage(editor: Editor): void
```

**Behavior**:
| Selection State | Action |
|-----------------|--------|
| No selection | Insert `![]()`, cursor inside `[]` for alt text |
| Has selection | Use selection as alt text: `![selectedText]()`, cursor inside `()` |

### insertMarkdownVideo

**Signature**:
```ts
function insertMarkdownVideo(editor: Editor): void
```

**Behavior**:
| Selection State | Action |
|-----------------|--------|
| No selection | Insert `@[]()`, cursor inside `[]` for title |
| Has selection | Use selection as title: `@[selectedText]()`, cursor inside `()` |

---

## wordSegmentation

**File**: `word-segmentation.ts`

**Purpose**: CJK-aware word boundary detection using `Intl.Segmenter`

### WordBoundary

```ts
interface WordBoundary {
  start: number;
  end: number;
  word: string;
}
```

### findWordAtPosition

**Signature**:
```ts
function findWordAtPosition(text: string, cursorPos: number): WordBoundary | null
```

**Purpose**: Find the word at a given cursor position

**Features**:
- Uses `Intl.Segmenter` for proper CJK word boundary detection
- Falls back to regex matching if Segmenter unavailable
- Returns `null` if cursor is not within a word

**Fallback Regex**: `/[\w\u4e00-\u9fff\u3400-\u4dbf]+/g`
- Matches ASCII word characters
- Matches CJK unified ideographs (U+4E00-U+9FFF)
- Matches CJK extension A (U+3400-U+4DBF)

### getWords

**Signature**:
```ts
function getWords(text: string): WordBoundary[]
```

**Purpose**: Get all words in a text string

**Features**:
- Returns array of all word segments with positions
- Uses same `Intl.Segmenter` / regex fallback strategy
