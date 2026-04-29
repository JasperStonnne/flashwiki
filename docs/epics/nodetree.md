# Epic: NodeTree（目录与权限）

> **状态**: 设计冻结，可下发 Codex · 2026-04-22
> **上游**: [`docs/requirements.md`](../requirements.md) v1.0 §2.2 / §3 / §4.2 / §4.5 / §6.4~§6.5 / §7.8 / §8.1
> **视觉**: [`docs/ui/design-system.md`](../ui/design-system.md) v1.0
> **范围**: 节点 CRUD（文件夹/文档）、目录树按权限过滤、权限解析算法、节点权限设置、权限组 CRUD、组长管理成员、Manager 改角色、前端侧栏目录树、管理后台（用户/组/节点权限）
> **不包含**: TipTap 编辑器（M5）、Y.js 协同（M5）、doc_states/doc_updates 读写（M5）、搜索（M9）、通知（M8）、图床（M7）

本 Epic 产出的是"能建文件夹和文档、能按权限看到目录树、能管理权限组和节点权限"的完整目录与权限闭环：
- 创建文件夹/文档 → 侧栏目录树可见
- 重命名/移动/软删除 → 目录树实时更新
- 权限解析算法 → Manager 全局 manage，个人权限覆盖组权限，继承链向上查找
- 节点权限设置 → 替换式写入 `node_permissions`
- 权限组 CRUD → Manager 创建/重命名/删除组，指定组长
- 组长管理成员 → 加入/移除本组成员
- Manager 改角色 → 册封/撤销 Manager
- 前端侧栏 + 管理后台 → 对齐 design-system.md 方向 B

---

## 1. 已有基础设施（M1~M3 产出）

### 1.1 数据库已就绪

| 表 | 迁移 | 状态 |
|---|---|---|
| `nodes` | `0003_nodes_permissions.up.sql` | ✅ 含 parent_id / kind / title / owner_id / deleted_at / tsv + trigger |
| `node_permissions` | `0003_nodes_permissions.up.sql` | ✅ 含 UNIQUE(node_id, subject_type, subject_id) |
| `groups` | `0002_users_groups.up.sql` | ✅ 含 leader_id |
| `group_members` | `0002_users_groups.up.sql` | ✅ PK(group_id, user_id) |
| `users` | `0002_users_groups.up.sql` | ✅ 含 role 字段 |

### 1.2 后端已就绪

| 模块 | 文件 | 状态 |
|---|---|---|
| Router | `internal/httpserver/router.go` | ✅ `/api` group，admin group 已有 RequireRole("manager") |
| Auth 中间件 | `middleware/auth.go` | ✅ RequireAuth / RequireRole 完成 |
| Response | `handlers/response.go` | ✅ Envelope 封装 |
| UserRepo | `repository/user_repo.go` | ✅ CRUD 完成 |

### 1.3 前端已就绪

| 模块 | 文件 | 状态 |
|---|---|---|
| 路由 | `App.tsx` | ✅ `/admin/users` + `/admin/groups` 路由已有 |
| Shell | `layouts/MainShell.tsx` / `AdminShell.tsx` | 🟡 占位 |
| Pages | `AdminUsersPage.tsx` / `AdminGroupsPage.tsx` | 🟡 占位 |
| API Client | `api/client.ts` | ✅ request() + Envelope 完成 |

---

## 2. API 端点设计

严格对齐 requirements.md §8.1：

### 2.1 节点 CRUD

#### `POST /api/nodes` — F-08 创建文件夹/文档

**Request**:
```json
{
  "parent_id": "<uuid|null>",
  "kind": "folder|doc",
  "title": "新文档"
}
```

**校验**:
- `kind`: 必填，`folder` 或 `doc`
- `title`: 必填，1~200 字符
- `parent_id`: 可选，为 null 时创建根节点；非 null 时父节点必须存在且 kind=folder 且未删除

**鉴权**: RequireAuth + 父节点 ≥ `edit`（根节点创建需要 Manager 或特殊处理，见 §2.8）

**逻辑**:
1. 若 `parent_id != null`：查 `nodes WHERE id = parent_id AND deleted_at IS NULL`，不存在 → `404 parent_not_found`
2. 若 `parent_id != null`：`ResolvePermission(user, parent)` ≥ `edit`，否则 → `403 forbidden`
3. 插入 `nodes{id, parent_id, kind, title, owner_id=user.id}`
4. 若 `kind = doc`：插入 `doc_states{node_id, ydoc_state=空bytes, version=0}`（为 M5 预留，本 Epic 可选）

**Response** `201`:
```json
{
  "success": true,
  "data": {
    "id": "<uuid>",
    "parent_id": "<uuid|null>",
    "kind": "doc",
    "title": "新文档",
    "owner_id": "<uuid>",
    "created_at": "..."
  }
}
```

#### `GET /api/nodes?parent=<id|null>` — F-12 目录树

**鉴权**: RequireAuth

**逻辑**:
1. `parent=null` → 查 `nodes WHERE parent_id IS NULL AND deleted_at IS NULL`
2. `parent=<uuid>` → 查 `nodes WHERE parent_id = <uuid> AND deleted_at IS NULL`
3. 对每个结果节点执行 `ResolvePermission(user, node)`，过滤 `none`
4. 返回过滤后的节点列表，附带 `permission` 字段和 `has_children` 字段

**Response** `200`:
```json
{
  "success": true,
  "data": [
    {
      "id": "<uuid>",
      "parent_id": null,
      "kind": "folder",
      "title": "产品文档",
      "owner_id": "<uuid>",
      "permission": "edit",
      "has_children": true,
      "created_at": "...",
      "updated_at": "..."
    }
  ]
}
```

**性能考量**: 每次只查一级子节点（按需展开）。权限解析走 `WITH RECURSIVE` 批量查祖先链，在 Go 侧计算，不逐个查。

#### `GET /api/nodes/:id` — F-12 节点详情

**鉴权**: RequireAuth + ≥ `readable`

**逻辑**:
1. 查 `nodes WHERE id = :id AND deleted_at IS NULL`，不存在 → `404 not_found`
2. `ResolvePermission(user, node)`，`none` → `404 not_found`（不暴露存在性）
3. 返回节点详情 + 权限级别

#### `PATCH /api/nodes/:id` — F-09 重命名 / F-11 移动

**Request**（部分更新）:
```json
{
  "title": "新名字",
  "parent_id": "<uuid|null>"
}
```

**鉴权**:
- 仅改 `title` → ≥ `edit`
- 改 `parent_id`（移动）→ ≥ `manage` 对当前节点 **且** ≥ `edit` 对新父节点

**校验**:
- `title`: 1~200 字符（如提供）
- `parent_id`: 新父节点必须存在、kind=folder、未删除
- 不可移动到自身或自身子孙下（循环检测）

**逻辑**:
1. `SELECT id FROM nodes WHERE id = :id FOR UPDATE`（行锁）
2. 校验权限
3. 若移动：`WITH RECURSIVE` 查子孙确保无循环
4. 更新字段

#### `DELETE /api/nodes/:id` — F-10 软删除

**鉴权**: RequireAuth + ≥ `manage`

**逻辑**:
1. `ResolvePermission(user, node)` ≥ `manage`
2. `UPDATE nodes SET deleted_at = now() WHERE id = :id AND deleted_at IS NULL`
3. **级联**：递归标记所有子孙为 deleted（`WITH RECURSIVE` 更新子树）
4. 不返回数据

**Response** `200`: `{ "success": true, "data": null }`

#### `POST /api/nodes/:id/restore` — F-13 恢复删除（P1，本 Epic 实现）

**鉴权**: RequireAuth + ≥ `manage`

**逻辑**:
1. 查 `nodes WHERE id = :id AND deleted_at IS NOT NULL`，不存在 → `404`
2. 检查 `deleted_at` 距今 ≤ 30 天
3. 恢复该节点及其子孙的 `deleted_at = NULL`
4. 若父节点已删除 → `400 parent_deleted`

---

### 2.2 权限解析算法

严格对齐 requirements.md §3.2：

```go
func (s *PermissionService) ResolvePermission(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID) (string, error) {
    // 1. Manager 短路
    user, err := s.userRepo.FindByID(ctx, userID)
    if user.Role == "manager" {
        return "manage", nil
    }

    // 2. WITH RECURSIVE 查祖先链 + 每级权限行
    ancestors, err := s.nodeRepo.GetAncestorsWithPermissions(ctx, nodeID)

    // 3. 找生效节点：最近有权限行的节点
    effectiveNode := findEffectiveNode(ancestors)
    if effectiveNode == nil {
        return "none", nil
    }

    // 4. 个人权限优先
    indRows := filterBySubject(effectiveNode.permissions, "user", userID)
    if len(indRows) > 0 {
        if hasNone(indRows) { return "none", nil }
        return maxLevel(indRows), nil
    }

    // 5. 组权限
    groupIDs, _ := s.groupMemberRepo.GetGroupIDsByUser(ctx, userID)
    grpRows := filterBySubjectIDs(effectiveNode.permissions, "group", groupIDs)
    if len(grpRows) > 0 {
        if hasNone(grpRows) { return "none", nil }
        return maxLevel(grpRows), nil
    }

    return "none", nil
}
```

**SQL 核心**（`GetAncestorsWithPermissions`）:
```sql
WITH RECURSIVE ancestors AS (
  SELECT id, parent_id, 0 AS depth FROM nodes WHERE id = $1 AND deleted_at IS NULL
  UNION ALL
  SELECT n.id, n.parent_id, a.depth + 1
  FROM nodes n
  JOIN ancestors a ON n.id = a.parent_id
  WHERE n.deleted_at IS NULL
)
SELECT a.id AS node_id, a.depth,
       np.subject_type, np.subject_id, np.level
FROM ancestors a
LEFT JOIN node_permissions np ON np.node_id = a.id
ORDER BY a.depth ASC
```

一次查询拿到整条祖先链及每级的权限行，Go 侧按深度排序后执行 §3.2 算法。

---

### 2.3 节点权限管理

#### `GET /api/nodes/:id/permissions` — F-21

**鉴权**: RequireAuth + ≥ `manage`

**Response** `200`:
```json
{
  "success": true,
  "data": {
    "node_id": "<uuid>",
    "permissions": [
      {
        "id": "<uuid>",
        "subject_type": "user",
        "subject_id": "<uuid>",
        "subject_name": "张伟",
        "level": "edit"
      },
      {
        "subject_type": "group",
        "subject_id": "<uuid>",
        "subject_name": "设计组",
        "level": "readable"
      }
    ],
    "inherited_from": "<uuid|null>"
  }
}
```

`inherited_from`：当前节点无自有权限行时，标注生效节点 ID；有自有行时为 null。

#### `PUT /api/nodes/:id/permissions` — F-21 替换式写入

**Request**:
```json
{
  "permissions": [
    { "subject_type": "user", "subject_id": "<uuid>", "level": "edit" },
    { "subject_type": "group", "subject_id": "<uuid>", "level": "readable" }
  ]
}
```

**鉴权**: RequireAuth + ≥ `manage`

**逻辑**（对齐 §7.8）:
```
BEGIN
  SELECT id FROM nodes WHERE id = :id FOR UPDATE
  DELETE FROM node_permissions WHERE node_id = :id
  INSERT new rows (批量)
COMMIT
```

**校验**:
- `subject_type` ∈ `{user, group}`
- `level` ∈ `{manage, edit, readable, none}`
- `subject_id` 必须存在（user 或 group）
- 同一 `(subject_type, subject_id)` 不重复

---

### 2.4 权限组管理

#### `GET /api/admin/groups` — F-22

**鉴权**: RequireAuth + RequireRole("manager")

返回所有组，含组长信息和成员数。

**Response** `200`:
```json
{
  "success": true,
  "data": [
    {
      "id": "<uuid>",
      "name": "设计组",
      "leader": { "id": "<uuid>", "display_name": "张伟", "email": "..." },
      "member_count": 5,
      "created_at": "..."
    }
  ]
}
```

#### `POST /api/admin/groups` — F-22 创建组

**Request**:
```json
{
  "name": "设计组",
  "leader_id": "<uuid>"
}
```

**校验**:
- `name`: 必填，1~50 字符，唯一
- `leader_id`: 必填，用户必须存在

**逻辑**:
1. 插入 `groups`
2. 自动将 leader 加入 `group_members`

#### `PATCH /api/admin/groups/:id` — F-22 重命名/换组长

**Request**（部分更新）:
```json
{
  "name": "新名字",
  "leader_id": "<uuid>"
}
```

#### `DELETE /api/admin/groups/:id` — F-22 删除组

**逻辑**:
1. `DELETE FROM groups WHERE id = :id`
2. 级联删除 `group_members`（数据库 ON DELETE CASCADE）
3. 级联删除 `node_permissions WHERE subject_type='group' AND subject_id=:id`（需要应用层处理，因为 FK 未直接关联）

---

### 2.5 组成员管理

#### `POST /api/admin/groups/:id/members` — F-24 添加成员

**Request**:
```json
{
  "user_id": "<uuid>"
}
```

**鉴权**: RequireAuth + (Manager 或该组 Leader)

**逻辑**:
1. 查 group 存在
2. 鉴权：`user.role == "manager"` 或 `group.leader_id == user.id`
3. 查 user 存在
4. `INSERT INTO group_members` ON CONFLICT 忽略（幂等）

#### `DELETE /api/admin/groups/:id/members/:uid` — F-24 移除成员

**鉴权**: RequireAuth + (Manager 或该组 Leader)

**边界**: 不可移除组长自己（必须先换组长）

---

### 2.6 Manager 改角色

#### `PATCH /api/admin/users/:id/role` — F-25

**Request**:
```json
{
  "role": "manager|member"
}
```

**鉴权**: RequireAuth + RequireRole("manager")

**逻辑**:
1. `SELECT id FROM users WHERE id = :id FOR UPDATE`
2. 不可改自己 → `400 cannot_change_own_role`
3. `UPDATE users SET role = $1, updated_at = now() WHERE id = $2`
4. 若降级为 member → `token_version += 1`（使现有 token 失效，下次请求重新获取 role）

---

### 2.7 用户列表（管理后台）

#### `GET /api/admin/users` — F-45 补充

**鉴权**: RequireAuth + RequireRole("manager")

返回所有用户列表（管理后台用）。

**Response** `200`:
```json
{
  "success": true,
  "data": [
    {
      "id": "<uuid>",
      "email": "user@example.com",
      "display_name": "张伟",
      "role": "member",
      "created_at": "..."
    }
  ]
}
```

---

### 2.8 根节点创建策略

**决策**: 根节点（`parent_id IS NULL`）的创建仅限 Manager。普通用户只能在已有文件夹内创建子节点。

理由：根节点是目录树的顶层组织，应由管理者规划。

---

## 3. 后端分层结构

```
internal/
├── models/
│   └── node.go              # Node / NodePermission / NodeWithPermission struct
│   └── group.go             # Group / GroupMember struct
├── repository/
│   └── node_repo.go         # nodes 表 CRUD + WITH RECURSIVE 祖先查询
│   └── node_permission_repo.go  # node_permissions 表操作
│   └── group_repo.go        # groups + group_members 表操作
├── service/
│   └── node_service.go      # 节点 CRUD + 权限校验
│   └── permission_service.go # ResolvePermission 算法 + 权限设置
│   └── group_service.go     # 权限组 CRUD + 成员管理
├── httpserver/
│   └── handlers/
│       └── node.go          # 节点路由处理
│       └── permission.go    # 节点权限路由处理
│       └── group.go         # 权限组路由处理
│       └── admin.go         # 用户列表 + 改角色
│   └── middleware/
│       └── permission.go    # RequireNodePermission 中间件（可选，或在 handler 内校验）
│   └── router.go            # 新增路由注册
```

### 3.1 依赖注入链

```
router.go
  → handlers.NewNodeHandler(nodeService, permissionService)
  → handlers.NewPermissionHandler(permissionService)
  → handlers.NewGroupHandler(groupService)
  → handlers.NewAdminHandler(userService, authService)  // 扩展现有

nodeService
  → nodeRepo + permissionService

permissionService
  → nodeRepo + nodePermissionRepo + groupMemberRepo + userRepo

groupService
  → groupRepo + groupMemberRepo + nodePermissionRepo(删组时清权限行)
```

---

## 4. 前端实现

### 4.1 侧栏目录树

对齐 design-system.md §7.9 + 原型 HTML：

- 按需展开（lazy load）：点击文件夹 → `GET /api/nodes?parent=<id>`
- 根级别首屏加载 → `GET /api/nodes?parent=null`
- 节点图标：📁 文件夹 / 📄 文档
- Active 状态：当前打开的文档高亮
- 右键菜单：新建文档/文件夹、重命名、移动、删除
- 展开/折叠状态本地持久化（localStorage）

### 4.2 管理后台 — 用户管理页

- 用户列表表格：邮箱、昵称、角色、创建时间
- Manager 可点击角色标签切换角色（册封/撤销 Manager）
- 强制下线按钮（已有 API，M3 实现）

### 4.3 管理后台 — 权限组管理页

- 组列表：组名、组长、成员数
- 创建组弹窗：输入组名 + 选择组长
- 编辑组：改名 + 换组长
- 删除组确认弹窗
- 成员管理：展开组 → 成员列表 → 添加/移除

### 4.4 权限设置弹窗

- 在目录树或节点页触发（对有 manage 权限的节点）
- 展示当前权限列表（个人/组 + 级别）
- 添加/移除/修改权限条目
- 如果是继承的，显示"继承自 <节点名>"提示
- 提交 → `PUT /api/nodes/:id/permissions`

---

## 5. 安全清单

- [ ] 所有写操作使用 `SELECT ... FOR UPDATE` 行锁（D-01）
- [ ] 权限解析算法中 `none` 为最终判定，不可被 max 覆盖
- [ ] Manager 短路在算法入口执行，不查 `node_permissions`
- [ ] 软删除节点对 `none` 权限用户返回 404（不暴露存在性）
- [ ] 移动节点时循环检测（不可移到自身子孙下）
- [ ] 删除组时清理 `node_permissions` 中该组的所有行
- [ ] 改角色时 `token_version += 1` 使旧 token 失效

---

## 6. 任务清单

> Codex 按顺序实施。每个任务附 Acceptance Criteria。

### Phase 1: 后端 Models + Repository

| ID | 任务 | 产出文件 | Acceptance Criteria |
|---|---|---|---|
| T-25 | Node / NodePermission / Group models | `models/node.go`, `models/group.go` | 编译通过；结构体字段对齐 DDL |
| T-26 | NodeRepo（CRUD + 祖先链查询） | `repository/node_repo.go` | 编译通过；单测覆盖 Create / FindByID / ListByParent / SoftDelete / Restore / GetAncestorsWithPermissions |
| T-27 | NodePermissionRepo | `repository/node_permission_repo.go` | 编译通过；单测覆盖 ReplacePermissions / ListByNode |
| T-28 | GroupRepo（CRUD + 成员管理） | `repository/group_repo.go` | 编译通过；单测覆盖 CreateGroup / UpdateGroup / DeleteGroup / AddMember / RemoveMember / GetGroupIDsByUser / ListGroups |

### Phase 2: 后端 Service

| ID | 任务 | 产出文件 | Acceptance Criteria |
|---|---|---|---|
| T-29 | PermissionService（ResolvePermission 算法） | `service/permission_service.go` | 单测（mock repo）覆盖：Manager 短路、个人权限覆盖组、none 最终、继承链向上、无权限行返回 none |
| T-30 | NodeService（节点 CRUD + 权限校验） | `service/node_service.go` | 单测覆盖：创建（根/子）、重命名、移动（含循环检测失败）、软删除（含子树）、恢复 |
| T-31 | GroupService（权限组 CRUD + 成员管理） | `service/group_service.go` | 单测覆盖：创建组、删除组（含清理 node_permissions）、添加/移除成员、组长鉴权 |

### Phase 3: 后端 Handlers + 路由

| ID | 任务 | 产出文件 | Acceptance Criteria |
|---|---|---|---|
| T-32 | Node Handlers + Permission Handlers | `handlers/node.go`, `handlers/permission.go` | 编译通过；go vet 通过 |
| T-33 | Group Handlers + Admin Handlers | `handlers/group.go`, `handlers/admin.go` | 编译通过；go vet 通过 |
| T-34 | Router 装配 + 集成测试 | `router.go` 修改 + `handlers/*_integration_test.go` | 集成测试覆盖：创建根文件夹→创建子文档→重命名→移动→设权限→列目录树（过滤 none）→删除→恢复 |

### Phase 4: 前端

| ID | 任务 | 产出文件 | Acceptance Criteria |
|---|---|---|---|
| T-35 | API endpoints + types 补全 | `api/endpoints.ts`, `api/types.ts`（新建） | tsc 通过；包含所有 M4 API 路径 + 请求/响应类型 |
| T-36 | MainShell + 侧栏目录树 | `layouts/MainShell.tsx`, `components/Sidebar.tsx`, `components/NodeTree.tsx` | 浏览器可见侧栏；按需展开文件夹；节点按权限过滤 |
| T-37 | 右键菜单 + 节点 CRUD 操作 | `components/NodeContextMenu.tsx` | 右键新建/重命名/删除可用；操作调后端 API |
| T-38 | 权限设置弹窗 | `components/PermissionDialog.tsx` | 弹窗展示当前权限列表；可添加/移除/修改；提交成功 |
| T-39 | AdminUsersPage 实现 | `pages/AdminUsersPage.tsx` | 用户列表可见；改角色 + 强制下线可用 |
| T-40 | AdminGroupsPage 实现 | `pages/AdminGroupsPage.tsx` | 组列表 + CRUD + 成员管理全流程可用 |
| T-41 | 端到端冒烟验证 | `scripts/smoke_nodetree.sh` | curl 脚本覆盖：创建根文件夹→子文档→设权限→非授权用户 403/404→移动→删除→恢复→组 CRUD→改角色 |

---

## 7. Definition of Done（本 Epic）

本 Epic 完成 = **全部满足**：

1. T-25 ~ T-41 全部通过各自 Acceptance Criteria
2. `go test -race -cover ./...` 通过，node/permission/group 相关包覆盖率 ≥ 80%
3. `golangci-lint run` 零告警
4. 前端 `tsc --noEmit` 无错误
5. 浏览器 Manager 登录后：侧栏看到目录树，能创建文件夹/文档，能设置权限
6. 普通用户看不到 `none` 权限的节点；`readable` 用户看得到但不能编辑
7. 管理后台用户页：能册封/撤销 Manager，能强制下线
8. 管理后台权限组页：组 CRUD + 成员管理全流程可用

---

## 8. 风险与依赖

| 风险 | 缓解 |
|---|---|
| `WITH RECURSIVE` 在深层目录树上可能慢 | M1 阶段目录树不深（< 10 层）；若未来需要优化可引入 ltree（D-05 已预留） |
| 权限解析每次查 DB | 用户量小可接受；M12 加固阶段可加短 TTL 缓存 |
| 替换式权限写入在并发下可能丢失 | `SELECT ... FOR UPDATE` 行锁互斥（D-01） |
| 删除组后 `node_permissions` 残留 | 应用层显式清理，不依赖 FK 级联 |
| 前端目录树渲染大量节点 | 按需展开 + 虚拟滚动（若超 200 节点再优化） |

---

## 9. Changelog

| 版本 | 日期 | 变更 | 维护者 |
|---|---|---|---|
| v1.0 | 2026-04-22 | 初稿：API 设计 / 权限算法 / 分层结构 / 安全清单 / T-25~T-41 任务清单 | CC |
