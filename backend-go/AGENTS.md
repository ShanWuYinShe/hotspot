# Backend-Go - Agent Guide

> Go / Chi / PostgreSQL / S2 后端服务的 AI Agent 操作指南。

## 使用范围

本文件只补充 Go 后端特有的硬约束（分层、SQL/S2 规则等），其他实现细节和功能结构请 Agent 通过代码与 `spec/` 自行探索。

## 架构分层（强制）

```
Handler → Service → Repository
```

**严格禁止跨层调用。** 每一层的职责不可混淆：

| 层 | 职责 | 禁止 |
|----|------|------|
| **Handler** | 解析请求 → 调 Service → 返回 `ApiResponse` | 写 SQL、写业务逻辑、直接访问数据库 |
| **Service** | 业务规则、权限校验、事务、缓存、S2 计算、事件发布 | 写 SQL、直接构造 HTTP 响应 |
| **Repository** | SQL 执行与结果映射 | 业务判断、权限校验、发事件 |

> Handler / Service / Repository 的具体命名和目录结构，以现有代码为准，本文件不做约束。

## 数据访问与 SQL 规则

- 使用 `pgx` / `pgxpool` 执行 SQL，**禁止在 Handler 层直接使用数据库连接**。
- 所有 SQL 都在 Repository 层集中管理，禁止散落在 Service/Handler 中。
- 禁止字符串拼接 SQL 注入风险：
  - 使用占位符与绑定参数（`$1, $2, ...`）；
  - 对动态片段（排序字段等）必须使用白名单映射，而不是直接拼接客户端输入。
- 分页：
  - 只使用游标/keyset 分页，**禁止 OFFSET**；
  - cursor 编解码仅在 Repository 层完成，对上层完全不透明；
  - 行为必须与 `spec/api_spec_base.md` 中的分页规范一致。
- 事务：
  - 由 Service 层负责启动/提交/回滚；
  - Repository 接受 `pgx.Tx` 或 `pgxpool.Pool`，但不自行开启事务。
- PostGIS：
  - **backend-go 不启用 PostGIS 扩展，不使用 geometry/geography 类型，不调用 ST_ 系列函数**；
  - 所有地理计算由 S2 + 简单经纬度函数在 Service 层完成。

## 地理查询（S2）

- 所有带位置的实体必须统一存储以下字段：
  - `lat`：WGS-84 纬度（`DOUBLE PRECISION`）
  - `lng`：WGS-84 经度（`DOUBLE PRECISION`）
  - `s2_cell_id`：基础 S2 单元格 ID（`BIGINT`），通常为固定 `BASE_LEVEL`（如 14）的 `CellID`
  - `s2_level`：存储该 `s2_cell_id` 的 level（`SMALLINT`），如无特殊需要可固定为 14
- Service 层必须通过 `internal/geo/s2util` 封装的函数完成：
  - 经纬度 → `s2.CellID`（指定 level）；
  - 视口 bounds / center+radius → 覆盖的 S2 cell 列表；
  - 根据视口大小自动选择 S2 level。
- 附近搜索 / 范围查询：
  - **必须先用 S2 粗筛**：按 `s2_cell_id IN (...)` 或预聚合统计过滤候选；
  - 如需精确排序，才对候选集合使用 Haversine 或 S2 距离函数计算精确距离；
  - 禁止在前端/客户端自行计算距离或角度。

## 错误处理

- 在 Go 侧定义统一的 `ApiError` 类型或错误接口，并提供从领域错误到 `ApiResponse<T>` 的映射函数：
  - 错误码必须来自 `spec/api_spec_errors.md`，禁止自行编号；
  - 同一错误码可以由多个内部错误映射得到，以提高可读性。
- 所有 HTTP 接口统一返回 `HTTP 200 + ApiResponse<T>` 包装：
  - 业务成功：`code == 0`；
  - 业务失败：`code != 0`，`message` 为用户可见文案，不泄露内部技术细节。
- 日志：
  - 使用 zap，所有日志必须包含 request-id（从中间件注入）；
  - 严禁记录密码、Token、敏感隐私数据。

## 认证与中间件

- JWT 内容与 Rust 后端保持一致（仅包含 `user_id` 和 `session_id` 等必要字段）。
- 中间件链建议：

| 认证级别 | 中间件链 |
|----------|----------|
| Token 可选 | `optional_auth` → `rate_limit` → `resource_guard` → `idempotency` |
| Token 必须 | `auth` → `rate_limit` → `resource_guard` → `idempotency` |
| 管理员 | `auth` → `admin_auth` → `rate_limit` → `resource_guard` → `idempotency` |

- 认证/授权逻辑必须在 Service 层完成，Handler 不做权限判断。

## 额外禁止

除根目录 `AGENTS.md` 中的全局禁止项外，Go 后端额外禁止：
- 绕过 Handler → Service → Repository 分层直接跨层访问。
- 在 Handler 或 Service 中直接写 SQL，或在这些层使用数据库连接。
- 使用 OFFSET 分页，或在 Go 侧引入 PostGIS / geometry 类型 / ST_ 系列函数。

## 安全规则

- 文件删除、批量变更、Git 操作规范、数据库操作安全等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

> **⚠️ `tmp/` 路径规则**：所有临时文件必须存放在项目根目录的 `tmp/` 下。详见根目录 `AGENTS.md` 的「Agent 行为准则」章节。

## 验证命令

```bash
# 编译 / 构建
cd backend-go
go build ./cmd/server

# 测试
cd backend-go
go test ./...

# 静态检查
cd backend-go
go vet ./...
```

> 若未来引入额外工具（如 golangci-lint），应在此处补充，但仍需遵守根目录 `AGENTS.md` 中的权限与询问规则。
