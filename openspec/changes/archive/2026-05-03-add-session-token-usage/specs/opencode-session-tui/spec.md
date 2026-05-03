## ADDED Requirements

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
