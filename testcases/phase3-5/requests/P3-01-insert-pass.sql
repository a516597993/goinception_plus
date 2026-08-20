/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--db=gip_phase_test;--check=1;--execute=0;--backup=0;*/
inception_magic_start;
INSERT INTO users(id, username, score) VALUES (100, 'audit_only', 12.34);
inception_magic_commit;
