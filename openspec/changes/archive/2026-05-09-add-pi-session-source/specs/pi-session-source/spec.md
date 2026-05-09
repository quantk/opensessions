## ADDED Requirements

### Requirement: Pi session discovery
The system SHALL discover local Pi session files from the default Pi agent session directory and SHALL allow the Pi session root to be overridden by configuration or command-line input.

#### Scenario: Default Pi session root is available
- **WHEN** the user starts the TUI with Pi scanning enabled and no custom Pi session root
- **THEN** the system reads Pi session JSONL files from the default local Pi session directory

#### Scenario: Custom Pi session root is provided
- **WHEN** the user starts the TUI with a custom Pi session root
- **THEN** the system reads Pi session JSONL files from that root instead of the default Pi session directory

#### Scenario: Pi session root is missing
- **WHEN** Pi scanning is enabled but the Pi session root does not exist
- **THEN** the system continues to browse sessions from other available sources
- **AND** the missing Pi root does not create or modify Pi directories or files

### Requirement: Read-only Pi session access
The system MUST NOT create, update, delete, rename, compact, fork, resume, label, share, or otherwise mutate Pi session files during scanning, browsing, searching, tagging, bookmarking, or raw/detail viewing.

#### Scenario: User browses Pi sessions
- **WHEN** the user opens, searches, tags, bookmarks, previews, or views raw/details for Pi sessions
- **THEN** the system leaves all Pi session JSONL files unchanged
- **AND** local opensession tags, bookmarks, search documents, and scan metadata are written only to the opensession database

### Requirement: Pi JSONL session indexing
The system SHALL parse Pi JSONL session files into source-aware indexed sessions while preserving Pi session metadata, append order, entry IDs, parent entry IDs, and safe renderable content.

#### Scenario: Linear Pi session is scanned
- **WHEN** the scanner processes a Pi JSONL file with a session header and linear entries
- **THEN** the system creates one Pi session in the local index
- **AND** the session records the Pi session ID, working directory, source file path, created or first-seen time, updated time, available display name, model metadata, message count, part count, and source kind

#### Scenario: Branched Pi session is scanned
- **WHEN** the scanner processes a Pi JSONL file whose entries contain multiple children for the same parent entry
- **THEN** the system stores the Pi entry tree relationships using the entry IDs and parent entry IDs
- **AND** the system can identify branch points and leaf entries for later browsing

#### Scenario: Session name entry is present
- **WHEN** a Pi session contains a `session_info` entry with a display name
- **THEN** the indexed session title uses the most recent session display name available on the session path or file
- **AND** the original Pi session file is not modified

### Requirement: Pi content normalization
The system SHALL normalize supported Pi entry and message content into timeline parts with bounded previews, searchable text, source metadata, and guarded raw payloads.

#### Scenario: User and assistant text content is scanned
- **WHEN** a Pi message entry contains safe text content blocks
- **THEN** the system indexes those blocks as text timeline parts with user or assistant role metadata
- **AND** assistant text remains eligible for existing markdown rendering behavior

#### Scenario: Assistant reasoning content is scanned
- **WHEN** a Pi assistant message contains thinking or reasoning content blocks
- **THEN** the system indexes those blocks as reasoning timeline parts
- **AND** reasoning parts remain hidden by default until the user explicitly toggles reasoning visibility

#### Scenario: Tool call and tool result content is scanned
- **WHEN** a Pi assistant message contains tool call blocks or a Pi tool result message contains tool output
- **THEN** the system indexes them as tool timeline parts with tool name, status or error state, input summary, output preview, and safe searchable text when available

#### Scenario: Bash execution message is scanned
- **WHEN** a Pi session contains a bash execution message entry
- **THEN** the system indexes the command and bounded output as a tool-like timeline part
- **AND** the detail view identifies the command, exit state, cancellation state, and truncation state when available

#### Scenario: Compaction and branch summary entries are scanned
- **WHEN** a Pi session contains compaction or branch summary entries
- **THEN** the system indexes them as summary timeline parts that can be displayed in the relevant branch timeline
- **AND** searchable summary text is indexed when it is safe and bounded

#### Scenario: Unsupported or custom entry is scanned
- **WHEN** a Pi session contains an unsupported entry type or custom extension entry
- **THEN** the system records safe metadata or a compact placeholder without failing the entire scan
- **AND** displayable custom message content is indexed only when it is safe and intended for display

### Requirement: Pi branch timeline viewing
The system SHALL display Pi timelines as projections of a selected branch path through the Pi entry tree rather than as the raw append order of the JSONL file.

#### Scenario: User opens a Pi session
- **WHEN** the user selects a Pi session and opens it
- **THEN** the system displays the timeline for the default selected branch path
- **AND** the default selected branch is the latest leaf branch when no other branch has been selected
- **AND** mutually exclusive branch entries outside that path are not interleaved into the visible timeline

#### Scenario: User switches branch
- **WHEN** the user selects a different Pi branch or tree entry from the branch navigator
- **THEN** the system displays the timeline path from the session root to that selected entry or leaf
- **AND** the timeline header indicates that the visible timeline is a Pi branch projection

#### Scenario: Branch contains compaction or branch summary
- **WHEN** the selected Pi branch path contains a compaction or branch summary entry
- **THEN** the timeline displays a bounded summary row at the appropriate position in the visible branch path

### Requirement: Pi session tree navigation
The system SHALL provide a keyboard-accessible tree or branch navigator for Pi sessions so users can inspect branch points, leaves, labels, and alternate paths.

#### Scenario: User opens Pi tree navigation
- **WHEN** the user is viewing a Pi session timeline and activates the tree navigation action
- **THEN** the system displays the Pi session entry tree or branch list without leaving the session context
- **AND** branch points and leaves are distinguishable in the view

#### Scenario: User selects tree entry
- **WHEN** the user selects a Pi tree entry or branch leaf and opens it
- **THEN** the system returns to the timeline view for the selected branch path
- **AND** the selected tree entry or branch becomes the active timeline context

#### Scenario: Pi labels are available
- **WHEN** a Pi session contains label entries targeting tree entries
- **THEN** the tree navigator displays the latest label for each targeted entry when available

#### Scenario: Back returns from tree navigation
- **WHEN** the user backs out of Pi tree navigation with the standard back key
- **THEN** the system returns to the previous Pi timeline context without changing the selected branch

### Requirement: Pi search behavior
The system SHALL align Pi search behavior with existing opensession search scopes while respecting Pi branches.

#### Scenario: Search from session list finds Pi sessions
- **WHEN** the user searches from the session list
- **THEN** the system searches safe indexed content from Pi sessions along with other enabled sources
- **AND** matching Pi sessions appear in the session search results with their source indicator

#### Scenario: Search from Pi timeline searches visible branch
- **WHEN** the user searches while viewing a Pi session timeline
- **THEN** the system searches the currently visible Pi branch timeline
- **AND** results from other branches are not shown as if they were visible in the current timeline

#### Scenario: Search from Pi tree navigation searches the session tree
- **WHEN** the user searches while Pi tree navigation is active
- **THEN** the system searches safe indexed entries within the open Pi session tree
- **AND** matching entries can be selected to view their branch path

### Requirement: Pi raw and detail guardrails
The system SHALL apply existing safe indexing, bounded rendering, message detail, pretty tool detail, and explicit raw viewing guardrails to Pi session parts.

#### Scenario: Safe Pi text detail is opened
- **WHEN** the user opens a safe Pi text part from the timeline
- **THEN** the system displays a bounded message detail view using the same text and markdown rules as other sources

#### Scenario: Heavy Pi tool payload is scanned
- **WHEN** a Pi tool result, bash output, custom content, or raw payload exceeds safety limits or looks binary
- **THEN** the system records bounded metadata and previews only
- **AND** raw/detail views display a guard instead of rendering the unsafe payload normally

#### Scenario: User requests Pi raw JSON
- **WHEN** the user opens a Pi part detail view and explicitly toggles raw JSON
- **THEN** the system displays the stored bounded raw payload when available
- **AND** the system displays a guard when the raw payload was skipped or is unsafe

### Requirement: Pi incremental indexing
The system SHALL use application-owned scan metadata to avoid reparsing unchanged Pi session files during startup scans.

#### Scenario: First Pi scan indexes files
- **WHEN** the application starts with no scan metadata for discovered Pi session files
- **THEN** the system parses available Pi JSONL files and persists source-aware sessions, tree entries, timeline parts, search documents, and scan metadata in the opensession database
- **AND** the system leaves Pi session files unchanged

#### Scenario: Unchanged Pi file is skipped
- **WHEN** a Pi session file has the same source path, size, and modification time as the stored scan metadata
- **THEN** the system reuses the existing application-indexed records for that file without reparsing it

#### Scenario: Changed Pi file is refreshed
- **WHEN** a Pi session file size or modification time differs from stored scan metadata
- **THEN** the system reparses that Pi session file and refreshes its indexed session, tree entries, timeline parts, search documents, and scan metadata
