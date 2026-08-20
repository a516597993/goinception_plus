SELECT VERSION() AS mysql_version,
       @@sql_mode AS sql_mode,
       @@lower_case_table_names AS lower_case_table_names;
SELECT @@log_bin AS log_bin,
       @@binlog_format AS binlog_format,
       @@binlog_row_image AS binlog_row_image;
SELECT COUNT(*) AS users_count FROM gip_manual.users;
SELECT COUNT(*) AS orders_count FROM gip_manual.orders;
SELECT id,username,age,amount,HEX(payload),profile,created_at
FROM gip_manual.users ORDER BY id;

