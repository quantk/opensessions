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

### Requirement: Session preview and timeline
The system SHALL provide a session preview and a detailed timeline view containing user messages, assistant messages, tool events, file references, and bounded text previews.

#### Scenario: User opens a session
- **WHEN** the user selects a session and opens it
- **THEN** the system displays a chronological timeline for that session without loading unrelated sessions

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
The system SHALL display aggregate token usage for an OpenCode session when token usage metadata is available from assistant messages. The aggregate SHALL include total, input, output, reasoning, cache read, and cache write token counts where those values are available or derivable from message-level token metadata. The system MUST treat missing token usage metadata as unavailable rather than as zero token usage. The system MUST NOT display monetary cost as part of this requirement.

#### Scenario: Session usage is available
- **WHEN** the user browses or opens a session with assistant message token metadata
- **THEN** the system displays session-level token usage in the session browsing experience

#### Scenario: Session usage is unavailable
- **WHEN** the user browses or opens a session with no assistant message token metadata
- **THEN** the system indicates that token usage is unavailable or omits the token total without displaying zero usage

#### Scenario: Session usage includes cache tokens
- **WHEN** a session contains cache read or cache write token metadata
- **THEN** the system includes those cache token counts in the session-level usage breakdown

#### Scenario: Cost metadata exists
- **WHEN** OpenCode data contains monetary cost metadata for a session
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

### Requirement: Explicit raw part viewing
The system SHALL allow raw part content to be shown only through an explicit user action from the part detail view, separate from normal session list, timeline rendering, and default pretty detail rendering.

#### Scenario: User toggles raw JSON for a safe part
- **WHEN** the user opens a safe part detail view and requests raw JSON with the detail-view raw toggle
- **THEN** the system displays the loaded raw content for that part in the detail view

#### Scenario: User opens a heavy part
- **WHEN** the user explicitly opens a heavy part
- **THEN** the system attempts to load that part in a detail view and indicates if the content is too large or unsafe to display normally

### Requirement: Local SQLite index and tags
The system SHALL store its search index, scan metadata, tags, and bookmarks in an application-owned SQLite database outside OpenCode storage.

#### Scenario: User adds a local tag
- **WHEN** the user tags a session
- **THEN** the tag is saved in the application database and no OpenCode storage file is modified

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

### Requirement: Context-sensitive search
The system SHALL make `/` enter search mode for the currently active view and SHALL derive search scope from that view.

#### Scenario: Search from session list
- **WHEN** the user presses `/` on the start or session-list view and enters a query
- **THEN** the system searches or filters sessions in that view

#### Scenario: Search from session detail
- **WHEN** the user presses `/` in a session detail view and enters a query
- **THEN** the system searches within the current session timeline

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

### Requirement: Vim-first navigation
The system SHALL support vim-first keyboard navigation in the MVP using `h`, `j`, `k`, `l`, `/`, `Enter`, `Esc`, and `q`.

#### Scenario: User navigates lists
- **WHEN** the user presses `j` or `k` in a list view
- **THEN** selection moves down or up without requiring a mouse

#### Scenario: User opens and backs out of views
- **WHEN** the user presses `l` or `Enter` on a selectable item and then presses `h` or `Esc`
- **THEN** the system opens the item and returns to the previous context using keyboard-only navigation

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

### Requirement: Nix development shell
The repository SHALL provide a Nix flakes devShell based on nixos-unstable with Go 1.25, gopls, SQLite tools, and direnv integration while allowing ancillary quality tools to come from the host environment.

#### Scenario: Developer enters the repository
- **WHEN** the developer runs `nix develop` or uses direnv with the project
- **THEN** the Go toolchain and required development tools are available from the devShell
