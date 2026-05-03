## 1. Session List State and Projection

- [x] 1.1 Add session-list view mode state to the TUI model with flat mode as the default.
- [x] 1.2 Add a `ViewSessions` keybinding to toggle between flat and grouped session-list modes.
- [x] 1.3 Build a session-list row projection that can represent selectable session rows and non-selectable project/global header rows.
- [x] 1.4 Implement grouped row construction from the currently visible session set, grouping by project/global identity.
- [x] 1.5 Sort grouped sections by most recent visible session update time, including `Global sessions` in the same ordering, and sort sessions inside each group by update time.

## 2. Navigation and Rendering

- [x] 2.1 Update session-list movement, paging, jump, and scroll handling to operate on projected rows while keeping selection on session rows.
- [x] 2.2 Preserve selected session by session ID when toggling view modes and when applying or clearing session-list search.
- [x] 2.3 Update session-list rendering to show grouped project/global headers with label, session count, and active timestamp.
- [x] 2.4 Update wide and narrow session-list rendering so preview/open behavior uses the selected session from the active projection.
- [x] 2.5 Update session-list header/footer help to expose the active list mode and the toggle key.

## 3. Verification

- [x] 3.1 Add tests for toggling between flat and grouped modes while preserving the selected session.
- [x] 3.2 Add tests that project groups and `Global sessions` are ordered by latest visible session activity.
- [x] 3.3 Add tests that grouped search results remain grouped and sort groups by the latest matching session.
- [x] 3.4 Add tests that navigation skips group headers and keeps the selected session visible under constrained terminal height.
- [x] 3.5 Run `gofmt` on touched Go files and `go test ./...`.
