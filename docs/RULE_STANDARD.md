# Rule 文件与统一编码规范

## 1. 稳定编码

规则编码格式：

    GIP-{DOMAIN}-{SCENE}-{NNN}

字段含义：

| 字段 | 规则 |
|---|---|
| GIP | 项目固定前缀 |
| DOMAIN | COR、DDL、DML、META、EXE、BAK、PRO |
| SCENE | 两位大写场景码，如 CT 表示 CREATE TABLE |
| NNN | 场景内三位递增编号；发布后永不复用 |

示例：

| 编码 | 含义 |
|---|---|
| GIP-COR-ST-001 | 未准入语句类型 |
| GIP-DDL-CT-001 | CREATE TABLE 必须有主键 |
| GIP-DDL-CT-002 | CREATE TABLE 必须有表注释 |
| GIP-DDL-CT-003 | CREATE TABLE 字段必须有注释 |
| GIP-DDL-CT-004 | CREATE TABLE 最大字段数 |

编码一旦进入发布版本，不得修改含义。废弃规则保留注册项并标记 deprecated，
不得把旧编号分配给新规则。

## 2. Domain 与 Scene

| Domain | 范围 |
|---|---|
| COR | Parser、请求和内核安全约束 |
| DDL | 数据库及表结构审核 |
| DML | INSERT、UPDATE、DELETE、SELECT 审核 |
| META | 元数据完整性和兼容性 |
| EXE | 执行安全和故障状态 |
| BAK | 备份与回滚完整性 |
| PRO | MySQL 协议和结果兼容 |

常用 Scene：

| Scene | 场景 |
|---|---|
| ST | 通用 Statement |
| DB | Database DDL |
| CT | CREATE TABLE |
| AT | ALTER TABLE |
| DT | DROP/TRUNCATE TABLE |
| IX | Index |
| IN | INSERT |
| UP | UPDATE |
| DE | DELETE |
| SE | SELECT |

## 3. 注册表要求

所有可返回给用户的规则必须先登记在 internal/audit/catalog.go，并包含：

- Code：稳定编码。
- LegacyKey：旧 goInception 配置或错误级键；无对应项时为空。
- DefaultLevel：默认级别。
- Phase：首次交付阶段。
- Summary：不带动态值的一句话说明。

ValidateRuleCatalog 必须保证：

- 编码格式合法。
- 编码全局唯一。
- 非空 LegacyKey 全局唯一。

规则实现中禁止手写编码字符串，只能引用注册表常量。

## 4. 文件和类型命名

目录按领域组织：

    internal/audit/rules/
      core.go
      ddl_create_table.go
      ddl_alter_table.go
      ddl_index.go
      dml_insert.go
      dml_update.go
      dml_delete.go

规则类型使用业务行为命名，不带 Check 前缀。例如：

    RequirePrimaryKey
    RequireTableComment
    LimitColumnCount

聚合执行器只负责调度，不得成为包含数十条规则的超大函数。

## 5. 实现约束

每条规则必须满足：

1. 确定性：相同 Statement、Policy、Metadata 得到相同结果。
2. 无副作用：不得修改数据库、全局配置或输入 AST。
3. 无直接 I/O：只能通过 AuditContext 的接口读取元数据。
4. Parser 隔离：不得导入 TiDB ast/parser 包。
5. 明确适用范围：非目标 StatementKind 立即返回 nil。
6. 安全失败：缺少必要元数据时返回明确问题，不得按通过处理。
7. 参数化消息：RuleDefinition 保持稳定，动态对象只进入 AuditIssue.Message。
8. 可配置级别：从 Policy.Level(code) 获取，不硬编码告警级别。

## 6. 测试规范

每条规则至少包含：

- 命中案例。
- 不命中案例。
- 关闭规则案例。
- Warning 和 Error 两种级别案例。
- MySQL 5.7/8.0 差异案例（存在差异时）。
- 旧 goInception 对照案例（有 LegacyKey 时）。

测试名使用：

    TestRule_{Code}_{Condition}_{Expected}

例如：

    TestRule_GIP_DDL_CT_001_MissingPrimaryKey_Warning

禁止只断言 issues 数量；必须同时断言编码、级别、序号和关键消息。

## 7. 新增规则流程

1. 在 ROADMAP 对应阶段确认范围。
2. 在 catalog.go 分配唯一编码和 LegacyKey。
3. 在稳定 AST 投影或 Metadata 模型补充最小字段。
4. 在对应领域文件实现规则。
5. 添加完整测试矩阵。
6. 在规则目录表记录状态和旧版映射。
7. 运行 gofmt、go test ./... 和 git diff --check。

