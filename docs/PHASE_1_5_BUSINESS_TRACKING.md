# goInception Plus 一至五期业务追踪

更新日期：2026-08-19

## 总体边界

项目面向 MySQL 5.7/8.0 的 SQL 审核、执行、备份和回滚。TiDB 8.5.3 仅提供 Parser/AST 及必要基础类型，不引入 TiKV、PD、TiDB Session、Planner、Executor、DDL、Domain 或 MockTiKV。

稳定原则：TiDB AST 只能存在于 `internal/parser`；规则层仅依赖内部投影。请求内按 SQL 顺序维护 Schema Snapshot。整批审核存在 Error 时不执行任何语句；执行阶段固定单物理连接、串行执行、首错停止。

## 第一期：请求与 Parser 边界

- 词法状态机解析 inception 控制块，严格验证 start/commit、directive、请求大小和语句数量。
- TiDB Parser 输出稳定 Statement、DDL/DML 投影、原文位置和 warning。
- 无法投影或目标 MySQL 不支持的语法按 Unsupported/Error 处理，不得误审核通过。

## 第二期：元数据与 DDL 闭环

- 每个请求建立目标 MySQL 连接及独立 Metadata Cache，读取真实版本、sql_mode、大小写和字符集事实。
- 支持 USE 上下文、information_schema/SHOW CREATE、表/列/索引元数据和请求内 Schema Snapshot 演进。
- CREATE、ALTER、DROP、RENAME、TRUNCATE、INDEX 等 DDL 必须有稳定投影和明确准入规则。

## 第三期：DML 审核

- INSERT、UPDATE、DELETE、SELECT 通过内部 DML 投影审核对象、字段和危险条件。
- UPDATE/DELETE 无 WHERE、目标表/字段不存在及无可靠唯一键等问题进入统一 Rule Catalog。
- 规则级别由 Policy 和旧 `[inc_level]` 语义控制。

## 第四期：执行链

- 审核全部通过后才生成执行链；同批 SQL 使用单一连接顺序执行。
- 记录单条耗时、影响行数和错误；运行时首错停止后续 SQL。
- DDL 遵循 MySQL 实际隐式提交边界，不伪造跨 DDL 事务回滚。

## 第五期：ROW Binlog 备份与回滚

### 已确定方案

只采用旧 goInception 的 ROW Binlog 模式，不使用执行前数据快照。执行备份前必须验证：

```text
log_bin = ON
binlog_format = ROW
binlog_row_image = FULL
```

每条待备份语句在固定执行连接上记录起止 Binlog Position 和 `CONNECTION_ID()`；读取区间内 event，并以事务 Query event 的 thread id 过滤本连接 WRITE/UPDATE/DELETE ROW event。事务 XID 后清除连接归属，捕获行数必须与 affected_rows 一致，否则失败关闭。

### 回滚生成

- INSERT：使用可靠主键/非空唯一键生成 DELETE。
- DELETE：使用 before image 生成 INSERT。
- UPDATE：用 before image 恢复所有列，以 after image 的可靠键定位。
- NULL、decimal、时间、JSON、转义和 binary/blob 使用类型感知字面量；任意二进制统一写为十六进制字面量。
- 无可靠键、FULL image 不完整或列数不匹配时拒绝生成不确定回滚。

### 兼容备份库

- 库名保持 `host_port_schema`。
- 信息表保持 `$_$Inception_backup_information$_$`。
- 回滚表保持原表名及 `id/rollback_statement/opid_time` 结构。
- 信息表 `type` 保持旧值：INSERT、UPDATE、DELETE、CREATEDB、CREATETABLE、ALTERTABLE、DROPTABLE、RENAMETABLE、CREATEINDEX、DROPINDEX。

### 失败语义

目标 SQL 执行成功但 Binlog 捕获、回滚生成或备份落库失败时，结果为 `Stage=BACKUP`、`BackupFailed`、Error，并停止后续语句，绝不报告完整成功。

DDL 只为能够确定无损反向操作的类型生成回滚。TRUNCATE、DROP DATABASE、缺少完整定义的 DROP INDEX 和破坏性 ALTER 明确拒绝，不以“看似可回滚”的 SQL 掩盖数据损失。

## 当前验收状态

- Go 单元测试、vet 和 Windows 构建通过。
- MySQL 8.0.39 已通过 INSERT/UPDATE/DELETE、可逆 DDL、备份表、逆序回滚和逐列恢复。
- 已验证同表并发写入不会混入审计连接 OPID。
- MySQL 5.7 Docker 独立实例回归已通过；覆盖服务端变量、元数据、JSON、生成列、索引、VIEW 及 UPDATE EXPLAIN。
- WSL2 Linux amd64、Go 1.23.6、CGO/GCC 环境已通过 `go test -race ./...`，MySQL 5.7 集成测试也已在 race 模式通过。
- Archery 真实 API 联调已完成审核、审批、执行、任务完成状态、12 列结果解析、ROW Binlog 备份及回滚 SQL 读取；工单 379–389 为验收样本。
