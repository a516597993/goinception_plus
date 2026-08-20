/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--db=gip_phase_test;--check=1;--execute=1;--backup=0;*/
inception_magic_start;
INSERT INTO users(id, username) VALUES (1, 'duplicate_primary_key');
INSERT INTO users(id, username) VALUES (902, 'must_not_execute');
inception_magic_commit;
