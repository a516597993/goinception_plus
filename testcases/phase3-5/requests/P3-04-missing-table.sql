/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--db=gip_phase_test;--check=1;--execute=0;--backup=0;*/
inception_magic_start;
UPDATE table_does_not_exist SET value = 1 WHERE id = 1;
inception_magic_commit;
