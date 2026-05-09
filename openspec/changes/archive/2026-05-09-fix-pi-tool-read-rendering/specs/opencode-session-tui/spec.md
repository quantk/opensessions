## MODIFIED Requirements

### Requirement: Pretty part detail viewing
The system SHALL render opened safe tool, patch, and file parts as structured quick-reading detail views by default instead of raw JSON dumps, except linked `task` tool parts whose primary open action navigates to a child session timeline. For tool output values that use recognized structured text shapes, such as content block arrays with `text` fields, the pretty detail view SHALL display the safe text content rather than the JSON representation of those blocks.

#### Scenario: User opens a safe non-linked tool part
- **WHEN** the user opens a non-heavy tool part from the session timeline
- **AND** that tool part is not a `task` part linked to a child session
- **THEN** the system displays a structured detail view that highlights high-signal tool information such as tool name, status, title or description, input fields, and output preview when available

#### Scenario: User opens a tool with structured text output
- **WHEN** the user opens a safe non-linked tool part whose output is stored as recognized text-like structured data
- **THEN** the pretty detail view displays the output as readable text
- **AND** the pretty detail view does not show an indented JSON dump merely because the output was stored as content blocks

#### Scenario: User opens an unknown tool shape
- **WHEN** the user opens a safe tool part whose raw JSON shape is not recognized by a tool-specific renderer
- **AND** that tool part is not a `task` part linked to a child session
- **THEN** the system displays a generic structured detail view using available summary, input, output, and metadata fields without failing

#### Scenario: User opens a patch or file part
- **WHEN** the user opens a safe patch or file part from the session timeline
- **THEN** the system displays a structured detail view focused on file paths, titles, MIME or filename metadata, and safe text or diff summaries when available

#### Scenario: User views the timeline
- **WHEN** the user browses the session timeline
- **THEN** tool, patch, and file parts remain rendered as compact timeline cards without the pretty detail layout being shown inline
