/*--host={{HOST}};--port={{PORT}};--user={{USER}};--password={{PASSWORD}};--db=gip_phase_test;--check=1;--execute=1;--backup=0;*/
inception_magic_start;
INSERT INTO users(id, username, score) VALUES (900, 'phase4', 1.00);
UPDATE users SET score = 2.00 WHERE id = 900;
DELETE FROM users WHERE id = 900;
inception_magic_commit;
