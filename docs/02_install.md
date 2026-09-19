# Installation

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

## Unix socket communication

*The communication is built on netkit (described below), which allows the network type, address, and other
parameters to be configured.*

### Run from source

- `cd <path_to_project>`
- `go run ./cmd/netapp` outputs usage.
- `go run ./cmd/netapp server` starts a server.
- `go run ./cmd/netapp client` starts a client.

### Run from build

- `cd <path_to_project>`
- `go build -o ./builds/deltapilot-net$(go env GOEXE) ./cmd/netapp` builds the executable.
- Run the resulting executable accordingly.

## Related projects

These projects are reusable subpackages. They are currently not published as standalone packages because they
depend on other packages used internally by their APIs or test code.

### netkit

netkit provides a flexible client-server architecture built around the Go [net](https://pkg.go.dev/net)
package.
- `netkit/transport` exposes the communication API.
- `netkit/cli` exposes an API for CLI integration.

When running a CLI application powered by netkit, please keep the following in mind.
- Server: for `-nettype` and `-netaddr`, see [net.Listen](https://pkg.go.dev/net#Listen) for supported
network types, address syntax, and their semantics.
- Client: for `-nettype` and `-netaddr`, see [net.Dial](https://pkg.go.dev/net#Dial) for supported network
types, address syntax, and their semantics.
