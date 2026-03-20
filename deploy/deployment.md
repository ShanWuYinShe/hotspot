# HotSpot 部署运维文档

## 0. 当前部署目标

当前 `deploy/` 目录主要支持两类场景：

1. **Docker 一键测试**
   - 启动测试用 PostgreSQL / Redis
   - 在 Docker 里执行 `backend-go` 测试
   - 在 Docker 里执行前端校验
   - 每次都从干净环境开始

2. **Docker 运行预演**
   - 通过 nginx 作为唯一宿主机入口
   - 同时提供 frontend、backend-go、WebSocket、MinIO 文件访问
   - 使用项目 Dockerfile 构建 backend-go / frontend 镜像
   - app 层默认可重建，但 postgres / redis / minio 数据默认保留
   - 支持手动选择清空哪些数据服务
   - 支持多分支 / 多工作树并行使用独立环境名与独立端口
   - 形态尽量接近真实部署

## 1. 单机建议拓扑

| 层级 | 进程形态 | 默认入口 | 说明 |
|---|---|---|---|
| 开发层 | 本机进程 | frontend `5173` / backend-go 自定义 | 改代码、热更新、调试 |
| 测试层 | Docker | 无宿主机公开入口 | 一键执行 `backend-go` tests + frontend 校验 |
| 运行预演层 | Docker + nginx | `https://localhost:8443` | 统一通过 nginx 使用整套服务 |

## 2. Docker 测试流

入口脚本：

```bash
./deploy/scripts/test-in-docker.sh
```

实际使用的 compose 文件：

- `deploy/docker-compose.test.yml`

测试流包含：

- `postgres-test`
- `redis-test`
- `backend-go-test`
- `frontend-validate`

执行顺序：

1. 启动 `postgres-test` / `redis-test`
2. `backend-go-test` 运行 `go test ./...`
3. `frontend-validate` 运行 `pnpm check && pnpm build --mode test`
4. 自动清理测试容器

## 3. Docker 运行预演流

这条流现在支持：

- 多分支 / 多工作树并行运行独立环境
- app 容器重建而不清空数据
- 只清理某个数据服务
- 手动指定 nginx 对外端口


入口脚本：

```bash
./deploy/scripts/up-runtime.sh
```

并行环境示例：

```bash
./deploy/scripts/up-runtime.sh --env-name feature-a --http-port 18088 --https-port 18443
./deploy/scripts/up-runtime.sh --env-name feature-b --http-port 28088 --https-port 28443
```

实际使用的 compose 文件：

- `deploy/docker-compose.runtime.yml`

### 3.1 服务拓扑

```text
Browser
  -> nginx (HTTPS)
      -> frontend_svelte
      -> backend-go (/api/v1/*)
      -> backend-go (/ws/v1)
      -> minio (/s3/*)
```

宿主机默认只开放 nginx：

- HTTP 重定向入口：`http://localhost:8088`
- HTTPS 主入口：`https://localhost:8443`

但在多环境并行时，建议显式指定不同端口，避免冲突。

### 3.2 统一入口路由

由 `deploy/nginx/hotspot.runtime.nginx.conf` 提供：

- `/` -> frontend
- `/api/v1/` -> backend-go
- `/ws/v1` -> backend-go
- `/s3/` -> minio

这意味着你不需要分别连接 frontend、backend-go、MinIO 的宿主机端口。

## 4. HTTPS 与鉴权

`backend-go/internal/platform/auth/cookies.go` 当前会写入 `Secure` Cookie。

因此运行预演时建议始终走 **HTTPS nginx 入口**，否则浏览器登录态与刷新流程可能不工作。

推荐使用 `mkcert` 生成本地证书：

```bash
mkdir -p tmp/certs
mkcert -cert-file tmp/certs/localhost.crt -key-file tmp/certs/localhost.key localhost 127.0.0.1 ::1
```

脚本默认读取：

- `tmp/certs/localhost.crt`
- `tmp/certs/localhost.key`

也可以通过环境变量覆盖：

- `HOTSPOT_TLS_CERT_PATH`
- `HOTSPOT_TLS_KEY_PATH`

## 5. backend-go 运行配置

`backend-go` 当前主要依赖以下环境变量：

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

在运行预演流中，backend-go 默认监听容器内 `8080`。

## 6. 验证方式

### 6.1 首页

```bash
curl -k https://localhost:8443/
```

### 6.2 系统健康检查

backend-go 当前健康检查是：

- `POST /api/v1/system/health`
- body 为 `{}`

示例：

```bash
curl -k -X POST https://localhost:8443/api/v1/system/health \
  -H 'Content-Type: application/json' \
  -d '{}'
```

### 6.3 WebSocket

- `wss://localhost:8443/ws/v1`

### 6.4 MinIO 文件访问

- `https://localhost:8443/s3/...`

### 3.3 查看 / 检查 / 日志 / 停止 / 删除环境

```bash
# 查看环境状态
./deploy/scripts/status-runtime.sh --env-name feature-a

# 做入口烟雾检查
./deploy/scripts/check-runtime.sh --env-name feature-a --https-port 18443

# 查看日志
./deploy/scripts/logs-runtime.sh --env-name feature-a --services backend,nginx --tail 300

# 停掉某套环境
./deploy/scripts/down-runtime.sh --env-name feature-a

# 删除该环境的容器，但保留数据
./deploy/scripts/down-runtime.sh --env-name feature-a --remove

# 删除该环境的所有容器和数据
./deploy/scripts/down-runtime.sh --env-name feature-a --clean-all
```

## 7. AI / 自动化巡检建议

如果是 AI 改完代码后要快速检查某一套环境，优先直接调用：

```bash
./deploy/scripts/inspect-runtime.sh --env-name feature-a --https-port 18443
```

它会自动：

1. 查看环境状态
2. 做入口检查
3. 仅在失败时自动打印关键日志

如果你希望无论成功失败都输出日志：

```bash
./deploy/scripts/inspect-runtime.sh --env-name feature-a --https-port 18443 --logs
```

## 8. 运行控制建议

常见用法：

```bash
# 默认：保留 postgres / redis / minio 数据，只重建需要的 app 容器
./deploy/scripts/up-runtime.sh --env-name feature-a --http-port 18088 --https-port 18443

# 代码变了，只想强制重建 backend / frontend / nginx
./deploy/scripts/up-runtime.sh --env-name feature-a --recreate backend,frontend,nginx --http-port 18088 --https-port 18443

# 只清 postgres 数据
./deploy/scripts/up-runtime.sh --env-name feature-a --clean postgres --http-port 18088 --https-port 18443

# 全量清空这套环境
./deploy/scripts/up-runtime.sh --env-name feature-a --clean-all --http-port 18088 --https-port 18443
```

## 9. 设计说明

这套运行流的重点是：

1. 统一通过 nginx 提供 frontend、backend-go、WebSocket 与 MinIO 文件访问
2. 使用 Dockerfile 构建应用容器，尽量保证运行内容一致
3. 支持保留数据、局部清理数据、强制重建 app 层容器
4. 支持通过 `--env-name` + 端口参数并行启动多套环境
5. 适合接入 Cloudflare Full 模式，并在源站侧使用本地证书或 Cloudflare Origin Certificate
