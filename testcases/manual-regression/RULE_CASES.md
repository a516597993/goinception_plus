# GIP审核规则完整手工测试用例

本文件覆盖当前`internal/audit/catalog.go`注册的全部RuleID。SQL均使用手册中的审核控制块；除运行期规则外先用`execute=0,backup=0`，防止触发SQL实际执行。

## 1. 通用执行法

对存在规则级别的规则执行以下矩阵：

| 子用例 | 设置 | 触发SQL预期 |
|---|---|---|
| A | RuleID=2，关联开关为“不允许/启用检查” | ErrorLevel=2且消息含目标RuleID |
| B | RuleID=1，关联开关同上 | ErrorLevel=1且消息含目标RuleID |
| C | RuleID=0，关联开关同上 | 不出现目标RuleID |
| D | RuleID=2，使用“通过SQL” | 不出现目标RuleID |

若规则还有准入开关，再增加“开关允许＋RuleID=2”，预期不产生Issue。修改规则级别：

```sql
inception set level GIP-DDL-CT-001 = 2;
inception show levels like 'GIP-DDL-CT-001';
```

`LegacyKey`不为空的规则还要抽样把同一级别写入`[inc_level]`，重启后确认GIP RuleID生效。

## 2. 核心、Parser与元数据规则

| RuleID | LegacyKey | 触发操作 | 通过/对照 | 预期 |
|---|---|---|---|---|
| GIP-COR-ST-001 | - | `PREPARE s FROM 'SELECT 1';`或其他未稳定投影语句 | 已投影的`SELECT id FROM users WHERE id=1` | 未准入语句明确Error |
| GIP-COR-PS-001 | - | 使用Parser可接受但产生warning的语料；以实际warning语料库为准 | 普通合法SQL | Warning可见，不被丢弃 |
| GIP-COR-PS-002 | - | `UPDTE users SET age=1;` | `UPDATE users SET age=1 WHERE id=1` | 语句级Parser Error和位置 |
| GIP-COR-SF-001 | - | 服务`audit_only=true`，请求`execute=1` | `execute=0` | Error且目标数据不变 |
| GIP-META-CN-001 | - | 错host/port/password或撤销元数据权限 | 正确连接 | 连接/权限Error，不当成对象不存在 |
| GIP-META-CN-002 | - | 5.7审核CTE/RENAME COLUMN/8.0专属collation | 8.0审核支持且已投影的对应语法 | 版本不兼容Error |
| GIP-META-DB-001 | - | `USE gip_missing;` | `USE gip_manual;` | schema不存在/不可读取 |
| GIP-META-DB-002 | - | 不带db和USE：`ALTER TABLE users ADD c INT;` | directive指定db | No database selected |

## 3. DDL元数据与建表规则

下表的“通过SQL”可以只写相对触发点的修正版；其他启用规则若同时触发，应在该用例临时设0，使目标RuleID可独立判定。

| RuleID | LegacyKey | 前置/开关 | 触发SQL | 通过SQL |
|---|---|---|---|---|
| GIP-DDL-MD-001 | - | 无 | `ALTER TABLE missing_t ADD c INT;` | `ALTER TABLE users ADD c INT DEFAULT 0;` |
| GIP-DDL-CT-001 | er_table_must_have_pk | `check_primary_key=true` | `CREATE TABLE r1(a INT);` | `CREATE TABLE r1(id BIGINT PRIMARY KEY);` |
| GIP-DDL-CT-002 | er_table_must_have_comment | `check_table_comment=true` | `CREATE TABLE r2(id BIGINT PRIMARY KEY);` | 末尾加`COMMENT='r2'` |
| GIP-DDL-CT-003 | er_column_have_no_comment | `check_column_comment=true` | `CREATE TABLE r3(id BIGINT PRIMARY KEY) COMMENT='r3';` | 字段加`COMMENT 'id'` |
| GIP-DDL-CT-004 | er_max_column_count | `max_column_count=2` | `CREATE TABLE r4(a INT,b INT,c INT);` | 只保留2列 |
| GIP-DDL-CT-005 | er_table_engine | `required_engine=innodb` | `CREATE TABLE r5(id INT) ENGINE=MyISAM;` | `ENGINE=InnoDB` |
| GIP-DDL-CT-006 | er_column_existed | 无 | `CREATE TABLE r6(id INT,id BIGINT);` | 字段名唯一 |
| GIP-DDL-CT-007 | - | 无 | `CREATE TABLE r7(id INT,KEY idx_a(id),KEY idx_a(id));` | 索引名唯一 |
| GIP-DDL-CT-008 | er_column_not_existed | 无 | `CREATE TABLE r8(id INT,KEY idx_x(missing));` | 索引引用id |
| GIP-DDL-CT-009 | er_invalid_ident | `check_identifier=true`；小写测试加`check_identifier_lower=true` | ``CREATE TABLE `bad-name`(id INT);``；`CREATE TABLE UpperName(id INT);` | 合法且符合大小写策略的名字 |
| GIP-DDL-CT-010 | er_ident_use_keyword | `enable_identifer_keyword=true`或配置自定义关键字；该旧键表示“启用关键字检查” | ``CREATE TABLE r10(`select` INT);`` | 非关键字列名 |
| GIP-DDL-CT-011 | er_index_name_idx_prefix | `check_index_prefix=true,index_prefix=idx_` | `KEY bad_name(id)` | `KEY idx_id(id)` |
| GIP-DDL-CT-012 | er_index_name_uniq_prefix | `check_index_prefix=true,uniq_index_prefix=uniq_` | `UNIQUE KEY bad_u(id)` | `UNIQUE KEY uniq_id(id)` |
| GIP-DDL-CT-013 | er_too_many_key_parts | `max_key_parts=2` | `KEY idx_abc(a,b,c)` | 索引最多2列 |
| GIP-DDL-CT-014 | er_pk_too_many_parts | `max_primary_key_parts=2` | `PRIMARY KEY(a,b,c)` | 主键最多2列 |
| GIP-DDL-CT-015 | er_too_many_keys | `max_keys=2` | 建表定义3个普通/唯一索引 | 索引总数不超过2 |
| GIP-DDL-CT-016 | er_pk_cols_not_int | `enable_pk_columns_only_int=true` | `CREATE TABLE r16(id VARCHAR(16) PRIMARY KEY);` | BIGINT主键 |
| GIP-DDL-CT-017 | er_set_data_type_int_bigint | `check_autoincrement_datatype=true` | `id DECIMAL(10,0) AUTO_INCREMENT PRIMARY KEY` | BIGINT AUTO_INCREMENT |
| GIP-DDL-CT-018 | er_autoinc_unsigned | `enable_autoincrement_unsigned=true` | `id BIGINT AUTO_INCREMENT PRIMARY KEY` | `BIGINT UNSIGNED` |
| GIP-DDL-CT-019 | er_auto_incr_id_warning | RuleID控制 | `seq BIGINT AUTO_INCREMENT PRIMARY KEY` | 字段名id |
| GIP-DDL-CT-020 | er_inc_init_err | RuleID控制 | 建表尾部`AUTO_INCREMENT=100` | 不写或`AUTO_INCREMENT=1` |
| GIP-DDL-CT-021 | er_must_have_columns | `must_have_columns=created_at` | 建表不含created_at | 包含该字段 |
| GIP-DDL-CT-022 | er_table_charset_must_null | `enable_set_charset=false` | 建表显式`DEFAULT CHARSET=utf8mb4` | 不显式指定 |
| GIP-DDL-CT-023 | er_table_charset_must_utf8 | `enable_set_charset=true,support_charset=utf8mb4` | `DEFAULT CHARSET=latin1` | utf8mb4 |
| GIP-DDL-CT-024 | er_charset_on_column | `enable_column_charset=false` | `name VARCHAR(20) CHARACTER SET latin1` | 字段不显式字符集/排序规则 |
| GIP-DDL-CT-025 | er_blob_cant_have_default | RuleID控制 | `b BLOB DEFAULT 'x'`或`t TEXT DEFAULT 'x'` | 无非NULL默认值 |
| GIP-DDL-CT-026 | er_invalid_data_type | 对应类型准入开关关闭 | 例如`ts TIMESTAMP`且`enable_timestamp_type=false` | 使用已准入类型 |
| GIP-DDL-CT-027 | er_json_type_support | `enable_json_type=false` | `doc JSON` | 开关true或使用VARCHAR |
| GIP-DDL-CT-028 | er_not_allowed_nullable | `enable_nullable=false` | `name VARCHAR(20) NULL` | `NOT NULL DEFAULT ''` |
| GIP-DDL-CT-029 | er_text_not_nullable_error | RuleID控制 | `content TEXT NOT NULL` | TEXT允许NULL |
| GIP-DDL-CT-030 | er_with_default_add_column | RuleID控制 | `ALTER TABLE users ADD COLUMN c INT NOT NULL;` | 加`DEFAULT 0` |
| GIP-DDL-CT-031 | er_datetime_default | RuleID控制 | `ALTER TABLE users ADD COLUMN dt DATETIME NOT NULL;` | 加合法DEFAULT |
| GIP-DDL-CT-032 | er_char_to_varchar_len | `max_char_length=8` | `code CHAR(16)` | CHAR(8)或VARCHAR(16) |
| GIP-DDL-CT-033 | er_too_much_auto_timestamp_cols | `check_timestamp_count=true` | 两列同时使用自动DEFAULT/ON UPDATE | 只保留允许数量 |
| GIP-DDL-CT-034 | - | `check_index_column_repeat=true` | 同时定义`idx_a(a)`和`idx_ab(a,b)` | 去掉冗余前导索引 |
| GIP-DDL-CT-035 | er_incorrect_datetime_value | 严格sql_mode | `dt DATETIME DEFAULT '0000-00-00 00:00:00'` | 合法日期或CURRENT_TIMESTAMP |
| GIP-DDL-CT-036 | er_use_enum | `enable_enum_set_bit=false` | 分别测试`ENUM('a','b')`、`SET('a','b')`、`BIT(8)` | 开关true或普通类型 |
| GIP-DDL-CT-037 | er_use_text_or_blob | `enable_blob_type=false` | 分别测试TEXT/BLOB | 开关true或VARCHAR/VARBINARY |

### ENUM/SET/BIT强制组合矩阵

三种类型分别完整执行：

| enable_enum_set_bit | GIP-DDL-CT-036 | 预期 |
|---|---:|---|
| true | 0/1/2 | 不产生Issue |
| false | 0 | 不产生Issue |
| false | 1 | Warning |
| false | 2 | Error |

同时核对`config.toml.default`和`config.toml.gip`默认均为`false + 2`。

## 4. ALTER和DDL安全规则

在以下用例前执行`setup.sql`恢复`users`结构；类型变更必须从真实元数据加载旧字段。

| RuleID | LegacyKey | 前置/开关 | 触发SQL | 通过/对照 |
|---|---|---|---|---|
| GIP-DDL-AT-001 | er_alter_table_once | RuleID控制 | 同批两条`ALTER TABLE users ADD c1 INT; ALTER TABLE users ADD c2 INT;` | 合并为一条ALTER |
| GIP-DDL-AT-002 | er_cant_change_column | RuleID控制 | `ALTER TABLE users CHANGE COLUMN age user_age INT NOT NULL DEFAULT 0;` | 不用CHANGE或规则0 |
| GIP-DDL-AT-003 | er_cant_change_column_position | RuleID控制 | `MODIFY age INT FIRST`及`... AFTER username` | 不调整位置 |
| GIP-DDL-AT-004 | er_change_column_type | `check_column_type_change=true` | `MODIFY username VARCHAR(8)`、`MODIFY amount INT` | 不缩短/不危险改型 |
| GIP-DDL-SF-001 | er_cant_drop_database | `enable_drop_database=false` | `DROP DATABASE gip_manual;` | 开关true时仍只在隔离库测试 |
| GIP-DDL-SF-002 | er_cant_drop_table | `enable_drop_table=false` | `DROP TABLE users;` | 开关true/规则0 |
| GIP-DDL-SF-003 | er_cant_truncate_table | `enable_truncate_table=false` | `TRUNCATE TABLE users;` | 开关true/规则0 |
| GIP-DDL-SF-004 | er_foreign_key | `enable_foreign_key=false` | 建表或ALTER加`FOREIGN KEY(user_id) REFERENCES users(id)` | 不使用外键 |
| GIP-DDL-SF-005 | er_partition_not_allowed | `enable_partition_table=false` | 建表尾部`PARTITION BY HASH(id) PARTITIONS 2` | 不分区 |
| GIP-DDL-SF-006 | er_cant_set_charset | `enable_set_charset=false` | `ALTER DATABASE gip_manual CHARACTER SET utf8mb4;` | 不显式设置或开关true |
| GIP-DDL-SF-007 | er_cant_set_collation | `enable_set_collation=false` | `ALTER DATABASE gip_manual COLLATE utf8mb4_general_ci;` | 不显式设置或开关true |
| GIP-DDL-SF-008 | er_cant_set_engine | `enable_set_engine=false` | `ALTER TABLE users ENGINE=InnoDB;` | 不显式设置或开关true且在support_engine中 |
| GIP-DDL-SF-009 | er_view_support | RuleID控制 | 分别`CREATE VIEW`、`DROP VIEW`；ALTER VIEW若Parser不支持应返回Parser Unsupported | 规则0时对已稳定投影语法放行 |

VIEW测试：

```sql
CREATE VIEW v_users AS SELECT id,username FROM users;
DROP VIEW v_users;
ALTER VIEW v_users AS SELECT id FROM users;
```

第三条当前若TiDB 8.5 Parser不能稳定解析，必须明确不支持，不能为了RuleID=0而透传。

## 5. DML审核规则

| RuleID | LegacyKey | 前置/开关 | 触发SQL | 通过SQL |
|---|---|---|---|---|
| GIP-DML-MD-001 | - | 元数据可读 | `UPDATE missing_t SET a=1 WHERE id=1;`；`UPDATE users SET missing=1 WHERE id=1;` | 表列均存在 |
| GIP-DML-SF-001 | er_no_where_condition | `check_dml_where=true` | `UPDATE users SET age=1;`及`DELETE FROM users;` | 加可靠WHERE |
| GIP-DML-SF-002 | - | backup=1 | `UPDATE no_key SET value_num=3 WHERE value_text='a';` | 使用有主键/非空唯一键表 |
| GIP-DML-SF-003 | er_with_limit_condition | `check_dml_limit=true` | `UPDATE users SET age=1 WHERE id>0 LIMIT 1;` | 去掉LIMIT |
| GIP-DML-SF-004 | er_with_orderby_condition | `check_dml_orderby=true` | `DELETE FROM users WHERE id>0 ORDER BY id LIMIT 1;` | 去掉ORDER BY |
| GIP-DML-SF-005 | er_join_no_on_condition | RuleID控制 | `SELECT u.id FROM users u JOIN orders o;` | `... JOIN orders o ON o.user_id=u.id` |
| GIP-DML-SF-006 | er_select_only_star | `enable_select_star=false` | `SELECT * FROM users WHERE id=1;` | 显式字段 |
| GIP-DML-SF-007 | er_with_insert_field | `check_insert_field=true` | `INSERT INTO users VALUES(...);` | `INSERT INTO users(username,age,...) VALUES(...)` |
| GIP-DML-SF-008 | er_ordery_by_rand | RuleID控制 | `SELECT id FROM users ORDER BY RAND();` | 确定性ORDER BY |
| GIP-DML-SF-009 | er_wrong_and_expr | RuleID控制 | 使用可疑常量AND表达式，如`WHERE id=1 AND 1` | 完整布尔谓词 |
| GIP-DML-SF-010 | er_implicit_type_conversion | `check_implicit_type_conversion=true` | `int_col='1'`、`1=str_col`、`time_col=1`、`json_col=TRUE` | 同类型字面量、NULL、unknown不误报 |
| GIP-DML-IM-001 | - | 开启影响评估 | 撤销EXPLAIN权限或构造无法安全评估的DML | 可成功EXPLAIN的谓词 |
| GIP-DML-IM-002 | er_change_too_much_rows | `max_update_rows=1` | `UPDATE users SET age=age+1 WHERE id IN(1,2);` | 只影响1行或提高阈值 |
| GIP-DML-IM-003 | er_insert_too_much_rows | `max_insert_rows=1` | 两行`INSERT ... VALUES(...),(...)` | 只插1行或提高阈值 |

### Comparison语义类型专项

以下矩阵需在字段位于比较符左右两侧各测一次：

| 列类型 | literal | 预期 |
|---|---|---|
| INT | `1`、`-1`、超大UNSIGNED、DECIMAL/FLOAT | 不因string转换误报 |
| INT | `'1'`、`X'31'` | GIP-DML-SF-010 |
| VARCHAR | `'1'` | 不误报 |
| VARCHAR | `1`、TRUE/FALSE | GIP-DML-SF-010 |
| DATETIME | 合法时间字符串 | 不因数字转换误报 |
| DATETIME | `1` | GIP-DML-SF-010 |
| JSON | JSON/字符串语义输入 | 按投影策略审核 |
| JSON | `1`、TRUE/FALSE | GIP-DML-SF-010 |
| 任意 | NULL或unknown | 不误报 |

结果消息或内部输出不得出现`int64`、`uint8`、`[]uint8`、`types.Datum`等Go/TiDB反射类型名。

## 6. 执行、备份运行期规则

| RuleID | 触发操作 | 预期与数据检查 |
|---|---|---|
| GIP-EXE-RN-001 | execute=1执行重复主键INSERT、权限失败、锁超时或死锁 | 当前语句Error；记录MySQL错误；后续语句停止 |
| GIP-BAK-BL-001 | backup=1且log_bin关闭、非ROW或非FULL；也测无复制权限 | 执行前失败，目标数据不变 |
| GIP-BAK-RN-001 | 备份期间停备份库、撤写权限、破坏binlog捕获连接 | 目标已执行时标严重失败；停止后续；不能报完整成功 |

## 7. 完整性核对

完成后把实际RuleID集合与目录比较：

```powershell
rg 'Rule[A-Za-z].*= "GIP-' .\internal\audit\catalog.go
```

验收要求：

- 本文每个catalog RuleID均至少有一个触发结果或明确的运行期故障记录。
- 每个LegacyKey至少抽样一次旧配置加载；别名`er_column_must_have_comment`、`er_insert_field`、`er_sql_no_where`也要验证。
- 配置为0的规则不产生Issue；配置为1/2时级别准确。
- 同一个SQL可能触发多个规则；测试单条规则时应临时关闭非目标规则，避免误判。
- 任何“无法构造触发SQL”的规则都不能直接记为通过，必须登记为测试阻塞并补语料或说明实现边界。
