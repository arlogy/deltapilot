# Changelog

*Snapshot-processing workflows, among other features, are designed to be versatile. However, we expect
real-world usage to challenge our design choices before we consider the contract stable. For this reason, we
are currently releasing `0.*` versions only.*

## 0.1.0 - 2026/08/30

First release.
- Introduce snapshot concepts with `ChronicleSnapshot` and `TransitionSnapshot`.
- Provide a store interface per snapshot type: `ChronicleStore` and `TransitionStore`, each with in-memory and
SQL storage implementations.
- Establish the architecture envisioned for a socket-based CLI and separate HTTP/S interface.
