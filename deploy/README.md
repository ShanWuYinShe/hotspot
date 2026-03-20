# HotSpot Docker 部署入口

当前 `deploy/` 目录收敛为两条主流程：

- **一键测试**：Docker 内启动测试依赖并直接执行 `backend-go` 测试与前端校验
- **一键运行预演**：只暴露 nginx 端口，通过 nginx 统一访问 frontend + backend-go + MinIO

两条流程都使用 Dockerfile 在容器内从头构建并启动，尽量减少环境干扰，并让 runtime 与将来发布时的运行内容保持一致。

## 关键文件

| 文件 | 用途 |
|---|---|
| `docker-compose.test.yml` | Docker 测试入口：`postgres-test`、`redis-test`、`backend-go-test`、`frontend-validate` |
| `docker-compose.runtime.yml` | 运行预演入口：使用 Dockerfile 构建 backend-go / frontend 镜像并统一通过 nginx 提供服务 |
| `nginx/hotspot.runtime.nginx.conf` | 统一入口 nginx 配置，代理 `/`、`/api/v1/`、`/ws/v1`、`/s3/` |
| `scripts/test-in-docker.sh` | 一键测试脚本（每次从干净环境开始） |
| `scripts/up-runtime.sh` | 一键运行预演脚本 |
| `scripts/down-runtime.sh` | 停止、移除或清理指定 runtime 环境 |
| `scripts/status-runtime.sh` | 查看指定 runtime 环境的容器与数据卷状态 |
| `scripts/logs-runtime.sh` | 查看指定 runtime 环境的日志 |
| `scripts/check-runtime.sh` | 对指定 runtime 环境做入口与健康检查 |
| `scripts/inspect-runtime.sh` | 一条命令串联状态、检查，并在失败时自动打印日志 |

## 一键测试

```bash
./deploy/scripts/test-in-docker.sh
```

如果不传 `--env-name`，脚本会自动使用当前 git 分支 / worktree 名生成隔离环境名，减少多 agent / 多 worktree 并行时的手动命名负担。

如果你同时有多个分支/工作树在开发，也仍然可以手动指定一个更明确的测试环境名：

```bash
./deploy/scripts/test-in-docker.sh --env-name feature-a
./deploy/scripts/test-in-docker.sh --env-name feature-b
```

这个脚本会：

1. 启动 `postgres-test` 与 `redis-test`
2. 在 Docker 中执行 `backend-go` 的 `go test ./...`
3. 在 Docker 中执行前端校验：`pnpm check && pnpm build --mode test`
4. 返回真实退出码并自动清理测试容器

默认测试范围：

- `backend-go`
- `frontend_svelte` 校验

## 一键运行预演

这条流程会：

- 使用 Dockerfile 构建 backend-go / frontend 镜像
- 在容器里从头启动整套服务
- 启动前清空旧容器和卷，保证每次都是干净环境
- 只暴露 nginx 入口

### 先准备本地 HTTPS 证书

当前 Web 鉴权 Cookie 是 `Secure`，所以要通过 **HTTPS 的 nginx 入口** 使用整套服务。

推荐使用 `mkcert` 生成：

```bash
mkdir -p tmp/certs
mkcert -cert-file tmp/certs/localhost.crt -key-file tmp/certs/localhost.key localhost 127.0.0.1 ::1
```

### 启动

```bash
./deploy/scripts/up-runtime.sh
```

如果不传 `--env-name`，脚本会自动使用当前 git 分支 / worktree 名生成隔离环境名；如果不显式指定端口，脚本会基于环境名派生一组默认端口，并在端口冲突时自动向后寻找可用端口。

多个分支并行开发时，你仍然可以显式指定名字和端口，获得更可控的环境：

```bash
./deploy/scripts/up-runtime.sh --env-name feature-a --http-port 18088 --https-port 18443
./deploy/scripts/up-runtime.sh --env-name feature-b --http-port 28088 --https-port 28443
```

默认入口：

- HTTPS：`https://localhost:8443`
- HTTP 重定向入口：`http://localhost:8088`

### 运行链路

宿主机只需要访问 nginx：

- `/` -> `frontend_svelte`
- `/api/v1/*` -> `backend-go`
- `/ws/v1` -> `backend-go`
- `/s3/*` -> `minio`

也就是说你不需要单独去连 frontend、backend-go、MinIO 的宿主机端口。

### 查看 / 检查 / 日志 / 停止 / 删除环境

```bash
# 查看环境状态
./deploy/scripts/status-runtime.sh --env-name feature-a

# 查看整套日志
./deploy/scripts/logs-runtime.sh --env-name feature-a

# 只看 backend 和 nginx 日志
./deploy/scripts/logs-runtime.sh --env-name feature-a --services backend,nginx

# 对入口做烟雾检查
./deploy/scripts/check-runtime.sh --env-name feature-a --https-port 18443

# 停掉整套环境，但保留容器和数据
./deploy/scripts/down-runtime.sh --env-name feature-a

# 删除整套环境容器，但保留数据
./deploy/scripts/down-runtime.sh --env-name feature-a --remove

# 删除整套环境和全部数据
./deploy/scripts/down-runtime.sh --env-name feature-a --clean-all

# 只删 postgres 数据
./deploy/scripts/down-runtime.sh --env-name feature-a --clean postgres
```

### 给 AI / 自动化巡检的推荐入口

在 AI 改完代码后，推荐默认这样做：

```bash
# 代码改动影响 runtime 行为时，先重建 app 层
./deploy/scripts/up-runtime.sh --env-name feature-a --recreate backend,frontend,nginx --http-port 18088 --https-port 18443

# 然后统一巡检
./deploy/scripts/inspect-runtime.sh --env-name feature-a --https-port 18443
```

如果改动主要是测试逻辑或接口实现，也可以直接先跑：

```bash
./deploy/scripts/test-in-docker.sh --env-name feature-a
```

`inspect-runtime.sh` 会自动：

1. 查看环境状态
2. 做入口与健康检查
3. 仅在失败时自动打印关键日志

如果你想强制把日志也一起打出来：

```bash
./deploy/scripts/inspect-runtime.sh --env-name feature-a --https-port 18443 --logs
```

## 运行控制

### runtime 脚本参数

- `--env-name NAME`：给当前环境命名，用于隔离容器 / network / volume
- `--http-port PORT`：指定 nginx 的 HTTP 入口端口
- `--https-port PORT`：指定 nginx 的 HTTPS 入口端口
- `--services a,b,c`：只启动指定服务
- `--recreate a,b,c`：强制重建指定服务
- `--clean-all`：清空整套 runtime 数据后重启
- `--clean postgres,redis,minio`：只清指定数据服务

示例：

```bash
# 保留数据，只重建 app 层
./deploy/scripts/up-runtime.sh --env-name feature-a --recreate backend,frontend,nginx --http-port 18088 --https-port 18443

# 只清 postgres 数据，再启动同一套环境
./deploy/scripts/up-runtime.sh --env-name feature-a --clean postgres --http-port 18088 --https-port 18443

# 全量清空这套环境
./deploy/scripts/up-runtime.sh --env-name feature-a --clean-all --http-port 18088 --https-port 18443
```

## 可选环境变量

### 运行预演

- `HOTSPOT_ENV_NAME`：默认 `default`
- `HOTSPOT_RUNTIME_HTTP_PORT`：默认 `8088`
- `HOTSPOT_RUNTIME_HTTPS_PORT`：默认 `8443`
- `HOTSPOT_PUBLIC_ORIGIN`：默认 `https://localhost:8443`
- `HOTSPOT_TLS_CERT_PATH`：默认 `tmp/certs/localhost.crt`
- `HOTSPOT_TLS_KEY_PATH`：默认 `tmp/certs/localhost.key`
- `AUTH_TOKEN_SECRET`
- `MINIO_ROOT_USER`
- `MINIO_ROOT_PASSWORD`
- `S3_BUCKET`
- `VITE_AMAP_KEY`

### backend-go 运行时配置

`backend-go` 主要使用这些环境变量：

- `APP_HOST`
- `APP_PORT`
- `POSTGRES_DSN`
- `REDIS_DSN`
- `AUTH_TOKEN_SECRET`
- `APP_SUPPORT_EMAIL`
- `APP_PRIVACY_POLICY_URL`
- `APP_TERMS_OF_SERVICE_URL`
- `APP_OFFICIAL_WEBSITE_URL`
- `APP_DEFAULT_USER_AVATAR_URL`
- `APP_DEFAULT_GROUP_AVATAR_URL`

## 验证建议

启动后可通过 nginx 入口检查：

```bash
curl -k https://localhost:8443/
curl -k -X POST https://localhost:8443/api/v1/system/health -H 'Content-Type: application/json' -d '{}'
```

WebSocket 入口：

- `wss://localhost:8443/ws/v1`

MinIO 文件入口：

- `https://localhost:8443/s3/...`

## 说明

- `deploy/` 目录负责定义容器编排、统一入口、数据生命周期和多环境隔离策略
- runtime 默认保留 `postgres` / `redis` / `minio` 数据；需要时使用 `--clean` 或 `--clean-all`
- 多环境并行运行时，请为每套环境指定不同的 `--env-name`、`--http-port`、`--https-port`
- 如果通过 Cloudflare Full 模式接入，nginx 源站证书可以使用 Cloudflare Origin Certificate

## 其他文档

- [deployment.md](deployment.md)
