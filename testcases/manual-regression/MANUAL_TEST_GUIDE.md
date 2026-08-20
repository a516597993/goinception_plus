# 手工复测操作与判定手册

## 1. 测试范围与安全边界

只允许在专用MySQL实例执行。配套脚本会删除并重建`gip_manual`，备份测试还会创建：

- MySQL 8.0：`127_0_0_1_3306_gip_manual`
- MySQL 5.7：`127_0_0_1_3307_gip_manual`

执行前必须确认这些库不含业务数据。不要把请求中的host、port、db替换成生产地址。

## 2. 环境清单

| 项目 | MySQL 5.7 | MySQL 8.0 |
|---|---|---|
| 示例地址 | 127.0.0.1:3307 | 127.0.0.1:3306 |
| 示例账号 | root | root |
| 示例密码 | 654321 | 按本机配置填写 |
| 测试库 | gip_manual | gip_manual |
| Plus协议端口 | 4000 | 4000，两个版本应分时测试 |
| Plus登录账号 | `[auth].username` | `[auth].username` |
| Plus登录密码 | `[auth].password` | `[auth].password` |

目标MySQL账号至少需要测试库DDL/DML、`information_schema`读取以及备份所需的复制权限。备份库账号需要建库、建表和写入权限。

## 3. 启动前检查

### 3.1 初始化目标库

MySQL 8.0：

```powershell
mysql -h127.0.0.1 -P3306 -uroot -p < .\testcases\manual-regression\setup.sql
```

MySQL 5.7：

```powershell
mysql -h127.0.0.1 -P3307 -uroot -p < .\testcases\manual-regression\setup.sql
```

### 3.2 检查配置

复制配置后再修改，不要在测试中反复覆盖生产联调模板：

```powershell
Copy-Item .\config\config.toml.default .\config\config.manual.toml
```

至少核对：

- `[server].port=4000`、`audit_only=false`。
- `[auth]`为连接4000端口的账号密码。
- `[backup]`指向用于保存兼容备份表的MySQL实例。
- `[inc]`开关和阈值与用例前置条件一致。
- `[inc_level]`或`[rules]`中目标规则级别为0、1或2。

启动：

```powershell
.\bin\windows-amd64\goinception-plus.exe -config .\config\config.manual.toml
```

看到监听成功日志后检查：

```powershell
Get-NetTCPConnection -LocalPort 4000 -State Listen |
  Select-Object LocalAddress,LocalPort,OwningProcess
```

### 3.3 检查ROW Binlog

在目标MySQL执行：

```sql
SELECT @@log_bin, @@binlog_format, @@binlog_row_image;
```

备份用例要求分别为`1/ON`、`ROW`、`FULL`。任一不满足时，正确行为是返回`GIP-BAK-BL-001`且不执行目标DML。

## 4. 提交请求

### 4.1 请求模板

把`TARGET_PORT`和密码替换为当前测试实例。审核模式：

```sql
/*--host=127.0.0.1;--port=TARGET_PORT;--user=root;--password=TARGET_PASSWORD;--db=gip_manual;--check=1;--execute=0;--backup=0;*/
inception_magic_start;
UPDATE users SET age=21 WHERE id=1;
inception_magic_commit;
```

执行但不备份：`--execute=1;--backup=0;`。

执行并备份：`--execute=1;--backup=1;`。若允许Warning继续执行，加`--ignore-warnings=1;`。

### 4.2 使用MySQL CLI连接4000端口

```powershell
mysql -h127.0.0.1 -P4000 -uarchery -p --default-character-set=utf8mb4
```

在客户端粘贴完整控制块。也可保存为`request.sql`后执行：

```powershell
Get-Content .\request.sql -Raw |
  mysql -h127.0.0.1 -P4000 -uarchery -p --default-character-set=utf8mb4
```

注意：目标MySQL密码位于控制块；4000端口密码来自`[auth]`，两者不是同一个概念。

## 5. 12列结果判定

结果列顺序必须固定：

| 序号 | 列名 | 判定要点 |
|---:|---|---|
| 1 | order_id | 请求内语句序号，从1开始 |
| 2 | stage | CHECKED、EXECUTED或BACKUP |
| 3 | error_level | 0通过、1 Warning、2 Error |
| 4 | stage_status | 审核、执行、备份状态的旧版兼容文本 |
| 5 | error_message | 无问题时NULL；有Issue时含RuleID/消息 |
| 6 | sql | 当前语句原文 |
| 7 | affected_rows | 执行或影响评估结果 |
| 8 | sequence | 操作ID；查询备份表回滚SQL时使用 |
| 9 | backup_dbname | 备份库名；无备份时NULL |
| 10 | execute_time | 执行耗时秒数字符串 |
| 11 | sqlsha1 | 兼容摘要；无值时NULL |
| 12 | backup_time | 备份耗时秒数字符串 |

判定原则：

- `error_level=2`：整批不得执行。
- `error_level=1`且未设置`ignore-warnings=1`：按兼容策略阻止执行。
- `stage=BACKUP`：还必须到目标库和备份库验证，不能只看结果行。
- Parser错误应落到对应语句结果中，不能只断开连接或返回空结果。

## 6. 规则级别与开关测试方法

每条可配置规则至少测四次：

1. 触发SQL＋规则级别2：期待Error。
2. 同一SQL＋规则级别1：期待Warning。
3. 同一SQL＋规则级别0：期待没有该RuleID。
4. 通过SQL＋规则级别2：期待没有该RuleID。

可以在控制块外使用管理命令修改进程内级别；修改只影响后续请求，重启恢复TOML：

```sql
inception set level GIP-DDL-CT-001 = 2;
inception show levels like 'GIP-DDL-CT-001';
```

旧键也要抽样验证：配置`[inc_level].er_table_must_have_pk=1`后，应映射到`GIP-DDL-CT-001`。若新旧级别同时存在，新`[rules]."GIP-DDL-CT-001"`优先。

规则同时受`[inc]`准入开关控制时，必须重启服务或通过已支持的`inception set`修改变量，并以`inception show variables`确认实际值。

## 7. 备份与回滚验证

兼容备份库名：`目标IP下划线形式_端口_目标库名`。例如：

```text
127_0_0_1_3306_gip_manual
```

每个业务表应有同名备份表，另有`$_$Inception_backup_information$_$`。使用结果第8列`sequence`查回滚SQL：

```sql
SELECT id, opid_time, rollback_statement
FROM `127_0_0_1_3306_gip_manual`.`users`
WHERE opid_time='结果第8列'
ORDER BY id DESC;
```

批次回滚顺序：语句倒序；同一语句多行也按备份记录倒序。回滚后执行：

```sql
SELECT id,username,age,amount,HEX(payload),profile,created_at
FROM gip_manual.users ORDER BY id;
```

与执行前逐列比较。重点确认：

- SQL NULL没有变成字符串`'NULL'`。
- VARBINARY按十六进制恢复。
- DECIMAL精度和尾随小数位不丢失。
- JSON内容可解析，字符串引号和反斜杠未破坏。
- DATETIME(6)微秒未丢失。
- 多表UPDATE/DELETE为每个发生行事件的表产生独立备份记录。

## 8. 失败记录模板

每个失败至少保存：

```text
用例ID：
测试时间：
Plus版本/commit：
MySQL版本：
配置文件及目标规则值：
请求SQL：
完整12列结果：
目标库执行前/后数据：
备份库名、表名、opid_time及rollback_statement：
Plus日志：
预期：
实际：
是否阻断发布：是/否
```

## 9. 结束清理

完成一个版本后执行`cleanup.sql`，再测试另一个版本。不要让5.7残留对象影响8.0判定。

