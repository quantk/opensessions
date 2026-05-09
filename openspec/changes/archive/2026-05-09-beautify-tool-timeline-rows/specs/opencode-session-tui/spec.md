## MODIFIED Requirements

### Requirement: Session preview and timeline
The system SHALL provide a source-aware session preview and a detailed timeline view containing user messages, assistant messages, compact glyph-led tool events, file references or tool targets, summaries, and bounded text previews for sessions from enabled sources. Timeline rows for tool, patch, file, and linked subagent/task parts SHALL avoid bracketed type labels such as `[tool]` and SHALL render status in the timeline with compact status symbols rather than status words.

#### Scenario: User opens a session
- **WHEN** the user selects a session from any enabled source and opens it
- **THEN** the system displays the appropriate chronological or branch-projected timeline for that session without loading unrelated sessions
- **AND** the timeline header identifies the session source when source identity is relevant to the view

#### Scenario: Tool-like rows use compact visual labels
- **WHEN** the user views a session timeline containing tool, patch, file, or linked subagent/task parts
- **THEN** those rows use compact glyph-led labels instead of bracketed textual labels
- **AND** tool status is shown with a compact symbol such as `✓`, `✗`, `…`, or `?` without displaying status words such as `completed` in the timeline row
- **AND** high-signal metadata such as tool name, title, file path or target, preview text, linked child-session information, and heavy flags remains visible when available

#### Scenario: Tool-like row actions are preserved
- **WHEN** the user focuses a compact tool-like timeline row
- **AND** the user opens it with `Enter` or `l`
- **THEN** the system keeps the existing detail-view or linked child-session navigation behavior for that row type
