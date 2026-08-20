# MySQL 5.7 / 8.0总回归测试

## 1. 目标

同一套业务、规则、执行、备份和协议用例分别连接真实MySQL 5.7和8.0。TiDB Parser能够解析不代表目标MySQL可以执行；版本门禁、服务端`sql_mode`及真实元数据是最终事实。

版本参考：MySQL 5.7.44与MySQL 8.0官方资料。MySQL 8.0官方示例确认CTE/窗口函数、JSON生成列及索引等能力；本项目仍要求先完成稳定AST投影，不能因为目标版本支持就自动放行。

## 2. 每个版本的完整执行顺序

### R-01 实例事实采集

```sql
SELECT VERSION(), @@version_comment;
SELECT @@sql_mode, @@lower_case_table_names,
       @@character_set_server, @@collation_server,
       @@explicit_defaults_for_timestamp;
SELECT @@log_bin, @@binlog_format, @@binlog_row_image;
SHOW GRANTS FOR CURRENT_USER();
```

保存输出。Plus审核结果中的版本行为不得与实例事实冲突。

### R-02 初始化

对当前实例执行`setup.sql`，然后执行`verify.sql`。确认users=3、orders=3。

### R-03 一至七期业务回归

执行`PHASE_1_7_CASES.md`全部用例。与目标版本无关的用例必须两边行为一致；差异项只允许出现在本文件批准的矩阵中。

### R-04 全规则回归

执行`RULE_CASES.md`。所有RuleID都要记录5.7结果、8.0结果，不能只在一个版本触发。

### R-05 真实执行与回滚闭环

至少完成：

1. 单行及多行INSERT/UPDATE/DELETE。
2. 多表UPDATE/DELETE且两张表均变化。
3. NULL、DECIMAL、VARBINARY、JSON、DATETIME(6)。
4. CREATE/ALTER DDL备份。
5. 查询兼容备份库和每表回滚记录。
6. 逆序执行全部回滚，逐列比较初始化快照。

### R-06 协议与Archery

在4000端口执行全部P6用例，并通过Archery真实审核、执行、回滚查询各提交一张工单。记录工单ID、审核12列、任务状态和回滚SQL页面截图/导出。

### R-07 自动化交叉验证

手工测试后运行，自动化失败不能用手工通过覆盖：

```powershell
go test ./... -count=1
go vet ./...
```

WSL私有Go 1.23.6：

```bash
cd /mnt/d/workspace/goinception_plus
private_root="$HOME/.local/share/goinception_plus"
export GOROOT="$private_root/toolchains/go1.23.6"
export PATH="$GOROOT/bin:$PATH"
export GOTOOLCHAIN=local
export CGO_ENABLED=1
export GOCACHE="$private_root/cache/go-build"
export GOMODCACHE="$private_root/cache/go-mod"
export GIP_MYSQL57_DSN="127.0.0.1:3307:root:654321"
go test -race -count=1 ./...
```

## 3. 5.7/8.0版本差异矩阵

| ID | SQL/能力 | MySQL 5.7预期 | MySQL 8.0预期 | Plus判定 |
|---|---|---|---|---|
| V-001 | `WITH c AS (...) SELECT...` | 不支持 | 支持 | 5.7 META-CN-002；8.0进入普通审核 |
| V-002 | CTE UPDATE | 不支持 | 支持 | 同上，且8.0检查WHERE/影响行数 |
| V-003 | CTE DELETE | 不支持 | 支持 | 同上 |
| V-004 | `ROW_NUMBER() OVER(...)` | 不支持 | 支持 | 5.7版本拒绝；8.0仅在稳定投影时准入 |
| V-005 | `ALTER TABLE ... RENAME COLUMN a TO b` | 不支持 | 支持 | 5.7版本拒绝；8.0更新Snapshot和索引引用 |
| V-006 | 表达式索引 | 不支持 | 8.0.13+支持 | 按实际8.0小版本门禁；元数据读取表达式 |
| V-007 | 降序索引 | 语法可能接受但无真实降序语义 | 支持 | Provider保留方向，不能误判等价 |
| V-008 | 不可见索引 | 不支持 | 支持 | 5.7拒绝；8.0读取可见性 |
| V-009 | CHECK约束 | 解析但通常忽略 | 8.0.16+执行 | 按精确服务版本，不假定语义一致 |
| V-010 | JSON字段、JSON函数 | 5.7.8+支持 | 支持 | 两边审核及ROW回滚均需通过 |
| V-011 | JSON生成列和索引 | 支持生成列方式 | 支持，能力更强 | Provider读取表达式；生成列不能当可靠回滚键 |
| V-012 | DEFAULT表达式 | 基本不支持 | 8.0.13+部分支持 | 5.7拒绝；8.0按小版本/投影门禁 |
| V-013 | `utf8mb4_0900_ai_ci` | 不支持 | 支持 | 5.7明确拒绝；8.0再受support_charset/collation规则控制 |
| V-014 | 8.0新增保留字作标识符 | 5.7可能可用 | 8.0必须引用 | 依据目标版本关键字校验 |
| V-015 | `NO_ZERO_DATE`及零日期 | 受5.7 sql_mode影响 | 严格模式语义变化 | 从服务端读取sql_mode，非法日期规则准确 |
| V-016 | 隐式默认TIMESTAMP | 与explicit_defaults设置相关 | 默认行为不同 | 读取真实变量，自动时间规则不硬编码 |
| V-017 | information_schema.STATISTICS字段 | 无8.0表达式/可见性列 | 包含更多属性 | Provider内部适配，规则模型一致 |
| V-018 | SHOW CREATE VIEW返回列 | 实际动态列结构 | 实际动态列结构 | 两边均不得固定按两列扫描 |

### 版本SQL语料

```sql
WITH c AS (SELECT id FROM users WHERE id=1)
SELECT id FROM c;

WITH c AS (SELECT id FROM users WHERE id=1)
UPDATE users SET age=age+1 WHERE id IN (SELECT id FROM c);

WITH c AS (SELECT id FROM users WHERE id=1)
DELETE FROM users WHERE id IN (SELECT id FROM c);

SELECT id, ROW_NUMBER() OVER (ORDER BY id) AS rn FROM users;

ALTER TABLE type_matrix RENAME COLUMN str_col TO text_col;

CREATE TABLE v_json(
  id BIGINT PRIMARY KEY,
  doc JSON,
  doc_id INT GENERATED ALWAYS AS
    (JSON_UNQUOTE(JSON_EXTRACT(doc,'$.id'))) STORED,
  KEY idx_doc_id(doc_id)
);
```

表达式索引请在确认实例为8.0.13+后审核：

```sql
CREATE TABLE v_expr(
  id BIGINT PRIMARY KEY,
  name VARCHAR(64),
  KEY idx_lower_name ((LOWER(name)))
);
```

## 4. 元数据回归矩阵

每个版本都要覆盖：

| ID | 对象 | 检查内容 |
|---|---|---|
| M-001 | InnoDB普通表 | engine、comment、charset、collation、row count、create options |
| M-002 | 字段 | 类型、长度、unsigned、nullable、默认值类别、comment、charset/collation |
| M-003 | 自增/时间字段 | auto_increment、DEFAULT、ON UPDATE、精度 |
| M-004 | JSON/二进制/小数字段 | 类型保持，不转为普通字符串模型 |
| M-005 | 生成列 | expression和生成方式正确 |
| M-006 | 复合主键/唯一索引 | 字段顺序、prefix length、nullable可靠性 |
| M-007 | VIEW | SHOW CREATE动态扫描成功 |
| M-008 | 分区表 | 分区属性可见并触发配置规则 |
| M-009 | 大小写对象 | 遵循lower_case_table_names |
| M-010 | 权限/网络瞬时错误 | 与“不存在”区分，恢复后能重新加载 |

## 5. DML与EXPLAIN回归矩阵

两版本分别验证：

- INSERT VALUES精确行数及max_insert_rows。
- UPDATE/DELETE单表EXPLAIN估算。
- JOIN UPDATE/DELETE在`explain_rule=first|max`下的差异。
- WHERE主键、唯一键、普通索引及无索引谓词。
- 隐式类型转换字面量枚举：signed/unsigned/float/decimal/string/binary/temporal/boolean/null。
- JOIN字段类型不一致及别名解析。
- EXPLAIN权限不足、表不存在、锁等待时不得按0行放行。

建议先在目标MySQL直接保存EXPLAIN输出，再和Plus的`affected_rows`及RuleID比较：

```sql
EXPLAIN UPDATE users SET age=age+1 WHERE id IN (1,2);
EXPLAIN DELETE u,o FROM users u JOIN orders o ON o.user_id=u.id WHERE u.id=3;
```

## 6. ROW Binlog和备份回归矩阵

| ID | 场景 | 两版本共同预期 |
|---|---|---|
| B-001 | 前置变量均正确 | 允许进入捕获，不触发BAK-BL-001 |
| B-002 | 分别关闭/修改三个变量 | 执行前失败且不改数据 |
| B-003 | INSERT/UPDATE/DELETE | 兼容库名、同名备份表、正确回滚方向 |
| B-004 | NULL/DECIMAL/BINARY/JSON/TIME | 类型感知编码，逐列恢复 |
| B-005 | 多行操作 | 每个row event可追踪，逆序恢复 |
| B-006 | 多表UPDATE/DELETE | 所有实际变化表均有备份，不只第一表 |
| B-007 | 并发噪声连接 | 只捕获审核连接所属事务 |
| B-008 | DDL | 备份信息表记录类型和可行反向DDL |
| B-009 | 备份库失败 | 严重失败、停止后续、状态不伪成功 |

## 7. 结果记录总表

复制下表，每个版本各填一份：

| 模块 | 用例范围 | 通过 | 失败 | 阻塞 | 证据位置 |
|---|---|---:|---:|---:|---|
| 环境事实 | R-01～R-02 | | | | |
| 一期 | P1-001～P1-014 | | | | |
| 二期 | P2-001～P2-014 | | | | |
| 三期 | P3-001～P3-017 | | | | |
| 四期 | P4-001～P4-010 | | | | |
| 五期 | P5-001～P5-015 | | | | |
| 六期 | P6-001～P6-020 | | | | |
| 七期 | P7-001～P7-012 | | | | |
| 全规则 | RULE_CASES全部RuleID | | | | |
| 版本差异 | V-001～V-018 | | | | |
| 元数据 | M-001～M-010 | | | | |
| Binlog备份 | B-001～B-009 | | | | |
| 自动化/race | R-07 | | | | |

## 8. 发布门槛

- 两个版本全部核心/Parser/元数据安全规则通过。
- 默认级别为2的规则不存在漏触发或误放行。
- 版本矩阵差异均符合预期，没有“5.7执行后才报语法错”的审核漏洞。
- 两版本ROW Binlog备份和逆序回滚逐列一致。
- 多表UPDATE/DELETE所有变化表均生成对应回滚。
- Archery无需修改即可解析固定12列、执行状态和回滚查询。
- `go test`、`go vet`、Linux CGO race全部通过。
- 所有失败都有负责人、原因、处理结论；未批准的Error级差异阻断发布。

