# Q&A Log

> 开发过程中的概念性问题与解答记录。

---

### Q: 每个 Step 大致在干什么，为什么流程这么长？
**日期**: 2026-04-19

- Step A: CC 下发任务卡 + AC（1 分钟）
- Step B: 实现者写规划，CC 确认（2 分钟）
- Step C: 写代码（2~10 分钟）
- Step D: 跑验证（docker + psql，5~10 分钟）
- READY: 贴验证报告，CC 确认关闭（1 分钟）
- 大部分时间花在 Step D 数据库验证上，因为 SQL 没有编译器兜底，跑过才算数

---

### Q: 每个 T 都大概干了什么，为什么前期准备这么长，有没有直接调用现成的？
**日期**: 2026-04-19

- T-01~T-03: Go 骨架 + 配置 + 迁移工具接入
- T-04~T-08: 五个迁移任务，逐组建表（users → nodes → docs → uploads → notifications）
- T-09~T-14: 真正写 Go/前端代码（路由、WS、React、docker-compose）
- 拆成多个迁移是为了回滚粒度和外键依赖顺序，但任务下发可以合并
- 自定义 schema + zhparser + 特定业务约束没有现成脚手架能直接生成

---

### Q: 项目 API 的设计是怎样的？
**日期**: 2026-04-19

- 统一信封格式：`{success, data, error}`
- 30 个 REST 端点覆盖鉴权、目录、权限、快照、图床、通知、搜索
- 1 个 WebSocket 端点 `/api/ws?doc=<node_id>` 用于实时协同
- 鉴权分三层：无鉴权、JWT 必需、节点权限校验
- 详见 `requirements.md` 第 8 章

---

### Q: 写一个项目一定要规划 API 吗？为什么说做到最后项目就是个 API？
**日期**: 2026-04-19

- 不一定提前规划，取决于项目复杂度；多人协作 / 前后端分离时必须先定 API 合同
- "项目就是 API" 的意思：每一层对外暴露的都是接口（前端→后端→数据库→OS）
- API 是系统的骨架，实现是填充的肉；骨架定了肉可以换，骨架歪了整个都歪
- 好的 API 设计允许内部重构而不影响调用方

---

### Q: 所以最后整个网站就是前端后端两个 API 互相调用的结果？
**日期**: 2026-04-19

- 是的，本质就是一串 API 调用串起来的流水线
- 用户 → 前端（UI 接口）→ 后端（HTTP/WS）→ 数据库（SQL）→ 原路返回
- 每一层都在提供接口、消费接口
- API 设计先行，定好接头形状，每段管道可以独立施工

---

### Q: Gin 框架是什么，为何与 Go 绑定在一起？
**日期**: 2026-04-19

- Gin 是 Go 语言的 HTTP Web 框架，封装了路由匹配、参数解析、JSON 序列化、中间件链
- 并非绑定，而是 Gin 是用 Go 写的、给 Go 用的，就像 Express 之于 Node.js
- Gin 是 Go 生态最流行的 Web 框架（80k+ star）
- 选 Gin 因为成熟、中间件开箱即用、性能够用、API 简洁

---

### Q: Gin 框架调用 API 吗？框架里的内容和含义是什么？
**日期**: 2026-04-19

- Gin 不调用 API，而是接收并处理 API 请求
- 五个核心概念：
  - **路由（Router）**: 地址 → 函数的映射表
  - **Handler**: 处理一个具体请求的函数
  - **Context**: 一次请求的工具箱（读参数、写响应）
  - **中间件（Middleware）**: 洋葱皮模型，请求前后的通用逻辑（日志、鉴权、异常捕获等）
  - **路由分组（Group）**: 给一批路由加统一前缀和中间件

---

### Q: Router 就像 Hub 吗？为啥不能用 map 或者 KV 引擎？
**日期**: 2026-04-19

- Router 和 Hub 都是分发器：东西进来 → 看标签 → 送到对应地方
- 可以用 map，但 map 不支持动态参数（`/nodes/:id`）、中间件链、通配符
- Gin Router 内部用 radix tree（压缩前缀树），O(路径长度) 查找，还能处理动态参数
- Router 本质是增强版 map

---

### Q: 为啥不用 KV 存储引擎？
**日期**: 2026-04-19

- 路由匹配在内存里、纳秒级、每次请求都要做；KV 引擎（Redis 等）是给持久化存储用的
- 用 Redis 做路由要走网络，比内存查找慢约 1000 倍
- 路由表生命周期是进程内的，启动时写死、运行期不变，不需要持久化
- 引入外部依赖还增加了故障点（Redis 挂了服务就挂了）

---

### Q: Docker Desktop 是什么，为什么每次都要启动，类似于本地数据库吗？
**日期**: 2026-04-19

- Docker Desktop 是让 Windows 能跑 Docker 容器的底层虚拟化引擎
- 我们用 Docker 临时跑带 zhparser 的 PostgreSQL，验证完就扔，不污染系统
- 相比本地安装数据库：一次性、不留常驻服务、zhparser 在 Windows 上难装
- 可在 Docker Desktop Settings 里勾选开机自启，免去每次手动开

---

### Q: 怎么通过 CC 和 Codex 开展瀑布模型项目？
**日期**: 2026-04-19

- CC（Claude Code）= 架构师 + PM，负责冻结需求、写 Epic 设计、拆任务、验收
- Codex（用户）= 实现者，负责写代码、跑验证、提交
- 流程：需求冻结 → Epic 设计 → 拆任务清单 → 逐任务循环（下发→规划→实现→验证→验收）→ 更新 roadmap → 下一个任务
- 文档先行：`requirements.md` 是功能真源，`docs/epics/*.md` 是实施真源，`docs/roadmap.md` 是时间真源
- CC 管"做什么"和"对不对"，Codex 管"怎么做"和"跑起来"

---

### Q: Epic 是什么意思？
**日期**: 2026-04-19

- Epic = 一组相关任务的集合，对应一个大的功能主题
- 层级：项目 → Epic → Task → AC（验收条件）
- 每个 Epic 有自己的详细设计文档（`docs/epics/*.md`），冻结后再拆任务实现
- 术语来自敏捷开发，但在瀑布模型中含义相同

---

### Q: 为什么要验证迁移？为什么要安装 zhparser 扩展？PostgreSQL 相较其他数据库有什么优势？编写数据库用什么语言？
**日期**: 2026-04-19

- **验证迁移**: SQL 没有编译器，拼写错误只有跑了才知道；还要验证约束生效、up/down 对称
- **zhparser**: PostgreSQL 自带分词器按空格切词，中文没空格，需要 zhparser 正确切分中文
- **PostgreSQL 优势**: 严格约束（CHECK/外键）、jsonb 可索引、内置全文搜索 tsvector、扩展生态丰富（zhparser/PostGIS/pg_trgm）
- **语言**: SQL（Structured Query Language），所有关系型数据库通用

---

### Q: 给我讲讲 API 设计
**日期**: 2026-04-20

- **资源建模**: URL 是名词（`/api/nodes`），动作用 HTTP Method（GET/POST/PATCH/DELETE）
- **统一信封**: 所有响应用 `{success, data, error}` 一致格式，前端统一处理
- **嵌套资源**: URL 路径表达归属（`/documents/:id/snapshots`），一般不超过 2 层
- **鉴权分层**: 无鉴权 → JWT 必需 → 节点权限校验，中间件逐层拦截
- **REST + WS 互补**: REST 做一次性操作，WebSocket 做实时双向通信，共用 JWT 鉴权

---

### Q: 详细讲讲 API 设计
**日期**: 2026-04-20

- **资源建模**: 数据库表 → URL 资源，用 HTTP Method 表达 CRUD，URL 只放名词
- **无状态**: 每次请求自带 JWT，服务端不存 session，可水平扩展
- **统一信封**: `{success, data, error}` + 结构化错误码（`token_expired` / `not_found`），前端程序化处理
- **鉴权分层**: 无鉴权 → JWT 中间件 → 节点权限解析 → 角色校验，Gin 中间件洋葱皮模型
- **分页策略**: 偏移分页适合可跳页场景，游标分页适合无限滚动（通知列表）
- **幂等性**: GET/PUT/DELETE 幂等可安全重试；POST 不幂等需防重复提交
- **版本控制**: 加字段不破坏，删/改字段是破坏性改动，需 `/api/v2/` 隔离
- **安全设计**: `none` 返回 404 不暴露存在性；WS 用 subprotocol 传 JWT；uploads 走鉴权 Proxy

---

### Q: 为什么 Docker 在 Windows 上要用虚拟机？不就是本地跑个数据库吗？
**日期**: 2026-04-27

- Docker 容器依赖 Linux 内核特性（namespaces 隔离进程、cgroups 限制资源），Windows 内核没有这些
- Docker Desktop 在 Windows 上通过 WSL2（本质是 Hyper-V 轻量虚拟机）运行一个真正的 Linux 内核
- 所有镜像、容器数据、Postgres 数据都存在 WSL2 虚拟磁盘（.vhdx 文件）里，这是 C 盘空间消耗的主因
- 在 Linux 上 Docker 零虚拟机开销，容器直接跑在宿主内核上；macOS 同理需要虚拟机层
- 数据路径：Windows → WSL2 虚拟机 → Docker 容器 → Postgres 进程 → 数据文件
