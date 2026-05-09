## ADDED Requirements

### Requirement: Immediate startup with cached data
The system SHALL start the TUI without waiting for source scanning or indexing when an application-owned index database can be opened, and SHALL use currently cached sessions as the initial session list.

#### Scenario: Cached sessions are shown immediately
- **WHEN** the user starts the TUI with scanning enabled
- **AND** the application database already contains indexed sessions
- **THEN** the system opens the TUI using the cached session list before the startup source scan completes
- **AND** the system indicates that a background index refresh is in progress

#### Scenario: Empty cache still opens the TUI
- **WHEN** the user starts the TUI with scanning enabled
- **AND** the application database contains no indexed sessions for the enabled sources
- **THEN** the system opens the TUI without waiting for the source scan to complete
- **AND** the system displays an empty or loading state that explains indexing is in progress

#### Scenario: Scan is disabled
- **WHEN** the user starts the TUI with `--no-scan`
- **THEN** the system opens the TUI using only the cached application database contents
- **AND** the system does not start a background source scan

### Requirement: Background indexing status and refresh
The system SHALL run enabled source scanning and index refresh work in the background after TUI startup, SHALL display truthful indexing status in the TUI, and SHALL refresh cached session browsing data after indexing completes.

#### Scenario: Background indexing reports progress
- **WHEN** a startup background index refresh is running
- **THEN** the TUI displays the current indexing state using available source, phase, or count information
- **AND** normal keyboard navigation of already loaded views remains responsive

#### Scenario: Background indexing completes
- **WHEN** a startup background index refresh completes successfully
- **THEN** the TUI updates its indexing status to completed or up to date
- **AND** the session list cache is refreshed from the application database
- **AND** the system preserves the currently selected session when that session is still present in the refreshed visible list

#### Scenario: Background indexing fails
- **WHEN** a startup background index refresh fails
- **THEN** the TUI displays a non-fatal indexing error status
- **AND** cached sessions that were already loaded remain available for browsing
- **AND** the system does not modify OpenCode or Pi source storage as part of error recovery

## MODIFIED Requirements

### Requirement: Incremental startup indexing
The system SHALL use application-owned scan metadata to avoid reparsing and rewriting unchanged OpenCode records during normal startup index refreshes, SHALL use row-level freshness for OpenCode SQLite database-backed records when available, and MUST keep OpenCode storage read-only while doing so.

#### Scenario: First scan indexes storage
- **WHEN** the application starts with no existing scan metadata for the configured OpenCode storage
- **THEN** the system scans available OpenCode records and persists session, message, part, search, and scan metadata in the application database
- **AND** the system leaves all OpenCode storage files unchanged

#### Scenario: Unchanged filesystem storage is skipped
- **WHEN** the application performs a startup index refresh after a completed scan
- **AND** an OpenCode filesystem source record has the same source path, size, and modification time as the stored scan metadata
- **THEN** the system reuses the existing application-indexed record without reparsing the source payload
- **AND** the system does not rewrite unchanged index rows for that source record

#### Scenario: Changed filesystem storage is refreshed
- **WHEN** an OpenCode filesystem source record has changed size or modification time since the stored scan metadata
- **THEN** the system reparses that source record and refreshes its dependent session summary, timeline, search, and scan metadata in the application database

#### Scenario: Unchanged OpenCode database rows are skipped after database mtime changes
- **WHEN** the `opencode.db` file modification time changes
- **AND** an OpenCode database-backed source row has the same synthetic source path and row-level freshness metadata as the stored scan metadata
- **THEN** the system reuses the existing application-indexed record without reparsing that database row payload
- **AND** the system does not rewrite unchanged index rows for that database-backed source record

#### Scenario: Changed OpenCode database rows are refreshed
- **WHEN** an OpenCode database-backed source row has changed row-level freshness metadata since the stored scan metadata
- **THEN** the system reparses that database row and refreshes its dependent session summary, timeline, search, and scan metadata in the application database

#### Scenario: Duplicate storage sources are not double processed
- **WHEN** the same OpenCode record is available from both JSON storage and `opencode.db`
- **THEN** the system selects one current source for indexing that record before doing expensive payload classification
- **AND** the system does not parse duplicate heavy payloads solely to discard them during deduplication
