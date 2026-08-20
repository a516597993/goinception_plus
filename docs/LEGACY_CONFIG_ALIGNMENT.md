# 旧 goInception 配置对齐

## 新旧规则名并行

`config/config.toml.default` 保留旧 `[inc_level]` 规则名，供 Archery 和现有运维脚本继续使用。
`config/config.toml.gip` 是等价的新 RuleID 完整模板，使用 `[rules]` 及稳定 GIP RuleID。
两个节可以在同一文件中并存；同一规则重复配置时 `[rules]` 优先，值仍为
0（关闭）、1（Warning）、2（Error）。

```toml
[inc_level]
er_table_must_have_pk = 1

[rules]
"GIP-DDL-CT-001" = 2 # 最终生效为 Error
```

事实来源为 `config (1).toml`、旧 goInception源码与测试。Plus使用 `config/config.toml.default` 作为完整模板，`config/config.minimal.toml` 作为最小启动模板；旧文件不能原样启动Plus。

## 配置迁移

| 旧位置 | Plus位置 | 处理 |
|---|---|---|
| 顶层 `host/port` | `[server].host/port` | 必须迁移 |
| `[inc].backup_*` | `[backup].host/port/user/password` | Loader兼容旧键；不得和新节同时配置 |
| `[inc].max_allowed_packet` | `[server].max_allowed_packet` | Loader在新键缺失时自动迁移 |
| `[inc]`审核与会话键 | `[inc]`同名键 | 保持旧语义 |
| `[inc_level]` | `[inc_level]` | 0关闭、1 Warning、2 Error |
| `[osc]`、`[ghost]` | 无 | 不支持，不静默执行 |
| `path`、`ignore_sighup`、旧日志轮转 | 无 | 不属于Plus运行模型 |

目标文件的41个`[inc]`键均有显式结构映射，52个`[inc_level]`键均映射到稳定GIP RuleID。契约测试会在漏映射时失败。

## 兼容别名

| 旧/兼容键 | 规范键 | GIP规则 |
|---|---|---|
| `er_column_must_have_comment` | `er_column_have_no_comment` | `GIP-DDL-CT-003` |
| `er_insert_field` | `er_with_insert_field` | `GIP-DML-SF-007` |
| `er_sql_no_where` | `er_no_where_condition` | `GIP-DML-SF-001` |

管理命令接受两侧别名，但`inception get levels`只展示规范旧键。

## 规则分组

- 字段类型：`er_invalid_data_type`、`er_use_enum`、`er_use_text_or_blob`、`er_json_type_support`、默认值、NULL、DATETIME和自动时间戳。
- 标识符：`er_invalid_ident`、`er_ident_use_keyword`、大小写、自定义关键字。
- 索引：普通/唯一索引前缀、字段数、索引数、重复前导字段、主键字段数与类型。
- ALTER：合并ALTER、CHANGE、字段位置和不安全类型转换。
- DML：WHERE、LIMIT、ORDER BY、JOIN ON、SELECT星号、INSERT字段、错误AND、ORDER BY RAND及隐式转换。
- DDL准入：字符集、排序规则、引擎、VIEW、外键、分区和DROP。

完整编码、Legacy key、默认级别和实施状态见 `RULE_CATALOG.md`。

## 会话语义

- 非空`sql_mode`同时作用于Parser和执行连接；空值使用目标MySQL实际模式。
- `sql_safe_updates`和`lock_wait_timeout`取值`-1`时不设置，其他合法值在执行前设置到专用连接。
- `default_charset`执行为安全校验后的`SET NAMES`。
- `explain_rule=first|max`分别取EXPLAIN第一行或最大`rows`。
- `skip_grant_table=false`时读取当前目标账号的GRANTS并做数据库级前置检查。
- Runtime修改只影响后续请求，重启后恢复TOML。

## 验收要求

1. 每个规则至少有通过、触发、级别0和级别1/2测试。
2. MySQL 5.7与8.0分别运行元数据、版本差异及会话集成测试。
3. Archery验证审核、Warning/Error门禁、执行、固定12列结果和回滚SQL。
4. 最终运行`go test ./...`、`go vet ./...`、Linux race和Windows构建。
