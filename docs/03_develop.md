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
relies on `testing.Short()` to skip concurrency tests, which could otherwise take several seconds to run
against database-backed stores.
