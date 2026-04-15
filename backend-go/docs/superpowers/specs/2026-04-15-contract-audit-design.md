# backend-go 接口契约审计设计

## Context

backend-go 当前已经完成模块级 `handler_test.go` 与 `service_test.go` 补全，并通过了全量 `go test -short ./...` 与 `go vet ./...`。在测试覆盖基础已经建立后，下一步高价值工作不是继续机械补测试，而是检查实现是否真正符合 `spec/` 中定义的 API 契约。

本轮审计不做全仓逐端点穷举，而是优先检查最容易造成客户端兼容问题的高风险契约项：错误码、错误消息、必填字段校验、分页 cursor 语义、以及成功/失败响应形状。为控制范围，首批只覆盖高频核心模块：`auth`、`users`、`moments/comments`。

## Goal

产出一份面向实现修复的契约偏差清单，帮助后续在最小范围内修正 backend-go 与 spec 之间的高风险不一致。

本轮目标不是“证明没有问题”，而是尽快发现：
- 会让客户端错误处理失效的偏差
- 会让请求参数校验与文档不一致的偏差
- 会让分页或成功响应语义不稳定的偏差

## Scope

### 审计模块
- `spec/api_spec_auth.md` ↔ `internal/modules/auth/`
- `spec/api_spec_user.md` ↔ `internal/modules/users/`
- `spec/api_spec_moment.md` ↔ `internal/modules/moments/` 与 `internal/modules/comments/`

### 审计对象
对每个端点，检查以下文件层：
- `routes.go` / `module.go`：确认路由存在且路径匹配
- `handler.go`：确认参数解析、错误映射、响应结构
- `types.go`：确认请求/响应字段名与结构
- 必要时补充查看 `service.go`：确认 handler 返回的业务错误是否符合 spec 预期

### 本轮不包含
- 不逐字段审计所有成功响应示例
- 不检查每个 Postgres 仓库实现是否逐列匹配 spec
- 不把 WebSocket 事件作为本轮重点
- 不修改 spec 文档，除非后续确认 spec 明显过时且实现已被多处依赖

## Audit Method

### 1. 路由存在性
从 spec 提取端点路径，核对 `routes.go` / `module.go` 是否注册了对应路径。

### 2. 请求校验一致性
重点检查：
- spec 标记为必填的字段，handler 是否强制校验
- handler 是否错误地要求了 spec 中未要求的字段
- 空字符串、非法枚举、非法分页 cursor 是否返回 spec 约定的错误码

### 3. 错误契约一致性
重点检查：
- not found / unauthorized / forbidden / invalid request / conflict / unavailable 等常见错误
- handler 中的 `httpx.Error(code, message)` 是否与 spec 约定一致
- service 返回的 sentinel error 是否被 handler 正确映射

### 4. 分页契约一致性
重点检查：
- 请求中的 `pagination.cursor` 是否透传和校验
- 非法 cursor 是否返回参数错误而不是 10600
- 响应中的 `next_cursor` 是否仅在有下一页时出现

### 5. 成功响应形状一致性
重点检查：
- 是否统一使用 `ApiResponse<T>`，并以 `code == 0` 表示成功
- `data` 是否为 spec 约定的对象/分页结构/空对象
- 空成功响应是否错误返回 `null` 或缺少 `data`

## Output Format

审计结果将整理为一份偏差清单，按严重度分类：

### P0 / 高风险
会直接导致客户端兼容性问题，例如：
- 错误码不一致
- 必填字段与 spec 不一致
- 返回结构与 spec 不一致

### P1 / 中风险
行为与 spec 有偏差，但短期通常可绕过，例如：
- 错误消息文本不一致
- 分页边界条件处理与文档不完全一致

### P2 / 低风险
文档与实现存在细微差异，但不影响正常调用，例如：
- 次要字段命名/顺序/描述不一致

每条偏差至少包含：
- spec 文件与端点
- 实现文件位置
- 观察到的当前行为
- 预期契约行为
- 推荐修复方向（优先改实现）

## Recommended Fix Strategy

默认策略：**优先修实现，不先改 spec**。

只有在满足以下条件时，才考虑反向修 spec：
1. 实现行为在多个模块中已经稳定复用
2. 客户端明显已经依赖当前实现
3. spec 的表述与仓库整体设计冲突，且实现更合理

## Verification

审计阶段完成后，不修改代码，只提交偏差报告。

若后续进入修复阶段，验证方式为：
- 为每个修复端点补充或更新对应 handler/service 测试
- 运行模块级 `go test -short ./internal/modules/<module>/...`
- 最终运行：
  - `go test -short ./...`
  - `go vet ./...`

## Success Criteria

当以下条件满足时，本轮审计完成：
1. `auth`、`users`、`moments/comments` 三个模块都完成高风险契约核对
2. 每个模块都有明确的偏差列表或“未发现高风险偏差”结论
3. 结果能够直接转化为后续实现修复任务，而不是停留在泛泛描述
