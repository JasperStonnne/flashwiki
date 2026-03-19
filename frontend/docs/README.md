# Markdown Web Editor Documentation

Rich text editor built with TipTap 3.20.1 + React 19 + TypeScript 5.9.

## Contents

- [Architecture](./architecture.md)
- [Extensions](./extensions.md)
- [Components](./components.md)
- [Library Utilities](./lib.md)
- [Styling](./styles.md)
- [PDF API Contract](./pdf-api.md)

## Technology Stack

| Dependency | Version | Purpose |
|------------|---------|---------|
| React | 19.2.4 | UI framework |
| TipTap | 3.20.1 | Editor core (ProseMirror wrapper) |
| Vite | 7.3.1 | Build tool |
| TypeScript | 5.9.3 | Type system |
| tiptap-markdown | 0.9.0 | Bidirectional Markdown conversion |
| CodeMirror | 6.0.2 | Source mode editor |
| KaTeX | 0.16.37 | Math formula rendering |
| lowlight | 3.3.0 | Syntax highlighting |
| Phosphor Icons | 2.1.10 | Icon library |
| plantuml-encoder | 1.4.0 | PlantUML diagram encoding |
| tippy.js | 6.3.7 | Popup positioning |

## Project Structure

```
src/
├── i18n/                    # Internationalization
│   ├── types.ts                 # Locale, Dictionary, LocaleRef types
│   ├── en.ts                    # English dictionary
│   ├── zh.ts                    # Chinese dictionary
│   ├── context.tsx              # LocaleProvider + useLocale() hook
│   └── index.ts                 # Barrel export
├── components/Editor/       # React components
│   ├── Editor.tsx               # Main editor container
│   ├── ExportButton.tsx         # PDF export button
│   ├── ContextMenu/             # Right-click context menu
│   ├── TableMenu.tsx            # Table operations menu
│   ├── SlashMenu.tsx            # Slash command menu
│   ├── SourceEditor.tsx         # CodeMirror source editor
│   ├── ViewToggle.tsx           # View mode toggle
│   ├── LanguageToggle.tsx       # EN/中 language toggle button
│   └── ResizeHandle.tsx         # Resize handle component
├── extensions/              # TipTap extensions
│   ├── slash-command.tsx        # Slash command extension + items
│   ├── math-inline.ts          # Inline math formula
│   ├── math-block.ts           # Block math formula
│   ├── plantuml-block.ts       # PlantUML diagrams
│   ├── image.ts                # Resizable images
│   ├── video-block.ts          # Resizable video
│   ├── code-block.ts           # Syntax-highlighted code
│   ├── link.ts                 # Links
│   ├── underline.ts            # Underline mark
│   ├── highlight.ts            # Highlight mark (==...==)
│   ├── table.ts                # Tables
│   ├── task-list.ts            # Task lists
│   ├── custom-keymap.ts        # Custom keyboard shortcuts
│   ├── typora-mode.ts          # Typora-style heading markers
│   ├── tab-handler.ts          # Tab/Shift+Tab key handling
│   ├── math-utils.ts           # Shared math input rule regex
│   └── index.ts                # Barrel export
├── lib/                     # Utility functions
│   ├── use-markdown-editor.ts       # Editor initialization hook
│   ├── slash-command-suggestion.tsx  # Slash command Suggestion config
│   ├── export-pdf.ts                # PDF export (fetch + blob download)
│   ├── pdf-config.ts                # PDF_API_URL from VITE_PDF_API_URL
│   ├── link-utils.ts                # Link/image/video insertion helpers
│   └── word-segmentation.ts         # CJK word boundary detection
├── styles/                  # Global styles
│   ├── reset.css                # CSS variables and reset
│   ├── editor.css               # Editor content styles
│   ├── hljs.css                 # Code highlighting
│   └── katex.css                # Math formula styles
└── types/                   # Type definitions
    ├── editor.ts                # Editor-related types
    ├── strapi.ts                # Strapi CMS types
    └── plantuml-encoder.d.ts    # plantuml-encoder type declarations
```

## Quick Start

```bash
pnpm install
pnpm dev      # Development server
pnpm build    # Production build
pnpm lint     # ESLint check
```

## Features

| Feature | Description |
|---------|-------------|
| Rich text editing | Headings H1-H6, bold, italic, underline, strikethrough, inline code, highlight |
| Lists | Bullet lists, numbered lists, task lists (nested) |
| Tables | Insert, edit, row/column operations |
| Code blocks | Syntax highlighting via lowlight |
| Math formulas | Inline `$...$`, block `$$...$$` (KaTeX) |
| PlantUML | UML diagram rendering via plantuml.com |
| Media | Resizable images and videos (10-100% width) |
| Links | Plain Markdown syntax insertion |
| Slash commands | `/` triggered quick insertion with grouping and fuzzy search |
| Dual view | Rich text / Markdown source toggle (Ctrl+M) |
| Heading markers | Typora-style heading syntax shown when cursor is inside headings |
| Context menu | Right-click for formatting, insertion, and clipboard operations |
| PDF export | One-click export to PDF via backend service; no print dialog |
| i18n | English and Chinese with runtime toggle; slash menu fuzzy-searches both languages |
