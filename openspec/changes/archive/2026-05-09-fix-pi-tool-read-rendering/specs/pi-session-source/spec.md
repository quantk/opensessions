## MODIFIED Requirements

### Requirement: Pi content normalization
The system SHALL normalize supported Pi entry and message content into timeline parts with bounded previews, searchable text, source metadata, and guarded raw payloads. The system SHALL represent a Pi tool call and its associated Pi tool result as one semantic tool timeline part when the result references a known tool call ID, carrying final status or error state, input summary, output preview, and safe searchable output text on that tool part without rendering the associated result as a separate lifecycle row.

#### Scenario: User and assistant text content is scanned
- **WHEN** a Pi message entry contains safe text content blocks
- **THEN** the system indexes those blocks as text timeline parts with user or assistant role metadata
- **AND** assistant text remains eligible for existing markdown rendering behavior

#### Scenario: Assistant reasoning content is scanned
- **WHEN** a Pi assistant message contains thinking or reasoning content blocks
- **THEN** the system indexes those blocks as reasoning timeline parts
- **AND** reasoning parts remain hidden by default until the user explicitly toggles reasoning visibility

#### Scenario: Tool call and tool result content is scanned
- **WHEN** a Pi assistant message contains tool call blocks and a Pi tool result message references one of those calls by tool call ID
- **THEN** the system indexes them as a single tool timeline part with tool name, final status or error state, input summary, output preview, and safe searchable text when available
- **AND** the associated tool result message does not add a separate visible tool lifecycle row to the timeline

#### Scenario: Tool result output is too large or unsafe
- **WHEN** a Pi tool result references a known tool call ID but its output exceeds safety limits or looks binary
- **THEN** the system still updates the matching tool call part with final status and bounded output metadata when available
- **AND** the system records skipped or heavy raw payload state instead of storing or rendering the full unsafe output
- **AND** the associated tool result message does not add a separate visible tool lifecycle row to the timeline

#### Scenario: Tool result has no matching call
- **WHEN** a Pi tool result message has no tool call ID or references no known tool call in the indexed Pi session
- **THEN** the system indexes a bounded standalone tool result part rather than failing the scan
- **AND** the standalone part preserves available tool name, status, preview, and safety guard metadata

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

### Requirement: Pi raw and detail guardrails
The system SHALL apply existing safe indexing, bounded rendering, message detail, pretty tool detail, and explicit raw viewing guardrails to Pi session parts. Pretty tool detail for Pi tools SHALL present safe text-like output in readable text form even when the source payload stores that output as structured content blocks, while explicit raw viewing SHALL remain available only for stored bounded raw payloads.

#### Scenario: Safe Pi text detail is opened
- **WHEN** the user opens a safe Pi text part from the timeline
- **THEN** the system displays a bounded message detail view using the same text and markdown rules as other sources

#### Scenario: Safe Pi read tool detail is opened
- **WHEN** the user opens a safe merged Pi `read` tool part whose output is stored as text content blocks
- **THEN** the pretty detail view displays the read output as readable text rather than an indented JSON dump of the content block array
- **AND** the raw JSON toggle still displays the stored raw payload when available

#### Scenario: Heavy Pi tool payload is scanned
- **WHEN** a Pi tool result, bash output, custom content, or raw payload exceeds safety limits or looks binary
- **THEN** the system records bounded metadata and previews only
- **AND** raw/detail views display a guard instead of rendering the unsafe payload normally

#### Scenario: User requests Pi raw JSON
- **WHEN** the user opens a Pi part detail view and explicitly toggles raw JSON
- **THEN** the system displays the stored bounded raw payload when available
- **AND** the system displays a guard when the raw payload was skipped or is unsafe

### Requirement: Pi incremental indexing
The system SHALL use application-owned scan metadata to avoid reparsing unchanged Pi session files during startup scans, while also ensuring Pi files are reparsed when the Pi parser/index representation changes in a way that affects stored session, tree, timeline, search, or detail behavior. The system MUST preserve indexed messages, parts, searchable documents, and branch leaves for unchanged Pi session files when another Pi session file in the same project or working directory is refreshed.

#### Scenario: First Pi scan indexes files
- **WHEN** the application starts with no scan metadata for discovered Pi session files
- **THEN** the system parses available Pi JSONL files and persists source-aware sessions, tree entries, timeline parts, search documents, and scan metadata in the opensession database
- **AND** the system leaves Pi session files unchanged

#### Scenario: Unchanged Pi file is skipped
- **WHEN** a Pi session file has the same source path, size, and modification time as the stored scan metadata
- **AND** the stored Pi parser/index representation version is current
- **THEN** the system reuses the existing application-indexed records for that file without reparsing it
- **AND** the reused session remains able to display its existing timeline and branch leaves

#### Scenario: Changed Pi file is refreshed
- **WHEN** a Pi session file size or modification time differs from stored scan metadata
- **THEN** the system reparses that Pi session file and refreshes its indexed session, tree entries, timeline parts, search documents, and scan metadata

#### Scenario: Sibling Pi file remains unchanged during project refresh
- **WHEN** two or more Pi session files belong to the same project or working directory
- **AND** one Pi session file is refreshed because its source metadata changed
- **AND** a sibling Pi session file has unchanged source metadata and a current parser/index representation version
- **THEN** the system preserves or reuses the sibling session's existing messages, parts, searchable documents, and branch leaves
- **AND** opening the sibling Pi session does not show an empty timeline solely because another session in the same project was refreshed

#### Scenario: Pi parser representation changes
- **WHEN** the application starts after a Pi parser or indexing representation change that affects persisted timeline or detail behavior
- **AND** existing Pi scan metadata was written by an older representation
- **THEN** the system reparses affected Pi session files and refreshes their indexed rows even if the source path, size, and modification time are unchanged
- **AND** the system leaves Pi session files unchanged
