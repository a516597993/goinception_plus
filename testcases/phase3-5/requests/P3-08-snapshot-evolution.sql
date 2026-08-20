/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--db=gip_phase_test;--check=1;--execute=0;--backup=0;*/
inception_magic_start;
CREATE TABLE phase_batch (id BIGINT PRIMARY KEY);
ALTER TABLE phase_batch ADD COLUMN name VARCHAR(32);
INSERT INTO phase_batch(id, name) VALUES (1, 'snapshot-visible');
inception_magic_commit;
