# Styling

Global styles are in `src/styles/`, component styles use CSS Modules.

## Contents

- [reset.css](#resetcss)
- [editor.css](#editorcss)
- [hljs.css](#hljscss)
- [katex.css](#katexcss)
- [CSS Modules](#css-modules)

---

## reset.css

**File**: `src/styles/reset.css`

**Purpose**: CSS variables and global reset

### Design Tokens

**Spacing**:
| Variable | Value |
|----------|-------|
| --space-1 | 4px |
| --space-2 | 8px |
| --space-3 | 12px |
| --space-4 | 16px |
| --space-6 | 24px |
| --space-8 | 32px |

**Editor Area Colors**:
| Variable | Value | Description |
|----------|-------|-------------|
| --color-bg | #ffffff | Background |
| --color-bg-subtle | #f8f9fa | Secondary background |
| --color-border | #e2e4e8 | Border |
| --color-border-strong | #cfd1d6 | Strong border |
| --color-text | #1c1e21 | Primary text |
| --color-text-secondary | #656d76 | Secondary text |
| --color-text-placeholder | #a0a5ad | Placeholder text |
| --color-accent | #2563eb | Accent color |
| --color-accent-hover | #1d4ed8 | Accent hover |

**Toolbar Colors (Dark Theme)**:
| Variable | Value | Description |
|----------|-------|-------------|
| --color-toolbar-bg | #1c1c1e | Toolbar background |
| --color-toolbar-border | rgba(255, 255, 255, 0.08) | Toolbar border |
| --color-toolbar-text | rgba(255, 255, 255, 0.65) | Toolbar text |
| --color-toolbar-text-hover | rgba(255, 255, 255, 0.95) | Hover text |
| --color-toolbar-text-active | #ffffff | Active text |
| --color-toolbar-btn-hover-bg | rgba(255, 255, 255, 0.1) | Button hover background |
| --color-toolbar-btn-active-bg | var(--color-accent) | Button active background |

**Fonts**:
| Variable | Value |
|----------|-------|
| --font-sans | -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, ... |
| --font-mono | "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace |
| --font-size-base | 16px |
| --line-height-base | 1.625 |

**Border Radius**:
| Variable | Value |
|----------|-------|
| --radius-sm | 6px |
| --radius-md | 8px |
| --radius-lg | 10px |

**Shadows**:
| Variable | Value |
|----------|-------|
| --shadow-sm | 0 1px 2px rgba(0, 0, 0, 0.05) |
| --shadow-md | 0 4px 12px rgba(0, 0, 0, 0.08), 0 1px 3px rgba(0, 0, 0, 0.05) |
| --shadow-lg | 0 8px 24px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.06) |

---

## editor.css

**File**: `src/styles/editor.css`

**Purpose**: TipTap editor content area styles

**Selector Prefix**: `.tiptap`

### Base Elements

| Selector | Description |
|----------|-------------|
| `.tiptap` | Editor root container, padding: var(--space-6), min-height: 300px |
| `.tiptap h1/h2/h3` | Heading styles |
| `.tiptap p` | Paragraph, margin-bottom: 0.75em |
| `.tiptap strong/em/u/s` | Text decorations |
| `.tiptap code` | Inline code, gray background + border radius |
| `.tiptap pre` | Code block, left 3px accent border |
| `.tiptap blockquote` | Blockquote, left blue border + gradient background |
| `.tiptap ul/ol` | Lists |
| `.tiptap ul[data-type="taskList"]` | Task list, flex layout |
| `.tiptap hr` | Horizontal rule |
| `.tiptap a` | Links, blue accent color |
| `.tiptap img` | Images, max-width: 100% |
| `.tiptap table/th/td` | Tables, border-collapse |

### Special Block Styles

| Class | Description |
|-------|-------------|
| `.plantuml-block` | PlantUML container, green gradient background |
| `.plantuml-block-input` | PlantUML textarea input |
| `.plantuml-block-preview` | PlantUML preview area |
| `.plantuml-error` | PlantUML error message |
| `.video-block` | Video container |
| `.video-block-player` | Video player, black background |

### Resizable Media

| Class | Description |
|-------|-------------|
| `.resizable-image` | Image container, inline-block |
| `.resizable-video` | Video container, inline-block |
| `.resize-handle` | Resize handle, transparent, visible on hover |
| `.resize-handle-left` | Left handle |
| `.resize-handle-right` | Right handle |
| `.resize-handle::after` | Handle indicator bar, blue 4px wide |

### Media Fallbacks

| Class | Description |
|-------|-------------|
| `.image-fallback` | Image fallback container when load fails |
| `.image-fallback-prefix` | `!` prefix for failed images |
| `.image-fallback-link` | Link to original image URL |
| `.video-fallback` | Video fallback container when load fails |
| `.video-fallback-prefix` | `@` prefix for failed videos |
| `.video-fallback-link` | Link to original video URL |

### Placeholder

```css
.tiptap p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  color: var(--color-text-placeholder);
}
```

### Live Markdown Markers

| Class | Description |
|-------|-------------|
| `.live-md-heading-marker` | Heading markers (`#`, `##`, etc.) |

---

## hljs.css

**File**: `src/styles/hljs.css`

**Purpose**: Code syntax highlighting (GitHub Light theme)

**Class Prefix**: `.hljs-`

**Main Syntax Classes**:
| Class | Color | Purpose |
|-------|-------|---------|
| `.hljs-keyword` | #cf222e | Keywords |
| `.hljs-string` | #0a3069 | Strings |
| `.hljs-number` | #0550ae | Numbers |
| `.hljs-comment` | #6e7781 | Comments |
| `.hljs-function` | #8250df | Function names |
| `.hljs-variable` | #953800 | Variables |
| `.hljs-title` | #8250df | Titles/function definitions |

---

## katex.css

**File**: `src/styles/katex.css`

**Purpose**: KaTeX math formula styles

**Import**: `katex/dist/katex.min.css`

### Inline Formulas

```css
.tiptap .math-inline {
  display: inline;
  cursor: pointer;
  padding: 0 2px;
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}

.tiptap .math-inline:hover {
  background: var(--color-bg-subtle);
}
```

### Block Formulas

| Class | Description |
|-------|-------------|
| `.math-block` | Formula container, blue gradient background |
| `.math-block-input` | Input textarea, blue glow on focus |
| `.math-block-preview` | Preview area, centered display |
| `.math-error` | Error message, red background |

---

## CSS Modules

Component-level styles use CSS Modules (`*.module.css`).

| File | Components |
|------|------------|
| `Editor.module.css` | Editor wrapper, Toolbar row, TableMenu, DragHandle (tooltip via `attr(data-tooltip)`) |
| `ContextMenu.module.css` | ContextMenu panels and items |
| `SlashMenu.module.css` | SlashMenu container, items, scrollbar |
| `SourceEditor.module.css` | SourceEditor |
| `ViewToggle.module.css` | ViewToggle buttons |
| `LanguageToggle.module.css` | LanguageToggle button |
| `ResizeHandle.module.css` | ResizeHandle |
| `BubbleToolbar.module.css` | (exists but not currently imported by any component) |

### Editor.module.css Main Classes

| Class | Description |
|-------|-------------|
| `.wrapper` | Editor outer container, max-width: 800px, box-shadow |
| `.toolbarRow` | Toolbar row, dark background, flex with gap |
| `.editorArea` | Editor area, min-height: 400px |
| `.tableMenu` | Table menu, dark glass effect |
| `.tableMenuBtn` | Table menu button, gap: 4px |
| `.tableMenuDivider` | Table menu divider, white 12% opacity |
| `.dragHandle` | Drag handle, shows locale-aware tooltip on hover via `content: attr(data-tooltip)` |

### ContextMenu.module.css Main Classes

| Class | Description |
|-------|-------------|
| `.panel` | Menu panel, dark glass effect, fixed position, min-width 220px |
| `.submenuPanel` | Submenu panel, same style as panel |
| `.item` | Menu item, flex layout with gap 8px |
| `.itemCheck` | Checkmark gutter (16px width) |
| `.itemLabel` | Item label, flex: 1 |
| `.itemShortcut` | Keyboard shortcut display, monospace 11px |
| `.itemArrow` | Submenu arrow indicator |
| `.itemIcon` | Icon container |
| `.disabled` | Disabled item state, opacity 0.4 |
| `.separator` | Menu separator line |
| `.submenuWrapper` | Wrapper for submenu items, position: relative |

### SlashMenu.module.css Main Classes

| Class | Description |
|-------|-------------|
| `.container` | Menu container, dark glass effect, custom scrollbar |
| `.groupHeader` | Group header, 11px uppercase |
| `.groupDivider` | Group separator |
| `.item` | Menu item, flex layout |
| `.icon` | Icon container, 32x32px |
| `.title` | Title, 14px white |
| `.description` | Description, 12px semi-transparent |
| `.empty` | Empty state message |

**Custom Scrollbar**:
- Firefox: `scrollbar-width: thin`, `scrollbar-color: rgba(255,255,255,0.15) transparent`
- WebKit: 6px width, transparent track, semi-transparent thumb
