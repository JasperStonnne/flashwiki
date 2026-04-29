# FPGWiki Roadmap

> **版本**: v0.1 · 2026-04-16
> **上游**: [`docs/requirements.md`](./requirements.md) v1.0
> **维护**: CC（首席架构师与 PM）；每完成一个里程碑更新本文件 §6 进度与 §7 Changelog

---

## 1. 本文件的作用

本文件是**时间维度的真源**（requirements.md 是功能维度的真源，Epic 文档是单 Epic 实施维度的真源）。

- 回答"现在在地图的哪里 / 下一个能看到什么 / 公网上线还要多久"
- 每个里程碑绑定可演示交付物（demo），而非内部技术指标
- **不是合同**。时间是诚实估计，踩坑会后移；优先保质，不优先保 deadline

---

## 2. 节奏原则

1. **Waterfall + docs-first**：每个 Epic 开工前先冻结设计文档（`docs/epics/<epic>.md`）
2. **每周至少一个可演示交付**：避免"两周看不到东西"的焦虑
3. **并行两条线**：Codex 推进代码线；CC 推进下一个 Epic 的设计线（不阻塞）
4. **UI spec 前置**：每个 Epic 的 UI 章节必须在开工前签字；`docs/ui/design-system.md` 在 Auth Epic 前落地

---

## 3. 里程碑一览

> 周次从 2026-04-13（本周一）起算。时间区间为**乐观 ~ 保守**，踩坑会整体后移。

| 里程碑 | Epic | 周次 | Demo（外部可见的东西） | 里程碑 DoD |
|---|---|---|---|---|
| **M1** | Infra 三服务启动 | Week 1 ~ 3 | `docker compose up` 三服务起来，`/ping` 通，前端占位页可打开 | infra.md §10 全部达标 |
| **M2** | UI 视觉定调 | Week 3 末 | `docs/ui/design-system.md` 冻结 + 3~5 张视觉参考图锁定 | 视觉方向、tokens、组件库选型签字 |
| **M3** | Auth 可登录 | Week 4 | 浏览器打开 `/login` 输账号密码能登进占位首页，刷新不掉线，登出清 token | F-01 ~ F-05 打钩 |
| **M4** | 目录与权限 | Week 5 ~ 6 | 侧栏能建目录/文档，能设组织/组/个人权限，越权访问被拒 | F-06 ~ F-15 打钩 |
| **M5** | 实时协同 MVP | Week 7 ~ 9 | 两个浏览器同时打开同一文档，能看到对方光标 + 实时同步编辑，断线重连不丢字 | F-16 ~ F-24 打钩 · **最大风险点** |
| **M6** | 历史回溯 | Week 10 | 历史面板看得到版本列表、能预览差异、能一键回滚 | F-25 ~ F-28 打钩 |
| **M7** | 图床 | Week 11 前半 | 编辑器里粘贴图片能直传，无权限用户拿不到 URL（Proxy 生效） | F-29 ~ F-32 打钩 |
| **M8** | 通知 | Week 11 后半 | 被 @、文档被改、权限变更能在铃铛看到红点，点击跳转正确 | F-33 ~ F-38 打钩 |
| **M9** | 搜索 | Week 12 | 顶部搜索框输中文能命中正文，结果高亮，空态友好 | F-39 ~ F-43 打钩 |
| **M10** | Frontend Shell 完整化 | Week 13 ~ 14 | 所有页面有正式 UI（不再是占位），Admin 页可用，主题统一 | F-44 ~ F-47 打钩 |
| **M11** | **内网 MVP 可用** | Week 14 末 | 团队内网 3~5 人试用 1 周不崩；bug 收敛到只有 P2/P3 | requirements.md §12 全部达标 |
| **M12** | 部署加固 | Week 15 | `docker-compose.prod.yml`、Nginx+TLS、Postgres 备份、密钥管理、压测 20 人同编 | 见 §4 部署加固清单 |
| **M13** | **公网可访问 MVP** | Week 16 | 域名访问、HTTPS 锁、陌生人能注册试用 | 见 §5 公网上线清单 |

---

## 4. M12 部署加固清单

上公网前必须全部打钩：

- [ ] `docker-compose.prod.yml`（Nginx 前置 + 真实 TLS 证书；不是 Vite dev server）
- [ ] Postgres WAL 归档 + 每日 `pg_dump` 异机备份脚本
- [ ] 密钥管理：`.env` 不进仓库；通过环境变量或 Vault 注入
- [ ] Nginx 配置：CSP / HSTS / X-Frame-Options / Referrer-Policy
- [ ] 后端：全局速率限制中间件（requirements.md §10 NFR）
- [ ] 结构化日志落盘 + 轮转（至少 Loki + Grafana 或同等方案）
- [ ] 压测：20 人同时打开同一文档持续 30 分钟无 panic、无内存泄漏
- [ ] 压测：压盘 `doc_updates` 高频写入 + compact 触发正确
- [ ] 错误监控：前端 Sentry 或同类；后端 panic 捕获 + 告警
- [ ] 健康检查端点 + Docker HEALTHCHECK
- [ ] Dockerfile 镜像扫描无 HIGH/CRITICAL CVE
- [ ] Backup 演练：从备份恢复 postgres 一次并验证文档可访问

---

## 5. M13 公网上线清单

- [ ] 域名购买 + DNS 解析
- [ ] ICP 备案完成（如面向中国大陆公网；通常 7~20 个工作日，**建议 M11 开始就并行申请**）
- [ ] TLS 证书（Let's Encrypt 自动续期）
- [ ] 隐私政策页 + 服务条款页（一页 Markdown 就够，但必须有）
- [ ] 运营邮箱（密码重置、通知发件）
- [ ] 首次冷启动脚本：建第一个 Manager 账号 + 默认组织
- [ ] 流量限制策略（防爬、防注册滥用）
- [ ] 上线后一周值守计划

---

## 6. 当前进度

> 每完成一个任务就更新这一节

**当前里程碑**: M5（实时协同 MVP）
**当前 Epic**: Collab（实时协同编辑）— 设计冻结 2026-04-27
**当前任务**: T-43 DocUpdateRepo

**已完成**:

**M5 Collab Phase 1 Repository 层（进行中）**
- ✅ T-42 DocState / DocUpdate models + DocStateRepo（commit `cbf85ad` · 2026-04-29，AC1~AC18 全通过）

**M4 目录与权限 ✅ 已关单（2026-04-27）**
- ✅ NodeTree Epic 设计文档冻结（nodetree.md v1.0 · 2026-04-22）
- ✅ T-25 Node / NodePermission / Group models（commit `9b075b6` · 2026-04-22）
- ✅ T-26 NodeRepo CRUD + 祖先链查询（commit `b768d26` · 2026-04-22）
- ✅ T-27 NodePermissionRepo 替换式写入（commit `9ef4a90` · 2026-04-22）
- ✅ T-28 GroupRepo CRUD + 成员管理（commit `d4ed2b5` · 2026-04-22）
- ✅ T-29 PermissionService ResolvePermission 算法（commit `5ab84c5` · 2026-04-22）
- ✅ T-30 NodeService 节点 CRUD + 权限校验（commit `8884339` · 2026-04-22）
- ✅ T-31 GroupService 权限组 CRUD + 成员管理（commit `fb7f881` · 2026-04-22）
- ✅ T-32 Node Handlers + Permission Handlers（commit `0917b52` · 2026-04-23）
- ✅ T-33 Group Handlers + Admin Handlers（commit `5c090e6` · 2026-04-23）
- ✅ T-34 Router 装配 + 集成测试（commit `4e3df4a` · 2026-04-23）
- ✅ T-35 API endpoints + types 补全（commit `d9ce610` · 2026-04-23）
- ✅ T-36 MainShell + 侧栏目录树（commit `499390c` · 2026-04-23）
- ✅ T-37 右键菜单 + 节点 CRUD 操作（commit `566b2a7` · 2026-04-23）
- ✅ T-38 权限设置弹窗（commit `bb8f041` · 2026-04-27）
- ✅ T-39 AdminUsersPage 实现（commit `cd95275` · 2026-04-27）
- ✅ T-40 AdminGroupsPage 实现（commit `6027bf6` · 2026-04-27）
- ✅ T-41 端到端冒烟验证（commit `6a5c19b` · 2026-04-27）— smoke_nodetree.sh 28/28 PASS

**M3 Auth 可登录 ✅ 已关单（2026-04-22）**
- ✅ Auth Epic 设计文档冻结（auth.md v1.0 · 2026-04-21）
- ✅ T-15 Models + Repository 层（commit `7aa137f` · 2026-04-21）
- ✅ T-16 Auth Service 核心逻辑（commit `20eadee` · 2026-04-21）
- ✅ T-17 User Service 查询与更新（commit `cc5794e` · 2026-04-21）
- ✅ T-18 鉴权中间件与角色校验（commit `709f690` · 2026-04-21）
- ✅ T-19 Auth Handlers + 路由装配（commit `bf4342d` · 2026-04-22）
- ✅ T-20 User Handlers: GetMe / UpdateMe / ForceLogout（commit `a855a42` · 2026-04-22）
- ✅ T-21 Manager Seed 脚本（commit `cd64b08` · 2026-04-22）— DB 验证待 Docker 启动后补跑
- ✅ T-22 前端登录/注册页 UI（2026-04-22）— 待提交
- ✅ T-23 AuthContext + API client 填充（2026-04-22）— 待提交
- ✅ T-24 端到端冒烟验证（commit `e97c0dc` · 2026-04-22）— smoke_auth.sh 15/15 PASS

**M2 UI 视觉定调 ✅ 已关单（2026-04-21）**
- ✅ 视觉方向选定：方向 B "清晨"（浅色、通透、柔和）
- ✅ 原型 4 页面完成：登录页、主界面编辑器、权限弹窗、管理后台（用户/组/节点 3 tab）
- ✅ design-system.md v1.0 冻结签字（`docs/ui/design-system.md` · 2026-04-21）

**M1 Infra ✅ 已关单（2026-04-21）**
- ✅ v1.0 Baseline 冻结（requirements.md · 2026-04-16）
- ✅ Epic 1 Infra 详细设计（infra.md v1.1 · 2026-04-16）
- ✅ T-01 Go module + 目录骨架（commit `4f677c9` · 2026-04-16）
- ✅ T-02 config + logger + `/ping` 健康检查（commit `69e474f` · 2026-04-16）
- ✅ T-02 fix: DB 探针错误透出 + Go 版本锁同步（commit `b19a29c` · 2026-04-16）
- ✅ T-03 golang-migrate 接入 + 0001 扩展迁移（commit `38986cb` · 2026-04-16）
- ✅ T-04 迁移 0002 users/groups/refresh_tokens（commit `b9c3082` · 2026-04-17）
- ✅ T-05 迁移 0003 nodes/node_permissions/tsv trigger（commit `f71d931` · 2026-04-17）
- ✅ T-06 迁移 0004 doc_states/doc_updates/doc_snapshots（commit `1b33ca4` · 2026-04-19）
- ✅ T-07 迁移 0005 uploads（commit `27571c2` · 2026-04-19）
- ✅ T-08 迁移 0006 subscriptions/notifications（commit `ebfe47c` · 2026-04-19）
- ✅ T-09 Gin 路由装配 + 中间件骨架（commit `8540502` · 2026-04-19）
- ✅ T-10 WS 握手骨架 + Hub/HubManager（commit `b90ec04` · 2026-04-20）
- ✅ T-11 前端 Vite + React + 路由骨架（commit `3fc9599` · 2026-04-21）
- ✅ T-12 API client + AuthContext 占位骨架（commit `64cfed0` · 2026-04-21）
- ✅ T-13 docker-compose 三服务联调（commit `c79a5bd` · 2026-04-21）
- ✅ T-14 Makefile + README + smoke.sh（commit `5b8f2f3` · 2026-04-21）
- ✅ M1 DoD 六项全部通过（commit `1edd939` lint 豁免 · 2026-04-21）

**备注**:
- WS force_logout 即时推送降级到 M5；M3 通过 HTTP 路径（token_version → 401）保障
- T-21 AC3/AC4/AC7 的 DB 验证已在 T-24 smoke 中覆盖（seed 幂等 + 全流程跑通）
- T-22/T-23 待补 git commit（代码已就绪）

**下一个 demo 节点**: M5 实时协同 MVP（两个浏览器同时编辑同一文档，能看到对方光标 + 实时同步，断线重连不丢字）

---

## 7. Changelog

| 版本 | 日期 | 变更 | 维护者 |
|---|---|---|---|
| v0.1 | 2026-04-16 | 初稿：M1 ~ M13 里程碑、部署清单、进度追踪 | CC |
| v0.2 | 2026-04-21 | M1 Infra 关单；DoD 六项全部通过；进入 M2 | CC |
| v0.3 | 2026-04-21 | M2 UI 视觉定调关单；design-system.md v1.0 冻结；进入 M3 Auth | CC |
| v0.4 | 2026-04-22 | T-19/T-20/T-21 完成，后端鉴权全栈就绪；进入前端阶段 T-22 | CC |
| v0.5 | 2026-04-22 | M3 Auth 关单；T-15~T-24 全通过；smoke 15/15；进入 M4 目录与权限 | CC |
| v0.6 | 2026-04-22 | M4 Phase 1 推进：T-25~T-27 完成（Models + NodeRepo + NodePermissionRepo）；进入 T-28 GroupRepo | CC |
| v0.7 | 2026-04-22 | M4 Phase 1 完成：T-28 GroupRepo 落地；Phase 1（Repository 层）四任务全部通过；进入 Phase 2 Service 层 | CC |
| v0.8 | 2026-04-22 | M4 Phase 2 完成：T-29~T-31（PermissionService + NodeService + GroupService）全部通过；进入 Phase 3 Handlers 层 | CC |
| v0.9 | 2026-04-23 | T-32 完成：NodeHandler（6 方法）+ PermissionHandler（2 方法）落地；独立 DTO 序列化；进入 T-33 | CC |
| v0.10 | 2026-04-23 | T-33 完成：GroupHandler（7 方法）+ AdminHandler（2 方法）落地；补齐 UserRepo.ListAll/UpdateRole + UserService.ListAll/ChangeRole；Phase 3 Handlers 层仅剩 T-34 Router 装配 | CC |
| v0.11 | 2026-04-23 | T-34 完成：router.go 装配 17 条新路由 + 集成测试闭环通过；**后端 Phase 1~3（T-25~T-34）全部收官**；进入 Phase 4 前端 | CC |
| v0.12 | 2026-04-23 | T-35 完成：endpoints.ts 补齐 11 条新端点 + types.ts 全部 M4 请求/响应类型落地；进入 T-36 侧栏目录树 | CC |
| v0.13 | 2026-04-23 | T-36 完成：MainShell 正式布局 + Sidebar/NodeTree 按需展开目录树 + localStorage 持久化；进入 T-37 右键菜单 | CC |
| v0.14 | 2026-04-23 | T-37 完成：NodeContextMenu + InlineNodeInput 落地；右键新建/重命名/删除 + 局部刷新；进入 T-38 权限弹窗 | CC |
| v0.15 | 2026-04-27 | T-38 完成：PermissionDialog 落地；权限列表展示/增删改/替换式保存/继承提示全流程可用；进入 T-39 AdminUsersPage | CC |
| v0.16 | 2026-04-27 | T-39 完成：AdminShell 正式布局 + AdminUsersPage 用户管理表格落地；改角色/强制下线/行级反馈/三态全可用；进入 T-40 AdminGroupsPage | CC |
| v0.17 | 2026-04-27 | T-40 完成：AdminGroupsPage 权限组管理页落地；组 CRUD + 成员展开/添加/移除 + 新建弹窗全流程可用；进入 T-41 冒烟验证 | CC |
| v0.18 | 2026-04-27 | **M4 关单**：T-41 完成，smoke_nodetree.sh 28/28 PASS；T-25~T-41 全部通过；进入 M5 实时协同 MVP 设计阶段 | CC |
| v0.19 | 2026-04-27 | M5 Collab Epic 设计冻结（collab.md v1.0）：TipTap+Y.js 协同架构、WS 协议、Hub 改造、Compact sidecar、任务 T-42~T-52 拆分完成；进入 T-42 | CC |
| v0.20 | 2026-04-29 | T-42 完成（commit cbf85ad）：DocState/DocUpdate models + DocStateRepo 落地，AC1~AC18 全通过；进入 T-43 DocUpdateRepo 验收 | CC |
