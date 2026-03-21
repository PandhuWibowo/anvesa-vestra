# File Preview

The file preview panel lets you inspect files without downloading them. Click the **eye icon** on any file row, or press `Space` on a focused row, to open the preview.

For full details on the advanced preview features, read on.

---

## Fullscreen Mode

By default the preview opens as a **400px side panel** on the right. Click the **expand icon** in the panel header (or press `f`) to enter **fullscreen mode** — the panel expands to cover the entire viewport, giving you maximum space for reading code, viewing images, or inspecting spreadsheets.

Press `Escape` to exit fullscreen. Pressing `Escape` again closes the preview entirely.

---

## Image Preview

Images load directly from a signed URL and are displayed inline.

### Zoom Controls

A **toolbar** appears below the panel header with zoom controls:

| Control | Description |
|---|---|
| **−** button | Zoom out by 25% |
| **+** button | Zoom in by 25% |
| **Fit** button | Scale image to fit the panel width (default) |
| **1:1** button | Show image at actual pixel size |
| Percentage label | Shows current zoom level (e.g. `150%`) or `Fit` |

### Mouse Zoom

Hold `Ctrl` (or `Cmd` on macOS) and **scroll the mouse wheel** over the image to zoom in/out continuously. The zoom range is **10% – 500%**.

When zoomed beyond the panel size, the image container becomes scrollable so you can pan to any area.

### Supported Image Formats

`jpg` · `jpeg` · `png` · `gif` · `webp` · `svg` · `ico` · `bmp` · `tiff` · `avif` · `heic` · `apng`

---

## Text & Code Preview

Plain text, source code, config files, and JSON are rendered in a monospace `<pre>` block with advanced features.

### Progressive Loading

Text files are fetched up to **5 MB** (previously 100 KB). Content is rendered **progressively**:

1. The first **500 lines** are shown immediately.
2. As you **scroll toward the bottom**, the next 500 lines load automatically.
3. A **"Load more" bar** at the bottom shows exactly how many lines remain and lets you load the next chunk manually.
4. The **"Load all"** button in the toolbar loads every remaining line at once.
5. A **progress indicator** in the toolbar shows `500 / 12,345 lines` so you always know where you are.

This means you can read through an entire log file or source file of any length without truncation.

### Line Numbers

A **line number gutter** appears on the left side of the code block. Toggle it on/off with the **line numbers button** in the toolbar (the numbered-list icon).

Line numbers stay aligned with the content as you scroll and load more lines.

### Word Wrap

By default, long lines wrap to fit the panel width. Click the **word wrap button** in the toolbar (the wrap-arrow icon) to toggle between:

- **Wrap on** — lines break at the panel edge (default)
- **Wrap off** — lines scroll horizontally, preserving original formatting

### Supported Text Formats

The preview recognises a wide range of file types:

| Category | Extensions |
|---|---|
| **Web** | `js` `mjs` `cjs` `jsx` `ts` `tsx` `vue` `svelte` `html` `css` `scss` `sass` `less` |
| **Systems** | `c` `h` `cpp` `cc` `go` `rs` `swift` `java` `kt` `cs` `zig` `scala` |
| **Scripting** | `py` `rb` `php` `lua` `sh` `bash` `zsh` `bat` `ps1` `pl` `r` `jl` |
| **Data / Query** | `sql` `graphql` `proto` `json` `jsonl` `json5` `geojson` |
| **Config** | `yaml` `yml` `toml` `ini` `conf` `env` `editorconfig` `gitignore` `dockerfile` |
| **Markup** | `md` `markdown` `mdx` `xml` `rst` `tex` `org` `txt` `log` |

---

## JSON Preview

JSON files are **auto-formatted** with 2-space indentation for readability. All text/code features apply:

- Progressive loading (large JSON files load in 500-line chunks)
- Line numbers
- Word wrap toggle

If the JSON is malformed, the raw content is shown as-is.

---

## Markdown Preview

Markdown files (`.md`, `.markdown`, `.mdx`) are **rendered to HTML** — headings, lists, code blocks, links, and tables all display as formatted prose. The rendering uses `marked` with HTML sanitisation via `DOMPurify`.

---

## CSV & TSV Preview

CSV and TSV files are parsed and displayed as a **sortable table** with:

- A **sticky header row** that stays visible while scrolling
- Up to **2,000 rows** (increased from 200)
- Hover highlighting on rows
- Proper handling of quoted fields and escaped delimiters

A truncation notice appears when the file has more rows than the display limit.

---

## Excel / Spreadsheet Preview

Excel files (`.xlsx`, `.xls`, `.xlsm`, `.ods`) are parsed client-side using SheetJS:

- **Multi-sheet tabs** — click a tab to switch between worksheets
- Up to **2,000 rows per sheet** (increased from 500)
- Row numbers in the first column
- Sticky header row
- Truncation notice for large sheets

---

## Word Document Preview

Word files (`.docx`) are converted to HTML client-side using Mammoth. The preview renders:

- Headings, paragraphs, and lists
- Tables
- Inline images
- Links and basic formatting

---

## Video Preview

Video files are played inline with the browser's native HTML5 `<video>` player:

- Play/pause, seek, and volume controls
- Supported formats depend on the browser: `mp4`, `webm`, and `mov` work in most browsers

---

## Audio Preview

Audio files show a **music note icon** with the browser's native `<audio>` controls:

- Play/pause, seek, volume, and duration
- Supported: `mp3`, `wav`, `ogg`, `flac`, `aac`, `m4a`, `opus`

---

## PDF Preview

PDF files are rendered in an `<iframe>` using the browser's built-in PDF viewer. All standard PDF features (zoom, search, print) are available within the frame.

---

## Other Formats

Files that cannot be previewed inline show a descriptive message:

| Type | Message |
|---|---|
| Archives (`.zip`, `.tar.gz`, …) | "Download to extract contents." |
| Fonts (`.woff2`, `.ttf`, …) | "Download to install or preview in a font viewer." |
| Presentations (`.pptx`, `.keynote`) | "Download to open in your presentation app." |
| Unknown | "No preview available for .xyz files." |

---

## Preview Footer

The bottom bar of the preview panel always shows:

- **File size** (formatted: KB, MB, GB)
- **Language/type label** (e.g. "JavaScript", "PNG", "PDF")
- **Download button** — download the file directly from the preview

---

## Keyboard Shortcuts

| Key | Action |
|---|---|
| `Space` | Toggle preview for the focused file |
| `f` | Toggle fullscreen preview |
| `Escape` | Exit fullscreen → close preview → clear search |

---

## Tips

- **Large log files**: Open a multi-megabyte log, then use the browser's `Ctrl+F` to search within the loaded lines. Click "Load all" first to search the entire file.
- **Image comparison**: Use fullscreen mode with 1:1 zoom to inspect images at native resolution. Hold `Ctrl` and scroll to fine-tune the zoom.
- **Code review**: Enable line numbers and disable word wrap to read source code with the same formatting as your editor.
- **JSON inspection**: Large API responses or config dumps are auto-formatted. Use progressive loading to navigate to the section you need without loading the entire file.
