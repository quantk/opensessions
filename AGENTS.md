# AGENTS.md

## Commands

- Enter the intended toolchain with `nix develop` or `direnv allow`; the flakes-only devShell provides Go 1.25, `gopls`, and `sqlite`, not lint/CI wrappers.
- Run the TUI with `CGO_ENABLED=0 go run ./cmd/opensession`; useful flags are `--storage-root`, `--db`, `--no-scan`, and `--version`.
- Run all tests with `go test ./...`.
- Run focused tests by package, for example `go test ./internal/opencode -run TestScanIncludesSQLiteDatabaseSessions`, `go test ./internal/index -run TestStoreUpsertsSearchTagsBookmarksAndScanMetadata`, or `go test ./internal/tui -run TestModelRawPartGuard`.
- There is no Makefile, task runner, CI workflow, or repo lint config; use `gofmt` on touched Go files and direct `go test` commands.

## Architecture

- `cmd/opensession/main.go` wires the app: resolve config, open the opensession SQLite index, scan OpenCode storage unless `--no-scan`, then start the Bubble Tea TUI.
- `internal/opencode` is the read-only OpenCode scanner/classifier; it reads JSON storage directories and also the sibling `opencode.db` when the storage root is `.../storage`.
- `internal/index` owns the application SQLite schema/store/search plus local tags and bookmarks; it uses pure-Go `modernc.org/sqlite`, so `CGO_ENABLED=0` should keep working.
- `internal/tui` owns Bubble Tea rendering/navigation and talks to storage through its `Repository` interface; keep scanner/index details out of UI code when possible.
- Test fixtures live under `testdata/opencode/storage` and mirror OpenCode's `project`, `session`, `message`, and `part` layout.

## Data Safety

- OpenCode storage must stay read-only: scanning, searching, raw previews, tags, and bookmarks must not create or mutate files under the OpenCode storage root.
- Local writable state belongs in the opensession SQLite DB: DB path resolution is `--db`, `OPENSESSION_DB`, `$XDG_STATE_HOME/opensession/opensession.sqlite`, then `~/.local/state/opensession/opensession.sqlite`.
- Storage root resolution is `--storage-root`, `OPENSESSION_STORAGE_ROOT`, `OPENCODE_STORAGE_ROOT`, `$XDG_DATA_HOME/opencode/storage`, then `~/.local/share/opencode/storage`.
- Heavy or binary-looking raw parts must stay out of normal indexing/rendering; `internal/opencode/classify.go` sets `SkippedRaw`, and the TUI raw view guards large/unsafe content.
- Reasoning parts are hidden in timelines by default and only shown after the explicit `r` toggle.

## OpenSpec

- The canonical project spec is `openspec/specs/opencode-session-tui/spec.md`; archived change history is under `openspec/changes/archive/`.
- Repo-local OpenCode commands `/opsx-propose`, `/opsx-apply`, `/opsx-archive`, and `/opsx-explore` exist for OpenSpec workflow tasks, but ordinary code edits do not require them.
