CREATE DATABASE IF NOT EXISTS gip_phase_test CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE gip_phase_test;

DROP TABLE IF EXISTS users;
CREATE TABLE users (
    id BIGINT NOT NULL,
    username VARCHAR(64) NOT NULL,
    score DECIMAL(10,2) NULL,
    payload VARBINARY(64) NULL,
    updated_at DATETIME(6) NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_username (username)
) ENGINE=InnoDB COMMENT='phase 3-5 test users';

INSERT INTO users(id, username, score, payload, updated_at) VALUES
    (1, 'alice', 10.50, X'0102', '2026-08-18 10:00:00.123456'),
    (2, 'bob',   NULL,  NULL,    NULL);

DROP TABLE IF EXISTS no_key;
CREATE TABLE no_key (
    value VARCHAR(32) NULL
) ENGINE=InnoDB;
INSERT INTO no_key VALUES ('before');

DROP TABLE IF EXISTS phase4_ddl;
DROP TABLE IF EXISTS phase_batch;
