# 播放功能设计文档

> 日期: 2026-06-19

## 需求摘要

在现有音乐搜索 Agent 中新增完整播放功能：
- 底部迷你播放条 + 可展开播放面板
- QQ 音乐扫码登录，完整播放权限
- 播放队列管理（不含歌单 CRUD）
- 音频获取：优先直连 QQ Music CDN，失败时回退后端代理（混合模式）

## 架构总览

```
┌──────────────────────────────────────────────────────────────┐
│  ChatPage                                                     │
│  ┌──────────────┐  ┌──────────────┐                          │
│  │ AgentMessage │  │ TracePanel   │                          │
│  │ List         │  │ (现有)        │                          │
│  └──────────────┘  └──────────────┘                          │
├──────────────────────────────────────────────────────────────┤
│  PlayerBar (底部固定，迷你播放条)                              │
│  🎵 晴天 - 周杰伦  ───●───────────────── 2:34 / 4:29        │
├──────────────────────────────────────────────────────────────┤
│  PlayerPanel (展开面板，覆盖聊天区下部)                        │
│  ┌──────────────┬──────────────────┬────────────────────────┐│
│  │  正在播放     │     歌词          │    播放队列             ││
│  └──────────────┴──────────────────┴────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

## 混合模式数据流

```
播放请求 → GET /api/player/url/{id}
            ├── 成功 + URL 有效 → <audio src="CDN_URL"> 直连
            └── 失败 / 空 / CORS → fallback:
                    GET /api/player/stream/{id} → 后端代理音频流
```

## 后端设计

### 新增 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/player/url/{songId}` | 获取 CDN 播放 URL |
| `GET` | `/api/player/stream/{songId}` | 后端代理音频流（fallback，支持 Range） |
| `GET` | `/api/player/lyrics/{songId}` | 获取歌词 |
| `POST` | `/api/qqmusic/login/qrcode` | 获取登录二维码 |
| `GET` | `/api/qqmusic/login/status/{key}` | 查询扫码状态 |
| `GET` | `/api/qqmusic/login/status` | 当前登录状态 |

### 新增文件

| 文件 | 说明 |
|------|------|
| `server/internal/tme/login.go` | QQ 音乐登录 API（QR 码、状态轮询、凭证换取） |
| `server/internal/tme/credential_store.go` | 凭证内存 + 文件持久化存储 |
| `server/internal/api/player.go` | 播放相关 Handler（URL、歌词） |
| `server/internal/api/player_stream.go` | 音频代理流 Handler |
| `server/internal/api/qqmusic_login.go` | 登录相关 Handler |

### 修改文件

| 文件 | 说明 |
|------|------|
| `server/internal/api/router.go` | 注册新路由 |
| `server/cmd/server/main.go` | 注入 CredentialStore，初始化登录模块 |

### 端点详情

**`GET /api/player/url/{songId}`**

```
响应: { "song_id": "...", "url": "https://isure...", "expires_in_seconds": 3600, "source": "cdn" }
```

封装现有 `tme.Client.GetSongURL()`。

**`GET /api/player/stream/{songId}`**

- 从 CDN 下载并流式转发，透传 Content-Type
- 支持 Range 请求（用于 seek 拖拽进度条）
- 设置 Cache-Control 头

**`GET /api/player/lyrics/{songId}`**

- 封装现有 `tme.Client.GetLyrics()`（含 DES3 解密 + zlib 解压）
- 返回 `{ song_id, plain_text, synced_text }`

### QQ 音乐登录

登录时序：
```
前端 POST /api/qqmusic/login/qrcode → 获取 { qrcode_url, key }
前端展示二维码，每 2s 轮询 GET /api/qqmusic/login/status/{key}
  → pending → scanned → confirmed（凭证写入 CredentialStore）
前端收到 confirmed，登录完成
```

- `CredentialStore` 启动时从文件恢复凭证
- `tme.Client.SetCredential()` 在每次 API 调用前注入凭证
- 登录后 `GetSongURL` 返回完整歌曲 URL（非预览）

## 前端设计

### 组件树

```
ChatPage.tsx (修改)
├── AgentMessageList.tsx (修改：SongCards 增加交互)
│   └── SongCards.tsx (修改：点击播放 + 加入队列按钮)
├── TracePanel.tsx (不变)
├── PlayerBar.tsx           ← 底部迷你条
└── PlayerPanel.tsx         ← 展开面板
    ├── NowPlaying.tsx      ← 封面 + 播放控制
    ├── LyricsPanel.tsx     ← 同步歌词
    └── QueuePanel.tsx      ← 队列管理
```

### 核心 Hook：`usePlayerStore`

状态：
```typescript
interface PlayerState {
  currentSong: Song | null;
  queue: Song[];
  queueIndex: number;
  isPlaying: boolean;
  currentTime: number;
  duration: number;
  volume: number;
  playbackMode: 'sequential' | 'repeat_one' | 'repeat_all' | 'shuffle';
  urlSource: 'cdn' | 'proxy' | null;
  urlExpiresAt: number | null;
  lyrics: LyricLine[] | null;
  activeLyricIndex: number;
  panelOpen: boolean;
}
```

持久化：localStorage（queue, queueIndex, playbackMode, volume）。

### 混合模式 URL 获取

```
ensurePlayableURL(song):
  1. 检查缓存（未过期 CDN URL）→ 有则直接用
  2. 调用 GET /api/player/url/{id}
     ├── 成功 → 缓存，标记 source: "cdn"
     └── 失败 → 切 source: "proxy"，使用 /api/player/stream/{id}
  3. 设置 <audio>.src
  4. 监听 error 事件 → MEDIA_ERR_SRC_NOT_SUPPORTED / MEDIA_ERR_NETWORK → 切 proxy 重试
  5. URL 剩余 < 60s → 后台静默刷新
```

### QQ 音乐登录 Hook：`useQQMusicLogin`

```typescript
{
  loginStatus: 'idle' | 'loading' | 'pending_scan' | 'scanned' | 'confirmed' | 'expired' | 'error';
  qrcodeUrl: string | null;
  userName: string | null;
  startLogin(): Promise<void>;
  logout(): Promise<void>;
  isLoggedIn: boolean;
}
```

### 新增 / 修改文件（前端）

| 文件 | 操作 | 说明 |
|------|------|------|
| `web/src/types.ts` | ✏️ | 新增 PlayerState, LyricLine 等类型 |
| `web/src/hooks/usePlayerStore.ts` | 🆕 | 播放器引擎 |
| `web/src/hooks/useQQMusicLogin.ts` | 🆕 | QQ 音乐登录状态 |
| `web/src/api/player.ts` | 🆕 | 播放器 API 客户端 |
| `web/src/components/PlayerBar.tsx` | 🆕 | 底部播放条 |
| `web/src/components/PlayerPanel.tsx` | 🆕 | 展开面板 |
| `web/src/components/NowPlaying.tsx` | 🆕 | 正在播放 |
| `web/src/components/LyricsPanel.tsx` | 🆕 | 歌词 |
| `web/src/components/QueuePanel.tsx` | 🆕 | 队列 |
| `web/src/components/SongCards.tsx` | ✏️ | 增加点击播放/加队列 |
| `web/src/pages/ChatPage.tsx` | ✏️ | 嵌入 PlayerBar + PlayerPanel |
| `web/src/pages/LoginPage.tsx` | ✏️ | 集成 QQ 音乐登录 |

## 关键数据流

```
用户点击歌曲卡片
  → SongCards.onPlay(song)
  → usePlayerStore.play(song)
  → ensurePlayableURL(song)
      → 返回 CDN URL 或 proxy URL
  → <audio>.src = url
  → <audio>.play()
  → PlayerBar 显示当前歌曲
  → 异步获取歌词：/api/player/lyrics/{id}
```

```
用户点击 "+ 加入队列"
  → SongCards.onAddToQueue(song)
  → usePlayerStore.addToQueue(song)
  → queue.push(song)
  → localStorage 更新
  → QueuePanel 刷新
```
