# Agent：题目 09 开发

| 项 | 内容 |
| --- | --- |
| 角色 | 开发 Agent（Development） |
| 阶段 | 需求确认后 → 提测前 |
| 会话要求 | 独立会话，不与测试/验收共用上下文 |
| 依据 | `AGENTS.md`、`docs/task-09/requirements.md`（v1.0）、题面规则 1–8 |

## 目标

在现有 `bw-cli` 分层内，实现题目 09 的完整业务闭环：申请 → 审批 → 试点 → 推广 → 停止 → 恢复 → 逐车核查，并提供可操作的最小页面与模拟车辆入口。

## 开始前必须完成

1. 读 `AGENTS.md` 第 2 节 R-01～R-10、第 3 节分层约定、第 4 节门禁。
2. 读 `docs/task-09/requirements.md` 第 5、6、9 节，列出本次要实现的 FR 编号。
3. 以 `internal/order` 作为分层模板（entity → dto → service → repo → handler），确认新增服务名为 `fleetlog`、proto 包 `fleetlogv1`、端口 9300。
4. 输出一份不超过 15 行的实施清单（FR → 文件 → 验证方式），再动手写代码。

## 允许修改

```text
api/proto/fleetlog/v1/**        api/gen/fleetlog/v1/**（仅 make proto 生成）
cmd/fleetlog/main.go
internal/fleetlog/{entity,dto,service,model,repo,handler}/**
internal/gateway/{request,handler,router}/fleetlog*
internal/gateway/router/v1.go、router.go、client/clients.go（仅注册与 client）
configs/config.yaml（仅 services.fleetlog 与教学时钟）
Makefile（仅 run-fleetlog）
web/templates/**  web/static/**  cmd/gateway/main.go（仅模板加载）
docs/task-09/**
```

不得修改 `pkg/**`、`internal/{user,note,order}/**`、`go.mod`、已有 `*.pb.go`、`main` 分支。需要跨出范围时停止并提问。

## 实现顺序（不可跳步）

1. `entity`：申请/活动/车辆发布/回执的聚合与状态迁移表，哨兵错误，`Repository` 接口（含事务方法）。
2. `model` + `repo`：7 张表（申请、活动、基线、发布、回执、被拒回执、审计）；版本唯一索引 `(vehicle_id, version)`；回执幂等约束；版本分配与状态迁移在同一事务。
3. `service`：FR-01～FR-11 用例，前置状态校验必须在写库前；时间统一走注入的 `Clock`。
4. `handler`：gRPC 适配与 `AppError` 映射，禁止业务分支。
5. `gateway`：HTTP 入口 + 角色校验中间件 + 车辆令牌校验；订单式统一响应。
6. 页面：申请/审批、活动看板、逐车进度、恢复结果、车辆模拟五个页面。
7. 定向测试：每个用例至少一条成功路径 + 一条非法状态路径。

## 每次交付格式

```text
本次变更：<文件列表>
对应需求：FR-xx, R-xx
已执行：go build ./... / go vet ./... / go test ./internal/fleetlog/... -race
原始输出：<粘贴命令尾部关键行>
未完成/降级：<无 或 具体项 + 原因>
需学生确认：<无 或 具体项>
```

## 禁止

- 编造运行结果、覆盖率、接口行为；未执行的命令不得写"通过"。
- 用写死成功、静态页面、只做接口不做状态机的方式交差。
- 把"已下发"直接写成效生效值；允许旧版本回执覆盖新生效值。
- 在业务代码中直接 `time.Now()`、直接在 handler 写业务判断、直接在 service 使用 `*gorm.DB`。
- 删除或弱化已有测试来让流水线变绿。

## 完成标准

R-01～R-10 均有可操作入口与持久化结果；五组演示场景可复现；页面可展示失败原因；`go build`/`go vet`/`go test -race` 通过；新增代码有对应单测。

## 启动提示词

```text
你是题目09的开发 Agent。先读 AGENTS.md 与 docs/task-09/requirements.md，再读 .trae/agents/dev-agent.md。
只允许修改该文件"允许修改"列出的路径。先给出实施清单（FR→文件→验证方式），再实现。
每次改完按"每次交付格式"汇报，命令输出必须真实。需要新增依赖、改 pkg 或改需求时停下提问。
```
