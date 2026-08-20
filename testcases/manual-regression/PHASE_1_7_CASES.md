# 1–7期业务功能手工测试用例

除特别说明外，每个用例在5.7和8.0各执行一次。所有SQL使用`MANUAL_TEST_GUIDE.md`中的控制块包装。

## 一期：请求、Parser/AST和审核内核

| ID | 场景与操作 | 预期 |
|---|---|---|
| P1-001 | 合法控制块，`USE gip_manual; SELECT id FROM users WHERE id=1;` | 每条语句有CHECKED结果，无Parser Error |
| P1-002 | 删除`inception_magic_commit` | 请求级1064错误，不审核、不执行 |
| P1-003 | 重复start或commit | 明确协议错误，不截取其中一段继续执行 |
| P1-004 | commit位于start之前 | 明确顺序错误 |
| P1-005 | SQL字符串含`'inception_magic_commit'` | 字符串不被当作控制标记 |
| P1-006 | 普通SQL注释含控制标记文本 | 注释文本不导致提前截断 |
| P1-007 | 未知directive `--not-exists=1` | 1064 unknown directive |
| P1-008 | 重复host、非法port、非法布尔值 | 分别返回明确校验错误，不采用默认值 |
| P1-009 | `backup=1,execute=0` | 参数组合错误，不连接备份执行链 |
| P1-010 | 多语句含分号、引号、反斜杠及UTF-8注释 | 原文正确切分，语句数正确 |
| P1-011 | 语法错误`UPDTE users SET age=1;` | `GIP-COR-PS-002`，包含出错语句/位置 |
| P1-012 | 未准入语句，例如Prepared Statement语法 | `GIP-COR-ST-001`或明确不支持，不得通过 |
| P1-013 | 超过配置的请求大小、SQL条数、单SQL长度 | 在资源耗尽前拒绝 |
| P1-014 | 同一请求含`USE gip_manual; USE missing_db;` | 第二个USE失败，后续不得错误沿用不存在库 |

建议SQL：

```sql
SELECT 'inception_magic_commit; still text' AS marker FROM users WHERE id=1;
UPDATE users SET username='a; b\\c' WHERE id=1;
```

## 二期：连接、元数据、Snapshot与DDL

| ID | 场景与操作 | 预期 |
|---|---|---|
| P2-001 | 正确目标连接；审核`SELECT VERSION()`以外的业务SQL | 读取真实version/sql_mode/lower_case_table_names |
| P2-002 | 错host、port、密码及无权限账号 | `GIP-META-CN-001`或准确权限错误 |
| P2-003 | 不写`--db`且不执行USE，审核`ALTER TABLE users...` | `GIP-META-DB-002` |
| P2-004 | `USE missing_db` | `GIP-META-DB-001` |
| P2-005 | CREATE→ALTER→INSERT同批次 | 后两条看到内存中新表/新列，不能报不存在 |
| P2-006 | RENAME TABLE→UPDATE新表名 | UPDATE看到新名；旧名在Snapshot中不可用 |
| P2-007 | DROP TABLE→后续SELECT该表 | 后续命中不存在/tombstone，不从真实库重新加载旧表 |
| P2-008 | 单条ALTER含ADD、MODIFY、CHANGE、DROP、ADD INDEX | 按spec顺序演进，后续语句看到最终结构 |
| P2-009 | CREATE TABLE LIKE users | 从源表复制结构审核，不误报为空表 |
| P2-010 | CREATE TABLE AS SELECT | 明确准入或明确不支持；不得当普通空表通过 |
| P2-011 | CREATE/DROP INDEX、RENAME INDEX | Snapshot字段与索引同步 |
| P2-012 | SHOW CREATE VIEW元数据加载 | 不发生扫描列数错误；VIEW按配置拒绝/放行 |
| P2-013 | 大小写表名，分别在lower_case_table_names实际模式测试 | 缓存键与服务端模式一致 |
| P2-014 | 读取元数据时断网/撤权后恢复 | 瞬时错误不缓存成“对象不存在” |

Snapshot示例：

```sql
CREATE TABLE phase2_t(id BIGINT PRIMARY KEY, v VARCHAR(16));
ALTER TABLE phase2_t ADD COLUMN score INT DEFAULT 0,
  CHANGE COLUMN v name VARCHAR(32),
  ADD KEY idx_name(name);
INSERT INTO phase2_t(id,name,score) VALUES(1,'n',1);
RENAME TABLE phase2_t TO phase2_t_new;
UPDATE phase2_t_new SET score=2 WHERE id=1;
```

## 三期：DML审核与影响行数

| ID | 场景与操作 | 预期 |
|---|---|---|
| P3-001 | 显式字段INSERT | 通过基础DML审核 |
| P3-002 | INSERT不写字段列表，启用对应规则 | `GIP-DML-SF-007` |
| P3-003 | INSERT VALUES行数超过`max_insert_rows` | `GIP-DML-IM-003` |
| P3-004 | UPDATE/DELETE没有WHERE | `GIP-DML-SF-001` |
| P3-005 | UPDATE/DELETE含LIMIT、ORDER BY | 分别触发SF-003/SF-004配置级别 |
| P3-006 | JOIN缺ON | `GIP-DML-SF-005` |
| P3-007 | SELECT *禁用 | `GIP-DML-SF-006` |
| P3-008 | ORDER BY RAND() | `GIP-DML-SF-008` |
| P3-009 | 数字列与字符串、字符列与数字比较 | `GIP-DML-SF-010`；NULL/同类型不误报 |
| P3-010 | 表或字段不存在 | `GIP-DML-MD-001` |
| P3-011 | max_update_rows设为1，更新预计2行 | `GIP-DML-IM-002`，affected_rows/估算可解释 |
| P3-012 | EXPLAIN权限不足或失败 | `GIP-DML-IM-001`，不能按0行放行 |
| P3-013 | explain_rule分别为first/max | 多表计划按配置选择估算值 |
| P3-014 | 多表UPDATE两张表 | 每张表、字段、JOIN条件正确投影并审核 |
| P3-015 | 多表DELETE删除两个别名 | 删除目标表集合正确，不能只看到第一张表 |
| P3-016 | INSERT SELECT和ON DUPLICATE KEY UPDATE | 明确审核字段/影响范围；不支持投影则拒绝 |
| P3-017 | CTE SELECT/UPDATE/DELETE | 5.7版本错误；8.0进入正常规则审核 |

多表DML：

```sql
UPDATE users u JOIN orders o ON o.user_id=u.id
SET u.age=u.age+1, o.status=2
WHERE u.id=1 AND o.id=1;

DELETE u,o FROM users u JOIN orders o ON o.user_id=u.id
WHERE u.id=3;
```

隐式转换：

```sql
SELECT id FROM type_matrix WHERE int_col='1';
SELECT id FROM type_matrix WHERE 1=str_col;
SELECT id FROM type_matrix WHERE time_col=1;
SELECT id FROM type_matrix WHERE json_col=TRUE;
SELECT id FROM type_matrix WHERE int_col=NULL;
```

## 四期：串行执行与失败状态机

| ID | 场景与操作 | 预期 |
|---|---|---|
| P4-001 | execute=1，三条合法DML | 按顺序执行，各自记录耗时/影响行数 |
| P4-002 | 第二条为Error级审核问题 | 整批一条也不执行 |
| P4-003 | 第一条执行时主键冲突，第二条合法INSERT | 第一条EXE-RN-001，第二条不执行 |
| P4-004 | UPDATE后DDL失败 | 明确部分执行边界，不声称整批事务回滚 |
| P4-005 | 设置短lock_wait_timeout并制造锁等待 | 超时错误、停止后续语句、连接可恢复 |
| P4-006 | 两连接制造死锁 | MySQL死锁错误准确记录，后续停止 |
| P4-007 | 执行中`KILL QUERY connection_id` | 当前任务取消，协议连接保留 |
| P4-008 | 执行中`KILL CONNECTION connection_id` | 当前任务取消且连接关闭，processlist清理 |
| P4-009 | audit_only=true收到execute=1 | `GIP-COR-SF-001`且目标数据不变 |
| P4-010 | OSC/gh-ost配置请求 | 明确不支持，不静默直接DDL |

## 五期：ROW Binlog备份与回滚

所有用例使用`execute=1;backup=1;ignore-warnings=1`，执行前保存逐列快照。

| ID | 场景与操作 | 预期 |
|---|---|---|
| P5-001 | log_bin/ROW/FULL任一不满足 | `GIP-BAK-BL-001`且目标数据不变 |
| P5-002 | INSERT一行 | 业务同名备份表生成DELETE回滚 |
| P5-003 | UPDATE一行 | 生成按可靠键恢复旧值的UPDATE |
| P5-004 | DELETE一行 | 生成完整列INSERT回滚 |
| P5-005 | 一条UPDATE影响多行 | 每个行事件有回滚记录，逆序执行可恢复 |
| P5-006 | no_key表UPDATE/DELETE | `GIP-DML-SF-002`，不生成不确定回滚 |
| P5-007 | NULL/VARBINARY/DECIMAL/JSON/DATETIME(6)更新 | 回滚字面量类型安全，逐列完全恢复 |
| P5-008 | JSON含引号、反斜杠、数组、Unicode | 回滚SQL合法，JSON语义一致 |
| P5-009 | 多表UPDATE两张表 | users和orders两个备份表均产生回滚记录 |
| P5-010 | 多表DELETE两个目标表 | 两表均有INSERT回滚，逆序恢复引用数据 |
| P5-011 | 同期其他连接写相同表 | 只捕获被审核连接的事务行事件 |
| P5-012 | 目标执行成功但备份库写入失败 | `GIP-BAK-RN-001`严重失败并停止后续语句 |
| P5-013 | CREATE TABLE、ALTER TABLE | 保存兼容备份信息及可行反向DDL |
| P5-014 | DROP/TRUNCATE等不能无损回滚DDL | 明确标记不可无损回滚，不伪造成功 |
| P5-015 | 服务捕获期间中断/重启 | 不报告完整成功；残留状态可定位 |

JSON专项：

```sql
UPDATE users
SET profile=JSON_OBJECT('text','a"b\\c','list',JSON_ARRAY(1,NULL,'中文')),
    payload=X'00FF10', amount=999999.99,
    created_at='2026-08-20 12:34:56.123456'
WHERE id=1;
```

## 六期：MySQL协议、Archery和管理命令

| ID | 场景与操作 | 预期 |
|---|---|---|
| P6-001 | 正确账号密码登录4000 | mysql_native_password认证成功 |
| P6-002 | 错账号/密码 | 1045，不能进入命令处理 |
| P6-003 | `SET NAMES utf8mb4`、`SET autocommit=1` | Archery初始化命令兼容 |
| P6-004 | `SELECT VERSION()` | 返回含goInception-Plus的版本 |
| P6-005 | `SHOW DATABASES`及LIKE | 仅information_schema，LIKE正确过滤 |
| P6-006 | SHOW TABLES/TABLE STATUS | 兼容列结构空结果 |
| P6-007 | SHOW COLUMNS不存在对象 | 标准table-not-found错误 |
| P6-008 | SHOW WARNINGS | 返回当前连接最近命令Warning |
| P6-009 | SHOW GRANTS | 只读兼容描述，不暗示目标库授权 |
| P6-010 | SELECT CONNECTION_ID() | 返回稳定连接ID，可用于KILL |
| P6-011 | 普通`SELECT 1` | 明确不支持；不是目标MySQL代理 |
| P6-012 | Prepared Statement/COM_STMT_PREPARE | 明确不支持 |
| P6-013 | 提交审核请求 | 恰好12列，列名、顺序、类型、NULL行为一致 |
| P6-014 | inception show/get variables/levels | 列名、过滤和密码掩码兼容 |
| P6-015 | inception set global/session变量及level | 后续请求生效，运行中请求不变，重启恢复TOML |
| P6-016 | inception processlist | 10列，目标directive、状态、耗时和Info正确 |
| P6-017 | KILL不存在ID | 兼容错误，不影响其他连接 |
| P6-018 | 第二个进程抢占4000端口 | 启动明确失败，不能打印listening |
| P6-019 | Archery审核页 | Error/Warning、SQL文本、12列解析正常 |
| P6-020 | Archery执行及回滚查询 | 状态、备份库和回滚SQL可查询 |

管理命令建议逐条执行：

```sql
SHOW DATABASES;
SHOW DATABASES LIKE 'information%';
SHOW TABLES FROM information_schema;
SHOW TABLE STATUS FROM information_schema;
SHOW WARNINGS;
SHOW GRANTS;
SELECT CONNECTION_ID();
inception show variables like 'check_dml_where';
inception show levels where value=2;
inception get full processlist;
```

## 七期：生产化与可观测性

Plus第七期不负责双跑、diff和灰度切流；这些由Archery实现。这里只验证Plus生产化交付。

| ID | 场景与操作 | 预期 |
|---|---|---|
| P7-001 | 启用观测端口，访问健康检查 | 存活/就绪状态与4000监听和依赖状态一致 |
| P7-002 | 访问指标端点并执行一批请求 | 连接数、请求数、耗时、错误等指标变化 |
| P7-003 | log.format=json后执行成功/失败请求 | 每行合法JSON，含时间、级别、请求/连接上下文；密码脱敏 |
| P7-004 | 启动时提供未知配置、非法RuleID或非法level | 配置校验失败，不启动服务 |
| P7-005 | SIGTERM/正常停止 | 停止接收新连接，在shutdown_timeout内清理任务 |
| P7-006 | Docker镜像启动并挂载配置 | 4000/观测端口可用，版本信息可追溯 |
| P7-007 | 100条批量审核 | 成功，记录耗时、内存峰值、元数据查询量 |
| P7-008 | 1000条批量审核 | 不超时/崩溃，结果数和输入一致 |
| P7-009 | 10000条或达到配置上限 | 在允许范围完成，超限时明确拒绝而非OOM |
| P7-010 | max_connections并发连接边界 | 上限内可用，超限明确拒绝，连接关闭后恢复 |
| P7-011 | 模拟目标MySQL慢响应 | audit_timeout生效，任务/连接注册表最终清理 |
| P7-012 | 新版本替换失败后回退旧二进制/配置 | 能恢复4000服务；测试库无不明执行 |

容量测试使用已有脚本：

```powershell
.\testcases\phase7\run-capacity.ps1
```

每个规模至少记录：SQL条数、请求字节数、总耗时、P95/P99（多次运行时）、进程峰值内存、CPU、错误数、结果行数和日志大小。

