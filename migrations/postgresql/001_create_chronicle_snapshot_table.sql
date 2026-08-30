-- +goose up
CREATE TABLE chronicle_snapshot (
    id uuid NOT NULL,
    scope_id varchar(128) DEFAULT NULL,
    resource_id varchar(128) NOT NULL,
    variant_id varchar(128) DEFAULT NULL,
    data bytea DEFAULT NULL,
    created_at timestamptz(6) NOT NULL,

    PRIMARY KEY (id)
);

CREATE INDEX idx_chronicle_snapshot_scope_id    ON chronicle_snapshot (scope_id);
CREATE INDEX idx_chronicle_snapshot_resource_id ON chronicle_snapshot (resource_id);
CREATE INDEX idx_chronicle_snapshot_variant_id  ON chronicle_snapshot (variant_id);

-- +goose down
DROP TABLE chronicle_snapshot;
