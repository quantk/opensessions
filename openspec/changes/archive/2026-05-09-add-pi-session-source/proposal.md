## Why

`opensession` is currently a read-only browser for OpenCode history, but the same local-first search and inspection workflow is useful for Pi agent sessions as well. Pi stores conversations as JSONL trees with branches, compactions, tool calls, reasoning, and labels; supporting them requires source-aware indexing and tree-aware session viewing rather than a one-off importer.

## What Changes

- Add Pi session discovery from the local Pi session directory and scan Pi JSONL session files read-only.
- Introduce source identity in the indexed data model so OpenCode, Pi, and future agent sources can coexist without ID collisions.
- Show OpenCode and Pi sessions together in the session list, with visible source badges and project/cwd grouping that works across sources.
- Normalize Pi session entries into browsable timelines, including user/assistant text, reasoning, tool calls, tool results, bash executions, compactions, branch summaries, labels, model changes, and session names.
- Add full Pi session tree/branch browsing so users can inspect branches rather than only the latest linear path.
- Keep search semantics aligned with existing behavior: session-list search finds matching sessions, while in-session search searches the current viewed timeline/branch.
- Preserve read-only access guarantees for both OpenCode storage and Pi session files.

## Capabilities

### New Capabilities
- `multi-agent-session-sources`: Defines source-aware indexing, source badges, combined session browsing, and read-only guarantees for multiple local agent session sources.
- `pi-session-source`: Defines Pi session discovery, JSONL parsing, tree/branch browsing, timeline rendering, and Pi-specific search behavior.

### Modified Capabilities
- `opencode-session-tui`: Existing session browsing, grouping, search, and timeline requirements must operate correctly when OpenCode sessions are shown alongside sessions from other source kinds.

## Impact

- Scanner/source architecture: add a Pi source adapter and a normalized source abstraction while preserving the existing OpenCode scanner behavior.
- SQLite schema/index: add source identity and tree-entry metadata needed for Pi branches and future sources.
- Repository interface and TUI: support combined source lists, source badges, Pi branch/tree navigation, and source-aware timeline/search queries.
- Configuration/CLI: add Pi session root discovery/override and source selection behavior as needed.
- Tests/fixtures: add Pi JSONL fixtures covering linear sessions, branches, compactions, tool calls/results, labels, and safe/heavy content guards.
