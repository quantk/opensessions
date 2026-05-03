# opensession

> A local, read-only terminal browser for your OpenCode session history.

`opensession` turns OpenCode's local session storage into a fast, searchable,
keyboard-first TUI. It scans OpenCode JSON files and `opencode.db` in read-only
mode, builds its own SQLite index, and lets you browse sessions, timelines,
tool calls, markdown answers, token usage, and raw part details without touching
OpenCode's storage.

![Go 1.25+](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat-square&logo=go)
![SQLite](https://img.shields.io/badge/SQLite-local-003B57?style=flat-square&logo=sqlite)
![Terminal UI](https://img.shields.io/badge/UI-terminal-4EAA25?style=flat-square)
![Read-only](https://img.shields.io/badge/OpenCode_storage-read--only-blue?style=flat-square)

## Why

OpenCode sessions contain a lot of valuable context: prompts, assistant
responses, tool calls, file references, subagent tasks, token usage, and raw
event metadata.

Once those sessions pile up, finding the one conversation where you solved a
bug, changed a config, or ran a specific command becomes painful.

`opensession` gives that history a local index and a focused terminal UI.

```text
OpenCode storage              opensession index              Terminal UI
JSON files / opencode.db      local SQLite DB                browse, search, inspect
        |                            |                              |
        | read-only scan             | app-owned writes             |
        v                            v                              v
+------------------+       +---------------------+       +----------------------+
| project/session  | ----> | sessions/messages   | ----> | grouped session list |
| message/part     |       | parts/search/tags   |       | timelines/details    |
+------------------+       +---------------------+       +----------------------+
```

## Features

- Read-only OpenCode scanning: `opensession` never creates, edits, deletes, archives, or compacts files inside OpenCode storage.
- Local SQLite index: sessions, messages, parts, search text, scan metadata, tags, and bookmarks live in an application-owned database.
- Fast startup rescans: unchanged OpenCode records are skipped using source path, size, and modification time metadata.
- Flat or project-grouped browsing: switch between recency-ordered sessions and project-grouped views.
- Timeline reader: open a session to inspect user messages, assistant messages, tool events, patches, files, and bounded previews.
- Context-sensitive search: `/` searches sessions from the session list and searches only the current timeline inside a session.
- Assistant markdown rendering: assistant text is rendered as terminal markdown by default, with a source-mode toggle.
- Reasoning guardrails: reasoning parts are hidden by default and only shown after an explicit toggle.
- Subagent navigation: linked `task` tool rows can open child or subagent session timelines and return to the parent context.
- Pretty part details: tool, patch, and file parts open into structured detail views instead of raw JSON dumps.
- Explicit raw JSON view: raw content is only shown after an intentional action, and unsafe payloads are guarded.
- Heavy payload protection: large tool artifacts, snapshots, binary-looking data URLs, and oversized text are summarized instead of fully indexed or rendered.
- Token usage summaries: displays available total, input, output, reasoning, cache read, and cache write token counts without showing monetary cost.
- Local tags and bookmarks: metadata is stored locally in the opensession database and never written back to OpenCode storage.

## Install / Run

Run directly from a checkout:

```sh
CGO_ENABLED=0 go run ./cmd/opensession
```

Build a local binary:

```sh
CGO_ENABLED=0 go build -o opensession ./cmd/opensession
./opensession
```

Show the version:

```sh
./opensession --version
```

## Usage

Use the default OpenCode storage location:

```sh
CGO_ENABLED=0 go run ./cmd/opensession
```

Use a custom OpenCode storage root:

```sh
CGO_ENABLED=0 go run ./cmd/opensession --storage-root /path/to/opencode/storage
```

Use a custom opensession SQLite index:

```sh
CGO_ENABLED=0 go run ./cmd/opensession --db /path/to/opensession.sqlite
```

Open the existing index without scanning first:

```sh
CGO_ENABLED=0 go run ./cmd/opensession --no-scan
```

## Keyboard

| Key | Action |
| --- | --- |
| `j` / `k` | Move selection or scroll |
| `l` / `Enter` | Open a session, part, message detail, or linked child session |
| `h` / `Esc` | Go back |
| `/` | Search the current view |
| `v` | Toggle flat/grouped session list mode |
| `r` | Show or hide reasoning parts in a timeline |
| `m` | Toggle assistant markdown rendering/source view |
| `R` | Toggle pretty detail/raw JSON in part detail view |
| `g` / `G` | Jump to top or bottom |
| `PgUp` / `PgDown` | Page through long views |
| `q` | Quit |

## Storage Paths

OpenCode storage root resolution order:

1. `--storage-root`
2. `OPENSESSION_STORAGE_ROOT`
3. `OPENCODE_STORAGE_ROOT`
4. `$XDG_DATA_HOME/opencode/storage`
5. `~/.local/share/opencode/storage`

opensession SQLite database path resolution order:

1. `--db`
2. `OPENSESSION_DB`
3. `$XDG_STATE_HOME/opensession/opensession.sqlite`
4. `~/.local/state/opensession/opensession.sqlite`

## Data Safety

`opensession` treats OpenCode storage as source material, not as writable state.

```text
OpenCode storage
        |
        | read only
        v
opensession scanner
        |
        | writes only here
        v
opensession.sqlite
```

Browsing, searching, opening raw previews, reading child sessions, tags, and
bookmarks do not mutate OpenCode files.

Large or unsafe raw parts are guarded. Normal views use bounded previews and
searchable summaries instead of rendering full raw payloads.

## Development

Enter the Nix development shell:

```sh
nix develop
```

Or use direnv:

```sh
direnv allow
```

The dev shell provides Go 1.25, `gopls`, and SQLite tools.

Run tests:

```sh
go test ./...
```

Run a focused package test:

```sh
go test ./internal/tui -run TestModelRawPartGuard
```

Format touched Go files:

```sh
gofmt -w ./path/to/file.go
```

## Tech Stack

- Go 1.25
- Bubble Tea terminal UI
- Lip Gloss styling
- Glamour markdown rendering
- Pure-Go SQLite via `modernc.org/sqlite`
- Nix flakes dev shell

## Status

`opensession` is an early local-first tool focused on safe browsing, search,
and inspection of OpenCode session history.

Possible next areas:

- Dedicated TUI shortcuts for editing local tags and bookmarks.
- Packaged release binaries.
- Richer filters for projects, models, dates, tools, and tags.
- Export views for selected sessions or timelines.
