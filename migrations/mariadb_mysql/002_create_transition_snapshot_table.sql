-- +goose up
CREATE TABLE transition_snapshot (
    id varchar(36) NOT NULL,
    scope_id varchar(128) NOT NULL,
    resource_id varchar(128) NOT NULL,
    variant_id varchar(128) NOT NULL,
    baseline_data blob DEFAULT NULL,
    target_data blob DEFAULT NULL,
    created_at datetime(6) NOT NULL,
    updated_at datetime(6) NOT NULL,

    PRIMARY KEY (id),
    CONSTRAINT uq_transition_snapshot_scope_id_resource_id_variant_id UNIQUE (
        scope_id, resource_id, variant_id
    )
);

-- +goose down
DROP TABLE transition_snapshot;
