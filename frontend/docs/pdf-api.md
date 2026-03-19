# PDF Rendering API Contract

Single source of truth for the interface between the frontend editor and the backend PDF rendering service.

## Endpoint

```
POST /api/pdf
Content-Type: application/json
```

### Request Body

```json
{
  "markdown": "string",
  "locale": "en" | "zh"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `markdown` | `string` | Yes | Raw markdown content from the editor |
| `locale` | `"en"` \| `"zh"` | Yes | Language for localized strings (video card labels) |

### Response (success)

```
Status: 200 OK
Content-Type: application/pdf
Content-Disposition: attachment; filename="<title>.pdf"
Body: PDF binary
```

The filename is derived from the first H1 in the markdown. Fallback: `"Markdown Export.pdf"`.

### Response (error)

```
Status: 400 | 500
Content-Type: application/json

{ "error": "string" }
```

## Localized Strings

The backend needs these strings for video QR card rendering:

| Key | `en` | `zh` |
|-----|------|------|
| `videoLabel` | `"Video"` | `"视频"` |
| `scanToWatch` | `"Scan to watch"` | `"扫码观看"` |

## PDF Specification

- **Page size**: A4
- **Margins**: top 20mm, right 18mm, bottom 20mm, left 18mm
- **No browser headers/footers**
- **Page numbers**: Centered at bottom, format `"1 / 3"`
- **Background colors preserved** (code blocks, highlights, blockquotes)

## Markdown Custom Syntax

The backend must support these custom syntaxes in addition to standard GFM:

| Syntax | Description |
|--------|-------------|
| `$...$` | KaTeX inline math |
| `$$...$$` | KaTeX block math (fenced on own lines) |
| `` ```plantuml `` | PlantUML diagram rendered as SVG from plantuml.com |
| `==text==` | Highlight (mark) |
| `@[title](url)` | Video block rendered as QR code card with link |
| Standard GFM | Tables, task lists, strikethrough, etc. |
| Code fences | Syntax highlighting via lowlight/highlight.js |
