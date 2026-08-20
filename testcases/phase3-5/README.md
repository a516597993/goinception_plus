# 三至五期手工与半自动测试包

## 前置环境

- 已编译 `bin/windows-amd64/goinception-plus.exe`。
- MySQL 5.7 或 8.0 可访问。
- 测试账号可读 `information_schema`，并对 `gip_phase_test` 有建库、DDL 和 DML 权限。
- `mysql.exe` 已加入 PATH；也可以用 `-MySQL` 指定完整路径。
- 本套件会创建并修改 `gip_phase_test`，不要指向包含同名重要数据库的实例。

建议先确认五期环境事实：

```sql
SELECT @@log_bin, @@binlog_format, @@binlog_row_image;
```

生产目标必须为 `1/ON`、`ROW`、`FULL`。当前版本尚未接通 replication 捕获器，即使三个值正确，P5-01 也应该以 `GIP-BAK-BL-001` 安全失败，并且目标数据不得改变。

## 一键运行

在项目根目录执行：

```powershell
.\testcases\phase3-5\run.ps1 `
  -HostName 127.0.0.1 `
  -Port 3306 `
  -User root `
  -Password 123456
```

若 `mysql.exe` 不在 PATH：

```powershell
.\testcases\phase3-5\run.ps1 `
  -User test `
  -Password test `
  -MySQL 'C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe'
```

脚本会依次：

1. 执行 `setup.sql` 重建测试数据。
2. 根据 `manifest.json` 运行全部请求并校验 StageStatus/RuleID。
3. 把完整 JSON 保存至 `results`。
4. 执行 `verify.sql` 检查整批阻断和首错停止是否真的没有写入后续数据。

重复测试时可以使用 `-SkipSetup`；只检查 JSON 时可以使用 `-SkipVerify`。测试结束后如需删除环境：

```powershell
Get-Content .\testcases\phase3-5\cleanup.sql -Raw |
  mysql --host=127.0.0.1 --port=3306 --user=test --password=test
```

## 案例清单

| 编号 | 场景 | 预期 |
|---|---|---|
| P3-01 | 合法 INSERT，只审核 | Checked，不写入数据 |
| P3-02 | UPDATE 无 WHERE | GIP-DML-SF-001 |
| P3-03 | DELETE 无 WHERE | GIP-DML-SF-001 |
| P3-04 | 目标表不存在 | GIP-DML-MD-001 |
| P3-05 | 更新字段不存在 | GIP-DML-MD-001 |
| P3-06 | 无可靠键表要求备份 | GIP-DML-SF-002，不能执行 |
| P3-07 | 无 USE、使用全限定表名 | Checked |
| P3-08 | CREATE→ALTER→INSERT 快照演进 | 三条均 Checked |
| P4-01 | 三条 DML 串行成功 | 均 ExecuteSuccessfully，最终无 id=900 |
| P4-02 | 后续语句审核失败 | 整批不执行，id=901 不存在 |
| P4-03 | 第一条运行时主键冲突 | 第一条 ExecuteFailed，第二条不执行，id=902 不存在 |
| P4-04 | CREATE→ALTER 实际执行 | phase4_ddl 存在 id/note 两列 |
| P5-01 | ROW Binlog 捕获尚未配置 | GIP-BAK-BL-001，users.id=1 的 score 不变 |

## 五期回滚生成器自动测试

类型安全 SQL 的单元测试不依赖真实 MySQL：

```powershell
.\.tools\go\bin\go.exe test .\internal\rollback -v
```

目前覆盖 INSERT→DELETE、DELETE→INSERT、UPDATE before/after image、NULL 和二进制字面量。真实 Binlog 恢复闭环需要等 replication 捕获与兼容备份表接通后再加入本套件。
