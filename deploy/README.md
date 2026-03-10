# HotSpot 数据库脚本

本目录包含 HotSpot 项目的数据库管理脚本，以及 Docker 编译/测试环境准备脚本。

> **📌 环境说明**
> - 本地原生 PostgreSQL 脚本默认按 `config.sh` 中的 `POSTGRES_VERSION=17` 检测工具链。
> - Docker 开发栈使用 `deploy/docker-compose.yml` 中定义的独立数据库配置，供 `cargo sqlx prepare`、`cargo check`、`cargo test` 等编译验证使用。
> - `deploy/docker-compose.yml` 中的 `backend`/`frontend` 已放入 `runtime` profile，默认 `docker compose up -d` 只会启动基础设施层。

## 📦 脚本列表

| 脚本 | 用途 | 职责 |
|------|------|------|
| `config.sh` | 配置文件 | 统一管理版本号和数据库配置 |
| `setup_database.sh` | 数据库管理脚本 | 初始化模板 + 创建/重置用户数据库 |
| `prepare_docker.sh` | Docker 编译环境准备脚本 | 拉起 PostgreSQL、执行迁移、生成 SQLx 离线元数据 |
| `init_template.sql` | 模板初始化 SQL | 创建模板数据库（由脚本调用） |

## ⚙️ 配置管理

### 修改 PostgreSQL 版本

编辑 `config.sh` 文件：

```bash
# 修改版本号
export POSTGRES_VERSION="17"  # 改为你安装的版本

# 其他可配置项
export DB_USER="hotspot_user"
export DB_TEMPLATE="hotspot_template"
export DB_NAME="hotspot_db"
```

修改后重新运行初始化脚本即可。

> **💡 自动检测**  
> `config.sh` 会自动检测 PostgreSQL 工具路径和超级用户，实现跨平台兼容：
> - **工具路径**：支持标准安装、版本后缀命令、Homebrew 路径
> - **超级用户**：Linux 使用 `postgres` 用户，macOS Homebrew 使用当前用户
> - **无需修改**：同一套脚本可在 Linux 和 macOS 上运行

### 查看当前配置

```bash
source ./deploy/config.sh
show_config
```

## 🎯 设计理念

**职责分离**：
- **配置文件** (`config.sh`) - 统一管理所有配置（一处修改，处处生效）
- **SQL 脚本** (`init_template.sql`) - 只创建模板数据库（一次性操作）
- **Shell 脚本** (`setup_database.sh`) - 管理用户数据库（可重复执行）

这种设计的好处：
- ✅ 版本号集中管理，修改一处即可
- ✅ 模板创建是一次性的，不会重复执行
- ✅ 用户数据库可以反复创建/删除/重置
- ✅ 职责单一，更符合 Unix 哲学

## 🚀 快速开始

### 单机推荐拓扑

一台电脑上建议同时保留三套互不冲突的角色：

| 角色 | 运行方式 | 默认宿主机端口 | 说明 |
|------|----------|----------------|------|
| 开发 | 本机进程 | backend `5496` / frontend `5173` | 改代码、热更新、调试 |
| 测试 | 本机进程 + Docker 基础设施 | PostgreSQL `5432` / Redis `6380` | 测试只连 `hotspot_test` 和 `redis-test` |
| 运行预演 | Docker `runtime` profile | backend `8080` / frontend `3000` | 模拟部署后运行状态 |

这样在同一台机器上可以做到：开发不中断、测试不污染、运行预演可并存。

### 编译 / 测试环境准备

如果你要在全新机器上运行后端类型检查、测试或 Docker 离线构建，先准备 SQLx 编译环境：

```bash
./deploy/prepare_docker.sh

# 然后即可运行
cd backend
DATABASE_URL="postgresql://hotspot:hotspot@127.0.0.1:5432/hotspot" cargo check
DATABASE_URL="postgresql://hotspot:hotspot@127.0.0.1:5432/hotspot_test" REDIS_URL="redis://127.0.0.1:6380/0" TEST_REDIS_URL="redis://127.0.0.1:6380/0" cargo test
```

这个脚本会自动：
- 启动 `deploy/docker-compose.yml` 中的 PostgreSQL 容器
- 启动独立的测试 Redis 容器（`6380`）
- 等待数据库就绪
- 分别对运行时数据库和测试数据库执行 `cargo sqlx migrate run`
- 执行 `cargo sqlx prepare --workspace`，生成离线编译所需元数据

### 自动模式（推荐）

```bash
# 脚本会自动检测并执行需要的操作
# macOS: 直接运行
./deploy/setup_database.sh

# Linux: 需要 sudo（切换到 postgres 用户）
sudo ./deploy/setup_database.sh

# 配置认证
# macOS: 通常不需要修改 pg_hba.conf
# Linux: 需要配置后重新加载
sudo -u postgres pg_ctl reload

# 运行迁移
cd backend
export DATABASE_URL="postgresql://hotspot_user@localhost:5432/hotspot_db"
sqlx migrate run
```

### 手动模式

```bash
# 1. 初始化模板数据库
# macOS: 直接运行
./deploy/setup_database.sh --init-template
# Linux: 需要 sudo
sudo ./deploy/setup_database.sh --init-template

# 2. 配置 pg_hba.conf 认证
# macOS: 通常不需要
# Linux: 需要重新加载配置
sudo -u postgres pg_ctl reload

# 3. 创建用户数据库
./deploy/setup_database.sh --create

# 4. 运行迁移
cd backend && sqlx migrate run
```

## 🔄 日常开发

### 使用脚本重置

```bash
# 重置数据库（删除 + 从模板重建）
./deploy/setup_database.sh --reset
```

### 使用 SQLx 管理

```bash
cd backend

# 方式 1：分步操作
sqlx database drop     # 删除数据库（会提示确认）
sqlx database create   # 从模板重新创建
sqlx migrate run       # 运行迁移

# 方式 2：一步到位
sqlx database reset    # 等同于 drop + create + migrate
```

## 💡 使用场景

| 场景 | macOS | Linux |
|------|-------|-------|
| 首次设置 | `./setup_database.sh` | `sudo ./setup_database.sh` |
| 重建模板 | `./setup_database.sh --init-template` | `sudo ./setup_database.sh --init-template` |
| 创建数据库 | `./setup_database.sh --create` | `./setup_database.sh --create` |
| 重置数据库 | `./setup_database.sh --reset` | `./setup_database.sh --reset` |
| 查看帮助 | `./setup_database.sh --help` | `./setup_database.sh --help` |

## 📚 其他文档

- [部署运维文档](deployment.md) - 完整的生产环境部署指南

## 🧪 环境约定

- `DATABASE_URL`：本地原生 PostgreSQL，默认来自 `config.sh`
- `DOCKER_DATABASE_URL`：Docker 开发栈数据库，默认 `postgresql://hotspot:hotspot@127.0.0.1:5432/hotspot`
- `TEST_DATABASE_URL`：测试专用 PostgreSQL 连接串，默认 `postgresql://hotspot:hotspot@127.0.0.1:5432/hotspot_test`
- `TEST_REDIS_URL`：测试专用 Redis 连接串，默认 `redis://127.0.0.1:6380/0`

## 🐳 运行预演层

需要在同一台机器上启动“接近部署态”的容器服务时，使用 `runtime` profile：

```bash
docker compose -f deploy/docker-compose.yml --profile runtime up -d backend frontend
```

如果默认端口冲突，可以临时覆盖 host 端口：

```bash
HOTSPOT_BACKEND_HOST_PORT=18080 \
HOTSPOT_FRONTEND_HOST_PORT=13000 \
HOTSPOT_MINIO_API_HOST_PORT=19000 \
HOTSPOT_MINIO_CONSOLE_HOST_PORT=19001 \
docker compose -f deploy/docker-compose.yml --profile runtime up -d backend frontend minio
```

停止运行预演层：

```bash
docker compose -f deploy/docker-compose.yml --profile runtime stop backend frontend
```

## ⚙️ 配置摘要

- **用户**：`hotspot_user`（CREATEDB 权限，无密码本地认证）
- **模板数据库**：`hotspot_template`（包含 PostGIS 等扩展，不可连接）
- **用户数据库**：`hotspot_db`（基于模板创建）

## ⚠️ 注意事项

这些脚本仅用于**开发环境**。
