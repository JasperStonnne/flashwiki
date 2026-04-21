# FPGWiki

FPGWiki 是一个面向团队协作的在线文档系统，支持权限管理、实时协同、通知与检索等能力。

## 前置依赖

- Go 1.25+
- Node.js 20
- pnpm 9（建议通过 `corepack` 管理）
- Docker + Docker Compose
- GNU Make（Windows 用户请使用 WSL 或 Git Bash 环境）

## 快速上手

1. 克隆仓库并进入目录。
2. 设置必须环境变量 `JWT_SECRET`：
   - Bash: `export JWT_SECRET=dev_secret`
   - PowerShell: `$env:JWT_SECRET='dev_secret'`
3. 启动服务：`make up`
4. 打开以下地址验证：
   - Frontend: [http://localhost:5173](http://localhost:5173)
   - Backend health: [http://localhost:8080/ping](http://localhost:8080/ping)
5. 停止服务：`make down`

## Makefile 命令速查

| 命令 | 作用 |
|---|---|
| `make up` | `docker compose up --build` |
| `make down` | `docker compose down` |
| `make logs` | `docker compose logs -f` |
| `make migrate-up` | 执行全部 up 迁移 |
| `make migrate-down` | 回滚 1 步迁移 |
| `make migrate-new name=xxx` | 生成新迁移文件 |
| `make backend-test` | 运行后端测试 |
| `make frontend-dev` | 启动前端开发服务 |
| `make frontend-test` | 运行前端测试 |
| `make frontend-typecheck` | 前端 TypeScript 类型检查 |
| `make smoke` | 运行 e2e 冒烟脚本 |

## 冒烟脚本

`scripts/smoke.sh` 会执行完整流程：

`compose up -> 轮询 /ping -> WS 收到 not_implemented -> compose down`

执行方式：

```bash
bash scripts/smoke.sh
```

Windows 下请在 WSL/Git Bash 中运行（PowerShell 默认不提供 GNU Bash）。

## 目录结构概览

```text
backend/             Go 后端服务
frontend/            Vite + React 前端
docker/              容器构建资源
scripts/             脚本工具（含 smoke）
docs/                需求与 Epic 文档
docker-compose.yml   本地三服务编排
```

## 文档入口

- [产品需求基线](docs/requirements.md)
- [Infra Epic 设计与任务清单](docs/epics/infra.md)
