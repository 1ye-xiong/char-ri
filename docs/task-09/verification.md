# 题目 9 交付验证记录（fleetlog 车队日志策略发布闭环）

记录日期：2026-11-13
验证方式：真实 MySQL + 真实 gRPC/HTTP 调用，不使用任何 mock 数据或静态假页面。

## 1. 环境与启动方式

- 代码目录：`C:\Users\35795\Desktop\char-ri`
- 运行时目录：仓库外的临时副本（避免占用被 Docker 使用的 8080），复制 `configs/`、`web/` 后仅把网关端口改为 8081
- 数据库：MySQL `root@127.0.0.1:3306/char`（每个活动、回执、超时、审计均实际落库）
- 服务端口：fleetlog gRPC `9300`，gateway HTTP `8081`
- 超时配置：`fleetlog.timeout_seconds: 60`

启动命令：

```powershell
go build -o bin/fleetlog.exe ./cmd/fleetlog
go build -o bin/gateway.exe ./cmd/gateway
.\bin\fleetlog.exe      # 工作目录为运行目录
.\bin\gateway.exe       # 工作目录为运行目录
```

运行目录中的 `configs/config.yaml` 与仓库保持一致，仅网关端口不同；页面静态资源由 `web/` 提供。

## 2. 场景执行结果

脚本位于 [scripts/](scripts/)，原始输出位于 [evidence/](evidence/)。以下为每个场景的实测要点。

### 场景 1：正常闭环（申请 → 审批 → 试点 → 推广 → 完成）

证据：[evidence/scenario1.log](evidence/scenario1.log)

- 申请人 `rd_wang` 创建间隔 10 秒申请并提交，负责人 `lead_li` 批准；
- 运营 `ops_zhao` 选 V1、V3 试点，试点车回执成功后 `can_promote=true`；
- 推广下发 V2、V4、V5，5 台车全部成功；
- 负责人标记完成，活动状态 `completed`，汇总 `success=5`、`effective=5`。

### 场景 2：试点失败 → 停止 → 恢复（R-04 / R-08 / R-09）

证据：[evidence/scenario2.log](evidence/scenario2.log)

- V1 回执成功、V4 回执失败（`write config failed: checksum mismatch`）均被接受并记入各自版本；
- 确认推广返回 `HTTP 409 fleetlog_pilot_not_passed`，页面同步展示 `promote_blocked_reason=试点未全部成功：V1=success，V4=failed`；
- 停止活动后 `stop_scope` 精确记录了已发布 V1/V4（版本、批次、状态）与未发布 V2/V3/V5，停止人与原因入库；
- 未批准时执行恢复返回 `HTTP 401 fleetlog_forbidden`；
- 负责人批准后恢复版本 = 当前版本 + 1，目标为活动前基线；两台车恢复回执成功后标记 `restored`。

### 场景 3：超时与迟到回执（教学时钟，60 秒）

证据：[evidence/scenario3.log](evidence/scenario3.log)

- V3 回执成功；V2 只拉取不回执；
- 约 60 秒后后台扫描把 V2 发布置为 `timeout` 并写入超时事件（`timeout_at` 有真实时间）；
- 超时后 V2 再上报成功仍被接受，行明细 `status=success late=true`，超时事件的 `resolved_at` 被补齐，迟到成功不清除超时历史；
- 随后停止、批准恢复、恢复回执、标记 `restored` 全流程通过。

### 场景 4：回执六判定（重复/未知/过期/跨车/非法载荷）

证据：[evidence/scenario4.log](evidence/scenario4.log)

| 用例 | 结果 |
| --- | --- |
| 同一版本第二次成功 | 拒绝 `duplicate_receipt` |
| 从未发布的版本 9999 | 拒绝 `unknown_version` |
| 小于当前期望版本的旧版本 | 拒绝 `stale_version` |
| V1 凭证上报 V4 的回执 | 拒绝 `cross_vehicle` |
| `result` 非 success/failure | 拒绝 `invalid_payload` |

全部拒绝记录出现在 `GET /rejected-receipts`，字段包含车辆、版本、原因与中文说明。补齐试点与推广回执后 `complete` 成功（`success=5`）。

### 场景 5：身份、角色、单活动与离线约束

证据：[evidence/scenario5.log](evidence/scenario5.log)

| 用例 | 结果 |
| --- | --- |
| 无会话访问 | 401 `fleetlog_missing_token` |
| 伪造令牌访问 | 401 `fleetlog_invalid_token` |
| 申请人尝试审批 | 401 `fleetlog_forbidden` |
| 负责人尝试创建申请 | 401 `fleetlog_forbidden` |
| 已有未结束活动时再创建 | 409 `fleetlog_open_activity` |
| 试点阶段提前完成 | 409 `fleetlog_conflict` |
| 推广后仍有未确认车辆时完成 | 409 `fleetlog_unconfirmed_vehicles` |
| 离线车辆拉取配置 | 409 `fleetlog_vehicle_offline` |

最终活动完成，详情接口返回 10 条审计（创建、试点下发、试点通过、推广、5 条回执成功、完成），可按活动过滤。

## 3. 页面验证

通过浏览器实际操作：

- `/fleetlog`：`ops_zhao / ops123456` 登录成功，页面显示“运营-赵工 / 发布运营”，车辆区列出 V1～V5 在线状态；数据来自后台接口，刷新后随数据库变化。
- `/fleetlog/vehicles`：5 张车辆卡片渲染正常；点击 V1“上线拉取”返回真实结果 `拉取成功：v12 / 15 秒（发布状态 成功）`，该版本号与场景 2 恢复后的数据库状态一致。
- 两个页面浏览器控制台均无报错。

## 4. 测试与覆盖率

执行：`go test ./internal/fleetlog/... -cover -count=1`

| 包 | 语句覆盖率 | 说明 |
| --- | --- | --- |
| `internal/fleetlog/dto` | 100.0% | 视图转换与空值分支 |
| `internal/fleetlog/entity` | 97.1% | 申请/活动状态机、回执判定、超时与幂等 |
| `internal/fleetlog/service` | 90.1% | 用例编排、错误映射、并发分配版本 |
| `internal/fleetlog/handler` | 44.7% | 鉴权、错误码、登录、列表 RPC 与全部 DTO→proto 映射 |
| `internal/fleetlog/repo` | 12.5% | 模型与实体双向转换、列表编码、唯一键判定 |

说明：

- 业务核心包（entity / service / dto）均达到并超过 85%。
- `repo` 与 `handler` 是适配层：repo 的 SQL 分支依赖真实 MySQL，handler 的 24 个 RPC 已在场景 1～5 中真实调用验证；单元测试覆盖了其纯函数与鉴权/映射路径，未使用数据库 mock 来虚增覆盖率。
- `-race` 在本机无法执行：`go test -race` 报 `go: -race requires cgo`，本机 `CGO_ENABLED=0` 且无 gcc。并发正确性改用真实并发用例验证：4 个 goroutine 同时触发试点，断言只产生 2 条有效下发，且数据库 `(vehicle_id, version)` 唯一索引与 `SELECT ... FOR UPDATE` 保证版本不重复。

## 5. 实现过程中发现并修复的问题

1. **业务错误被降级为 500**：handler 先把 `AppError` 转成 gRPC status，随后 gRPC 拦截器又把非 `AppError` 重新包装为 `Internal`。修复为 handler 直接返回 `AppError`，由 `pkg/grpcx` 拦截器统一映射；实测恢复为 401/404/409 及稳定业务码（`fleetlog_pilot_not_passed`、`fleetlog_forbidden`、`fleetlog_not_found` 等）。
2. **页面空白**：`LoadHTMLGlob("web/templates/*/*.html")` 的模板名是文件基名，路由却使用 `fleetlog/console.html`。修复为 `console.html` / `vehicles.html` 后页面正常渲染（HTTP 200，内容 7.6KB / 1.9KB）。
