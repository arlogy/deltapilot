# Concepts

## Snapshot types

[Snapshots](../revision) capture resource state at the request of API callers for subsequent processing. There
are several types of snapshots.

`ChronicleSnapshot`
- A snapshot capturing a resource state at a specific point in time.
- It is intended to be immutable.

`TransitionSnapshot`
- A snapshot capturing a resource transition from a baseline state to a target state.
- Both states are initialized during snapshot creation.
- The baseline state can be set only once through the API, whereas the target state may evolve; this leads to
the following scenarios.
    - When the target state is expected to remain unchanged after baseline establishment: in a review workflow
    for example, the target state may represent the desired state of the resource relative to its baseline
    state. The proposed changes would then be the differences between the two states.
    - When the target state may change after baseline establishment: in a review workflow for example, the
    state transition would be evaluated in the same way as when the target state is expected to remain
    unchanged. However, allowing the target state to be updated over time is more suitable when the final
    target state may not be fully known at snapshot creation time, such as when repairs are made to a system
    incrementally and its new state is recorded after each step. In this case, the review workflow would
    actually be evaluating whether the final target state is appropriate, ensuring that the repairs are
    effective given the baseline condition.
    - Note: in both cases, when the baseline state is `nil`, the target state may represent the state of a
    newly created resource, or, for an existing resource, a proposed new state to be reviewed against an
    external reference state maintained centrally, such as in a database.

## Snapshot storage

Each snapshot type can be stored using different storage backends: in-memory or database-backed.

In-memory stores are useful for rapid development or quick testing.

Database stores are recommended for production, with the following considerations in mind.
- Go models.
    - We could have modeled snapshot `[]byte` data fields as a separate structure keyed by a computed
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
    fields are stored. The mappings below are specific to PostgreSQL and are [adapted](../migrations) as
    needed for other database vendors.
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
