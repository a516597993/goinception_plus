/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--db=gip_phase_test;--check=1;--execute=1;--backup=0;*/
inception_magic_start;
CREATE TABLE phase4_ddl (id BIGINT PRIMARY KEY);
ALTER TABLE phase4_ddl ADD COLUMN note VARCHAR(64);
inception_magic_commit;
