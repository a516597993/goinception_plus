# Rule 目录

| Code | Legacy key | 默认级别 | 阶段 | 状态 | 说明 |
|---|---|---:|---:|---|---|
| GIP-COR-ST-001 | - | Error | 1 | 已实现 | 拒绝未准入的语句类型 |
| GIP-COR-PS-001 | - | Warning | 1 | 已实现 | 展示 Parser warning |
| GIP-COR-PS-002 | - | Error | 1 | 已实现 | 将 Parser error 转为语句级审核结果 |
| GIP-COR-SF-001 | - | Error | 7 | 已实现 | audit-only 服务拒绝执行或备份 |
| GIP-META-CN-001 | - | Error | 2 | 已实现 | 目标 MySQL 元数据连接失败 |
| GIP-META-CN-002 | - | Error | 2 | 已实现 | 目标版本或版本专属语法不兼容 |
| GIP-META-DB-001 | - | Error | 2 | 已实现 | 数据库不存在或不可读取 |
| GIP-META-DB-002 | - | Error | 2 | 已实现 | 对象语句没有选择数据库 |
| GIP-DDL-MD-001 | - | Error | 2 | 已实现 | DDL 对象或 Snapshot 校验失败 |
| GIP-DDL-CT-001 | er_table_must_have_pk | Warning | 2 | 已实现 | CREATE TABLE 必须有主键 |
| GIP-DDL-CT-002 | er_table_must_have_comment | Warning | 2 | 已实现 | CREATE TABLE 必须有表注释 |
| GIP-DDL-CT-003 | er_column_must_have_comment | Warning | 2 | 已实现 | CREATE TABLE 字段必须有注释 |
| GIP-DDL-CT-004 | er_max_column_count | Warning | 2 | 已实现 | CREATE TABLE 最大字段数 |
| GIP-DDL-CT-005 | er_table_engine | Error | 2 | 已实现 | 表引擎必须符合配置 |
| GIP-DDL-CT-006 | er_column_existed | Error | 2 | 已实现 | 禁止重复字段 |
| GIP-DDL-CT-007 | er_index_name_idx_prefix | Error | 2 | 已实现 | 禁止重复索引名 |
| GIP-DDL-CT-008 | er_column_not_existed | Error | 2 | 已实现 | 索引字段必须存在 |
| GIP-DDL-CT-009 | er_invalid_ident | Error | 2 | 已实现 | 标识符只能使用允许字符 |
| GIP-DDL-CT-010 | er_ident_use_keyword | Error | 2 | 已实现 | 禁止关键字标识符 |
| GIP-DDL-CT-011 | er_index_name_idx_prefix | Warning | 2 | 已实现 | 普通索引名前缀 |
| GIP-DDL-CT-012 | er_index_name_uniq_prefix | Warning | 2 | 已实现 | 唯一索引名前缀 |
| GIP-DDL-CT-013 | er_too_many_key_parts | Error | 2 | 已实现 | 索引字段数上限 |
| GIP-DDL-CT-014 | er_pk_too_many_parts | Error | 2 | 已实现 | 主键字段数上限 |
| GIP-DDL-CT-015 | er_too_many_keys | Warning | 2 | 已实现 | 表索引数上限 |
| GIP-DDL-CT-016 | er_pk_cols_not_int | Warning | 2 | 已实现 | 主键字段建议使用整数 |
| GIP-DDL-CT-017 | er_set_data_type_int_bigint | Warning | 2 | 已实现 | 自增字段必须使用整数 |
| GIP-DDL-CT-018 | er_autoinc_unsigned | Warning | 2 | 已实现 | 自增字段建议使用 unsigned |
| GIP-DDL-CT-019 | er_auto_incr_id_warning | Warning | 2 | 已实现 | 自增字段建议命名为 ID |
| GIP-DDL-CT-020 | er_inc_init_err | Warning | 2 | 已实现 | AUTO_INCREMENT 初始值应为1 |
| GIP-DDL-CT-021 | er_must_have_columns | Error | 2 | 已实现 | 表必须包含配置字段 |
| GIP-DDL-CT-022 | er_table_charset_must_null | Error | 2 | 已实现 | 禁止显式表字符集 |
| GIP-DDL-CT-023 | er_table_charset_must_utf8 | Error | 2 | 已实现 | 表字符集允许列表 |
| GIP-DDL-CT-024 | er_charset_on_column | Error | 2 | 已实现 | 禁止字段字符集或排序规则 |
| GIP-DDL-CT-025 | er_blob_cant_have_default | Warning | 2 | 已实现 | 大对象字段禁止默认值 |
| GIP-DDL-CT-026 | er_invalid_data_type | Warning | 2 | 已实现 | 禁止未启用的数据类型 |
| GIP-DDL-CT-027 | er_json_type_support | Warning | 2 | 已实现 | JSON 类型开关 |
| GIP-DDL-CT-028 | er_not_allowed_nullable | Warning | 2 | 已实现 | 禁止字段可空 |
| GIP-DDL-CT-029 | er_text_not_nullable_error | Warning | 2 | 已实现 | TEXT/BLOB 禁止 NOT NULL |
| GIP-DDL-CT-030 | er_with_default_add_column | Error | 2 | 已实现 | 新增字段必须有默认值 |
| GIP-DDL-CT-031 | er_datetime_default | Warning | 2 | 已实现 | DATETIME 字段必须有默认值 |
| GIP-DDL-CT-032 | er_char_to_varchar_len | Warning | 2 | 已实现 | 长 CHAR 建议改为 VARCHAR |
| GIP-DDL-CT-033 | er_too_much_auto_timestamp_cols | Warning | 2 | 已实现 | 自动时间戳字段数量上限 |
| GIP-DDL-CT-034 | - | Warning | 2 | 已实现 | 禁止冗余索引前导字段 |
| GIP-DDL-CT-035 | er_incorrect_datetime_value | Error | 2 | 已实现 | 禁止非法日期时间值 |
| GIP-DDL-CT-036 | er_use_enum | Error | 2 | 已实现 | 禁止 ENUM 类型 |
| GIP-DDL-CT-037 | er_use_text_or_blob | Warning | 2 | 已实现 | TEXT/BLOB 类型开关 |
| GIP-DDL-AT-001 | er_alter_table_once | Warning | 2 | 已实现 | 同一表 ALTER 应合并 |
| GIP-DDL-AT-002 | er_cant_change_column | Warning | 2 | 已实现 | 禁止 CHANGE COLUMN |
| GIP-DDL-AT-003 | er_cant_change_column_position | Warning | 2 | 已实现 | 禁止调整字段位置 |
| GIP-DDL-AT-004 | er_change_column_type | Warning | 2 | 已实现 | 字段类型变更风险 |
| GIP-DDL-SF-001 | er_cant_drop_database | Error | 2 | 已实现 | 默认禁止 DROP DATABASE |
| GIP-DDL-SF-002 | er_cant_drop_table | Error | 2 | 已实现 | 默认禁止 DROP TABLE |
| GIP-DDL-SF-003 | er_cant_truncate_table | Error | 2 | 已实现 | 默认禁止 TRUNCATE TABLE |
| GIP-DDL-SF-004 | er_foreign_key | Error | 2 | 已实现 | 默认禁止外键 |
| GIP-DDL-SF-005 | er_partition_not_allowed | Error | 2 | 已实现 | 默认禁止分区表 |
| GIP-DDL-SF-006 | er_cant_set_charset | Error | 2 | 已实现 | 禁止显式字符集 |
| GIP-DDL-SF-007 | er_cant_set_collation | Error | 2 | 已实现 | 禁止显式排序规则 |
| GIP-DDL-SF-008 | er_cant_set_engine | Error | 2 | 已实现 | 禁止显式存储引擎 |
| GIP-DDL-SF-009 | er_view_support | Error | 2 | 已实现 | 禁止VIEW |
| GIP-DML-MD-001 | - | Error | 3 | 已实现 | DML 表和引用字段必须存在 |
| GIP-DML-SF-001 | er_sql_no_where | Error | 3 | 已实现 | UPDATE/DELETE 必须包含 WHERE |
| GIP-DML-SF-002 | - | Error | 3 | 已实现 | 备份 DML 必须有可靠主键或非空唯一键 |
| GIP-DML-SF-003 | er_with_limit_condition | Warning | 3 | 已实现 | UPDATE/DELETE 包含 LIMIT |
| GIP-DML-SF-004 | er_with_orderby_condition | Warning | 3 | 已实现 | UPDATE/DELETE 包含 ORDER BY |
| GIP-DML-SF-005 | er_join_no_on_condition | Error | 3 | 已实现 | JOIN 必须包含 ON 条件 |
| GIP-DML-SF-006 | er_select_only_star | Warning | 3 | 已实现 | 禁止 SELECT * |
| GIP-DML-SF-007 | er_insert_field | Warning | 3 | 已实现 | INSERT 必须显式指定字段 |
| GIP-DML-SF-008 | er_ordery_by_rand | Error | 3 | 已实现 | 禁止 ORDER BY RAND() |
| GIP-DML-SF-009 | er_wrong_and_expr | Warning | 3 | 已实现 | 禁止可疑 AND 表达式 |
| GIP-DML-SF-010 | er_implicit_type_conversion | Error | 3 | 已实现 | 禁止隐式类型转换 |
| GIP-DML-IM-001 | - | Error | 3 | 已实现 | 无法安全评估 DML 影响行数 |
| GIP-DML-IM-002 | er_change_too_much_rows | Error | 3 | 已实现 | 预计影响行数超过配置上限 |
| GIP-DML-IM-003 | er_insert_too_much_rows | Error | 3 | 已实现 | INSERT VALUES 行数超过配置上限 |
| GIP-EXE-RN-001 | - | Error | 4 | 已实现 | 目标 SQL 执行失败 |
| GIP-BAK-BL-001 | - | Error | 5 | 已实现 | ROW/FULL Binlog 前置条件不满足 |
| GIP-BAK-RN-001 | - | Error | 5 | 已实现 | Binlog捕获、回滚生成或备份写入失败 |

共 76 条。本表是面向开发者的可读索引；`internal/audit/catalog.go` 是运行时唯一事实来源。新增或调整规则时必须同步本表，并由 Catalog 一致性测试阻止漏登记或重复编码。
