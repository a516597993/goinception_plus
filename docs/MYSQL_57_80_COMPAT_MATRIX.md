# MySQL 5.7 / 8.0 审核兼容矩阵

本项目使用 TiDB 8.5 Parser 做语法解析，但以目标 MySQL 的真实版本、`sql_mode` 和元数据为审核事实。Parser 能解析不等于目标实例能够执行。

| 能力 | MySQL 5.7 | MySQL 8.0 | Plus 当前策略 |
|---|---|---|---|
| CTE (`WITH`) | 不支持 | 支持 | 5.7 返回目标版本不支持；8.0 支持已投影的 DML/SELECT |
| 窗口函数 | 不支持 | 支持 | 同上，不交给目标执行前必须完成版本校验 |
| 表达式索引 | 不支持 | 8.0.13+ | 低于 8.0.13 返回目标版本不支持；Provider 读取表达式元数据 |
| 降序索引 | 语法接受但无实际降序 | 支持 | Provider保留排序方向，规则可区分 |
| 不可见索引 | 不支持 | 支持 | Provider读取可见性；5.7不允许 |
| CHECK约束 | 解析但忽略 | 8.0.16+执行 | 按实际服务版本判断，不假定语义一致 |
| JSON | 5.7.8+ | 支持 | 稳定投影；备份/回滚使用类型感知编码 |
| 生成列 | 支持 | 支持 | 读取生成表达式；禁止作为不可靠回滚键 |
| 默认值表达式 | 基本不支持 | 8.0.13+部分支持 | 依目标版本校验 |
| `ALTER TABLE RENAME COLUMN` | 不支持 | 支持 | 5.7 返回目标版本不支持；8.0 更新字段及关联索引 Snapshot |
| `utf8mb4_0900_*` | 不支持 | 支持 | 5.7 拒绝该 collation |
| 8.0新增保留字 | 普通标识符可能合法 | 必须引用 | 使用目标版本关键字表校验（待扩充逐词表） |
| `NO_ZERO_DATE`等sql_mode | 行为依5.7模式 | 已并入严格模式语义 | 读取服务端实际`sql_mode`后审核 |

## 已实现闭环

- 目标服务只允许 MySQL 5.7/8.0；版本和 `sql_mode` 从服务端读取。
- DDL/DML 都先转换为稳定投影；无法完整投影的语句不允许静默通过。
- 请求级 Metadata/Snapshot 支持同批次结构演进。
- ALTER Snapshot 覆盖 ADD/DROP/MODIFY/CHANGE/RENAME COLUMN、ADD/DROP/RENAME INDEX 和 RENAME TABLE；字段变更同步演进索引引用。
- UPDATE、DELETE、INSERT SELECT在配置影响行数阈值时使用 `EXPLAIN`，INSERT VALUES使用精确行数。
- 自动化版本矩阵覆盖 MySQL 5.7、8.0.12、8.0.13 和 8.0.39 的 RENAME COLUMN、表达式索引及 `utf8mb4_0900_*` 准入边界。

## 真实实例验收记录

- 2026-08-19：MySQL 5.7 Docker实例（WSL2，宿主端口3307）通过服务端变量、schema、InnoDB表、JSON、生成列、索引、VIEW `SHOW CREATE`及UPDATE `EXPLAIN`影响行数集成测试。
- 2026-08-19：WSL2 / Linux amd64 / Go 1.23.6 / `CGO_ENABLED=1` / GCC环境通过 `go test -race ./...`；MySQL 5.7集成测试也在race模式单独通过。
- MySQL 8.0真实实例回归沿用此前8.0.39验收记录。后续新增版本差异规则时仍必须分别重跑5.7与8.0用例。
