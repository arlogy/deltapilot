# Development

Codebase notes.
- We use a maximum line width of 110 characters to reduce horizontal scrolling. A vertical ruler is
recommended in IDEs to help enforce this rule.

Refresh the project's Go module after source code changes.
- `cd <path_to_project>`
- `go mod tidy` (adds missing dependencies and remove unused ones from `go.mod` and `go.sum`)

Run tests.
- `cd <path_to_project>`
- Run `go test ./...` to test the whole project, or `go test ./<dir_path>` to test a specific package by its
path.
- Use the `-count=1` option to force tests to run again instead of using cached results.
- Use the `-short` or `-short=true` option for faster development feedback. This works because the test code
relies on `testing.Short()` to skip concurrency tests (which can take more than a minute to run against some
database-backed stores).

Flaky test detection.
- Note: we aim to identify tests that exhibit recurring failures due to timing, scheduling, or synchronization
issues, rather than configuring excessively long timeouts that mask underlying problems. To achieve this, we
repeat each test command that is prone to these failures a fixed number of times, using an arbitrary
repetition limit, as illustrated below.
- netkit: run `go test ./netkit/<package_name> -count=40` or `go test ./netkit/... -count=40` to detect
unreliable read/write timeouts and verify that tests pass consistently across repeated runs.
