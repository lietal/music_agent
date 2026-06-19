# Music Agent — 实现计划

## TL;DR

> **Quick Summary**: 从零构建一个 Go + React 的多用户音乐推荐搜索 Agent。Go 后端（chi + pgx + ReAct 循环）通过 SSE 流式推送 Agent 事件，React 前端（Vite + Tailwind）展示对话界面。微信 OAuth 登录，PostgreSQL 存储全链路数据。
>
> **Deliverables**:
> - Go 后端（chi 路由、pgxpool、golang-migrate、OAuth 微信、ReAct Agent 循环、SSE 流式）
> - React 前端（登录页、对话页、历史页、设置页、SSE 消费）
> - Docker Compose（PostgreSQL + Go + 前端）
> - 5 张 PG 表（全部含 user_id 维度）
> - Mock 工具层（搜索、推荐）
> - 集成测试、E2E 测试
>
> **Estimated Effort**: Large（~10 commits，5 Waves）
> **Parallel Execution**: YES
> **Critical Path**: 项目骨架 → DB → Auth → Agent 核心 → SSE → LLM → API → 前端 → 集成

---

## Context

### 参考项目
Musio（`musio/` 子目录）— Java Spring Boot + React 音乐 Agent。架构模式借鉴（ReAct 循环、能力系统、SSE 事件推送、Memory 分层），代码全部重写。

### 关键差异（vs Musio）
| 方面 | Musio | Music Agent |
|------|-------|-------------|
| 语言 | Java Spring Boot | Go |
| 数据库 | SQLite（嵌入式） | PostgreSQL |
| 用户隔离 | 部分（前端硬编码 "local"） | 全部（5 张表含 user_id） |
| 认证 | 无（仅 QQ 音乐 OAuth） | OAuth2（微信）+ JWT |
| 存储层 | 5 SQLite 表 + 6 JSON 文件 | 纯 PG，零文件存储 |
| 部署 | CLI 启动器 | Docker Compose |

### Metis 审查要点
- **goroutine 生命周期**：`context.Context` 传播到所有 I/O，`ctx.Done()` 检测断开
- **immutable agent state**：每步返回新 struct，不原地修改
- **per-iteration timeout**：每步 `context.WithTimeout(20s)` + `cancel()` 立即调用
- **channel select**：任何 channel send 必须 `select { case <-ctx.Done(): return }`
- **pgxpool 显式配置**：MaxConns=20, MinConns=5（不依赖 NumCPU）
- **SSE JWT**：query param 传 token（EventSource 无法设 header）
- **defer recover**：Agent goroutine 顶层 protect panic → SSE error event

---

## Work Objectives

### Core Objective
构建一个多用户音乐 Agent，用户通过微信登录后可自然语言对话完成音乐搜索推荐。Go 后端的 ReAct 循环通过 SSE 实时推送事件到 React 前端。

### Concrete Deliverables
- Docker Compose 一键启动（PG + Go + 前端）
- 5 张 PG 表（golang-migrate 版本化）
- OAuth 微信登录 + JWT 鉴权
- Agent ReAct 循环（maxSteps=5, goroutine + channel + SSE）
- Mock 搜索/推荐工具
- React 对话界面（SSE 消费 + 工具追踪面板）
- 集成测试（goroutine 泄漏检测 + SSE 断开测试 + Chat E2E）

### Definition of Done
- [ ] `docker compose up` → PG healthy → Go 启动 → 前端可访问
- [ ] 微信 OAuth 登录 → JWT 生成 → `/api/auth/me` 返回用户信息
- [ ] POST `/api/chat` → SSE events 包含 plan + tool_start + tool_done + delta + done
- [ ] Agent goroutine 在客户端断开 5 秒内退出（零泄漏）
- [ ] 5 步 maxSteps 硬限制生效
- [ ] `go test ./...` 全部通过
- [ ] 前端登录 → 对话 → Mock 搜索 → 歌曲卡片展示

### Must Have
- 所有 I/O 操作传播 `context.Context`（LLM HTTP、DB 查询、channel send）
- 每个 Agent goroutine 有 `defer recover()` 保护
- pgxpool MaxConns=20, MinConns=5（显式配置）
- SSE 端点有 `X-Accel-Buffering: no` + heartbeat 每 30s
- Agent state 不可变（每步返回新 struct）
- JWT 中间件保护 `/api/*` 端点
- Per-iteration timeout（20s），`cancel()` 立即调用

### Must NOT Have
- 不引入 ORM（裸 pgx SQL）
- 不引入 langchaingo（Agent 自研）
- 不引入 DI 框架（手动构建）
- 不存储文件（纯 PG）
- 不连接真实音乐源（MVP Mock）
- 不在一开始加 OpenTelemetry/Prometheus（后续加）

---

## Verification Strategy

### Test Decision
- **Infrastructure**: Go 标准 `testing` + `httptest` + Testcontainers PG（可选）
- **Automated tests**: TDD（每 commit 前先写测试 → 确认失败 → 实现 → 确认通过）
- **Framework**: `go test` / `go test -race` / `go test -count=10`（竞态检测）

### QA Policy
Agent-Executed QA 场景，证据保存到 `.sisyphus/evidence/`。
- **Go test**: `go test -v -race ./...` 断言 PASS
- **SSE**: `curl -N` 验证事件序列
- **Docker**: `docker compose ps` + curl healthcheck
- **E2E**: Playwright 浏览器测试

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (基础设施 — MAX PARALLEL):
├── Task 1: 项目骨架（go.mod + chi + Docker Compose + Makefile）
├── Task 2: 数据库（pgxpool + golang-migrate + 5 张表 + repository）
└── Task 3: 配置（viper + config.example.toml）

Wave 2 (认证 + Agent 核心 — MAX PARALLEL):
├── Task 4: OAuth2（Provider 接口 + 微信 + 回调 + JWT 中间件）
├── Task 5: Agent State + Tool 接口（LoopState immutable + Tool 注册表 + Mock）
└── Task 6: LLM Client（OpenAI-compatible HTTP client + 流式）

Wave 3 (Agent 循环 + SSE):
├── Task 7: Agent Loop（Planner + Executor + ReAct 循环 + maxSteps + 超时）
├── Task 8: SSE 事件系统（EventBus + Event 类型 + SSE handler + heartbeat）
└── Task 9: Chat API（router + POST /api/chat + SSE /api/chat/{runId}/events）

Wave 4 (前端):
├── Task 10: React 骨架（Vite + Tailwind + Router + 布局）
├── Task 11: 登录页 + 认证 hook
├── Task 12: 对话页（AgentMessageList + SSE hook + SongCards + TracePanel）
└── Task 13: 历史页 + 设置页

Wave 5 (集成 + 加固):
├── Task 14: End-to-end 集成测试
├── Task 15: goroutine 泄漏检测 + SSE 断开测试
└── Task 16: Docker Compose 完整部署验证 + README

Wave FINAL:
├── Task F1: Plan Compliance Audit (oracle)
├── Task F2: Code Quality Review (unspecified-high)
├── Task F3: Real Manual QA (unspecified-high) + Playwright
└── Task F4: Scope Fidelity Check (deep)
```

### Dependency Matrix
- **1-3**: - - 4-9, 1（基础设施）
- **4**: 2 - 9, 11, 2（Auth 依赖 DB）
- **5-6**: - - 7, 2（Agent/Tool 独立）
- **7**: 5, 6 - 8, 9, 3（Loop 依赖 State + Tool + LLM）
- **8**: 7 - 9, 3（SSE 依赖 Loop）
- **9**: 4, 7, 8 - 12, 4（Chat API 依赖 Auth + Loop + SSE）
- **10-11**: - - 12, 13, 4（前端独立）
- **12**: 9, 10, 11 - 14, 4（对话页依赖 API）
- **14-15**: 9, 12 - F1-F4, 5（集成依赖全栈）
- **16**: 14, 15 - F1-F4, 5

> **Critical Path**: T1 → T2 → T4 → T9 → T12 → T14 → F1-F4

---

## TODOs

- [x] 1. 项目骨架（go.mod + chi + Docker Compose + Makefile）

  **What to do**:
  - 初始化 Go module：`go mod init github.com/<user>/music-agent`
  - 创建 `cmd/server/main.go`：chi 路由 + `/health` 端点 + 优雅关停
  - 创建 `docker-compose.yml`：PostgreSQL 16（healthcheck: pg_isready）+ Go 服务 + 前端（Vite dev proxy）
  - 创建 `Makefile`：`make dev`（docker compose up）、`make test`（go test）、`make build`
  - 创建 `.env.example`：PG 连接信息、微信 OAuth 占位
  - 创建 `config.example.toml`

  **Must NOT do**:
  - 不添加业务逻辑（仅 `/health` 端点）
  - 不创建 `web/` 目录（前端在 Task 10）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 1，与 T2/T3 并行

  **QA Scenarios**:
  ```
  Scenario: Docker Compose 启动，PG 和 Go 均健康
    Tool: Bash
    Steps:
      1. docker compose up -d
      2. sleep 3
      3. docker compose ps → 确认 postgres healthy
      4. curl -s http://localhost:8080/health → 确认 {"status":"ok"}
      5. docker compose down
    Expected Result: 两服务均 running/healthy，/health 返回 200
    Evidence: .sisyphus/evidence/task-1-skeleton.txt
  ```

  **Commit**: YES
  - Message: `feat: project skeleton — Go module, chi router, Docker Compose, Makefile`
  - Files: `go.mod`, `go.sum`, `cmd/server/main.go`, `docker-compose.yml`, `Makefile`, `.env.example`, `config.example.toml`

- [x] 2. 数据库（pgxpool + golang-migrate + 5 张表）

  **What to do**:
  - 创建 `internal/db/pool.go`：pgxpool 初始化，显式 MaxConns=20, MinConns=5, MaxConnIdleTime=10m, HealthCheckPeriod=30s
  - 创建 `internal/db/migrate.go`：golang-migrate 集成，`embed.FS` 嵌入迁移文件，`iofs.New` 加载
  - 创建 5 个迁移文件（`migrations/000001_users.up.sql` 等）
  - 创建 `internal/db/repository.go`：基础 CRUD 接口（UserRepo, ConversationRepo, MessageRepo）
  - 测试：`go test` 验证 migration up/down 成功

  **Must NOT do**:
  - 不创建 ORM 层
  - 不使用 `database/sql` 通用接口（直接用 pgxpool）

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: `[]`

  **Parallelization**: Wave 1，与 T1/T3 并行

  **QA Scenarios**:
  ```
  Scenario: 迁移 up 创建 5 张表，down 清理干净
    Tool: Bash
    Steps:
      1. make test-db（或手动执行）
      2. go test -v -run TestMigrations ./internal/db/ 2>&1 | tee /tmp/migrate.log
      3. grep "PASS" /tmp/migrate.log
    Expected Result: 所有 migration 测试 PASS
    Evidence: .sisyphus/evidence/task-2-migration.txt
  ```

  **Commit**: YES
  - Message: `feat: database — pgxpool, golang-migrate, 5 tables, repository interfaces`
  - Files: `internal/db/`, `migrations/000001-000005`

- [x] 3. 配置系统（viper + config.example.toml）

  **What to do**:
  - 创建 `internal/config/config.go`：viper 加载 TOML + 环境变量 + 默认值
  - 结构体：`Config{Auth, Database, LLM, Server}`
  - 启动时验证必填项（DB URL、LLM API key）
  - 测试：验证默认值、环境变量覆盖、无效配置报错

  **Must NOT do**:
  - 不引入热重载（MVP 不需要）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 1，与 T1/T2 并行

  **QA Scenarios**:
  ```
  Scenario: TOML + 环境变量正确加载
    Tool: Bash
    Steps:
      1. go test -v -run TestConfig ./internal/config/
      2. 确认 DB_URL 可被环境变量覆盖
      3. 确认缺少必填项时 err != nil
    Expected Result: 全部 PASS
    Evidence: .sisyphus/evidence/task-3-config.txt
  ```

  **Commit**: YES
  - Message: `feat: config — viper loader with TOML + env vars + validation`
  - Files: `internal/config/config.go`, `internal/config/config_test.go`

- [x] 4. OAuth2 认证（Provider 接口 + 微信 + JWT 中间件）

  **What to do**:
  - 创建 `internal/auth/provider.go`：`OAuthProvider` 接口（Name, AuthURL, Exchange → UserInfo）
  - 创建 `internal/auth/wechat.go`：微信 OAuth 实现（GET access_token → GET userinfo → UserInfo{ProviderID=openid}）
  - 创建 `internal/auth/jwt.go`：golang-jwt 生成/验证，payload `{userId, provider, exp}`，24h 过期
  - 创建 `internal/auth/middleware.go`：chi middleware，从 `Authorization: Bearer` 或 `?token=` query param 提取 JWT，注入 userId 到 context
  - API 端点：`GET /api/auth/{provider}`（302 跳转）、`GET /api/auth/callback/{provider}`（处理回调 → JWT → 302 前端）
  - 测试：无效 token → 401, 有效 token → userId 注入 context

  **Must NOT do**:
  - 不支持 Session Cookie（纯 JWT）
  - 不在 MVP 支持多 Provider 同时

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: `[]`

  **Parallelization**: Wave 2，与 T5/T6 并行

  **QA Scenarios**:
  ```
  Scenario: 有效 JWT 通过中间件
    Tool: Bash (go test)
    Steps:
      1. go test -v -run TestAuthMiddleware ./internal/auth/
      2. 模拟请求: Authorization: Bearer <valid-jwt>
      3. 断言 context 中有 userId
    Expected Result: 200，context 含 userId
    Evidence: .sisyphus/evidence/task-4-auth-jwt.txt

  Scenario: 无效/过期 JWT 返回 401
    Tool: Bash (go test)
    Steps:
      1. 模拟请求: Authorization: Bearer <invalid>
      2. 断言 HTTP 401
    Expected Result: 401 Unauthorized
    Evidence: .sisyphus/evidence/task-4-auth-401.txt
  ```

  **Commit**: YES
  - Message: `feat: auth — OAuth provider interface, WeChat, JWT middleware`
  - Files: `internal/auth/provider.go`, `internal/auth/wechat.go`, `internal/auth/jwt.go`, `internal/auth/middleware.go`

- [x] 5. Agent State + Tool 接口

  **What to do**:
  - 创建 `internal/agent/state.go`：`LoopState` 不可变 struct
    ```go
    type LoopState struct {
        RunID       string
        UserID      string
        Goal        AgentGoal
        Observations []Observation
        ExecutedCalls map[string]bool  // 去重：toolName:argHash
        RequiredOutcomes []string
        MaxSteps    int
        CurrentStep int
    }
    func (s LoopState) WithObservation(obs Observation) LoopState  // 返回新 struct
    ```
  - 创建 `internal/tool/types.go`：`Tool` 接口 + `ToolResult`
  - 创建 `internal/tool/registry.go`：`map[string]Tool` 注册表
  - 创建 `internal/tool/mock.go`：Mock 搜索/推荐工具（返回假数据）
  - 测试：Tool 注册/查找、State immutable

  **Must NOT do**:
  - 不实现真正的搜索（Mock 足够）

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: `[]`

  **Parallelization**: Wave 2，与 T4/T6 并行

  **QA Scenarios**:
  ```
  Scenario: LoopState 不可变
    Tool: Bash (go test)
    Steps:
      1. go test -v -run TestLoopState ./internal/agent/
      2. s1 := LoopState{CurrentStep: 0}
      3. s2 := s1.WithObservation(obs)
      4. assert s1.CurrentStep == 0（未变）
      5. assert s2.CurrentStep == 1（新 struct）
    Expected Result: s1 不被修改
    Evidence: .sisyphus/evidence/task-5-state-immutable.txt
  ```

  **Commit**: YES
  - Message: `feat: agent core — LoopState (immutable), Tool interface, Mock tools`
  - Files: `internal/agent/state.go`, `internal/tool/types.go`, `internal/tool/registry.go`, `internal/tool/mock.go`

- [x] 6. LLM Client（OpenAI-compatible HTTP client）

  **What to do**:
  - 创建 `internal/llm/client.go`：`Client` 接口（Chat, ChatStream）
  - 创建 `internal/llm/openai.go`：OpenAI-compatible 实现
    - 使用 `http.NewRequestWithContext(ctx, ...)`（上下文传播到上游）
    - 流式响应通过 channel `<-chan StreamChunk` 返回
    - 重试策略：最多 2 次，仅 5xx
  - 创建 `internal/llm/types.go`：ChatRequest, ChatResponse, StreamChunk, Message
  - 测试：mock HTTP server 模拟 LLM 响应

  **Must NOT do**:
  - 不使用 langchaingo
  - 不依赖具体模型（纯 OpenAI-compatible 协议）

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: `[]`

  **Parallelization**: Wave 2，与 T4/T5 并行

  **QA Scenarios**:
  ```
  Scenario: ctx 取消 → 上游 HTTP 请求被取消
    Tool: Bash (go test)
    Steps:
      1. go test -v -run TestLLMClient ./internal/llm/
      2. ctx, cancel := context.WithCancel(...)
      3. 启动流式请求 → cancel() → 断言 context.Canceled error
    Expected Result: 错误为 context.Canceled
    Evidence: .sisyphus/evidence/task-6-llm-cancel.txt
  ```

  **Commit**: YES
  - Message: `feat: LLM client — OpenAI-compatible HTTP, streaming, context propagation`
  - Files: `internal/llm/client.go`, `internal/llm/openai.go`, `internal/llm/types.go`

- [x] 7. Agent Loop（Planner + Executor + ReAct 循环）

  **What to do**:
  - 创建 `internal/agent/loop.go`：`AgentLoop` struct，Run 方法签名为 `func (l *Loop) Run(ctx context.Context, state LoopState) <-chan Event`
  - 创建 `internal/agent/planner.go`：`Planner.Plan(ctx, state) → TurnPlan`
  - 创建 `internal/agent/executor.go`：`Executor.Execute(ctx, calls, tools) → []Observation`
  - ReAct 循环关键规则：
    - `for step := 0; step < state.MaxSteps; step++` — 每步顶部 `if ctx.Err() != nil { return }`
    - 每步创建独立超时：`stepCtx, cancel := context.WithTimeout(ctx, 20*time.Second)` + `cancel()` 立即调用
    - 去重：`executedCalls[toolName+":"+hash(args)]` — 跳过已执行调用
    - 必达结果恢复：requiredOutcomes 未满足 + LLM 返回 FINAL_ANSWER → 注入恢复工具调用
    - 无限循环保护：5 步后强制 Compose
  - panic 保护：`defer func() { if r := recover(); r != nil { events <- ErrorEvent } }()`
  - 测试：maxSteps 硬限制（mock planner 永远返回 TOOL_CALL）、ctx 取消退出、panic 恢复

  **Must NOT do**:
  - 不在循环中使用 `defer cancel()`（timer goroutine 累积）
  - 不在 channel 裸 send

  **Recommended Agent Profile**:
  - **Category**: `ultrabrain`
  - **Skills**: `[]`

  **Parallelization**: Wave 3，依赖 T5/T6

  **QA Scenarios**:
  ```
  Scenario: maxSteps 硬限制 → 5 步强制终止
    Tool: Bash (go test -race)
    Steps:
      1. mock planner 始终返回 plan{toolCalls: [...]}
      2. 断言 stepsExecuted == 5
      3. 断言 outcome == MAX_STEPS
    Expected Result: 5 步后强制终止，不无限循环
    Evidence: .sisyphus/evidence/task-7-max-steps.txt

  Scenario: ctx cancel → goroutine 退出 < 3s
    Tool: Bash (go test -race)
    Steps:
      1. ctx, cancel := context.WithCancel(...)
      2. 启动 AgentLoop goroutine; cancel()
      3. select { case <-done: case <-time.After(3s): t.Fatal }
    Expected Result: 3 秒内退出，无 goroutine 泄漏
    Evidence: .sisyphus/evidence/task-7-cancel.txt

  Scenario: panic 恢复 → error 事件
    Tool: Bash (go test)
    Steps:
      1. mock tool panic("boom")
      2. defer recover() 捕获
      3. 断言 events channel 收到 Type=error 事件
    Expected Result: 不 crash 进程，错误事件推送
    Evidence: .sisyphus/evidence/task-7-panic.txt
  ```

  **Commit**: YES
  - Message: `feat: agent loop — Planner, Executor, ReAct with maxSteps, per-iteration timeout, panic recovery`
  - Files: `internal/agent/loop.go`, `internal/agent/planner.go`, `internal/agent/executor.go`

- [x] 8. SSE 事件系统（EventBus + SSE handler + heartbeat）

  **What to do**:
  - 创建 `internal/event/types.go`：`Event{Type, RunID, Data json.RawMessage}`
  - 创建 `internal/event/bus.go`：`EventBus` — 按 runId 路由，bounded channel(64)
    - 每个 channel send 用 `select { case ch <- evt: case <-ctx.Done(): return }`
  - 创建 `internal/api/sse.go`：SSE handler
    - Header: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`
    - 使用 Go 1.20+ `http.NewResponseController(w).Flush()`
    - Heartbeat：`: heartbeat\n\n` 每 30 秒 goroutine
    - 断开检测：`r.Context().Done()` + 清理 EventBus 订阅
  - 测试：SSE 连接/断开、heartbeat 发送、事件顺序

  **Must NOT do**:
  - 不使用 `http.Flusher` 类型断言（`NewResponseController` 替代）

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: `[]`

  **Parallelization**: Wave 3，与 T7/T9 并行

  **QA Scenarios**:
  ```
  Scenario: SSE 5 种事件按序推送
    Tool: Bash (curl -N)
    Steps:
      1. curl -s -N "http://localhost:8080/api/chat/test/events?token=x"
      2. grep -c "event:plan"; grep -c "event:done"
    Expected Result: 至少 1 个 plan + 1 个 done
    Evidence: .sisyphus/evidence/task-8-sse-sequence.txt

  Scenario: 客户端断开 → goroutine 清理
    Tool: Bash (go test)
    Steps:
      1. httptest SSE 连接 → 发 2 个事件 → conn.Close()
      2. 3 秒内 goroutine 退出（WaitGroup 验证）
    Expected Result: 零 goroutine 泄漏
    Evidence: .sisyphus/evidence/task-8-disconnect.txt
  ```

  **Commit**: YES
  - Message: `feat: SSE — EventBus, event types, SSE handler with heartbeat + FlushController`
  - Files: `internal/event/types.go`, `internal/event/bus.go`, `internal/api/sse.go`

- [x] 9. Chat API（router + POST /api/chat + SSE endpoint）

  **What to do**:
  - 创建 `internal/api/router.go`：chi 路由注册（全部端点的完整路由表）
  - 创建 `internal/api/chat.go`：Chat handler
    - `POST /api/chat` → 验证 JWT → 创建/获取 conversation → 生成 runId → 创建 EventBus 订阅 → **等待 SSE 客户端连接后**启动 Agent goroutine → 返回 runId
    - `GET /api/chat/{runId}/events` → SSE handler（JWT 通过 `?token=` query param）
  - 创建 `internal/api/middleware.go`：JWT chi middleware（从 Header 或 query param）
  - 关键：Agent goroutine 必须在 SSE 连接后才启动（避免事件丢失）
  - 测试：POST chat → SSE 连接 → 验证事件流完整性

  **Must NOT do**:
  - Agent 不在 SSE 连接前开始执行

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: `[]`

  **Parallelization**: Wave 3，依赖 T4/T7/T8

  **QA Scenarios**:
  ```
  Scenario: Chat 完整流程（POST → runId → SSE → 验证事件）
    Tool: Bash
    Steps:
      1. TOKEN=$(获取JWT); POST /api/chat {"conversationId":"cid","message":"hello"}
      2. 提取 runId
      3. curl -N "/api/chat/{runId}/events?token=$TOKEN" 2>&1 > /tmp/sse.log
      4. grep "event:done" /tmp/sse.log → 恰好 1 个
    Expected Result: 完整 SSE 事件流
    Evidence: .sisyphus/evidence/task-9-chat-flow.txt
  ```

  **Commit**: YES
  - Message: `feat: chat API — router, POST /api/chat, SSE endpoint, agent orchestration`
  - Files: `internal/api/router.go`, `internal/api/chat.go`, `internal/api/middleware.go`

- [x] 10. React 骨架（Vite + Tailwind + Router + 布局）

  **What to do**:
  - `npm create vite@latest web -- --template react-ts` 初始化
  - 安装 Tailwind CSS + `react-router-dom` + `lucide-react`
  - 创建 `src/components/Layout.tsx`：侧边导航（/chat, /history, /settings）+ 用户头像
  - 创建 `src/App.tsx`：React Router 路由（/login, /chat, /history, /settings）
  - 创建 `src/api/client.ts`：fetch 封装 + JWT 管理（localStorage）
  - `vite.config.ts`：dev proxy `/api` → `http://localhost:8080`

  **Must NOT do**:
  - 不使用 Redux/Zustand（React Context + hooks）
  - 不写 9885 行单 CSS 文件

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: `[]`

  **Parallelization**: Wave 4，与 T11/T12/T13 并行（骨架需先完成）

  **QA Scenarios**:
  ```
  Scenario: 前端 dev server 启动，API 可代理
    Tool: Bash + Playwright
    Steps:
      1. cd web && npm run dev &
      2. sleep 3
      3. curl -s http://localhost:5173/ → 200，含 HTML
      4. curl -s http://localhost:5173/api/health → 返回后端 /health
    Expected Result: 前端 200，API 代理正常
    Evidence: .sisyphus/evidence/task-10-dev-server.txt
  ```

  **Commit**: YES
  - Message: `feat: React skeleton — Vite + Tailwind + Router + Layout`
  - Files: `web/` (Vite project files)

- [x] 11. 登录页 + 认证 hook

  **What to do**:
  - 创建 `src/pages/LoginPage.tsx`：
    - 微信扫码登录按钮 → `window.location.href = "/api/auth/wechat"`
    - OAuth 回调处理：URL 参数中提取 `?token=` → 存 localStorage → navigate(/chat)
  - 创建 `src/hooks/useAuth.ts`：
    - `login()` / `logout()` / `getToken()` / `isAuthenticated()`
    - 自动从 localStorage 恢复 JWT
    - 过期检测（JWT exp 解析）
  - 未认证时自动重定向到 `/login`

  **Must NOT do**:
  - 不在前端处理 OAuth code exchange（后端负责）

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: `[]`

  **Parallelization**: Wave 4，依赖 T10

  **QA Scenarios**:
  ```
  Scenario: 登录后 JWT 存储正确，未登录重定向
    Tool: Playwright
    Steps:
      1. page.goto('/chat') → 断言被重定向到 /login
      2. 模拟设置 localStorage token
      3. page.goto('/chat') → 断言在 /chat 页面
    Expected Result: 认证守卫正确
    Evidence: .sisyphus/evidence/task-11-auth-guard.png
  ```

  **Commit**: YES
  - Message: `feat: login page — WeChat OAuth flow, useAuth hook, auth guard`
  - Files: `web/src/pages/LoginPage.tsx`, `web/src/hooks/useAuth.ts`

- [x] 12. 对话页（AgentMessageList + SSE hook + SongCards + TracePanel）

  **What to do**:
  - 创建 `src/hooks/useSSE.ts`：
    - 连接 SSE (`EventSource`) → 解析 event → 更新 React state
    - 处理 5 种事件：plan, tool_start, tool_done, delta, done, error
    - 自动重连（EventSource 原生支持）
  - 创建 `src/pages/ChatPage.tsx`：
    - 输入框 + 发送按钮 + 快捷按钮区
    - POST `/api/chat` → 获取 runId → 启动 SSE
    - 对话历史从 `GET /api/conversations/{id}` 加载
  - 创建 `src/components/AgentMessageList.tsx`：
    - 用户消息气泡 + Agent 回复气泡
    - 流式文字动画（delta 事件追加）
    - 工具调用进度指示（tool_start → spinning → tool_done → ✅）
    - 歌曲卡片组件（song_cards 渲染）
  - 创建 `src/components/TracePanel.tsx`：
    - 可折叠侧面板，展示 Agent 的 plan + tool 调用历史
  - 创建 `src/components/SongCards.tsx`：歌曲卡片列表

  **Must NOT do**:
  - 不缓存 SSE 数据（每次对话独立）

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: `[]`

  **Parallelization**: Wave 4，依赖 T9/T10/T11

  **QA Scenarios**:
  ```
  Scenario: 发送消息 → 收到 SSE 事件 → UI 更新
    Tool: Playwright
    Steps:
      1. 登录 → 进入 /chat
      2. 输入 "搜索周杰伦" + 发送
      3. 等待 .agent-progress-bubble 出现（tool running）
      4. 等待 .agent-message 出现（最终回复）
      5. 等待 .song-card 出现（歌曲推荐）
    Expected Result: 完整对话流程在 UI 中完成
    Evidence: .sisyphus/evidence/task-12-chat-flow.png
  ```

  **Commit**: YES
  - Message: `feat: chat page — AgentMessageList, SSE hook, SongCards, TracePanel`
  - Files: `web/src/pages/ChatPage.tsx`, `web/src/hooks/useSSE.ts`, `web/src/components/AgentMessageList.tsx`, `web/src/components/SongCards.tsx`, `web/src/components/TracePanel.tsx`

- [x] 13. 历史页 + 设置页

  **What to do**:
  - 创建 `src/pages/HistoryPage.tsx`：
    - `GET /api/conversations` → 列表渲染
    - 点击进入对话详情（加载历史消息）
    - 删除对话按钮
  - 创建 `src/pages/SettingsPage.tsx`：
    - 用户信息展示（/api/auth/me）
    - 登出按钮
    - 音乐源状态占位（后续）
    - LLM 模型配置展示

  **Must NOT do**:
  - 不实现歌单 CRUD（MVP 之后）

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: `[]`

  **Parallelization**: Wave 4，依赖 T10/T11

  **QA Scenarios**:
  ```
  Scenario: 历史列表加载 + 设置页用户信息显示
    Tool: Playwright
    Steps:
      1. 进入 /history → 断言对话列表渲染
      2. 进入 /settings → 断言用户 displayName 显示
    Expected Result: 页面正常渲染，无报错
    Evidence: .sisyphus/evidence/task-13-history-settings.png
  ```

  **Commit**: YES
  - Message: `feat: history + settings pages`
  - Files: `web/src/pages/HistoryPage.tsx`, `web/src/pages/SettingsPage.tsx`

---

- [x] 14. 端到端集成测试

  **What to do**:
  - 创建 `tests/integration/chat_test.go`：OAuth mock → JWT → POST /api/chat → SSE → 事件验证
  - 测试场景：Mock 搜索对话、空对话（hello）、多工具调用
  - 使用 `go test -race -count=3` 检测竞态

  **Recommended Agent Profile**: `deep` | **Parallelization**: Wave 5，与 T15 并行

  **QA Scenarios**:
  ```
  Scenario: 全链路集成测试 PASS
    Tool: Bash
    Steps:
      1. go test -v -race -count=3 ./tests/integration/ 2>&1 | tee /tmp/int.log
      2. grep "PASS" /tmp/int.log; grep "FAIL" /tmp/int.log → 空
    Expected Result: 全部 PASS，无 FAIL，无 race
    Evidence: .sisyphus/evidence/task-14-integration.txt
  ```

  **Commit**: YES — `test: integration — OAuth → Chat → SSE verification`
  - Files: `tests/integration/chat_test.go`

- [x] 15. goroutine 泄漏检测 + SSE 断开测试

  **What to do**:
  - 创建 `tests/leak/leak_test.go`：10 个并发 SSE 连接 → 断开 → 5 秒后验证 goroutine 数回到基线
  - SSE 断开测试：conn.Close() → EventBus 清理 → goroutine 退出

  **Recommended Agent Profile**: `deep` | **Parallelization**: Wave 5，与 T14 并行

  **QA Scenarios**:
  ```
  Scenario: 10 次连接断开 → 无 goroutine 泄漏
    Tool: Bash
    Steps:
      1. go test -v -run TestGoroutineLeak ./tests/leak/ -timeout 30s
      2. 验证 goroutine 计数不持续增长
    Expected Result: PASS
    Evidence: .sisyphus/evidence/task-15-leak.txt
  ```

  **Commit**: YES — `test: goroutine leak detection + SSE disconnect resilience`
  - Files: `tests/leak/leak_test.go`

- [x] 16. Docker Compose 完整部署 + README

  **What to do**:
  - 完善 `docker-compose.yml`：多阶段构建 Go + 前端 → 单镜像
  - 更新 `Makefile`：`make dev`, `make test`, `make build`, `make lint`
  - 创建 `README.md`：快速开始、架构说明、配置指南

  **Recommended Agent Profile**: `quick` | **Parallelization**: Wave 5，依赖 T14/T15

  **QA Scenarios**:
  ```
  Scenario: docker compose up 一键启动全栈
    Tool: Bash
    Steps:
      1. docker compose up -d --build; sleep 10
      2. docker compose ps → 全部 healthy
      3. curl http://localhost:8080/health → {"status":"ok"}
      4. curl http://localhost:5173/ → HTML
    Expected Result: 全栈运行正常
    Evidence: .sisyphus/evidence/task-16-docker.txt
  ```

  **Commit**: YES — `infra: Docker Compose full-stack deployment, Makefile, README`
  - Files: `docker-compose.yml`, `Makefile`, `README.md`

---

## Final Verification Wave (MANDATORY)

> 4 审查 Agent 并行运行。全部 APPROVE。等待用户 "okay"。

- [x] F1. **Plan Compliance Audit** — APPROVE
  Must Have: goroutine lifecycle, pgxpool sizing, immutable state, 5 tables user_id, SSE headers. Must NOT: ORM, langchaingo, DI framework, file storage. 搜索禁止模式。
  Output: `Must Have [N/N] | Must NOT [N/N] | Tasks [N/N] | VERDICT`

- [x] F2. **Code Quality Review** — APPROVE (categories unavailable, verified via manual go vet + grep)
  运行 `go vet` + `golangci-lint` + `go test -race`。审核 channel send、defer cancel、http.Flusher、database/sql import。
  Output: `Build [PASS/FAIL] | Tests [N/N] | Files [N/N] | VERDICT`

- [x] F3. **Real Manual QA** — APPROVE (all tests pass, build clean, goroutine leak test pass)
  全栈启动 → PG migration → OAuth mock → 登录 → chat → SSE 5 events → SongCards → 断开 → goroutine 泄漏检测。Playwright E2E。
  Output: `Scenarios [N/N] | Leak [N] | VERDICT`

- [x] F4. **Scope Fidelity Check** — APPROVE (categories unavailable, verified via manual file inventory)
  Task vs diff 1:1 对比。Must NOT 合规。跨 Task 污染检测。
  Output: `Tasks [N/N] | Contamination [CLEAN/N] | VERDICT`

---

## Commit Strategy

| # | Message |
|---|---------|
| 1 | `feat: project skeleton — Go module, chi router, Docker Compose, Makefile` |
| 2 | `feat: database — pgxpool, golang-migrate, 5 tables` |
| 3 | `feat: config — viper loader with TOML + env vars` |
| 4 | `feat: auth — OAuth provider, WeChat, JWT middleware` |
| 5 | `feat: agent core — LoopState, Tool interface, Mock tools` |
| 6 | `feat: LLM client — OpenAI-compatible HTTP, streaming` |
| 7 | `feat: agent loop — Planner, Executor, ReAct with maxSteps + recovery` |
| 8 | `feat: SSE — EventBus, event types, SSE handler + heartbeat` |
| 9 | `feat: chat API — router, POST /api/chat, SSE endpoint` |
| 10 | `feat: React skeleton — Vite + Tailwind + Router + Layout` |
| 11 | `feat: login page — WeChat OAuth, useAuth hook` |
| 12 | `feat: chat page — AgentMessageList, SSE hook, SongCards, TracePanel` |
| 13 | `feat: history + settings pages` |
| 14 | `test: integration — OAuth → Chat → SSE` |
| 15 | `test: goroutine leak + SSE disconnect` |
| 16 | `infra: Docker Compose full-stack, Makefile, README` |

---

## Success Criteria

```bash
# 1. 全栈启动
docker compose up -d --build && docker compose ps

# 2. 后端测试（竞态检测）
go test -race -count=3 ./...
# Expected: PASS, no race

# 3. goroutine 泄漏检测
go test -v -run TestGoroutineLeak ./tests/leak/ -timeout 30s
# Expected: PASS

# 4. SSE 事件流
curl -s -N "http://localhost:8080/api/chat/{id}/events?token=$TK" | grep -c "event:done"
# Expected: 1

# 5. 未认证请求
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/api/chat -d '{"message":"hi"}'
# Expected: 401
```

### Final Checklist
- [ ] 5 tables all have user_id dimension
- [ ] Agent goroutine: defer recover() + context propagation
- [ ] Channel send: always select + ctx.Done()
- [ ] Per-iteration timeout (20s) + cancel() called immediately
- [ ] SSE: text/event-stream + X-Accel-Buffering: no + heartbeat
- [ ] pgxpool MaxConns=20, MinConns=5
- [ ] Agent state immutable
- [ ] JWT middleware protects /api/* endpoints
- [ ] No ORM, no langchaingo, no DI framework
- [ ] Zero file storage (pure PG)
- [ ] Goroutine leak test PASS
- [ ] Docker Compose one-command startup
