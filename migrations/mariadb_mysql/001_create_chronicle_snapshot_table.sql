-- +goose up
CREATE TABLE chronicle_snapshot (
    id varchar(36) NOT NULL,
    scope_id varchar(128) DEFAULT NULL,
    resource_id varchar(128) NOT NULL,
    variant_id varchar(128) DEFAULT NULL,
    `data` blob DEFAULT NULL,
    created_at datetime(6) NOT NULL,

    PRIMARY KEY (id),
    INDEX idx_chronicle_snapshot_scope_id (scope_id),
    INDEX idx_chronicle_snapshot_resource_id (resource_id),
    INDEX idx_chronicle_snapshot_variant_id (variant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose down
DROP TABLE chronicle_snapshot;
