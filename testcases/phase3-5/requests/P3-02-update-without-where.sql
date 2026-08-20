/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--db=gip_phase_test;--check=1;--execute=0;--backup=0;*/
inception_magic_start;
UPDATE users SET score = 0;
inception_magic_commit;
