# Agent Loop 重构：多意图路由 + ReAct Pipeline

## TL;DR

> **目标**：将当前单步 LLM planner 替换为 Intent Router（多意图分类）+ Sequential Pipeline（逐个执行）+ ReAct Loop（每意图多步推理）
>
> **核心变化**：一句话支持多意图，前一步的输出流向后一步

## 现状

```
用户消息 → LLM.Plan() → 工具调用 → Executor.Execute() → 结果
         ↑                                            ↓
         └─────── Planner.Next() ← 继续/结束 ─────────┘
                      ↓ 结束
         AnswerGenerator.GenerateAnswer() → Done
```

问题：
- LLM 一次性决定所有工具调用，容易选错（如 search_songs 而非 add_to_playlist）
- 仅支持单轮 search → answer，不支持多步推理
- 不支持多意图复合查询

## 新设计

```
用户: "帮我找一首最火的周杰伦的歌，加入到我的通勤歌单"

  ┌──────────────────┐
  │  Intent Router   │ → [
  │ (LLM 解析多意图)  │     {type: search_music, query: "周杰伦 最火"},
  └────────┬─────────┘     {type: add_to_playlist, playlist: "通勤歌单"}
           │              ]
           ▼
  ┌──────────────────────────────────────────────────┐
  │           Sequential Intent Pipeline             │
  │                                                  │
  │  Intent 1: search_music                          │
  │  ┌─────────────────────────┐                     │
  │  │       ReAct Loop        │                     │
  │  │  Think → Act → Observe  │                     │
  │  │    ↓                    │                     │
  │  │  Next? → FINAL_ANSWER   │                     │
  │  └──────────┬──────────────┘                     │
  │             ↓ output: {song, id}                 │
  │                                                  │
  │  Intent 2: add_to_playlist                       │
  │  ┌─────────────────────────┐                     │
  │  │       ReAct Loop        │                     │
  │  │  输入: prev output       │                     │
  │  │  Think → Act → Observe  │                     │
  │  └─────────────────────────┘                     │
  └──────────────────┬───────────────────────────────┘
                     ▼
             Generate Final Answer
```

### 组件职责

#### 1. Intent Router

- 输入：用户消息
- 输出：`[]Intent{ Type, Query, Params }`
- 实现：LLM 调用，prompt 要求返回 JSON 数组
- Intent 类型：`search_music`, `recommend_music`, `manage_playlist`, `chat`

```go
type Intent struct {
    Type   string         `json:"type"`   // search_music | recommend_music | playlist_write | playlist_read | chat
    Query  string         `json:"query"`  // 搜索关键词（如有）
    Params map[string]any `json:"params"` // 额外参数（playlist 名等）
}
```

#### 2. Pipeline Executor

- 输入：`[]Intent`
- 逐个执行，前一步的 ReAct output 传递给下一步作为 context
- 每个 Intent 跑 ReAct Loop，输出标准化为 `PipelineContext`

#### 3. ReAct Loop

- 每个 Intent 内部的多步推理：
  - Think：LLM 分析当前状态，决定下一步 action
  - Act：执行工具调用
  - Observe：记录结果
  - Next：LLM 判断继续还是 FINAL_ANSWER
- 最多 N 步（可配置，默认 5）
- 与旧 loop 的区别：命名明确 Think/Act/Observe 阶段，非单一 Plan

#### 4. Answer Generator（保持）

- 所有 Intent 执行完后，LLM 基于全部 observations 生成最终回复

### Intent → 可用工具映射

| Intent Type | 触发场景 | 可用工具 |
|-------------|---------|---------|
| `search_music` | 找歌、搜歌手、歌名 | `search_songs` |
| `recommend_music` | 主动推荐、"推荐给我" | `recommend_songs` |
| `playlist_write` | 创建歌单、加入歌曲、删除歌曲、改名 | `create_playlist`, `add_to_playlist`, `remove_song`, `rename_playlist` |
| `playlist_read` | 查看歌单列表、查看歌单内容、播放歌单 | `list_playlists`, `get_playlist`, `play_playlist` |
| `chat` | 闲聊、问候 | 无工具，直接回答 |

### Pipeline Context（意图间传数据）

```go
type PipelineContext struct {
    UserMessage string
    UserID      string
    Intents     []Intent
    Results     []IntentResult  // 已完成的 intent 结果
    Observations []Observation  // 全局观察
}

type IntentResult struct {
    Intent Intent
    Output any       // ReAct 输出（如搜索到的歌曲列表）
    Error  string
}
```

### SSE Event 流

```
event: plan        → { "intents": [...] }
event: intent_start → { "intent": { "type": "search_music", ... } }
event: tool_start   → { "toolName": "search_songs", "args": {...} }
event: tool_done    → { "result": {...} }
event: delta        → { "message": "..." }
event: intent_done  → { "intent": {...}, "output": {...} }
event: done
```

### Prompt 独立配置

每个 LLM 调用点使用独立 prompt，从配置文件加载，允许精调：

| 组件 | 配置 key | 职责 |
|------|---------|------|
| Intent Router | `prompts.intent_router` | 解析多意图，返回 `[{type, query, params}]` |
| ReAct Think | `prompts.react_think` | 每步决策：分析 state → 选择工具/参数 → next |
| Answer Generator | `prompts.answer_gen` | 汇总所有 observations 生成最终回复 |

```toml
[prompts]
intent_router = """
You are an intent classifier. Parse the user message into one or more intents.
Available intents: search_music, recommend_music, playlist_write, playlist_read, chat.
Return a JSON array: [{"type":"...","query":"...","params":{}}]
"""

react_think = """
You are a music agent. Based on the current state and observations, decide the next action.
Available tools: {tools}
Current observations: {observations}
Return JSON: {"toolName":"...","args":{...},"next":"CONTINUE|FINAL_ANSWER"}
"""

answer_gen = """
You are a music assistant. Generate a friendly response based on the conversation.
Context: {intent_results}
"""
```

**好处**：
- 调 prompt 不需要改代码、重新编译
- 不同环境（dev/prod）可以用不同 prompt
- 快速 A/B 测试 prompt 效果

### 文件结构

```
server/internal/agent/
├── loop.go          → 旧 loop，保留兼容
├── pipeline.go      → 新 Pipeline Executor
├── intent.go        → Intent Router + Intent 类型
├── react.go         → ReAct Loop
├── planner.go       → 旧 Planner 接口，保留
├── planner_llm.go   → 旧 LLM Planner，保留
├── state.go         → LoopState + PipelineContext
├── executor.go      → 旧 Executor，保留复用
└── answer.go        → Answer Generator（提取）

server/internal/config/
└── config.go        → 新增 PromptsConfig 结构
```

### 分阶段实施

1. 新建 `intent.go` + `react.go` + `pipeline.go`
2. 在 `main.go` 新增 `AgentPipeline`，与旧 AgentLoop 并存
3. 前端适配新的 SSE event 类型
4. 旧 AgentLoop 逐步废弃

## 成功标准

- [ ] Intent Router 正确解析多意图复合查询
- [ ] "帮我找一首最火的周杰伦的歌，加入到我的通勤歌单" 正确执行两个 intent
- [ ] ReAct Loop 支持多步推理
- [ ] 新增 `play_playlist` 工具
- [ ] Intent Router / ReAct Think / Answer Generator 各用独立 prompt 配置
- [ ] 所有现有测试通过
- [ ] SSE 事件流兼容前端
