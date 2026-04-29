# Epic: Infra（基础设施）

> **状态**: 设计冻结，可下发 Codex · 2026-04-16
> **上游**: [`docs/requirements.md`](../requirements.md) v1.0
> **范围**: 仓库目录骨架、PostgreSQL Schema 与扩展、Go 后端分层骨架、WebSocket Hub 基础、前端工程骨架、配置与日志、docker-compose 全栈联调
> **不包含**: 任何业务 handler 的具体逻辑（登录、权限解析、协同等——留给后续 Epic）

本 Epic 产出的是"能启动但还没功能"的骨架：
- 三个服务能起来、能互联
- 数据库迁移能跑通且可回滚
- `/ping` 通
- WS 可握手但不处理消息
- 前端能跑起来并能登录页框架渲染（不接后端业务）

---

## 1. 目录布局

```text
FPGWiki/
├── docs/
│   ├── requirements.md             # 本项目真源
│   └── epics/
│       └── infra.md                # 本文件
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go             # 入口：加载 config、连 DB、启 HTTP
│   ├── internal/
│   │   ├── config/                 # envconfig 读配置；启动缺项即 fatal
│   │   ├── db/                     # pgxpool + 健康检查
│   │   ├── logger/                 # zerolog 结构化日志
│   │   ├── httpserver/
│   │   │   ├── router.go           # gin 路由装配
│   │   │   ├── middleware/         # requestID / logger / recover / auth(占位)
│   │   │   └── handlers/           # 每个业务模块一个子包（当前仅 health.go）
│   │   ├── ws/
│   │   │   ├── hub.go              # 文档级 Hub 骨架（见 §6）
│   │   │   ├── manager.go          # HubManager: node_id → *Hub
│   │   │   └── ws.go               # upgrader + 握手入口（当前仅 echo 框架）
│   │   ├── models/                 # Go struct 对应表（见 requirements §6）
│   │   ├── repository/             # SQL 层，按表划分（users_repo.go 等）
│   │   └── service/                # 业务服务（按 Epic 填充）
│   ├── migrations/                 # golang-migrate 文件
│   │   ├── 0001_init_extensions.up.sql
│   │   ├── 0001_init_extensions.down.sql
│   │   ├── 0002_users_groups.up.sql
│   │   ├── 0002_users_groups.down.sql
│   │   ├── 0003_nodes_permissions.up.sql
│   │   ├── 0003_nodes_permissions.down.sql
│   │   ├── 0004_docs_snapshots.up.sql
│   │   ├── 0004_docs_snapshots.down.sql
│   │   ├── 0005_uploads.up.sql
│   │   ├── 0005_uploads.down.sql
│   │   └── 0006_subs_notifications.up.sql
│   ├── Dockerfile
│   ├── go.mod
│   └── .env.example
├── frontend/
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx                 # 路由外壳
│   │   ├── pages/
│   │   │   ├── LoginPage.tsx       # 占位空壳
│   │   │   ├── RegisterPage.tsx
│   │   │   ├── Home.tsx
│   │   │   ├── DocPage.tsx
│   │   │   ├── AdminUsersPage.tsx
│   │   │   └── AdminGroupsPage.tsx
│   │   ├── layouts/
│   │   │   ├── MainShell.tsx       # Sidebar + TopBar + Outlet（占位）
│   │   │   └── AdminShell.tsx      # 仅 Manager（占位）
│   │   ├── auth/
│   │   │   ├── RequireAuth.tsx     # localStorage 占位守卫（T-11）
│   │   │   ├── RequireRole.tsx     # localStorage 角色守卫（T-11）
│   │   │   └── AuthContext.tsx     # JWT 状态 + 刷新 + 登出（T-12）
│   │   ├── api/
│   │   │   ├── client.ts           # fetch 封装；自动注入 JWT；401 触发 refresh
│   │   │   └── endpoints.ts        # 常量
│   │   ├── ws/
│   │   │   └── WSClient.ts         # 单例封装，后续 Epic 接入 y-websocket 风格协议
│   │   └── styles/
│   ├── public/
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── Dockerfile
│   └── .env.example
├── docker-compose.yml
├── Makefile                        # 常用命令封装
└── README.md
```

**约束**：

- 单文件 ≤ 800 行；handler 文件 ≤ 400 行
- `internal/` 禁止被外部仓库 import，保证业务逻辑封装
- SQL 放在 `migrations/` 与 repository 文件的 const 字符串中；禁止散落在 handler 里

---

## 2. Runtime 依赖与版本

| 组件 | 版本（下限） | 来源 |
|---|---|---|
| Go | 1.25 | Dockerfile `golang:1.25-alpine` |
| PostgreSQL | 16 | `postgres:16-alpine` + 挂载 zhparser 插件 |
| zhparser | 2.2 | `pgpool/zhparser:16-alpine` 或自构 image |
| Node | 20 LTS | `node:20-alpine` |
| pnpm | 9.x | corepack |

zhparser 官方无 alpine 版本预编译；**采用方案**：在 `docker/postgres/Dockerfile` 里基于 `postgres:16` 安装 `postgresql-16-zhparser`（Debian 系 APT 包 `postgresql-16-zhparser` 来自 amutu/zhparser 仓库镜像或官方 pgxn）。

---

## 3. PostgreSQL Schema（DDL 定稿）

### 3.1 扩展与自定义配置（0001）

```sql
-- 0001_init_extensions.up.sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;      -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;        -- 邮箱大小写无关
CREATE EXTENSION IF NOT EXISTS pg_trgm;       -- 模糊匹配
CREATE EXTENSION IF NOT EXISTS zhparser;      -- 中文分词

CREATE TEXT SEARCH CONFIGURATION chinese (PARSER = zhparser);
ALTER  TEXT SEARCH CONFIGURATION chinese
  ADD MAPPING FOR n,v,a,i,e,l,j WITH simple;
```

Down：`DROP TEXT SEARCH CONFIGURATION chinese; DROP EXTENSION ...`（按逆序）。

### 3.2 users / groups / refresh_tokens（0002）

```sql
CREATE TABLE users (
  id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  email          citext      NOT NULL UNIQUE,
  password_hash  text        NOT NULL,
  display_name   text        NOT NULL,
  role           text        NOT NULL CHECK (role IN ('manager','member')),
  token_version  bigint      NOT NULL DEFAULT 0,
  locale         text        NOT NULL DEFAULT 'zh' CHECK (locale IN ('en','zh')),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE groups (
  id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  name       text        NOT NULL UNIQUE,
  leader_id  uuid        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE group_members (
  group_id  uuid        NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  user_id   uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  joined_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (group_id, user_id)
);
CREATE INDEX group_members_user_idx ON group_members(user_id);

CREATE TABLE refresh_tokens (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  bytea       NOT NULL,
  expires_at  timestamptz NOT NULL,
  revoked_at  timestamptz,
  replaced_by uuid        REFERENCES refresh_tokens(id),
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX refresh_tokens_hash_idx ON refresh_tokens(token_hash);
CREATE INDEX        refresh_tokens_user_idx ON refresh_tokens(user_id);
```

### 3.3 nodes + node_permissions + tsv 触发器（0003）

```sql
CREATE TABLE nodes (
  id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_id  uuid        REFERENCES nodes(id) ON DELETE RESTRICT,
  kind       text        NOT NULL CHECK (kind IN ('folder','doc')),
  title      text        NOT NULL,
  owner_id   uuid        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  tsv        tsvector
);
CREATE INDEX nodes_parent_idx         ON nodes(parent_id);
CREATE INDEX nodes_live_idx           ON nodes(parent_id) WHERE deleted_at IS NULL;
CREATE INDEX nodes_tsv_gin_idx        ON nodes USING GIN (tsv);
CREATE INDEX nodes_title_trgm_idx     ON nodes USING GIN (title gin_trgm_ops);

-- tsv 维护：初始仅对 title；markdown_plain 在 doc_states 里单独建 tsv 索引
-- 方案：nodes.tsv = to_tsvector('chinese', title)；搜索时 UNION doc_states.tsv
CREATE FUNCTION nodes_tsv_trigger() RETURNS trigger AS $$
BEGIN
  NEW.tsv := to_tsvector('chinese', coalesce(NEW.title, ''));
  NEW.updated_at := now();
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER nodes_tsv_upd
BEFORE INSERT OR UPDATE OF title ON nodes
FOR EACH ROW EXECUTE FUNCTION nodes_tsv_trigger();

CREATE TABLE node_permissions (
  id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id      uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  subject_type text        NOT NULL CHECK (subject_type IN ('user','group')),
  subject_id   uuid        NOT NULL,
  level        text        NOT NULL CHECK (level IN ('manage','edit','readable','none')),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (node_id, subject_type, subject_id)
);
CREATE INDEX node_permissions_node_idx    ON node_permissions(node_id);
CREATE INDEX node_permissions_subject_idx ON node_permissions(subject_type, subject_id);
```

**权限解析工具 SQL**（给 repository 用）：

```sql
-- 获取 node 的祖先链（含自身），从自身到根
WITH RECURSIVE ancestors AS (
  SELECT id, parent_id, 0 AS depth FROM nodes WHERE id = $1
  UNION ALL
  SELECT n.id, n.parent_id, a.depth + 1
  FROM nodes n JOIN ancestors a ON n.id = a.parent_id
)
SELECT a.id, a.depth,
       p.subject_type, p.subject_id, p.level
FROM ancestors a
LEFT JOIN node_permissions p ON p.node_id = a.id
ORDER BY a.depth;
```

Go 侧按 `depth` 升序扫描、遇到第一个存在权限行的 depth 即为 effective node（其余 depth 的行忽略）。

### 3.4 doc_states / doc_updates / doc_snapshots（0004）

```sql
CREATE TABLE doc_states (
  node_id            uuid        PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
  ydoc_state         bytea       NOT NULL,
  version            bigint      NOT NULL DEFAULT 0,
  markdown_plain     text        NOT NULL DEFAULT '',
  markdown_plain_tsv tsvector,
  last_compacted_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX doc_states_tsv_gin_idx ON doc_states USING GIN (markdown_plain_tsv);

CREATE FUNCTION doc_states_tsv_trigger() RETURNS trigger AS $$
BEGIN
  NEW.markdown_plain_tsv := to_tsvector('chinese', coalesce(NEW.markdown_plain, ''));
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER doc_states_tsv_upd
BEFORE INSERT OR UPDATE OF markdown_plain ON doc_states
FOR EACH ROW EXECUTE FUNCTION doc_states_tsv_trigger();

CREATE TABLE doc_updates (
  id         bigserial   PRIMARY KEY,
  node_id    uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  update     bytea       NOT NULL,
  client_id  text        NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX doc_updates_node_idx ON doc_updates(node_id, id);

CREATE TABLE doc_snapshots (
  id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  node_id         uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  ydoc_state      bytea       NOT NULL,
  title           text        NOT NULL,
  created_by      uuid        NOT NULL REFERENCES users(id),
  trigger_reason  text        NOT NULL CHECK (trigger_reason IN ('manual','scheduled_daily')),
  created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX doc_snapshots_node_idx ON doc_snapshots(node_id, created_at DESC);
```

快照保留 50 条 = 在 `doc_snapshots` 的 service 层插入后执行：

```sql
DELETE FROM doc_snapshots
WHERE node_id = $1
  AND id NOT IN (
    SELECT id FROM doc_snapshots
    WHERE node_id = $1
    ORDER BY created_at DESC
    LIMIT 50
  );
```

### 3.5 uploads（0005）

```sql
CREATE TABLE uploads (
  id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id    uuid        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  node_id     uuid        REFERENCES nodes(id) ON DELETE SET NULL,
  filename    text        NOT NULL,
  stored_path text        NOT NULL,
  mime_type   text        NOT NULL,
  size_bytes  bigint      NOT NULL CHECK (size_bytes > 0),
  sha256      text        NOT NULL UNIQUE,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX uploads_node_idx  ON uploads(node_id) WHERE node_id IS NOT NULL;
CREATE INDEX uploads_owner_idx ON uploads(owner_id);
```

### 3.6 subscriptions / notifications（0006）

```sql
CREATE TABLE subscriptions (
  user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  node_id    uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, node_id)
);
CREATE INDEX subscriptions_node_idx ON subscriptions(node_id);

CREATE TABLE notifications (
  id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  node_id    uuid        NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  event_type text        NOT NULL,
  payload    jsonb       NOT NULL DEFAULT '{}'::jsonb,
  read_at    timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_user_unread_idx
  ON notifications(user_id, created_at DESC)
  WHERE read_at IS NULL;
```

---

## 4. Go 后端骨架

### 4.1 配置（`internal/config/config.go`）

用 `envconfig`。**启动时缺字段即 `log.Fatal`**：

```go
type Config struct {
    AppEnv        string        `envconfig:"APP_ENV"        default:"dev"`
    HTTPAddr      string        `envconfig:"HTTP_ADDR"      default:":8080"`

    PostgresDSN   string        `envconfig:"POSTGRES_DSN"   required:"true"`
    PostgresMaxConns int32      `envconfig:"POSTGRES_MAX_CONNS" default:"20"`

    JWTSecret     string        `envconfig:"JWT_SECRET"     required:"true"`
    JWTAccessTTL  time.Duration `envconfig:"JWT_ACCESS_TTL" default:"15m"`
    JWTRefreshTTL time.Duration `envconfig:"JWT_REFRESH_TTL" default:"168h"`

    UploadDir     string        `envconfig:"UPLOAD_DIR"     default:"/data/uploads"`
    MaxImageBytes int64         `envconfig:"MAX_IMAGE_BYTES" default:"52428800"`    // 50MB
    MaxVideoBytes int64         `envconfig:"MAX_VIDEO_BYTES" default:"524288000"`   // 500MB

    LogLevel      string        `envconfig:"LOG_LEVEL"      default:"info"`
}
```

### 4.2 DB 连接（`internal/db/pg.go`）

- `pgxpool.New(ctx, cfg.PostgresDSN)`
- `MaxConns = cfg.PostgresMaxConns`
- 启动后 `SELECT 1` 探活；失败 fatal
- 暴露 `HealthCheck(ctx) error` 供 `/ping` 调用

### 4.3 日志（`internal/logger/logger.go`）

zerolog，JSON 输出，字段：`time`、`level`、`msg`、`request_id`、`user_id`、`node_id`、`latency_ms`。

### 4.4 中间件管道顺序

```text
Recover → RequestID → Logger → CORS → Auth(optional/required)
```

`Auth(optional)`：有 token 就解，无也放过（用于 `/api/me` 等 handler 自己决定）。
`Auth(required)`：无 token → 401。

Auth 中间件在 Infra Epic 里**仅落占位**，签名:

```go
func RequireAuth(cfg Config) gin.HandlerFunc  // 占位：解析 header，当前直接 next
func OptionalAuth(cfg Config) gin.HandlerFunc
```

真实实现由 Auth Epic 填。

### 4.5 路由装配

```go
// internal/httpserver/router.go
r := gin.New()
r.Use(middleware.Recover, middleware.RequestID, middleware.Logger(log), middleware.CORS(cfg))

r.GET("/ping", handlers.Health(db))

api := r.Group("/api")
// 后续 Epic 在此注册
// api.POST("/auth/login", ...)
// api.GET ("/me", middleware.RequireAuth(cfg), ...)

r.GET("/api/ws", ws.Handler(cfg, hubManager))  // Infra 阶段：握手成功但收到消息直接回 "not_implemented"
```

### 4.6 迁移工具

选型：**golang-migrate**（`github.com/golang-migrate/migrate/v4`）。

- Makefile 任务：
  - `make migrate-up`
  - `make migrate-down`
  - `make migrate-new name=xxx`
- 后端启动时自动执行 `migrate.Up`（dev 环境），生产需显式 flag `-migrate`
- 回滚测试：每个 up/down 对必须在 CI 中走一遍"up → down → up"

---

## 5. Hub 与 WebSocket 骨架

> Infra Epic 只搭骨架 + 握手；Y.js 协议具体消息处理在 Collab Epic。

### 5.1 数据结构

```go
// internal/ws/hub.go
type Client struct {
    ID     string        // 随机生成
    UserID uuid.UUID
    Level  string        // manage | edit | readable
    Conn   *websocket.Conn
    Send   chan []byte
}

type Hub struct {
    NodeID     uuid.UUID
    clients    map[string]*Client    // by client.ID
    register   chan *Client
    unregister chan *Client
    inbound    chan inboundMsg       // 来自任意 client，在 Hub goroutine 内串行处理
    shutdown   chan struct{}
}

type inboundMsg struct {
    Client  *Client
    Payload []byte
}

// internal/ws/manager.go
type HubManager struct {
    mu   sync.Mutex
    hubs map[uuid.UUID]*Hub   // node_id → Hub
}

func (m *HubManager) GetOrCreate(nodeID uuid.UUID) *Hub
func (m *HubManager) Release(nodeID uuid.UUID)   // 最后一个 client 断开 60s 后调用
```

### 5.2 Hub 主循环（伪代码）

```go
func (h *Hub) Run(ctx context.Context) {
    for {
        select {
        case c := <-h.register:
            if len(h.clients) >= 20 {
                c.Send <- mustEncode("join_rejected", "doc_full")
                close(c.Send); continue
            }
            h.clients[c.ID] = c

        case c := <-h.unregister:
            if _, ok := h.clients[c.ID]; ok {
                delete(h.clients, c.ID)
                close(c.Send)
            }
            if len(h.clients) == 0 {
                // 通知 Manager 延迟释放
            }

        case msg := <-h.inbound:
            // Infra 阶段：仅 log，不处理
            // Collab Epic：分发到 onUpdate/onAwareness
            _ = msg

        case <-h.shutdown:
            return
        case <-ctx.Done():
            return
        }
    }
}
```

**关键不变量**：
- 所有 per-hub 状态的读写**只在 Hub goroutine 内发生**（inbound/register/unregister 通过 channel 串入）
- 广播通过遍历 `clients` 往各自 `Send` channel 投递，消费者 goroutine 再写 WS
- 因此 Hub 内部**不需要锁**；D-10b 的 20 并发上限在 Hub goroutine 内判断，天然无竞态

### 5.3 握手（`internal/ws/handler.go`）

```go
func Handler(cfg Config, hm *HubManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 从 Sec-WebSocket-Protocol 取 "access.<jwt>"
        //    若无效或 node_id 缺失 → HTTP 401/400（在 Upgrade 之前）
        // 2. ResolvePermission；none → 401
        //    （Infra 阶段临时：只检查 jwt 占位；permission 留给后续）
        // 3. websocket.Accept with Subprotocols = []string{protoAfterPrefix}
        // 4. hub := hm.GetOrCreate(nodeID); go hub.Run(ctx)（如首次）
        //    client := newClient(...); hub.register <- client
        // 5. 启两个 goroutine：readPump / writePump
    }
}
```

`nhooyr.io/websocket` 示例细节放 Collab Epic，Infra 阶段 readPump 收到的消息直接回 `{type:"not_implemented"}`。

---

## 6. 前端工程骨架

### 6.1 package.json 关键依赖

```json
{
  "dependencies": {
    "react": "^19.2.0",
    "react-dom": "^19.2.0",
    "react-router-dom": "^6.30.0"
  },
  "devDependencies": {
    "typescript": "^5.9.0",
    "vite": "^7.3.0",
    "@vitejs/plugin-react": "^4.3.0",
    "@types/react": "^19.2.0",
    "@types/react-dom": "^19.2.0"
  }
}
```

协同、编辑器、Y.js 等依赖留到 Editor/Collab Epic 再加，避免骨架期间锁定过多版本。

### 6.2 Vite 配置要点

```ts
// vite.config.ts
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api':    { target: 'http://localhost:8080', changeOrigin: true },
      '/api/ws': { target: 'ws://localhost:8080',   ws: true, changeOrigin: true },
    },
  },
});
```

### 6.3 API 客户端约定

`src/api/client.ts`：

```ts
type Envelope<T> =
  | { success: true;  data: T;    error: null }
  | { success: false; data: null; error: { code: string; message: string } };

async function request<T>(method, path, body?): Promise<T> {
  // 1. 从 AuthContext 读 access；注入 Authorization
  // 2. 若 401 且原请求非 /auth/refresh：
  //      串行触发 refresh；成功则重试一次；失败则 AuthContext.logout()
  // 3. 解包 Envelope；success=false → throw ApiError(error.code, message)
}
```

### 6.4 路由骨架

```tsx
<Routes>
  <Route path="/login"    element={<LoginPage/>}/>
  <Route path="/register" element={<RegisterPage/>}/>
  <Route element={<RequireAuth><MainShell/></RequireAuth>}>
    <Route path="/"             element={<Home/>}/>
    <Route path="/doc/:id"      element={<DocPage/>}/>
  </Route>
  <Route element={<RequireRole role="manager"><AdminShell/></RequireRole>}>
    <Route path="/admin/users"  element={<AdminUsersPage/>}/>
    <Route path="/admin/groups" element={<AdminGroupsPage/>}/>
  </Route>
</Routes>
```

**所有页面 Infra 阶段仅渲染"Hello / 路由可达"占位**，无后端联调。

---

## 7. docker-compose.yml（定稿）

```yaml
services:
  postgres:
    build: ./docker/postgres   # 基于 postgres:16 + zhparser
    environment:
      POSTGRES_DB: fpgwiki
      POSTGRES_USER: fpgwiki
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-fpgwiki_dev}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports: ["5432:5432"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U fpgwiki -d fpgwiki"]
      interval: 5s
      retries: 20

  backend:
    build: ./backend
    environment:
      POSTGRES_DSN: postgres://fpgwiki:${POSTGRES_PASSWORD:-fpgwiki_dev}@postgres:5432/fpgwiki?sslmode=disable
      JWT_SECRET: ${JWT_SECRET:?JWT_SECRET required}
      UPLOAD_DIR: /data/uploads
      APP_ENV: dev
    volumes:
      - uploads:/data/uploads
    ports: ["8080:8080"]
    depends_on:
      postgres:
        condition: service_healthy

  frontend:
    build: ./frontend
    environment:
      VITE_API_BASE: http://localhost:8080
    ports: ["5173:5173"]
    depends_on: [backend]

volumes:
  postgres_data:
  uploads:
```

> 生产 compose 独立文件 `docker-compose.prod.yml`（nginx 前置 + 静态前端 + 非 dev server），不在 Infra Epic 范围。

---

## 8. Makefile（常用命令）

```makefile
.PHONY: up down logs migrate-up migrate-down migrate-new backend-test frontend-dev

up:             ; docker compose up --build
down:           ; docker compose down
logs:           ; docker compose logs -f
migrate-up:     ; cd backend && go run ./cmd/migrate up
migrate-down:   ; cd backend && go run ./cmd/migrate down 1
migrate-new:    ; cd backend && migrate create -ext sql -dir migrations -seq $(name)
backend-test:   ; cd backend && go test ./...
frontend-dev:   ; cd frontend && pnpm dev
```

---

## 9. 任务分解（给 Codex 的执行清单）

每个任务 ≤ 1 PR；序号即执行顺序。

| # | 任务 | 输出 | Acceptance Criteria |
|---|---|---|---|
| T-01 | 初始化 Go module + 目录骨架 | `backend/go.mod`, `cmd/server/main.go`, `internal/{config,db,logger,httpserver/{router,middleware,handlers},ws,models,repository,service}` 空包 | `go build ./...` 成功 |
| T-02 | config + logger + `/ping` | `config/config.go`, `logger/logger.go`, `handlers/health.go` | 设置 env 后 `go run` 启动；`curl /ping` 返回 `{"success":true,"data":{"status":"ok","db":"ok"}}`；缺 `JWT_SECRET` 时启动失败 |
| T-03 | golang-migrate 接入 + 迁移 0001 | `cmd/migrate/main.go`, `migrations/0001_*` | `make migrate-up` 后 `\dx` 能看到 4 个扩展 + `chinese` 文本搜索配置 |
| T-04 | 迁移 0002 users/groups/refresh_tokens | `migrations/0002_*` | up/down 对称；新建后能手动 `INSERT` users 一行并 query |
| T-05 | 迁移 0003 nodes + node_permissions + tsv trigger | `migrations/0003_*` | 插入 `nodes` 后 `SELECT tsv` 非空；中文标题能被 `to_tsquery('chinese','xxx')` 命中 |
| T-06 | 迁移 0004 doc_states/updates/snapshots | `migrations/0004_*` | up/down 成功；`ydoc_state` 支持写入 ≥ 1MB bytea |
| T-07 | 迁移 0005 uploads | `migrations/0005_*` | sha256 唯一约束生效 |
| T-08 | 迁移 0006 subscriptions/notifications | `migrations/0006_*` | `notifications` 部分索引 `WHERE read_at IS NULL` 存在 |
| T-09 | Gin 路由装配 + 中间件骨架（auth 占位） | `router.go` + `middleware/*` | `/ping` 经中间件仍工作；日志含 request_id |
| T-10 | WS 握手骨架 + Hub/HubManager 骨架 | `ws/*.go` | 浏览器用 `new WebSocket("ws://localhost:8080/api/ws?doc=<uuid>","access.dummy")` 能握手成功；发任何消息回 `{"type":"not_implemented"}`；关掉 60s 后 hub 被释放（log 可见） |
| T-11 | 前端 Vite + React + 路由骨架 | `frontend/*` | `pnpm dev` 启动；能访问 `/login`、`/` 渲染占位文字；`/admin/users` 无角色时跳转 `/login` 占位 |
| T-12 | API client + AuthContext（占位） | `src/api/client.ts`, `src/auth/AuthContext.tsx` | 单测：401 触发 refresh 分支（用 mock）正确执行重试 |
| T-13 | docker-compose 三服务联调 | `docker-compose.yml` + `docker/postgres/Dockerfile` + `backend/Dockerfile` + `frontend/Dockerfile` | `docker compose up` 后：前端 5173 可访问占位页；后端 8080 `/ping` 通；postgres `\dx` 看得到 zhparser |
| T-14 | Makefile + README 初版 | `Makefile`, `README.md` | `make up`, `make migrate-up`, `make backend-test`, `make frontend-dev` 全部可用 |

---

## 10. Definition of Done（本 Epic）

本 Epic 完成 = **全部满足**：

1. T-01 ~ T-14 全部通过各自 Acceptance Criteria
2. `docker compose up` 从空状态到三服务健康就绪 ≤ 90 秒
3. 所有迁移可 `up → down → up` 往返，数据无残留
4. `go vet ./...` 与 `golangci-lint run` 无告警（默认配置）
5. 前端 `tsc --noEmit` 无错误；`pnpm build` 产物 < 300KB gzipped（骨架阶段）
6. 至少一个 e2e 冒烟脚本（可以是 `scripts/smoke.sh`）：启动 compose → 轮询 `/ping` → 建 WS → 收到 `not_implemented` → 关闭 → compose down
7. `docs/requirements.md` §5 的 F-47 状态更新为"✅ Infra 完成"
8. 本文件末尾 Changelog 追加实际完成日期

---

## 11. 风险与回避

| 风险 | 影响 | 规避 |
|---|---|---|
| zhparser 镜像构建失败（Alpine 不兼容） | 阻断 postgres 启动 | `docker/postgres/Dockerfile` 基于 `postgres:16`（Debian 系）而非 alpine |
| golang-migrate 在首次启动 + 并行的 backend pod 场景下竞态 | MVP 单实例不受影响 | MVP 只跑单 backend；未来多实例时用 `migrate` 的 advisory lock |
| WS 反向代理路径在 compose 中走 Vite dev server 会断连 | 本地 dev 无法开发 | 前端的 Vite dev proxy 里 `ws: true` 已显式配置；生产部署由 nginx 统一转发 |
| `pgx` bytea 大于 PG 默认 `work_mem` 触发磁盘溢写 | 协同 compact 时慢 | 在 postgres 容器设置 `work_mem=64MB`（compose env 里 `command` 覆盖） |

---

## 12. Changelog

| 版本 | 日期 | 变更 | 维护者 |
|---|---|---|---|
| v1.0 | 2026-04-16 | 初稿：目录 / Schema / Hub 骨架 / compose / T-01 ~ T-14 任务清单 | CC |
| v1.1 | 2026-04-16 | §2 Go 版本下限由 1.22 上调到 1.25（Dockerfile 基镜像同步改为 `golang:1.25-alpine`）。触发：T-02 引入 gin v1.12.0 / pgx v5.9.1，两者均声明 `go 1.25.0`，`go mod tidy` 会把 `backend/go.mod` 回弹至 1.25；requirements.md §1.1 允许 "Go 1.22+"，不存在宪法冲突 | CC |
| v1.2 | 2026-04-21 | §1 前端目录结构：`src/routes/` 拆分为 `src/pages/`（页面组件）+ `src/layouts/`（布局外壳）+ `src/auth/`（守卫组件）；新增 `RequireAuth.tsx`、`RequireRole.tsx` 到目录树；职责更清晰，T-11 实现采用此结构 | CC |
