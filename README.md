# opensession

`opensession` is a local, read-only TUI for browsing OpenCode session storage. It scans OpenCode JSON files into an application-owned SQLite index, then renders grouped session lists, session timelines, contextual search, and guarded raw-part detail views.

## Development Shell

This repository provides a flakes-only devShell:

```sh
nix develop
```

Direnv integration is enabled with `.envrc`:

```sh
direnv allow
```

The shell includes Go 1.25, `gopls`, and SQLite. Other tools are expected from the host environment.

## Usage

Run the TUI:

```sh
CGO_ENABLED=0 go run ./cmd/opensession
```

Select a custom OpenCode storage root:

```sh
CGO_ENABLED=0 go run ./cmd/opensession --storage-root /path/to/opencode/storage
```

Skip scanning and open the existing index:

```sh
CGO_ENABLED=0 go run ./cmd/opensession --no-scan
```

Use a custom SQLite index path:

```sh
CGO_ENABLED=0 go run ./cmd/opensession --db /path/to/opensession.sqlite
```

## Paths

Storage root selection order:

1. `--storage-root`
2. `OPENSESSION_STORAGE_ROOT`
3. `OPENCODE_STORAGE_ROOT`
4. `$XDG_DATA_HOME/opencode/storage`
5. `~/.local/share/opencode/storage`

SQLite index path selection order:

1. `--db`
2. `OPENSESSION_DB`
3. `$XDG_STATE_HOME/opensession/opensession.sqlite`
4. `~/.local/state/opensession/opensession.sqlite`

The scanner reads OpenCode storage only. It writes index data, scan metadata, tags, and bookmarks only to the opensession SQLite database.

## Navigation

Core keys:

```text
j/k       move selection
l/Enter   open session or raw part detail
h/Esc     go back
/         search current view
r         toggle reasoning in the timeline
q         quit
```

Session list search matches titles, project paths, model/provider values, safe indexed chat text, tool summaries, file paths, and local tags. Timeline search is scoped to the currently open session.

## Tags And Bookmarks

Tags and bookmarks are local-only records in the opensession SQLite database. They never mutate OpenCode storage. The current MVP implements persistence and search support in the index layer; dedicated TUI editing shortcuts can be added without changing OpenCode files.

## Raw Part Safeguards

Normal list and timeline views use bounded previews and safe searchable summaries. Large tool metadata, binary-looking data URLs, snapshots, and heavy patch/tool artifacts are not indexed as full raw payloads.

Opening raw content is an explicit action from the timeline. Heavy or unsafe parts show a guard message instead of rendering the full payload.
