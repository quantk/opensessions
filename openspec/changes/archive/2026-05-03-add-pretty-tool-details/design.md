## Context

The TUI already treats tool, patch, and file parts as explicit-open items from the session timeline. Pressing `l` or `Enter` opens `ViewRawPart`, loads `index.RawPart`, guards heavy/binary/skipped payloads, and renders `formatRawContent(...)` under a `Raw JSON` heading.

That flow is safe, but it optimizes for debugging storage shape rather than quickly understanding a session. The existing scanner/index safety model should stay intact: normal timeline rendering uses bounded summaries, safe raw JSON is available only for non-heavy parts, and heavy or unsafe parts show a guard instead of rendering payloads.

## Goals / Non-Goals

**Goals:**

- Make opened tool, patch, and file details readable by default without requiring the user to inspect JSON structure.
- Preserve raw JSON as an explicit detail-view toggle for debugging safe payloads.
- Keep heavy, binary, and skipped raw guards unchanged.
- Keep timeline rendering and focus behavior unchanged.
- Keep parsing tolerant so unknown OpenCode tool shapes degrade into a generic readable summary.

**Non-Goals:**

- Redesign timeline cards or timeline navigation.
- Add an unsafe override for heavy or binary raw payloads.
- Change scanner/index safety rules or SQLite schema.
- Fully model every possible OpenCode tool schema before rendering anything useful.

## Decisions

### Keep Pretty Rendering In The TUI Layer

The detail renderer should parse safe raw JSON when a part is explicitly opened and keep the resulting representation in TUI state or derive it during rendering. This avoids a database migration and avoids making the scanner responsible for presentation-specific fields.

Alternatives considered: extend `index.RawPart` with many normalized tool fields, or store a new detail JSON projection in SQLite. Both add persistence surface area before the tool shapes are stable enough to justify it.

### Default To Pretty Mode, Toggle Raw Mode Explicitly

`ViewRawPart` should become a detail view with two display modes: pretty detail by default and raw JSON on a hotkey such as `R`. Existing `/` filtering and scroll behavior can apply to the currently displayed content.

Alternatives considered: always show pretty detail followed by raw JSON, or split into separate views. Always showing raw keeps the current readability problem, while a separate view adds navigation complexity for little benefit.

### Dispatch By Part Kind First, Then Tool Name

Rendering should branch by `RawPart.Kind` for tool, patch, and file detail views. Tool rendering can then branch by `ToolName` for high-value layouts and fall back to a generic tool renderer.

Initial useful tool families:

- `bash`: command, workdir, description, status, output.
- Search/list tools such as `grep` and `glob`: query/pattern, base path, include filters, matches or file list when available.
- File inspection/mutation tools such as `read`, `write`, `edit`, and `apply_patch`: target path, relevant ranges, size or diff summary, and safe snippets when available.
- Workflow/helper tools such as `task`, `todowrite`, `question`, and `webfetch`: primary request metadata and concise result preview.

Alternatives considered: implement only one generic renderer first. That is safer but misses the main quick-reading benefit for the most common tools. A generic fallback still limits risk.

### Preserve Summary-Only Behavior For Guarded Parts

When a part is heavy, binary, skipped, or exceeds the raw display limit, the detail view should keep the current guard behavior. It may show safe indexed metadata such as kind, tool name, title, status, source path, and size, but it must not load or render the unsafe raw payload.

Alternatives considered: add a second confirmation to force raw display. That would conflict with the current safety posture and is out of scope.

## Risks / Trade-offs

- OpenCode tool JSON shapes may vary across versions -> Use tolerant `map[string]any` parsing, omit missing fields, and always provide a generic fallback.
- Pretty renderers may hide debugging details -> Keep raw JSON available through an explicit hotkey for safe payloads.
- Detail rendering could become cluttered if every field is shown -> Optimize for quick reading by showing high-signal fields first and leaving exhaustive data to raw mode.
- Search/filter semantics may be ambiguous between pretty and raw mode -> Apply `/` to whichever content is currently displayed and label the active mode in the view.
