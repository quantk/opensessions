## ADDED Requirements

### Requirement: Pretty part detail viewing
The system SHALL render opened safe tool, patch, and file parts as structured quick-reading detail views by default instead of raw JSON dumps.

#### Scenario: User opens a safe tool part
- **WHEN** the user opens a non-heavy tool part from the session timeline
- **THEN** the system displays a structured detail view that highlights high-signal tool information such as tool name, status, title or description, input fields, and output preview when available

#### Scenario: User opens an unknown tool shape
- **WHEN** the user opens a safe tool part whose raw JSON shape is not recognized by a tool-specific renderer
- **THEN** the system displays a generic structured detail view using available summary, input, output, and metadata fields without failing

#### Scenario: User opens a patch or file part
- **WHEN** the user opens a safe patch or file part from the session timeline
- **THEN** the system displays a structured detail view focused on file paths, titles, MIME or filename metadata, and safe text or diff summaries when available

#### Scenario: User views the timeline
- **WHEN** the user browses the session timeline
- **THEN** tool, patch, and file parts remain rendered as compact timeline cards without the pretty detail layout being shown inline

## MODIFIED Requirements

### Requirement: Explicit raw part viewing
The system SHALL allow raw part content to be shown only through an explicit user action from the part detail view, separate from normal session list, timeline rendering, and default pretty detail rendering.

#### Scenario: User toggles raw JSON for a safe part
- **WHEN** the user opens a safe part detail view and requests raw JSON with the detail-view raw toggle
- **THEN** the system displays the loaded raw content for that part in the detail view

#### Scenario: User opens a heavy part
- **WHEN** the user explicitly opens a heavy part
- **THEN** the system attempts to load that part in a detail view and indicates if the content is too large or unsafe to display normally
