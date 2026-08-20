# goInception Plus 项目功能地图

## 1. 产品边界

goInception Plus 是面向 MySQL 5.7/8.0 的 SQL 审核、执行、备份和回滚服务。
TiDB 8.5.3 仅提供 Parser/AST，不引入 TiKV、PD、TiDB Session、Planner、
Executor、DDL、Domain 或 MockTiKV。

首个生产版本保持旧 goInception 的 inception 控制块、MySQL 协议结果列和
主要错误语义。首版不支持 pt-online-schema-change 和 gh-ost。

## 2. 功能依赖图

    MySQL Client / Archery
              |
              v
    MySQL Protocol Gateway
              |
              v
    Legacy Request Decoder
              |
              v
    Lightweight Session State Machine
              |
       +------+------------------+
       |                         |
       v                         v
    TiDB 8.5 Parser          Target MySQL
       |                         |
       v                         v
    Stable AST Projection    Metadata Provider
       |                         |
       +-----------+-------------+
                   v
             Schema Snapshot
                   |
                   v
              Rule Engine
                   |
          +--------+---------+
          |                  |
          v                  v
      Audit Result      Execution Plan
                              |
                     +--------+---------+
                     |                  |
                     v                  v
                Backup Engine     SQL Executor
                     |                  |
                     +--------+---------+
                              v
                     Rollback Generator

## 3. 代码模块地图

| 模块 | 当前职责 | 状态 |
|---|---|---|
| cmd/goinception-plus | 标准输入入口、MySQL 协议服务和配置启动 | 已完成 |
| internal/request | inception 控制块和目标连接参数解析 | 已完成基础版 |
| internal/parser | TiDB Parser、SQLMode、AST 稳定投影 | 已完成一期 |
| internal/model | 跨层稳定业务模型 | 已完成基础版 |
| internal/audit | 规则接口、策略、76 条规则目录和编排 | 旧配置规则已对齐，长尾语义持续对照回归 |
| internal/audit/rules | 独立审核规则实现 | 持续迁移 |
| internal/metadata/mysql | MySQL 5.7/8.0 元数据与影响评估 | 已完成核心能力 |
| internal/metadata | 请求级缓存和批次内 Schema Snapshot | 已完成核心能力 |
| internal/session | 审核、执行、备份请求状态机 | 已完成核心能力 |
| internal/execution | 串行执行、失败边界和状态记录 | 已完成直接执行模式 |
| internal/backup | ROW Binlog 捕获和旧备份库写入 | 已完成核心能力 |
| internal/rollback | 类型安全的 DML/可逆 DDL 回滚生成 | 已完成核心能力 |
| internal/protocol | MySQL 握手、管理命令和 12 列结果集 | 已完成第六期 |
| internal/observability | 健康检查、指标和运行观测 | 已完成第七期基础能力 |

## 4. 核心数据流约束

1. TiDB AST 只能出现在 internal/parser 内部；规则只使用稳定投影。
2. Metadata Provider 只读取目标库，不承载业务规则。
3. 每个请求创建独立元数据缓存，禁止跨请求缓存表结构。
4. 同批次 DDL 必须先更新 Schema Snapshot，后续 SQL 再审核。
5. 任何错误级审核结果默认阻止整个批次执行。
6. 执行成功但备份失败必须停止后续语句并返回严重失败。
7. 协议层只转换结果，不包含审核或执行判断。

## 5. 当前可运行能力

    request.sql
      -> inception envelope
      -> TiDB 8.5.3 parser
      -> stable CREATE TABLE projection
      -> configured rules
      -> JSON AuditRecord

服务通过 TOML 配置连接目标及备份 MySQL，并默认在 4000 端口提供兼容旧 goInception 调用方式的 MySQL 协议服务；标准输入入口继续用于开发调试。
