## Context

The TUI currently keeps session timelines responsive by rendering bounded text previews: text extraction is cached, assistant markdown rendering is cached by width and mode, and `displayPartText` truncates timeline text to `maxTranscriptRunes`. Tool, patch, and file parts can already be opened into a scrollable detail view with optional raw JSON, but text and reasoning parts are focusable timeline rows whose open action is currently a no-op.

The change should preserve the timeline as a fast preview surface while adding an explicit, user-triggered inspection path for selected user and assistant messages. OpenCode storage remains read-only, and large or unsafe raw payloads must not be rendered unbounded.

## Goals / Non-Goals

**Goals:**

- Let users open focused text parts from the timeline with the same `Enter`/`l` action used for other openable rows.
- Show substantially more message text than the timeline preview in a scrollable detail view.
- Preserve assistant markdown rendering state when entering detail, while keeping user text as source text.
- Apply a separate pre-render safety cap for message detail so very large text cannot freeze the TUI.
- Reuse existing detail-view navigation, back, search, and raw JSON concepts where practical.

**Non-Goals:**

- Do not remove or raise the timeline preview limit as part of this change.
- Do not make raw OpenCode storage writable or add any storage mutation path.
- Do not attempt unbounded full-transcript rendering or full raw payload rendering.
- Do not change database schema, CLI flags, or indexing requirements unless implementation discovers an unavoidable gap.

## Decisions

1. Keep timeline previews bounded and add detail as the escape hatch.

   Timeline layout and markdown rendering stay based on `maxTranscriptRunes`. The detail view uses a distinct larger limit, for example `MaxMessageDetailRunes = 128 * 1024`, applied before wrapping or markdown rendering. This keeps normal navigation fast and bounds the worst-case explicit detail operation.

   Alternative considered: remove the timeline truncation and rely on viewport scrolling. This would make ordinary navigation and markdown layout proportional to large message size, which conflicts with the existing responsive rendering requirement.

2. Reuse the existing part detail mode for text parts, with text-specific rendering.

   Text and visible reasoning parts become openable. The existing detail view already has scroll, back, search, and raw toggle behavior, so the implementation should extend that path rather than introduce a parallel navigation mode. Text detail needs a dedicated renderer because generic pretty detail intentionally truncates text previews to `maxDetailPreviewRunes`.

   Alternative considered: create a new `ViewMessageDetail` mode. That would be clearer semantically but duplicates much of the current detail-view state and key handling for little benefit.

3. Use source text for safety decisions and markdown/source rendering.

   The detail view should extract message text from raw JSON when available, or from the read-only source path when explicit opening makes that safe and the payload is below the raw display guard. If neither source is safe or available, the view should show a guard plus the indexed preview rather than pretending the content is complete.

   Assistant text detail follows the current `renderMarkdown` setting at open time. User text detail remains source text even if it contains markdown syntax, matching timeline behavior. The `m` toggle can be available in text detail for assistant parts, and `R` can continue to expose raw JSON only when the raw payload passes existing guards.

4. Show truncation explicitly.

   When message detail hits its larger cap, append a clear marker such as `[message truncated at 128 KiB]`. This avoids implying that the detail view always contains the full source message.

## Risks / Trade-offs

- Markdown rendering for a 128 KiB assistant message can still be noticeable -> apply the cap before calling the markdown renderer and keep rendered rows cached by part, width, mode, and capped content.
- Reading source JSON for skipped text parts could accidentally broaden what explicit detail loads -> keep existing binary/unsafe checks, file-size guards, and read-only access; do not load source payloads over the guard.
- Reusing `ViewRawPart` may make naming less precise internally -> keep user-facing labels as `Message Detail` for text parts while limiting internal churn.
- Detail search over rendered markdown could produce surprising ANSI-related matches -> search/filter should operate on source/capped text for text details, not rendered ANSI output.

## Migration Plan

No data migration is required. The change is additive UI behavior behind an explicit open action. Rollback is removing the new text open/detail behavior while leaving existing timeline, tool detail, and raw JSON behavior unchanged.

## Open Questions

- The initial detail cap should be validated during implementation; `128 * 1024` runes is the proposed starting point unless tests or manual TUI use show it is too high.
