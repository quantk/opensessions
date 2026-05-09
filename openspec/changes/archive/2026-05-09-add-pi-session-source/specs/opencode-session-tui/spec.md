## MODIFIED Requirements

### Requirement: Project-grouped session browsing
The system SHALL allow the user to browse top-level sessions from enabled sources in both a flat recency-ordered session-list mode and a project-grouped session-list mode when project or working-directory metadata is available, SHALL include top-level global or unknown-project sessions in a distinct group in grouped mode, SHALL omit OpenCode child or subagent sessions from top-level session browsing, and SHALL preserve visible source identity for each selectable session row.

#### Scenario: User switches session list mode
- **WHEN** the user is viewing the session list
- **AND** the user toggles the session-list view mode
- **THEN** the system switches between flat and project-grouped session browsing
- **AND** the system keeps the same session selected when that session is still present in the visible results

#### Scenario: Flat session list is shown
- **WHEN** flat session-list mode is active
- **THEN** the system displays top-level sessions from enabled sources in a single list ordered by most recent update first
- **AND** each selectable session row identifies its source kind

#### Scenario: Sessions belong to multiple projects
- **WHEN** project-grouped session-list mode is active
- **THEN** the user can browse top-level sessions grouped by their normalized project, working directory, or global scope
- **AND** sessions from different enabled sources can appear under the same group when their normalized project key matches

#### Scenario: Grouped projects are ordered by visible activity
- **WHEN** project-grouped session-list mode is active
- **THEN** the system orders project and global groups by the most recent updated top-level session visible in each group
- **AND** the system orders sessions within each group by most recent update first
- **AND** global or unknown-project sessions participate in the same activity ordering as project groups

#### Scenario: Search results remain grouped
- **WHEN** the user searches from the session list while project-grouped session-list mode is active
- **THEN** the system displays matching top-level sessions under project, working-directory, global, or unknown-project group headers
- **AND** the system orders groups by the most recent matching top-level session visible in each group
- **AND** each matching session row identifies its source kind

#### Scenario: Child sessions are omitted from top-level browsing
- **WHEN** OpenCode storage contains a session with explicit parent session metadata
- **THEN** the system does not display that child session as a selectable top-level row in flat session-list mode
- **AND** the system does not display that child session as a selectable top-level row in project-grouped session-list mode

### Requirement: Session preview and timeline
The system SHALL provide a source-aware session preview and a detailed timeline view containing user messages, assistant messages, tool events, file references or tool targets, summaries, and bounded text previews for sessions from enabled sources.

#### Scenario: User opens a session
- **WHEN** the user selects a session from any enabled source and opens it
- **THEN** the system displays the appropriate chronological or branch-projected timeline for that session without loading unrelated sessions
- **AND** the timeline header identifies the session source when source identity is relevant to the view

### Requirement: Session-level token usage
The system SHALL display aggregate token usage for a session when token usage metadata is available from the source. The aggregate SHALL include total, input, output, reasoning, cache read, and cache write token counts where those values are available or derivable from message-level token metadata. The system MUST treat missing token usage metadata as unavailable rather than as zero token usage. The system MUST NOT display monetary cost as part of this requirement.

#### Scenario: Session usage is available
- **WHEN** the user browses or opens a session with available token metadata
- **THEN** the system displays session-level token usage in the session browsing experience

#### Scenario: Session usage is unavailable
- **WHEN** the user browses or opens a session with no token usage metadata
- **THEN** the system indicates that token usage is unavailable or omits the token total without displaying zero usage

#### Scenario: Session usage includes cache tokens
- **WHEN** a session contains cache read or cache write token metadata
- **THEN** the system includes those cache token counts in the session-level usage breakdown

#### Scenario: Cost metadata exists
- **WHEN** source data contains monetary cost metadata for a session
- **THEN** the system does not display monetary cost in the session-level token usage UI

### Requirement: Local SQLite index and tags
The system SHALL store its source-aware search index, scan metadata, tags, and bookmarks in an application-owned SQLite database outside all indexed source storage roots.

#### Scenario: User adds a local tag
- **WHEN** the user tags a session from any enabled source
- **THEN** the tag is saved in the application database and no source storage file is modified
- **AND** the tag remains associated with the selected source session rather than another session with the same external identifier

### Requirement: Context-sensitive search
The system SHALL make `/` enter search mode for the currently active view and SHALL derive search scope from that view.

#### Scenario: Search from session list
- **WHEN** the user presses `/` on the start or session-list view and enters a query
- **THEN** the system searches or filters sessions from enabled sources in that view
- **AND** matching rows retain their source indicators

#### Scenario: Search from session detail
- **WHEN** the user presses `/` in a session detail view and enters a query
- **THEN** the system searches within the current session timeline
- **AND** source-specific timeline projections, such as a selected Pi branch, define the searched timeline scope

### Requirement: Responsive search on large indexes
The system SHALL keep session-list and in-session searches responsive on large local indexes containing sessions from one or more sources by using indexed or bounded query paths and non-blocking TUI behavior for user-triggered searches.

#### Scenario: Session-list search remains interactive
- **WHEN** the user searches from the session list while the local index contains many sessions and searchable part documents from one or more sources
- **THEN** the system performs the search without freezing normal TUI rendering or input handling until the query completes
- **AND** the system displays matching top-level sessions ordered consistently with the session-list view
- **AND** matching rows retain their source indicators

#### Scenario: In-session search remains interactive
- **WHEN** the user searches within a session timeline that contains many parts
- **THEN** the system performs the search without freezing normal TUI rendering or input handling until the query completes
- **AND** the system searches safe indexed/source content rather than rendered ANSI markdown output
- **AND** the system respects the current source-specific timeline scope

#### Scenario: Stale search results are ignored
- **WHEN** multiple searches are requested in quick succession
- **AND** an older search completes after a newer search has been requested
- **THEN** the system ignores the stale older result and keeps the current search state consistent with the newest query

### Requirement: Responsive rendering for large sessions
The system SHALL keep list, source-aware timeline, and source-specific tree rendering, navigation, and scroll calculations bounded to visible content plus small cached layout metadata, and SHALL NOT concatenate raw transcripts, render full raw payloads, or rerender all assistant markdown parts during normal navigation.

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

#### Scenario: Source-specific tree navigation remains bounded
- **WHEN** the user navigates a source-specific tree view for a large branched session
- **THEN** the system renders only visible tree rows and bounded labels or summaries needed for navigation
- **AND** normal navigation does not render every timeline part from every branch
