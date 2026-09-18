# deltapilot

deltapilot is a change orchestration engine.
- It consumes changes to payloads and allows configured workflows or actions to be triggered in response.
- Review is one possible outcome.

deltapilot supports communication via a CLI-accessible Unix socket for local clients and an HTTP/S interface
for network clients.

## Architecture diagram

*The project is currently a work in progress, with many features not yet implemented.*

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
```

- The Go library provides Go applications with direct access to all exposed functionality.
- The Unix socket client is available through the CLI.
- The Unix socket server handles requests from socket clients and returns responses.
- The backend layer is a set of shared components.
- The HTTP client could be a web browser or any HTTP-compatible application.
- The HTTP server handles requests from HTTP clients and returns responses.
- Note: both the Unix socket and HTTP servers can be run as daemons.

## Project guides

- [Concepts](./docs/01_concepts.md)
- [Installation](./docs/02_install.md)
- [Development](./docs/03_develop.md)
