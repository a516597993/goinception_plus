# goInception Plus 分期路线图

## 总体验收原则

- 每期必须保持 go test ./... 通过。
- 新规则必须进入统一规则注册表并有旧版映射。
- 新增数据库行为必须同时覆盖 MySQL 5.7 和 8.0。
- 未实现能力必须明确拒绝，禁止静默放行或降级。
- 每期完成后生成 Windows AMD64 和 Linux AMD64 产物。

## 一期：Parser/AST 与审核内核

目标：建立不依赖完整 TiDB Server 的最小审核链。

工作：

- 固定 TiDB Parser v8.5.3 本地源码。
- 解析 SQLMode、多语句和 inception 控制块。
- 建立 Statement、AuditIssue、AuditRecord 稳定模型。
- 建立 Rule、Policy、Engine 和轻量 Session。
- 拒绝未准入语句类型。

验收：

- USE、DDL、DML 能正确分类。
- 不支持的语法或语句不会被误判为通过。
- Windows/Linux 可交叉编译。

状态：已完成。

## 二期：元数据、Schema Snapshot 与 DDL 审核

目标：完成 CREATE/ALTER/DROP/RENAME/TRUNCATE 的只审核能力。

工作：

- 读取 schema、table、columns、indexes 和 SHOW CREATE。
- 接入请求级缓存，并把 Provider 注入 Session。
- 实现 USE 对当前数据库的切换。
- 实现 CREATE/ALTER/DROP/RENAME 后的内存结构演进。
- 迁移全部旧版 DDL 配置、错误码和规则测试。
- 建立 MySQL 5.7/8.0 DDL 语法差异矩阵。

验收：

- 同一批次 CREATE TABLE 后的 ALTER/INSERT 能看到新结构。
- 同一个对象每个请求最多从目标库加载一次。
- 全部旧 DDL 规则有编码、旧键映射及正反测试。
- 真实 MySQL 5.7/8.0 集成测试通过。

状态：核心能力已完成。Provider、请求级缓存、完整 DDL 投影、9 条 DDL 安全规则及 CREATE/ALTER/DROP/RENAME/TRUNCATE/INDEX Snapshot 演进已落地；MySQL 5.7/8.0 真实回归已通过。后续按旧 goInception 规则映射表持续补充细粒度 DDL 规则。

## 三期：DML 审核与影响评估

目标：完整审核 INSERT、UPDATE、DELETE 和必要的 SELECT。

工作：

- 建立表、列、谓词、连接和子查询的稳定 AST 投影。
- 检查 WHERE、LIMIT、ORDER BY、主键/唯一键和全表操作。
- 使用 EXPLAIN 或受控 COUNT 评估影响行数。
- 迁移 SQL safe updates、最大影响行数和禁止函数等规则。
- 处理多表 UPDATE/DELETE、INSERT SELECT 和 ON DUPLICATE KEY。

验收：

- 无 WHERE 更新、删除按配置准确拒绝。
- 影响行数超过阈值准确报告。
- 规则不直接执行用户 DML。
- 全部旧 DML 规则完成对照测试。

状态：核心能力已完成。表/字段校验、WHERE/LIMIT/ORDER BY、可靠键、JOIN ON、INSERT 字段列表、SELECT *、EXPLAIN 影响评估、阈值、多表 UPDATE/DELETE 及 CTE 版本门禁均已落地并完成真实工单回归；旧版长尾规则仍按映射表持续迁移。

## 四期：串行执行与故障状态机

目标：审核通过后安全执行直接 DDL/DML。

工作：

- 建立 ExecutionPlan、Executor 和逐语句状态。
- 执行前重新校验目标对象身份和权限。
- 处理锁等待、死锁、连接中断、超时和取消。
- 明确 DDL 隐式提交及不可事务回滚边界。
- OSC 和 gh-ost 配置统一返回不支持，不允许静默直执行。

验收：

- 任一错误级审核结果阻止执行。
- 部分执行有准确的成功/失败边界。
- 每条语句记录耗时、影响行数和数据库错误。

## 五期：备份与回滚 SQL

目标：实现与旧 goInception 兼容的数据备份和回滚。

工作：

- INSERT 记录可靠键并生成 DELETE。
- UPDATE/DELETE 在执行前锁定并读取旧行。
- 类型安全编码 NULL、binary、decimal、time 和 JSON。
- 写入兼容备份表并保存回滚 SQL。
- 保存 DDL 前置 SHOW CREATE；不可无损回滚时明确标记。

验收：

- DML 执行后再执行回滚，数据逐列恢复一致。
- 无主键或可靠唯一键时不生成不确定回滚。
- 目标执行成功但备份失败时停止后续语句。

## 六期：MySQL 协议与 Archery 兼容

目标：替换开发期 JSON 入口，对现有客户端无感切换。

工作：

- 实现握手、认证、COM_QUERY、错误包和文本结果集。
- 保持 inception_magic_start/commit/end。
- 固定旧结果列名称、顺序、类型和 NULL 行为。
- 不支持普通查询和 Prepared Statement 时返回明确错误。
- 完成 Archery 联调与协议抓包回归。

验收：

- mysql 客户端和 Archery 无需修改即可调用。
- 新旧服务结果逐字段对比通过。
- 端口、认证、超时和最大包配置可部署。

## 七期：双跑、灰度与生产化

目标：安全替换旧 goInception。

工作：

- 建立新旧审核结果 diff 工具和差异白名单。
- 双跑阶段只由旧服务执行，新服务旁路审核。
- 增加结构化日志、指标、追踪、健康检查和配置校验。
- 完成容器镜像、版本信息、升级/回退文档和容量测试。
- 分租户、分数据库逐步切流。

职责调整：双跑、diff、差异白名单和灰度切流由 Archery 实现；Plus 第七期仅交付结构化日志、指标、追踪、健康检查、配置校验、镜像、版本、回退文档和容量测试。

验收：

- 未批准审核差异为零。
- 100、1000、10000 条批量 SQL 性能达标。
- 灰度期间无不可解释的数据或协议异常。
- 具备一键回退旧服务能力。

## 八期及以后：可选增强

- gh-ost 和 pt-online-schema-change 以独立执行器插件恢复。
- HTTP/gRPC 管理 API。
- 规则热加载、租户级规则包和规则版本冻结。
- MariaDB、TiDB 目标库适配。
- 基于 binlog 的备份和回滚增强。
