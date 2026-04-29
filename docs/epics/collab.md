# Epic: Collab（实时协同编辑 MVP）

> **状态**: 设计冻结，可下发 Codex · 2026-04-27
> **上游**: [`docs/requirements.md`](../requirements.md) v1.0 §4.3 / §6.6~§6.7 / §7.4~§7.5 / §8.2 / D-06 / D-09a / D-10a / D-10b / D-13
> **视觉**: [`docs/ui/design-system.md`](../ui/design-system.md) v1.0
> **范围**: TipTap 编辑器集成、Y.js CRDT 协同、WebSocket 协议实装、doc_states/doc_updates 读写、Compact 调度、Awareness 光标、权限级别约束（readable 只读）、20 人并发限制
> **不包含**: 历史快照/回溯（M6）、图床上传（M7）、通知推送（M8）、搜索（M9）、CodeMirror 源码视图（远期）

本 Epic 产出的是"两个浏览器同时打开同一文档，能看到对方光标 + 实时同步编辑，断线重连不丢字"的完整协同闭环：
- 创建文档 → 点击打开 → TipTap 编辑器加载
- WS 握手 + 鉴权 + 权限校验 → 加入文档 Hub
- Y.js sync 协议：sync-step1 → sync-step2 → 增量 update 广播
- Awareness：光标位置 + 用户名 + 颜色实时同步
- Readable 用户：可观察、接收 awareness，但发送 update 被服务端丢弃
- 20 人并发上限：超限返回 `join_rejected`
- 增量 update 落库 → Compact 异步合并 → doc_states 更新
- 断线重连：自动重连 + state vector 差量同步

---

## 1. 已有基础设施（M1~M4 产出）

### 1.1 数据库已就绪

| 表 | 迁移 | 状态 |
|---|---|---|
| `doc_states` | `0004_docs_snapshots.up.sql` | ✅ node_id PK / ydoc_state bytea / version / markdown_plain / last_compacted_at |
| `doc_updates` | `0004_docs_snapshots.up.sql` | ✅ bigserial PK / node_id / update bytea / client_id / created_at / 索引 (node_id, id) |
| `doc_snapshots` | `0004_docs_snapshots.up.sql` | ✅ 含 ydoc_state / trigger_reason / created_at（M6 使用，本 Epic 不涉及） |
| `nodes` | `0003_nodes_permissions.up.sql` | ✅ 含 kind='doc' |
| `node_permissions` | `0003_nodes_permissions.up.sql` | ✅ 权限解析依赖 |

### 1.2 后端 WS 骨架已就绪

| 模块 | 文件 | 状态 | 说明 |
|---|---|---|---|
| Hub | `internal/ws/hub.go` | ✅ | 20 人上限、register/unregister/forceKick/inbound channel、idle 60s 自动释放 |
| HubManager | `internal/ws/manager.go` | ✅ | GetOrCreate / Release / DisconnectUser |
| Handler | `internal/ws/handler.go` | ✅ | WS 握手 + JWT 解析 + subprotocol `access.<jwt>` + readPump/writePump |
| Router | `internal/httpserver/router.go` | ✅ | `api.GET("/ws", ws.Handler(...))` 已注册 |

**需要改造的点**：
- `hub.go` inbound 分支当前返回 `not_implemented` → 需实装 Y.js 协议路由
- `handler.go` 中 `client.Level` 硬编码为 `"readable"` → 需调用 PermissionService 解析真实权限
- Hub 缺少对 DocStateRepo/DocUpdateRepo 的依赖注入 → 需要重构构造函数
- Hub 缺少 sync-step1 初始化逻辑（首次连接发送 doc state）

### 1.3 前端已就绪

| 模块 | 文件 | 状态 |
|---|---|---|
| DocPage | `pages/DocPage.tsx` | 🟡 占位（仅 `<h1>Doc {id}</h1>`） |
| API Client | `api/client.ts` | ✅ `request<T>()` + auto-refresh |
| AuthContext | `auth/AuthContext.tsx` | ✅ 提供 accessToken |
| MainShell | `layouts/MainShell.tsx` | ✅ 侧栏目录树 + 主区域 Outlet |
| Tokens | `styles/tokens.css` | ✅ 设计系统 token 全量 |

---

## 2. 协同架构总览

```
┌─────────────────────────────────────────────────────────────────┐
│  Browser A                      Browser B                       │
│  ┌──────────┐                  ┌──────────┐                     │
│  │ TipTap   │                  │ TipTap   │                     │
│  │ Editor   │                  │ Editor   │                     │
│  └────┬─────┘                  └────┬─────┘                     │
│       │ y-prosemirror                │ y-prosemirror             │
│  ┌────┴─────┐                  ┌────┴─────┐                     │
│  │ Y.Doc    │                  │ Y.Doc    │                     │
│  └────┬─────┘                  └────┬─────┘                     │
│       │ FpgWsProvider                │ FpgWsProvider             │
│       │ WebSocket                    │ WebSocket                 │
└───────┼──────────────────────────────┼──────────────────────────┘
        │                              │
        ▼                              ▼
┌───────────────────────────────────────────────────────────────┐
│  Go Backend                                                    │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ WS Handler: JWT 校验 + PermissionService.Resolve       │    │
│  └────────────────────┬───────────────────────────────────┘    │
│                       ▼                                        │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ Hub (per document)                                      │    │
│  │ - clients map[string]*Client                            │    │
│  │ - inbound → route by msg.type                           │    │
│  │ - sync_step1: 发送 doc_states.ydoc_state                │    │
│  │ - sync_step2/update: 广播 + 落库 doc_updates            │    │
│  │ - awareness: 仅转发，不落库                              │    │
│  │ - readable client → update 丢弃 + log                   │    │
│  └────────────────────────────────────────────────────────┘    │
│                       │                                        │
│  ┌────────────────────┴───────────────────────────────────┐    │
│  │ Compact Scheduler (后台 goroutine)                      │    │
│  │ - 每 60s 扫描：count(doc_updates) ≥ 200                 │    │
│  │   或 last_compacted_at < now()-30min 且有 pending        │    │
│  │ - merge updates → doc_states.ydoc_state                 │    │
│  │ - 抽取纯文本 → markdown_plain（给搜索用）               │    │
│  └────────────────────────────────────────────────────────┘    │
│                       │                                        │
│  ┌────────────────────┴───────────────────────────────────┐    │
│  │ PostgreSQL                                              │    │
│  │ doc_states ← compact 后的完整 Y.Doc                     │    │
│  │ doc_updates ← 每条增量 update                           │    │
│  └────────────────────────────────────────────────────────┘    │
└───────────────────────────────────────────────────────────────┘
```

---

## 3. WebSocket 消息协议

### 3.1 消息信封

所有 WS 消息统一 JSON 信封：

```json
{ "type": "<kind>", "payload": "<base64 encoded bytes | json object>" }
```

**注意**：Y.js binary data（sync/update/awareness）使用 base64 编码传输，不使用 binary frame。原因：JSON 信封统一解析逻辑、便于日志记录、与其他消息类型（join_rejected/reload/notify）保持一致。

### 3.2 消息类型一览

| 方向 | `type` | `payload` 格式 | 权限要求 | 说明 |
|---|---|---|---|---|
| S→C | `sync_step1` | `{ "stateVector": "<base64>" }` | — | 连接建立后服务端立即发送，包含当前文档的 Y.js state vector |
| C→S | `sync_step2` | `{ "update": "<base64>" }` | ≥ edit | 客户端收到 sync_step1 后，计算差量 update 发回 |
| C→S | `update` | `{ "update": "<base64>" }` | ≥ edit | 用户编辑产生的增量 update |
| S→C | `update` | `{ "update": "<base64>" }` | — | 服务端广播他人的 update |
| C→S | `awareness` | `{ "data": "<base64>" }` | any | 光标/选区/用户信息 |
| S→C | `awareness` | `{ "data": "<base64>" }` | — | 广播他人的 awareness |
| S→C | `join_rejected` | `{ "reason": "doc_full" \| "permission_denied" }` | — | 加入被拒 |
| S→C | `reload` | `{}` | — | 回溯后要求重置 Y.Doc（M6 实装） |
| S→C | `force_logout` | `{}` | — | Manager 强制下线（已实现） |
| S→C | `peer_join` | `{ "userId": "<uuid>", "displayName": "xxx", "level": "edit" }` | — | 新用户加入通知 |
| S→C | `peer_leave` | `{ "userId": "<uuid>" }` | — | 用户离开通知 |
| S→C | `peers` | `[{ "userId", "displayName", "level" }, ...]` | — | 连接建立后发送当前在线列表 |

### 3.3 连接生命周期

```
Client                              Server
  │                                    │
  │── WS Upgrade ──────────────────────>│  JWT 校验 + tv 验证
  │                                    │  ResolvePermission(user, node)
  │                                    │  if level=="none" → close 4403
  │                                    │  if hub.size >= 20 → join_rejected + close
  │                                    │  hub.register(client)
  │<──── peers ────────────────────────│  当前在线列表
  │<──── sync_step1 ───────────────────│  state vector from doc_states
  │                                    │  (如果 doc_states 无记录 → 空 Y.Doc)
  │──── sync_step2 ────────────────────>│  客户端差量 → 广播 + 落库
  │                                    │  broadcast peer_join to others
  │                                    │
  │ ─ ─ 正常编辑循环 ─ ─ ─ ─ ─ ─ ─ ─ │
  │──── update ─────────────────────── >│  if level ≥ edit → 广播 + 落库
  │                                    │  if level == readable → 丢弃 + log
  │<──── update ───────────────────────│  他人编辑
  │──── awareness ─────────────────────>│  转发（不落库）
  │<──── awareness ────────────────────│  他人光标
  │                                    │
  │ ─ ─ 断连 ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│
  │                                    │  hub.unregister
  │                                    │  broadcast peer_leave to others
  │                                    │  if hub.size==0 → 60s idle → release
```

---

## 4. 后端详细设计

### 4.1 新增 Repository

#### 4.1.1 DocStateRepo

文件：`backend/internal/repository/doc_state_repo.go`

```go
type DocStateRepo interface {
    // GetOrInit 获取文档状态；如果不存在则插入空 Y.Doc 初始状态并返回
    GetOrInit(ctx context.Context, nodeID uuid.UUID) (*models.DocState, error)
    // UpdateAfterCompact 合并后更新 ydoc_state + markdown_plain + version++
    UpdateAfterCompact(ctx context.Context, nodeID uuid.UUID, ydocState []byte, markdownPlain string) error
}
```

- `GetOrInit`：`INSERT INTO doc_states (node_id, ydoc_state) VALUES ($1, $2) ON CONFLICT (node_id) DO NOTHING; SELECT * FROM doc_states WHERE node_id = $1`
- 空 Y.Doc 初始状态：Go 侧硬编码 `[]byte{0, 0}`（Y.js 空文档的最小二进制表示）

#### 4.1.2 DocUpdateRepo

文件：`backend/internal/repository/doc_update_repo.go`

```go
type DocUpdateRepo interface {
    // Append 追加一条增量 update
    Append(ctx context.Context, nodeID uuid.UUID, update []byte, clientID string) error
    // ListSince 获取某 node 自 afterID 起的所有 update（compact 用）
    ListSince(ctx context.Context, nodeID uuid.UUID, afterID int64) ([]models.DocUpdate, error)
    // CountByNode 获取某 node 的 pending update 数量
    CountByNode(ctx context.Context, nodeID uuid.UUID) (int64, error)
    // DeleteUpTo 删除 id <= maxID 的所有 update（compact 后清理）
    DeleteUpTo(ctx context.Context, nodeID uuid.UUID, maxID int64) error
}
```

#### 4.1.3 Models

文件：`backend/internal/models/doc.go`

```go
type DocState struct {
    NodeID         uuid.UUID `db:"node_id"`
    YDocState      []byte    `db:"ydoc_state"`
    Version        int64     `db:"version"`
    MarkdownPlain  string    `db:"markdown_plain"`
    LastCompactedAt time.Time `db:"last_compacted_at"`
}

type DocUpdate struct {
    ID        int64     `db:"id"`
    NodeID    uuid.UUID `db:"node_id"`
    Update    []byte    `db:"update"`
    ClientID  string    `db:"client_id"`
    CreatedAt time.Time `db:"created_at"`
}
```

### 4.2 Hub 改造

#### 4.2.1 Hub 依赖注入

`NewHub` 签名变更：

```go
type HubDeps struct {
    DocStateRepo  repository.DocStateRepo
    DocUpdateRepo repository.DocUpdateRepo
    Log           zerolog.Logger
    Release       func(nodeID uuid.UUID)
}

func NewHub(nodeID uuid.UUID, deps HubDeps) *Hub
```

HubManager 相应持有 `HubDeps`（去掉 Release，在 GetOrCreate 时注入）。

#### 4.2.2 inbound 消息路由

替换当前 `not_implemented` 逻辑：

```go
case msg := <-h.inbound:
    var envelope wsMessage
    json.Unmarshal(msg.Payload, &envelope)

    switch envelope.Type {
    case "sync_step2", "update":
        h.handleUpdate(msg.Client, envelope)
    case "awareness":
        h.handleAwareness(msg.Client, envelope)
    default:
        // 忽略未知消息
    }
```

#### 4.2.3 handleUpdate

```go
func (h *Hub) handleUpdate(client *Client, msg wsMessage) {
    // 1. 权限检查：readable → 丢弃 + log
    if client.Level == "readable" {
        h.log.Warn().Str("client_id", client.ID).Msg("readable client sent update, dropped")
        return
    }

    // 2. 解码 base64 payload
    updateBytes := decodeBase64(msg.Payload.Update)

    // 3. 落库 doc_updates（异步，不阻塞广播）
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := h.deps.DocUpdateRepo.Append(ctx, h.NodeID, updateBytes, client.ID); err != nil {
            h.log.Error().Err(err).Msg("failed to persist doc update")
        }
    }()

    // 4. 广播给其他客户端
    outMsg := mustEncode(wsMessage{
        Type:    "update",
        Payload: msg.Payload, // 原样转发 base64
    })
    for _, c := range h.clients {
        if c.ID == client.ID {
            continue
        }
        select {
        case c.Send <- outMsg:
        default:
            h.log.Warn().Str("client_id", c.ID).Msg("outbound queue full, dropped update")
        }
    }
}
```

#### 4.2.4 handleAwareness

```go
func (h *Hub) handleAwareness(client *Client, msg wsMessage) {
    // awareness 任何权限级别都允许发送
    outMsg := mustEncode(wsMessage{
        Type:    "awareness",
        Payload: msg.Payload,
    })
    for _, c := range h.clients {
        if c.ID == client.ID {
            continue
        }
        select {
        case c.Send <- outMsg:
        default:
        }
    }
}
```

#### 4.2.5 register 时发送 sync_step1 + peers

在 `hub.go` register 分支，注册成功后：

```go
// 1. 发送当前在线列表
peers := make([]PeerInfo, 0, len(h.clients))
for _, c := range h.clients {
    if c.ID == client.ID { continue }
    peers = append(peers, PeerInfo{UserID: c.UserID, DisplayName: c.DisplayName, Level: c.Level})
}
client.Send <- mustEncode(wsMessage{Type: "peers", Payload: peers})

// 2. 发送 sync_step1（doc state）
go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    state, err := h.deps.DocStateRepo.GetOrInit(ctx, h.NodeID)
    if err != nil {
        h.log.Error().Err(err).Msg("failed to load doc state for sync_step1")
        return
    }
    // 也需要追加 pending doc_updates（还未 compact 的）
    updates, _ := h.deps.DocUpdateRepo.ListSince(ctx, h.NodeID, 0)
    // state vector = ydoc_state + all pending updates 合并后的 state vector
    // 注意：Go 侧不解析 Y.Doc 结构，直接把 ydoc_state + pending updates 全部发给客户端
    // 客户端用 Y.applyUpdate 逐个应用
    client.Send <- mustEncode(wsMessage{
        Type: "sync_step1",
        Payload: SyncStep1Payload{
            DocState: base64Encode(state.YDocState),
            Updates:  base64EncodeAll(updates),
        },
    })
}()

// 3. 广播 peer_join 给其他人
broadcastToOthers(client, wsMessage{
    Type: "peer_join",
    Payload: PeerInfo{UserID: client.UserID, DisplayName: client.DisplayName, Level: client.Level},
})
```

**关键设计决策：Go 侧不解析 Y.Doc 二进制**。服务端不引入 Y.js 运行时，只做字节流存储和转发。sync_step1 时把 `doc_states.ydoc_state` + 所有 pending `doc_updates` 打包发给客户端，由客户端侧的 Y.js 库完成合并。

这意味着 sync_step1 的 payload 结构为：
```json
{
  "docState": "<base64>",         // doc_states.ydoc_state
  "pendingUpdates": ["<base64>", "<base64>", ...]  // 未 compact 的 doc_updates
}
```

客户端收到后：
```typescript
const ydoc = new Y.Doc()
Y.applyUpdate(ydoc, base64Decode(payload.docState))
for (const update of payload.pendingUpdates) {
    Y.applyUpdate(ydoc, base64Decode(update))
}
// 然后绑定 TipTap editor
```

### 4.3 Handler 改造

文件：`backend/internal/ws/handler.go`

当前 `client.Level` 硬编码为 `"readable"`，需改为调用 PermissionService：

```go
// Handler 签名变更
func Handler(cfg config.Config, hm *HubManager, permSvc *service.PermissionService, log zerolog.Logger) gin.HandlerFunc

// 握手流程新增
level, err := permSvc.ResolveLevel(c.Request.Context(), userID, nodeID)
if err != nil || level == "none" {
    writeErr(c, http.StatusForbidden, "permission_denied", "no access to this document")
    return
}

client := &Client{
    // ...
    Level:       level,
    DisplayName: user.DisplayName,  // 需要查 user
}
```

Handler 还需要新增 `permSvc` 和 `userRepo` 依赖，以便在握手时获取用户 DisplayName（用于 peers/awareness）。

### 4.4 Client 结构扩展

```go
type Client struct {
    ID          string
    UserID      uuid.UUID
    DisplayName string    // 新增：用于 peers 列表和 awareness
    Level       string    // "manage" | "edit" | "readable"
    Conn        *websocket.Conn
    Send        chan []byte
}
```

### 4.5 Compact Scheduler

文件：`backend/internal/ws/compact.go`（新建）

```go
type CompactScheduler struct {
    docStateRepo  repository.DocStateRepo
    docUpdateRepo repository.DocUpdateRepo
    pool          *pgxpool.Pool
    log           zerolog.Logger
    interval      time.Duration  // 60s
}

func NewCompactScheduler(deps ...) *CompactScheduler

func (s *CompactScheduler) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            s.runOnce(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (s *CompactScheduler) runOnce(ctx context.Context) {
    // 查找需要 compact 的 node_id
    // 条件 1: count(doc_updates) >= 200
    // 条件 2: last_compacted_at < now()-30min AND EXISTS(doc_updates)
    nodeIDs := s.findCandidates(ctx)

    for _, nodeID := range nodeIDs {
        if err := s.compactOne(ctx, nodeID); err != nil {
            s.log.Error().Err(err).Str("node_id", nodeID.String()).Msg("compact failed")
        }
    }
}

func (s *CompactScheduler) compactOne(ctx context.Context, nodeID uuid.UUID) error {
    // BEGIN transaction
    // SELECT ydoc_state FROM doc_states WHERE node_id = $1 FOR UPDATE
    // SELECT id, update FROM doc_updates WHERE node_id = $1 ORDER BY id
    // 拼接所有 update bytes（Go 侧只做字节拼接，不解析 Y.js 结构）
    // 注意：不能在 Go 侧 merge Y.js updates，因为 Go 没有 Y.js 运行时
    // 方案：将 ydoc_state + 所有 updates 拼成一个列表存回 ydoc_state
    //   → 不行，这不是合法的 Y.Doc binary
    //
    // ★ 核心问题：Go 侧如何合并 Y.js updates？
    // 方案 A：引入 Go 的 Y.js 绑定（yrs-go / y-crdt-go）
    // 方案 B：不在 Go 侧合并，compact 时通过 sidecar 调用
    // 方案 C：不做服务端 merge，只做 "清理已广播的 updates"
    //
    // 采用方案 C（MVP 简化）：
    // compact 不做 Y.Doc merge，而是：
    // 1. 把所有 pending updates 追加到 doc_states.ydoc_state 末尾（用分隔格式）
    //    → 不可行，ydoc_state 应该是合法 Y.Doc binary
    //
    // 最终方案：引入 y-crdt Go 绑定，或用 Node.js sidecar
    // 考虑到项目复杂度，采用 Node.js sidecar worker：
    //   backend/scripts/compact-worker.js
    //   Go 通过 exec.Command 调用，传入 ydoc_state + updates，返回 merged binary
    //
    // 见 §4.6
}
```

### 4.6 Compact 策略：Node.js Sidecar

**问题**：Go 没有原生 Y.js 运行时，无法在 Go 侧合并 Y.Doc updates。

**方案**：使用 Node.js 脚本作为 sidecar worker，Go 通过 `exec.Command` 调用。

文件：`backend/scripts/compact-worker.mjs`

```javascript
// 从 stdin 读取 JSON: { docState: "<base64>", updates: ["<base64>", ...] }
// 输出 JSON: { merged: "<base64>", plainText: "<string>" }

import * as Y from 'yjs'

const input = JSON.parse(await readStdin())
const ydoc = new Y.Doc()

// 应用当前 doc state
if (input.docState) {
    Y.applyUpdate(ydoc, Buffer.from(input.docState, 'base64'))
}

// 应用所有 pending updates
for (const update of input.updates) {
    Y.applyUpdate(ydoc, Buffer.from(update, 'base64'))
}

// 输出合并后的完整状态
const merged = Buffer.from(Y.encodeStateAsUpdate(ydoc)).toString('base64')
const plainText = ydoc.getText('default').toString()

console.log(JSON.stringify({ merged, plainText }))
```

Go 调用方式：

```go
func (s *CompactScheduler) mergeUpdates(docState []byte, updates [][]byte) (mergedState []byte, plainText string, err error) {
    input := CompactInput{
        DocState: base64.StdEncoding.EncodeToString(docState),
        Updates:  make([]string, len(updates)),
    }
    for i, u := range updates {
        input.Updates[i] = base64.StdEncoding.EncodeToString(u)
    }

    inputJSON, _ := json.Marshal(input)
    cmd := exec.CommandContext(ctx, "node", "scripts/compact-worker.mjs")
    cmd.Stdin = bytes.NewReader(inputJSON)

    output, err := cmd.Output()
    // parse output JSON → merged + plainText
}
```

**compact 完整流程**：

```
每 60s：
  1. 查找候选 node_id（count ≥ 200 或 last_compacted_at 超时且有 pending）
  2. 对每个 node_id：
     BEGIN
     SELECT ydoc_state FROM doc_states WHERE node_id = $1 FOR UPDATE
     SELECT id, update FROM doc_updates WHERE node_id = $1 ORDER BY id
     调用 compact-worker.mjs 合并
     UPDATE doc_states SET ydoc_state = merged,
                           markdown_plain = plainText,
                           version = version + 1,
                           last_compacted_at = now()
     DELETE FROM doc_updates WHERE node_id = $1 AND id <= max_seen_id
     COMMIT
```

### 4.7 Router 改造

`router.go` 中 WS handler 需要传入 PermissionService 和 UserRepo：

```go
// 当前
api.GET("/ws", ws.Handler(cfg, hubManager, log))

// 改为
api.GET("/ws", ws.Handler(cfg, hubManager, permissionService, userRepo, log))
```

HubManager 构造也需要传入 DocStateRepo 和 DocUpdateRepo：

```go
docStateRepo := repository.NewDocStateRepo(pool)
docUpdateRepo := repository.NewDocUpdateRepo(pool)
hubManager := ws.NewHubManager(log, docStateRepo, docUpdateRepo)
```

CompactScheduler 在 `main.go` 中启动：

```go
compactScheduler := ws.NewCompactScheduler(docStateRepo, docUpdateRepo, pool, log)
go compactScheduler.Run(ctx)
```

---

## 5. 前端详细设计

### 5.1 依赖包

```bash
pnpm add @tiptap/react @tiptap/starter-kit @tiptap/extension-collaboration \
         @tiptap/extension-collaboration-cursor \
         yjs y-prosemirror y-protocols
```

| 包 | 用途 |
|---|---|
| `@tiptap/react` | TipTap React 绑定 |
| `@tiptap/starter-kit` | 基础编辑功能集（标题/粗斜/列表/代码/引用/分割线） |
| `@tiptap/extension-collaboration` | Y.js 协同扩展（内部使用 y-prosemirror） |
| `@tiptap/extension-collaboration-cursor` | 协同光标扩展 |
| `yjs` | Y.js CRDT 核心 |
| `y-prosemirror` | ProseMirror ↔ Y.js 绑定 |
| `y-protocols` | Y.js 编码/解码工具（awareness protocol） |

### 5.2 文件结构

```
frontend/src/
├── collab/
│   ├── FpgWsProvider.ts      # 自定义 WS Provider（替代 y-websocket）
│   ├── useCollabEditor.ts    # 组装 TipTap + Y.js 的 hook
│   └── awareness-colors.ts   # 协同光标颜色分配
├── components/
│   ├── Editor.tsx            # TipTap 编辑器组件
│   ├── editor.css            # 编辑器样式
│   ├── OnlineUsers.tsx       # 在线用户列表组件
│   └── online-users.css      # 在线用户样式
├── pages/
│   ├── DocPage.tsx           # 重写：加载文档 + 建立 WS + 渲染编辑器
│   └── doc-page.css          # 文档页样式
```

### 5.3 FpgWsProvider

文件：`frontend/src/collab/FpgWsProvider.ts`

不使用 `y-websocket` 的 `WebsocketProvider`，因为我们的 WS 协议是自定义 JSON 信封，不是 y-websocket 原生二进制协议。

```typescript
import * as Y from 'yjs'
import { Awareness } from 'y-protocols/awareness'

type ProviderStatus = 'connecting' | 'connected' | 'disconnected'

interface PeerInfo {
  userId: string
  displayName: string
  level: string
}

interface FpgWsProviderOptions {
  docId: string
  accessToken: string
  ydoc: Y.Doc
  awareness: Awareness
  onStatusChange: (status: ProviderStatus) => void
  onPeersChange: (peers: PeerInfo[]) => void
  onSynced: () => void
  onJoinRejected: (reason: string) => void
}

class FpgWsProvider {
  private ws: WebSocket | null = null
  private ydoc: Y.Doc
  private awareness: Awareness
  private options: FpgWsProviderOptions
  private peers: PeerInfo[] = []
  private synced = false
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectDelay = 1000  // 1s, 2s, 4s... 最大 30s
  private destroyed = false

  constructor(options: FpgWsProviderOptions) {
    this.ydoc = options.ydoc
    this.awareness = options.awareness
    this.options = options

    // 监听本地 Y.Doc 变更 → 发送 update
    this.ydoc.on('update', this.handleLocalUpdate)
    // 监听 awareness 变更 → 发送 awareness
    this.awareness.on('update', this.handleLocalAwareness)

    this.connect()
  }

  private connect() {
    if (this.destroyed) return

    const wsUrl = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/api/ws?doc=${this.options.docId}`

    this.ws = new WebSocket(wsUrl, [`access.${this.options.accessToken}`])
    this.options.onStatusChange('connecting')

    this.ws.onopen = () => {
      this.options.onStatusChange('connected')
      this.reconnectDelay = 1000  // 重置
    }

    this.ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)
      this.handleMessage(msg)
    }

    this.ws.onclose = () => {
      this.options.onStatusChange('disconnected')
      this.synced = false
      this.scheduleReconnect()
    }
  }

  private handleMessage(msg: { type: string; payload: unknown }) {
    switch (msg.type) {
      case 'sync_step1':
        this.handleSyncStep1(msg.payload)
        break
      case 'update':
        this.handleRemoteUpdate(msg.payload)
        break
      case 'awareness':
        this.handleRemoteAwareness(msg.payload)
        break
      case 'peers':
        this.peers = msg.payload as PeerInfo[]
        this.options.onPeersChange([...this.peers])
        break
      case 'peer_join':
        this.peers = [...this.peers, msg.payload as PeerInfo]
        this.options.onPeersChange([...this.peers])
        break
      case 'peer_leave':
        this.peers = this.peers.filter(p => p.userId !== (msg.payload as { userId: string }).userId)
        this.options.onPeersChange([...this.peers])
        break
      case 'join_rejected':
        this.options.onJoinRejected((msg.payload as { reason: string }).reason)
        break
      case 'force_logout':
        // AuthContext 会处理
        break
    }
  }

  private handleSyncStep1(payload: { docState: string; pendingUpdates: string[] }) {
    // 应用服务端状态
    Y.applyUpdate(this.ydoc, base64Decode(payload.docState))
    for (const update of payload.pendingUpdates) {
      Y.applyUpdate(this.ydoc, base64Decode(update))
    }
    this.synced = true
    this.options.onSynced()
  }

  private handleLocalUpdate = (update: Uint8Array, origin: unknown) => {
    // 只发送本地产生的 update（不转发从服务端收到的）
    if (origin === this) return
    this.send({ type: 'update', payload: { update: base64Encode(update) } })
  }

  private handleRemoteUpdate(payload: { update: string }) {
    // 标记 origin=this，避免 handleLocalUpdate 再转发
    Y.applyUpdate(this.ydoc, base64Decode(payload.update), this)
  }

  private handleLocalAwareness = () => {
    const encoded = awarenessProtocol.encodeAwarenessUpdate(this.awareness, [this.ydoc.clientID])
    this.send({ type: 'awareness', payload: { data: base64Encode(encoded) } })
  }

  private handleRemoteAwareness(payload: { data: string }) {
    awarenessProtocol.applyAwarenessUpdate(this.awareness, base64Decode(payload.data), this)
  }

  private send(msg: object) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg))
    }
  }

  private scheduleReconnect() {
    if (this.destroyed) return
    this.reconnectTimer = setTimeout(() => {
      this.connect()
    }, this.reconnectDelay)
    this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000)
  }

  destroy() {
    this.destroyed = true
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.ydoc.off('update', this.handleLocalUpdate)
    this.awareness.off('update', this.handleLocalAwareness)
    if (this.ws) {
      this.ws.onclose = null  // 防止触发重连
      this.ws.close()
    }
  }
}
```

### 5.4 useCollabEditor Hook

文件：`frontend/src/collab/useCollabEditor.ts`

```typescript
import { useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCursor from '@tiptap/extension-collaboration-cursor'
import * as Y from 'yjs'
import { Awareness } from 'y-protocols/awareness'

interface UseCollabEditorOptions {
  docId: string
  accessToken: string
  userDisplayName: string
  userColor: string
  isReadonly: boolean
}

interface UseCollabEditorReturn {
  editor: Editor | null
  provider: FpgWsProvider | null
  status: ProviderStatus
  peers: PeerInfo[]
  synced: boolean
  joinError: string | null
}

function useCollabEditor(options: UseCollabEditorOptions): UseCollabEditorReturn {
  const [status, setStatus] = useState<ProviderStatus>('connecting')
  const [peers, setPeers] = useState<PeerInfo[]>([])
  const [synced, setSynced] = useState(false)
  const [joinError, setJoinError] = useState<string | null>(null)

  const ydocRef = useRef<Y.Doc | null>(null)
  const awarenessRef = useRef<Awareness | null>(null)
  const providerRef = useRef<FpgWsProvider | null>(null)

  // 初始化 Y.Doc + Awareness + Provider
  useEffect(() => {
    const ydoc = new Y.Doc()
    const awareness = new Awareness(ydoc)
    ydocRef.current = ydoc
    awarenessRef.current = awareness

    // 设置本地 awareness 状态
    awareness.setLocalStateField('user', {
      name: options.userDisplayName,
      color: options.userColor,
    })

    const provider = new FpgWsProvider({
      docId: options.docId,
      accessToken: options.accessToken,
      ydoc,
      awareness,
      onStatusChange: setStatus,
      onPeersChange: setPeers,
      onSynced: () => setSynced(true),
      onJoinRejected: (reason) => setJoinError(reason),
    })
    providerRef.current = provider

    return () => {
      provider.destroy()
      ydoc.destroy()
    }
  }, [options.docId, options.accessToken])

  // TipTap editor
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        history: false,  // 禁用内置 history，改用 Y.js undo manager
      }),
      Collaboration.configure({
        document: ydocRef.current!,
        field: 'default',  // Y.Doc 中的 XMLFragment 字段名
      }),
      CollaborationCursor.configure({
        provider: providerRef.current!,  // 需要适配
        user: {
          name: options.userDisplayName,
          color: options.userColor,
        },
      }),
    ],
    editable: !options.isReadonly,
  }, [options.docId])  // docId 变化时重建 editor

  return { editor, provider: providerRef.current, status, peers, synced, joinError }
}
```

**注意**：`CollaborationCursor` 通常期望一个 `y-websocket` 兼容的 provider。由于我们自定义了 provider，需要让 `FpgWsProvider` 实现 `awareness` 属性（`provider.awareness`）。这是因为 `@tiptap/extension-collaboration-cursor` 内部通过 `provider.awareness` 获取 Awareness 实例。

在 `FpgWsProvider` 中暴露：
```typescript
get awareness(): Awareness {
  return this._awareness
}
```

### 5.5 Editor 组件

文件：`frontend/src/components/Editor.tsx`

```tsx
import { EditorContent } from '@tiptap/react'

interface EditorProps {
  editor: Editor | null
  synced: boolean
  status: ProviderStatus
}

function Editor({ editor, synced, status }: EditorProps) {
  if (!synced) {
    return (
      <div className="editor-loading">
        <div className="editor-loading-spinner" />
        <span>正在同步文档...</span>
      </div>
    )
  }

  return (
    <div className="editor-wrapper">
      {status === 'disconnected' && (
        <div className="editor-reconnecting-bar">
          连接已断开，正在重连...
        </div>
      )}
      <EditorContent editor={editor} className="editor-content" />
    </div>
  )
}
```

### 5.6 OnlineUsers 组件

文件：`frontend/src/components/OnlineUsers.tsx`

在编辑器顶栏右侧显示在线用户头像列表。

```tsx
interface OnlineUsersProps {
  peers: PeerInfo[]
  currentUser: { displayName: string; color: string }
}

function OnlineUsers({ peers, currentUser }: OnlineUsersProps) {
  const allUsers = [
    { displayName: currentUser.displayName, color: currentUser.color, isSelf: true },
    ...peers.map(p => ({ displayName: p.displayName, color: getAwarenessColor(p.userId), isSelf: false })),
  ]

  return (
    <div className="online-users">
      {allUsers.slice(0, 5).map((u, i) => (
        <div
          key={i}
          className="online-user-avatar"
          style={{ borderColor: u.color, zIndex: allUsers.length - i }}
          title={u.displayName + (u.isSelf ? ' (你)' : '')}
        >
          {u.displayName.charAt(0).toUpperCase()}
        </div>
      ))}
      {allUsers.length > 5 && (
        <div className="online-user-overflow">+{allUsers.length - 5}</div>
      )}
    </div>
  )
}
```

### 5.7 DocPage 重写

文件：`frontend/src/pages/DocPage.tsx`

```tsx
function DocPage() {
  const { id } = useParams<{ id: string }>()
  const { accessToken } = useAuth()
  const [nodeInfo, setNodeInfo] = useState<{ title: string; level: string } | null>(null)
  const [error, setError] = useState<string | null>(null)

  // 1. 先 fetch 节点信息（获取 title + 权限级别）
  useEffect(() => {
    if (!id) return
    request<NodeResponse>('GET', API_ENDPOINTS.node(id))
      .then(node => setNodeInfo({ title: node.title, level: node.effective_level }))
      .catch(err => setError(err instanceof ApiError ? err.message : '加载失败'))
  }, [id])

  // 2. 获取当前用户信息
  const [me, setMe] = useState<MeResponse | null>(null)
  useEffect(() => {
    request<MeResponse>('GET', API_ENDPOINTS.me).then(setMe)
  }, [])

  // 3. 初始化协同编辑器
  const { editor, status, peers, synced, joinError } = useCollabEditor({
    docId: id!,
    accessToken: accessToken!,
    userDisplayName: me?.display_name ?? '匿名',
    userColor: getAwarenessColor(me?.id ?? ''),
    isReadonly: nodeInfo?.level === 'readable',
  })

  if (error || joinError) {
    return <div className="doc-error">{error || (joinError === 'doc_full' ? '文档已满（最多 20 人）' : '无权访问')}</div>
  }

  if (!nodeInfo || !me) {
    return <div className="doc-loading">加载中...</div>
  }

  return (
    <div className="doc-page">
      <div className="doc-topbar">
        <h2 className="doc-title">{nodeInfo.title}</h2>
        <div className="doc-topbar-right">
          <OnlineUsers peers={peers} currentUser={{ displayName: me.display_name, color: getAwarenessColor(me.id) }} />
          <div className={`doc-status doc-status--${status}`} title={STATUS_LABELS[status]}>
            <span className="doc-status-dot" />
          </div>
        </div>
      </div>
      <Editor editor={editor} synced={synced} status={status} />
    </div>
  )
}
```

### 5.8 Awareness 颜色分配

文件：`frontend/src/collab/awareness-colors.ts`

```typescript
// 8 个高辨识度颜色，基于设计系统扩展
const AWARENESS_PALETTE = [
  '#0ea5e9',  // primary blue
  '#10b981',  // green
  '#f59e0b',  // orange
  '#8b5cf6',  // purple
  '#ec4899',  // pink
  '#14b8a6',  // teal
  '#f43f5e',  // rose
  '#6366f1',  // indigo
]

export function getAwarenessColor(userId: string): string {
  let hash = 0
  for (let i = 0; i < userId.length; i++) {
    hash = (hash * 31 + userId.charCodeAt(i)) >>> 0
  }
  return AWARENESS_PALETTE[hash % AWARENESS_PALETTE.length]
}
```

### 5.9 CSS 样式

#### doc-page.css

```css
.doc-page {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.doc-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  border-bottom: 1px solid var(--gray-200);
  background: var(--white);
}

.doc-title {
  font-family: var(--font-cn);
  font-size: 18px;
  font-weight: 600;
  color: var(--gray-800);
  margin: 0;
}

.doc-topbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.doc-status {
  display: flex;
  align-items: center;
}

.doc-status-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  transition: background var(--transition);
}

.doc-status--connected .doc-status-dot { background: var(--green); }
.doc-status--connecting .doc-status-dot { background: var(--orange); }
.doc-status--disconnected .doc-status-dot { background: var(--gray-300); }

.doc-error, .doc-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-family: var(--font-cn);
  color: var(--gray-500);
  font-size: 16px;
}

.doc-error { color: var(--rose); }
```

#### editor.css

```css
.editor-wrapper {
  flex: 1;
  overflow-y: auto;
  background: var(--gray-50);
  display: flex;
  justify-content: center;
  padding: 32px 0;
}

.editor-content {
  width: 100%;
  max-width: 800px;
  min-height: calc(100vh - 200px);
  background: var(--white);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
  padding: 48px 56px;
  font-family: var(--font-cn);
  font-size: 16px;
  line-height: 1.8;
  color: var(--gray-700);
}

/* TipTap ProseMirror 内部样式 */
.editor-content .ProseMirror {
  outline: none;
  min-height: 400px;
}

.editor-content .ProseMirror h1 {
  font-size: 28px;
  font-weight: 700;
  color: var(--gray-800);
  margin: 24px 0 12px;
  line-height: 1.3;
}

.editor-content .ProseMirror h2 {
  font-size: 22px;
  font-weight: 600;
  color: var(--gray-800);
  margin: 20px 0 10px;
  line-height: 1.4;
}

.editor-content .ProseMirror h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--gray-800);
  margin: 16px 0 8px;
}

.editor-content .ProseMirror p {
  margin: 8px 0;
}

.editor-content .ProseMirror code {
  background: var(--gray-100);
  border-radius: 3px;
  padding: 2px 4px;
  font-size: 14px;
  color: var(--primary-dark);
}

.editor-content .ProseMirror pre {
  background: var(--gray-900);
  color: var(--gray-100);
  border-radius: var(--radius-sm);
  padding: 16px 20px;
  font-size: 14px;
  line-height: 1.6;
  overflow-x: auto;
  margin: 16px 0;
}

.editor-content .ProseMirror blockquote {
  border-left: 3px solid var(--primary);
  padding-left: 16px;
  margin: 12px 0;
  color: var(--gray-500);
}

.editor-content .ProseMirror ul,
.editor-content .ProseMirror ol {
  padding-left: 24px;
  margin: 8px 0;
}

.editor-content .ProseMirror hr {
  border: none;
  border-top: 1px solid var(--gray-200);
  margin: 24px 0;
}

/* 协同光标样式 */
.collaboration-cursor__caret {
  position: relative;
  border-left: 2px solid;
  margin-left: -1px;
  margin-right: -1px;
  pointer-events: none;
  word-break: normal;
}

.collaboration-cursor__label {
  position: absolute;
  top: -1.4em;
  left: -1px;
  font-size: 11px;
  font-family: var(--font-ui);
  font-weight: 500;
  padding: 1px 6px;
  border-radius: 3px 3px 3px 0;
  color: var(--white);
  white-space: nowrap;
  user-select: none;
  pointer-events: none;
}

.editor-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 12px;
  color: var(--gray-400);
  font-family: var(--font-cn);
}

.editor-loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--gray-200);
  border-top-color: var(--primary);
  border-radius: var(--radius-full);
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.editor-reconnecting-bar {
  background: var(--orange-bg);
  color: var(--orange);
  font-family: var(--font-cn);
  font-size: 13px;
  text-align: center;
  padding: 6px;
  border-bottom: 1px solid var(--orange-light);
}
```

#### online-users.css

```css
.online-users {
  display: flex;
  align-items: center;
}

.online-user-avatar {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-full);
  background: var(--gray-100);
  border: 2px solid;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-ui);
  font-size: 11px;
  font-weight: 600;
  color: var(--gray-600);
  margin-left: -6px;
  position: relative;
  cursor: default;
}

.online-user-avatar:first-child {
  margin-left: 0;
}

.online-user-overflow {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-full);
  background: var(--gray-200);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-ui);
  font-size: 10px;
  font-weight: 600;
  color: var(--gray-500);
  margin-left: -6px;
}
```

---

## 6. GET /api/nodes/:id 响应扩展

当前 `GET /api/nodes/:id` 返回节点基本信息。M5 需要在 DocPage 加载时获取用户对该节点的有效权限级别，以决定编辑器是否只读。

两种方案：
- **A**: 在 `GET /api/nodes/:id` 响应中增加 `effective_level` 字段
- **B**: 新增 `GET /api/nodes/:id/my-access` 端点

**采用方案 A**：在现有 NodeHandler.GetNode 中，调用 PermissionService 解析当前用户权限，附加到响应中。

```go
// NodeResponse 新增字段
type NodeResponse struct {
    // ...existing fields
    EffectiveLevel string `json:"effective_level"` // "manage" | "edit" | "readable" | "none"
}
```

---

## 7. 文档创建时初始化 doc_states

当前创建 `kind='doc'` 的节点时，只插入 `nodes` 表。M5 需要同时在 `doc_states` 表中插入空 Y.Doc 初始状态。

在 `NodeService.CreateNode` 中：
```go
if input.Kind == "doc" {
    // 初始化空文档状态
    emptyYDoc := []byte{0, 0}  // Y.js 空文档最小二进制
    docStateRepo.Insert(ctx, nodeID, emptyYDoc)
}
```

或者在 `DocStateRepo.GetOrInit` 中用 `ON CONFLICT DO NOTHING` 实现惰性初始化（更鲁棒）。**采用 GetOrInit 惰性方案**，避免修改已稳定的 NodeService。

---

## 8. 任务拆分

### Phase 1: 后端 Repository + Models（2 任务）

| Task | 名称 | 依赖 | 说明 |
|---|---|---|---|
| **T-42** | DocState / DocUpdate models + DocStateRepo | — | `models/doc.go` + `repository/doc_state_repo.go`，含 GetOrInit / UpdateAfterCompact |
| **T-43** | DocUpdateRepo | T-42 | `repository/doc_update_repo.go`，含 Append / ListSince / CountByNode / DeleteUpTo |

### Phase 2: Hub 协议实装（3 任务）

| Task | 名称 | 依赖 | 说明 |
|---|---|---|---|
| **T-44** | Hub 依赖注入 + Client 扩展 + Handler 权限校验 | T-42, T-43 | Hub/HubManager 构造函数改造 + Client 加 DisplayName + Handler 调用 PermissionService + router.go 更新 |
| **T-45** | Hub inbound 协议路由 | T-44 | sync_step1 发送 + update 广播/落库 + awareness 转发 + readable 丢弃 + peers/peer_join/peer_leave |
| **T-46** | NodeHandler.GetNode 返回 effective_level | T-44 | NodeResponse 新增字段 + 调用 PermissionService |

### Phase 3: Compact（1 任务）

| Task | 名称 | 依赖 | 说明 |
|---|---|---|---|
| **T-47** | CompactScheduler + compact-worker.mjs | T-43 | 后台 goroutine + Node.js sidecar + docker-compose 加 node 环境 |

### Phase 4: 前端 TipTap + WS（4 任务）

| Task | 名称 | 依赖 | 说明 |
|---|---|---|---|
| **T-48** | 安装 TipTap/Y.js 依赖 + FpgWsProvider | — | pnpm add + collab/FpgWsProvider.ts + awareness-colors.ts |
| **T-49** | useCollabEditor hook | T-48 | collab/useCollabEditor.ts，组装 Y.Doc + Awareness + TipTap + Provider |
| **T-50** | Editor 组件 + editor.css | T-49 | components/Editor.tsx + 全部 ProseMirror 排版样式 + 协同光标样式 |
| **T-51** | DocPage 重写 + OnlineUsers + doc-page.css | T-49, T-50 | pages/DocPage.tsx 完整重写 + 在线用户头像 + 状态指示灯 |

### Phase 5: 集成验证（1 任务）

| Task | 名称 | 依赖 | 说明 |
|---|---|---|---|
| **T-52** | 端到端集成验证 | T-45, T-46, T-47, T-51 | 两浏览器同编 + 光标可见 + 断线重连 + readable 只读 + 20 人上限 |

---

## 9. 风险与缓解

| 风险 | 影响 | 缓解 |
|---|---|---|
| Y.js 空文档二进制 `[0,0]` 可能不正确 | sync_step1 失败 | T-42 编写时用 Node.js 验证 `Y.encodeStateAsUpdate(new Y.Doc())` 的输出 |
| TipTap CollaborationCursor 期望标准 y-websocket provider | 光标不显示 | FpgWsProvider 暴露 `awareness` 属性 + 必要时 monkey-patch |
| Compact sidecar Node.js 进程启动延迟 | compact 缓慢 | 可考虑长驻进程通过 stdio 通信，MVP 先用单次 exec |
| WS 消息乱序 | 文档状态不一致 | Y.js CRDT 天然容忍乱序，无需额外处理 |
| Docker 容器中需要 Node.js 运行时 | 镜像变大 | Dockerfile multi-stage：Go 编译 + Node.js runtime |
| token 过期时 WS 仍在连接 | 安全问题 | D-13：WS 建立后不强制重验，仅权限敏感事件触发（M5 MVP 不实装，留 M6+） |

---

## 10. Definition of Done

- [ ] WS 握手 + JWT 校验 + 权限解析 → 返回真实 level
- [ ] sync_step1 发送 doc_states + pending updates → 客户端正确初始化
- [ ] 用户编辑 → update 广播到其他客户端 + 落库 doc_updates
- [ ] awareness 光标/选区/用户名/颜色 → 实时同步可见
- [ ] readable 用户 → 可看到文档和光标，发送 update 被服务端丢弃
- [ ] 20 人并发上限 → 第 21 人收到 join_rejected
- [ ] compact 触发 → doc_states 更新 + doc_updates 清理 + markdown_plain 抽取
- [ ] 断线重连 → 自动重连 + 不丢数据
- [ ] TipTap 编辑器 → 标题/粗斜/列表/代码/引用/分割线/任务列表
- [ ] 在线用户头像列表 → 实时更新
- [ ] 连接状态指示灯 → connected/connecting/disconnected
- [ ] `GET /api/nodes/:id` → 返回 effective_level 字段
