## Purpose

Define the terminal user interface requirements for safely discovering, indexing, browsing, searching, and inspecting local agent session history.
## Requirements
### Requirement: OpenCode storage discovery
The system SHALL discover and read local OpenCode storage from the default OpenCode data directory and SHALL allow the storage root to be overridden by configuration or command-line input.

#### Scenario: Default storage root is available
- **WHEN** the user starts the TUI without a custom storage root
- **THEN** the system reads sessions from the default local OpenCode storage path

#### Scenario: Custom storage root is provided
- **WHEN** the user starts the TUI with a custom storage root
- **THEN** the system reads sessions from that storage root instead of the default path

### Requirement: Read-only OpenCode storage access
The system MUST NOT create, update, delete, rename, archive, or compact files inside OpenCode storage during the MVP.

#### Scenario: User browses sessions
- **WHEN** the user opens, searches, tags, or previews OpenCode sessions
- **THEN** the system leaves all OpenCode storage files unchanged

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

### Requirement: Bounded text message detail viewing
The system SHALL allow users to open safe text message parts from the session timeline into a scrollable detail view that displays more content than the bounded timeline preview while still applying a larger safety limit before wrapping or markdown rendering.

#### Scenario: User opens a user text part
- **WHEN** the user focuses a user text part in the session timeline
- **AND** the user opens it with `Enter` or `l`
- **THEN** the system displays a message detail view for that part
- **AND** the detail view shows source text rather than formatted markdown
- **AND** the detail view supports scrolling through the bounded detail content

#### Scenario: User opens an assistant text part
- **WHEN** the user focuses an assistant text part in the session timeline
- **AND** the user opens it with `Enter` or `l`
- **THEN** the system displays a message detail view for that part
- **AND** the detail view uses the current assistant markdown display mode from the timeline
- **AND** the detail view supports scrolling through the bounded detail content

#### Scenario: Detail content exceeds the safety limit
- **WHEN** the user opens a text part whose safe source text exceeds the message detail display limit
- **THEN** the system displays content only up to that detail limit
- **AND** the system indicates that the message detail content was truncated
- **AND** the system remains responsive during detail rendering and navigation

#### Scenario: Text part cannot be safely loaded
- **WHEN** the user opens a text part whose raw or source content is unsafe, binary-looking, unavailable, or too large to load within the detail guard
- **THEN** the system displays a guard message instead of attempting to render the full content
- **AND** the system may display available indexed preview metadata without implying it is complete

#### Scenario: Timeline preview remains bounded
- **WHEN** the user browses a timeline containing long user or assistant text parts
- **THEN** the timeline continues to render bounded text previews for normal navigation
- **AND** opening a text part for detail does not change the timeline preview limit

#### Scenario: Reasoning visibility is respected
- **WHEN** reasoning parts are hidden in the timeline
- **THEN** hidden reasoning content is not openable through the text message detail action
- **AND** when reasoning is explicitly shown, opening a focused reasoning part follows the same bounded detail safeguards as other text-like parts

#### Scenario: Existing part detail behavior is preserved
- **WHEN** the user opens a tool, patch, file, or linked task row from the timeline
- **THEN** the system keeps the existing detail or child-session navigation behavior for that row type

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

### Requirement: Assistant markdown timeline rendering
The system SHALL render assistant text parts as formatted terminal markdown by default in the session timeline, and SHALL provide an explicit timeline action to switch assistant text parts between rendered markdown and original markdown source.

#### Scenario: Assistant markdown is rendered by default
- **WHEN** the user opens a session timeline containing an assistant text part with markdown content
- **THEN** the system displays that assistant text part as formatted markdown rather than raw markdown source

#### Scenario: Fenced code block is highlighted
- **WHEN** an assistant text part contains a fenced code block with a recognized language
- **THEN** the system displays the code block with syntax highlighting in the rendered markdown view

#### Scenario: Unknown code fence language falls back safely
- **WHEN** an assistant text part contains a fenced code block with an unknown or unsupported language
- **THEN** the system displays the code block without failing the timeline render

#### Scenario: Inline code is styled
- **WHEN** an assistant text part contains inline markdown code spans
- **THEN** the system visually distinguishes the inline code from surrounding prose in the rendered markdown view

#### Scenario: User text remains source text
- **WHEN** a user text part contains markdown syntax
- **THEN** the system displays the user text part as source text rather than formatted markdown

#### Scenario: User toggles assistant markdown source
- **WHEN** the user activates the assistant markdown display toggle in the timeline view
- **THEN** the system switches assistant text parts between rendered markdown and original markdown source
- **AND** the system keeps user text parts, tool cards, patch cards, and file cards in their existing timeline formats

#### Scenario: Search uses source content
- **WHEN** the user searches within a session timeline that contains rendered assistant markdown
- **THEN** the system searches the underlying source or indexed text rather than rendered ANSI output

#### Scenario: Timeline rendering remains bounded
- **WHEN** the user browses a session timeline containing long assistant markdown content
- **THEN** the system renders only the bounded timeline rows needed for normal navigation and scrolling

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

### Requirement: Reasoning hidden by default
The system SHALL hide reasoning parts by default while preserving an explicit way to reveal them from the session detail view.

#### Scenario: Session contains reasoning parts
- **WHEN** the user opens the session detail view
- **THEN** reasoning content is not shown inline by default

#### Scenario: User requests reasoning details
- **WHEN** the user explicitly toggles or opens reasoning content
- **THEN** the system displays the selected reasoning content if it was indexed or can be safely loaded

### Requirement: Safe indexing of heavy parts
The system SHALL index safe text, tool summaries, statuses, file paths, titles, and metadata needed for navigation, and MUST avoid indexing large raw tool artifacts or binary-looking content by default.

#### Scenario: Part contains ordinary text
- **WHEN** the scanner processes a small text part
- **THEN** the text is indexed for search and preview

#### Scenario: Part contains a large tool artifact
- **WHEN** the scanner processes a heavy tool part with large raw metadata, diffs, snapshots, or binary-looking content
- **THEN** the system records summary metadata and size information without indexing the full raw payload by default

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

### Requirement: Explicit raw part viewing
The system SHALL allow raw part content to be shown only through an explicit user action from the part detail view, separate from normal session list, timeline rendering, and default pretty detail rendering.

#### Scenario: User toggles raw JSON for a safe part
- **WHEN** the user opens a safe part detail view and requests raw JSON with the detail-view raw toggle
- **THEN** the system displays the loaded raw content for that part in the detail view

#### Scenario: User opens a heavy part
- **WHEN** the user explicitly opens a heavy part
- **THEN** the system attempts to load that part in a detail view and indicates if the content is too large or unsafe to display normally

### Requirement: Local SQLite index and tags
The system SHALL store its source-aware search index, scan metadata, tags, and bookmarks in an application-owned SQLite database outside all indexed source storage roots.

#### Scenario: User adds a local tag
- **WHEN** the user tags a session from any enabled source
- **THEN** the tag is saved in the application database and no source storage file is modified
- **AND** the tag remains associated with the selected source session rather than another session with the same external identifier

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

### Requirement: Nord-inspired TUI visual theme
The system SHALL render the terminal user interface with a cohesive Nord-inspired semantic visual theme across session browsing, timeline browsing, session tree browsing, search prompts, footers, warnings, and part or message detail views.

#### Scenario: User views themed session list
- **WHEN** the user opens the session list
- **THEN** the system displays title, mode, metadata, source badges, selected rows, dim text, and footer help using a consistent semantic color scheme
- **AND** the session list remains readable after ANSI styling is stripped

#### Scenario: User views themed timeline
- **WHEN** the user opens a session timeline
- **THEN** the system displays role labels, source metadata, reasoning state, token metadata when available, tool rows, and footer help using the same semantic visual theme
- **AND** status or role meaning is not conveyed by color alone

#### Scenario: Terminal width is narrow
- **WHEN** the TUI renders in a narrow terminal
- **THEN** themed headers, rows, prompts, and footers continue to truncate or wrap within the available width without overflowing

### Requirement: Left focus rail
The system SHALL indicate the currently focused selectable row or content block with a left-side focus rail rather than a plain `>` text marker.

#### Scenario: Session row is focused
- **WHEN** the user moves focus to a session row in the session list
- **THEN** the focused session row displays a visible left focus rail
- **AND** the row does not use `>` as the focus indicator

#### Scenario: Timeline text block is focused
- **WHEN** the user moves focus to a text, reasoning, tool, patch, file, or linked task item in the timeline
- **THEN** the focused item displays a visible left focus rail
- **AND** multi-line focused content preserves alignment between the first focused line and continuation lines

#### Scenario: Session tree row is focused
- **WHEN** the user opens a source-specific session tree and moves focus between tree entries
- **THEN** the focused tree entry displays the same focus rail affordance used elsewhere in the TUI
- **AND** tree indentation and branch glyphs remain readable

#### Scenario: Focus rail consumes layout width
- **WHEN** focused content is wrapped or truncated
- **THEN** the system accounts for the focus rail width before wrapping or truncating the row content
- **AND** the rendered row remains within the terminal width

### Requirement: Visual status and source affordances
The system SHALL render source identity, role identity, bookmarks, and tool or part statuses with consistent textual labels and visual affordances that improve scanability without replacing existing information.

#### Scenario: Source badge is shown
- **WHEN** the system displays a selectable session row or session preview
- **THEN** the row or preview includes a source badge identifying the source kind
- **AND** the badge remains understandable as text when ANSI styling is stripped

#### Scenario: Tool status is shown
- **WHEN** the system displays a tool row or tool detail with known status metadata
- **THEN** the status is visually distinguished with a consistent affordance such as an icon, label, or themed style
- **AND** the status remains understandable from text alone

#### Scenario: Bookmark is shown
- **WHEN** a bookmarked session is displayed in the session list or preview
- **THEN** the bookmark indicator remains visually distinct from source identity, focus, and tool status indicators

#### Scenario: Warning or unsafe content guard is shown
- **WHEN** the system displays a warning, raw guard, or unsafe-content guard message
- **THEN** the warning uses the themed warning affordance
- **AND** the guard text remains explicit about what content is unavailable or unsafe

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

### Requirement: Vim-first navigation
The system SHALL support vim-first keyboard navigation in the MVP using `h`, `j`, `k`, `l`, `/`, `Enter`, `Esc`, and `q`.

#### Scenario: User navigates lists
- **WHEN** the user presses `j` or `k` in a list view
- **THEN** selection moves down or up without requiring a mouse

#### Scenario: User opens and backs out of views
- **WHEN** the user presses `l` or `Enter` on a selectable item and then presses `h` or `Esc`
- **THEN** the system opens the item and returns to the previous context using keyboard-only navigation

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

### Requirement: Nix development shell
The repository SHALL provide a Nix flakes devShell based on nixos-unstable with Go 1.25, gopls, SQLite tools, and direnv integration while allowing ancillary quality tools to come from the host environment.

#### Scenario: Developer enters the repository
- **WHEN** the developer runs `nix develop` or uses direnv with the project
- **THEN** the Go toolchain and required development tools are available from the devShell
