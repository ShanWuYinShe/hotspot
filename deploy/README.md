# HotSpot 部署与数据库（Docker）

本目录包含 HotSpot 项目的 Docker 部署配置和数据库相关说明，开发/测试与生产环境使用不同的 compose 文件组合。

> **📌 环境说明**
> - 开发/测试环境使用 `deploy/dev/` 下的 compose 文件组合。
> - 生产环境使用 `deploy/prod/` 下的 compose 文件组合。

## 📦 关键文件

| 文件 | 用途 |
|------|------|
| `dev/docker-compose.infra.yml` | 开发/测试环境基础设施（Postgres、Redis、MinIO） |
| `dev/docker-compose.app.yml`   | 开发/测试环境应用层（Backend、Frontend、Nginx） |
| `prod/docker-compose.infra.yml` | 生产环境基础设施（Postgres、Redis、MinIO） |
| `prod/docker-compose.app.yml`   | 生产环境应用层（Backend、Frontend、Nginx） |

## ⚙️ 配置管理

数据库用户、库名、密码等配置统一在对应环境目录下的 compose 文件（尤其是 `docker-compose.infra.yml` 的 `postgres` 服务）中管理。如需调整，请直接修改对应文件并重新启动容器。

## 🎯 设计理念

**职责分离（基于 docker-compose）**：
- **infra compose** - 管理数据库、缓存、对象存储等基础设施容器。
- **app compose** - 管理后端、前端、Nginx 等应用层容器。

## 🚀 快速开始

### 单机推荐拓扑

一台电脑上建议同时保留三套互不冲突的角色（可以按需选择是否全部启用 Docker）：

| 角色 | 运行方式 | 默认宿主机端口 | 说明 |
|------|----------|----------------|------|
| 开发 | 本机进程 | backend `5496` / frontend `5173` | 改代码、热更新、调试 |
| 测试 | 本机测试进程 + Docker 基础设施 | PostgreSQL `5432` / Redis `6379` | 测试可使用独立数据库或 Redis DB 索引隔离 |
| 运行预演 | Docker compose (dev/app.yml) | backend `8080` / frontend `3000` / Nginx `8888` | 模拟部署后运行状态 |

### 编译 / 测试环境准备（Docker 推荐）

如果你要在全新机器上运行后端类型检查、测试或 Docker 离线构建，推荐使用 dev 环境的 compose 文件：

```bash
# 启动基础设施（PostgreSQL + Redis + MinIO）
docker compose -f deploy/dev/docker-compose.infra.yml up -d

# 然后即可运行
cd backend
DATABASE_URL="postgresql://hotspot:hotspot@127.0.0.1:5432/hotspot" cargo check

# 测试环境推荐：使用独立数据库或 Redis DB 索引隔离
DATABASE_URL="postgresql://hotspot:hotspot@127.0.0.1:5432/hotspot_test" \
  REDIS_URL="redis://127.0.0.1:6379/0" cargo test
```

## 🔄 日常开发与使用场景

所有与数据库/缓存相关的开发和测试场景，均建议通过 `deploy/dev/docker-compose.infra.yml` 管理：

- **首次设置**：`docker compose -f deploy/dev/docker-compose.infra.yml up -d`
- **重置容器数据**：通过 `docker compose -f deploy/dev/docker-compose.infra.yml down -v` + 重新 `up` 完成
- **查看和修改配置**：直接阅读和修改 `deploy/dev/docker-compose.infra.yml`

## 📚 其他文档

- [部署运维文档](deployment.md) - 完整的生产环境部署指南

## 🧪 环境约定

- `DOCKER_DATABASE_URL`：Docker 开发栈数据库，默认 `postgresql://hotspot:hotspot@127.0.0.1:5432/hotspot`
- 测试环境推荐复用 `DATABASE_URL` / `REDIS_URL` 这组 key，通过 `.env.test` 或命令行覆盖值指向测试专用数据库或 Redis DB 索引

## 🐳 运行预演层

需要在同一台机器上启动“接近部署态”的容器服务时，使用 dev 环境 app compose：

```bash
docker compose -f deploy/dev/docker-compose.infra.yml -f deploy/dev/docker-compose.app.yml up -d nginx backend frontend
```

如果默认端口冲突，可以临时覆盖 host 端口：

```bash
HOTSPOT_BACKEND_HOST_PORT=18080 \
HOTSPOT_FRONTEND_HOST_PORT=13000 \
HOTSPOT_MINIO_API_HOST_PORT=19000 \
HOTSPOT_MINIO_CONSOLE_HOST_PORT=19001 \
HOTSPOT_NGINX_HTTP_PORT=18888 \
  docker compose -f deploy/dev/docker-compose.infra.yml -f deploy/dev/docker-compose.app.yml up -d nginx backend frontend minio
```

停止运行预演层：

```bash
docker compose -f deploy/dev/docker-compose.infra.yml -f deploy/dev/docker-compose.app.yml stop nginx backend frontend
```

## ⚙️ 配置摘要

- **用户**：`hotspot_user`（CREATEDB 权限，无密码本地认证）
- **模板数据库**：`hotspot_template`（包含 PostGIS 等扩展，不可连接）
- **用户数据库**：`hotspot`（基于模板创建）

## ⚠️ 注意事项

这些脚本仅用于**开发环境**。
