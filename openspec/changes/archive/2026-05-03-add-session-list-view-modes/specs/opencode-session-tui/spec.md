## MODIFIED Requirements

### Requirement: Project-grouped session browsing
The system SHALL allow the user to browse OpenCode sessions in both a flat recency-ordered session-list mode and a project-grouped session-list mode when project metadata is available, and SHALL include global sessions in a distinct group in grouped mode.

#### Scenario: User switches session list mode
- **WHEN** the user is viewing the session list
- **AND** the user toggles the session-list view mode
- **THEN** the system switches between flat and project-grouped session browsing
- **AND** the system keeps the same session selected when that session is still present in the visible results

#### Scenario: Flat session list is shown
- **WHEN** flat session-list mode is active
- **THEN** the system displays sessions in a single list ordered by most recent update first

#### Scenario: Sessions belong to multiple projects
- **WHEN** project-grouped session-list mode is active
- **THEN** the user can browse sessions grouped by their OpenCode project or global scope

#### Scenario: Grouped projects are ordered by visible activity
- **WHEN** project-grouped session-list mode is active
- **THEN** the system orders project and global groups by the most recent updated session visible in each group
- **AND** the system orders sessions within each group by most recent update first
- **AND** `Global sessions` participates in the same activity ordering as project groups

#### Scenario: Search results remain grouped
- **WHEN** the user searches from the session list while project-grouped session-list mode is active
- **THEN** the system displays matching sessions under project or global group headers
- **AND** the system orders groups by the most recent matching session visible in each group
