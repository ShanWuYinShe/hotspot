# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 这个后端做什么
这是 HotSpot 的 Go / chi 辅助后端。当前重点是认证、用户和地图相关接口；入口在 `cmd/server/main.go`，`internal/app/app.go` 负责把数据库、Redis、token manager、service 和 handler 组装起来。地理相关逻辑走 S2，不使用 PostGIS。

## 代码组织
- `cmd/server/main.go`：读取配置、初始化 logger、中间件和 HTTP server。
- `internal/app/`：依赖注入与路由注册。
- `internal/handler/`：HTTP 边界，只做参数提取、调用 service、返回响应。
- `internal/service/`：业务规则、权限、事务、会话管理。
- `internal/repository/`：数据库访问与结果映射。
- `internal/auth/`：密码、JWT、上下文相关逻辑。
- `internal/middlewares/`：request id、恢复、日志、超时、认证中间件。
- `internal/geo/s2util/`：S2 cell、视口和范围相关计算。
- `internal/api/response.go`、`internal/errors/api_error.go`、`internal/types/api_types.go`：响应封装、错误和通用类型。
- `migrations/`：数据库迁移。

## 关键约束
- 严格遵守 `Handler -> Service -> Repository` 分层。
- SQL 只在 repository 层写，使用 `pgx` / `pgxpool` 和参数绑定，不要在 handler/service 里直接查库。
- 分页只用 cursor / keyset，不用 OFFSET。
- 地理计算只用 S2 + 简单经纬度函数，不启用 PostGIS。
- 认证/会话/黑名单逻辑放在 service 层处理，日志带 request id。
- 不要自行执行 `deploy/` 目录下的脚本，也不要自行跑数据库迁移/重置命令。
- 修改 API 前先对照 `spec/`。

## 运行配置
- 运行时读取环境变量：`HTTP_ADDR`、`POSTGRES_URL`、`REDIS_ADDR`、`JWT_SECRET`。
- `HTTP_ADDR` 默认是 `:8080`。

## 常用命令
- `go build ./cmd/server`
- `go test ./...`
- `go vet ./...`
- 单测：`go test ./... -run 'TestName$'`
