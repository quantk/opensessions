## ADDED Requirements

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
