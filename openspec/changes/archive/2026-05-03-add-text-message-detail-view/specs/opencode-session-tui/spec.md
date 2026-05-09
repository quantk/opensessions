## ADDED Requirements

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
