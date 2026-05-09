## 1. Reproduce Current Behavior

- [x] 1.1 Add Pi scanner test coverage for a `read` tool call/result pair that must become one completed tool part with no standalone result part.
- [x] 1.2 Add Pi scanner test coverage for a matched tool result whose output is too large or raw-skipped but still suppresses the separate lifecycle row.
- [x] 1.3 Add TUI detail test coverage for tool output stored as content block arrays rendering as readable text in pretty detail mode.
- [x] 1.4 Add index/store regression coverage for two Pi sessions in the same project where refreshing one session does not remove timeline parts from the unchanged sibling session.

## 2. Pi Tool Merge Implementation

- [x] 2.1 Preserve tool call IDs and result call IDs in Pi part metadata or raw-safe helper structures so merging does not depend on result `RawJSON` availability.
- [x] 2.2 Update `mergeToolResults` to merge matched result status, preview/index text, timestamps, and safe raw output into the original call part.
- [x] 2.3 Ensure matched Pi tool result messages keep zero rendered parts after merging, including heavy/skipped output cases.
- [x] 2.4 Keep unmatched Pi tool results as bounded standalone tool parts with available status, tool name, preview, and guard metadata.

## 3. Pretty Detail Output Normalization

- [x] 3.1 Add a detail-layer helper that converts recognized text-like structured values, including `[{"type":"text","text":"..."}]`, into joined safe text.
- [x] 3.2 Use the helper for tool output sections in file, bash, search, and generic pretty detail renderers while retaining JSON fallback for unknown structures.
- [x] 3.3 Verify explicit raw JSON mode still shows stored raw payloads and safety guards are unchanged for skipped raw content.

## 4. Pi Incremental Index Preservation

- [x] 4.1 Update Pi dirty-session detection so synthetic project dirtiness does not cause unchanged sibling Pi sessions to have messages/parts deleted without reinsertion.
- [x] 4.2 Ensure Pi delete-and-reinsert behavior only runs for sessions whose full message/part subtree will be rewritten, or otherwise preserves reused subtree rows.
- [x] 4.3 Verify branch leaf lookup and `SessionTimeline` continue to return parts for unchanged Pi sibling sessions after another session in the same project refreshes.

## 5. Pi Reparse Migration

- [x] 5.1 Choose and implement a source-specific parser/index version or targeted Pi scan metadata invalidation mechanism.
- [x] 5.2 Ensure older Pi scan metadata causes affected Pi JSONL files to be reparsed once even when source path, size, and modification time are unchanged.
- [x] 5.3 Ensure the migration touches only the opensession SQLite database and never mutates Pi session files.

## 6. Validation

- [x] 6.1 Run `gofmt` on touched Go files.
- [x] 6.2 Run focused tests for Pi scanning, index/store incremental refresh, and TUI detail behavior.
- [x] 6.3 Run `go test ./...`.
