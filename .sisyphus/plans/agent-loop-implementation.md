# Agent Loop 重构 — 实施计划

> **For Claude:** Use subagent-driven-development to execute task-by-task.

**Goal:** 将 LLM Planner 替换为 Intent Router + ReAct Pipeline，支持多意图复合查询和多步推理

**Design:** `.sisyphus/plans/agent-loop-redesign.md`

**Tech Stack:** Go 1.25, chi router, log/slog, pgx, SSE streaming

---

## TL;DR

> 新建 4 个文件 (`intent.go`, `react.go`, `pipeline.go`, `prompts.go`)，修改 2 个文件 (`config.go`, `main.go`)，新增 `play_playlist` 工具，旧 AgentLoop 保留不变。三组件 Prompt 独立配置。

---

## Context

### 当前架构
```
LLM.Plan() → ToolCalls → Executor.Execute() → LLM.Next() → AnswerGenerator
```

### 目标架构
```
Intent Router → Pipeline(Intent 1 → ReAct) → Pipeline(Intent 2 → ReAct) → Answer Generator
```

### 5 意图
| Intent | 工具 |
|--------|------|
| `search_music` | `search_songs` |
| `recommend_music` | `recommend_songs` |
| `playlist_write` | `create_playlist`, `add_to_playlist`, `remove_song`, `rename_playlist` |
| `playlist_read` | `list_playlists`, `get_playlist`, `play_playlist` |
| `chat` | 无 |

---

## Work Objectives

### Core Objective
实现多意图路由 + ReAct Pipeline，不改动旧 AgentLoop

### Concrete Deliverables
- `server/internal/agent/intent.go` — Intent Router + 类型
- `server/internal/agent/react.go` — ReAct Loop
- `server/internal/agent/pipeline.go` — Pipeline Executor
- `server/internal/agent/prompts.go` — Prompt 常量+默认值
- `server/internal/tool/play_playlist.go` — play_playlist 工具
- `server/internal/config/config.go` — PromptsConfig（修改）
- `server/cmd/server/main.go` — 注册新 Pipeline（修改）

### Must NOT Have
- 不删除或修改旧 AgentLoop / loop.go / planner_llm.go
- 不修改前端 SSE 处理逻辑（先兼容）
- 不改动 executor.go / state.go（复用）

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: TDD (test-first)
- **Framework**: `go test`

### QA Policy
每任务包含 Go test 验证 + 手动 curl 端到端检查

---

## Execution Strategy

```
Wave 1: foundation (并行)
├── T1: prompts.go — Prompt 常量 + 默认模板
├── T2: intent.go — Intent 类型 + IntentRouter 接口
├── T3: config.go — PromptsConfig 结构 + config.toml
└── T4: play_playlist.go — play_playlist 工具

Wave 2: core logic (依赖 Wave 1)
├── T5: intent_llm.go — LLM Intent Router 实现
├── T6: react.go — ReAct Loop
└── T7: pipeline.go — Pipeline Executor

Wave 3: integration (依赖 Wave 2)
├── T8: main.go — 注册 AgentPipeline
└── T9: 端到端测试 — curl 验证多意图查询
```

---

## TODOs

- [x] 1. Prompt 常量 + 默认模板

  **What to do**:
  - 新建 `server/internal/agent/prompts.go`
  - 定义 `Prompts` 结构体，包含三个字段：`IntentRouter`, `ReactThink`, `AnswerGen`
  - 提供 `DefaultPrompts()` 返回默认 prompt 模板
  - 模板中包含 `{tools}`, `{observations}`, `{intent_results}` 占位符，运行时替换

  **Must NOT do**:
  - 不要把 prompt 写死在 intent.go 或 react.go 里

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 1, can run with T2-T4

  **Acceptance Criteria**:
  - [ ] `go build ./internal/agent/...` → PASS
  - [ ] Test: `DefaultPrompts()` 返回非空三字段

  **QA Scenarios**:
  ```
  Scenario: Default prompts are non-empty
    Tool: Bash
    Steps:
      1. go test -run TestDefaultPrompts ./internal/agent/
    Expected: PASS, all three prompt fields are non-empty strings
    Evidence: .sisyphus/evidence/task-1-prompts.txt
  ```

  **Commit**: YES
  - Message: `feat(agent): add prompt constants and defaults`
  - Files: `server/internal/agent/prompts.go`

- [x] 2. Intent 类型 + IntentRouter 接口

  **What to do**:
  - 新建 `server/internal/agent/intent.go`
  - 定义 `Intent` 结构体: `{ Type, Query, Params }`
  - 定义 `IntentRouter` 接口: `Route(ctx, message string) ([]Intent, error)`
  - 定义 `IntentResult` 结构体: `{ Intent, Output, Error }`

  **Must NOT do**:
  - 不实现具体 LLM Router（那是 T5）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 1, can run with T1, T3, T4

  **Acceptance Criteria**:
  - [ ] `go build ./internal/agent/...` → PASS
  - [ ] Test: `Intent` JSON 序列化/反序列化正确

  **QA Scenarios**:
  ```
  Scenario: Intent JSON round-trip
    Tool: Bash
    Steps:
      1. go test -run TestIntentJSON ./internal/agent/
    Expected: PASS, marshal/unmarshal preserves all fields
    Evidence: .sisyphus/evidence/task-2-intent.txt
  ```

  **Commit**: YES
  - Message: `feat(agent): add Intent type and IntentRouter interface`
  - Files: `server/internal/agent/intent.go`

- [x] 3. PromptsConfig 配置结构

  **What to do**:
  - 修改 `server/internal/config/config.go`
  - 新增 `PromptsConfig` 结构体
  - 添加到 `Config` 作为新字段 `Prompts PromptsConfig`
  - 更新 `config.example.toml` 增加 `[prompts]` 段

  **Must NOT do**:
  - 不改动现有 Config 字段名

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 1, can run with T1, T2, T4

  **Acceptance Criteria**:
  - [ ] `go build ./...` → PASS
  - [ ] Test: 配置文件解析 `[prompts]` 段正确

  **QA Scenarios**:
  ```
  Scenario: TOML config parses prompts section
    Tool: Bash
    Steps:
      1. go test -run TestConfig ./internal/config/
    Expected: PASS, PromptsConfig fields populated from config.toml
    Evidence: .sisyphus/evidence/task-3-config.txt
  ```

  **Commit**: YES
  - Message: `feat(config): add PromptsConfig for agent prompt customization`
  - Files: `server/internal/config/config.go`, `config.example.toml`

- [x] 4. play_playlist 工具

  **What to do**:
  - 新建 `server/internal/tool/play_playlist.go`
  - 实现 `playPlaylistTool`: Name="play_playlist", Desc="Play songs from a playlist"
  - Execute: 查询 playlist_songs 表，返回 `{songs: [{id, title, artist, coverUrl}]}` 格式
  - 在 `playlist_tools.go` 的 `NewPlaylistTools()` 注册此工具
  - 参考 `getPlaylistTool` 实现，但返回格式对齐 SongCard

  **Must NOT do**:
  - 不修改前端

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 1, can run with T1-T3

  **Acceptance Criteria**:
  - [ ] `go test ./internal/tool/...` → PASS
  - [ ] Test: play_playlist 返回 songs 数组，字段齐全

  **QA Scenarios**:
  ```
  Scenario: play_playlist returns songs
    Tool: Bash
    Steps:
      1. go test -run TestPlayPlaylist ./internal/tool/
    Expected: PASS, result contains songs array with id/title/artist/coverUrl
    Evidence: .sisyphus/evidence/task-4-playlist.txt
  ```

  **Commit**: YES
  - Message: `feat(tool): add play_playlist tool`
  - Files: `server/internal/tool/play_playlist.go`, `server/internal/tool/playlist_tools.go`

---

## Final Verification Wave

- [x] 5. LLM Intent Router 实现

  **What to do**:
  - 新建 `server/internal/agent/intent_llm.go`
  - 实现 `llmIntentRouter` 结构体，满足 `IntentRouter` 接口
  - `Route()` 调用 LLM，使用 `prompts.IntentRouter` 模板
  - LLM 返回 JSON 数组 `[{type, query, params}]`
  - 解析 JSON 返回 `[]Intent`
  - 错误处理：LLM 失败 → 返回 `[{type: "chat", query: message}]` fallback

  **Must NOT do**:
  - 不修改 intent.go 接口定义

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 2, blocked by T1+T2

  **References**:
  - `server/internal/agent/planner_llm.go` — 参考 LLM 调用模式
  - `server/internal/agent/intent.go` — Intent 类型定义
  - `server/internal/agent/prompts.go` — Prompt 模板

  **Acceptance Criteria**:
  - [ ] Test: `Route("找周杰伦的歌")` → `[{type:search_music, query:"周杰伦"}]`
  - [ ] Test: `Route("把晴天加入我的歌单")` → `[{type:playlist_write, ...}]`
  - [ ] Test: `Route("找周杰伦的歌，加入通勤歌单")` → `[{search_music...}, {playlist_write...}]`
  - [ ] Test: `Route("你好")` → `[{type:chat}]`

  **QA Scenarios**:
  ```
  Scenario: Single intent - search
    Tool: Bash
    Steps:
      1. go test -run TestIntentRouter_Search ./internal/agent/
    Expected: PASS, returns search_music intent
    Evidence: .sisyphus/evidence/task-5-intent-search.txt

  Scenario: Multi intent - search + playlist
    Tool: Bash
    Steps:
      1. go test -run TestIntentRouter_Multi ./internal/agent/
    Expected: PASS, returns 2 intents
    Evidence: .sisyphus/evidence/task-5-intent-multi.txt
  ```

  **Commit**: YES
  - Message: `feat(agent): implement LLM intent router`
  - Files: `server/internal/agent/intent_llm.go`

- [x] 6. ReAct Loop

  **What to do**:
  - 新建 `server/internal/agent/react.go`
  - 实现 `ReActLoop` 结构体
  - `Run(ctx, intent, tools, prompt, maxSteps) → chan event.Event`
  - 每步循环: Think (LLM 用 prompts.ReactThink) → Act (executor.Execute) → Observe → Next
  - 输出标准化为 `ToolResult`，存储在 context 中
  - 达到 maxSteps 或 LLM 返回 FINAL_ANSWER 时终止

  **Must NOT do**:
  - 不重复实现 executor 逻辑，复用 `executor.go`
  - 不发送 plan 事件（那是 Pipeline 的职责）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 2, blocked by T1+T2. Can run parallel with T5+T7

  **References**:
  - `server/internal/agent/loop.go` — 参考 Run 的 goroutine+channel 模式
  - `server/internal/agent/executor.go` — 复用 Execute
  - `server/internal/agent/prompts.go` — ReAct prompt 模板

  **Acceptance Criteria**:
  - [ ] Test: 单步 search_songs → 返回 songs
  - [ ] Test: 两步 search → FINAL_ANSWER
  - [ ] Test: 空工具列表 → 直接 FINAL_ANSWER
  - [ ] Test: maxSteps 达到 → 终止

  **QA Scenarios**:
  ```
  Scenario: React loop single step
    Tool: Bash
    Steps:
      1. go test -run TestReAct_SingleStep ./internal/agent/
    Expected: PASS, one tool execution, FINAL_ANSWER
    Evidence: .sisyphus/evidence/task-6-react.txt
  ```

  **Commit**: YES
  - Message: `feat(agent): implement ReAct loop`
  - Files: `server/internal/agent/react.go`

- [x] 7. Pipeline Executor

  **What to do**:
  - 新建 `server/internal/agent/pipeline.go`
  - 实现 `AgentPipeline` 结构体
  - `Run(ctx, message) → chan event.Event`:
    1. 调用 IntentRouter → 得到 `[]Intent`
    2. 发送 `plan` 事件 `{intents: [...]}`
    3. 对每个 intent:
       a. 发送 `intent_start` 事件
       b. 根据 intent.Type 选择工具子集
       c. 调用 ReAct Loop
       d. 收集 output 存入 PipelineContext
       e. 发送 `intent_done` 事件
    4. 调用 AnswerGenerator
    5. 发送 `done` 事件

  **Must NOT do**:
  - 不依赖旧 AgentLoop 的实现
  - 每个 intent 的工具子集从 `intentTools` map 查找

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 2, blocked by T5+T6

  **References**:
  - `server/internal/agent/loop.go` — SSE 事件发送模式
  - `server/internal/agent/intent.go` — Intent + IntentResult 类型
  - `server/internal/agent/react.go` — ReAct Loop

  **Acceptance Criteria**:
  - [ ] Test: 单个 search intent → 发送 plan/intent_start/tool_start/tool_done/delta/intent_done/done
  - [ ] Test: 两个 intent pipeline → 顺序执行，第二步收到第一步 output
  - [ ] Test: PipelineContext 正确传递前一步结果

  **QA Scenarios**:
  ```
  Scenario: Pipeline with search intent
    Tool: Bash
    Steps:
      1. go test -run TestPipeline_Search ./internal/agent/
    Expected: PASS, SSE events emitted in correct order
    Evidence: .sisyphus/evidence/task-7-pipeline.txt
  ```

  **Commit**: YES
  - Message: `feat(agent): implement intent pipeline executor`
  - Files: `server/internal/agent/pipeline.go`

- [x] 8. main.go 集成 Pipeline

  **What to do**:
  - 修改 `server/cmd/server/main.go`
  - 初始化 PromptsConfig（从 config 加载或使用默认值）
  - 创建 IntentRouter（llmIntentRouter）
  - 创建 AgentPipeline
  - 在 Handler 上新增 `SetPipeline()` 方法，与旧 `SetAgent()` 并存
  - handler 优先使用 Pipeline（如果设置），否则 fallback 旧 AgentLoop

  **Must NOT do**:
  - 不删除 `SetAgent()` 或旧 AgentLoop
  - 不修改 chat.go 的 createChatHandler / chatEventsHandler

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: `[]`

  **Parallelization**: Wave 3, blocked by T7

  **References**:
  - `server/cmd/server/main.go:121-129` — 旧 AgentLoop 初始化位置
  - `server/internal/api/handler.go` — Handler.SetAgent()

  **Acceptance Criteria**:
  - [ ] `go build ./...` → PASS
  - [ ] 服务启动 → Pipeline 初始化成功
  - [ ] 旧 AgentLoop 功能不变（向后兼容）

  **QA Scenarios**:
  ```
  Scenario: Service starts with pipeline
    Tool: Bash
    Steps:
      1. docker compose up -d --build
      2. curl http://localhost:8080/health
    Expected: 200, server running with pipeline initialized
    Evidence: .sisyphus/evidence/task-8-integration.txt
  ```

  **Commit**: YES
  - Message: `feat(agent): wire pipeline into main, coexist with legacy AgentLoop`
  - Files: `server/cmd/server/main.go`, `server/internal/api/handler.go`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

- [x] F1. **Go Unit Tests** — `quick`
  Run `go test ./internal/agent/... ./internal/tool/... ./internal/config/...`
  Output: all PASS, coverage ≥ 80% for new files

- [x] F2. **Go Full Suite** — `quick`
  Run `go test ./internal/...`
  Output: all packages PASS, no regressions

- [x] F3. **Curl E2E** — `quick`
  ```bash
  TOKEN=$(generate JWT)
  curl -X POST http://localhost:8080/api/chat -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"message":"帮我找一首周杰伦的歌，加入通勤歌单"}'
  ```
  Expected: 201, SSE stream with plan→intent_start→tool_start→tool_done→intent_done→done events

- [x] F4. **Legacy Compatibility** — `quick`
  Old AgentLoop routes still work: `POST /api/chat` with `message: "周杰伦的歌"`
  Expected: search_songs → agent response

---

## Commit Strategy

- **Wave 1**: `feat(agent): add intent types, prompts, play_playlist tool`
- **Wave 2**: `feat(agent): implement intent router, react loop, pipeline`
- **Wave 3**: `feat(agent): integrate pipeline into main`

## Success Criteria

- [ ] All "Acceptance Criteria" above met
- [ ] 5 intents routed correctly
- [ ] Multi-intent pipeline: search → add_to_playlist
- [ ] ReAct Loop supports multi-step
- [ ] Prompts independently configurable
- [ ] Legacy AgentLoop unaffected
- [ ] All tests pass
- [ ] F2. `go test ./internal/...` → 全部 PASS
- [ ] F3. `curl` 测试: "帮我找周杰伦的歌，加入通勤歌单"
- [ ] F4. 旧 AgentLoop 功能不受影响

---

## Commit Strategy

- Wave 1: `feat(agent): add intent types, prompts config, play_playlist tool`
- Wave 2: `feat(agent): implement intent router, react loop, pipeline`
- Wave 3: `feat(agent): wire pipeline into main`

## Success Criteria

- [ ] Intent Router 返回多意图 JSON 数组
- [ ] ReAct Loop 支持 Think→Act→Observe→Next
- [ ] Pipeline 顺序执行多意图，上游输出传到下游
- [ ] play_playlist 返回 SongCard 格式歌曲列表
- [ ] 三组件 prompt 独立配置
- [ ] 旧 AgentLoop 所有测试仍然通过
