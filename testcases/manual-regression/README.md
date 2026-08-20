# goInception Plus 1–7期手工回归测试手册

本目录用于发布前人工复测。测试分为四部分：

1. [环境、操作和结果判定](MANUAL_TEST_GUIDE.md)
2. [1–7期业务功能用例](PHASE_1_7_CASES.md)
3. [全部GIP审核规则用例](RULE_CASES.md)
4. [MySQL 5.7/8.0总回归矩阵](MYSQL_57_80_REGRESSION.md)

配套数据库脚本：

- `setup.sql`：创建隔离测试库、基础表和测试数据。
- `verify.sql`：检查关键数据及ROW Binlog前置条件。
- `cleanup.sql`：删除本套测试创建的数据库。

## 建议执行顺序

1. 阅读操作手册，备份配置并确认测试实例不是生产实例。
2. 分别在MySQL 5.7（默认示例端口3307）和8.0（默认示例端口3306）执行`setup.sql`。
3. 先跑1–7期业务功能，再跑全部规则矩阵。
4. 对5.7和8.0分别完成版本回归矩阵。
5. 每个用例记录版本、配置摘要、12列结果、目标库数据和备份库数据。
6. 执行`cleanup.sql`清理环境。

## 一键测试脚本

依赖Python 3和PyMySQL：

```powershell
python -m pip install -r .\testcases\manual-regression\requirements.txt
```

只跑1–7期业务用例：

```powershell
.\testcases\manual-regression\run-phase1-7.ps1 `
  -TargetPort 3307 `
  -TargetPassword 654321 `
  -GatewayPassword 123456 `
  -ConfirmDestructive
```

只跑76条审核规则：

```powershell
.\testcases\manual-regression\run-rules.ps1 `
  -TargetPort 3307 `
  -TargetPassword 654321 `
  -GatewayPassword 123456 `
  -ConfirmDestructive
```

MySQL 5.7和8.0全部一键回归：

```powershell
.\testcases\manual-regression\run-all.ps1 `
  -MySQL57Password 654321 `
  -MySQL80Password 123456 `
  -GatewayPassword 123456 `
  -ConfirmDestructive
```

脚本默认复制`config/config.toml.default`并在隔离的4400端口启动临时Plus进程，测试结束自动关闭，不修改或占用当前4000服务。若明确要测试已运行服务，可对单套脚本传入`-UseRunningService -GatewayPort 4000`。

`-ConfirmDestructive`是强制保护，因为初始化会删除并重建`gip_manual`。如已自行完成初始化，可传`-SkipSetup`代替。

每次运行在`results/时间-套件-mysql端口/`生成：

- `results.json`：完整列名、结果行、错误和断言。
- `summary.csv`：方便Excel筛选和登记。
- `summary.md`：PASS/FAIL/ERROR/MANUAL/SKIP汇总。

自动化不适合安全执行的断网、死锁、关闭Binlog、备份库故障、Archery页面、Docker回退和容量观测用例会记录为`MANUAL`，不会伪装成PASS。当前脚本自动执行54/102个分期用例和68/76个规则用例，其余项目按对应手册人工完成。

规则套件使用隔离服务的JSON结构化日志交叉验证`rule_codes`。固定12列协议只包含兼容错误文本，不直接暴露结构化RuleID；因此脚本只有在结果级别正确且日志中真实出现目标RuleID时才判定规则PASS。使用`-UseRunningService`时无法自动取得该服务的日志，规则结果只能完成协议级判定，推荐规则回归始终使用默认隔离服务模式。

## 总体验收标准

- Error级Issue必须阻止整批执行；Warning是否允许执行取决于`ignore-warnings`。
- 不支持或不能稳定投影的SQL必须明确失败，不能静默审核通过。
- 审核模式不得修改目标库；执行模式按语句串行，首个失败后停止后续语句。
- 备份模式必须验证`log_bin=ON`、`binlog_format=ROW`、`binlog_row_image=FULL`。
- INSERT、UPDATE、DELETE回滚后逐列恢复；NULL、二进制、JSON、小数和时间不得失真。
- MySQL协议结果固定为12列，列名及顺序与手册一致。
- MySQL 5.7不允许8.0专属语法；MySQL 8.0允许的语法仍须经过Plus稳定投影和规则审核。
