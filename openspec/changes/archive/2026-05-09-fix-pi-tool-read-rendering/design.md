## Context

Pi JSONL sessions represent a tool invocation as two related entries: an assistant `toolCall` block containing the tool name/input, followed by a `toolResult` message containing output and final error state. OpenCode storage represents tools as one part with a state object, so opensession users expect one compact tool row with a final status and readable detail view.

The current Pi scanner has a merge pass, but it depends on both call and result parts retaining parseable raw JSON. Large read outputs can mark the result part as heavy/skipped, which removes raw JSON and prevents the merge. Existing incremental scan metadata can also keep old split Pi parts in the local SQLite index until the source JSONL file changes.

There is also an incremental indexing data-loss bug for Pi projects: multiple Pi session files in the same working directory share one synthetic project ID, and the stored project `source_path` points at one of those session files. When the newest project source changes, dirty project propagation can mark sibling sessions dirty, the Pi refresh path deletes their messages, and unchanged per-message source checks then skip reinserting them. The session rows remain, but branch lookup finds no leaf messages, so the UI shows `No timeline parts` for all but the refreshed session.

Pretty tool detail rendering currently treats non-string output values as JSON. Pi read results commonly store output as content blocks, e.g. `[{"type":"text","text":"..."}]`, so detail output can become a JSON dump rather than file text.

## Goals / Non-Goals

**Goals:**

- Make each Pi tool call/result pair render as a single tool timeline part when the result references a known call ID.
- Preserve final tool status, input summary, output preview, and safe searchable text on the merged tool part.
- Keep associated Pi tool result messages from adding separate timeline rows, including when output is too large to keep as raw JSON.
- Show structured text-like tool outputs as readable text in pretty detail views for Pi and other sources.
- Refresh previously indexed Pi sessions whose rows were produced by the old parser behavior.
- Preserve messages, parts, search documents, and branch leaves for unchanged Pi sibling sessions during incremental scans.
- Keep source storages read-only and keep all heavy/binary safety guards.

**Non-Goals:**

- Reconstruct unrelated or missing tool results that have no tool call ID or no matching call in the same Pi session.
- Display full heavy read outputs in timeline or detail when they exceed existing safety limits.
- Change keyboard navigation, raw JSON toggling, or OpenCode source parsing semantics beyond detail output normalization.
- Redesign the source/project model beyond what is needed to stop dirty project propagation from deleting unchanged Pi timelines.
- Add new external dependencies.

## Decisions

1. **Merge by Pi `toolCallId` before final indexing, not by UI filtering.**
   - The scanner should build/retain enough metadata from both call and result entries to update the original call part with final status and output summary, then suppress the matched result part.
   - Rationale: search, raw/detail, counts, and timelines should all agree that this is one semantic tool operation.
   - Alternative considered: hide duplicate rows in `internal/tui`; rejected because stale searchable documents and part counts would still reflect split lifecycle rows.

2. **Allow result-to-call merging even when result raw JSON is skipped.**
   - The merge pass should identify result parts by stable structured fields (tool call ID, tool name, status, preview/index text, raw if available) rather than requiring `RawJSON` on the result part.
   - For safe bounded output, merged raw can include normalized `output`. For heavy output, merged raw should either omit full output or include only bounded preview metadata, while `Heavy/SkippedRaw` remains true as needed.
   - Alternative considered: always keep raw result payloads up to the full Pi file size; rejected because Pi JSONL files can be large and may contain unsafe or very large tool artifacts.

3. **Normalize text-like output for display at the TUI detail layer.**
   - Pretty detail rendering should convert content blocks and common `{text: ...}` shapes into joined text before falling back to indented JSON for unknown structured values.
   - Rationale: this benefits Pi immediately and prevents the same ugly output if OpenCode or future sources store structured text output.
   - Alternative considered: convert Pi output to a string only in scanner raw JSON; rejected as too source-specific and lossy for explicit raw views.

4. **Do not delete Pi session children unless that session will be fully reinserted.**
   - The index update path should avoid treating a dirty synthetic Pi project as proof that every sibling Pi session must have its messages deleted. If a Pi session source is unchanged and reused from the existing snapshot, its messages/parts/search documents must remain intact.
   - Practical options include limiting Pi dirty-session detection to the session file and its own messages/parts, excluding synthetic Pi project source changes from sibling session dirtiness, or making the Pi delete-and-reinsert path operate only when the full session subtree will be written.
   - Rationale: branch timelines depend on `messages`; deleting them while keeping the session row creates `No timeline parts` even though source data still exists.
   - Alternative considered: force a full Pi reparse on every startup; rejected because it defeats incremental indexing for large JSONL histories.

5. **Invalidate or version Pi scan metadata for this parser change.**
   - The implementation should ensure unchanged Pi JSONL files are reparsed once after the new scanner behavior lands. A small source/parser version marker or targeted Pi metadata invalidation is acceptable, provided source files are untouched.
   - Rationale: otherwise users with existing opensession databases keep split rows indefinitely and any already-empty Pi timelines may not be repaired.
   - Alternative considered: require users to delete their database; rejected as poor UX.

## Risks / Trade-offs

- **Risk: Heavy output loses raw detail visibility** → Keep bounded previews/search summaries and preserve explicit guard messaging for skipped raw payloads.
- **Risk: Merging reduces part count compared with previous indexes** → This is intended; counts should represent semantic timeline parts rather than lifecycle artifacts.
- **Risk: Parser-version invalidation causes one slower startup** → Limit invalidation to Pi source records and only when the stored version is older or absent.
- **Risk: Unchanged Pi timelines are deleted during partial refresh** → Ensure dirty detection and delete/reinsert decisions are session-scoped for Pi, and add a store-level regression test with two Pi sessions in the same project.
- **Risk: Unknown structured output is accidentally flattened incorrectly** → Restrict text normalization to recognized text block shapes and fall back to existing JSON formatting otherwise.

## Migration Plan

- Add tests that reproduce stale split Pi read rows from call/result pairs, structured output detail rendering, and unchanged Pi sibling sessions losing timeline rows after another session in the same project changes.
- Implement the scanner/detail changes, session-scoped Pi incremental refresh behavior, and a one-time Pi reparse trigger via application-owned metadata.
- On first startup after the change, affected Pi session files are reparsed or safely preserved in the opensession SQLite index; OpenCode storage and Pi session files remain unchanged.
- Rollback is safe: older binaries can still read indexed rows, though they may not preserve the new merged representation on future reparses.

## Open Questions

- Which metadata mechanism is simplest for parser-version invalidation within the existing schema: a dedicated schema/data version field, source-specific scan metadata versioning, or targeted deletion of Pi scan metadata during migration?
- Should synthetic Pi project records stop using a session file as `source_path`, or is it enough to keep project dirtiness from cascading into unchanged Pi session delete/reinsert decisions?
