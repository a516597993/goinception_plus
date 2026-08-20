/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--db=gip_phase_test;--check=1;--execute=1;--backup=1;*/
inception_magic_start;
UPDATE users SET score = 88.88 WHERE id = 1;
inception_magic_commit;
