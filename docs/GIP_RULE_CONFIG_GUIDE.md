# GIP 规则配置指南

本文档面向 goInception Plus 的配置和运维人员，说明
`config/config.toml.gip` 中 76 个稳定 RuleID 的用途。运行时的唯一事实来源仍是
`internal/audit/catalog.go`。

## 配置方法

```toml
[rules]
"GIP-DDL-CT-001" = 2
"GIP-DML-SF-001" = 2
```

- `0`：关闭规则，不返回 Issue。
- `1`：Warning，审核结果中可见；是否继续执行还受请求的 `ignore-warnings` 控制。
- `2`：Error，审核失败，默认不执行该批次。

RuleID 必须用引号包裹。未知 RuleID 或非 0/1/2 值会使服务启动失败。
`[rules]` 可与旧 `[inc_level]` 并存；同一规则重复时 `[rules]` 优先。

RuleID 格式为 `GIP-领域-类别-序号`：

- 领域：`COR` 核心、`META` 元数据、`DDL`、`DML`、`EXE` 执行、`BAK` 备份。
- 类别：`ST` 语句、`PS` 解析、`SF` 安全、`MD` 元数据、`CT` 建表/列/索引、
  `AT` ALTER、`IM` 影响行、`RN` 运行、`BL` Binlog。

## 规则与 `[inc]` 的关系

`[rules]` 只决定问题发生后的级别。`[inc]` 保留审核开关、允许列表和数量阈值。
例如：

```toml
[inc]
check_primary_key = true
max_update_rows = 1000
support_engine = "innodb"

[rules]
"GIP-DDL-CT-001" = 2
"GIP-DML-IM-002" = 2
"GIP-DDL-CT-005" = 2
```

`check_primary_key=false` 时不检查主键；`max_update_rows=0` 表示不设影响行数上限。
开关关闭时，即使对应 RuleID 为 1/2，也不会产生该类 Issue。

## 核心、Parser 和元数据

| RuleID | 含义 | 配置建议 |
|---|---|---|
| GIP-COR-ST-001 | SQL已解析，但Plus没有对该语句完成稳定投影或准入 | 保持2，禁止未审核语句透传 |
| GIP-COR-PS-001 | TiDB Parser返回可见warning | 建议1 |
| GIP-COR-PS-002 | SQL语法解析失败 | 保持2 |
| GIP-COR-SF-001 | `audit_only=true`的服务收到execute/backup请求 | 保持2 |
| GIP-META-CN-001 | 连接目标MySQL或读取服务器信息失败 | 保持2 |
| GIP-META-CN-002 | 目标非MySQL 5.7/8.0，或SQL不兼容该目标版本 | 保持2 |
| GIP-META-DB-001 | `USE`或directive指定的库不存在/无权读取 | 保持2 |
| GIP-META-DB-002 | SQL需要schema，但请求没有选择数据库 | 保持2 |

上述规则属于安全边界，不建议设为0。

## DDL建表、字段和索引

| RuleID | 什么情况下触发 | 关联 `[inc]` |
|---|---|---|
| GIP-DDL-MD-001 | DDL引用对象不存在，或内存Schema Snapshot校验失败 | 无 |
| GIP-DDL-CT-001 | CREATE TABLE没有主键 | `check_primary_key` |
| GIP-DDL-CT-002 | CREATE TABLE没有表COMMENT | `check_table_comment` |
| GIP-DDL-CT-003 | CREATE TABLE字段没有COMMENT | `check_column_comment` |
| GIP-DDL-CT-004 | 建表字段数超限 | `max_column_count`，0为不限制 |
| GIP-DDL-CT-005 | 表引擎不是要求的引擎 | `required_engine` |
| GIP-DDL-CT-006 | CREATE/ALTER出现重复字段 | 无 |
| GIP-DDL-CT-007 | CREATE/ALTER出现重复索引名 | 无 |
| GIP-DDL-CT-008 | 索引引用不存在的字段 | 无 |
| GIP-DDL-CT-009 | 库/表/字段/索引名含非法字符或不符合小写策略 | `check_identifier`, `check_identifier_lower` |
| GIP-DDL-CT-010 | 标识符是MySQL或自定义关键字 | `enable_identifer_keyword`, `custom_keywords` |
| GIP-DDL-CT-011 | 普通索引名不符合前缀 | `check_index_prefix`, `index_prefix` |
| GIP-DDL-CT-012 | 唯一索引名不符合前缀 | `check_index_prefix`, `uniq_index_prefix` |
| GIP-DDL-CT-013 | 单个索引的字段数超限 | `max_key_parts` |
| GIP-DDL-CT-014 | 主键字段数超限 | `max_primary_key_parts` |
| GIP-DDL-CT-015 | 表的索引总数超限 | `max_keys` |
| GIP-DDL-CT-016 | 主键字段不是整数类型 | `enable_pk_columns_only_int` |
| GIP-DDL-CT-017 | AUTO_INCREMENT字段不是整数 | `check_autoincrement_datatype` |
| GIP-DDL-CT-018 | AUTO_INCREMENT字段没有UNSIGNED | `enable_autoincrement_unsigned` |
| GIP-DDL-CT-019 | AUTO_INCREMENT字段名不是`id` | 无；不需要时将RuleID设为0 |
| GIP-DDL-CT-020 | 显式AUTO_INCREMENT起始值大于1 | 无 |
| GIP-DDL-CT-021 | 表缺少必须字段 | `must_have_columns` |
| GIP-DDL-CT-022 | 禁止显式指定表字符集时，DDL显式指定了字符集 | `enable_set_charset` |
| GIP-DDL-CT-023 | 显式字符集不在允许列表 | `enable_set_charset`, `support_charset` |
| GIP-DDL-CT-024 | 字段级显式指定CHARSET/COLLATE | `enable_column_charset` |
| GIP-DDL-CT-025 | TEXT/BLOB/JSON等大对象字段设置非NULL默认值 | 无 |
| GIP-DDL-CT-026 | 使用配置未准入的数据类型，如TIMESTAMP/ENUM/SET/BIT | `enable_timestamp_type`, `enable_enum_set_bit` |
| GIP-DDL-CT-027 | 使用JSON类型，但JSON未准入 | `enable_json_type` |
| GIP-DDL-CT-028 | 出现可空的普通字段 | `enable_nullable` |
| GIP-DDL-CT-029 | TEXT/BLOB/JSON字段被声明为NOT NULL | 无 |
| GIP-DDL-CT-030 | 新增的普通字段没有DEFAULT | 无 |
| GIP-DDL-CT-031 | DATETIME字段没有DEFAULT | 无 |
| GIP-DDL-CT-032 | CHAR长度超过建议阈值 | `max_char_length`，0为关闭 |
| GIP-DDL-CT-033 | 自动DEFAULT/ON UPDATE TIMESTAMP字段数量超限 | `check_timestamp_count` |
| GIP-DDL-CT-034 | 两个索引存在冗余的前导字段关系 | `check_index_column_repeat` |
| GIP-DDL-CT-035 | 日期/时间默认值在当前SQL Mode下非法 | `sql_mode` |
| GIP-DDL-CT-036 | 使用ENUM/SET/BIT | `enable_enum_set_bit` |
| GIP-DDL-CT-037 | 使用TEXT/BLOB | `enable_blob_type` |

### ENUM/SET/BIT 双层配置

`enable_enum_set_bit` 决定是否准入这3类字段，`GIP-DDL-CT-036` 决定不准入时的
Issue级别：

| `enable_enum_set_bit` | `GIP-DDL-CT-036` | 最终行为 |
|---|---:|---|
| `true` | 0/1/2 | 允许，不产生Issue |
| `false` | 0 | 不允许策略被级别0关闭，不产生Issue |
| `false` | 1 | 产生Warning |
| `false` | 2 | 产生Error |

完整新旧模板均默认为`enable_enum_set_bit=false`且`GIP-DDL-CT-036=2`，
即禁止ENUM/SET/BIT并按Error返回。`GIP-DDL-CT-026`用于TIMESTAMP等通用未准入类型；
ENUM/SET/BIT使用专用的`GIP-DDL-CT-036`，两者不应视为重复配置。

使用旧`[inc_level].er_use_enum`和新`[rules]."GIP-DDL-CT-036"`同时配置时，
新GIP RuleID的级别优先。

## ALTER与高危DDL

| RuleID | 什么情况下触发 | 关联 `[inc]` |
|---|---|---|
| GIP-DDL-AT-001 | 同一批次对同一表多次ALTER，建议合并 | 无 |
| GIP-DDL-AT-002 | 使用CHANGE COLUMN | 无；允许时设为0 |
| GIP-DDL-AT-003 | FIRST/AFTER调整字段位置 | 无 |
| GIP-DDL-AT-004 | MODIFY/CHANGE造成不安全的字段类型变更 | `check_column_type_change` |
| GIP-DDL-SF-001 | DROP DATABASE | `enable_drop_database` |
| GIP-DDL-SF-002 | DROP TABLE | `enable_drop_table` |
| GIP-DDL-SF-003 | TRUNCATE TABLE | `enable_truncate_table` |
| GIP-DDL-SF-004 | CREATE/ALTER中出现外键 | `enable_foreign_key` |
| GIP-DDL-SF-005 | CREATE/ALTER中出现分区 | `enable_partition_table` |
| GIP-DDL-SF-006 | CREATE/ALTER DATABASE显式指定字符集，但配置禁止 | `enable_set_charset` |
| GIP-DDL-SF-007 | DDL显式指定COLLATE，但配置禁止 | `enable_set_collation` |
| GIP-DDL-SF-008 | DDL显式指定ENGINE，但配置禁止 | `enable_set_engine`, `support_engine` |
| GIP-DDL-SF-009 | CREATE/ALTER/DROP VIEW | 无；当前通过级别控制 |

## DML、表达式和影响行数

| RuleID | 什么情况下触发 | 关联 `[inc]` |
|---|---|---|
| GIP-DML-MD-001 | DML引用的表/字段不存在，或无法解析多表归属 | 无 |
| GIP-DML-SF-001 | UPDATE/DELETE没有WHERE | `check_dml_where` |
| GIP-DML-SF-002 | 需要备份的UPDATE/DELETE表没有可靠主键或非空唯一键 | 备份安全规则 |
| GIP-DML-SF-003 | UPDATE/DELETE包含LIMIT | `check_dml_limit` |
| GIP-DML-SF-004 | UPDATE/DELETE包含ORDER BY | `check_dml_orderby` |
| GIP-DML-SF-005 | JOIN缺少ON条件 | 无 |
| GIP-DML-SF-006 | SELECT使用`*` | `enable_select_star` |
| GIP-DML-SF-007 | INSERT没有显式字段列表 | `check_insert_field` |
| GIP-DML-SF-008 | ORDER BY RAND() | 无 |
| GIP-DML-SF-009 | SET/WHERE中出现可疑的AND表达式 | 无 |
| GIP-DML-SF-010 | 数字列与字符串常量等比较导致隐式转换 | `check_implicit_type_conversion` |
| GIP-DML-IM-001 | EXPLAIN失败或无法安全得到影响行数 | `explain_rule=first|max` |
| GIP-DML-IM-002 | UPDATE/DELETE预计影响行数超限 | `max_update_rows`，0为不限制 |
| GIP-DML-IM-003 | INSERT VALUES行数超限 | `max_insert_rows`，0为不限制 |

## 执行、Binlog备份和回滚

| RuleID | 含义 | 配置建议 |
|---|---|---|
| GIP-EXE-RN-001 | 目标SQL执行失败，包括权限、锁等待、死锁或MySQL返回错误 | 保持2 |
| GIP-BAK-BL-001 | 备份前置不满足：`log_bin=ON`、`binlog_format=ROW`、`binlog_row_image=FULL` | 保持2 |
| GIP-BAK-RN-001 | Binlog捕获、事务归属、回滚SQL生成或写备份库失败 | 保持2，失败后停止后续语句 |

这3条是运行失败而非可宽松的代码规范，生产不应设为0/1。

## 运行时查询和修改

为保持旧Archery兼容，`inception get levels` 继续返回旧规则名，但数值是合并后的
最终有效级别。修改命令接受两种名字：

```sql
inception set level er_table_must_have_pk = 1;
inception set level GIP-DDL-CT-001 = 2;
inception get levels like 'er_table_must_have_pk';
```

运行时修改仅对后续请求生效，不回写TOML，重启后恢复文件配置。

## 建议的生产策略

- 核心、Parser、元数据、执行和备份失败保持2。
- DROP/TRUNCATE、无WHERE DML、无可靠键备份保持2。
- 命名、注释、索引前缀等团队规范可先设1，稳定后提升为2。
- 改动前先用审核模式回放真实SQL语料，不要直接将大量Warning一次性提升为Error。
