/*--host=127.0.0.1;
--port=3306;
--user=test;
--password=test;
--check=1;
--execute=0;
--backup=0;*/

inception_magic_start;

USE app;

CREATE TABLE users (
                       id BIGINT PRIMARY KEY,
                       username VARCHAR(64) NOT NULL
);

inception_magic_commit;