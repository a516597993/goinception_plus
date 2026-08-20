/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--db=gip_phase_test;--check=1;--execute=1;--backup=0;*/
inception_magic_start;
INSERT INTO users(id, username) VALUES (901, 'must_not_exist');
UPDATE users SET score = 99;
inception_magic_commit;
