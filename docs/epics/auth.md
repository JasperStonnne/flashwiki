# Epic: Auth（鉴权）

> **状态**: 设计冻结，可下发 Codex · 2026-04-21
> **上游**: [`docs/requirements.md`](../requirements.md) v1.0 §4.1 / §7.1 ~ §7.3 / §8.1
> **视觉**: [`docs/ui/design-system.md`](../ui/design-system.md) v1.0
> **范围**: 注册、登录、双 token 签发与刷新、登出、修改密码、Manager 强制下线、用户资料 CRUD、前端登录/注册页
> **不包含**: 权限组 CRUD（M4）、节点权限解析（M4）、角色修改 API（M4，F-25 虽有 Manager 鉴权但归属 NodeTree Epic）

本 Epic 产出的是"能登录、能保持会话、能被踢下线"的完整鉴权闭环：
- 注册 → 登录 → access token 签发 → 受保护 API 可调用
- access 过期 → refresh 自动续签 → 无缝体验
- refresh 泄漏 → 复用检测 → 全量吊销
- Manager 强制下线 → token_version 翻转 → WS + API 全部踢出
- 前端登录/注册页按 design-system.md 方向 B 实现

---

## 1. 已有基础设施（M1 产出）

### 1.1 后端已就绪

| 模块 | 文件 | 状态 |
|---|---|---|
| 数据库表 | `migrations/0002_users_groups.up.sql` | ✅ `users` + `refresh_tokens` 表已建 |
| Config | `internal/config/config.go` | ✅ `JWTSecret` / `JWTAccessTTL` / `JWTRefreshTTL` 已声明 |
| 路由 | `internal/httpserver/router.go` | ✅ 骨架已有 `/api` group，待填充 auth 路由 |
| Auth 中间件 | `internal/httpserver/middleware/auth.go` | 🟡 占位，`RequireAuth` / `OptionalAuth` 空壳 |
| Response | `internal/httpserver/handlers/response.go` | ✅ 通用 Envelope 封装已有 |

### 1.2 前端已就绪

| 模块 | 文件 | 状态 |
|---|---|---|
| API Client | `src/api/client.ts` | ✅ Envelope 类型 + 401 自动刷新逻辑已有 |
| Endpoints | `src/api/endpoints.ts` | ✅ auth 路由常量已声明 |
| AuthContext | `src/auth/AuthContext.tsx` | 🟡 占位：`login()` 存 localStorage，`refresh()` 直接 logout |
| Route Guards | `src/auth/RequireAuth.tsx` / `RequireRole.tsx` | 🟡 占位逻辑 |
| Pages | `src/pages/LoginPage.tsx` / `RegisterPage.tsx` | 🟡 占位文字 |

---

## 2. API 端点设计

严格对齐 requirements.md §8.1：

### 2.1 `POST /api/auth/register` — F-01

**Request**:
```json
{
  "email": "user@example.com",
  "password": "Str0ngP@ss",
  "display_name": "张伟"
}
```

**校验**:
- `email`: 必填，合法邮箱格式，转小写（citext 已处理，但应用层也校验）
- `password`: 必填，≥ 8 字符
- `display_name`: 必填，1~50 字符

**逻辑**:
1. 查重 `users.email`；已存在 → `409 email_taken`
2. `bcrypt.Generate(password, cost=12)` → `password_hash`
3. `role = 'member'`（首位 Manager 通过 seed 脚本创建）
4. 插入 `users`
5. 自动登录：签发 access + refresh（复用 §2.2 逻辑）

**Response** `201`:
```json
{
  "success": true,
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "<raw>",
    "user": {
      "id": "<uuid>",
      "email": "user@example.com",
      "display_name": "张伟",
      "role": "member",
      "locale": "zh"
    }
  }
}
```

### 2.2 `POST /api/auth/login` — F-02

**Request**:
```json
{
  "email": "user@example.com",
  "password": "Str0ngP@ss"
}
```

**逻辑**: 严格按 requirements.md §7.1：
1. `email` 转小写查 `users`；不存在 → `401 invalid_credentials`
2. `bcrypt.Compare(password, password_hash)`；失败 → `401 invalid_credentials`（统一错误，不泄漏是邮箱还是密码错）
3. 签发 `access = JWT{sub=user.id, role=user.role, tv=user.token_version, exp=now()+15m}`
4. `refresh_raw = crypto.Rand(32)`；`INSERT refresh_tokens{token_hash=SHA256(refresh_raw), expires_at=now()+7d, user_id}`
5. 返回 `{ access_token, refresh_token: hex(refresh_raw), user }`

**Response** `200`: 同 §2.1 格式。

### 2.3 `POST /api/auth/refresh` — F-03

**Request**:
```json
{
  "refresh_token": "<hex>"
}
```

**逻辑**: 严格按 requirements.md §7.2：
1. `hash = SHA256(hex_decode(refresh_token))`
2. 查 `refresh_tokens WHERE token_hash = hash`
3. 不存在 → `401 invalid_token`
4. 已过期（`expires_at < now()`）→ `401 token_expired`
5. `revoked_at IS NOT NULL` 或 `replaced_by IS NOT NULL`（泄漏检测）：
   - `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = row.user_id AND revoked_at IS NULL`（级联吊销）
   - `UPDATE users SET token_version = token_version + 1 WHERE id = row.user_id`
   - → `401 token_reused`
6. 正常路径：
   - `new_raw = crypto.Rand(32)`
   - `INSERT new_row{token_hash=SHA256(new_raw), ...}`
   - `UPDATE old_row SET replaced_by = new_row.id`
   - `new_access = JWT{sub, role, tv=user.token_version, exp=+15m}`
   - 返回 `{ access_token: new_access, refresh_token: hex(new_raw) }`

**Response** `200`: 同 §2.1 格式（但无 `user` 字段，仅 `access_token` + `refresh_token`）。

### 2.4 `POST /api/auth/logout` — F-04

**Request**: 空（或 `{ "refresh_token": "<hex>" }`，如果前端持有）

**鉴权**: `RequireAuth`

**逻辑**:
1. 从请求体取 `refresh_token`；如果有：`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = SHA256(hex_decode(token))`
2. 如果没有 `refresh_token`（只有 access）：静默成功（access 自然过期）
3. 前端清理 localStorage

**Response** `200`: `{ "success": true, "data": null }`

### 2.5 `POST /api/me/password` — F-05

**Request**:
```json
{
  "old_password": "OldP@ss",
  "new_password": "NewStr0ng"
}
```

**鉴权**: `RequireAuth`

**校验**: `new_password` ≥ 8 字符

**逻辑**:
1. `bcrypt.Compare(old_password, user.password_hash)`；失败 → `401 invalid_password`
2. `bcrypt.Generate(new_password, cost=12)` → 新 `password_hash`
3. `UPDATE users SET password_hash = ?, token_version = token_version + 1, updated_at = now()`
4. `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = ? AND revoked_at IS NULL`
5. 返回新的 `access_token` + `refresh_token`（让当前会话继续）

**Response** `200`: `{ access_token, refresh_token }`

### 2.6 `POST /api/admin/users/:id/force-logout` — F-06

**鉴权**: `RequireAuth` + `RequireRole("manager")`

**逻辑**: 严格按 requirements.md §7.3：
1. `BEGIN`
2. `UPDATE users SET token_version = token_version + 1 WHERE id = :id`
3. `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = :id AND revoked_at IS NULL`
4. `COMMIT`
5. 通过 `ws.HubManager` 向该用户所有在线 WS 连接推送 `{type:"force_logout"}`
6. 前端收到后清 storage → 跳登录页

**Response** `200`: `{ "success": true, "data": null }`

**边界**:
- Manager 不可下线自己 → `400 cannot_force_logout_self`
- 目标用户不存在 → `404 user_not_found`

### 2.7 `GET /api/me` — F-07

**鉴权**: `RequireAuth`

**Response** `200`:
```json
{
  "success": true,
  "data": {
    "id": "<uuid>",
    "email": "user@example.com",
    "display_name": "张伟",
    "role": "member",
    "locale": "zh",
    "created_at": "2026-04-21T10:00:00Z"
  }
}
```

### 2.8 `PATCH /api/me` — F-07

**Request** (部分更新):
```json
{
  "display_name": "张小伟",
  "locale": "en"
}
```

**校验**:
- `display_name`: 1~50 字符（如果提供）
- `locale`: `"en"` 或 `"zh"`（如果提供）

**Response** `200`: 返回更新后的完整用户对象。

---

## 3. JWT 结构

### 3.1 Access Token Claims

```json
{
  "sub": "<user_id uuid>",
  "role": "manager|member",
  "tv": 0,
  "exp": 1719504000,
  "iat": 1719503100
}
```

- `sub`: 用户 ID
- `role`: 用户角色
- `tv`: `users.token_version`，中间件校验时比对数据库值
- 签名算法: `HS256`
- Secret: `config.JWTSecret`

### 3.2 Auth 中间件逻辑

`RequireAuth`:
1. 读 `Authorization: Bearer <token>`
2. 解析并验证 JWT 签名 + 过期时间
3. 查 `users WHERE id = claims.sub`
4. 比对 `claims.tv == user.token_version`；不一致 → `401 token_revoked`
5. 注入 `c.Set("user_id", user.ID)` + `c.Set("user_role", user.Role)` + `c.Set("token_version", user.TokenVersion)`

`RequireRole(role)`:
1. 读 `c.Get("user_role")`
2. 不匹配 → `403 forbidden`

**性能考量**: `RequireAuth` 每次请求查一次 `users` 表（只查 `id, role, token_version` 三个字段）。M1 阶段用户量极小，可接受。未来如有压力可引入短 TTL 内存缓存（≤ 30s）。

---

## 4. 后端分层结构

```
internal/
├── httpserver/
│   ├── handlers/
│   │   └── auth.go            # Register / Login / Refresh / Logout / Password
│   │   └── user.go            # GetMe / UpdateMe / ForceLogout
│   └── middleware/
│       └── auth.go            # RequireAuth / RequireRole (填充实现)
├── service/
│   └── auth_service.go        # 业务逻辑：签发 token、刷新、吊销、密码
│   └── user_service.go        # 用户资料 CRUD
├── repository/
│   └── user_repo.go           # users 表 CRUD
│   └── refresh_token_repo.go  # refresh_tokens 表 CRUD
└── models/
    └── user.go                # User / RefreshToken struct（对齐 requirements §6）
    └── auth.go                # JWT Claims、Request/Response DTO
```

### 4.1 依赖注入链

```
router.go
  → handlers.NewAuthHandler(authService)
  → handlers.NewUserHandler(userService)
  → middleware.RequireAuth(cfg, userRepo)

authService
  → userRepo + refreshTokenRepo + config(JWT settings)

userService
  → userRepo
```

---

## 5. 前端实现

### 5.1 登录 / 注册页

按 design-system.md 方向 B 实现，对齐原型 `html/fpgwiki-direction-b-complete.html` 登录页：
- 左侧品牌叙事区 + 右侧登录卡片
- 登录/注册 pill tab 切换
- 表单提交调用 `POST /api/auth/login` 或 `/register`
- 成功后 `AuthContext.login(access_token, user.role)` 并存 `refresh_token` 到 localStorage
- 跳转 `/`（主页）

### 5.2 AuthContext 填充

当前占位逻辑需要升级：

1. **login()**: 存 `access_token` + `refresh_token` + `user_role` 到 localStorage
2. **refresh()**: 调用 `POST /api/auth/refresh`，用 localStorage 里的 `refresh_token`；成功后更新 `access_token` + `refresh_token`；失败则 logout
3. **logout()**: 调用 `POST /api/auth/logout`（带 refresh_token），清 localStorage，跳 `/login`

### 5.3 API Endpoints 补全

```typescript
export const API_ENDPOINTS = {
  authLogin: '/api/auth/login',
  authRegister: '/api/auth/register',
  authRefresh: '/api/auth/refresh',
  authLogout: '/api/auth/logout',
  me: '/api/me',
  mePassword: '/api/me/password',
} as const
```

### 5.4 force_logout WS 处理

前端在 WS 连接中监听 `{type: "force_logout"}` 消息：
- 收到后立即调用 `AuthContext.logout()`
- 清 localStorage → 跳登录页
- 显示 toast："你已被管理员强制下线"

---

## 6. Seed 脚本

首位 Manager 通过 seed 创建。新增 `backend/cmd/seed/main.go`：

```
go run ./cmd/seed --email=admin@fpg.com --password=Admin123! --name=Admin
```

- 插入 `users` 表，`role = 'manager'`
- 幂等：如果 email 已存在则跳过
- 仅用于开发/部署首次初始化

---

## 7. 安全清单

- [ ] 密码存储: bcrypt cost=12，永不明文
- [ ] JWT Secret: 从环境变量读取，≥ 32 字符
- [ ] 统一错误: 登录失败不区分"邮箱不存在"和"密码错误"
- [ ] refresh token: SHA256 后存库，原文仅在响应中出现一次
- [ ] 复用检测: 旧 token 被用 → 级联吊销 + token_version 翻转
- [ ] 密码修改: 旧密码验证 + token_version 翻转 + 全量吊销 refresh
- [ ] 强制下线: token_version 翻转 + WS 推送 + 前端清理
- [ ] CORS: M1 已配置，仅允许 `localhost:5173`（dev）
- [ ] Rate Limiting: 留到 M12 部署加固，当前不做

---

## 8. 任务清单

> Codex 按顺序实施。每个任务附 Acceptance Criteria。

| ID | 任务 | 产出文件 | Acceptance Criteria |
|---|---|---|---|
| T-15 | Models + Repository 层 | `models/user.go`, `models/auth.go`, `repository/user_repo.go`, `repository/refresh_token_repo.go` | 编译通过；单测覆盖 `CreateUser`, `FindByEmail`, `CreateRefreshToken`, `FindByTokenHash`, `RevokeAllByUserID` |
| T-16 | Auth Service（注册/登录/刷新/登出/改密码） | `service/auth_service.go` | 单测覆盖 §7.1 / §7.2 / §7.3 全部分支（mock repo）；复用检测路径有测试 |
| T-17 | User Service（GetMe / UpdateMe） | `service/user_service.go` | 单测覆盖 get + update + 不存在场景 |
| T-18 | RequireAuth / RequireRole 中间件填充 | `middleware/auth.go` | 单测：无 token → 401；过期 → 401；tv 不匹配 → 401；正常 → next；RequireRole 非 manager → 403 |
| T-19 | Auth Handlers（Register / Login / Refresh / Logout / Password） | `handlers/auth.go` | 集成测试（httptest）：注册 → 登录 → 拿 access 调 /me → 刷新 → 改密码 → 旧 access 失效 |
| T-20 | User Handlers（GetMe / UpdateMe / ForceLogout） | `handlers/user.go` | 集成测试：获取资料 → 更新 display_name → ForceLogout → 被踢用户 access 失效 |
| T-21 | Seed 脚本 | `cmd/seed/main.go` | `go run ./cmd/seed --email=admin@fpg.com --password=Admin123! --name=Admin` 可执行；重复运行幂等 |
| T-22 | 前端登录/注册页 UI | `pages/LoginPage.tsx`, `pages/RegisterPage.tsx` | 对齐 design-system.md 方向 B 原型；表单验证；提交调后端 API；成功跳转 `/` |
| T-23 | AuthContext + API client 填充 | `auth/AuthContext.tsx`, `api/client.ts`, `api/endpoints.ts` | refresh 自动续签可用；401 → 刷新 → 重试逻辑端到端验证；force_logout WS 处理 |
| T-24 | 端到端冒烟验证 | 手动或脚本 | `docker compose up` → 注册 → 登录 → 访问 /api/me → refresh → 改密码 → ForceLogout → 被踢 → 重新登录全流程通过 |

---

## 9. Definition of Done（本 Epic）

本 Epic 完成 = **全部满足**：

1. T-15 ~ T-24 全部通过各自 Acceptance Criteria
2. `go test -race -cover ./...` 通过，auth 相关包覆盖率 ≥ 80%
3. `golangci-lint run` 零告警
4. 前端 `tsc --noEmit` 无错误
5. 浏览器打开 `/login`，输账号密码能登进占位首页，刷新不掉线，登出清 token
6. Manager 强制下线后，被踢用户的 API 调用返回 401，WS 收到 `force_logout`

---

## 10. 风险与依赖

| 风险 | 缓解 |
|---|---|
| bcrypt cost=12 在低配机器上可能慢（~200ms） | 可接受；注册/登录/改密码低频操作 |
| RequireAuth 每请求查 DB | 用户量小可接受；M12 加固阶段评估缓存 |
| refresh_token 存 localStorage 有 XSS 风险 | CSP 在 M12 配置；当前 dev 环境可接受 |
| WS force_logout 依赖 HubManager（M1 已有） | 已就绪，仅需添加"按 user_id 推送"方法 |

---

## 11. Changelog

| 版本 | 日期 | 变更 | 维护者 |
|---|---|---|---|
| v1.0 | 2026-04-21 | 初稿：API 设计 / JWT 结构 / 分层 / 安全清单 / T-15 ~ T-24 任务清单 | CC |
