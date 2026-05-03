## ADDED Requirements

### Requirement: Incremental startup indexing
The system SHALL use application-owned scan metadata to avoid reparsing and rewriting unchanged OpenCode records during normal startup scans, and MUST keep OpenCode storage read-only while doing so.

#### Scenario: First scan indexes storage
- **WHEN** the application starts with no existing scan metadata for the configured OpenCode storage
- **THEN** the system scans available OpenCode records and persists session, message, part, search, and scan metadata in the application database
- **AND** the system leaves all OpenCode storage files unchanged

#### Scenario: Unchanged storage is skipped
- **WHEN** the application starts after a completed scan
- **AND** an OpenCode source record has the same source path, size, and modification time as the stored scan metadata
- **THEN** the system reuses the existing application-indexed record without reparsing the source payload
- **AND** the system does not rewrite unchanged index rows for that source record

#### Scenario: Changed storage is refreshed
- **WHEN** an OpenCode source record has changed size or modification time since the stored scan metadata
- **THEN** the system reparses that source record and refreshes its dependent session summary, timeline, search, and scan metadata in the application database

#### Scenario: Duplicate storage sources are not double processed
- **WHEN** the same OpenCode record is available from both JSON storage and `opencode.db`
- **THEN** the system selects one current source for indexing that record before doing expensive payload classification
- **AND** the system does not parse duplicate heavy payloads solely to discard them during deduplication

### Requirement: Responsive search on large indexes
The system SHALL keep session-list and in-session searches responsive on large local indexes by using indexed or bounded query paths and non-blocking TUI behavior for user-triggered searches.

#### Scenario: Session-list search remains interactive
- **WHEN** the user searches from the session list while the local index contains many sessions and searchable part documents
- **THEN** the system performs the search without freezing normal TUI rendering or input handling until the query completes
- **AND** the system displays matching top-level sessions ordered consistently with the session-list view

#### Scenario: In-session search remains interactive
- **WHEN** the user searches within a session timeline that contains many parts
- **THEN** the system performs the search without freezing normal TUI rendering or input handling until the query completes
- **AND** the system searches safe indexed/source content rather than rendered ANSI markdown output

#### Scenario: Stale search results are ignored
- **WHEN** multiple searches are requested in quick succession
- **AND** an older search completes after a newer search has been requested
- **THEN** the system ignores the stale older result and keeps the current search state consistent with the newest query

## MODIFIED Requirements

### Requirement: Responsive rendering for large sessions
The system SHALL keep list and timeline rendering, navigation, and scroll calculations bounded to visible content plus small cached layout metadata, and SHALL NOT concatenate raw transcripts, render full raw payloads, or rerender all assistant markdown parts during normal navigation.

#### Scenario: Session contains a very large tool part
- **WHEN** the user browses the session list or opens the session timeline
- **THEN** the UI remains based on summaries and bounded previews rather than rendering the full heavy payload

#### Scenario: Timeline contains many parts
- **WHEN** the user opens or navigates a timeline containing many messages and parts
- **THEN** the system renders only the visible timeline rows and the small context needed for focus and scroll behavior
- **AND** normal `j`, `k`, page, resize, and repaint operations do not rebuild the full transcript output each time

#### Scenario: Assistant markdown is cached for navigation
- **WHEN** the user navigates a timeline containing assistant markdown text parts
- **THEN** the system avoids reparsing and rerendering unchanged markdown content for every repaint or focus movement
- **AND** changing terminal width or markdown display mode invalidates only the affected rendered markdown layout

#### Scenario: Timeline text extraction is not repeated during repaint
- **WHEN** a safe text part has raw JSON available in the local index
- **THEN** the system does not repeatedly decode that raw JSON during normal timeline repaint or navigation
