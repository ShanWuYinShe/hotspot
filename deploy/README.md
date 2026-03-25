# HotSpot deploy workflow

`deploy/` 现在只保留三层语义：

- **verify**：默认本地验证层，服务多 agent + 多 worktree 并行开发
- **preview**：单实例本地预演层，验证最终候选结果的 HTTPS / nginx / websocket / MinIO / cookie 行为
- **release**：正式部署输入层，只产 image / artifact / manifest，不运行环境

## 快速入口

### 短版工作流
- `verify`：用于并行开发验证，支持多 agent + 多 worktree。默认验证当前工作目录 / worktree，并在 Docker 隔离环境里完成 backend-go/frontend 的测试与构建检查。示例：`./deploy/scripts/verify.sh --env-name feature-a`
- `preview`：用于最终候选结果预演，只有一个实例。先 `./deploy/scripts/preview-build.sh`，再 `./deploy/scripts/preview-up.sh`，最后 `./deploy/scripts/preview-inspect.sh`
- `release`：用于产出正式部署输入物，不运行环境。示例：`./deploy/scripts/release-build.sh --ref HEAD --label release-2026-03-21`

### 常用命令

#### verify
```bash
./deploy/scripts/verify.sh --env-name feature-a
```

#### preview
```bash
./deploy/scripts/preview-build.sh
./deploy/scripts/preview-up.sh
./deploy/scripts/preview-inspect.sh
```

停止或清理 preview：

```bash
./deploy/scripts/preview-down.sh
./deploy/scripts/preview-down.sh --remove
./deploy/scripts/preview-down.sh --clean-all
```

#### release
```bash
./deploy/scripts/release-build.sh --ref HEAD
./deploy/scripts/release-build.sh --ref <sha-or-tag> --label release-2026-03-21
```

## 当前原则
- 只有 `verify` 支持多实例并行。
- `preview` 只有一个实例，不再按 worktree 并行。
- `release` 不运行环境，只产物。
- `verify` 默认验证当前工作目录 / worktree，包含未提交修改。
- 所有测试、check、build 验证默认都通过 `deploy/` 下的 Docker 流程完成；不要把宿主环境直接运行 `go test`、`pnpm check`、`pnpm build` 当成最终验证。
- 测试和构建流程产生的中间缓存默认保留在容器临时目录中，不写回宿主 worktree；宿主侧只保留日志、release/preview 产物等需要交付或复盘的输出。
- `preview` / `release` 面向候选结果与确定性版本，不再承担多 env 开发隔离语义。
- `verify` / `preview` / `release` 现在都会把关键日志归档到 `tmp/deploy/logs/`，便于人工复盘与 AI 分析完整错误链。

## 日志归档
- `verify`：`tmp/deploy/logs/verify/<env-name>/<run-id>/`
- `preview`：`tmp/deploy/logs/preview/current/`
- `release`：`tmp/deploy/logs/release/<label>/`
- `verify` / `preview-build` / `release-build` 会保留完整 stdout/stderr。
- `preview-up` 会保留启动快照、`compose ps` 与 startup logs。
- `preview-inspect` 会保留检查输出、`compose ps`、完整 compose logs，以及 `GET /`、`POST /api/v1/system/health`、`GET /s3/` 的请求结果。
- 控制台仍保留简洁摘要；排障时优先把 `tmp/deploy/logs/...` 下的完整日志交给 AI 分析，而不是只依赖终端头尾输出。

## 依赖下载镜像源与镜像引用覆盖
- 这两类配置不是一回事：
  - **依赖下载镜像源**：解决墙内环境下 Go / npm / Alpine 包下载慢或失败的问题。
  - **容器镜像引用覆盖**：用于在特殊环境下替换服务镜像或基础镜像引用，不等于包管理镜像源。
- 依赖下载镜像源当前默认值：
  - Alpine：`https://mirrors.ustc.edu.cn/alpine`
  - Go：`GOPROXY=https://goproxy.cn,direct`
  - npm / pnpm：`https://registry.npmmirror.com`
- 外部服务镜像默认固定为仓库内的明确版本，同时支持环境变量覆盖，例如：
  - `HOTSPOT_PREVIEW_MINIO_IMAGE`（默认 `minio/minio:RELEASE.2025-09-07T16-13-09Z`）
  - `HOTSPOT_PREVIEW_MINIO_MC_IMAGE`（默认 `minio/mc:RELEASE.2025-08-13T08-35-41Z`）
  - `HOTSPOT_PREVIEW_NGINX_IMAGE`（默认 `nginx:1.28.3-alpine3.23`）
  - `HOTSPOT_PREVIEW_POSTGRES_IMAGE`（默认 `postgres:17.9-alpine3.23`）
  - `HOTSPOT_PREVIEW_REDIS_IMAGE`（默认 `redis:7.4.8-alpine3.21`）
  - `HOTSPOT_VERIFY_POSTGRES_IMAGE`（默认 `postgres:17.9-alpine3.23`）
  - `HOTSPOT_VERIFY_REDIS_IMAGE`（默认 `redis:7.4.8-alpine3.21`）
- Dockerfile 基础镜像默认固定为明确版本，同时支持 build arg 覆盖，例如：
  - `HOTSPOT_GOLANG_BASE_IMAGE`（默认 `golang:1.26.1-alpine3.23`）
  - `HOTSPOT_NODE_BASE_IMAGE`（默认 `node:22.22.1-alpine3.23`）
  - `HOTSPOT_ALPINE_BASE_IMAGE`（默认 `alpine:3.23.3`）
  - `HOTSPOT_ALPINE_REPO`
- 当前 `deploy/` 范围内没有 Rust Docker 构建链，因此这轮没有新增 cargo/rustup 镜像源配置；若后续引入 Rust Dockerfile，再单独补齐。

## 目录结构
- `compose/verify.yml`：verify 编排
- `compose/preview.yml`：preview 编排
- `dockerfiles/`：runner / app Dockerfiles
- `scripts/`：verify / preview / release 入口
- `nginx/hotspot.preview.nginx.conf`：preview 统一入口配置

更多设计说明见 [deployment.md](deployment.md)
