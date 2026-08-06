# HotSpot Backend (Go)

HotSpot 的 Go 主后端：模块化单体 + S2 地理索引，提供全部 HTTP API 与 WebSocket 实时能力。API 契约见仓库根 `spec/`（只读权威）。

## 功能

- 认证（注册/登录/刷新/登出/会话管理）、用户资料与社交（关注/好友/粉丝/拉黑）
- 群组（创建/成员/邀请/申请/公告/禁言/封禁/转让）、私聊与群聊（消息/已读/撤回）
- 生活瞬间（发布/点赞/评论/收藏）、Feed（附近/关注/热门）、搜索（用户/群组/联想）
- 地图（S2 网格热力）、分享、媒体（S3 预签名上传）、举报与处理
- 通知中心（站内 + WS 推送）、系统公告与配置、管理后台（RBAC/审计/封禁/举报处理）

## 运行

后端必须与前端一起通过 Docker Compose 运行（与生产一致，经 nginx 网关）：

```bash
# From the repository root:
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# 网关入口: http://localhost:8888（/api 反代到本后端）
```

dev 模式跑 `Dockerfile.dev`（air 热加载），改 Go 代码自动重编译，无需重启容器。

## 构建 & 验证

```bash
go build ./...        # 编译
go vet ./...          # 静态检查
gofmt -l .            # 格式检查
go test -short ./...  # 单元测试（-short 跳过需 DB 的集成测试）
```

集成测试（E2E，需要 Docker）：

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml --profile test up -d backend-test
BACKEND_URL=http://localhost:8082 go test -count=1 ./test/integration/
```

覆盖率门槛：`internal/modules` 下每包语句覆盖率 ≥70%（CI 强制，见 `.github/workflows/ci.yml`）。

## 目录结构

```
backend-go/
├── cmd/                    # 入口（server / migrate）
├── migrations/             # SQL 迁移（embed 进二进制，启动自动执行）
├── test/integration/       # 集成测试（E2E，需要 Docker）
├── internal/
│   ├── bootstrap/          # 装配：模块注册、路由、WS、worker、迁移
│   ├── modules/            # 18 个业务模块（auth/users/groups/chat/moments/…）
│   │   └── <模块>/         # 统一骨架：types/handler/service/repo(+memory/postgres/queries)/module
│   └── platform/           # 基础设施：auth/persistence/httpx/realtime/observability/…
└── spec 契约见仓库根 spec/（只读）
```

## 技术栈

Go（chi router）、PostgreSQL + pgx/v5、Redis（限流/幂等/WS fanout，可选）、google/s2（地理索引）、gorilla/websocket、zap（结构化日志）。
