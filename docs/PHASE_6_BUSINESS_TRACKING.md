# 第六期：MySQL 协议与 Archery 兼容追踪

更新日期：2026-08-19

## 业务目标

第六期把一至五期的新内核包装成 Archery 可调用的 goInception 服务。服务默认监听 `0.0.0.0:4000`，通过 MySQL 协议接收旧式 `inception_magic_start/commit` 请求；它不是通用 MySQL 查询服务器。

## 已确定的业务决策

1. 默认启动方式为 `goinception-plus.exe -config config/config.toml`；原 stdin JSON 方式保留为 `audit` 子命令。
2. 4000 端口使用独立的配置账号和 `mysql_native_password` 认证。该账号只保护审核网关，与 directive 中的目标 MySQL 账号无关。
3. 规则配置沿用 `[inc]` 和 `[inc_level]` 旧键；内部映射到稳定 GIP RuleID。未知键和尚未迁移的旧键必须导致启动失败，不能静默忽略。
4. 支持 Archery 初始化所需的 COM_QUERY、COM_PING、COM_QUIT、COM_INIT_DB、SET NAMES、SET autocommit、USE、VERSION 和 SHOW VARIABLES。
5. 普通查询、Prepared Statement、COM_FIELD_LIST 及其他未声明命令返回明确的 MySQL“不支持”错误。
6. 返回结果固定为旧版 12 列，保持列名、顺序、主要 MySQL 类型、NULL、stage/status、OPID 和耗时格式兼容。
7. 第一版不提供 TLS。监听非回环地址时必须通过防火墙、容器网络或内网访问控制隔离。

## 配置交付

- `config/config.minimal.toml`：服务启动必需参数模板，未写项目使用内置安全默认值。
- `config/config.toml`：常用完整模板。
- `config/config.toml.default`：全部受支持键及说明。
- `config/config.integration.toml`：本机集成测试专用，不用于生产。
- 网关和备份库密码在交付模板中直接使用 `password` 字段配置。配置文件包含明文凭据，生产环境必须限制文件读取权限且不得提交真实密码。

## 已实现与已验证

- TCP 连接上限、空闲超时、关闭等待和系统信号退出。
- `mysql_native_password` 握手认证，错误密码返回 1045。
- inception COM_QUERY 接入同一 Session 审核、执行和备份链。
- 兼容 Archery 执行请求固定发送的 `ignore-warnings`、`sleep`、`sleep_rows`；前者控制 Warning 执行门禁，后两者按旧版语句序号提供可取消节流。
- 12 列旧结果集适配；备份信息表旧 `type` 文本兼容。
- PyMySQL 已验证 SET/USE、VERSION、SHOW VARIABLES、审核执行、ROW Binlog 备份、DDL/DML 回滚、普通查询拒绝和错误认证。
- 并发连接在同一表写入时，备份仅收集执行 connection_id 对应的 ROW event。

## 第六期管理命令补充

- 裸 `SHOW DATABASES` 返回本地虚拟目录 `information_schema`；SHOW TABLES/TABLE STATUS 返回兼容空结果，不代理目标 MySQL。
- 支持 SHOW COLUMNS、WARNINGS、GRANTS、`SELECT CONNECTION_ID()` 及 USE/COM_INIT_DB 的本地协议语义。
- 支持 `inception show|get variables|levels|processlist`、`inception set` 和 `KILL QUERY|CONNECTION`。
- `inception set` 仅修改进程内运行配置，新审核请求读取不可变 Policy 快照；重启后恢复 TOML。
- processlist 使用旧版10列结构并记录来源连接和当前 directive 目标；密码始终脱敏。
- OSC 管理命令仍明确不支持，不伪造任务状态。
- 监听成功后才输出 listening；Windows 启动脚本会在端口占用时报告 PID、程序路径和命令行。

## 验收状态与边界

- 实际 Archery 部署已通过 API 完成 SQL 审核、审批、执行任务状态、回滚 SQL 查询及固定 12 列解析验证。
- 工单 379–389 覆盖 MySQL 5.7/8.0、JSON、CTE、JOIN、多表 UPDATE/DELETE 及多目标表回滚；387、389 验证同一语句的全部回滚集中写入主备份表并能被 Archery 完整读取。
- MySQL 8.0.39 和 MySQL 5.7 Docker 独立实例均已通过真实集成。
- WSL2 Linux CGO/GCC 环境已通过 `go test -race ./...`。
- TLS、Prepared Statement、普通查询服务、OSC 和 gh-ost 不属于第六期。

## 验证入口

```powershell
.\testcases\phase5-6\run.ps1 -TargetPassword <target-password> -GatewayPassword <gateway-password>
```

测试包含真实 ROW Binlog 备份/回滚、二进制和 JSON 等类型、并发连接隔离、旧备份表兼容及协议冒烟。
