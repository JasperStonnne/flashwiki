# Architecture

## Layer Structure

```
┌─────────────────────────────────────────┐
│              App.tsx                    │  Entry point
├─────────────────────────────────────────┤
│            i18n Layer                  │  Locale context
│  (LocaleProvider, useLocale, dicts)    │
├─────────────────────────────────────────┤
│           Components Layer             │  UI components
│  (Editor, ContextMenu, TableMenu, ...) │
├─────────────────────────────────────────┤
│             Library Layer              │  Utilities
│  (useMarkdownEditor, exportPdf, ...)   │
├─────────────────────────────────────────┤
│           Extensions Layer             │  TipTap extensions
│  (Image, Table, MathBlock, ...)        │
├─────────────────────────────────────────┤
│         TipTap / ProseMirror           │  Editor core
└─────────────────────────────────────────┘
```

## Data Flow

```
User Input
    │
    ▼
TipTap Editor (ProseMirror)
    │
    ▼
Extensions process nodes/marks
    │
    ▼
tiptap-markdown serialization
    │
    ▼
onUpdate callback → returns Markdown string
```

## View Mode Toggle

Bidirectional conversion using tiptap-markdown directly on ProseMirror document model:

```
Rich Text Mode                    Source Mode (CodeMirror 6)
     │                                 │
     │  Ctrl+M or click toggle         │
     ▼                                 ▼
editor.storage.markdown ────→   Markdown string
    .getMarkdown()                     │
                                       │
editor.commands ←──────────────  User edits
    .setContent(markdown)
```

## Directory Responsibilities

| Directory | Purpose |
|-----------|---------|
| `src/i18n/` | Locale types, en/zh dictionaries, LocaleProvider + useLocale() |
| `src/extensions/` | TipTap Node and Extension definitions |
| `src/components/Editor/` | React UI components |
| `src/lib/` | Editor initialization hook, utilities |
| `src/styles/` | CSS variables, editor content styles |
| `src/types/` | TypeScript type definitions |

## Extension Types

TipTap extensions fall into three categories:

| Type | Description | Examples |
|------|-------------|----------|
| Node | Block or inline content | Image, Table, MathBlock, VideoBlock |
| Mark | Text decoration | Underline, Link, Highlight |
| Extension | Feature enhancement | SlashCommand, CustomKeymap, TyporaMode, TabHandler |

## Key Design Decisions

### 1. Markdown Bidirectional Conversion

Using tiptap-markdown extension:
- `html: true` allows raw HTML (for Image/VideoBlock width preservation)
- `transformPastedText: true` converts pasted text
- `transformCopiedText: true` converts copied text
- Custom extensions define serialization/parsing via `addStorage().markdown`

### 2. Resizable Media

Image and VideoBlock use custom NodeView:
- Container div with left/right resize-handle elements
- mousedown/mousemove/mouseup events calculate width percentage
- `tr.setNodeMarkup()` updates node attributes
- Width range: 10% - 100%
- Serializes as `<img width="50%">` / `<video width="50%">`

### 3. Slash Commands

Based on TipTap Suggestion API:
- Listens for `/` character trigger
- Tippy.js positions popup menu (placement: bottom-start)
- Keyboard navigation (ArrowUp/Down, Tab/Shift+Tab, Enter, Escape)
- Command groups: text, list, block, media, advanced
- Items generated from `getSlashCommandItems(t: Dictionary)` with localized titles/descriptions
- Each item has `searchTerms` containing both English and Chinese terms
- Fuzzy subsequence matching on both `title` and `searchTerms`

### 4. Dual View Synchronization

- Rich Text → Source: `editor.storage.markdown.getMarkdown()`
- Source → Rich Text: `editor.commands.setContent(markdownString)`
- Shortcut: `Ctrl+M` toggles view
- Source editor uses CodeMirror 6 with Markdown language support

### 5. Portal Rendering

ContextMenu renders via `createPortal(element, document.body)`:
- Avoids React/ProseMirror DOM conflicts
- ProseMirror mutates DOM directly outside React's knowledge
- Rendering inside ProseMirror-managed DOM causes `insertBefore` errors
- Portal ensures menu DOM is managed separately

### 6. Live Heading Markers

TyporaMode extension uses ProseMirror decorations:
- Adds widget decorations for Markdown heading markers (`#`, `##`, etc.)
- Shows markers when cursor is inside a heading block
- Hides markers when cursor moves elsewhere (Typora-style)
- Wrapped in try-catch returning `DecorationSet.empty` on error

### 7. Placeholder System

Uses TipTap Placeholder extension with per-node-type function reading from `localeRef`:
```typescript
Placeholder.configure({
  placeholder: ({ node }) => {
    if (node.type.name === "heading") {
      const t = localeRef.current;
      return `${t.placeholder.heading} ${node.attrs.level}`;
    }
    return placeholder ?? localeRef.current.placeholder.default;
  },
})
```

### 8. i18n Architecture

No i18n framework. Simple dictionary + React Context + mutable ref:

- **Two locale dictionaries** (`en.ts`, `zh.ts`) as plain `Dictionary` objects
- **React components** use `useLocale()` hook returning `{ locale, setLocale, t }`
- **TipTap extensions** (non-React) receive a `localeRef: LocaleRef` object via `.configure()`. React syncs it via `useEffect` in `useMarkdownEditor` — no editor re-creation needed
- **External control**: `Editor` accepts optional `locale` prop + `onLocaleChange` callback. If `locale` is not provided, `LocaleProvider` manages state internally
- **Slash command items** are regenerated from `getSlashCommandItems(localeRef.current)` on each query, so titles/descriptions update immediately on locale change
- **Search terms** always contain both English and Chinese terms, so slash search works in both languages regardless of display locale

### 9. PDF Export

PDF export is fully server-side. The frontend collects the markdown string and locale, then delegates all rendering to a separate backend service:

```
ExportButton (toolbar)
    ↓
exportToPdf(markdown, locale)     src/lib/export-pdf.ts
    ↓
POST ${PDF_API_URL}/api/pdf
Content-Type: application/json
{ markdown, locale }
    ↓ (backend: markdown-it → HTML → Playwright page.pdf())
← 200 OK
Content-Type: application/pdf
Content-Disposition: attachment; filename="<title>.pdf"
    ↓
Blob URL → <a download> click → file saved
```

- `PDF_API_URL` is read from `VITE_PDF_API_URL` env var (empty = same origin).
- Vite dev proxy maps `/api` → `http://localhost:3001` for local development.
- Full API contract is in [`docs/pdf-api.md`](./pdf-api.md).

### 10. Locale Ref Pattern for Extensions

Extensions that display UI text receive a mutable `localeRef`:

```typescript
// In the extension definition:
addOptions() {
  return { localeRef: null };
}

// In NodeView or plugin:
const text = this.options.localeRef?.current.mathBlock.clickToAdd ?? "Click to add formula";
```

Extensions using `localeRef`: MathBlock, PlantUMLBlock, TabHandler.
