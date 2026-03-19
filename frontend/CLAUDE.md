# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
pnpm install      # Install dependencies
pnpm dev          # Start dev server (Vite)
pnpm build        # TypeScript check + Vite production build
pnpm preview      # Preview production build
pnpm lint         # Run ESLint
```

No test framework is currently configured.

## Architecture

This is a Markdown editor built with TipTap 3.20.1 (ProseMirror wrapper) + React 19 + TypeScript 5.9.

### Core Data Flow

The editor uses **tiptap-markdown** for bidirectional Markdown conversion, operating directly on the ProseMirror document model (no HTML intermediate):

```
Rich Text Mode (TipTap)  <->  Markdown String (tiptap-markdown)  <->  Source Mode (CodeMirror 6)
```

- **Rich -> Source**: `editor.storage.markdown.getMarkdown()`
- **Source -> Rich**: `editor.commands.setContent(markdownString)`

### Key Files

| File | Purpose |
|------|---------|
| `src/lib/use-markdown-editor.ts` | Main hook that configures TipTap with all extensions; creates `localeRef` and passes it to extensions |
| `src/lib/slash-command-suggestion.tsx` | `createSlashCommandSuggestion(localeRef)` factory; fuzzy match filtering on title + searchTerms |
| `src/lib/word-segmentation.ts` | Intl.Segmenter-based word boundary detection for CJK |
| `src/lib/link-utils.ts` | Functions for inserting Markdown link/image/video syntax |
| `src/lib/export-pdf.ts` | PDF export: `exportToPdf(markdown, locale)` — POST to `/api/pdf`, blob download |
| `src/lib/pdf-config.ts` | `PDF_API_URL` from `VITE_PDF_API_URL` env var (empty = same origin) |
| `src/components/Editor/Editor.tsx` | Root component: wraps content with `LocaleProvider`, manages dual-view (rich text / source), renders `LanguageToggle`, `ExportButton`, and `ViewToggle` in toolbar |
| `src/components/Editor/ExportButton.tsx` | PDF export button; gets `locale` from `useLocale()`, calls `exportToPdf(markdown, locale)` |
| `src/components/Editor/SourceEditor.tsx` | CodeMirror 6 Markdown editor for source mode |
| `src/components/Editor/ContextMenu/` | Right-click context menu (renders via portal) |
| `src/components/Editor/LanguageToggle.tsx` | Button to toggle between English and Chinese |
| `src/i18n/` | i18n system: types, en/zh dictionaries, `LocaleProvider` + `useLocale()` hook |
| `src/extensions/index.ts` | Barrel export for all custom TipTap extensions |
| `docs/pdf-api.md` | API contract between frontend and backend PDF rendering service |

### i18n System

The editor supports English (`en`) and Chinese (`zh`) via a simple dictionary + React Context + mutable ref pattern:

- **`src/i18n/types.ts`**: `Locale`, `Dictionary`, `LocaleRef` types
- **`src/i18n/en.ts` / `zh.ts`**: Full dictionary objects (~120 strings each)
- **`src/i18n/context.tsx`**: `LocaleProvider` (controlled/uncontrolled), `useLocale()` hook returning `{ locale, setLocale, t }`

**React components** use `useLocale()` to read the current dictionary. **TipTap extensions** (non-React) receive a mutable `localeRef: LocaleRef` via `.configure()` that React keeps in sync — no editor re-creation needed on language change.

**External control**: The `Editor` component accepts optional `locale` and `onLocaleChange` props. If `locale` is not provided, state is managed internally with a toggle button.

### Extension Pattern

Custom extensions in `src/extensions/` follow this pattern for Markdown round-trip via tiptap-markdown:

```typescript
addStorage() {
  return {
    markdown: {
      serialize(state: MarkdownSerializerState, node: ProseMirrorNode) {
        // Output Markdown syntax
        state.write("```plantuml\n");
        state.text(node.attrs.source, false);
        state.ensureNewLine();
        state.write("```");
        state.closeBlock(node);
      },
      parse: {
        setup(markdownit: MarkdownItInstance) {
          // Configure markdown-it rules for parsing
        },
      },
    },
  };
}
```

Extensions without standard Markdown syntax serialize as raw HTML with `html: true` enabled in the Markdown extension config. Image uses raw HTML to preserve width; VideoBlock uses custom `@[title](url)` syntax with markdown-it block rule.

### Custom Node Types

| Node | Syntax | Description |
|------|--------|-------------|
| MathInline | `$...$` | KaTeX inline math, click converts to editable `$latex$` text, paste rule support |
| MathBlock | `$$...$$` | KaTeX block math with collapsed/expanded NodeView toggle; accepts `localeRef` for UI strings |
| PlantUMLBlock | ` ```plantuml ` | PlantUML diagrams with collapsed/expanded NodeView toggle, encodes to plantuml.com SVG; accepts `localeRef` for UI strings |
| Image | `![alt](url)` + raw HTML | Resizable image with InputRule, PasteRule, drag-and-drop, load fallback |
| VideoBlock | `@[title](url)` | Resizable video with InputRule, PasteRule, load fallback |
| SlashCommand | `/` trigger | Command palette with groups (text/list/block/media/advanced); items generated from `getSlashCommandItems(t)` |
| Highlight | `==...==` | Marker pen style highlighting |
| TyporaMode | - | Shows heading markers when cursor inside heading |
| TabHandler | - | Tab/Shift+Tab handling for code blocks, lists, and normal text; accepts `localeRef` for indent hint message |

### Collapsed/Expanded NodeView Pattern

MathBlock and PlantUMLBlock use a collapsed/expanded toggle pattern:

- **Collapsed state (default)**: Shows only rendered result (KaTeX formula / PlantUML SVG), textarea hidden
- **Expanded state (on select)**: Shows only editable textarea, preview hidden
- Uses ProseMirror `selectNode`/`deselectNode` callbacks for atom nodes
- Click on preview enters edit mode
- Escape key or blur exits edit mode
- Empty content starts in expanded mode immediately

**Keyboard navigation at boundaries** (both extensions):
- Backspace at position 0: deletes entire node
- Arrow Up on first line: exits to block before
- Arrow Down on last line: exits to block after
- Arrow Left at position 0: exits to block before
- Arrow Right at end: exits to block after

### Styling

- CSS variables defined in `src/styles/reset.css`
- Editor content styles in `src/styles/editor.css` (target `.tiptap` class)
- Code highlighting in `src/styles/hljs.css` (GitHub Light theme)
- KaTeX math styles in `src/styles/katex.css`
- Component styles use CSS Modules (`*.module.css`)
- Icons use Phosphor Icons (`@phosphor-icons/react`)

### UI Components

- **ContextMenu**: Right-click menu with nested submenus (Format, Paragraph, Insert), rendered via `createPortal` to `document.body`; labels from `t.contextMenu.*`
- **SlashMenu**: Command palette triggered by `/`, positioned via Tippy.js, custom scrollbar with auto-scroll on keyboard navigation; group labels and "No results" text from `t.slashMenu.*`; fuzzy search matches both English and Chinese terms regardless of display language
- **TableMenu**: Floating bubble menu for table operations when cursor is in table; labels from `t.tableMenu.*`
- **ViewToggle**: Switches between rich text and source mode; title attributes from `t.viewToggle.*`
- **LanguageToggle**: Button showing "EN" / "中" to switch locale; uses `useLocale()` to toggle
- **ExportButton**: PDF export button in toolbar; disabled in source mode; gets `locale` from `useLocale()` and calls `exportToPdf(markdown, locale)`; title text from `t.exportButton.*`

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| Ctrl+B | Bold |
| Ctrl+I | Italic |
| Ctrl+U | Underline |
| Ctrl+Shift+H | Highlight |
| Ctrl+K | Insert `[]()` link syntax |
| Ctrl+M | Toggle source/rich text view |
| Ctrl+Alt+1-6 | Heading 1-6 |
| Ctrl+Alt+0 | Paragraph |
| Tab | Indent (4 spaces in code/text, sink list item, next table cell) |
| Shift+Tab | Outdent (remove indent in code/text, lift list item, previous table cell) |

### Important Implementation Notes

- **Portal rendering**: ContextMenu uses `createPortal` to render outside ProseMirror DOM tree, preventing React/ProseMirror DOM conflicts that cause `insertBefore` errors
- **TyporaMode safety**: The decorations callback is wrapped in try-catch, returning `DecorationSet.empty` on error (cosmetic failure only)
- **Placeholder**: Uses TipTap Placeholder extension with per-node-type function; reads placeholder text from `localeRef.current.placeholder` for locale-aware headings (e.g. "Heading 1" / "标题 1")
- **Link/Image/Video insertion**: Inserts plain Markdown syntax `[]()` / `![]()` / `@[]()` rather than using dialogs; InputRules auto-convert to rich nodes when user types closing `)`
- **Media load fallback**: Image and VideoBlock NodeViews show `!`/`@` prefix with link when media fails to load
- **MathInline editing**: Click on rendered math converts it back to editable `$latex$` text; InputRule re-renders when user finishes editing
- **Context menu submenus**: Uses DOM traversal in `useLayoutEffect` to find trigger button and parent panel for positioning; flips horizontally/vertically when overflowing viewport
- **Slash command search**: Uses `fuzzyMatch()` (subsequence matching) against both `title` and `searchTerms` array; searchTerms always contain both English and Chinese terms so search works in both languages regardless of display locale
- **Locale ref pattern**: Extensions that display UI text (MathBlock, PlantUMLBlock, TabHandler) receive a `localeRef: LocaleRef` via `addOptions()` / `.configure()`. The ref is synced by `useEffect` in `useMarkdownEditor` — no editor re-creation needed
- **Drag handle tooltip**: Uses `content: attr(data-tooltip)` in CSS with `data-tooltip` attribute set from `t.dragHandle.dragToMove` for locale-aware tooltip

### PDF Export Architecture

PDF export is delegated entirely to a separate backend service. The frontend only sends markdown + locale and receives a PDF binary.

```
ExportButton (toolbar)
  → exportToPdf(markdown, locale)        src/lib/export-pdf.ts
  → POST ${PDF_API_URL}/api/pdf
  ← PDF blob
  → <a download> click
```

- **`PDF_API_URL`**: `src/lib/pdf-config.ts` reads `VITE_PDF_API_URL` env var. Empty string = same origin.
- **Dev proxy**: `vite.config.ts` proxies `/api` to `http://localhost:3001` so local frontend can reach a local backend without CORS issues.
- **API contract**: See `docs/pdf-api.md` for full request/response spec.
- **ExportButton** is disabled in source mode (markdown must be serialized from TipTap doc).
