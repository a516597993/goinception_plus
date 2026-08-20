/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--check=1;--execute=0;--backup=0;*/
inception_magic_start;
UPDATE gip_phase_test.users SET score = 11.00 WHERE id = 1;
inception_magic_commit;
