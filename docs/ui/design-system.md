# FPGWiki Design System

> **状态**: 冻结 · 2026-04-21
> **视觉方向**: 方向 B "清晨" — 浅色、通透、柔和
> **原型文件**: `html/fpgwiki-direction-b-complete.html`
> **上游**: [`docs/requirements.md`](../requirements.md) §9 前端页面结构

---

## 1. 视觉方向

**风格关键词**: 清新、通透、纸张感、低饱和度、中文友好

- 浅色为主，不做暗色主题（MVP 阶段）
- 编辑器区域采用白底纸张卡片，营造"书写"感
- 圆角柔和，阴影轻薄
- 登录页使用三色渐变背景（蓝→绿→黄），传达"清晨"氛围

---

## 2. 色板（Color Tokens）

### 2.1 灰阶（Neutral）

| Token | 值 | 用途 |
|---|---|---|
| `--white` | `#ffffff` | 卡片/纸张背景 |
| `--gray-50` | `#f8fafb` | 页面底色、hover 背景 |
| `--gray-100` | `#f0f3f5` | 输入框背景、toolbar 背景、分隔线 |
| `--gray-200` | `#e2e7eb` | 边框、分隔线 |
| `--gray-300` | `#c9d1d8` | placeholder、禁用态边框 |
| `--gray-400` | `#9aa5b0` | 次要文字、标签、图标 |
| `--gray-500` | `#6b7a87` | 正文辅助文字 |
| `--gray-600` | `#4a5968` | 标签文字、表单 label |
| `--gray-700` | `#334155` | 正文文字 |
| `--gray-800` | `#1e293b` | 标题、强调文字 |
| `--gray-900` | `#0f172a` | 最深文字、品牌色背景 |

### 2.2 品牌色（Primary）

| Token | 值 | 用途 |
|---|---|---|
| `--primary` | `#0ea5e9` | 主按钮、active 状态、链接 |
| `--primary-dark` | `#0284c7` | 按钮渐变终点、code 颜色 |
| `--primary-light` | `#e0f2fe` | 工作区图标背景 |
| `--primary-bg` | `rgba(14,165,233,0.06)` | active 行淡蓝底、hover 态 |

### 2.3 语义色（Semantic）

| Token | 值 | 语义 | 用途 |
|---|---|---|---|
| `--green` | `#10b981` | 成功/在线/edit | 在线状态、edit 权限色、协同光标 |
| `--green-light` | `#d1fae5` | — | 标签背景 |
| `--green-bg` | `rgba(16,185,129,0.08)` | — | Group Leader 角色标签底色 |
| `--orange` | `#f59e0b` | 警告/readable | 继承横幅、readable 权限色 |
| `--orange-light` | `#fef3c7` | — | 标签背景 |
| `--orange-bg` | `rgba(245,158,11,0.08)` | — | 继承横幅底色 |
| `--rose` | `#f43f5e` | 危险/删除 | 通知红点、强制下线按钮、删除操作 |
| `--rose-light` | `#ffe4e6` | — | 标签背景 |
| `--rose-bg` | `rgba(244,63,94,0.06)` | — | 删除 hover 底色 |
| `--purple` | `#8b5cf6` | 组标识 | 组类型标签、组头像 |
| `--purple-bg` | `rgba(139,92,246,0.08)` | — | 组标签底色 |

### 2.4 权限级别 → 颜色映射

| 权限级别 | 边框/文字色 | 背景 |
|---|---|---|
| `manage` | `--primary` | `--primary-bg` |
| `edit` | `--green` | `--green-bg` |
| `readable` | `--orange` | `--orange-bg` |
| `none` | `--gray-300` / `--gray-400` | `--gray-50` |

### 2.5 角色标签 → 颜色映射

| 角色 | 底色 | 文字色 |
|---|---|---|
| Manager | `--primary-bg` | `--primary` |
| Group Leader | `--green-bg` | `--green` |
| Member | `--gray-100` | `--gray-500` |

---

## 3. 字体（Typography）

### 3.1 字体家族

| Token | 值 | 用途 |
|---|---|---|
| `--font-cn` | `'LXGW WenKai', serif` | 中文内容：标题、正文、表单 label、按钮文字 |
| `--font-ui` | `'DM Sans', sans-serif` | UI 元素：品牌名、标签、快捷键、角色标签 |

> **注意**: 霞鹜文楷为装饰性衬线体。如果后续长文档阅读体验不佳，可考虑正文降级为系统字体（`-apple-system, "PingFang SC", "Microsoft YaHei", sans-serif`），仅标题保留文楷。

### 3.2 字号规范

| 场景 | font 简写 |
|---|---|
| 登录页主标题 | `700 48px var(--font-cn)` |
| 编辑器文档标题 | `700 30px var(--font-cn)` |
| 管理后台页标题 | `700 22px var(--font-cn)` |
| 编辑器 H2 | `700 20px var(--font-cn)` |
| 品牌名 | `700 17px var(--font-ui)` |
| 弹窗节点名 | `600 16px var(--font-cn)` |
| 编辑器正文 | `400 15px/1.9 var(--font-cn)` |
| 表格正文/表单值 | `400 14px var(--font-cn)` |
| 表单 label | `500 13px var(--font-cn)` |
| 按钮文字 | `500~600 13px var(--font-cn)` |
| 树节点文字 | `400 13px var(--font-cn)` |
| 小标签/meta | `400 12px var(--font-cn)` |
| section label（大写） | `600 11px var(--font-ui)` |
| 角色/类型标签 | `500 11px var(--font-cn)` |
| 快捷键 | `500 11px var(--font-ui)` |
| 堆叠头像文字 | `600 10px var(--font-ui)` |

---

## 4. 间距与圆角（Spacing & Radius）

### 4.1 圆角

| Token | 值 | 用途 |
|---|---|---|
| `--radius-sm` | `6px` | 树节点、小按钮、头像、表格按钮 |
| `--radius-md` | `10px` | 输入框、卡片内小区域、工具栏 |
| `--radius-lg` | `16px` | 卡片、弹窗、纸张编辑器、数据表格 |
| `--radius-full` | `9999px` | 搜索框、pill 标签、pill tab、角色标签 |

### 4.2 间距规律

- Sidebar 宽度：`256px`
- Topbar 高度：`56px`
- 编辑器纸张最大宽度：`740px`
- 编辑器纸张内间距：`48px 56px 80px`（上 左右 下）
- 通用卡片内间距：`20px`
- 通用组间距：`16px`

---

## 5. 阴影（Shadows）

| Token | 值 | 用途 |
|---|---|---|
| `--shadow-xs` | `0 1px 2px rgba(0,0,0,0.04)` | tab pill active 态 |
| `--shadow-sm` | `0 2px 8px rgba(0,0,0,0.06)` | 编辑器纸张、组卡片 hover |
| `--shadow-md` | `0 4px 16px rgba(0,0,0,0.08)` | — |
| `--shadow-lg` | `0 12px 40px rgba(0,0,0,0.1)` | 登录卡片 |
| `--shadow-xl` | `0 20px 60px rgba(0,0,0,0.15)` | 权限弹窗 |

---

## 6. 动效（Motion）

| Token | 值 |
|---|---|
| `--transition` | `0.18s ease` |

### 6.1 约定

- 所有 hover/focus 过渡统一使用 `var(--transition)`
- 按钮 hover: `translateY(-1px)` + 阴影加深
- 弹窗入场: `scale(0.96) + translateY(8px)` → `scale(1) + translateY(0)`，0.25s ease
- 登录卡片入场: `translateY(12px)` → `translateY(0)`，0.4s ease
- 在线脉冲: `opacity 1→0.4→1`，2s infinite
- 仅动画 `transform`、`opacity`、`box-shadow` — 不动画布局属性

---

## 7. 组件规范

### 7.1 按钮

| 类型 | 样式 |
|---|---|
| Primary（保存、登录） | 渐变背景 `primary→primary-dark`、白色文字、`shadow 0 2px 8px rgba(14,165,233,0.3)` |
| Secondary（取消） | 白底、`gray-200` 边框、`gray-600` 文字 |
| Danger（强制下线） | 透明底、`rose` 边框+文字、hover 填充 `rose` 底+白字 |
| Dashed（新建文档、添加权限） | 透明底、`gray-300` 虚线边框、hover 变 `primary` |
| Toolbar | 无边框、`gray-400` 文字、hover 白底、active `primary-bg` 底 |

### 7.2 输入框

- 边框: `1.5px solid var(--gray-200)`
- 圆角: `var(--radius-md)`
- Focus: 边框变 `primary` + `box-shadow: 0 0 0 3px var(--primary-bg)`
- Placeholder: `var(--gray-300)`

### 7.3 下拉选择器（权限级别）

- 基础样式同输入框
- 根据选中值动态变色：manage 蓝、edit 绿、readable 橙、none 灰

### 7.4 标签（Tag / Badge）

- Pill 形状（`radius-full`）
- 背景使用对应语义色的 `*-bg` 变量
- 文字使用对应语义色

### 7.5 头像

- 尺寸：34px（topbar）、32px（sidebar/表格）、30px（权限行）、26px（协同/堆叠）
- 圆角: `var(--radius-sm)`（不是圆形）
- 背景: 渐变或纯语义色
- 堆叠时 `margin-left: -5~6px` + 白色 2px border

### 7.6 数据表格

- 外框: `radius-lg`、`gray-100` border、`shadow` 无
- 表头: `gray-50` 背景、`11px` 大写标签
- 行: hover 变 `gray-50`，最后一行无下边框
- 操作列: 内联按钮/下拉

### 7.7 卡片

- 背景白、`radius-lg`、`gray-100` 边框
- Hover: 边框加深 + `shadow-sm`
- 新建卡片: `gray-50` 底 + 虚线边框

### 7.8 弹窗

- 遮罩: `rgba(15,23,42,0.45)` + `backdrop-filter: blur(4px)`
- 弹窗体: 白底、`radius-lg`、`shadow-xl`
- 入场动画: scale + translate

### 7.9 侧栏目录树

- 节点: `13px` 文楷、`7px 12px` padding
- Active: `primary-bg` 底 + `primary` 文字 + `font-weight: 500`
- Hover: `gray-50` 底
- 子节点缩进: `padding-left: 20px`
- Section 标签: `11px` 大写 DM Sans

---

## 8. 布局结构

```
┌─────────────────────────────────────────────────┐
│  Topbar (56px)  [Brand] [Search ⌘K] [Bell] [Av] │
├──────────┬──────────────────────────────────────┤
│ Sidebar  │  Main Content                        │
│ (256px)  │                                      │
│          │  ┌─ Editor Topbar ──────────────┐    │
│ Workspace│  │ Breadcrumb   Collab/History   │    │
│ + New    │  └──────────────────────────────┘    │
│          │                                      │
│ Tree     │  ┌─ Paper (max 740px) ──────────┐   │
│  ├ 收藏  │  │ Title                         │   │
│  └ 文档  │  │ Meta                          │   │
│    ├ 📁  │  │ Toolbar                       │   │
│    │ └📄 │  │ Body (richtext)               │   │
│    └ 📁  │  │                               │   │
│          │  └──────────────────────────────┘    │
│ ─────────│                                      │
│ ⚙ Admin  │                                      │
└──────────┴──────────────────────────────────────┘
```

---

## 9. 页面清单与状态

| 页面 | 原型状态 | 关联 Feature |
|---|---|---|
| 登录/注册 | **已完成** | F-40 |
| 主界面（编辑器） | **已完成** | F-41, F-42, F-43 |
| 权限设置弹窗 | **已完成** | F-21 |
| 管理后台 — 用户管理 | **已完成** | F-06, F-25, F-45 |
| 管理后台 — 权限组管理 | **已完成** | F-22, F-24, F-45 |
| 管理后台 — 节点权限总览 | **已完成** | F-21, F-45 |
| 通知面板 | **待补** — M8 Epic 时设计 | F-46 |
| 历史版本抽屉 | **待补** — M6 Epic 时设计 | F-44 |
| 搜索结果面板 | **待补** — M9 Epic 时设计 | F-30 |

---

## 10. 已知待调整项

M2 签字后不阻塞开发，记入各 Epic 迭代时处理：

1. **正文字体**: 霞鹜文楷长文档阅读可能偏累。如 M5 阶段编辑器实际体验不佳，降级正文为系统字体。
2. **快捷键标注**: 原型写 `⌘K`，Windows 下应为 `Ctrl+K`，前端实现时做平台检测。
3. **编辑器占位文案**: 原型中 "Elasticsearch" 应为 "PostgreSQL tsvector"，"React 18" 应为 "React 19"。仅原型展示问题，不影响实现。
4. **通知面板**: 铃铛点击后的下拉/抽屉在 M8 Epic 时补设计。
5. **Manager 提示**: 权限弹窗中应为 Manager 用户显示短路覆盖提示（"Manager 角色默认拥有 manage 权限"）。

---

## 11. Changelog

| 版本 | 日期 | 变更 | 维护者 |
|---|---|---|---|
| v1.0 | 2026-04-21 | 初版冻结：方向 B 清晨，9 大 tokens 分类，7 个页面/弹窗原型签字 | CC |
