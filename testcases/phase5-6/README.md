# 第五、六期真实 MySQL 联调

要求本机 MySQL 8.0/5.7 已开启 `log_bin=ON`、`binlog_format=ROW`、`binlog_row_image=FULL`，测试账号具备目标库 DDL/DML、备份库建表及 `REPLICATION CLIENT/SLAVE` 权限。

```powershell
.\testcases\phase5-6\run.ps1 -TargetPassword 123456 -GatewayPassword archerypass
```

测试会重建 `gip_p56` 和 `127_0_0_1_3306_gip_p56` 两个测试库，验证：

- 4000 端口 mysql_native_password 认证；
- VERSION、SHOW VARIABLES 和固定 12 列结果；
- INSERT/UPDATE/DELETE 的 connection_id + Binlog position 过滤；
- NULL/decimal/binary/JSON/datetime 的回滚编码；
- 兼容备份信息表和按表回滚记录；
- 逆序执行回滚后逐列恢复初始数据。
