# deltapilot

deltapilot is a change orchestration engine.
- It consumes changes to payloads and allows configured workflows or actions to be triggered in response.
- Review is one possible outcome.

deltapilot supports communication via a CLI-accessible socket client or an HTTP/S interface.

## Architecture diagram

```
+------------+ +--------+ +--------+
| Go library | | Socket | |  HTTP  |
|    User    | | Client | | Client |
+------------+ +--------+ +--------+
  |              |               |
  |              v               v
  |            +--------+ +--------+
  |            | Socket | |  HTTP  |
  |            | Server | | Server |
  |            +--------+ +--------+
  |              |               |
  |              |               |
  |            +-|---------------|-+
  |            |      Backend      |
  |            |       Layer       |
  |            +-|---------------|-+
  |              |               |
  v              v               v
+---------------------------------------------------------+
|                       deltapilot                        |
| [core features, storage logic, transport adapters, ...] |
+---------------------------------------------------------+

- the Go library provides Go applications with direct access to all exposed functionality
- the socket client is available through the CLI
- the socket server is a long-running process that handles requests from socket clients and returns responses
- the backend layer is a set of shared components
- the HTTP client could be a web browser or any HTTP-compatible application
- the HTTP server is a long-running process that handles requests from HTTP clients and returns responses

note: both the socket and HTTP servers run as daemons
```

## Concepts

### Snapshot types

[Snapshots](./revision) capture resource state at the request of API callers for subsequent processing. There
are several types of snapshots.

`ChronicleSnapshot`
- A snapshot capturing a resource state at a specific point in time.
- It is intended to be immutable.

`TransitionSnapshot`
- A snapshot capturing a resource transition from an optional baseline state to a required target state, with
both states initialized during snapshot creation.
- Because the baseline state can be set only once through the API, the following scenarios are possible.
    - Baseline state known; target state expected to remain unchanged after baseline establishment: in a
    review workflow for example, the target state may represent the desired state of the resource relative to
    its baseline state. The proposed changes would then be the differences between the two states.
    - Baseline state known; target state expected to change after baseline establishment: in a review workflow
    for example, the state transition would be evaluated in the same way as when the target state is expected
    to remain unchanged. However, reviewing a target state recorded at a different time than the baseline is
    more suitable for evaluating whether the final proposed target remains relevant relative to the baseline,
    such as when repairs made to a system must be controlled relative to the baseline condition.
    - Baseline state unknown; target state expected to either remain unchanged or change after baseline
    establishment: in a review workflow for example, the target state may represent a new state proposed for
    review against another reference state, such as a centralized or database state.

### Snapshot storage

Each snapshot type can be stored using different storage backends: in-memory or database-backed.

In-memory stores are useful for rapid development or quick testing.

Database stores are recommended for production, with the following considerations in mind.
- Go models.
    - We could have stored snapshot `[]byte` data fields into a separate table structure keyed by a computed
    identifier, allowing identical snapshot data to be shared. However, this would introduce concerns about
    hashing costs and collision handling. Moreover, if the goal is storage-size optimization, there would also
    be additional complexity around deduplication (e.g. whether JSON documents differing only in whitespace,
    key ordering, or numeric representation such as `1` versus `1.0` should be considered identical).
    - We deliberately did not introduce a `Resource` or a `Variant` model, each with an `ExternalID` and a
    surrogate `ID` fields to back snapshot records. Indeed, snapshots are expected to outlive the
    caller-managed resources whose state they capture, so their identity is based directly on the
    caller-provided `ResourceID` and `VariantID` rather than on indirect references to models that would need
    to be retained indefinitely. For the same reason, every other identifier provided by callers, such as
    `ScopeID`, is associated directly with snapshots rather than referencing a separate model.
- SQL equivalents.
    - Summary: each snapshot Go model defines the meaning of its fields; the SQL schema determines how those
    fields are stored. The mappings below are specific to PostgreSQL and are [adapted](./migrations) as needed
    for other database vendors.
    - We use `uuid` for identifier columns generated internally. This accommodates UUIDv7 values generated
    using `NewUUIDv7()`.
    - We use `varchar(128)` for identifier columns generated externally. This type has a reasonable length
    limit intended to accommodate identifiers supplied by API callers in various formats, such as integers or
    strings. Callers are therefore responsible for providing a suitable opaque identifier that fits within
    this limit. However, the project can also be built from source with revised SQL types, allowing column
    sizes to be increased or decreased, or column definitions to be changed completely as needed. In that
    case, project tests must be run against the customized database schema for consistency.
    - We use `bytea` for arbitrary binary data fields represented as `[]byte` in Go.
    - We use `timestamptz(6)` for fields represented as `time.Time` in Go.

## Database configuration

### Initialization

Create a database dedicated to deltapilot.
- *You may adjust the creation statements accordingly. Using `deltapilot` is suggested for easier database
identification and isolation, though other names may be used.*
- MariaDB or MySQL: `CREATE DATABASE deltapilot CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;`.
- PostgreSQL: `CREATE DATABASE deltapilot TEMPLATE template0 ENCODING 'UTF8';`.

Configure database settings.
- `cd <path_to_project>`
- `cp .env.template .env` and set environment variables accordingly.

### Migration

Install goose, our database migration tool.
- `cd <path_to_project>`
- Run `go install github.com/pressly/goose/v3/cmd/goose@latest`.
- Run `goose.exe -version` on Linux/macOS, or `goose -version` on Windows, to verify the installation.
- More information at https://pressly.github.io/goose/.

Run the migrations.
```bash
# MariaDB or MySQL: goose -dir migrations/mariadb_mysql <command>
# PostgreSQL:       goose -dir migrations/postgresql <command>

# <command>:        status, up or down
```

## Development

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
