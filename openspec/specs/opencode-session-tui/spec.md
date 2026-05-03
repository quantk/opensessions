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

### Requirement: Session preview and timeline
The system SHALL provide a session preview and a detailed timeline view containing user messages, assistant messages, tool events, file references, and bounded text previews.

#### Scenario: User opens a session
- **WHEN** the user selects a session and opens it
- **THEN** the system displays a chronological timeline for that session without loading unrelated sessions

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

### Requirement: Context-sensitive search
The system SHALL make `/` enter search mode for the currently active view and SHALL derive search scope from that view.

#### Scenario: Search from session list
- **WHEN** the user presses `/` on the start or session-list view and enters a query
- **THEN** the system searches or filters sessions in that view

#### Scenario: Search from session detail
- **WHEN** the user presses `/` in a session detail view and enters a query
- **THEN** the system searches within the current session timeline

### Requirement: Vim-first navigation
The system SHALL support vim-first keyboard navigation in the MVP using `h`, `j`, `k`, `l`, `/`, `Enter`, `Esc`, and `q`.

#### Scenario: User navigates lists
- **WHEN** the user presses `j` or `k` in a list view
- **THEN** selection moves down or up without requiring a mouse

#### Scenario: User opens and backs out of views
- **WHEN** the user presses `l` or `Enter` on a selectable item and then presses `h` or `Esc`
- **THEN** the system opens the item and returns to the previous context using keyboard-only navigation

### Requirement: Responsive rendering for large sessions
The system SHALL keep list and timeline rendering bounded to visible content and SHALL NOT concatenate or render full raw transcripts for normal navigation.

#### Scenario: Session contains a very large tool part
- **WHEN** the user browses the session list or opens the session timeline
- **THEN** the UI remains based on summaries and bounded previews rather than rendering the full heavy payload

### Requirement: Nix development shell
The repository SHALL provide a Nix flakes devShell based on nixos-unstable with Go 1.25, gopls, SQLite tools, and direnv integration while allowing ancillary quality tools to come from the host environment.

#### Scenario: Developer enters the repository
- **WHEN** the developer runs `nix develop` or uses direnv with the project
- **THEN** the Go toolchain and required development tools are available from the devShell
