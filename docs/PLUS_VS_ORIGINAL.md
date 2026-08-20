# goInception Plus vs 原版 goInception：优化 / 升级 / 新增功能

> 适用版本：goInception Plus `0.7.0-dev`（基于 TiDB 8.5.3 Parser 重构）
> 对比对象：原版 goInception（[hanchuanchuan/goInception](https://github.com/hanchuanchuan/goInception)，基于旧 TiDB 3.x 派生）
> 结论：在 "审计 / 执行 / 备份 / 回滚" 核心闭环上可平替原版；**主动放弃 OSC 与内嵌 gh-ost 无锁 DDL**（业务决策，非能力缺失）。

---

## 1. 架构级重构（优化）

| 维度 | 原版 goInception | goInception Plus |
|------|------------------|------------------|
| 内核依赖 | 派生自旧 TiDB，耦合 TiKV/PD/Session/Planner/Executor/DDL/Domain，臃肿且难升级 | **Clean-room 重构**，仅依赖 TiDB 8.5.3 的 Parser/AST，**不依赖 TiKV、PD、Session、Planner、Executor、DDL、Domain、MockTiKV** |
| Parser | 旧 TiDB SQL 解析，长期滞后 | 升级到 **TiDB 8.5.3 Parser**，对 MySQL 8.0 新语法（CTE、窗口函数、 invisible index、generated column、role 等）支持更完整 |
| AST 边界 | 业务规则直接持有 TiDB AST 内部类型 | 引入 `internal/model` 稳定业务模型（`DDLSpec` / `DMLSpec` / `ColumnSpec` / `IndexSpec`），**TiDB AST 不越过 `internal/parser` 边界**，规则层与解析层解耦 |
| 规则定义 | 散落在代码中的 `er_*` 隐式规则，缺统一 ID 与文档 | **统一规则目录 `catalog.go`**，86 条规则带 `GIP-XXX` 稳定编号（`GIP-COR/META/DDL/DML/EXE/BAK`），并兼容老 `er_*` key 双映射 |
| 策略体系 | 单点 `inc_level` 开关，无分级回退 | `Policy` = `Legacy` 开关 map + `RuleLevels` 分级 map + `Level()` 三级回退（RuleLevels → 目录默认 → Warning） |

**收益**：内核可独立小版本升级而不牵连业务；规则可枚举、可测试、可文档化；新增 MySQL 8.0 语法支持随 Parser 升级自动获得。

---

## 2. 引擎与执行链路升级

| 维度 | 原版 | Plus |
|------|------|------|
| 执行连接 | 长连接池 / 复杂连接管理 | **请求级单物理连接 + 审计闸门（audit gate）**，连接语义更清晰、可控 |
| 元数据 | 全量/粗放缓存 | **请求级元数据缓存（single-flight）+ Schema 快照演进**，避免重复查询、保证同请求内 schema 一致 |
| 版本兼容 | 被动兼容 | 显式 `supportedMySQLVersion()` 拒绝 MariaDB / TiDB 作为目标（fail-closed），明确支持 **MySQL 5.7 / 8.0** |
| 失败策略 | 部分场景静默跳过 | **Fail-closed**：备份前置校验失败、备份执行失败、回滚无法生成（DROP DATABASE / TRUNCATE / DROP INDEX）均显式报错，不静默放行 |

---

## 3. 审计规则质量升级

- **86 条规则全量落地**，覆盖原版 `inc_level` 全部 key（`levelBindings` 50 项 + `legacyLevelRule` GIP 映射）。
- **DDL 规则 36 条**（`LegacyDDL.Check()`）：表/列/索引/字符集/主键/自增/外键/分区等，按语义枚举判定而非字符串 `%T` 反射（修复了旧式 `LiteralKind` 误判漏洞）。
- **DML 安全规则**（`DMLSafety.Check()`）：WHERE 缺失、无 LIMIT、无 ORDER BY 的 UPDATE/DELETE、JOIN 更新、SELECT *、INSERT 缺字段、ORDER BY RAND、错误的 AND、隐式类型转换（`implicitConversion()` 基于语义 `LiteralKind`，而非反射）。
- **隐式转换检测**：用 `GetType().GetFlag()` 得到语义字面量类型，避免旧实现按 Go 类型反射导致的漏报/误报。
- **时间值校验**：`sql_mode=TRADITIONAL` 下触发 `GIP-DDL-...-datetime` 类规则（对应原版 `er_incorrect_datetime_value`），修复旧版静默放行的漏洞。
- **自增开关对齐**：`check_autoincrement_datatype` / `enable_autoincrement_unsigned` 等开关已接线并测试。

---

## 4. 备份 / 回滚升级

| 维度 | 原版 | Plus |
|------|------|------|
| 备份机制 | 依赖目标库 binlog 解析 | **ROW Binlog 捕获**（binlogmysql）：按 `CONNECTION_ID` 线程号过滤、XID 事件清场、行数校验 **fail-closed**（`len(changes)==expected` 不匹配即报错） |
| 备份目标 | 指定备份库 | **可配置任意 MySQL 实例**（本地或远程），通过 `[backup]` host/port/user/password（`TargetOptions`）——**本地备份库与远程备份库均支持** |
| 存储契约 | `IP_PORT_schema` 表命名、回滚信息表 | 完全兼容：**`IP_PORT_schema` + `$_$Inception_backup_information$_$` + 回滚语句表**，保留 `INSERT/DELETE/UPDATE/CREATEDB` 老字符串，Archery 回滚工具可直接识别 |
| 回滚生成 | 文本拼装 | **类型感知回滚**（`rollback/generator.go`）：INSERT↔DELETE、UPDATE 前后镜像、NULL/decimal/binary/JSON/时间类型字面量精确处理、`keyWhere()` 仅在可靠主键或全非空唯一键时生成 |
| 无损边界 | 静默跳过危险操作 | 显式 fail-closed：DROP DATABASE / TRUNCATE / DROP INDEX 等无法无损回滚的语句**直接报错拦截** |

---

## 5. 协议网关与可观测性（新增功能）

- **MySQL 协议网关**（`internal/protocol`）：对外暴露 12 列结果集（与原版协议一致，Archery / PyMySQL 可直接对接），支持 `mysql_native_password`，保留管理命令面：`SHOW DATABASES/TABLES/WARNINGS/GRANTS`、`SELECT CONNECTION_ID()`、`inception show/get variables|levels|processlist`、`inception set`、`KILL QUERY|CONNECTION`。
- **可观测性端点**（**新增**）：`/healthz`、`/readyz`、`/metrics`（Prometheus `text/plain; version=0.0.4`），原生适配 K8s 探针与监控。
- **请求信封解析**（`internal/request/legacy.go`）：`ParseLegacyEnvelope` 含 `MaxRequestBytes=16MB`、`MaxStatementBytes=1MB`、`MaxStatements=10000` 与指令校验，防止超大/畸形请求。

---

## 6. 工程化与交付（新增）

- **容器化交付**（新增）：`Dockerfile` 多阶段构建（golang:1.22 → distroless），`CGO_ENABLED=0` 静态二进制，暴露 4000（协议）/ 4001（观测）。
- **配置模板化**：`config.minimal.toml` / `config.toml.default` / `config.toml.gip`（76 个稳定 GIP RuleID 全量配置）/ `config.integration.toml` 等，老 `[inc_level]` 与新 `[rules]` 可共存且 `[rules]` 优先。
- **测试与回归**：
  - 单元测试全绿（`.\.tools\go\bin\go.exe test ./internal/...`，15 个包）。
  - `testcases/manual-regression/`：PHASE_1_7_CASES（102 例，54 自动化 + 其余标注 MANUAL，未伪造 PASS）、RULE_CASES（86 规则）、MYSQL_57_80_REGRESSION。
  - 运行脚本：`run-phase1-7.ps1` / `run-rules.ps1` / `run-all.ps1`。
- **文档体系**（新增）：`docs/` 下 ROADMAP、RULE_CATALOG、RULE_STANDARD、PROJECT_MAP、各期业务追踪、MySQL 5.7/8.0 兼容矩阵、GIP 配置指南等 10 份。

---

## 7. 主动放弃的能力（设计决策，非缺陷）

| 能力 | 状态 | 说明 |
|------|------|------|
| OSC（Online Schema Change） | 明确放弃 | 业务选择不使用，管理命令显式拒绝 OSC 指令 |
| 内嵌 gh-ost 无锁 DDL | 明确放弃 | 不再内嵌第三方无锁变更工具；DDL 走原生执行 + 备份回滚 |
| TLS 双向认证接入 | 当前未实现 | 协议层未强制 mTLS，依赖网络层隔离 |

> 若业务仍需无锁 DDL，建议在外围（Archery / 独立 gh-ost 任务）完成，goInception Plus 负责审计与回滚保障。

---

## 8. 一句话总结

goInception Plus 用 **更新的 TiDB 8.5.3 Parser + 干净内核 + 稳定规则目录 + 可观测性 + 容器化**，在审计精度、执行安全（fail-closed）、备份回滚可靠性（类型感知 + 行数校验）、运维可观测性上全面优于原版，并补齐了原版隐式转换/时间值/超大请求等漏洞；**在 "审计/执行/备份/回滚" 闭环上可平替原版**，代价是放弃 OSC/gh-ost 无锁 DDL 这一被明确排除的能力。
