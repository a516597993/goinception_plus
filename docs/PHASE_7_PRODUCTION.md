# 第七期：生产化能力

第七期不在 Plus 内实现新旧双跑、结果 diff、差异白名单或灰度切流；这些由 Archery 负责。Plus 仅提供可安全部署和观测的服务能力。

## 部署模式

- 完整模式：`server.audit_only=false`，保留审核、执行、ROW Binlog备份和回滚。
- 旁路审核模式：`server.audit_only=true`，服务端拒绝所有 `execute=1` 或 `backup=1` 请求。
- `server.audit_timeout` 限制单次审核总时间，客户端断开和 `KILL QUERY` 同样取消请求。

生产环境建议使用 JSON 日志并启用独立观测端口。观测端口只提供：

- `GET /healthz`：进程存活。
- `GET /readyz`：4000端口已成功监听。
- `GET /metrics`：Prometheus文本格式的请求量、失败量、并发数、累计耗时、就绪状态和运行时间。

日志不会记录认证密码或完整SQL，使用 `sql_sha256` 关联请求。调用方可传 `--trace-id=archery:374`；未传时由连接ID和时间生成请求ID。

## 容器与版本

构建：

```powershell
docker build --build-arg VERSION=0.7.0 --build-arg COMMIT=<git-sha> --build-arg BUILD_TIME=<utc-time> -t goinception-plus:0.7.0 .
```

配置文件应以只读卷覆盖 `/etc/goinception-plus/config.toml`，不要把真实密码写进镜像。使用 `goinception-plus -version` 核对版本、提交和构建时间。

## 升级与回退

1. 备份当前二进制、TOML和镜像标签；不要修改目标库或备份库结构。
2. 使用新实例和新端口加载同一规则配置，验证 `/readyz`、MySQL登录和管理命令。
3. 完整跑通 phase3-6 测试及 `testcases/phase7/run-capacity.ps1`。
4. 由 Archery完成双跑和切流。Plus本身不持有工单状态，因此回退只需把Archery地址恢复到旧服务。
5. 回退后保留JSON日志；已有旧格式备份库无需迁移。

容量验收记录100、1000、10000条请求的总耗时、请求字节数、Plus CPU/内存和目标MySQL元数据查询量。服务端硬上限为10,000条，超过上限必须拒绝。
