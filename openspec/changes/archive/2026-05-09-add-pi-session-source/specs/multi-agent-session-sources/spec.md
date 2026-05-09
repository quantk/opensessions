## ADDED Requirements

### Requirement: Source-aware session identity
The system SHALL assign every indexed project, session, message or entry, part, searchable document, scan metadata record, tag, and bookmark to a stable source kind so records from multiple local agent sources can coexist without ID collisions.

#### Scenario: OpenCode and Pi session IDs overlap
- **WHEN** OpenCode and Pi sources contain sessions with the same external identifier
- **THEN** the system stores and displays them as distinct sessions
- **AND** tags, bookmarks, timeline parts, raw details, and search documents remain associated with the correct source session

#### Scenario: Existing OpenCode index is migrated
- **WHEN** the application opens an existing opensession database created before source-aware indexing
- **THEN** existing indexed OpenCode records are treated as `opencode` source records
- **AND** existing OpenCode browsing and search behavior remains available

### Requirement: Combined source session browsing
The system SHALL display sessions from enabled local agent sources in the same session browsing experience and SHALL show a visible source indicator for each selectable session.

#### Scenario: Multiple sources are enabled
- **WHEN** the local index contains top-level OpenCode sessions and Pi sessions
- **THEN** the session list includes selectable rows from both sources
- **AND** each row indicates whether it came from OpenCode, Pi, or another supported source kind

#### Scenario: Session preview shows source metadata
- **WHEN** the user focuses a session from any supported source
- **THEN** the preview identifies the session source kind
- **AND** the preview continues to show available project, model, update time, message count, part count, token usage, tags, and bookmark metadata

### Requirement: Source-aware project grouping
The system SHALL group sessions from multiple sources by their normalized project or working directory while preserving source identity for each session row.

#### Scenario: Same project has sessions from multiple sources
- **WHEN** OpenCode and Pi sessions belong to the same working directory or project path
- **AND** project-grouped session-list mode is active
- **THEN** the system displays those sessions under the same project group when their normalized project key matches
- **AND** each session row still shows its source indicator

#### Scenario: Source has no project path
- **WHEN** an enabled source provides a session without project or working-directory metadata
- **THEN** the system displays that session in a distinct global or unknown-project group
- **AND** the group participates in activity ordering with other groups

### Requirement: Source selection configuration
The system SHALL allow users to run with the default enabled local sources and SHALL provide configuration or command-line input to restrict scanning to a selected source or to override a source root when supported by that source.

#### Scenario: Default startup scans available sources
- **WHEN** the user starts the TUI without source-selection overrides
- **THEN** the system scans enabled default local sources that can be discovered on the machine
- **AND** missing optional source roots do not prevent browsing sessions from other available sources

#### Scenario: User restricts source scanning
- **WHEN** the user starts the TUI with a source-selection override for a supported source kind
- **THEN** the system scans and displays sessions from that selected source scope instead of all default sources

### Requirement: Multi-source read-only access
The system MUST NOT create, update, delete, rename, archive, compact, fork, resume, or otherwise mutate files inside any indexed local agent source during scanning, browsing, searching, tagging, bookmarking, or raw/detail viewing.

#### Scenario: User browses multiple source sessions
- **WHEN** the user opens, searches, tags, bookmarks, previews, or views raw/details for OpenCode and Pi sessions
- **THEN** the system leaves OpenCode storage files unchanged
- **AND** the system leaves Pi session files unchanged
- **AND** only the opensession application database is modified for local metadata or indexing

### Requirement: Source-aware local metadata
The system SHALL store local tags and bookmarks in the opensession database with enough source-aware identity to keep metadata attached to the intended session across multiple sources.

#### Scenario: User bookmarks sessions from different sources
- **WHEN** the user bookmarks an OpenCode session and a Pi session
- **THEN** both bookmarks are stored in the opensession database
- **AND** neither source storage location is modified
- **AND** each bookmark appears only on the matching source session
