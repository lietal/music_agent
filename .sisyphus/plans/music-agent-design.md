# Music Agent — 设计文档

> **参考项目**: Musio (musio/) — 架构借鉴，代码重写
> **原则**: 多用户隔离从 Day 1、PostgreSQL 从 Day 1、Agent 自研、工具可 Mock

---

## 1. 项目目标

构建一个**服务端彻底分用户**的音乐推荐搜索 Agent。微信 OAuth 登录，自然语言对话完成音乐搜索、推荐。Agent ReAct 循环 + SSE 流式推送。

### MVP 范围
- Agent 对话链路（用户消息 → 意图分类 → 工具调用 → 回复）
- 微信 OAuth 登录
- 工具层 Mock
- React Web 控制台（登录 + 对话）

---

## 2. 技术栈

### 后端
| 组件 | 选型 |
|------|------|
| 语言 | Go 1.22+ |
| HTTP | chi |
| 数据库 | pgx/v5 + pgxpool |
| 迁移 | golang-migrate |
| OAuth2 | x/oauth2 + golang-jwt |
| SSE | net/http Flusher |
| LLM | 自建 HTTP client (OpenAI-compatible) |
| 配置 | viper |
| 日志 | slog |

### 前端
| 组件 | 选型 |
|------|------|
| 框架 | React 18 + TypeScript |
| 构建 | Vite 5 |
| 路由 | React Router v6 |
| 样式 | Tailwind CSS |
| SSE | EventSource API |

### 基础设施
- PostgreSQL 16 (Docker Compose)
- 参考项目: musio/

---

## 3. 项目结构

```
music-agent/
├── cmd/server/main.go
├── internal/
│   ├── agent/          # Agent 核心（loop, planner, executor）
│   ├── tool/           # Tool 接口 + 注册表 + Mock 工具
│   ├── llm/            # LLM Client 抽象 + OpenAI 实现
│   ├── provider/       # MusicProvider 抽象 + QQ 音乐（后续）
│   ├── auth/           # OAuth Provider 接口 + 微信 + JWT
│   ├── memory/         # 对话历史 + 偏好（PG）
│   ├── event/          # 事件总线 + SSE 发布器
│   ├── api/            # chi 路由 + handler + middleware
│   ├── config/         # viper 配置
│   └── db/             # pgxpool + golang-migrate
├── web/                # React 前端
│   └── src/
│       ├── pages/      # LoginPage, ChatPage, HistoryPage, SettingsPage
│       ├── components/ # AgentMessageList, SongCards, TracePanel, Layout
│       ├── hooks/      # useSSE, useAuth
│       └── api/        # client.ts
├── migrations/         # golang-migrate SQL 文件
├── config.example.toml
├── docker-compose.yml
├── go.mod
└── Makefile
```

---

## 4. 数据库（5 张表，全部含 user_id）

```sql
-- users: OAuth 用户
-- conversations: 对话会话 (user_id FK)
-- messages: 对话消息 (conversation_id FK, role, metadata JSONB)
-- user_preferences: 偏好 (user_id, key PK)
-- user_providers: 音乐源配置 (user_id, provider UNIQUE)
```

---

## 5. Agent 循环

```
用户消息 → POST /api/chat → {runId}
  → goroutine: AgentLoop (maxSteps=5)
    → Planner.Plan() → LLM 意图分类
    → Memory.Load() → 对话摘要 + 偏好
    → ReAct 循环: Executor → Planner.Next() → 重复 或 Compose
  → SSE Events: plan | tool_start | tool_done | delta | done | error
```

---

## 6. 认证流程

```
OAuthProvider 接口: Name() / AuthURL() / Exchange() → UserInfo
微信流程: 授权跳转 → callback → exchange → upsert users → JWT(24h)
Middleware: Authorization: Bearer → 验证 → userId → context.Context
```

---

## 7. API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/auth/{provider}` | OAuth 跳转 |
| GET | `/api/auth/callback/{provider}` | OAuth 回调 |
| GET | `/api/auth/me` | 当前用户 |
| POST | `/api/conversations` | 创建对话 |
| GET | `/api/conversations` | 对话列表 |
| GET | `/api/conversations/{id}` | 对话详情 |
| DELETE | `/api/conversations/{id}` | 删除对话 |
| POST | `/api/chat` | 发送消息 |
| SSE | `/api/chat/{runId}/events` | Agent 事件流 |
| GET | `/api/tools` | 工具列表 |
| GET | `/api/providers` | 音乐源状态 |
| PUT | `/api/providers/{name}/config` | 配置音乐源 |

---

## 8. 前端页面

| 路由 | 页面 | 核心组件 |
|------|------|----------|
| `/login` | 微信扫码登录 | OAuth 跳转 |
| `/chat` | Agent 对话 | AgentMessageList + SSE |
| `/history` | 历史对话 | 列表 + 搜索 |
| `/settings` | 设置 | 音乐源配置 |

---

## 9. MVP 里程碑

1. 项目骨架: Go module + PG + migrate + chi
2. 用户系统: 微信 OAuth + JWT + 5 张表
3. Agent 核心: Tool 接口 + Mock + LLM Client
4. Agent 循环: Planner + Executor + SSE
5. 前端: 登录 + 对话 + SSE 消费
6. 集成: 登录 → 对话 → Agent → Mock 搜索 → 回复
