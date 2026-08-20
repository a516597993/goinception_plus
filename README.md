# goInception Plus

> 基于 **TiDB 8.5.3 Parser** 重构的 SQL 审计 / 执行 / 备份 / 回滚 引擎，面向 **MySQL 5.7 / 8.0**。
> 版本：`0.7.0-dev`（Clean-room 重构，不依赖 TiKV / PD / TiDB Session / Planner / Executor / DDL / Domain）。

goInception Plus 是 [原版 goInception](https://github.com/hanchuanchuan/goInception) 的现代重构版。它在保留原版 12 列协议兼容（可直接对接 **Archery / PyMySQL**）的前提下，用更新的解析内核、稳定的规则目录、fail-closed 安全策略与可观测性，提升审计精度与执行可靠性。

**设计边界（重要）**：本项目**主动放弃 OSC 与内嵌 gh-ost 无锁 DDL**。核心闭环（审计 / 执行 / 备份 / 回滚）可平替原版。

---

## 1. 功能概览

- **SQL 审计**：86 条规则（`GIP-XXX` 稳定编号），覆盖表 / 列 / 索引 / 字符集 / 主键 / 自增 / 外键 / 分区 / DML 安全 / 隐式转换 / 时间值等；兼容原版 `er_*` key。
- **SQL 执行**：请求级单物理连接 + 审计闸门，支持 MySQL 5.7 / 8.0，fail-closed 拒绝 MariaDB / TiDB 作为目标。
- **备份 / 回滚**：ROW Binlog 捕获（按 `CONNECTION_ID` 过滤、行数校验 fail-closed），类型感知反向 SQL 生成；备份到**可配置的任意 MySQL 实例（本地或远程）**，存储契约与原版一致（Archery 回滚工具可直接识别）。
- **协议网关**：MySQL 协议 12 列结果集 + `mysql_native_password` + 管理命令面（`inception show/set`、`SHOW ...`、`KILL`）。
- **可观测性**：`/healthz`、`/readyz`、`/metrics`（Prometheus）。

---

## 2. 项目结构

```
cmd/goinception-plus/        入口：协议服务 / audit 子命令
config/                      配置模板（minimal / default / gip / integration）
docs/                        设计文档、规则目录、路线图、回归追踪
internal/
  parser/       TiDB 8.5.3 解析适配（AST 不越过此边界）
  request/      原版 inception 指令信封解析（含大小/条数限制）
  model/        稳定业务模型（DDL/DML/Column/Index Spec）
  audit/        规则契约与编排（catalog.go 规则目录）
  audit/rules/  具体规则实现（legacy_ddl / dml / core ...）
  policy.go     三级策略回退（RuleLevels → 默认 → Warning）
  session/      轻量请求状态机（解析→元数据→快照→规则→执行）
  execution/    执行引擎接口
  execution/mysql/  MySQL 执行实现（含备份触发、DDL 回滚）
  backup/       ROW Binlog 捕获 + 兼容存储
  backup/binlogmysql/   binlog 捕获与校验
  backup/mysqlstore/    备份库写入（IP_PORT_schema 契约）
  rollback/     类型感知反向 SQL 生成
  protocol/     MySQL 协议网关 + 管理命令
  observability/ 健康/就绪/指标端点
scripts/        编译与启动脚本（Linux / Windows）
```

---

## 3. 快速开始

### 3.1 前置要求

- **Go 1.22+**（TiDB 8.5.3 Parser 模块要求）。Windows 下若 Go 不在 PATH，本项目使用 `.tools\go\bin\go.exe`。
- 目标 MySQL 实例（执行对象）：**MySQL 5.7 或 8.0**，`log_bin=ON`、`binlog_format=ROW`、`binlog_row_image=FULL`（开启 `backup=1` 时必需）。
- 备份 MySQL 实例（可选，开启 `backup=1` 时必需）：任意 MySQL，与目标库可同机或独立远端。

### 3.2 编译

```bash
# Linux
./scripts/bianyi-linux.ps1        # 或 scripts/start-linux.sh 内含构建

# Windows
.\scripts\bianyi-windows.ps1
```

产出二进制：`goinception-plus`（容器镜像见 `Dockerfile`）。

### 3.3 配置

复制并编辑模板（首次务必修改 `auth.password` 与 `backup.password`）：

```bash
cp config/config.toml.default config/config.toml
# 或用稳定 GIP 规则全量模板：
# cp config/config.toml.gip config/config.toml
```

关键配置块：

```toml
[server]
host = "0.0.0.0"
port = 4000            # 协议网关端口（Archery 对接此端口）

[auth]
password = "change-me" # 网关接入密码（明文，限制文件权限，勿提交真实值）

[backup]
host = "127.0.0.1"     # 备份库地址，可填远端独立 MySQL
port = 3306
user = "backup"
password = "change-me"

[inc]
backup = 1             # 默认是否备份（请求内 inception set backup=1 可覆盖）
```

> 老 `[inc_level]` 与新 `[rules]` 可共存，`[rules]` 优先。模板说明见 `docs/GIP_RULE_CONFIG_GUIDE.md`。

### 3.4 启动

```bash
# Windows（默认用 config/config.toml.default）
.\scripts\start-windows.ps1

# Linux
./scripts/start-linux.sh

# 指定配置
./goinception-plus -config config/config.toml

# 容器
docker build -t goinception-plus .
docker run -p 4000:4000 -p 4001:4001 -v $PWD/config:/etc/goinception-plus goinception-plus
```

启动后：
- 协议服务监听 `:4000`（Archery 对接）。
- 观测端点监听 `:4001`：`http://localhost:4001/healthz`、`/readyz`、`/metrics`。

### 3.5 提交一条审计 + 执行请求（legacy 信封）

```sql
/*--user=root;--password=xxx;--host=127.0.0.1;--port=3306;--execute=1;--backup=1;--password=xxx;--ignore-warnings=0;--sleep=0;*/
inception_magic_start;
use test;
insert into t1(id,name) values(1,'a');
inception_magic_commit;
```

通过 MySQL 客户端连接 `:4000` 执行上述文本即可（与原版协议一致）。

### 3.6 仅做审计（CLI 子命令）

```bash
go run ./cmd/goinception-plus audit < request.sql
# 或单独开启若干 DDL 规则：
go run ./cmd/goinception-plus audit -check-primary-key -check-table-comment -max-column-count 50 < request.sql
```

审计子命令从标准输入读取一条 legacy 请求，输出协议无关的 JSON。

---

## 4. 运维与可观测性

- **健康检查**：`GET /healthz`、`GET /readyz`（K8s liveness/readiness 探针）。
- **指标**：`GET /metrics`（Prometheus `text/plain; version=0.0.4`）。
- **管理命令**（协议网关内）：
  - `SHOW DATABASES` / `SHOW TABLES` / `SHOW WARNINGS` / `SHOW GRANTS`
  - `SELECT CONNECTION_ID()`
  - `inception show variables|levels|processlist`、`inception get variables|levels`
  - `inception set ...`（含 `backup=1` 等运行时开关）
  - `KILL QUERY|CONNECTION`
- 协议网关**不代理**普通查询到目标 MySQL，仅提供上述本地管理面与审计/执行接入。

---

## 5. 备份与回滚边界（务必阅读）

- 开启 `backup=1` 需要：**备份库已配置** + 目标库 `log_bin=ON` / `binlog_format=ROW` / `binlog_row_image=FULL`。前置不满足时**显式报错**（fail-closed）。
- **本地备份库与远程备份库均支持**——备份目标由 `[backup]` 的 host/port 决定，可指向同机或独立远端 MySQL。
- 以下操作**无法无损回滚**，引擎会直接拦截报错：
  - `DROP DATABASE` / `TRUNCATE TABLE` / `DROP INDEX`（含 `ALTER ... DROP INDEX`）
- 备份库表命名遵循 `IP_PORT_schema`，回滚信息表为 `$_$Inception_backup_information$_$`，与原版兼容，Archery 回滚可直接使用。

---

## 6. 测试与回归

```bash
# 单元测试（Windows 示例，Go 不在 PATH 时使用 .tools 内版本）
.\.tools\go\bin\go.exe test ./internal/...

# 业务回归（PowerShell）
.\testcases\manual-regression\run-phase1-7.ps1
.\testcases\manual-regression\run-rules.ps1
.\testcases\manual-regression\run-all.ps1
```

回归覆盖：PHASE_1_7（102 例，54 自动化 + 其余标注 MANUAL，未伪造 PASS）、RULE_CASES（86 规则）、MySQL 5.7/8.0 兼容矩阵。详见 `docs/` 与 `testcases/manual-regression/`。

---

## 7. 设计文档索引

- `docs/ROADMAP.md`：阶段 1–8 路线图与验收门槛。
- `docs/RULE_CATALOG.md`：86 条规则清单与老版映射。
- `docs/RULE_STANDARD.md`：规则 ID / 布局 / 测试标准。
- `docs/PROJECT_MAP.md`：产品边界、模块图、数据流。
- `docs/PLUS_VS_ORIGINAL.md`：与原版 goInception 的优化 / 升级 / 新增功能对比。
- `docs/GIP_RULE_CONFIG_GUIDE.md`：稳定 GIP RuleID 与开关中文指南。
- `docs/MYSQL_57_80_COMPAT_MATRIX.md`：5.7 / 8.0 兼容矩阵。
- `docs/PHASE_1_5_BUSINESS_TRACKING.md`、`PHASE_6_BUSINESS_TRACKING.md`、`PHASE_7_PRODUCTION.md`：各期业务追踪与生产部署。

---

## 8. 与原版的关系

goInception Plus 在 "审计 / 执行 / 备份 / 回滚" 闭环上**可平替**原版 goInception，并修复其若干漏洞（隐式转换误判、时间值静默放行、超大/畸形请求无限制等）。**主动放弃 OSC 与内嵌 gh-ost 无锁 DDL**——如业务需要无锁变更，建议在外围（Archery / 独立 gh-ost 任务）完成，由本引擎负责审计与回滚保障。

---

## 9. 许可证与致谢

本项目为 Clean-room 重构，解析内核取自 [TiDB](https://github.com/pingcap/tidb) v8.5.3（Apache-2.0）。审计规则语义参考原版 goInception 的 `inc_level` 配置约定。
