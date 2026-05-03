## ADDED Requirements

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
