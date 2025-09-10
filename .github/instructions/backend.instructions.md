# HotSpot 项目后端开发规范

---

> **本规范与前端、通用规范的关系**：
> - 本文件专注后端 Rust/PostgreSQL 开发规范，包括 sqlx 查询、Option 字段极简化、分层架构等
> - 配合 `frontend.instructions.md`（前端Vue3开发规范）和 `project.instructions.md`（通用项目规范）
> - 三个规范文件风格统一、内容互补，重点强调 Option 字段极简原则和 sqlx ! 语法强制使用

---

## 1. 依赖管理规范

### 包管理操作规则
- **禁止直接修改 Cargo.toml 文件**，必须通过包管理器命令操作
- 使用 `cargo add xxx --features yyy,zzz` 增加依赖
- 使用 `cargo remove xxx` 删除依赖
- 使用包管理器确保添加的依赖和特性是真实存在的，避免引用不存在的特性
### 依赖安全与兼容性要求
- 引入新依赖前，需联网查询该依赖包的最新版本、稳定性、维护状态和可能的替代方案
- 评估依赖安全性和兼容性，避免使用已弃用包
- 定期审查和更新项目依赖，保持技术栈现代性

## 2. 代码质量管理规范

#### 官方文档优先原则（强制）

**核心原则**：
- **文档优先验证**：使用任何第三方库、框架、crate前，必须先查阅官方文档确认正确用法
- **零假设开发**：不得基于经验或猜测使用API，所有用法必须有官方文档支撑
- **实时验证要求**：在编写代码时立即验证API的存在性和正确性

**后端开发强制要求**：
- **Rust Crate**：使用任何 crate 前必须查阅 docs.rs 或官方文档
  - 验证 crate 版本兼容性
  - 确认 API 方法和参数的正确用法
  - 理解 feature flags 的作用和启用方式
- **sqlx 查询**：确认 query!、query_as!、query_scalar! 宏的正确用法
- **异步编程**：验证 async-trait、tokio 等异步相关 API 的最新用法
- **数据库操作**：确认 PostgreSQL 语法和 PostGIS 扩展的正确用法

**违规处理**：
- **编译错误**：因未查阅文档导致的编译错误必须立即修复
- **运行时错误**：因 API 误用导致的运行时错误视为严重质量问题
- **代码审查**：代码审查时必须验证是否遵循了文档优先原则

**文档查阅工具和资源**：
- docs.rs、doc.rust-lang.org、postgresql.org、postgis.net
- GitHub README、官方示例代码、Stack Overflow（仅作参考）

#### 死代码处理策略
- **零警告政策**：项目必须保持 `cargo check` 无警告状态
- **架构保留原则**：对于未来有价值的代码架构，使用 `#[allow(dead_code)]` 注解标记而不删除
- **架构合理性原则**：定期检查代码设计是否符合真实业务场景，清理不合理的存储和处理逻辑
- **分类标记**：
  - 完整的功能模块（如 WebSocket 系统）：在模块级别添加 `#[allow(dead_code)]`
  - 单独的函数/结构体：在项目级别添加注解
  - 临时未使用代码：添加 TODO 注释说明预期使用时间
- **定期审查**：每个版本发布前审查已标记的死代码，决定激活或清理
- **不合理设计清理**：
  - 通知系统应使用实时WebSocket，避免数据库持久化存储
  - API设计应简洁单一，避免多次调用组合完成单一业务逻辑
  - 数据库表设计应基于真实业务需求，避免"可能有用"的冗余存储
- **保留的核心架构**：
  - WebSocket 实时通信系统（连接管理、房间管理、消息路由）
  - 智能通知系统（多策略通知分发）
  - 缓存抽象层（高性能数据访问接口）
  - 作业处理框架（后台任务和异步处理）
  - 监控和指标系统（性能监控工具）

#### 编译质量保证
- 所有提交必须通过 `cargo check` 无警告检查
- 使用 `cargo clippy` 进行代码质量检查

#### 后端特定清理要求
- **测试文件归位**：所有测试文件必须放在 `src/tests/` 目录下，不得散布在业务代码目录中
- **依赖清理**：移除未使用的依赖包和feature
- **编译产物管理**：及时清理临时编译输出文件

## 3. 项目架构规范

### 后端代码组织原则
- 严格分层架构设计（Controller/Service/Repository/Model）
- 数据库表定义在 `migrations` 目录下

### 分层架构

#### Controller 层
- 处理 HTTP 请求参数解析和校验
- 调用 Service 层方法完成业务逻辑
- 将 Service 层返回的领域模型转换为 API 响应格式
- 处理错误并返回适当的 HTTP 响应

#### Service 层
- 实现业务逻辑和规则
- 直接与数据库交互
- 如有复杂数据访问逻辑可引入 Repository 层，避免 Service 层过重
- 返回领域模型或工具模型，不返回 HTTP 相关响应格式
- 遵循原子化原则，每个函数实现最小化功能

#### Model 层
- 定义领域模型数据结构
- 实现数据转换方法
- 定义业务实体间的关系和约束

#### Utils/Config 层
- 提供通用工具函数和配置
- 实现跨服务的共享功能
- 提供中间件和辅助组件

## 4. WebSocket 系统架构规范

#### 核心组件职责划分

##### WebSocketManager
- **职责**：WebSocket 连接的基础管理
- **功能**：连接生命周期管理、房间订阅/取消订阅、基础协议处理（ping/pong）
- **使用场景**：仅在需要连接管理或状态查询时使用

##### WebSocketNotificationService  
- **职责**：业务逻辑相关的 WebSocket 通知
- **功能**：消息、群组事件、系统通知等的发送
- **使用场景**：Controllers 调用来发送特定的业务通知

#### 智能通知策略系统

##### NotificationPolicyService
- **策略类型**：
  - 好友关系通知：只向好友发送状态更新
  - 群组成员通知：向共同群组成员发送通知
  - 地理位置通知：向附近用户发送通知（可配置距离半径）
  - 智能组合：支持多种策略同时启用，自动去重

##### 环境配置策略（字段与类型）
- 生产环境推荐：
  - enable_friend_notifications: bool
  - enable_group_notifications: bool
  - enable_nearby_notifications: bool
  - nearby_radius_km: f64
  - enable_global_broadcast: bool（建议为 false）

- 开发环境推荐：
  - enable_friend_notifications: bool
  - enable_group_notifications: bool
  - enable_nearby_notifications: bool
  - nearby_radius_km: f64（可更大便于调试）
  - enable_global_broadcast: bool（可选启用）

#### 通知发送规范
- **禁止全局广播**：严禁使用 `broadcast()` 方法向所有在线用户发送通知
- **精准推送**：必须通过策略服务获取目标用户列表，然后定向发送
- **去重处理**：多策略组合时自动去重，避免用户收到重复通知
- **性能监控**：记录通知发送数量和目标用户数，监控系统性能

#### WebSocket系统完整性保护
- **架构完整性**：整个WebSocket系统（连接管理、房间管理、消息路由、通知策略等）必须作为一个整体保留
- **死代码标记**：所有暂未使用的WebSocket相关代码已通过 `#[allow(dead_code)]` 注解保护，禁止删除
- **功能模块清单**：
  - `ws/manager.rs`：WebSocket连接管理器，处理连接生命周期
  - `ws/connection_manager.rs`：连接池管理，用户连接映射
  - `ws/room_manager.rs`：房间管理，群组和私聊房间
  - `ws/message_router.rs`：消息路由分发，替代全局广播
  - `ws/message_types.rs`：完整的消息类型定义
  - `ws/errors.rs`：WebSocket错误处理
  - `services/notification/`：智能通知策略系统
- **激活条件**：当需要实现实时功能时，直接移除相关 `#[allow(dead_code)]` 注解即可激活
- **维护要求**：定期验证WebSocket系统的编译完整性，确保随时可激活

## 5. 数据库操作规范

#### 核心SQL查询规范
- **强制要求：所有数据库查询只能使用 `sqlx::query!`、`sqlx::query_as!`、`sqlx::query_scalar!` 这三个宏版本，禁止使用非宏版本的函数（如 `sqlx::query`、`sqlx::query_as`、`sqlx::query_scalar` 等）**
- `sqlx::query_as!`：查询返回结构体时使用
- `sqlx::query_scalar!`：查询返回单个值时使用
- `sqlx::query!`：查询不返回结果时使用
- 所有参数必须通过参数绑定传递，禁止字符串拼接 SQL
- 使用枚举类型代替字符串常量
- 被sqlx::query_as! 使用的结构体不需要自己去实现 `FromRow` trait，query_as! 会自动处理

#### Option字段极简规范与sqlx非NULL标记规范

**Option字段极简原则：**
- sqlx::query_as! 查询结构体字段只在数据库字段确实可能为 NULL 时才使用 Option<T>，其他情况一律使用非 Option 类型

**sqlx ! 语法强制使用规范：**
- 对于数据库中 NOT NULL 字段、聚合函数（COUNT、SUM 等）、COALESCE 处理后的字段、布尔表达式等确保非 NULL 的字段，必须在 SQL 查询中使用 `"field_name!"` 语法告诉 sqlx 该字段不会为 NULL

**具体应用规则：**
- 数据库 NOT NULL 字段：结构体用非 Option，SQL 用 `"field_name!"`
- COUNT、SUM、AVG 等聚合函数：结构体用非 Option，SQL 用 `"COUNT(*) as count!"`
- COALESCE 处理的字段：结构体用非 Option，SQL 用 `"COALESCE(field, default) as field!"`
- 布尔表达式（EXISTS、比较运算等）：结构体用非 Option，SQL 用 `"expression as result!"`
- 数据库 NULL 字段且业务逻辑确实需要 NULL 值：结构体用 Option<T>，SQL 不加 `!`

字段规范示例（结构与类型）：
- UserRow：
  - id: i64（NOT NULL 主键，对应 SQL 标注 "id!"）
  - username: String（NOT NULL，对应 "username!"）
  - email: Option<String>（可为 NULL）
  - is_active: bool（布尔表达式结果）
  - message_count: i64（聚合结果）

#### 参数绑定顺序
- 业务参数（ID、关键词等）
- 默认值/配置参数（如默认经纬度等）
- 分页参数（limit、cursor）

#### 数据库迁移规范
- 每个表一个文件
- 开发阶段可每次都用建表的方式而不是带有更改，因为会重建整个数据库
- 生产环境应采用增量迁移，避免全量重建，保障数据安全和迁移效率

#### Option字段和sqlx最佳实践总结
**核心原则：极简化Option使用，强制sqlx非NULL标记**

1. **Option字段极简化**：只有数据库字段确实可能为NULL时才使用Option<T>
2. **sqlx ! 语法强制化**：对于任何确保非NULL的字段都必须使用"field_name!"语法
3. **常见应用场景**：
   - 主键、外键（NOT NULL约束）：非Option + "id!"
   - 聚合函数结果：非Option + "COUNT(*) as count!"
   - COALESCE处理：非Option + "COALESCE(field, default) as field!"
   - 布尔表达式：非Option + "expression as result!"
   - 可选业务字段（如email、phone）：Option<T>，不加!
4. **代码重构指南**：现有代码中发现Option<i64>、Option<String>等用于非NULL字段时，必须重构为非Option并在SQL中加"!"标记
5. **编译验证**：重构后必须通过cargo check确保编译无误，sqlx会在编译时验证SQL和结构体的匹配性

## 6. 分页实现规范

#### 基本原则
1. 所有分页查询必须使用游标分页，禁止使用偏移量分页
2. 游标必须始终与表的 `id` 字段（自增主键，i64 类型）比较，禁止与 `user_id`、UUID 或其他业务主键比较
3. 返回的下一页游标值必须是查询结果中最后一条记录的 `id` 字段值
4. 所有消息、群组等相关分页必须用游标分页，且游标字段为表的自增主键 id

#### SQL 查询模式（描述）
- 降序：id < cursor，按 id DESC 排序，取 limit + 1 条
- 升序：id > cursor，按 id ASC 排序，取 limit + 1 条

#### 游标初始值处理
- 首次加载（无游标）时，默认游标值为 0，即：`cursor = 0 OR cursor IS NULL`
- SQL 中处理初始游标：`WHERE ($cursor = 0 OR id < $cursor)`
- 请求时通过传递上一次返回的游标获取下一页数据

#### 分页结果处理
1. 查询时总是请求 `limit + 1` 条记录
2. 如果返回记录数等于 `limit + 1`，则表示还有更多数据（`has_more = true`）
3. 返回给客户端的记录不包括额外查询的一条记录
4. 下一页游标应设置为最后一条返回记录的 `id` 字段值

#### 复杂排序情况处理（描述）
- 使用主排序字段 + id 次级排序，保证分页稳定性

#### 分页数据模型（字段说明）
- PaginationParams: cursor: Option<i64>；limit: Option<i64>
- PaginatedResponse<T>: items: Vec<T>；pagination: PaginationMeta
- PaginationMeta: has_more: bool；next_cursor: Option<i64>

#### 参考实现
- `user_service.rs`：`find_nearby_users`、`find_users_by_name`
- `group_service.rs`：`find_nearby_groups`、`find_groups_by_name`

## 7. PostgreSQL 与 Rust 类型映射规范

#### 枚举类型映射
- 数据库中使用 enum 类型
- Rust 中使用对应的枚举定义并添加 sqlx 注解

#### 地理位置查询
- 使用 PostGIS 扩展进行地理位置查询
- 使用 `ST_Distance` 计算地理距离
- 使用 `::geography` 类型转换确保计算基于地理距离（以米为单位）
- 使用 `COALESCE` 处理可能的 NULL 值
- 使用空间索引优化查询性能

## 8. Trait/泛型与异步接口规范

- 所有异步 trait（如 Repository、Service、JobHandler 等）目前使用 `async-trait` crate 来处理 trait 中的 async fn
- trait 定义必须添加 `#[async_trait]` 注解
- trait 实现（impl 块）也必须添加 `#[async_trait]` 注解
- trait/impl 中直接写 `async fn`，由 async-trait 宏处理底层的 Future/Pin/Box 包裹
- 函数签名应尽量不显式声明生命周期，除非必要（如参数/返回值确实涉及引用类型且需指定生命周期）
- Service、Repository、JobHandler、Cache、Notification 等分层依赖注入时，优先使用具体类型或泛型参数，避免 trait object
- 代码中如需扩展通用性，推荐使用泛型 trait bound（如 `T: GroupRepository + Send + Sync`），而不是 trait object
- 发现 trait object 用法（如 `dyn Trait`、`Box<dyn Trait>`、`Arc<dyn Trait>`）时，必须重构为具体类型或泛型
- 相关代码示例和重构方式请参考 `group_service.rs`、`PgGroupRepository` 等文件

#### 接口定义规范（字段/签名说明，不含代码）
- UserRepository：
  - find_by_id(user_id: i64) -> Result<Option<UserModel>, AppError>
  - set_password(user_id: i64, password_hash: &str) -> Result<(), AppError>

#### async-trait 使用示例
```rust
use async_trait::async_trait;

// trait 定义必须添加 #[async_trait]
#[async_trait]
pub trait UserRepository: Send + Sync {
    async fn find_by_id(&self, user_id: i64) -> Result<Option<UserModel>, AppError>;
    async fn set_password(&self, user_id: i64, password_hash: &str) -> Result<(), AppError>;
}

// trait 实现也必须添加 #[async_trait]
#[async_trait]
impl UserRepository for PostgresUserRepository {
    async fn find_by_id(&self, user_id: i64) -> Result<Option<UserModel>, AppError> {
        // ...实现...
    }
    async fn set_password(&self, user_id: i64, password_hash: &str) -> Result<(), AppError> {
        // ...实现...
    }
}
```

> 说明：虽然 Rust 1.75+ 已原生支持 trait 里的 async fn，但在复杂场景下（如auto trait bounds、生命周期推断等）仍有一些限制。在Rust编译器完全支持trait中async fn的自动推断之前，使用成熟的 async-trait crate 可以避免这些问题，提供更好的兼容性和稳定性。未来当Rust编译器完全支持时，可以逐步迁移到原生语法。

## 9. 异步任务队列规范

### 任务队列系统架构

#### 核心原则
- **业务解耦**: 主业务流程不得因异步任务失败而阻断
- **专业队列**: 所有异步操作必须通过专业任务队列系统处理，禁止使用简单的 `tokio::spawn`
- **任务重试**: 失败任务必须支持自动重试机制
- **状态监控**: 任务执行状态必须可追踪和监控

#### 任务队列组件架构

##### JobDispatcher (jobs/dispatcher.rs)
- **职责**: 任务调度和分发的中央控制器
- **功能**: 任务提交、处理器注册、任务路由
- **使用场景**: Service层通过JobDispatcher提交异步任务

##### TaskHandler (jobs/*_handlers.rs)
- **职责**: 具体任务类型的处理逻辑实现
- **功能**: 任务执行、错误处理、业务逻辑调用
- **类别**:
  - `UserHandlers`: 用户相关任务（位置初始化、欢迎邮件、通知推送等）
  - `GroupHandlers`: 群组相关任务（消息摘要、成员活动等）
  - `LocationHandlers`: 位置相关任务（签到索引、附近通知等）

##### TaskModels (jobs/*_models.rs)
- **职责**: 任务数据结构定义
- **功能**: 任务参数封装、序列化支持
- **要求**: 所有任务数据必须支持JSON序列化/反序列化

#### 任务队列集成规范

##### Service层集成模式（字段说明）
- UserService 必须包含：
  - config_service: ConfigService
  - user_repository: PgUserRepository
  - job_dispatcher: Option<JobDispatcher>

##### 任务提交模式（描述）
- 业务流程中检查 `job_dispatcher` 是否存在，存在则提交任务；失败记录日志但不阻断主流程

#### 任务处理器实现规范

##### Handler接口实现（描述）
- handle(&self, job: &Job<T>) -> JobHandlerResult：读取 job.data 执行业务逻辑；成功返回 Ok，失败分类记录错误

##### 错误处理规范
- **分类错误**: 区分临时错误（可重试）和永久错误（不可重试）
- **详细日志**: 记录足够的上下文信息便于排查问题
- **优雅降级**: 任务失败不应影响系统整体可用性

#### 主要任务类型与应用场景

##### 用户位置初始化 (LocationIndexJob)
- **触发时机**: 用户注册、临时用户创建时
- **处理逻辑**: 为新用户设置默认地理位置
- **原则**: 位置初始化失败不得阻断用户注册流程

##### 通知推送 (NotificationJob)
- **触发时机**: 系统事件、用户交互、群组活动时
- **处理逻辑**: 通过WebSocket或其他渠道发送通知
- **原则**: 支持多种通知类型和发送策略

##### 欢迎邮件 (WelcomeEmailJob)
- **触发时机**: 新用户注册完成时
- **处理逻辑**: 发送欢迎邮件和入门指导
- **原则**: 邮件发送失败不影响用户正常使用

#### 任务队列配置管理

##### 应用启动时集成（描述）
- JobService：基于配置条件初始化；UserService 通过 with_job_dispatcher 注入 dispatcher

##### 环境配置要求
- **开发环境**: 任务队列可选，允许禁用以简化开发
- **生产环境**: 任务队列必须启用，确保系统健壮性
- **测试环境**: 提供Mock任务调度器以隔离测试

#### 任务队列监控与运维

##### 日志记录规范
- **任务开始**: 记录任务ID、类型、参数
- **任务进度**: 关键步骤的执行状态
- **任务完成**: 执行结果、耗时统计
- **任务失败**: 错误类型、重试次数、失败原因

##### 性能监控指标
- **任务吞吐量**: 每秒处理的任务数量
- **任务延迟**: 从提交到开始执行的时间
- **失败率**: 任务失败的比例和原因分析
- **重试统计**: 重试次数分布和成功率

#### 任务队列最佳实践

##### 任务设计原则
- **幂等性**: 任务可以安全地重复执行
- **原子性**: 任务内部操作要么全部成功要么全部失败
- **时效性**: 合理设置任务的有效期和超时时间
- **独立性**: 任务之间避免强依赖关系

##### 代码质量要求
- **类型安全**: 所有任务数据结构必须有完整的类型定义
- **错误处理**: 完善的错误分类和处理机制
- **测试覆盖**: 每个任务处理器都必须有对应的测试用例
- **文档完善**: 清晰的任务接口和使用文档

##### 迁移指导
- **现有代码重构**: 将 `tokio::spawn` 调用替换为任务队列调用
- **渐进式迁移**: 优先迁移关键业务流程，逐步覆盖所有异步操作
- **兼容性保证**: 保持API接口不变，内部实现升级为任务队列
- **性能验证**: 确保任务队列化后的性能满足业务要求

---

*规范版本：v1.4*  
*最后更新：2025年7月15日*
