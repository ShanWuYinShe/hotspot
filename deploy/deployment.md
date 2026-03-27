# HotSpot verify / preview / release 设计说明

## 目标

这套体系服务三个主目标，并在 verify 内补一个 inspect 子层：

1. **verify**：支持多 agent + 多 worktree 并行开发时的默认本地验证
2. **verify-inspect**：开发内环的隔离调试环境层，为 Claude 提供页面访问入口与干净运行环境，不把具体验证页面逻辑塞进 deploy
3. **preview**：提供一个唯一的本地预演环境，验证待交付结果的完整入口行为
4. **release**：产出正式部署输入物，不直接运行环境

## 为什么这样拆

过去 `deploy/` 同时承载测试、本地 runtime 预演和部分看起来像正式部署的语义，导致：
- README 与 `deployment.md` 重复
- 脚本职责重叠
- runtime 也被设计成多 env，多余且复杂
- preview image 与正式部署 image 的语义混淆

现在改成：
- **verify 多实例**
- **verify-inspect 复用 verify 的多实例隔离**
- **preview 单实例**
- **release 只产物**

## 一、verify

### 语义
- 面向开发内环
- 默认验证当前 worktree
- 包含未提交修改
- 支持多 worktree 并行
- 主目录应始终保持 `main` + clean；如果要验证带修改的结果，应在承载改动的 worktree 中运行

### 隔离模型
按 `env-name` 隔离：
- 当前工作目录 / worktree
- compose project
- lock
- 测试依赖容器

宿主侧只保留当前 checkout 下 `tmp/deploy/` 里的日志、构建产物、release manifest 与 deploy-managed worktree；Go / pnpm 等中间缓存默认留在容器临时目录内，不再写回宿主 worktree，从根上避免 worktree 清理时被缓存权限卡住。

### 入口

```bash
./deploy/scripts/verify.sh --env-name feature-a
```

### 技术实现
- `compose/verify.yml`
- 固定 runner image：
  - `dockerfiles/backend-go.verify.Dockerfile`
  - `dockerfiles/frontend.verify.Dockerfile`
- 当前工作树通过只读挂载注入容器，并复制到容器临时目录后执行测试，避免把 pnpm store、node_modules、Go module cache 或其他中间产物写回仓库根
- verify runner 以内置的容器临时缓存运行 backend/frontend 校验，并在 `go test ./...` 前先执行 backend-go schema bootstrap
- 每次 verify 都会额外把完整 runner 输出、`docker compose ps` 和完整 compose logs 归档到 `tmp/deploy/logs/verify/<env-name>/<run-id>/`
- `verify-inspect` 复用同一日志根，用于归档环境启动、入口准备与 compose 运行日志；具体页面检查证据不由 deploy 产出

### verify-inspect
- 语义上属于 verify 内的 inspect/debug 子层，而不是新的候选环境。
- 复用 verify 的 `env-name`、lock、日志归档与 worktree 输入语义。
- 目标是让 Claude 在受控 Docker 环境中获得一个可访问的前端/后端隔离入口，而不是依赖宿主机直跑命令或把 preview 当高频调试沙箱。
- 页面可打开、console/UI/交互核查由 Claude Code 在该入口上主动完成；这些检查不应固化进 deploy 测试用例。
- `verify-inspect` 成功后默认保留环境，供 Claude 继续访问；后续可由下一次 `verify` / `verify-inspect` 重建或显式清理回收。

## 二、preview

### 语义
- 面向本地集成分支或候选结果
- 只有一个实例
- 验证 HTTPS + nginx + frontend + backend-go + websocket + MinIO
- 不再为多个 worktree 同时起多套 preview
- 不承担开发内环高频调试职责；页面/console/UI/交互的自动确认默认由 verify-inspect 负责

### 入口

```bash
./deploy/scripts/preview-build.sh
./deploy/scripts/preview-up.sh
./deploy/scripts/preview-inspect.sh
```

### 输入
- 默认：当前工作目录 / worktree
- 可选：通过 `preview-build.sh --ref <git-ref>` 使用确定性 git 版本

### 运行拓扑
- nginx 为唯一宿主入口
- `/` -> frontend
- `/api/v1/` -> backend-go
- `/ws/v1` -> backend-go
- `/s3/` -> MinIO

### compose
- `compose/preview.yml`
- 固定 compose project：`hotspot-preview`
- 默认镜像版本与基础镜像默认值统一由 `deploy/scripts/lib/images.sh` 导出
- 固定 preview image：
  - `hotspot/backend-go:preview`
  - `hotspot/frontend-svelte:preview`

### 数据语义
保留：
- 默认保留 postgres / redis / minio 数据
- 支持 `--clean` / `--clean-all`
- backend-go 在启动时会自动执行 schema bootstrap 并校验关键表；对空库 preview，先清 postgres volume 再启动也应能自举成功
- preview 构建、启动与 inspect 结果都会归档到 `tmp/deploy/logs/preview/current/`，其中既保留构建期完整 stdout/stderr，也保留 runtime `compose ps` / `compose logs` 快照与 HTTP/S3 检查结果
- preview 仍保留 MinIO 服务与 `/s3/` 入口检查，但不再额外引入 `minio/mc` 初始化容器；对象存储桶与策略不在 preview 基础拓扑中预创建

## 三、release

### 语义
- 不运行环境
- 只生成正式部署输入物
- 必须基于确定性 git 版本

### 入口

```bash
./deploy/scripts/release-build.sh --ref HEAD
./deploy/scripts/release-build.sh --ref <sha-or-tag> --label release-xxx
```

### 输出
- backend image
- frontend image
- manifest.json
- `tmp/deploy/logs/release/<label>/` 下的完整构建日志

manifest 至少包含：
- ref
- resolved SHA
- release label
- built_at
- source_root
- backend/frontend runner image
- base image 结果
- backend image
- frontend image
- log_root

### 为什么 release 不运行
- preview 负责行为验证
- release 负责交付确定性
- 这样能把“当前工作树验证”和“正式部署输入”彻底拆开

## 四、Cloudflare / HTTPS / 真实部署边界

当前本地 preview 环境通过 nginx + HTTPS 模拟最终入口，已经足够验证：
- Secure cookie
- 同域 API / ws / s3 入口
- nginx 代理行为

但它不等于完整正式部署。当前仓库没有闭环实现：
- 真实证书签发/续期
- Cloudflare API 自动化
- 源站 IP 信任链
- 远程发布编排

因此当前 `release-build.sh` 只负责**产出正式部署输入物**，不负责真正远端发布。

## 五、依赖下载镜像源与镜像引用覆盖

### 依赖下载镜像源
- 这一层解决的是墙内环境下容器内部下载 Go 模块、npm 包、Alpine 包时的访问问题。
- 当前默认值：
  - Alpine：`https://mirrors.ustc.edu.cn/alpine`
  - Go：
    - `GOPROXY=https://goproxy.cn,direct`
    - `GOSUMDB=sum.golang.google.cn`
  - npm / pnpm：`https://registry.npmmirror.com`
- 当前 `deploy/` 范围内没有 Rust Dockerfile，因此这轮没有补 cargo/rustup 镜像源；若后续引入 Rust Docker 构建链，再单独补齐。

### 容器镜像引用覆盖
- 这一层是“容器镜像引用/版本”的覆盖能力，不等于依赖下载镜像源。
- `compose/preview.yml` 与 `compose/verify.yml` 中的服务镜像默认固定为明确版本，同时支持通过环境变量覆盖。
- 当前默认版本：
  - Postgres：`postgres:18.3-alpine3.23`
  - Redis：`redis:8.6.2-alpine3.23`
  - MinIO：`minio/minio:RELEASE.2025-09-07T16-13-09Z`
  - nginx：`nginx:1.28.3-alpine3.23`
- 当前支持的覆盖项包括：
  - `HOTSPOT_PREVIEW_POSTGRES_IMAGE`
  - `HOTSPOT_PREVIEW_REDIS_IMAGE`
  - `HOTSPOT_PREVIEW_MINIO_IMAGE`
  - `HOTSPOT_PREVIEW_NGINX_IMAGE`
  - `HOTSPOT_VERIFY_POSTGRES_IMAGE`
  - `HOTSPOT_VERIFY_REDIS_IMAGE`
  - `HOTSPOT_AMAP_JSCODE`
- Dockerfile 基础镜像默认也固定为明确版本，并支持 build arg 覆盖：
  - `HOTSPOT_GOLANG_BASE_IMAGE`（默认 `golang:1.26.1-alpine3.23`）
  - `HOTSPOT_NODE_BASE_IMAGE`（默认 `node:24.14.0-alpine3.23`）
  - `HOTSPOT_ALPINE_BASE_IMAGE`（默认 `alpine:3.23.3`）
  - `HOTSPOT_ALPINE_REPO`

## 六、目录约定

```text
deploy/
  README.md
  deployment.md
  compose/
    verify.yml
    preview.yml
  dockerfiles/
    backend-go.verify.Dockerfile
    frontend.verify.Dockerfile
    backend-go.app.Dockerfile
    frontend.app.Dockerfile
  scripts/
    common.sh
    verify.sh
    preview-build.sh
    preview-up.sh
    preview-down.sh
    preview-inspect.sh
    release-build.sh
  nginx/
    hotspot.preview.nginx.conf
```

## 七、必须保留的行为
- 本地优先开发与本地集成分支流程不变
- verify 默认继续走 Docker
- verify 默认继续验证当前工作目录 / worktree
- 会影响运行环境或在多 worktree / 多 agent 下可能冲突的测试、build、preview、release、环境准备动作默认都通过 `deploy/` 下的隔离流程执行
- 纯静态检查如 `pnpm check`、`go vet` 若确认不会污染宿主环境，也可以直接在宿主环境执行
- 测试与构建过程中产生的缓存、pnpm store、临时产物不得写回仓库根
- preview 保留 HTTPS + nginx + MinIO + inspect 链
- AMap 代理能力仍保留，但 `jscode` 必须通过 `HOTSPOT_AMAP_JSCODE` 注入，不能再硬编码在 nginx 模板中
- 数据保留 / 清理语义保留
- 多 worktree 并行隔离只属于 verify

## 八、验证要求

### verify
- 不同 worktree 使用不同 `env-name` 可以并行运行
- 工作目录 / compose project / lock 不串；容器内中间缓存不写回宿主 worktree
- backend-go 与 frontend 验证都能通过

### preview
- `preview-build.sh` 成功产出 preview image
- `preview-up.sh` 启动成功
- `preview-inspect.sh` 能通过：
  - `GET /`
  - `POST /api/v1/system/health`
  - `GET /s3/`

### release
- `release-build.sh --ref <sha>` 能稳定产出 release image 与 manifest
- release 结果不依赖当前未提交工作树
- 产物、manifest 与临时 worktree 都应由当前宿主用户创建，完成后可直接清理，不需要 sudo
