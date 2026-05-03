## 1. Detail State And Content Pipeline

- [x] 1.1 Add detail-view state for pretty vs raw mode and reset it when opening a new part.
- [x] 1.2 Preserve existing heavy, binary, skipped, and size guards before any raw payload rendering.
- [x] 1.3 Add tolerant parsing helpers for safe raw JSON that can extract tool state, input, output, file metadata, and patch metadata without failing on unknown shapes.
- [x] 1.4 Refactor raw display content generation so scrolling and `/` filtering operate on the currently selected pretty or raw content.

## 2. Pretty Detail Renderers

- [x] 2.1 Implement the default detail renderer that shows structured metadata for the opened part instead of `Raw JSON` by default.
- [x] 2.2 Implement a `bash` tool detail layout for command, workdir, description/title, status, and output preview.
- [x] 2.3 Implement generic tool detail rendering for unknown or irregular tool shapes using available summary, input, output, and metadata fields.
- [x] 2.4 Implement high-signal layouts for search/list and file-oriented tools where safe fields are available.
- [x] 2.5 Implement structured patch and file part detail layouts focused on paths, titles, MIME/filename metadata, and safe text or diff summaries.

## 3. Navigation And View Integration

- [x] 3.1 Add a detail-view hotkey to toggle raw JSON for safe loaded parts.
- [x] 3.2 Update the detail footer and headings so users can see whether they are in pretty or raw mode.
- [x] 3.3 Keep timeline compact card rendering and focus behavior unchanged.
- [x] 3.4 Ensure toggling raw mode reclamps scroll state and behaves correctly with active detail filtering.

## 4. Verification

- [x] 4.1 Update model tests so opening a safe tool part defaults to structured detail rather than raw JSON.
- [x] 4.2 Add tests for raw JSON toggle behavior in the detail view.
- [x] 4.3 Add tests for generic unknown tool detail rendering and patch/file detail rendering.
- [x] 4.4 Verify heavy or unsafe parts still show the guard and do not render raw payloads.
- [x] 4.5 Run `gofmt` on touched Go files and `go test ./...`.
