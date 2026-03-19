# Markdown Web Editor

A rich text editor with bidirectional Markdown conversion, built on TipTap 3 (ProseMirror) + React 19 + TypeScript.

## Features

- **Dual-view editing**: Rich text mode with live formatting, or Markdown source mode (CodeMirror 6)
- **Markdown round-trip**: Direct conversion between ProseMirror document model and Markdown via tiptap-markdown
- **PDF export**: One-click export to PDF via a server-side rendering service (no print dialog)
- **i18n**: English and Chinese support with runtime locale switching; slash menu fuzzy-searches both languages regardless of display locale
- **Headings H1-H6**: Full heading level support
- **Text formatting**: Bold, italic, underline, strikethrough, inline code, highlight (`==text==`)
- **Lists**: Bullet lists, numbered lists, task lists with checkboxes
- **Tables**: Insert and edit tables with row/column operations
- **Code blocks**: Syntax highlighting via lowlight (common languages)
- **Math formulas**: Inline `$...$` and block `$$...$$` with KaTeX rendering
- **PlantUML diagrams**: Render UML diagrams via plantuml.com
- **Resizable media**: Images and videos with drag-to-resize (10-100% width)
- **Slash commands**: `/` triggered command palette for quick insertion with fuzzy search
- **Live heading markers**: Typora-style heading syntax shown when cursor is inside headings
- **Context menu**: Right-click menu with nested submenus for formatting, paragraph styles, inserting blocks, and clipboard operations

## Quick Start

```bash
pnpm install
pnpm dev          # Start development server
pnpm build        # TypeScript check + production build
pnpm preview      # Preview production build
pnpm lint         # Run ESLint
```

## PDF Export

PDF export sends the markdown to a separate backend service (`POST /api/pdf`) and downloads the result directly — no print dialog.

Configure the backend URL via environment variable:

```bash
# .env.local
VITE_PDF_API_URL=https://pdf.example.com   # cross-origin deployment
# or leave empty for same-origin / Vite proxy
```

In development, the Vite dev server proxies `/api` to `http://localhost:3001`. See [`docs/pdf-api.md`](./docs/pdf-api.md) for the full API contract.


## Usage

```tsx
import { Editor } from "@/components/Editor";

function App() {
  return (
    <Editor
      content="# Hello World"
      onUpdate={(markdown) => console.log(markdown)}
    />
  );
}
```

### Props

| Prop | Type | Description |
|------|------|-------------|
| `content` | `string` | Initial Markdown content |
| `placeholder` | `string` | Placeholder text for empty editor |
| `onUpdate` | `(markdown: string) => void` | Called with Markdown string on content change |
| `locale` | `"en" \| "zh"` | Controlled locale (optional; manages state internally if omitted) |
| `onLocaleChange` | `(locale: Locale) => void` | Called when locale changes |
| `className` | `string` | Additional CSS class for the wrapper |

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| Ctrl+B | Bold |
| Ctrl+I | Italic |
| Ctrl+U | Underline |
| Ctrl+Shift+H | Highlight |
| Ctrl+K | Insert Markdown link `[]()` |
| Ctrl+M | Toggle rich text / source view |
| Ctrl+Alt+1-6 | Heading 1-6 |
| Ctrl+Alt+0 | Paragraph |
| Tab | Indent (4 spaces in code/text, sink list item) |
| Shift+Tab | Outdent (remove indent in code/text, lift list item) |

## Project Structure

```
src/
├── i18n/                     # Internationalization
│   ├── types.ts                  # Locale, Dictionary, LocaleRef types
│   ├── en.ts                     # English dictionary
│   ├── zh.ts                     # Chinese dictionary
│   ├── context.tsx               # LocaleProvider + useLocale() hook
│   └── index.ts                  # Barrel export
├── components/Editor/        # React components
│   ├── Editor.tsx                # Main editor container (wraps with LocaleProvider)
│   ├── SourceEditor.tsx          # CodeMirror 6 source editor
│   ├── ContextMenu/              # Right-click context menu
│   ├── SlashMenu.tsx             # Slash command palette
│   ├── TableMenu.tsx             # Table operations menu
│   ├── ViewToggle.tsx            # Rich/source view toggle
│   ├── LanguageToggle.tsx        # EN/中 language toggle button
│   ├── ExportButton.tsx          # PDF export button
│   └── ResizeHandle.tsx          # Resize handle component
├── extensions/               # TipTap extensions
│   ├── math-inline.ts            # Inline math ($...$)
│   ├── math-block.ts             # Block math ($$...$$)
│   ├── plantuml-block.ts         # PlantUML diagrams
│   ├── image.ts                  # Resizable images
│   ├── video-block.ts            # Resizable video player
│   ├── highlight.ts              # Highlight mark (==...==)
│   ├── typora-mode.ts            # Typora-style heading markers
│   ├── tab-handler.ts            # Tab/Shift+Tab key handling
│   ├── slash-command.tsx         # Slash command extension + items
│   └── ...
├── lib/                      # Hooks and utilities
│   ├── use-markdown-editor.ts    # Editor initialization hook
│   ├── slash-command-suggestion.tsx # Suggestion config with fuzzy match
│   ├── link-utils.ts             # Link/image/video insertion helpers
│   ├── export-pdf.ts             # PDF export (fetch + blob download)
│   ├── pdf-config.ts             # PDF_API_URL from VITE_PDF_API_URL env var
│   └── word-segmentation.ts      # CJK word boundary detection
└── styles/                   # Global styles
    ├── reset.css                 # CSS variables and reset
    ├── editor.css                # TipTap content styles
    ├── hljs.css                  # Code highlighting
    └── katex.css                 # Math formula styles
```

## Technology Stack

| Dependency | Version | Purpose |
|------------|---------|---------|
| React | 19.2.4 | UI framework |
| TipTap | 3.20.1 | Editor core (ProseMirror wrapper) |
| tiptap-markdown | 0.9.0 | Bidirectional Markdown conversion |
| CodeMirror | 6.0.2 | Source mode editor |
| KaTeX | 0.16.37 | Math formula rendering |
| lowlight | 3.3.0 | Syntax highlighting |
| Vite | 7.3.1 | Build tool |
| TypeScript | 5.9.3 | Type system |

## Documentation

See [`docs/`](./docs/) for detailed technical documentation:

- [Architecture](./docs/architecture.md)
- [Extensions](./docs/extensions.md)
- [Components](./docs/components.md)
- [Library Utilities](./docs/lib.md)
- [Styling](./docs/styles.md)
- [PDF API Contract](./docs/pdf-api.md)

## License

Private project.
