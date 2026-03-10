# HotSpot

基于地理位置的社交应用，采用 Monorepo 结构管理多个子项目。

## 技术栈

| 子项目 | 技术栈 | 说明 |
|--------|--------|------|
| `backend/` | Rust / Axum / SQLx / PostGIS | 后端服务 |
| `frontend_svelte/` | SvelteKit / Svelte 5 / TypeScript | Web 前端 |
| `harmonyapp/` | HarmonyOS ArkTS (.ets) | 鸿蒙客户端 |

## 快速开始

### 前置要求

- **Rust**: 1.81+
- **Node.js**: 20+
- **pnpm**: 10+
- **PostgreSQL**: 16+ (需 PostGIS)
- **Redis**: 7.0+

### 本地开发

```bash
# 1. 克隆项目
git clone <repository-url>
cd hotspot

# 2. 启动基础设施层（数据库、Redis、测试 Redis、MinIO）
# 注意：默认不会启动 backend/frontend 容器，避免与本机 cargo run / pnpm dev 冲突
docker compose -f deploy/docker-compose.yml up -d

# 3. 初始化数据库
cd deploy
./setup_database.sh
cd ..

# 4. 启动后端
cd backend
cp .env.example .env
# 编辑 .env 配置
cargo run

# 5. 启动前端
cd frontend_svelte
pnpm install
pnpm dev

# 6. 启动鸿蒙客户端
# 使用 DevEco Studio 打开 harmonyapp 目录
```

### 单机分层运行

同一台机器上建议分成三层，避免互相抢端口或污染数据：

- 开发层：本机进程运行 `cargo run`、`pnpm dev`，推荐端口分别为 `5496`、`5173`
- 测试层：使用 `hotspot_test` 数据库和 `6380` 端口的 `redis-test`
- 运行预演层：使用 Docker `runtime` profile 启动容器化前后端，默认端口 `8080`、`3000`

单机预演部署/运行时，使用：

```bash
docker compose -f deploy/docker-compose.yml --profile runtime up -d backend frontend
```

如果本机已有其他服务占用这些端口，可在启动前覆盖 host 端口：

```bash
HOTSPOT_BACKEND_HOST_PORT=18080 \
HOTSPOT_FRONTEND_HOST_PORT=13000 \
HOTSPOT_MINIO_API_HOST_PORT=19000 \
HOTSPOT_MINIO_CONSOLE_HOST_PORT=19001 \
docker compose -f deploy/docker-compose.yml --profile runtime up -d backend frontend minio
```

## 项目结构

```
hotspot/
├── AGENTS.md              # AI Agent 操作指南
├── backend/               # Rust 后端服务
│   ├── src/              # 源代码
│   ├── migrations/       # 数据库迁移
│   ├── Dockerfile
│   ├── Cargo.toml
│   ├── .env.example
│   └── README.md
├── frontend_svelte/       # SvelteKit Web 前端
│   ├── src/
│   ├── Dockerfile
│   ├── package.json
│   └── README.md
├── harmonyapp/            # HarmonyOS 鸿蒙客户端
│   ├── entry/            # 应用入口模块
│   ├── AppScope/         # 应用配置
│   └── README.md
├── spec/                 # API 规范文档
│   ├── api_spec_*.md    # 各模块 API 定义
│   └── AGENTS.md        # Spec 目录 AI 指南
└── deploy/               # 部署运维
    ├── docker-compose.yml
    ├── deployment.md
    ├── nginx/
    └── *.sh
```

## 文档索引

- [后端开发指南](backend/README.md)
- [前端开发指南](frontend_svelte/README.md)
- [鸿蒙客户端](harmonyapp/AGENTS.md)
- [API 规范](spec/api_spec_index.md)
- [部署文档](deploy/deployment.md)

## AI Agent

所有项目代码由 AI Agent 生成。AI Agent 操作规范见 [AGENTS.md](AGENTS.md)。

## 许可证

MIT
