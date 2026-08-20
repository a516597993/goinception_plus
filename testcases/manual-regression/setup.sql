DROP DATABASE IF EXISTS gip_manual;
CREATE DATABASE gip_manual CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
USE gip_manual;

CREATE TABLE users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'primary id',
  username VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'user name',
  age INT NOT NULL DEFAULT 0 COMMENT 'age',
  amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT 'amount',
  payload VARBINARY(32) NULL COMMENT 'binary payload',
  profile JSON NULL COMMENT 'json profile',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'created time',
  PRIMARY KEY (id),
  UNIQUE KEY uniq_username (username),
  KEY idx_age (age)
) ENGINE=InnoDB COMMENT='users';

CREATE TABLE orders (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'primary id',
  user_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'user id',
  order_no VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'order no',
  amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT 'amount',
  status TINYINT NOT NULL DEFAULT 0 COMMENT 'status',
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT 'updated time',
  PRIMARY KEY (id),
  UNIQUE KEY uniq_order_no (order_no),
  KEY idx_user_id (user_id)
) ENGINE=InnoDB COMMENT='orders';

CREATE TABLE no_key (
  value_text VARCHAR(64) NULL,
  value_num INT NULL
) ENGINE=InnoDB;

CREATE TABLE type_matrix (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  int_col INT NOT NULL DEFAULT 0,
  str_col VARCHAR(64) NOT NULL DEFAULT '',
  time_col DATETIME NULL,
  json_col JSON NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

INSERT INTO users(username,age,amount,payload,profile) VALUES
('alice',20,10.50,X'0102','{"role":"user","active":true}'),
('bob',30,20.25,NULL,NULL),
('carol',40,30.75,X'FF00','{"role":"admin","active":false}');

INSERT INTO orders(user_id,order_no,amount,status) VALUES
(1,'O-001',10.50,0),(1,'O-002',11.50,1),(2,'O-003',20.25,0);

INSERT INTO no_key VALUES ('a',1),('b',2);
INSERT INTO type_matrix(int_col,str_col,time_col,json_col) VALUES
(1,'1','2026-08-20 10:00:00','{"n":1}');

