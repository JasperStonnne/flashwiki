# 云文档系统 架构与需求文档

> **文档状态**: v1.0 Baseline（冻结）· 2026-04-16
> **对齐产品需求**: v0.4
> **维护者**: CC（首席架构师 & PM）
> **流程**: 瀑布模型。本文件是产品/技术决策的单一真源（Single Source of Truth）。所有 Epic 详细设计位于 `docs/epics/`，并必须反向链接本文件中的 Feature ID 与 Decision ID。
> **下游**: Codex 按本文中的 Feature、API、Data Schema、Logic Flow 与 Definition of Done 实现，不做自由发挥。

---

## 0. 项目概述

### 0.1 项目定位

仿飞书的团队在线文档编辑系统：支持多人实时协同编辑 Markdown 文档，具备组织权限管理、历史版本回溯、图床、Web 通知、文档搜索、协同光标显示与目录结构能力。

### 0.2 与前版本关系

本仓库曾包含单用户 TipTap Markdown 编辑器（git `a4e85f2`），已于 `29d6b88 chore: reset project` 整体清空。**编辑器层设计可复用**（TipTap 扩展、Markdown 双向转换、CodeMirror 源码视图、PDF 契约等），但本次按完整团队协同系统重新规划。

### 0.3 当前仓库状态

工作区空，仅保留 `.git/` 与 `.claude/`。本文件落盘后进入阶段 1（基础设施）。

---

## 1. 技术约束（Technical Constraints）

> §1 是本项目架构决策的法律层。未经本文件显式更新，Codex 与后续设计不得偏离。

### 1.1 技术栈（来自 v0.4 §2，锁定）

| 层 | 选型 | 说明 |
|---|---|---|
| 前端语言 | TypeScript 5.x | |
| 前端框架 | React 19 + Vite | 沿用旧版本地验证过的基座 |
| 前端编辑器 | **TipTap 3.x + y-prosemirror** | 见 D-06 |
| 前端包管理 | pnpm | |
| 源码视图 | CodeMirror 6 | 复用旧设计 |
| 后端语言 | Go（1.22+） | v0.4 锁定 |
| 后端 HTTP 框架 | Gin | 单一 REST 入口 |
| 实时通信 | `nhooyr.io/websocket` | v0.4 锁定 |
| 协同算法 | Y.js（YATA CRDT） | 前端用 `yjs` + `y-prosemirror`，后端仅传输+存储字节流，不解析结构 |
| 数据库 | PostgreSQL 16+ | v0.4 锁定 |
| 搜索 | PostgreSQL `tsvector` + `pg_trgm` + **`zhparser`** | 见 D-08 |
| 文件存储 | 本地文件系统（元信息入库） | 见 D-10 |
| 鉴权 | JWT 双 token（access 15min + refresh 7d） | 见 D-03, D-11 |
| 部署 | docker-compose（frontend / backend / postgres） | |

### 1.2 架构决策记录（Frozen Decisions）

| ID | 主题 | 决策 | 关联 Feature |
|---|---|---|---|
| **D-01** | Manager 数量 | 允许多个 Manager。权限敏感写操作（改角色、改权限组、改文档权限）在服务端用 PostgreSQL `SELECT … FOR UPDATE` 行锁互斥 | F-21, F-22, F-25 |
| **D-02** | 组织边界 | 单组织部署，不引入 `organization_id`。未来多租户作为 v2 改造议题 | 全局 |
| **D-03** | JWT 刷新 | refresh token 采用**一次性轮换 + 复用检测**：每次刷新签发新 refresh 并将旧的标记 `replaced_by`；若检测到已 `replaced` 或 `revoked` 的 refresh 再次出现 → 视为泄漏，级联吊销该用户所有 refresh | F-02, F-03 |
| **D-04** | 强制下线实现 | `users.token_version` 计数器，JWT Claims 携带 `tv`；中间件校验 `tv == user.token_version`，Manager 强制下线时 `token_version += 1`，其余所有 access + refresh 立即失效 | F-06 |
| **D-05** | 文档树 | **Adjacency List**：`nodes.parent_id` 自引用，查询用 `WITH RECURSIVE`。ltree 留作 v2 性能优化 | F-09 ~ F-13 |
| **D-06** | 编辑器基座 | TipTap + `y-prosemirror`（不自研编辑器）。Y.Doc 在前端持有，通过 `y-websocket` 风格协议与后端同步 | F-14 ~ F-18 |
| **D-07a** | 权限继承 | **显式覆盖语义**：节点若存在任何 `node_permissions` 行即"脱钩父级"，否则沿 `parent_id` 链向上找最近一个有权限行的祖先作为"生效节点" | F-21 |
| **D-07b** | 权限级别合并 | 在生效节点上：个人权限存在即覆盖组权限；`none` 为显式禁止，优先级最高；同一作用域内其余取 max(readable < edit < manage) | F-21 |
| **D-07c** | Manager 全局覆盖 | `users.role = 'manager'` 对任何节点短路返回 `manage`，不查 `node_permissions` | F-21 |
| **D-08** | 中文搜索 | 启用 PostgreSQL `zhparser` 扩展构建中文 tsvector；GIN 索引；`pg_trgm` 用于标题模糊匹配 | F-30, F-31 |
| **D-09a** | Y.js 增量存储 | 每条 WS update 以 bytea 单行写入 `doc_updates`；异步任务按阈值（节点上 update 行数 ≥ 200 **或**距上次 compact ≥ 30 分钟）合并进 `doc_states.ydoc_state` 并清理已合并行 | F-16, F-19 |
| **D-09b** | 快照内容 | `doc_snapshots` 存**完整 Y.Doc 二进制**（非 Markdown 文本）。生成时机：手动触发 + 每日 02:00 每个活跃文档自动保留一份 | F-19, F-20 |
| **D-09c** | 回溯广播 | 回溯时后端用快照替换 `doc_states.ydoc_state`、清空 `doc_updates`，并向该文档所有 WS 客户端推送 `{type: "reload", new_state_vector: <base64>}`；前端收到后重置 Y.Doc，重新订阅 | F-20 |
| **D-10a** | Readable 协同 | readable 用户可建立 WS 并接收他人 awareness（看到光标），但服务端拦截其所有 `sync-update` 消息（丢弃并记日志） | F-15, F-18 |
| **D-10b** | 20 人并发上限 | 每文档维护一个 goroutine hub，`Join` 时原子计数；超限返回 `{type:"join_rejected", reason:"doc_full"}` | F-15 |
| **D-11** | 通知持久化 | `notifications` 表持久化所有订阅事件；WS 在线即推送 `{type:"notify", id, ...}`；离线上线后拉 `GET /api/notifications?unread=1` 补发 | F-26, F-27 |
| **D-12** | 文件访问 | 所有 `/api/uploads/:id` 经后端鉴权 Proxy：中间件解析 JWT → 按文件所属节点做 §3 权限解析 → 读磁盘流回 | F-23, F-25 |
| **D-13** | WS Token 宽限 | WS 握手时严格校验 access；会话建立后不强制重验。access 过期期间，前端静默用 refresh 换新 token；新 token 通过下一条业务消息携带。**仅权限敏感事件（打开新文档、收到服务端的 `auth_renew_required`）触发强制重验** | F-15 ~ F-18 |

---

## 2. 用户与组织模型

### 2.1 角色

| 角色 | 全局能力 |
|---|---|
| **Manager** | 管理所有成员；创建/删除权限组；指定组长；对所有节点默认 `manage`；强制用户下线；可修改其他用户角色（包括册封/撤销 Manager） |
| **Group Leader** | 仅能管理本组成员的加入/退出 |
| **Member** | 基础用户，按节点权限访问 |

> 首位 Manager：数据库初始化 seed 中创建的管理员账户（详见 `docs/epics/infra.md`）。

### 2.2 权限组（Group）

- 多对多：用户 ∈ 0 ~ N 个组
- 组长由 Manager 指定；一个组仅一个组长（`groups.leader_id` 字段）
- Manager 可创建、重命名、删除组
- 删除组：所有 `group_members` 级联删除；`node_permissions` 中针对该组的行级联删除

---

## 3. 文档权限模型

### 3.1 四级权限

| 级别 | 内容 | 隐含能力 |
|---|---|---|
| `manage` | 编辑 + 管理该节点权限 | 读 + 写 + 改权限 + 改名 + 移动 + 删除 |
| `edit` | 读写内容 | 读 + 写内容 + 改名 |
| `readable` | 只读 | 可打开、可接收 awareness；不能发送 update |
| `none` | 无访问 | 不可见于目录树与搜索结果；API 返回 404（不暴露存在性） |

### 3.2 权限解析算法（D-07 的规范化伪代码）

```text
ResolvePermission(user, node) -> Level:
  if user.role == "manager":
    return "manage"                                  # D-07c

  effectiveNode = FindEffectiveNode(node)
  if effectiveNode == null:
    return "none"                                    # 整条链都无权限行 → 默认拒绝

  indRows = permissions(effectiveNode) where subject=user
  if indRows.nonEmpty():
    if any row.level == "none": return "none"
    return max(indRows.level in {manage, edit, readable})

  userGroupIds = groups(user)
  grpRows = permissions(effectiveNode) where subject_type="group" and subject_id in userGroupIds
  if grpRows.nonEmpty():
    if any row.level == "none": return "none"
    return max(grpRows.level in {manage, edit, readable})

  return "none"                                      # 生效节点有权限行但无一条对该用户适用

FindEffectiveNode(node):
  cur = node
  while cur != null:
    if exists any row in node_permissions(cur): return cur
    cur = cur.parent
  return null
```

实现层：用一次 `WITH RECURSIVE` 查询拿到 `node` 的所有祖先链与每级的权限行，然后在 Go 侧按上述算法解析，避免多次往返。

### 3.3 继承与覆盖

- 节点**未设置任何权限行**时 → 向上找最近有权限行的祖先，用其权限集
- 节点**一旦设置第一条权限行** → 立即"脱钩父级"；此后父级权限变更对其无影响
- 权限集中存在 `none` → 最终结果必为 `none`（即使同时存在 `manage`）

### 3.4 操作鉴权对照

| 操作 | 最低要求 |
|---|---|
| 目录树中看到节点、搜索结果中出现 | ≥ `readable` |
| 打开文档只读（含 awareness） | ≥ `readable` |
| 发送 Y.js update | ≥ `edit` |
| 改文档名 | ≥ `edit` |
| 删除文档 / 移动 / 重命名文件夹 | ≥ `manage` |
| 设置本节点权限 | `manage` |
| 创建快照 | ≥ `edit` |
| 回溯历史版本 | ≥ `manage` |
| 订阅通知 | ≥ `readable` |

---

## 4. 功能需求模块

### 4.1 鉴权模块（F-01 ~ F-07）

- 邮箱 + 密码注册 / 登录
- JWT 双 token：access 15min，refresh 7d
- refresh token SHA256 后入库，支持一次性轮换与复用检测（D-03）
- Manager 强制某用户下线（D-04）
- 用户资料：`display_name`、`locale (en|zh)`、密码修改

### 4.2 文档与目录模块（F-08 ~ F-13）

- 节点（`nodes`）统一建模：`kind ∈ {folder, doc}`，`parent_id` 自引用根节点 `parent_id IS NULL`
- 创建 / 重命名 / 软删除（`deleted_at`）/ 移动（改 `parent_id`，校验新父级权限）
- 目录树按 §3 权限过滤（`none` 不可见）
- Markdown 编辑走 §4.3 协同通道，不再提供单用户 HTTP 保存接口

### 4.3 实时协同模块（F-14 ~ F-18）

- 每文档一个 goroutine hub：维护在线客户端集合、广播 update、路由 awareness
- 并发上限 20（D-10b）
- Y.js update 字节流：前端 `y-websocket` 风格协议，后端不解析，只存储 + 转发
- compact 策略见 D-09a
- Awareness（光标、选区、用户名、颜色）**不落库**，仅在 hub 内存中广播；连接断开即清除

### 4.4 历史版本模块（F-19 ~ F-20）

- 自动触发：每日 02:00 对活跃文档（过去 24h 内有 update 的）生成快照
- 手动触发：`POST /api/documents/:id/snapshots`
- 快照内容：完整 Y.Doc 二进制（D-09b）
- 回溯：用快照覆盖 `doc_states`，清空未合并 updates，向 hub 广播 `reload`（D-09c）
- 保留策略：每个节点最近 50 份（溢出删最旧）

### 4.5 权限管理模块（F-21 ~ F-25）

- Manager: 对节点设置个人/组的权限级别；CRUD 权限组；指定组长；改全局角色
- Group Leader: 仅加/移本组成员
- 所有写操作在涉及的 `groups` / `users` / `node_permissions` 行上 `SELECT … FOR UPDATE`（D-01）

### 4.6 图床模块（F-23）

- drag-drop 上传到当前文档
- `POST /api/uploads` multipart/form-data，字段 `file` + `node_id`
- 限制：图片 ≤ 50MB（jpg / png / gif / webp）；视频 ≤ 500MB（mp4 / webm）
- 落盘 `${UPLOAD_DIR}/<yyyy>/<mm>/<sha256>.<ext>`；sha256 去重
- 访问：`GET /api/uploads/:id` 经后端鉴权 Proxy（D-12）
- 文档中引用形式：`![](/ api/uploads/<id>)`（Markdown 标准语法；前端 TipTap image 扩展拦截 drop/paste 生成该 URL）

### 4.7 通知模块（F-26 ~ F-28）

订阅单位为节点（文档或文件夹）。触发事件：

| 事件 | `event_type` | 触发时机 |
|---|---|---|
| 文档被编辑保存 | `doc_updated` | compact 完成（不是每条 update，避免刷屏） |
| 文档被回溯 | `doc_restored` | 回溯完成 |
| 文档权限变更 | `doc_permission_changed` | 通知订阅者 + 通知被新加入/移除权限的个体 |

- 在线：WS `{type:"notify", ...}` 实时推送
- 离线：写入 `notifications` 表；上线后前端调 `GET /api/notifications?unread=1` 拉取
- 前端可 `PATCH /api/notifications/:id {read: true}` 或 `unsubscribe`

### 4.8 搜索模块（F-30 ~ F-31）

- 入口：`GET /api/search?q=...`
- 建索引：`nodes.tsv` 列由 `title`（标题权重 A）+ `markdown_plain`（正文权重 B）组合生成；`markdown_plain` 在 compact 时从 Y.Doc 抽取纯文本并写回 `doc_states`
- 分词：`zhparser` + `pg_trgm`（后者用于用户名/标题模糊）
- 结果：标题 + 高亮摘要（`ts_headline`）+ 节点路径；服务端在 SQL 层内联权限过滤（子查询只返回当前用户 `!= none` 的 `node_id`）

---

## 5. Feature List

优先级：**P0** = MVP，**P1** = MVP 后首批，**P2** = 远期

| ID | 模块 | 功能 | 优先级 |
|----|------|------|--------|
| F-01 | 鉴权 | 注册（邮箱+密码） | P0 |
| F-02 | 鉴权 | 登录 + 签发 access/refresh | P0 |
| F-03 | 鉴权 | 刷新 token（一次性轮换 + 复用检测） | P0 |
| F-04 | 鉴权 | 登出（吊销 refresh） | P0 |
| F-05 | 鉴权 | 修改密码 | P0 |
| F-06 | 鉴权 | Manager 强制用户下线 | P0 |
| F-07 | 鉴权 | 获取 / 更新当前用户资料 | P0 |
| F-08 | 目录 | 创建文件夹 / 文档 | P0 |
| F-09 | 目录 | 重命名 | P0 |
| F-10 | 目录 | 软删除 | P0 |
| F-11 | 目录 | 移动（改 parent_id） | P0 |
| F-12 | 目录 | 目录树按权限过滤展示 | P0 |
| F-13 | 目录 | 恢复软删除节点（30 天内） | P1 |
| F-14 | 编辑器 | TipTap + Markdown 基础编辑（标题/粗斜/列表/表格/代码/引用/任务列表） | P0 |
| F-15 | 协同 | WS 握手 + 加入文档 hub + 20 并发上限 | P0 |
| F-16 | 协同 | Y.js update 广播 + 落库 + compact | P0 |
| F-17 | 协同 | Y.js Awareness 广播（光标/颜色/用户名） | P0 |
| F-18 | 协同 | readable 用户只观察不发送 update | P0 |
| F-19 | 历史 | 自动+手动生成 Y.Doc 快照 | P0 |
| F-20 | 历史 | 查看列表 + 回溯 + 广播 reload | P0 |
| F-21 | 权限 | 节点设置个人 / 组的权限级别 | P0 |
| F-22 | 权限 | 权限组 CRUD + 指定组长 | P0 |
| F-23 | 图床 | 图片/视频 drag-drop 上传 + 鉴权 Proxy 访问 | P0 |
| F-24 | 组织 | Group Leader 管理本组成员 | P0 |
| F-25 | 组织 | Manager 修改用户全局角色 | P0 |
| F-26 | 通知 | 订阅 / 取消订阅节点 | P0 |
| F-27 | 通知 | 事件触发 + WS 推送 + 离线拉取 | P0 |
| F-28 | 通知 | 标记已读 | P0 |
| F-30 | 搜索 | 标题 + 正文全文搜索（含中文） | P0 |
| F-31 | 搜索 | 结果按权限过滤 + ts_headline 摘要 | P0 |
| F-40 | 前端 | 登录 / 注册页 | P0 |
| F-41 | 前端 | 主界面 Sidebar + 顶部栏 + 主区域布局 | P0 |
| F-42 | 前端 | 目录树（折叠/拖拽/右键菜单） | P0 |
| F-43 | 前端 | 编辑页多人光标 + 在线人数 | P0 |
| F-44 | 前端 | 历史版本面板 | P0 |
| F-45 | 前端 | 管理后台页（仅 Manager 可见） | P0 |
| F-46 | 前端 | 通知铃铛面板 | P0 |
| F-47 | 部署 | docker-compose 一键启动全栈 | P0 |

---

## 6. Data Schema

> PostgreSQL DDL 的规范性版本见 `docs/epics/infra.md`。以下为 Go 结构体视角的最终形态。时间一律 `time.Time`（UTC）；ID 一律 `uuid.UUID`。

### 6.1 `users`

```go
type User struct {
    ID           uuid.UUID `db:"id"`
    Email        string    `db:"email"`         // 唯一索引，citext 小写
    PasswordHash string    `db:"password_hash"` // bcrypt cost=12
    DisplayName  string    `db:"display_name"`
    Role         string    `db:"role"`          // 'manager' | 'member'
    TokenVersion int64     `db:"token_version"` // D-04 强制下线
    Locale       string    `db:"locale"`        // 'en' | 'zh'
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### 6.2 `groups` + `group_members`

```go
type Group struct {
    ID        uuid.UUID
    Name      string     // 唯一
    LeaderID  uuid.UUID  // → users.id
    CreatedAt time.Time
    UpdatedAt time.Time
}

type GroupMember struct {
    GroupID  uuid.UUID
    UserID   uuid.UUID
    JoinedAt time.Time
}
// PK: (group_id, user_id)
```

### 6.3 `refresh_tokens`（D-03）

```go
type RefreshToken struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    TokenHash   []byte     // sha256(raw_token)
    ExpiresAt   time.Time
    RevokedAt   *time.Time
    ReplacedBy  *uuid.UUID // 轮换链；复用检测依据
    CreatedAt   time.Time
}
```

### 6.4 `nodes`（D-05）

```go
type Node struct {
    ID        uuid.UUID
    ParentID  *uuid.UUID  // NULL = 根
    Kind      string      // 'folder' | 'doc'
    Title     string
    OwnerID   uuid.UUID   // 创建者
    DeletedAt *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
    TSV       string      // tsvector，通过 trigger 维护；zhparser 分词
}
```

索引：`(parent_id)`、`(deleted_at) WHERE deleted_at IS NULL`、`GIN(tsv)`。

### 6.5 `node_permissions`（D-07）

```go
type NodePermission struct {
    ID          uuid.UUID
    NodeID      uuid.UUID
    SubjectType string      // 'user' | 'group'
    SubjectID   uuid.UUID
    Level       string      // 'manage' | 'edit' | 'readable' | 'none'
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
// UNIQUE(node_id, subject_type, subject_id)
```

### 6.6 `doc_states`

```go
type DocState struct {
    NodeID          uuid.UUID  // PK
    YDocState       []byte     // 最近一次 compact 后的完整 Y.Doc 二进制
    Version         int64      // compact 次数
    MarkdownPlain   string     // compact 时抽取的纯文本（给搜索用）
    LastCompactedAt time.Time
}
```

### 6.7 `doc_updates`（D-09a）

```go
type DocUpdate struct {
    ID        int64      // bigserial，单调递增
    NodeID    uuid.UUID
    Update    []byte     // Y.js binary update
    ClientID  string     // 前端生成的稳定 ID
    CreatedAt time.Time
}
// 索引 (node_id, id)
```

### 6.8 `doc_snapshots`（D-09b）

```go
type DocSnapshot struct {
    ID            uuid.UUID
    NodeID        uuid.UUID
    YDocState     []byte     // 完整二进制
    Title         string     // 生成时刻的 node.title 冻结
    CreatedBy     uuid.UUID
    TriggerReason string     // 'manual' | 'scheduled_daily'
    CreatedAt     time.Time
}
// 索引 (node_id, created_at DESC)，每个 node_id 最多保留 50 条
```

### 6.9 `uploads`

```go
type Upload struct {
    ID         uuid.UUID
    OwnerID    uuid.UUID
    NodeID     *uuid.UUID  // 引用文档；允许 NULL（草稿期间上传）
    Filename   string
    StoredPath string      // 相对 UPLOAD_DIR
    MimeType   string
    SizeBytes  int64
    SHA256     string      // 去重键
    CreatedAt  time.Time
}
// UNIQUE(sha256)
```

### 6.10 `subscriptions`

```go
type Subscription struct {
    UserID    uuid.UUID
    NodeID    uuid.UUID
    CreatedAt time.Time
}
// PK: (user_id, node_id)
```

### 6.11 `notifications`（D-11）

```go
type Notification struct {
    ID        uuid.UUID
    UserID    uuid.UUID   // 接收人
    NodeID    uuid.UUID
    EventType string      // 'doc_updated' | 'doc_restored' | 'doc_permission_changed'
    Payload   []byte      // jsonb（actor_id、old_level、new_level 等事件相关字段）
    ReadAt    *time.Time
    CreatedAt time.Time
}
// 索引 (user_id, read_at NULLS FIRST, created_at DESC)
```

---

## 7. Logic Flow（核心业务流程伪代码）

### 7.1 登录 + 双 token 签发（F-02）

```text
POST /api/auth/login { email, password }
1. email → 小写查 users；不存在 → 401 invalid_credentials
2. bcrypt.Compare；失败 → 401（统一错误，不泄漏）
3. access = JWT{sub, tv=user.token_version, exp=+15m, HS256}
4. refresh_raw = crypto.Rand(32)
   insert refresh_tokens{token_hash=sha256(refresh_raw), exp=+7d}
5. 返回 { access, refresh: refresh_raw, user }
```

### 7.2 Token 刷新 + 复用检测（F-03, D-03）

```text
POST /api/auth/refresh { refresh }
1. sha256(refresh) 查 refresh_tokens
2. 不存在 → 401
3. 存在但 revoked_at != null OR replaced_by != null:
     # 泄漏检测：攻击者用了旧 token
     UPDATE refresh_tokens SET revoked_at=now() WHERE user_id=row.user_id
     UPDATE users SET token_version = token_version + 1 WHERE id=row.user_id  # 踢掉所有 access
     → 401 token_reused
4. 正常路径：
     new_raw = crypto.Rand(32)
     insert new_row
     UPDATE old_row SET replaced_by = new_row.id
     new_access = JWT{...}
     → { access: new_access, refresh: new_raw }
```

### 7.3 Manager 强制下线（F-06, D-04）

```text
POST /api/admin/users/:id/force-logout  (require role=manager)
1. BEGIN
2. UPDATE users SET token_version = token_version + 1 WHERE id = :id
3. UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = :id AND revoked_at IS NULL
4. COMMIT
5. 通过 hub 向该用户所有在线 WS 连接推送 {type:"force_logout"}；前端收到后清 storage + 跳登录页
```

### 7.4 协同 WS 生命周期（F-15 ~ F-18）

```text
Client → WS GET /api/ws?doc=<node_id>
  Header Sec-WebSocket-Protocol: access.<jwt>

Server Upgrade:
  1. 解 JWT，校验 tv，识别 user
  2. level = ResolvePermission(user, node)
     if level == "none": close 4403
  3. hub = getOrCreateHub(node_id)
  4. if hub.size() >= 20: send {type:"join_rejected"}; close
  5. hub.register(conn{user, level})
  6. send {type:"sync_step1", state_vector: Y.encodeStateVector(hub.doc)}  # 首次协同
  7. 客户端发送 sync_step2 / update → 服务端:
       if level in {"edit","manage"}:
         append doc_updates
         broadcast to other conns
         # 不解析 update 内部结构
       else:  # readable → 丢弃 + log
         no-op
  8. awareness 消息 → 任意 level 均允许，仅转发不存储
  9. 断连：hub.unregister；若 size==0 → 延迟 60s 销毁 hub
```

### 7.5 Compact 调度（D-09a）

```text
后台 goroutine 每 60s：
  SELECT node_id, count(*) FROM doc_updates GROUP BY node_id HAVING count >= 200
  UNION
  SELECT node_id FROM doc_states WHERE last_compacted_at < now() - interval '30 min'
     AND EXISTS (SELECT 1 FROM doc_updates WHERE node_id = doc_states.node_id)

对每个命中节点：
  BEGIN
  SELECT ydoc_state FROM doc_states FOR UPDATE
  SELECT update FROM doc_updates WHERE node_id=? ORDER BY id
  merged = Y.mergeUpdates([ydoc_state, ...updates])
  plain  = extractText(merged)                        # 喂搜索
  UPDATE doc_states SET ydoc_state=merged,
                        markdown_plain=plain,
                        version=version+1,
                        last_compacted_at=now()
  DELETE FROM doc_updates WHERE node_id=? AND id <= :max_seen_id
  COMMIT

  触发 notify({event:"doc_updated", node_id, actor_ids=DISTINCT(update.client_user)})
```

### 7.6 历史版本回溯（F-20, D-09c）

```text
POST /api/documents/:id/snapshots/:sid/restore  (require manage)
  BEGIN
  SELECT ydoc_state FROM doc_snapshots WHERE id=:sid
  UPDATE doc_states SET ydoc_state=snapshot.ydoc_state, version=version+1, last_compacted_at=now()
  DELETE FROM doc_updates WHERE node_id=:id
  COMMIT

  hub.broadcast({type:"reload"})
  notify({event:"doc_restored", node_id=:id, snapshot_id=:sid})
```

### 7.7 图床访问鉴权（F-23, D-12）

```text
GET /api/uploads/:id
  1. JWT 校验
  2. SELECT u.*, u.node_id FROM uploads u WHERE id=:id
  3. if u.node_id == null:
        # 草稿上传；只有 owner 可访问
        if user != u.owner → 404
     else:
        level = ResolvePermission(user, u.node_id)
        if level == "none" → 404
  4. stream file from disk, set Content-Type, Content-Length
  5. 可选：ETag = sha256
```

### 7.8 权限变更（F-21, D-01）

```text
PUT /api/nodes/:id/permissions { permissions: [{subject_type, subject_id, level}, ...] }  (require manage)
  BEGIN
  SELECT id FROM nodes WHERE id=:id FOR UPDATE
  DELETE FROM node_permissions WHERE node_id=:id
  INSERT new rows
  COMMIT

  发 notifications:
    - 订阅者 → event=doc_permission_changed
    - 新增/移除/变更 level 的 user/group 成员 → 同事件类型
```

---

## 8. API 清单

> 统一信封（见 D-11 的通知也纳入）：
>
> ```json
> {"success": true, "data": {...}, "error": null}
> {"success": false, "data": null, "error": {"code": "...", "message": "..."}}
> ```

### 8.1 REST

| Method | Path | 鉴权 | 最低权限 | 说明 |
|---|---|---|---|---|
| `POST` | `/api/auth/register` | — | — | F-01 |
| `POST` | `/api/auth/login` | — | — | F-02 |
| `POST` | `/api/auth/refresh` | — | — | F-03 |
| `POST` | `/api/auth/logout` | JWT | — | F-04 |
| `GET` | `/api/me` | JWT | — | F-07 |
| `PATCH` | `/api/me` | JWT | — | F-07 |
| `POST` | `/api/me/password` | JWT | — | F-05 |
| `POST` | `/api/admin/users/:id/force-logout` | JWT | Manager | F-06 |
| `PATCH` | `/api/admin/users/:id/role` | JWT | Manager | F-25 |
| `GET` | `/api/nodes?parent=<id|null>` | JWT | `readable` (filter) | F-12 |
| `POST` | `/api/nodes` | JWT | parent ≥ `edit` | F-08 |
| `GET` | `/api/nodes/:id` | JWT | ≥ `readable` | F-12 |
| `PATCH` | `/api/nodes/:id` | JWT | ≥ `edit`（改名）/ ≥ `manage`（移动） | F-09, F-11 |
| `DELETE` | `/api/nodes/:id` | JWT | ≥ `manage` | F-10 |
| `POST` | `/api/nodes/:id/restore` | JWT | ≥ `manage` | F-13 |
| `GET` | `/api/nodes/:id/permissions` | JWT | ≥ `manage` | F-21 |
| `PUT` | `/api/nodes/:id/permissions` | JWT | ≥ `manage` | F-21 |
| `GET` | `/api/documents/:id/snapshots` | JWT | ≥ `readable` | F-20 |
| `POST` | `/api/documents/:id/snapshots` | JWT | ≥ `edit` | F-19 |
| `POST` | `/api/documents/:id/snapshots/:sid/restore` | JWT | ≥ `manage` | F-20 |
| `GET` | `/api/admin/groups` | JWT | Manager | F-22 |
| `POST` | `/api/admin/groups` | JWT | Manager | F-22 |
| `PATCH` | `/api/admin/groups/:id` | JWT | Manager | F-22 |
| `DELETE` | `/api/admin/groups/:id` | JWT | Manager | F-22 |
| `POST` | `/api/admin/groups/:id/members` | JWT | Manager 或 该组 Leader | F-24 |
| `DELETE` | `/api/admin/groups/:id/members/:uid` | JWT | Manager 或 该组 Leader | F-24 |
| `POST` | `/api/uploads` | JWT | 目标 node ≥ `edit` | F-23 |
| `GET` | `/api/uploads/:id` | JWT | 归属 node ≥ `readable` | F-23 |
| `GET` | `/api/subscriptions` | JWT | — | F-26 |
| `POST` | `/api/nodes/:id/subscribe` | JWT | ≥ `readable` | F-26 |
| `DELETE` | `/api/nodes/:id/subscribe` | JWT | — | F-26 |
| `GET` | `/api/notifications?unread=0|1` | JWT | — | F-27 |
| `PATCH` | `/api/notifications/:id` | JWT | 所属本人 | F-28 |
| `GET` | `/api/search?q=` | JWT | 结果 filter ≥ `readable` | F-30 |
| `GET` | `/ping` | — | — | 健康检查 |

### 8.2 WebSocket

**单一端点**：`GET /api/ws?doc=<node_id>`

鉴权：首选 `Sec-WebSocket-Protocol: access.<jwt>`（因浏览器 WS 不能自定义 header）

消息 envelope：

```json
{ "type": "<kind>", "payload": {...} }
```

| 方向 | `type` | 说明 |
|---|---|---|
| C→S | `sync_step2` / `update` | Y.js 协议；payload 含 base64 bytes |
| C→S | `awareness` | 光标/选区 |
| S→C | `sync_step1` / `update` | 同上反向 |
| S→C | `awareness` | 广播他人 |
| S→C | `join_rejected` | `reason ∈ {doc_full, permission_denied}` |
| S→C | `reload` | 回溯后要求前端重置 Y.Doc |
| S→C | `notify` | 订阅通知（含权限变更导致的被踢） |
| S→C | `force_logout` | Manager 强制下线 |
| S→C | `auth_renew_required` | 仅权限敏感场景需要重验 |

---

## 9. 前端页面结构（对齐 v0.4 §6）

| 路由 | 页面 | 主要组件 | 关联 Feature |
|---|---|---|---|
| `/login` `/register` | 鉴权页 | 登录 / 注册表单 | F-40 |
| `/` | 主界面 | 左 Sidebar（目录树） + 顶部栏（搜索、通知铃铛、用户头像） + 右主区域 | F-41, F-42, F-46 |
| `/doc/:id`（嵌入主区域） | 编辑页 | TipTap 编辑器 + 在线人数 + 光标叠加 + 右侧历史版本抽屉 | F-14, F-43, F-44 |
| `/admin/users` `/admin/groups` `/admin/nodes/:id/permissions` | 管理后台 | 仅 Manager 可见；用户列表、组管理、权限矩阵 | F-45 |
| 顶部铃铛抽屉 | 通知面板 | 订阅事件列表、在此取消订阅 | F-46 |

---

## 10. 非功能性要求

| 项目 | 要求 |
|---|---|
| 协同延迟 | 同局域网 p95 < 100ms（WS update 往返） |
| 单文档并发 | ≤ 20 人（D-10b） |
| 列表/搜索接口 | p95 < 200ms（数据量 ≤ 1 万节点） |
| 文件大小 | 图片 ≤ 50MB；视频 ≤ 500MB |
| 密码 | bcrypt cost ≥ 12 |
| JWT 密钥 | `JWT_SECRET` 环境变量，启动缺失即拒启 |
| 文件访问 | 一律经后端鉴权 Proxy，禁暴露静态路径（D-12） |
| 日志 | 结构化 JSON（`request_id`, `user_id`, `node_id`, `latency_ms`） |
| 测试覆盖 | 后端 handler + service 单元测试 ≥ 80%；权限解析算法 ≥ 95% |
| 部署 | `docker-compose up` 三服务联调启动（frontend / backend / postgres） |

---

## 11. 暂不实现（对齐 v0.4 §8）

- 文档内评论
- 文档分享链接（匿名访问）
- 富文本格式（仅 Markdown）
- 分片上传
- 文档模板
- 操作日志 / 审计

---

## 12. Definition of Done（全局）

一个 Epic 完成 = 以下全部：

1. 对应 Feature 的 REST / WS 路由已注册并通过集成测试
2. 数据模型迁移脚本（`backend/migrations/`）落库，可一键 rollback
3. 单元测试覆盖率达标（§10）；权限解析算法（§3.2）必须 100% 分支覆盖
4. 前端对应页面组件打通 happy path + 一个典型异常路径
5. `docker-compose up` 启动后可端到端演练该 Epic 的功能
6. 本文件同步：Feature 状态 + 若有决策变更须新增 Changelog
7. 提交遵循 `feat: / fix: / chore: / docs:` 约定

---

## 13. Roadmap

| 阶段 | Epic | 对应 Features | 详细设计 |
|---|---|---|---|
| 1 | **Infra**（基础设施） | F-47 + 所有 P0 的运行环境 | `docs/epics/infra.md` ← 进行中 |
| 2 | Auth（鉴权） | F-01 ~ F-07 | `docs/epics/auth.md` |
| 3 | NodeTree（目录树 + 权限解析） | F-08 ~ F-13, F-21 ~ F-22, F-24 ~ F-25 | `docs/epics/node-tree.md` |
| 4 | Collab（协同编辑） | F-14 ~ F-18 | `docs/epics/collab.md` |
| 5 | History（版本） | F-19 ~ F-20 | `docs/epics/history.md` |
| 6 | Uploads（图床） | F-23 | `docs/epics/uploads.md` |
| 7 | Notifications（通知） | F-26 ~ F-28 | `docs/epics/notifications.md` |
| 8 | Search（搜索） | F-30 ~ F-31 | `docs/epics/search.md` |
| 9 | Frontend Shell（前端总装） | F-40 ~ F-46 | `docs/epics/frontend-shell.md` |

阶段之间允许前端/后端在同一阶段内部并行，但跨阶段严格顺序。

---

## 14. 变更日志

| 版本 | 日期 | 变更 | 维护者 |
|---|---|---|---|
| v0.1 | 2026-04-16 | 初始化骨架（MVP 猜测范围） | CC |
| v1.0 | 2026-04-16 | 对齐产品需求 v0.4；冻结 D-01 ~ D-13（原 A-1 ~ A-11 的规范化版本）；删除"架构评审待定项"转为"技术约束"章节；完成 Feature、Schema、Logic Flow、API、前端结构、NFR 全量规范化 | CC |
