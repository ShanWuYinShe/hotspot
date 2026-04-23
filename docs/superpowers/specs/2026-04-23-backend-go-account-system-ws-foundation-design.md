# backend-go 账户 / 系统 / WebSocket 底座补齐设计

## 背景

`spec/` 是 HotSpot API 契约的唯一权威来源。当前 `backend-go` 已具备多数业务模块骨架，但在账户、系统公开配置、会话生命周期与 WebSocket 接入底座上仍存在若干缺口：

- `system` 公开端点未完全覆盖第一批目标范围
- `auth` 中部分外部依赖能力不适合在当前批次实现，但其余无第三方依赖能力需要继续按真实逻辑补齐
- `users` 账户闭环相关能力需要与 `auth` principal / session 语义保持一致
- realtime handler 已存在，但 `/ws/v1` 的正式装配与账户会话联动需要作为统一底座收口

本设计把这些共享依赖强、容易互相牵连的内容打成一个任务包，一次补齐，避免后续 `feed/search/chat/admin` 再反复穿透 auth / config / realtime 底座。

## 目标

第一批一次性补齐 `backend-go` 中与以下能力相关的所有无第三方依赖能力：

- 系统公开配置与系统公开信息
- 账户认证与会话生命周期
- 当前用户账户自助闭环
- WebSocket 接入底座与会话一致性

实现目标是：

> 复用现有 auth / session / realtime 架构，只修正契约缺口与装配层，确保 HTTP 与 WebSocket 在 principal 和 session 语义上保持一致。

## 范围

### 包含范围

#### 1. `internal/modules/system/`

补齐第一批范围内的 system 公开读取能力，统一由真实配置与公告来源提供数据。第一批的 system 范围固定为 `@spec/api_spec_system.md` 中已定义的 4 个端点：

- `POST /api/v1/system/get_meta`
- `POST /api/v1/system/health`
- `POST /api/v1/system/list_announcements`
- `POST /api/v1/system/get_client_config`

约束：

- `AppConfig -> ClientConfig` 投影作为唯一数据源
- 默认值、limits、contact 等信息不得散落在 handler 中手写
- 公告读取保持真实数据路径，不允许 stub

#### 2. `internal/modules/auth/`

保留现有 token / session 主链，只补齐第一批无第三方依赖能力：

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/guest`
- `POST /api/v1/auth/refresh_token`
- `POST /api/v1/auth/check_status`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/me/change_password`
- `POST /api/v1/auth/me/sessions/list`
- `POST /api/v1/auth/me/sessions/revoke`
- `POST /api/v1/auth/me/sessions/revoke_all`

实现要求：

- web / app 两端 token 投递语义必须与 `@spec/api_spec_auth.md` 一致
- session list / revoke / revoke_all 必须基于真实会话存储工作
- logout / revoke / revoke_all 必须与 realtime 连接关闭行为一致
- 不重写现有 auth 架构，只校正契约缺口和行为不一致

#### 3. `internal/modules/users/`

补齐与账户闭环直接相关、且不依赖第三方的当前用户能力：

- `POST /api/v1/users/me/profile/get`
- `POST /api/v1/users/me/profile/update`
- `POST /api/v1/users/me/location/update`
- `POST /api/v1/users/me/settings/get`
- `POST /api/v1/users/me/settings/update`
- `POST /api/v1/users/me/upgrade`

若实现过程中发现上述 `me/*` 端点依赖的 profile / settings / relationship 读取链路与契约不一致，可在 `users` 模块内一并修正其底层读取逻辑；但第一批不扩展公开搜索、社交列表或内容列表接口范围。

实现要求：

- 当前用户身份统一来自 principal
- 写操作写真实库，读操作读真实库
- 不在 users 模块自行管理 token / session 语义
- `guest -> normal upgrade` 沿用现有账户模型，不另起机制

#### 4. `internal/platform/realtime/` + `internal/bootstrap/`

将现有 realtime handler 正式装配到 HTTP 路由层，并与 session 生命周期保持一致：

- 在 bootstrap 路由层挂载 `/ws/v1`
- 握手鉴权继续复用 `platform/auth`
- 建连注册继续复用 `DefaultHub().RegisterUser(...)`
- session 撤销后关闭对应连接，保持 logout / revoke / revoke_all 行为一致

实现要求：

- 不新建第二套 websocket 鉴权体系
- 不新建第二套 hub / session 管理机制
- 保持现有 typing / subscription 能力仍可工作

### 排除范围

以下内容不属于第一批：

- `POST /api/v1/auth/request_password_reset`
- `POST /api/v1/auth/reset_password`
- `POST /api/v1/users/me/email/request_change`
- `POST /api/v1/users/me/email/confirm_change`
- `POST /api/v1/users/me/phone/request_change`
- `POST /api/v1/users/me/phone/confirm_change`
- 任何邮件、短信、验证码第三方发送链路
- `feed/search/map/share/media/reports` 扩展
- `groups/private_chats/private_messages/group_messages/notifications/admin` 扩展
- 大规模 WebSocket 业务事件补全
- 为未来需求提前引入新的横切抽象

## 模块边界

### `system`

`system` 作为系统公开读取模型，统一承担：

- API 版本元信息
- 健康检查
- 客户端公开配置
- 系统公告
- 第一批新增 system 公开信息

分层约束：

- handler：解析入参与返回 envelope
- service：组装 spec 所需响应结构
- source / repo：读取真实配置与真实公告数据

禁止在 handler 中直接写默认值或绕过 service 直接读取配置。

### `auth`

`auth` 只负责：

- 身份验证
- token 签发与刷新
- session 生命周期
- 当前 session / 其他 session 撤销

`auth` 不直接承担 profile / settings 业务规则。

### `users`

`users` 只负责：

- 当前用户 profile
- location
- settings
- guest upgrade

`users` 不直接生成 token，不直接管理 session 生命周期。

### `realtime`

`realtime` 只负责：

- `/ws/v1` 连接接入
- 鉴权后的连接注册
- 订阅与 typing 现有能力
- 连接与 session 失效的一致性

第一批不在 realtime 中扩展新的业务事件体系。

## 数据流

### `system` 数据流

所有 `system` 端点统一走：

`handler -> service -> config / announcement source`

目标是让 `get_meta`、`health`、`list_announcements`、`get_client_config` 与第一批新增 system 公开端点共享同一读取模型，避免多入口和硬编码常量。

### `auth` 数据流

所有 auth 端点延续当前主链：

`handler -> service -> auth/session/token primitives`

其中：

- `register/login/guest`：签发 session 与 token，并按 `client_type` 分流 web cookie / app body
- `refresh_token`：沿用当前 refresh 逻辑，只校正 web / app 契约语义
- `logout/revoke/revoke_all`：先撤销 session，再关闭对应 websocket 连接

核心原则：session 是权威，websocket 连接是附属资源。

### `users` 数据流

账户闭环相关 users 端点统一走：

`principal -> handler -> service -> repository`

其中 principal 是唯一当前用户来源。users 不自行推断当前用户，不单独维护会话状态。

### `/ws/v1` 数据流

`/ws/v1` 正式接入后延续当前 handler 逻辑：

- 建连
- 第一条消息必须为 `Auth`
- 使用 `platform/auth` 完成鉴权
- 注册到 `DefaultHub`
- 继续支持现有 subscription / typing 行为
- session 撤销时由 auth 主动关闭对应连接

## 状态变化边界

第一批只允许以下四类状态源发生变化：

### 1. 用户状态

由 `users` 模块更新：

- profile
- location
- settings
- guest upgrade

### 2. 会话状态

由 `auth` 模块更新：

- 创建 session
- refresh rotation
- logout current session
- revoke one session
- revoke other sessions

### 3. 连接状态

由 `realtime` 维护，但只能被建连 / 断连 / session 变化驱动：

- 鉴权成功后注册连接
- 客户端主动断开
- logout / revoke / revoke_all 触发断连

### 4. 系统公开配置状态

由 `AppConfig` 与公告来源提供，只读投影给客户端：

- meta
- client config
- announcements
- health

## 完成定义

### `system`

满足以下条件才算完成：

- 第一批范围内的 system 公开端点全部存在
- 返回结构与 `@spec/api_spec_system.md` 对齐
- 数据来自真实 source，不是写死 mock
- `ClientConfig` 投影规则收口一致

### `auth`

满足以下条件才算完成：

- 非第三方依赖 auth 端点全部契约对齐
- web / app token 投递语义正确
- session list / revoke / revoke_all 行为正确
- logout / revoke / revoke_all 能关闭对应 websocket 连接
- 不引入 stub 或临时假逻辑

### `users`

满足以下条件才算完成：

- 当前用户账户闭环可用
- 更新类接口写真实库
- 读取类接口读真实库
- `upgrade` 等账户动作复用现有 principal / session 体系

### `/ws/v1`

满足以下条件才算完成：

- 路由已正式挂载
- 认证成功可建立连接
- 现有 subscription / typing 能力仍可工作
- session 撤销可关闭对应连接
- 没有并行的 websocket 鉴权或 hub 体系

## 验证策略

### 模块级验证

补充并修正以下测试：

- `system` 新增端点测试
- `auth` cookie / body 契约测试
- logout / revoke / revoke_all 与 `CloseSession` 联动测试
- `users` 当前用户账户闭环测试
- realtime handler 路由接入后的握手测试

### 工程级验证

第一批开发完成的最低验证线：

- `go test -short ./...`
- `go build ./...`
- `go vet ./...`

### 契约对照验证

逐项核对：

- 端点存在性
- 认证要求
- `ApiResponse<T>` envelope
- `code / message / data` 语义
- web / app 差异行为

## 实现顺序

为减少返工，按以下顺序实现：

1. 补 bootstrap 装配与 `/ws/v1` 路由接入
2. 补 `system` 全量公开读取面
3. 修 `auth` 契约缺口与 session / websocket 一致性
4. 补 `users` 账户闭环
5. 一次性跑测试与构建收尾

## 风险控制

### 不重写已有主链

第一批默认复用：

- `platform/auth`
- 现有 session / token 机制
- `realtime.DefaultHub()`
- 现有 module 结构
- 现有 handler -> service -> repo 分层

只有在与 spec 冲突、装配缺失或无法满足第一批闭环时才调整实现。

### 不引入新的横切抽象

第一批禁止顺手引入：

- 新的 account orchestrator
- 新的统一 facade 层
- 第二套 websocket auth middleware
- 第二套 config 读取入口
- 为未来模块预留的大抽象

### WebSocket 只解决接入与一致性

第一批只解决：

- `/ws/v1` 正式可连
- session 撤销时连接能被正确关闭

不做事件补发、离线重放、复杂恢复或新的业务事件体系。

## 结果

第一批将作为一个单任务包实施：

> 一次性补齐 `backend-go` 中与账户、系统公开配置、会话生命周期、WebSocket 接入底座相关的所有无第三方依赖能力；复用现有 auth / session / realtime 架构，只修正缺口与装配层，确保 HTTP 与 WebSocket 在 principal 和 session 语义上保持一致。
