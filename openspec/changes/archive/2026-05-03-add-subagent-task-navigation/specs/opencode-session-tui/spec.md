## ADDED Requirements

### Requirement: Subagent task row navigation
The system SHALL expose OpenCode child or subagent sessions through linked `task` tool rows in the parent session timeline.

#### Scenario: Linked task opens child session
- **WHEN** the user opens a focused `task` tool row that links to a child session
- **THEN** the system displays the chronological timeline for that child session
- **AND** the system indicates that the displayed timeline is nested under the parent session context

#### Scenario: Back returns to parent task context
- **WHEN** the user opens a child session from a parent timeline `task` row
- **AND** the user backs out with `h` or `Esc`
- **THEN** the system returns to the parent session timeline
- **AND** the previously opened `task` row remains selected or visible as the active context

#### Scenario: Unlinked task keeps detail behavior
- **WHEN** the user opens a focused `task` tool row that does not link to a child session
- **THEN** the system opens the task part detail view using the existing tool detail behavior

#### Scenario: Child session remains read-only
- **WHEN** the user opens, searches, previews, or backs out of a child session timeline
- **THEN** the system leaves all OpenCode storage files unchanged

## MODIFIED Requirements

### Requirement: Project-grouped session browsing
The system SHALL allow the user to browse top-level OpenCode sessions in both a flat recency-ordered session-list mode and a project-grouped session-list mode when project metadata is available, SHALL include top-level global sessions in a distinct group in grouped mode, and SHALL omit OpenCode child or subagent sessions from top-level session browsing.

#### Scenario: User switches session list mode
- **WHEN** the user is viewing the session list
- **AND** the user toggles the session-list view mode
- **THEN** the system switches between flat and project-grouped session browsing
- **AND** the system keeps the same session selected when that session is still present in the visible results

#### Scenario: Flat session list is shown
- **WHEN** flat session-list mode is active
- **THEN** the system displays top-level sessions in a single list ordered by most recent update first

#### Scenario: Sessions belong to multiple projects
- **WHEN** project-grouped session-list mode is active
- **THEN** the user can browse top-level sessions grouped by their OpenCode project or global scope

#### Scenario: Grouped projects are ordered by visible activity
- **WHEN** project-grouped session-list mode is active
- **THEN** the system orders project and global groups by the most recent updated top-level session visible in each group
- **AND** the system orders sessions within each group by most recent update first
- **AND** `Global sessions` participates in the same activity ordering as project groups

#### Scenario: Search results remain grouped
- **WHEN** the user searches from the session list while project-grouped session-list mode is active
- **THEN** the system displays matching top-level sessions under project or global group headers
- **AND** the system orders groups by the most recent matching top-level session visible in each group

#### Scenario: Child sessions are omitted from top-level browsing
- **WHEN** OpenCode storage contains a session with explicit parent session metadata
- **THEN** the system does not display that child session as a selectable top-level row in flat session-list mode
- **AND** the system does not display that child session as a selectable top-level row in project-grouped session-list mode

### Requirement: Pretty part detail viewing
The system SHALL render opened safe tool, patch, and file parts as structured quick-reading detail views by default instead of raw JSON dumps, except linked `task` tool parts whose primary open action navigates to a child session timeline.

#### Scenario: User opens a safe non-linked tool part
- **WHEN** the user opens a non-heavy tool part from the session timeline
- **AND** that tool part is not a `task` part linked to a child session
- **THEN** the system displays a structured detail view that highlights high-signal tool information such as tool name, status, title or description, input fields, and output preview when available

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
