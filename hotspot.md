# HotSpot 项目需求文档

## 文档结构

- [HotSpot 项目需求文档](#hotspot-项目需求文档)
  - [文档结构](#文档结构)
  - [1. 项目概览](#1-项目概览)
    - [1.1 项目简介](#11-项目简介)
    - [1.2 当前实现状态 ✅](#12-当前实现状态-)
      - [1.2.1 已完成功能模块（✅）](#121-已完成功能模块)
      - [1.2.2 技术架构完成度](#122-技术架构完成度)
      - [1.2.3 API接口完成度](#123-api接口完成度)
      - [1.2.4 待优化功能（🔄）](#124-待优化功能)
      - [1.2.5 架构优化完成（✅ 2025年7月15日）](#125-架构优化完成-2025年7月15日)
      - [1.2.6 路由架构重构完成（✅ 2025年7月27日）](#126-路由架构重构完成-2025年7月27日)
    - [1.3 目标与愿景](#13-目标与愿景)
    - [1.4 核心功能概述](#14-核心功能概述)
    - [1.5 用户旅程与场景](#15-用户旅程与场景)
  - [2. 系统架构](#2-系统架构)
    - [2.1 总体架构](#21-总体架构)
      - [2.1.1 系统组件关系](#211-系统组件关系)
      - [2.1.2 数据流向](#212-数据流向)
    - [2.2 技术栈选择](#22-技术栈选择)
      - [2.2.1 前端技术栈](#221-前端技术栈)
      - [2.2.2 后端技术栈](#222-后端技术栈)
      - [2.2.3 开发工具与环境](#223-开发工具与环境)
    - [2.3 路由架构设计](#23-路由架构设计)
      - [2.3.1 路由分层策略](#231-路由分层策略)
      - [2.3.2 命名规范](#232-命名规范)
      - [2.3.3 架构优势](#233-架构优势)
    - [2.4 后端重构方案（端口/适配器 + 能力装配）](#24-后端重构方案端口适配器--能力装配)
      - [核心理念](#核心理念)
      - [模块/目录结构（描述）](#模块目录结构描述)
      - [Feature 与默认能力](#feature-与默认能力)
      - [数据访问与分页规范（强制）](#数据访问与分页规范强制)
      - [实时通道与策略](#实时通道与策略)
      - [任务队列与缓存](#任务队列与缓存)
      - [密码安全实现架构](#密码安全实现架构)
      - [迁移步骤（低风险）](#迁移步骤低风险)
  - [3. 功能需求](#3-功能需求)
    - [3.1 用户管理功能](#31-用户管理功能)
      - [3.1.1 功能需求](#311-功能需求)
      - [3.1.2 认证系统API](#312-认证系统api)
      - [3.1.3 用户数据API](#313-用户数据api)
    - [3.2 地理位置服务](#32-地理位置服务)
      - [3.2.1 功能需求](#321-功能需求)
      - [3.2.2 地图显示功能](#322-地图显示功能)
      - [3.2.3 地理位置计算实现](#323-地理位置计算实现)
      - [3.2.4 地理位置数据处理原则](#324-地理位置数据处理原则)
    - [3.3 社交功能](#33-社交功能)
      - [3.3.1 功能需求](#331-功能需求)
      - [3.3.2 API 接口](#332-api-接口)
    - [3.4 消息通讯系统](#34-消息通讯系统)
      - [3.4.1 功能需求](#341-功能需求)
      - [3.4.2 API 接口](#342-api-接口)
  - [4. 代码质量管理](#4-代码质量管理)
    - [4.1 代码质量标准](#41-代码质量标准)
      - [4.1.1 官方文档优先原则（强制）](#411-官方文档优先原则强制)
      - [4.1.2 编译质量要求](#412-编译质量要求)
      - [4.1.3 后端代码合理性检查规范](#413-后端代码合理性检查规范)
      - [4.1.4 死代码处理原则](#414-死代码处理原则)
    - [4.2 项目清理规范](#42-项目清理规范)
      - [4.2.1 Agent模式代码生成清理要求](#421-agent模式代码生成清理要求)
      - [4.2.2 清理执行标准](#422-清理执行标准)
      - [4.2.3 长期维护原则](#423-长期维护原则)
    - [4.3 认证和Token存储调试规范](#43-认证和token存储调试规范)
      - [4.3.1 Token存储故障诊断](#431-token存储故障诊断)
      - [4.3.2 调试工具要求](#432-调试工具要求)
      - [4.3.3 Pinia持久化调试](#433-pinia持久化调试)
      - [4.3.4 故障排查流程](#434-故障排查流程)
    - [4.4 架构保留策略](#44-架构保留策略)
      - [4.4.1 WebSocket实时通信系统（已完整实现并激活）](#441-websocket实时通信系统已完整实现并激活)
      - [4.4.2 保留的扩展架构](#442-保留的扩展架构)
      - [4.4.3 维护策略](#443-维护策略)
      - [4.4.4 实时通道策略（当前与规划）](#444-实时通道策略当前与规划)
  - [5. 测试架构设计](#5-测试架构设计)
    - [5.1 后端测试架构](#51-后端测试架构)
      - [5.1.1 当前测试状态与改进方向](#511-当前测试状态与改进方向)
      - [5.1.2 测试分层架构（文字描述）](#512-测试分层架构文字描述)
  - [6. 数据库设计](#6-数据库设计)
    - [6.1 数据库架构概览](#61-数据库架构概览)
      - [6.1.1 核心设计原则](#611-核心设计原则)
      - [6.1.2 数据库扩展](#612-数据库扩展)
    - [6.2 核心数据表设计](#62-核心数据表设计)
      - [6.2.1 用户系统表](#621-用户系统表)
      - [6.2.2 群组系统表](#622-群组系统表)
      - [6.2.3 消息系统表](#623-消息系统表)
      - [6.2.4 生活瞬间系统表](#624-生活瞬间系统表)
    - [6.3 数据库性能优化](#63-数据库性能优化)
      - [6.3.1 地理位置查询优化](#631-地理位置查询优化)
      - [6.3.2 数据清理策略](#632-数据清理策略)
  - [7. API接口设计](#7-api接口设计)
    - [7.1 API设计原则](#71-api设计原则)
      - [7.1.1 统一响应格式（与当前实现保持一致）](#711-统一响应格式与当前实现保持一致)
      - [7.1.2 API 类型定义规范](#712-api-类型定义规范)
      - [7.1.3 公共数据模型（API 与 WebSocket 复用）](#713-公共数据模型api-与-websocket-复用)
    - [7.2 认证与用户API](#72-认证与用户api)
      - [7.2.1 认证接口（8个端点，前缀：/api/v1/auth）](#721-认证接口8个端点前缀apiv1auth)
      - [7.2.2 用户数据接口（9个端点）](#722-用户数据接口9个端点)
    - [7.3 群组管理API](#73-群组管理api)
      - [7.3.1 群组操作 (9个端点)](#731-群组操作-9个端点)
    - [7.6 管理接口](#76-管理接口)
      - [7.6.1 用户ID分区管理 (6个端点，前缀：/api/v1/protected/admin，需管理员认证）](#761-用户id分区管理-6个端点前缀apiv1protectedadmin需管理员认证)
    - [7.7 API性能与安全](#77-api性能与安全)
      - [7.7.1 性能优化](#771-性能优化)
      - [7.7.2 安全措施](#772-安全措施)
    - [7.8 后端密码安全实现指南](#78-后端密码安全实现指南)
      - [7.8.1 核心组件实现](#781-核心组件实现)
      - [7.8.2 认证流程实现](#782-认证流程实现)
      - [7.8.3 安全注意事项](#783-安全注意事项)
    - [7.9 接口与 WebSocket 速查表](#79-接口与-websocket-速查表)
  - [8. 部署与运维](#8-部署与运维)
    - [8.1 部署架构](#81-部署架构)
      - [8.1.1 生产环境部署](#811-生产环境部署)
      - [8.1.2 环境配置](#812-环境配置)
    - [8.2 监控与维护](#82-监控与维护)
      - [8.2.1 性能监控](#821-性能监控)
      - [8.2.2 日志管理](#822-日志管理)
    - [8.3 数据备份与恢复](#83-数据备份与恢复)
      - [8.3.1 数据库备份策略](#831-数据库备份策略)
      - [8.3.2 灾难恢复计划](#832-灾难恢复计划)
  - [总结](#总结)

## 1. 项目概览

### 1.1 项目简介

HotSpot（HotSpot）是一个基于地理位置的社交应用，允许用户在特定地理位置发布动态、发现附近的用户和群组，以及进行实时聊天交流。该项目采用现代化的技术栈，提供流畅的用户体验和强大的社交功能。

### 1.2 当前实现状态 ✅

**项目已完成核心功能开发，实现状态如下：**

#### 1.2.1 已完成功能模块（✅）

1. **用户认证系统**
   - 用户注册/登录
   - 临时用户创建
   - 密码设置/修改
   - 令牌刷新机制
   - JWT身份验证

2. **地理位置服务**
   - 用户位置更新
   - 基于位置的用户搜索
   - 地理距离计算
   - PostGIS地理位置索引

3. **动态系统**
   - 创建/删除动态
   - 附近动态搜索
   - 动态点赞/取消点赞
   - 动态评论功能
   - 标签管理系统
   - 动态热力图数据

4. **群组管理**
   - 群组创建/更新/删除
   - 群组成员管理
   - 角色权限控制
   - 群组搜索功能
   - 群组邀请/加入/退出

5. **消息通讯系统**
   - 私聊会话管理
   - 群组消息
   - 消息发送/接收
   - 消息状态管理
   - 未读消息统计
   - 消息搜索功能

6. **WebSocket实时通知**
   - 实时消息推送
   - 在线状态更新
   - 群组事件通知
   - 新动态提醒
   - 打字指示器
   - 智能通知管理

7. **前端完整实现**
   - Nuxt4 + TypeScript架构
   - Nuxt UI v3组件
   - Pinia状态管理
   - 响应式设计
   - PWA支持
   - 微信JSSDK集成

#### 1.2.2 技术架构完成度

- **后端（Rust）**: 100% 完成
  - Axum REST API框架
  - PostgreSQL + PostGIS数据库
  - Redis缓存系统
  - WebSocket服务
  - 异步任务队列
  - 监控与健康检查

- **前端（Vue3）**: 100% 完成
  - 完整的页面和组件
  - 状态管理和API服务
  - WebSocket通知系统
  - 移动端适配
  - 错误处理机制

#### 1.2.3 API接口完成度

所有核心API接口已实现并可用：
- 用户模块：10个接口 ✅
- 群组模块：9个接口 ✅
- 动态模块：8个接口 ✅
- 消息模块：10个接口 ✅
- WebSocket通知：完整实现 ✅

#### 1.2.4 待优化功能（🔄）

1. **测试覆盖率提升**
   - 单元测试补充
   - 集成测试完善
   - API测试自动化

2. **性能优化**
   - 数据库查询优化
   - WebSocket连接优化

3. **监控增强**
   - 详细的业务指标
   - 性能监控仪表板

4. **功能模块整合**
   - 系统重构：已完成从checkin到moment的系统重构，提供更好的社交体验和生活分享功能
   - 现状：临时保留接口但返回"功能未实现"错误

#### 1.2.5 架构优化完成（✅ 2025年7月15日）

**已完成的架构清理**：
1. **notification数据库存储系统移除**
   - 删除了 `user_notifications` 数据库表及相关索引
   - 移除了 `notification_repository.rs` 冗余代码文件
   - 清理了用户仓库中不必要的通知方法
   - 统一为WebSocket实时通知，提升了系统架构的合理性

**设计原则优化**：
- **实时性优先**：通知系统改为纯WebSocket实时推送，删除不必要的数据库持久化
- **架构简化**：移除了双重通知系统（数据库+WebSocket），统一为单一高效的实时通知
- **合理性验证**：基于真实世界使用场景，通知应该是瞬时的而非持久化的

**技术收益**：
- 减少了数据库存储压力和查询复杂度
- 提升了通知推送的实时性和性能
- 简化了代码维护复杂度

#### 1.2.6 路由架构重构完成（✅ 2025年7月27日）

**路由重构成果**：
1. **认证系统独立化**
   - 创建独立的 `auth_routes.rs` 模块
   - 将认证相关功能从用户路由中分离
   - 实现公开认证路由和受保护用户数据路由的清晰分离

2. **API路径标准化**
   - 采用 `动词_名词` 语义化命名规范
   - 重构16个API端点的路径命名
   - 提升了API的可读性和维护性

3. **架构合理性提升**
   - 职责分离：认证逻辑与用户数据管理分离
   - 安全性：公开和受保护路由明确区分
   - 可扩展性：模块化设计便于未来功能扩展

重构结果摘要（文字描述）：
- 原 /api/v1/user/* 同时承载认证与用户数据，现拆分为 /api/v1/auth/*（认证）与 /api/v1/user/*（用户数据）。
- 命名统一采用语义化风格，如 create_temp、upgrade_account、search_by_id 等，去除歧义，提高一致性。

**技术收益**：
- 前后端代码架构更加清晰
- API语义更加明确，便于团队协作
- 为微服务拆分预留了架构空间

### 1.3 目标与愿景

创建强大、易用、具有创新性的地理位置社交平台，为用户提供：

- **即时发现**：发现附近有趣的人和地点
- **实时互动**：无延迟的消息传递和位置共享
- **社区建设**：帮助用户建立本地化的社交网络
- **隐私保护**：在分享与隐私之间找到最佳平衡

### 1.4 核心功能概述

基于地理位置的核心功能：
- **生活瞬间分享**：记录和分享生活中的精彩时刻
- **附近发现**：寻找附近的用户、群组和活动
- **实时聊天**：支持一对一和群组聊天
- **群组管理**：创建和管理基于位置的群组

### 1.5 用户旅程与场景

**新用户引导流程**：
1. 快速注册或使用临时用户身份
2. 授权位置服务（可选）
3. 发现附近的群组和用户
4. 参与互动和分享生活瞬间

**典型使用场景**：
- 在咖啡厅分享moment并发现其他咖啡爱好者
- 在活动现场创建临时群组进行交流
- 通过位置标签寻找有共同兴趣的人
- 在新城市通过附近用户获取当地建议

## 2. 系统架构

### 2.1 总体架构

#### 2.1.1 系统组件关系

系统组件关系（文字描述）：前端（Nuxt4+Vue3，Nuxt UI v3，Pinia）通过 HTTP（Axum）与 WebSocket 与后端交互；后端使用 PostgreSQL+PostGIS 存储与空间查询，Redis 用于缓存，RabbitMQ 处理异步任务，Prometheus 采集性能与业务指标。

#### 2.1.2 数据流向

1. **用户请求流**：前端 → 中间件验证 → 控制器 → 服务层 → 数据库
2. **实时通知流**：WebSocket → 消息路由 → 订阅管理 → 前端推送
3. **地理位置流**：GPS → 前端 → 位置服务 → PostGIS → 空间查询

### 2.2 技术栈选择

#### 2.2.1 前端技术栈

**核心框架与库**：
- **框架**: Nuxt 4 + Vue 3
- **UI组件**: Nuxt UI v3 + Tailwind CSS
- **状态管理**: Pinia + pinia-plugin-persistedstate
- **类型检查**: TypeScript
- **工具库**: @vueuse/core, dayjs

**开发工具与集成**：
- **构建工具**: Vite (通过Nuxt集成)
- **样式处理**: Tailwind CSS + Sass
- **地图服务**: 高德地图 API
- **微信集成**: weixin-js-sdk
- **二维码**: qrcode
- **图标系统**: Nuxt UI v3 内置图标系统（Heroicons、Lucide、Tabler等）
- **包管理**: pnpm

**代码组织原则**：
- **自动导入**: Nuxt自动导入系统，支持组件、API和工具函数
- **按需加载**: Nuxt UI v3组件按需导入优化
- **分层架构**: Pages → Components → Composables → Services → Stores
- **职责分离**: 业务逻辑与UI组件分离
- **类型安全**: 完整的TypeScript类型定义和自动类型生成
- **响应式设计**: 移动优先的设计原则

**游戏化UI设计原则**：
- **技能栏系统**: 底部技能栏作为主要功能入口，所有按钮配备图标
- **弹窗式交互**: 复杂功能（如地图）采用弹窗模式，避免固定位置占用空间
- **功能整合**: 移除重复功能模块，如"探索附近"整合到地图弹窗中
- **游戏化体验**: 借鉴游戏UI设计，提供直观的视觉反馈和交互体验

#### 2.2.2 后端技术栈

**核心框架与服务**：
- **Web框架**: Rust + Axum (从Actix-web迁移至Axum)
- **数据库**: PostgreSQL + PostGIS (地理位置扩展)
- **ORM**: SQLx + 自定义Repository模式
- **缓存**: Redis + 连接池管理
- **认证**: JWT令牌验证 (jsonwebtoken)
- **WebSocket**: Axum内置WebSocket支持 (tokio-tungstenite)
- **密码哈希**: Argon2（用户密码存储）+ SHA-256（动态认证哈希）
- **加密库**: sha2 crate（SHA-256 计算）、rand crate（nonce 生成）
- **任务队列**: RabbitMQ客户端 (lapin)
- **监控**: Prometheus指标

**依赖管理**：
- **异步运行时**: Tokio 完整功能
- **HTTP客户端**: 集成在Axum生态系统中
- **CORS**: tower-http CORS中间件
- **日志**: tracing + tracing-subscriber
- **文档**: utoipa 自动生成OpenAPI文档
- **错误处理**: thiserror

**API响应规范**：
- **统一HTTP状态**: 所有API响应统一返回HTTP 200状态码
- **业务状态码**: 通过响应体中的`code`字段标识业务处理结果
  - `code: 0` - 操作成功
  - `code: 1000-1999` - 客户端请求错误
  - `code: 5000-5999` - 服务器内部错误
- **错误处理**: 统一的错误响应格式和前后端错误码映射

#### 2.2.3 开发工具与环境

**前端开发环境**：
- **开发服务器**: Nuxt Dev Server (支持HMR)
- **类型检查**: Vue TypeScript支持和自动类型生成
- **代码质量**: ESLint + Prettier
- **依赖分析**: dependency-cruiser, ts-prune

**后端开发环境**：
- **编译器**: Rust 1.75+
- **代码质量**: cargo clippy (零警告策略)
- **测试框架**: Rust内置测试 + 集成测试
- **性能监控**: 内置健康检查和指标收集

### 2.3 路由架构设计

#### 2.3.1 路由分层策略

**公开路由**（无需认证）:
- **认证相关**: `/api/v1/auth/*` - 处理用户登录、注册、临时用户创建、token操作等
- **系统状态**: `/health`, `/metrics` - 系统健康检查和监控指标
- **WebSocket连接**: `/ws/` - 实时通信连接
- **查询类API**: 所有搜索、查看、获取类API均为公开访问，提升用户体验
  - **用户查询**: `/api/v1/user/search_*`, `/api/v1/user/get_profile`, `/api/v1/user/find_nearby`
  - **群组查询**: `/api/v1/group/search_*`, `/api/v1/group/members`
  - **生活瞬间查询**: `/api/v1/moments/search_*`, `/api/v1/moments/history`
  - **全局搜索**: `/api/v1/search/*`
  - **活动查询**: `/api/v1/activities/*`
  - **评论查询**: `/api/v1/comments/moment/*`, `/api/v1/comments/user`

**受保护路由**（需要JWT认证）:
- **用户管理**: `/api/v1/user/update_*`, `/api/v1/user/delete_*`, `/api/v1/user/upgrade_*` - 用户数据修改操作
- **群组管理**: `/api/v1/group/create`, `/api/v1/group/join`, `/api/v1/group/leave`, `/api/v1/group/kick` - 群组管理操作
- **消息通讯**: `/api/v1/chat/*` - 私聊、群聊、消息历史等（全部需认证）
- **生活瞬间管理**: `/api/v1/moments/create`, `/api/v1/moments/delete`, `/api/v1/moments/like` - 瞬间创建和互动
- **好友系统**: `/api/v1/friends/*` - 好友请求、管理等
- **评论管理**: `/api/v1/comments/create`, `/api/v1/comments/delete` - 评论创建和删除
- **文件上传**: `/api/v1/upload/*` - 文件上传功能
- **管理功能**: `/api/v1/admin/*` - 系统管理和监控功能（需管理员权限）

#### 2.3.2 命名规范

**API路径命名原则**:
- **语义化命名**: 采用 `动词_名词` 格式，如 `create_temp`, `search_users`, `get_profile`
- **功能导向**: 路径名称直接反映功能用途，避免技术术语
- **一致性**: 同类操作使用统一的动词前缀
- **可读性**: 避免缩写和歧义词汇，优先选择明确的英文词汇

路径重构示例（文字描述）：
- /api/v1/user/register → /api/v1/auth/register（认证逻辑分离）
- /api/v1/user/temp → /api/v1/auth/create_temp（语义化命名）
- /api/v1/user/search_by_id → /api/v1/user/get_profile（功能导向命名）
- /api/v1/user/upgrade → /api/v1/user/upgrade_account（完整语义）

#### 2.3.3 架构优势

**职责分离**: 认证逻辑与用户数据管理清晰分离
**安全性**: 公开和受保护路由明确区分，便于权限控制
**可维护性**: 语义化路径便于理解和维护
**可扩展性**: 模块化设计为未来功能扩展提供清晰架构

### 2.4 后端重构方案（端口/适配器 + 能力装配）

#### 核心理念
- 采用 Hexagonal（端口/适配器）架构：`domain`（纯领域）→ `application`（用例服务）→ `ports`（接口）→ `adapters`（实现）。
- 启动时通过 `CapabilityRegistry` 统一装配“通知通道、缓存、任务队列、策略”等能力。
- 编译期通过 feature 选择适配器，运行时通过配置控制行为。

#### 模块/目录结构（描述）
- `domain/`：领域模型、值对象、领域事件、错误（模块内 `thiserror`，统一转 `AppError`）。
- `application/`：用例服务（User/Group/Message/Moment）、通知策略聚合、CQRS 命令与查询模型。
- `ports/`：抽象接口（Repository、Cache、Dispatcher、NotificationTransport、Policy、GeoService 等）。
- `adapters/`：
  - `adapters-pg/`：PostgreSQL（仅用 sqlx 宏；NOT NULL/聚合/COALESCE/布尔表达式字段 SQL 必加 `!`；Option 极简）。
  - `adapters-cache-memory/`（默认）、`adapters-cache-redis/`（可选）。
  - `adapters-jobs-lite/`（默认）、`adapters-jobs-redis/`（可选）。
  - `adapters-notify-ws/`（默认唯一实时通道；SSE/轮询仅预留端口，暂不实现）。
- `presentation/`：
  - `http/`：controllers/routes/validators/middleware。
  - `ws/`：连接/房间/消息路由/消息类型/错误；禁止全局广播，遵循“策略选人→定向推送”。
- `bootstrap/`：`AppState` 组装、依赖注入、配置加载、能力注册、路由挂载（最终 bin）。

#### Feature 与默认能力
- 默认：`ws`, `jobs-lite`, `cache_in_memory`, `policy_friend`, `policy_group`, `metrics_basic`。
- 可选：`policy_nearby`、`cache_redis`、`jobs-redis`、`prometheus_metrics`、`email_channel`。
- 说明：本阶段仅实现 WebSocket；SSE/长轮询/Web Push 为规划，未实现（详见 4.4.4）。

#### 数据访问与分页规范（强制）
- 仅用 `sqlx::query!|query_as!|query_scalar!`；NOT NULL/聚合/COALESCE/布尔表达式字段 SQL 一律加 `!`。
- Option 极简：仅数据库真实可 NULL 使用 `Option<T>`。
- 统一游标分页：游标与主键 `id` 比较，请求 `limit+1` 判断 `has_more`，返回最后一条 `id` 作为 `next_cursor`。

#### 实时通道与策略
- 当前仅 `WebSocketTransport`：握手放行，首条消息鉴权（3–5s 超时），心跳/空闲超时，粘性会话或外部状态确保订阅稳定。
- `NotificationPolicyService`：好友/群组（默认），附近（可选），自动去重后定向发送。

#### 任务队列与缓存
- 默认 `jobs-lite` 与 `cache_in_memory`；失败不阻断主路径。
- 可选 `jobs-redis` 与 `cache_redis` 替换注入。

#### 密码安全实现架构
- **认证服务层**：
  - `AuthService`：处理 nonce 生成、密码哈希验证、用户认证逻辑
  - `NonceManager`：管理 nonce 生命周期，基于 Redis 实现
  - `PasswordHasher`：封装 Argon2 和 SHA-256 哈希算法
- **数据层**：
  - `UserRepository`：用户密码存储和验证（Argon2 哈希）
  - `NonceRepository`：nonce 存储和验证（Redis TTL）
- **控制器层**：
  - `AuthController`：处理认证相关 HTTP 请求
  - 验证请求参数、调用认证服务、返回标准响应

#### 迁移步骤（低风险）
1. 抽象 `ports/`，保留原实现。
2. 迁移现有实现至 `adapters-*`，`application` 仅依赖 `ports`。
3. 引入 `CapabilityRegistry` 与强类型 `AppState` 装配（非 trait object，优先泛型 + 类型别名）。
4. 统一收敛 sqlx 宏/"!"/游标分页规范。
5. **实现密码安全机制**：添加 nonce 管理、SHA-256 哈希验证、防重放攻击。
6. 按需开启可选适配器；SSE/轮询仅保留端口不实现。

## 3. 功能需求

### 3.1 用户管理功能

#### 3.1.1 功能需求

**用户注册与认证**:
- 支持邮箱/手机号注册
- 临时用户快速访问
- JWT令牌认证
- 密码安全策略

**用户档案管理**:
- 基本信息维护
- 头像上传
- 位置偏好设置
- 隐私控制

#### 3.1.2 认证系统API

**认证API（7个端点）**：
- `/api/v1/auth/register` - 用户注册
- `/api/v1/auth/login` - 用户登录
- `/api/v1/auth/create_temp` - 创建临时用户
- `/api/v1/auth/refresh_token` - 刷新Token
- `/api/v1/auth/check_status` - 检查认证状态
- `/api/v1/auth/set_password` - 设置密码
- `/api/v1/auth/password_status` - 检查密码状态

#### 3.1.3 用户数据API

**用户数据管理API（9个端点）**：

**查询类接口（公开访问，无需认证）**：
- `/api/v1/public/user/search_by_name` - 按昵称搜索用户
- `/api/v1/public/user/search_by_id` - 获取用户资料
- `/api/v1/public/user/search_by_location` - 搜索附近用户

**管理类接口（需要认证：Authorization: Bearer <jwt>）**：
- `/api/v1/protected/user/update_location` - 更新用户位置
- `/api/v1/protected/user/update_profile` - 更新用户资料
- `/api/v1/protected/user/upgrade_account` - 账户升级
- `/api/v1/protected/user/get_settings` - 获取用户设置
- `/api/v1/protected/user/update_settings` - 更新用户设置
- `/api/v1/protected/user/delete_account` - 删除账户

### 3.2 地理位置服务

#### 3.2.1 功能需求

**位置获取与处理**:
- GPS位置获取
- 地址反解析
- 位置缓存策略
- 离线位置支持

**基于位置的搜索**:
- 附近用户发现
- 地理围栏功能
- 距离计算优化
- 位置隐私保护

#### 3.2.2 地图显示功能

- 交互式地图界面
- 用户位置标记
- 群组位置展示
- moment热力图

#### 3.2.3 地理位置计算实现

使用PostGIS扩展提供高性能地理位置计算：
- 距离计算：`ST_Distance(geom1, geom2)`
- 范围查询：`ST_DWithin(geom, point, radius)`
- 空间索引：GiST索引优化查询性能

#### 3.2.4 地理位置数据处理原则

- **精度控制**: 适当降低精度保护隐私
- **缓存策略**: 合理缓存位置数据减少计算
- **索引优化**: 使用空间索引提升查询效率
- **数据同步**: 位置更新的实时同步机制

### 3.3 社交功能

#### 3.3.1 功能需求

**群组管理**:
- 群组创建与配置
- 成员角色管理
- 群组搜索与加入
- 群组活动管理

**社交互动**:
- 好友系统（预留）
- 生活瞬间分享
- 内容点赞评论
- 社交分享

#### 3.3.2 API 接口

**群组管理API（9个端点）**：

**查询类接口（公开访问，无需认证）**：
- `/api/v1/public/group/search_by_location` - 按位置搜索群组
- `/api/v1/public/group/search_by_name` - 按名称搜索群组
- `/api/v1/public/group/search_by_id` - 按ID获取群组详情
- `/api/v1/public/group/members` - 获取群组成员

**管理类接口（需要认证：Authorization: Bearer <jwt>）**：
- `/api/v1/protected/group/create` - 创建群组
- `/api/v1/protected/group/join` - 加入群组
- `/api/v1/protected/group/leave` - 离开群组
- `/api/v1/protected/group/kick` - 踢出群组成员
- `/api/v1/protected/group/transfer_ownership` - 转移群组所有权

### 3.4 消息通讯系统

#### 3.4.1 功能需求

**实时消息**:
- 一对一私聊
- 群组聊天
- 消息状态跟踪
- 离线消息同步

**消息增强**:
- 富文本支持
- 图片分享
- 位置分享
- 消息搜索

#### 3.4.2 API 接口

**消息通讯API（需要认证：Authorization: Bearer <jwt>，前缀：/api/v1/protected/chat）**：
- `/api/v1/protected/chat/group/send` - 发送群组消息
- `/api/v1/protected/chat/group/history` - 获取群组消息历史
- `/api/v1/protected/chat/group/delete` - 删除群组消息
- `/api/v1/protected/chat/group/mark-read` - 标记群组消息已读
- `/api/v1/protected/chat/group/unread_count` - 获取群组未读消息数
- `/api/v1/protected/chat/user/send` - 发送私聊消息
- `/api/v1/protected/chat/user/history` - 获取私聊消息历史
- `/api/v1/protected/chat/user/conversations` - 获取会话列表
- `/api/v1/protected/chat/user/delete` - 删除私聊消息
- `/api/v1/protected/chat/user/mark-read` - 标记私聊消息已读
- `/api/v1/protected/chat/user/unread_count` - 获取私聊未读消息数

**WebSocket实时通信**：
- **连接端点**: `/ws/` - WebSocket连接入口
- **消息类型**: 支持20+种消息类型（用户状态、群组事件、消息通知、moment更新等）
- **房间管理**: 自动订阅/取消订阅相关房间
- **实时通知**: 群组消息、用户状态、系统通知等实时推送

## 4. 代码质量管理

### 4.1 代码质量标准

#### 4.1.1 官方文档优先原则（强制）

**核心原则**：
- **文档优先验证**：使用任何第三方库、框架、组件前，必须先查阅官方文档确认正确用法
- **零假设开发**：不得基于经验或猜测使用API，所有用法必须有官方文档支撑
- **实时验证要求**：在编写代码时立即验证API的存在性和正确性

**前端开发强制要求**：
- **Nuxt UI v3 组件**：使用任何 UButton、UModal、UPopover、UDropdownMenu 等组件前，必须到 ui.nuxt.com 查阅官方文档
  - 验证组件名称是否存在
  - 确认所有 Props、Emits、Slots 的正确用法
  - 理解组件的设计意图和最佳实践
  - 特别注意插槽命名（如 UPopover 使用 `#content` 而非 `#panel`）
- **Nuxt 框架 API**：使用 definePageMeta、useRoute、navigateTo、useFetch 等 API 前，必须验证最新用法
- **Vue 3 API**：确认 Composition API、响应式 API 的正确使用方式
- **第三方库**：任何 npm 包的使用都必须基于官方文档或 README

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
- 前端：ui.nuxt.com、nuxt.com、vuejs.org、developer.mozilla.org
- 后端：docs.rs、doc.rust-lang.org、postgresql.org、postgis.net
- 通用：GitHub README、官方示例代码、Stack Overflow（仅作参考）

#### 4.1.2 编译质量要求
- **零警告政策**：项目必须保持 `cargo check` 无警告状态
- **编译速度**：后端项目编译时间应控制在1秒以内（开发环境）
- **代码规范**：使用 `cargo clippy` 进行代码质量检查

#### 4.1.3 后端代码合理性检查规范

**架构合理性验证原则**：
- **真实世界合理性**：定期检查后端设计是否符合真实世界的业务逻辑和使用场景
- **数据存储合理性**：验证数据库表和存储策略是否必要，避免过度设计
- **功能重复性检查**：识别和清理功能重复或不必要的代码模块

**已执行的合理性优化示例（描述）**：
- 移除通知数据库持久化（user_notifications + notification_repository.rs），采用纯 WebSocket 实时通知

**代码审查要点**：
1. **数据库表必要性**：每个表都应该有明确的业务价值，避免"可能有用"的表
2. **Repository层精简**：只为真正需要的数据操作创建repository方法
3. **服务层职责单一**：避免同一功能的多种实现方式并存
4. **API设计简洁**：API应该反映真实的用户需求，而非技术便利

**定期审查机制**：
- **月度架构审查**：每月审查一次后端架构合理性
- **功能下线检查**：当业务需求变化时，及时清理过时的代码
- **性能导向重构**：基于实际使用数据优化代码结构

#### 4.1.4 死代码处理原则
- **架构保留原则**：对于未来有价值的代码架构，使用 `#[allow(dead_code)]` 注解标记而不删除
- **分类标记策略**：
  - 完整的功能模块（如 WebSocket 系统）：在模块级别添加注解
  - 单独的函数/结构体：在具体位置添加注解
  - 临时未使用代码：添加 TODO 注释说明预期使用时间
- **定期审查**：每个版本发布前审查已标记的死代码，决定激活或清理

### 4.2 项目清理规范

#### 4.2.1 Agent模式代码生成清理要求

**清理原则**：
- **中间文件零容忍**：Agent模式生成代码完成后，必须彻底清理所有过程中产生的中间文件、总结文档和临时代码
- **需求文档统一**：所有新增的需求或设计内容必须合并到主需求文档（`hotspot.md`）中，不得保留分散的Agent总结文档
- **即时清理执行**：功能开发完成后立即执行清理，确保项目结构始终保持整洁

**严禁保留的文件类型**：
- **Agent总结文档**：AUTO_USER_REGISTRATION_README.md、*COMPLETION*.md、*VALIDATION*.md、*IMPROVEMENT*.md等
- **中间测试文件**：test_completion.rs、auto-user-demo.vue等验证性临时文件
- **占位服务文件**：comment.service.ts等仅含TODO注释的服务文件
- **重构残留目录**：backend-go/等重构过程中遗留的旧技术栈目录
- **临时调试工具**：debug-*.html、*test*.html等临时调试页面

#### 4.2.2 清理执行标准

**验证要求**：
1. **编译检查**：前端执行 `npm run build`，后端执行 `cargo check`，确保无编译错误
2. **引用完整性**：检查删除文件的所有引用，妥善处理引用错误（如改为"功能暂未开放"提示）
3. **功能完整性**：确保核心业务功能不受清理过程影响
4. **文档整合**：将有价值的设计内容迁移到主需求文档，保持信息完整性

**清理流程**：
1. **识别临时文件**：扫描项目目录，识别所有Agent模式产生的临时文件
2. **内容价值评估**：评估文件内容是否需要迁移到主文档
3. **安全删除**：验证文件无重要引用后执行删除
4. **构建验证**：清理后验证前后端构建成功

#### 4.2.3 长期维护原则

**持续整洁**：
- **一次性彻底清理**：每次Agent模式完成后进行彻底的一次性清理
- **预防性管理**：日常开发中避免创建不必要的临时文件
- **定期审查**：定期检查项目结构，防止新的临时文件累积
- **文档集中管理**：所有需求、设计、架构信息统一在主需求文档中维护

### 4.3 认证和Token存储调试规范

#### 4.3.1 Token存储故障诊断

**问题场景**：API请求成功但Token存储失败，导致前端认为没有Token而认证状态异常。

**调试策略**：
- **分步验证法**：将认证流程分解为 API调用 → Store设置 → localStorage存储 → 页面刷新验证 四个独立步骤
- **多层日志记录**：在每个关键步骤添加详细的控制台日志，包括API响应内容、Store状态变化、持久化结果
- **备份存储机制**：在重要的认证数据设置时，同时使用backup localStorage存储作为调试参考
- **Native HTML验证**：使用原生HTML页面验证localStorage行为，排除框架干扰

#### 4.3.2 调试工具要求

**系统性测试页面**：
- **简单认证测试页面** (`/simple-auth-test`) - 按步骤验证认证流程的每个环节
- **Token测试页面** (`/token-test`) - 详细测试Token存储和读取机制
- **原生HTML调试工具** (`/debug-token.html`) - 无框架依赖的localStorage调试工具

**调试日志标准（描述）**：在认证相关函数中应输出：API响应、Store状态变更前后、LocalStorage备份键值

#### 4.3.3 Pinia持久化调试

**配置要求**：
- 使用官方 `pinia-plugin-persistedstate` 插件，避免自定义持久化冲突
- 简化持久化配置为 `persist: true`，避免复杂的序列化逻辑
- 启用调试模式：`piniaPluginPersistedstate({ debug: true })`

**日期对象处理**：
- 避免在需要持久化的Store中直接存储Date对象
- 使用ISO字符串格式存储时间相关数据
- 在读取时进行适当的类型转换

#### 4.3.4 故障排查流程

1. **API响应验证**：确认后端API返回正确的认证数据
2. **Store设置检查**：验证Pinia store中的数据设置是否成功
3. **持久化验证**：检查localStorage中是否正确存储了认证数据
4. **页面刷新测试**：验证页面刷新后认证状态是否正确恢复
5. **框架无关验证**：使用原生HTML工具验证localStorage基本功能

### 4.4 架构保留策略

#### 4.4.1 WebSocket实时通信系统（已完整实现并激活）

**当前状态：100%完成并在生产环境中使用**

WebSocket系统已完全实现并集成到项目中，包含以下核心组件：

**连接管理层**：
- **WebSocketManager** (`ws/manager.rs`) - 完整的WebSocket连接生命周期管理
- **ConnectionManager** (`ws/connection_manager.rs`) - 用户连接映射和状态管理
- **RoomManager** (`ws/room_manager.rs`) - 群组和私聊房间订阅管理
- **WebSocket处理器** (`ws/handler.rs`) - 连接建立、认证和消息处理

**消息分发层**：
- **MessageRouter** (`ws/message_router.rs`) - 智能消息路由，支持点对点、群组和广播
- **消息类型系统** (`ws/message_types.rs`) - 完整的消息格式定义和类型安全
- **错误处理** (`ws/errors.rs`) - WebSocket专用错误类型和处理机制

**业务通知层**：
- **WebSocketNotificationService** (`services/notification/websocket_notification.rs`) - 业务通知发送
- **NotificationManager** (`services/notification/manager.rs`) - 通知策略统一管理
 - **策略服务** (`services/notification/policy.rs`) - 智能通知策略（避免全局广播）

**前端集成（当前代码）**：
- 组合式函数：`frontend/app/composables/useWebSocket.ts`
- 实时消息：`frontend/app/composables/useRealTimeMessages.ts`
- 消息状态：`frontend/app/stores/messages.ts`
- 连接状态组件：`frontend/app/components/RealtimeStatus.vue`

**实时功能覆盖**：
- ✅ 私聊消息实时推送
- ✅ 群组消息实时分发
- ✅ 用户在线状态同步
- ✅ 群组事件通知（加入/退出/角色变更）
- ✅ 新生活瞬间通知
- ✅ 系统通知推送
- ✅ 打字指示器
- ✅ 消息已读回执
- ✅ 连接状态管理和自动重连

#### 4.4.2 保留的扩展架构

**缓存优化层**（部分实现，预留扩展）：
- 缓存抽象层：`cache/` - 为高频访问数据提供缓存接口
- 性能监控：`utils/metrics.rs` - 系统性能指标收集
- 任务调度：`utils/scheduler.rs` - 后台任务管理

**标记策略**：
- 使用 `#[allow(dead_code)]` 注解保护未激活的扩展模块
- 为未来性能优化和监控功能保留架构空间

#### 4.4.3 维护策略
- **WebSocket系统**：已激活，需要定期性能监控和连接池优化
- **扩展模块**：定期验证编译完整性，保持依赖包版本更新
- **文档同步**：保持架构文档与实际代码实现状态同步


#### 4.4.4 实时通道策略（当前与规划）

当前状态（2025-08）：
- 仅启用并支持 WebSocket 实时通道；生产环境必须启用 WS，作为唯一实时通知与消息分发方式。
- SSE（Server‑Sent Events）、长轮询（Long Polling）、Web Push 等替代/补充通道暂未实现，计划在后续迭代中补充，作为在特定网络或平台限制下的降级方案。

规划与原则（未实现，后续迭代）：
- 优先级：WebSocket > SSE > Long Polling；Web Push 仅用于离线/回流提醒，不承担高频聊天。
- 策略一致性：无论采用何种通道，均遵循“策略服务选人 → 精准推送 → 禁止全局广播”的架构原则。
- 运行时策略：默认启用 WS；当环境确实无法建立/维持 WS 时，前端与后端将按优先级自动降级（待实现）。
- 状态与指标：各通道需暴露连接数、心跳、降级次数、队列长度等指标（待实现）。

注意：
- 本阶段不引入 SSE/轮询 的服务端与路由实现；文档仅记录架构规划，后续按版本里程碑逐步落地。

## 5. 测试架构设计

### 5.1 后端测试架构

#### 5.1.1 当前测试状态与改进方向

**当前测试实现状态**：
项目已建立了完整的测试基础设施，包含：

**已实现的测试组件**：
- **集成测试框架**：`/backend/src/tests/integration_tests/` - 已有完整的用户服务集成测试
- **测试工具**：用户创建、数据清理、数据库连接等测试辅助函数
- **API测试页面**：`api-response-test.html`, `test-frontend-api.html` 等完整的前端测试套件
- **错误处理测试**：`test-error-scenarios.html`, `frontend-error-test.html` 专项错误场景测试
- **自动化测试脚本**：多个 `test-*.sh` 脚本支持自动化测试执行

**测试覆盖现状**：
- ✅ **用户认证模块**：完整的密码设置、验证、权限检查测试
- ✅ **API接口层**：所有主要API端点都有对应的测试用例
- ✅ **错误处理机制**：前后端错误处理链路的完整测试
- ✅ **WebSocket通信**：实时通信功能的基础测试框架
- ⚠️ **地理位置计算**：基础功能完成，测试覆盖待补强
- ⚠️ **群组权限管理**：核心逻辑完成，边界情况测试不足

**改进优先级**：

**1. 测试覆盖率提升**（短期1-2周）

当前状态：核心功能测试已覆盖，但覆盖率有提升空间
改进方向：
- 补充地理位置计算模块的单元测试，重点覆盖PostGIS查询和距离计算算法
- 增强群组权限管理的边界测试，包括角色变更、并发加入等场景
- 目标从当前覆盖率提升至80%

**2. 性能测试增强**（中期2-4周）

当前状态：基础性能监控已部署，压力测试框架需要加强
改进方向：
- 建立WebSocket连接的压力测试，模拟高并发聊天场景
- 地理位置查询的性能基准测试，优化PostGIS查询效率
- 引入监控指标和性能回归检测

**3. 端到端测试扩展**（中长期1-2个月）

当前状态：前端测试页面功能完整，但自动化程度不够
改进方向：
- 将现有的手动测试页面转换为自动化E2E测试
- 集成Playwright或Cypress等现代测试框架
- 建立CI/CD流水线中的自动化测试

#### 5.1.2 测试分层架构（文字描述）
- E2E 测试：基于现有测试页面扩展到完整端到端覆盖
- 集成测试：模块间协作与 API/数据层联调（已覆盖用户服务与部分端点）
- 单元测试：地理计算、权限管理等核心规则需补强

**推荐测试工具链**：
- **单元测试**: Rust内置测试框架 + mockall crate
- **集成测试**: 自定义测试数据库 + axum-test
- **E2E测试**: 基于现有测试页面升级至Playwright

## 6. 数据库设计

### 6.1 数据库架构概览

#### 6.1.1 核心设计原则

**地理位置优先**：
- 基于PostGIS扩展，所有位置数据使用WGS84坐标系（SRID: 4326）
- 自动触发器管理geometry列，保证数据一致性
- 空间索引优化地理查询性能

**用户体验导向**：
- 支持临时用户和注册用户的双轨制设计
- 过期机制自动清理临时数据
- 软删除策略保护重要数据

#### 6.1.2 数据库扩展

（扩展与触发器：仅保留结构与类型说明，示例 SQL 省略）

### 6.2 核心数据表设计

#### 6.2.1 用户系统表

**users表** - 用户基础信息
（users 表结构：字段与类型说明，示例 SQL 省略；含基础信息、索引与时间戳字段）

**user_locations表** - 用户位置信息
（表结构字段与类型说明）
CREATE TABLE user_locations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    longitude DOUBLE PRECISION NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    geom GEOGRAPHY(POINT, 4326),                 -- PostGIS地理列
    address TEXT,                                -- 地址描述
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 自动更新地理列的触发器
（地理列触发器与空间索引：保留说明，示例 SQL 省略）

#### 6.2.2 群组系统表

**groups表** - 群组基础信息（字段与类型说明，示例 SQL 省略；包含经纬度/地理索引、成员容量、公开性与时间戳字段）

**group_members表** - 群组成员关系（字段与类型说明，示例 SQL 省略；包含成员角色、入群时间、唯一约束与常用索引）

#### 6.2.3 消息系统表

**conversations表** - 会话管理（字段与类型说明，示例 SQL 省略；包含 UUID 会话ID、双用户去重、最近消息时间与索引）

**group_messages表** - 群组消息（字段与类型说明，示例 SQL 省略；包含 UUID 消息ID、地理列、触发器与常用索引）

**direct_messages表** - 私聊消息（字段与类型说明，示例 SQL 省略；包含会话ID、经纬度/地理列、已读时间与常用索引）

#### 6.2.4 生活瞬间系统表

**moments表** - 生活瞬间记录（字段与类型说明，示例 SQL 省略；包含经纬度/地理列、标签与点赞的关联表、常用索引与时间戳）

**tags表** - 标签系统（字段与类型说明，示例 SQL 省略；包含 UUID 标签ID、唯一名称与计数字段及索引）

**moment_tags表** - 生活瞬间标签关联（字段与类型说明，示例 SQL 省略；包含唯一约束与常用索引）

**moment_likes表** - 生活瞬间点赞（字段与类型说明，示例 SQL 省略；包含唯一约束与常用索引）

### 6.3 数据库性能优化

#### 6.3.1 地理位置查询优化

**空间索引策略**（文字描述）：使用 GIST 地理索引优化距离与范围查询，必要处增加复合索引（如 user_id+updated_at、user_id+created_at）以支撑常见过滤排序场景。

**常用地理查询模式**（文字描述）：附近用户/瞬间查询采用 ST_DWithin 判断距离范围并按距离升序返回，统一使用 geography 类型与 WGS84 坐标，避免平面距离误差。

#### 6.3.2 数据清理策略

**临时数据清理**：
- 定期清理过期的临时用户数据
- 级联删除确保数据一致性
- 日志保留30天后自动清理

**分区策略** (未来扩展)：
- 按时间分区消息表，提高查询性能
- 按地理区域分区位置数据，优化空间查询

## 7. API接口设计

### 7.1 API设计原则

#### 7.1.1 统一响应格式（与当前实现保持一致）

**标准响应结构**：
```typescript
interface ApiResponse<T> {
  code: number;    // 业务状态码：0-成功，非0-错误
  msg: string;     // 响应消息（字段名为 msg）
  data: T | null;  // 成功时为具体业务数据；失败时通常为 null 或省略
}
```

**HTTP状态码策略**：
- 所有API请求统一返回HTTP 200状态码
- 通过响应体中的`code`字段区分业务处理结果
- 便于前端统一处理和错误管理

**错误码分组与含义**：
- 1000 参数错误（BadRequest）
- 1002 未授权（Unauthorized）
- 1003 权限不足（Forbidden）
- 1004 资源不存在（NotFound）
- 1005 资源冲突（Conflict）
- 1006 功能未实现（NotImplemented）
- 5000 服务器内部错误（InternalServerError）
- 5001 数据库错误（DatabaseError）
- 5002 外部服务错误（ExternalServiceError）

#### 7.1.2 API 类型定义规范

**类型定义原则**：
- 所有 API 端点必须有明确的请求类型（Request）和响应类型（Response）定义
- 使用 TypeScript 接口定义，包含详细的字段注释和验证规则
- 请求类型以 `Request` 结尾，响应类型以 `Response` 结尾或直接使用公共模型
- 所有类型定义与前端 `types/index.ts` 保持完全一致

**安全设计原则**：
- **敏感信息保护**：API 响应中不得包含用户登录凭据（邮箱/手机号/密码）
- **最小权限原则**：响应只包含业务必需的字段，避免过度暴露用户信息
- **传输安全**：所有包含敏感信息的请求必须通过 HTTPS 传输
- **位置隐私**：地理位置信息应适当降低精度以保护用户隐私
- **密码安全机制**：
  - **禁止明文传输**：API 请求中严禁出现明文密码
  - **动态哈希认证**：使用 SHA-256 + 动态盐值的哈希机制
  - **防重放攻击**：每次认证使用服务端返回的随机 nonce 和时间戳
  - **哈希算法**：`hash = SHA256(password + nonce + timestamp + login_id)`
  - **有效期控制**：nonce 有效期为 5 分钟，防止长时间重放攻击
  - **前端实现**：使用 Web Crypto API 或 crypto-js 库进行 SHA-256 计算
  - **认证流程**：
    1. 调用 `get_nonce` 获取 nonce_id、nonce 和 timestamp
    2. 前端计算 `SHA256(password + nonce + timestamp + login_id)`
    3. 发送哈希值、nonce_id、timestamp 到认证接口（不发送 nonce）
    4. 服务端通过 nonce_id 从 Redis 获取 nonce，重新计算哈希进行比对
  - **后端实现要求**：
    - **Nonce 管理**：使用 Redis 存储 nonce，key 为 `auth_nonce:{login_id}:{nonce_id}`，TTL 为 5 分钟
    - **多设备支持**：每个 nonce 有唯一 ID，支持同一用户多设备同时获取不同 nonce
    - **哈希验证**：服务端重新计算 `SHA256(stored_password + nonce + timestamp + login_id)` 与请求哈希比对
    - **时间戳验证**：验证 timestamp 在 nonce 有效期内且不超过服务器时间 ±30 秒
    - **密码存储**：用户密码使用 Argon2 算法存储，不存储明文或简单哈希
    - **防重放机制**：同一 nonce 只能使用一次，使用后立即从 Redis 删除
- **文件上传安全**：
  - 严格限制文件类型（仅允许图片格式：JPG、PNG、GIF、WebP）
  - 限制文件大小（单文件≤10MB，总大小≤100MB）
  - 文件名随机化，避免路径遍历攻击
  - 图片自动压缩和格式标准化
  - 恶意文件扫描和内容验证

#### 7.1.3 公共数据模型（API 与 WebSocket 复用）

**类型转换规范**：
由于 JavaScript 无法安全处理大整数和高精度浮点数，以下类型在 API 传输层需要特殊处理：
- **64位整数 (`i64`)**: 转换为 `string` 类型
  - **需要转换的字段**：`user_id`、`id`（数据库主键）、`cursor`/`next_cursor`（分页游标）
  - **原因**：这些字段可能超出 JavaScript 安全整数范围（2^53-1）
- **32位整数 (`i32`)**: 转换为 `number` 类型
  - **包含字段**：`member_count`、`likes_count`、`comments_count`、`unread_count`、`code`（错误码）等
  - **原因**：32位整数在 JavaScript 安全范围内，可直接使用 number
- **64位浮点数 (`f64`)**: 转换为 `number` 类型，但注意精度限制
  - **包含字段**：`latitude`、`longitude`、`distance`、`radius`（地理坐标和距离）

后端内部使用原始类型存储，但在 JSON 序列化时进行适当转换。

**核心数据模型**：

```typescript
// 认证响应
interface AuthResponse {
  user_id: string;          // 64位整数转字符串
  nickname: string;         // 用户昵称
  access_token: string;     // 访问令牌
  refresh_token: string;    // 刷新令牌
  expires_at: string;       // 过期时间（ISO 8601）
  is_temp?: boolean;        // 是否临时用户
}

// 用户信息
interface User {
  user_id: string;          // 64位整数转字符串
  nickname: string;         // 用户昵称
  avatar_url?: string;      // 头像URL
  bio?: string;             // 个人简介
  is_temp?: boolean;        // 是否临时用户
  created_at?: string;      // 创建时间（ISO 8601）
  last_active_at?: string;  // 最后活跃时间（ISO 8601）
  latitude?: number;        // 纬度
  longitude?: number;       // 经度
  location_name?: string;   // 位置名称
  distance?: number;        // 距离（米）
}

// 群组信息
interface Group {
  group_id: string;         // UUID转字符串
  name: string;             // 群组名称
  description?: string;     // 群组描述
  owner_id: string;         // 群主ID（64位整数转字符串）
  creator_name: string;     // 创建者昵称
  created_at: string;       // 创建时间（ISO 8601）
  last_active_at: string;   // 最后活跃时间（ISO 8601）
  latitude: number;         // 纬度
  longitude: number;        // 经度
  member_count: number;     // 成员数量（32位整数）
  distance?: number;        // 距离（米）
  location_name: string;    // 位置名称
  is_password_required: boolean; // 是否需要密码
  is_member: boolean;       // 当前用户是否为成员
  user_role: string;        // 当前用户角色
  avatar_url?: string;      // 群组头像URL
}

// 消息信息
interface Message {
  message_id: string;       // UUID转字符串
  sender_id: string;        // 发送者ID（64位整数转字符串）
  sender_name: string;      // 发送者昵称
  content: string;          // 消息内容
  message_type: MessageType; // 消息类型枚举
  sent_at: string;          // 发送时间（ISO 8601）
  group_id?: string;        // 群组ID（UUID转字符串）
  conversation_id?: string; // 会话ID（UUID转字符串）
  latitude?: number;        // 纬度
  longitude?: number;       // 经度
}

// 生活瞬间
interface Moment {
  moment_id: string;        // UUID转字符串
  user_id: string;          // 用户ID（64位整数转字符串）
  nickname: string;         // 用户昵称
  user_avatar?: string;     // 用户头像URL
  description: string;      // 瞬间描述
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name: string;    // 位置名称
  created_at: string;       // 创建时间（ISO 8601）
  likes_count: number;      // 点赞数（32位整数）
  comments_count: number;   // 评论数（32位整数）
  liked_by_me: boolean;     // 当前用户是否已点赞
  distance?: number;        // 距离（米）
  tags: string[];           // 标签数组
}

// 分页响应
interface PaginatedResponse<T> {
  items: T[];               // 数据项数组
  has_more: boolean;        // 是否有更多数据
  next_cursor?: string;     // 下一页游标（64位整数转字符串）
}

// 密码状态结果
interface PasswordStatusResult {
  has_password: boolean;    // 是否已设置密码
}

// 基础成功响应
interface BasicSuccess {
  success: boolean;         // 操作是否成功
}
```

**扩展公共模型**：

```typescript
// 地理坐标
interface Coordinates {
  latitude: number;         // 纬度，范围：[-90, 90]，精度可适当降低以保护隐私
  longitude: number;        // 经度，范围：[-180, 180]，精度可适当降低以保护隐私
}

// 请求分页参数
interface PaginationParams {
  cursor?: string;          // 游标分页，字符串类型
  limit?: number;           // 每页数量，范围：1-100，默认20
}

// 响应分页元信息
interface PaginationMeta {
  has_more: boolean;        // 是否有更多数据
  next_cursor?: string;     // 下一页游标（64位整数转字符串）
}

// 会话概要
interface ConversationSummary {
  conversation_id: string;  // 会话ID（UUID转字符串）
  peer_user_id: string;     // 对方用户ID（64位整数转字符串）
  peer_nickname: string;    // 对方昵称
  last_message_preview: string; // 最后消息预览
  last_message_at: string;  // 最后消息时间（ISO 8601）
  unread_count: number;     // 未读消息数（32位整数）
}

// 未读计数模型
interface UnreadCount {
  group_id?: string;        // 群组ID（UUID转字符串）
  conversation_id?: string; // 会话ID（UUID转字符串）
  unread_count?: number;    // 具体未读数（32位整数）
  total_unread_count?: number; // 汇总未读数（32位整数）
}

// 消息类型枚举
type MessageType = 'text' | 'image' | 'voice' | 'video' | 'location' | 'file';

// 房间类型枚举
type RoomType = 'group' | 'conversation';

// WebSocket 消息通知负载
interface WsMessageNotification {
  message_id: string;       // 消息ID（UUID转字符串）
  room_id: string;          // 房间ID（UUID转字符串）
  room_type: RoomType;      // 房间类型
  sender_id: string;        // 发送者ID（64位整数转字符串）
  sender_nickname: string;  // 发送者昵称
  content: string;          // 消息内容
  message_type: MessageType; // 消息类型
  created_at: string;       // 创建时间（ISO 8601）
  latitude?: number;        // 纬度
  longitude?: number;       // 经度
}

// 群组事件负载
interface GroupEvent {
  group_id: string;         // 群组ID（UUID转字符串）
  event_type: 'member_joined' | 'member_left' | 'member_kicked' | 'owner_transferred' | 'group_updated';
  actor_id: string;         // 操作者ID（64位整数转字符串）
  actor_nickname: string;   // 操作者昵称
  target_id?: string;       // 目标用户ID（64位整数转字符串）
  target_nickname?: string; // 目标用户昵称
  created_at: string;       // 事件时间（ISO 8601）
}

// 系统通知负载
interface SystemNotification {
  notification_id: string;  // 通知ID（UUID转字符串）
  notification_type: 'friend_request' | 'group_invitation' | 'system' | 'announcement' | 'new_moment' | 'group_digest';
  title: string;            // 通知标题
  content: string;          // 通知内容
  data?: object;            // 附加数据
  created_at: string;       // 创建时间（ISO 8601）
}
```

**WebSocket 专用错误码**：
- `INVALID_JSON` - 无效的 JSON 格式
- `INVALID_MESSAGE_TYPE` - 消息类型错误（握手阶段仅接受文本）
- `INVALID_AUTH_MESSAGE` - 首条消息非认证消息
- `INVALID_TOKEN` - 无效认证令牌
- `UNAUTHORIZED` - 未授权访问
- `FORBIDDEN` - 权限不足/禁止访问
- `ROOM_NOT_FOUND` - 房间不存在
- `CONNECTION_LIMIT_EXCEEDED` - 连接数量超限
- `MESSAGE_TOO_LARGE` - 消息大小超限
- `INTERNAL_ERROR` - 服务器内部错误

### 7.2 认证与用户API

#### 7.2.1 认证接口（8个端点，前缀：/api/v1/auth）

- `POST /api/v1/auth/get_nonce` - 获取认证随机数（认证预处理）
  - 请求类型：GetNonceRequest
    ```typescript
    interface GetNonceRequest {
      login_id: string;    // 邮箱/手机号，用于生成专用nonce
    }
    ```
  - 认证：无需
  - 响应 data：NonceResponse
    ```typescript
    interface NonceResponse {
      nonce_id: string;    // nonce唯一标识符（UUID）
      nonce: string;       // 32位随机字符串
      timestamp: number;   // 服务器时间戳（毫秒）
      expires_at: number;  // nonce过期时间戳（毫秒）
    }
    ```

- `POST /api/v1/auth/register` - 用户注册
  - 请求类型：RegisterRequest
    ```typescript
    interface RegisterRequest {
      login_id: string;    // 3-32字符，字母/数字/下划线，唯一
      nickname: string;    // 2-32字符，去首尾空格
      password_hash: string; // SHA256(password + nonce + timestamp + login_id)
      nonce_id: string;    // 从 get_nonce 接口获取的nonce标识符
      timestamp: number;   // 从 get_nonce 接口获取的时间戳
    }
    ```
  - 认证：无需
  - 响应 data：AuthResponse

- `POST /api/v1/auth/login` - 用户登录
  - 请求类型：LoginRequest
    ```typescript
    interface LoginRequest {
      login_id: string;    // 邮箱/手机号，必填
      password_hash: string; // SHA256(password + nonce + timestamp + login_id)
      nonce_id: string;    // 从 get_nonce 接口获取的nonce标识符
      timestamp: number;   // 从 get_nonce 接口获取的时间戳
    }
    ```
  - 认证：无需
  - 响应 data：AuthResponse

- `POST /api/v1/auth/create_temp` - 创建临时用户
  - 请求类型：CreateTempUserRequest
    ```typescript
    interface CreateTempUserRequest {
      nickname?: string;   // 可选，2-32字符
    }
    ```
  - 认证：无需
  - 响应 data：AuthResponse

- `POST /api/v1/auth/refresh_token` - 刷新Token
  - 请求类型：RefreshTokenRequest
    ```typescript
    interface RefreshTokenRequest {
      refresh_token: string;  // 刷新令牌，必填
    }
    ```
  - 认证：无需
  - 响应 data：AuthResponse

- `POST /api/v1/auth/check_status` - 检查认证状态
  - 请求类型：无请求体
  - 认证：需要（Authorization: Bearer <jwt>)
  - 响应 data：AuthResponse

- `POST /api/v1/auth/set_password` - 设置密码
  - 请求类型：SetPasswordRequest
    ```typescript
    interface SetPasswordRequest {
      new_password_hash: string; // SHA256(new_password + nonce + timestamp + user_id)
      old_password_hash?: string; // SHA256(old_password + nonce + timestamp + user_id)，首次设置可为空
      nonce_id: string;        // 从 get_nonce 接口获取的nonce标识符
      timestamp: number;       // 从 get_nonce 接口获取的时间戳
    }
    ```
  - 认证：需要
  - 响应 data：BasicSuccess

- `POST /api/v1/auth/password_status` - 检查密码状态
  - 请求类型：无请求体
  - 认证：需要
  - 响应 data：PasswordStatusResult

字段校验与业务规则（认证）：
- /auth/get_nonce：
  - login_id 必填，用于生成用户专用的 nonce；
  - 每次请求生成新的 nonce_id 和 nonce，支持多设备同时获取；
  - 返回的 nonce 有效期为 5 分钟，超时需重新获取；
  - 同一用户可同时持有多个有效 nonce（不同 nonce_id）。
- /auth/register：
  - login_id 必填，3-32 个字符，允许字母/数字/下划线，需唯一（区分大小写）；
  - nickname 必填，2-32 个字符，去首尾空格，禁止仅空白；
  - password_hash 必填，使用 SHA256(password + nonce + timestamp + login_id) 算法；
  - nonce_id 和 timestamp 必须来自有效的 get_nonce 响应；
  - 服务端通过 nonce_id 从 Redis 获取对应的 nonce 进行验证；
  - 成功后立即返回 AuthResponse，并可异步派发位置初始化任务（不阻塞响应）。
- /auth/login：
  - login_id、password_hash 均必填；错误不区分具体原因，统一返回未授权；
  - password_hash 使用 SHA256(password + nonce + timestamp + login_id) 算法；
  - 服务端重新计算哈希值进行比对验证；
  - 多次失败可能触发风控（节流/限流）。
- /auth/create_temp：
  - nickname 可选，提供时 2-32 个字符；
  - 返回 is_temp=true 的 AuthResponse，权限受限（如部分创建操作限制）。
- /auth/refresh_token：
  - refresh_token 必填，需未过期且未被吊销；支持轮换策略；
  - 返回新的 access_token 与 refresh_token。
- /auth/check_status：
  - 需携带有效 Bearer Token；
  - 返回当前身份信息与 token 剩余有效期。
- /auth/set_password：
  - new_password_hash 必填，使用 SHA256(new_password + nonce + timestamp + user_id) 算法；
  - old_password_hash 可选，若已设置密码且要求校验，需提供；
  - nonce_id 和 timestamp 必须来自有效的 get_nonce 响应；
  - 服务端通过 nonce_id 从 Redis 获取对应的 nonce 进行验证；
  - 成功后建议使旧 access_token 失效并下发新 token（实现可选）。
- /auth/password_status：
  - 返回 has_password 用于前端引导设置/修改密码。

#### 7.2.2 用户数据接口（9个端点）

**查询类接口（公开访问，无需认证，前缀：/api/v1/public/user）**：
- `POST /api/v1/public/user/search_by_name` - 按昵称搜索用户
- `POST /api/v1/public/user/search_by_id` - 获取用户资料
- `POST /api/v1/public/user/search_by_location` - 按位置搜索附近用户

**管理类接口（需要认证：Authorization: Bearer <jwt>，前缀：/api/v1/protected/user）**：
- `POST /api/v1/protected/user/update_location` - 更新用户位置
- `POST /api/v1/protected/user/update_profile` - 更新用户资料
- `POST /api/v1/protected/user/upgrade_account` - 账户升级
- `POST /api/v1/protected/user/get_settings` - 获取用户设置
- `POST /api/v1/protected/user/update_settings` - 更新用户设置
- `POST /api/v1/protected/user/delete_account` - 删除账户

**请求类型定义：**

- **SearchUserByNameRequest**：
  ```typescript
  interface SearchUserByNameRequest {
    query: string;        // 搜索关键词，1-32字符
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **SearchUserByIdRequest**：
  ```typescript
  interface SearchUserByIdRequest {
    user_id: string;      // 用户ID（64位整数转字符串）
  }
  ```

- **SearchUserByLocationRequest**：
  ```typescript
  interface SearchUserByLocationRequest {
    latitude: number;     // 纬度，-90~90
    longitude: number;    // 经度，-180~180
    radius: number;       // 搜索半径，单位米，>0且≤100000
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **UpdateLocationRequest**：
  ```typescript
  interface UpdateLocationRequest {
    latitude: number;     // 纬度，-90~90
    longitude: number;    // 经度，-180~180
    location_name: string; // 位置名称，1-120字符
  }
  ```

- **UpdateProfileRequest**：
  ```typescript
  interface UpdateProfileRequest {
    nickname: string;     // 用户昵称，2-32字符
  }
  ```

- **UpgradeAccountRequest**：
  ```typescript
  interface UpgradeAccountRequest {
    login_id: string;     // 邮箱/手机号，3-32字符，唯一
    password_hash: string; // SHA256(password + nonce + timestamp + login_id)
    nonce_id: string;    // 从 get_nonce 接口获取的nonce标识符
    timestamp: number;   // 从 get_nonce 接口获取的时间戳
  }
  ```

- **UserSettingsRequest**：
  ```typescript
  interface UserSettingsRequest {
    notification_level?: 'all' | 'mention' | 'important' | 'muted';
    message_notifications?: boolean;
    group_notifications?: boolean;
    system_notifications?: boolean;
    location_notifications?: boolean;
    quiet_hours_enabled?: boolean;
    quiet_hours_start?: string;      // 格式：'HH:MM:SS'
    quiet_hours_end?: string;        // 格式：'HH:MM:SS'
    notification_radius_km?: number;
    sound_enabled?: boolean;
    vibration_enabled?: boolean;
  }
  ```

**响应类型定义：**

```typescript
// 搜索用户响应
interface SearchUserByNameResponse extends PaginatedResponse<User> {}
interface SearchUserByIdResponse extends User {}
interface SearchUserByLocationResponse extends PaginatedResponse<User> {}

// 用户操作响应
interface UpdateLocationResponse extends BasicSuccess {}
interface UpdateProfileResponse {
  nickname: string;         // 更新后的昵称
}
interface UpgradeAccountResponse extends AuthResponse {}

// 用户设置响应（仅返回给用户本人，不对外暴露）
interface UserSettingsResponse {
  notification_level: 'all' | 'mention' | 'important' | 'muted';
  message_notifications: boolean;
  group_notifications: boolean;
  system_notifications: boolean;
  location_notifications: boolean;
  quiet_hours_enabled: boolean;
  quiet_hours_start?: string;      // 格式：'HH:MM:SS'
  quiet_hours_end?: string;        // 格式：'HH:MM:SS'
  notification_radius_km: number;
  sound_enabled: boolean;
  vibration_enabled: boolean;
}
interface UpdateSettingsResponse extends UserSettingsResponse {}
interface DeleteAccountResponse extends BasicSuccess {}
```

字段校验与业务规则（用户）：
- /user/search_by_name：
  - query 必填，1-32 字符，模糊匹配昵称，统一大小写与空白处理；
  - 游标分页：limit 1-100，缺省 20；cursor 缺省 0。
- /user/search_by_id：
  - user_id 必填，为平台内用户号（API传输层为string，后端内部为i64）；
  - 若请求自身资料，可直接返回完整信息；他人资料按隐私策略裁剪（如隐藏邮箱）。
- /user/search_by_location：
  - latitude/longitude 必填，范围：纬度 [-90,90]，经度 [-180,180]；
  - radius 必填，>0 且 ≤ 100000（米）；
  - 返回结果按距离升序并采用游标分页；
  - 位置缺失的用户不返回，或以默认距离策略过滤。
- /user/update_location：
  - latitude/longitude 必填，范围同上；location_name 必填，1-120 字符；
  - 更新频率建议≥10 秒一次；
  - 后端可做去抖与最小移动距离阈值（如 10 米）以降低写入与通知风暴。
- /user/update_profile：
  - nickname 必填，2-32 字符；
  - 若为临时用户，允许更新昵称；正式用户同样允许，需去重策略视实现；
  - 成功后可通过 WebSocket 推送 UserProfileUpdated（如需）。
- /user/upgrade_account：
  - login_id 规则同注册；password_hash 使用 SHA256(password + nonce + timestamp + login_id) 算法；
  - nonce_id 和 timestamp 必须来自有效的 get_nonce 响应；
  - 服务端通过 nonce_id 从 Redis 获取对应的 nonce 进行验证；
  - 临时用户升级后 user_id 保持不变，数据应无缝迁移；
  - 若 login_id 已占用，返回冲突错误码。
- /user/get_settings & /user/update_settings：
  - boolean/enum 字段需做范围校验；时间格式严格为 HH:MM:SS；
  - 更新后返回最新设置；部分字段可按环境启用（如开发环境放宽限制）。
- /user/delete_account：
  - 需二次确认机制（前端 UI 与后端速率限制）；
  - 删除为不可逆操作，触发数据清理与会话失效；
  - 若存在未完成的任务（如异步迁移），采用延迟删除策略。

### 7.3 群组管理API

#### 7.3.1 群组操作 (9个端点)

**查询类接口（公开访问，无需认证，前缀：/api/v1/public/group）**：
- `POST /api/v1/public/group/search_by_name` - 按名称搜索群组
- `POST /api/v1/public/group/search_by_location` - 按位置搜索附近群组
- `POST /api/v1/public/group/search_by_id` - 按ID搜索群组
- `POST /api/v1/public/group/members` - 获取群组成员列表

**管理类接口（需要认证：Authorization: Bearer <jwt>，前缀：/api/v1/protected/group）**：
- `POST /api/v1/protected/group/create` - 创建群组
- `POST /api/v1/protected/group/join` - 加入群组
- `POST /api/v1/protected/group/leave` - 退出群组
- `POST /api/v1/protected/group/transfer_ownership` - 转让群主权限
- `POST /api/v1/protected/group/kick` - 踢出群组成员

**请求类型定义：**

- **CreateGroupRequest**：
  ```typescript
  interface CreateGroupRequest {
    name: string;         // 群组名称，2-32字符
    description?: string; // 群组描述，≤300字符
    latitude: number;     // 纬度，-90~90
    longitude: number;    // 经度，-180~180
    radius: number;       // 群组半径，单位米，>0且≤100000
    is_public: boolean;   // 是否公开群组
    password?: string;    // 群组密码，可选，1-32字符
  }
  ```

- **SearchGroupByNameRequest**：
  ```typescript
  interface SearchGroupByNameRequest {
    query: string;        // 搜索关键词，1-32字符
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **SearchGroupByLocationRequest**：
  ```typescript
  interface SearchGroupByLocationRequest {
    latitude: number;     // 纬度，-90~90
    longitude: number;    // 经度，-180~180
    radius: number;       // 搜索半径，单位米，>0且≤100000
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **SearchGroupByIdRequest**：
  ```typescript
  interface SearchGroupByIdRequest {
    group_id: string;     // 群组ID（UUID转字符串）
  }
  ```

- **JoinGroupRequest**：
  ```typescript
  interface JoinGroupRequest {
    group_id: string;     // 群组ID（UUID转字符串）
    password?: string;    // 群组密码，当群组需要密码时必填，1-32字符
  }
  ```

- **LeaveGroupRequest**：
  ```typescript
  interface LeaveGroupRequest {
    group_id: string;     // 群组ID（UUID转字符串）
  }
  ```

- **GroupMembersRequest**：
  ```typescript
  interface GroupMembersRequest {
    group_id: string;     // 群组ID（UUID转字符串）
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **TransferOwnershipRequest**：
  ```typescript
  interface TransferOwnershipRequest {
    group_id: string;     // 群组ID（UUID转字符串）
    target_user_id: string; // 目标用户ID（64位整数转字符串）
  }
  ```

- **KickMemberRequest**：
  ```typescript
  interface KickMemberRequest {
    group_id: string;     // 群组ID（UUID转字符串）
    target_user_id: string; // 目标用户ID（64位整数转字符串）
  }
  ```

**响应类型定义：**

```typescript
// 群组查询响应
interface SearchGroupByNameResponse extends PaginatedResponse<Group> {}
interface SearchGroupByLocationResponse extends PaginatedResponse<Group> {}
interface SearchGroupByIdResponse extends Group {}
interface GroupMembersResponse extends PaginatedResponse<User> {}

// 群组操作响应
interface CreateGroupResponse extends Group {}
interface JoinGroupResponse extends BasicSuccess {
  group?: Group;            // 可选，加入成功后的群组信息
}
interface LeaveGroupResponse extends BasicSuccess {}
interface TransferOwnershipResponse extends BasicSuccess {}
interface KickMemberResponse extends BasicSuccess {}
```

字段校验与业务规则（群组）：
- /group/create：
  - name 必填，2-32 字符；description 可选，≤ 300 字符；
  - 经纬度与 radius（>0 且 ≤ 100000 米）必填；is_public 必填；
  - password 可选，1-32 字符；设置密码后群组需要密码才能加入；
  - 创建者自动加入为群主；可异步派发欢迎/摘要任务；
  - 名称可做去重或相似度检测（实现可选）。
- /group/search_by_name：
  - query 必填，1-32 字符；游标分页规则同用户搜索；
  - 返回按相关性/活跃度排序（实现可选）。
- /group/search_by_location：
  - 经纬度/半径规则同用户位置搜索；
  - 结果可附带距离字段；按距离升序；
  - 可启用空间索引，限制最大半径以保护性能。
- /group/search_by_id：
  - group_id 必填；可返回是否已加入、成员数等扩展字段；
  - 若群组为私有且用户非成员，限制返回内容。
- /group/join 与 /group/leave：
  - group_id 必填；加入/退出需校验成员状态与容量限制；
  - 若群组设置了密码（is_password_required=true），则 password 字段必填且需验证正确；
  - 私有群需额外权限/口令（如有）；
  - 成功后通过 WebSocket 广播 GroupEvent（MemberJoined/MemberLeft）。
- /group/members：
  - group_id 必填；分页；
  - 返回成员基础信息及角色；
  - 非成员可限制可见字段。
- /group/transfer_ownership 与 /group/kick：
  - 仅群主可执行；target_user_id 必填且为现有成员；
  - 成功后推送 GroupEvent（OwnerTransferred/MemberKicked）。
- /group/user_groups 与 /group/owned_groups：
  - 需认证；支持分页；
  - 列表项可包含最近活跃时间与未读摘要（实现可选）。

**示例：创建群组**
（结构字段说明，示例省略）
POST /api/v1/protected/group/create
{
    "name": "咖啡爱好者",
    "description": "分享咖啡文化，探讨咖啡制作技巧",
    "longitude": 116.397128,
    "latitude": 39.916527,
    "radius": 2000,
    "is_public": true,
    "password": "coffee123"
}

Response:
{
    "code": 0,
    "msg": "群组创建成功",
    "data": {
        "group_id": 9876543210,
        "name": "咖啡爱好者",
        "description": "分享咖啡文化，探讨咖啡制作技巧",
        "creator_id": 1234567890,
        "member_count": 1,
        "is_password_required": true,
        "created_at": "2024-01-15T10:30:00Z"
    }
}

**示例：加入需要密码的群组**
POST /api/v1/protected/group/join
{
    "group_id": "9876543210",
    "password": "coffee123"
}

Response:
{
    "code": 0,
    "msg": "成功加入群组",
    "data": {
        "success": true,
        "group": {
            "group_id": "9876543210",
            "name": "咖啡爱好者",
            "member_count": 2,
            "is_member": true,
            "user_role": "member"
        }
    }
}
```

### 7.4 消息通讯API

#### 7.4.1 消息操作 (11个端点，前缀：/api/v1/protected/chat）

**消息发送**：
- `POST /api/v1/protected/chat/group/send` - 发送群组消息
- `POST /api/v1/protected/chat/user/send` - 发送私聊消息

**消息历史**：
- `POST /api/v1/protected/chat/group/history` - 获取群组消息历史
- `POST /api/v1/protected/chat/user/history` - 获取私聊消息历史

**消息管理**：
- `POST /api/v1/protected/chat/group/delete` - 删除群组消息
- `POST /api/v1/protected/chat/user/delete` - 删除私聊消息
- `POST /api/v1/protected/chat/user/conversations` - 获取会话列表
- `POST /api/v1/protected/chat/group/mark-read` - 标记群组消息已读
- `POST /api/v1/protected/chat/user/mark-read` - 标记私聊消息已读
- `POST /api/v1/protected/chat/group/unread_count` - 获取群组未读消息数
- `POST /api/v1/protected/chat/user/unread_count` - 获取私聊未读消息数

通用要求：以上端点均需要认证（Authorization: Bearer <jwt>）

**请求类型定义：**

- **SendGroupMessageRequest**：
  ```typescript
  interface SendGroupMessageRequest {
    group_id: string;     // 群组ID（UUID转字符串）
    content: string;      // 消息内容，1-2000字符
    message_type?: MessageType; // 消息类型，默认'text'
  }
  ```

- **SendDirectMessageRequest**：
  ```typescript
  interface SendDirectMessageRequest {
    recipient_id: string; // 接收者ID（64位整数转字符串）
    content: string;      // 消息内容，1-2000字符
    message_type?: MessageType; // 消息类型，默认'text'
  }
  ```

- **GroupMessageHistoryRequest**：
  ```typescript
  interface GroupMessageHistoryRequest {
    group_id: string;     // 群组ID（UUID转字符串）
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **DirectMessageHistoryRequest**：
  ```typescript
  interface DirectMessageHistoryRequest {
    conversation_id?: string; // 会话ID（UUID转字符串），优先使用
    recipient_id?: string;    // 接收者ID（64位整数转字符串），备用
    cursor?: string;          // 游标分页
    limit?: number;           // 1-100，默认20
  }
  ```

- **DeleteMessageRequest**：
  ```typescript
  interface DeleteMessageRequest {
    message_id: string;   // 消息ID（UUID转字符串）
  }
  ```

- **ConversationsRequest**：
  ```typescript
  interface ConversationsRequest {
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **MarkReadRequest**：
  ```typescript
  interface MarkReadRequest {
    group_id?: string;    // 群组ID（UUID转字符串），群组消息时使用
    conversation_id?: string; // 会话ID（UUID转字符串），私聊消息时使用
    last_read_message_id?: string; // 最后已读消息ID（UUID转字符串）
  }
  ```

- **GetUnreadCountRequest**：
  ```typescript
  interface GetUnreadCountRequest {
    group_id?: string;        // 群组ID（UUID转字符串），获取群组未读数
    conversation_id?: string; // 会话ID（UUID转字符串），获取私聊未读数
    // 两者都为空时返回汇总未读数
  }
  ```

**响应类型定义：**

```typescript
// 消息发送响应
interface SendGroupMessageResponse extends Message {}
interface SendDirectMessageResponse extends Message {}

// 消息历史响应
interface GroupMessageHistoryResponse extends PaginatedResponse<Message> {}
interface DirectMessageHistoryResponse extends PaginatedResponse<Message> {}

// 消息管理响应
interface DeleteMessageResponse extends BasicSuccess {}
interface ConversationsResponse extends PaginatedResponse<ConversationSummary> {}
interface MarkReadResponse extends BasicSuccess {}

// 未读计数响应
interface GroupUnreadCountResponse extends UnreadCount {}
interface UserUnreadCountResponse extends UnreadCount {}
```

字段校验与业务规则（聊天/消息）：
- /chat/*/send：
  - content 必填且去首尾空格，长度 1-2000；message_type 可选，默认为 text；
  - 群聊需校验成员身份；私聊需校验对端是否存在；
  - 服务端可拦截敏感/非法内容并返回参数错误；
  - 成功后通过 WebSocket NewMessage 广播至房间。
- /chat/*/history：
  - cursor/limit 采用游标分页：limit 1-100，默认 20；
  - group_id 或 conversation_id 必填（按端点）；
  - 返回消息按时间倒序或稳定排序（实现以 id 游标保证）。
- /chat/*/delete：
  - message_id 必填；仅发送者或具备管理权限者可删除；
  - 删除后通过 WebSocket MessageDeleted 通知房间。
- /chat/user/conversations：
  - 分页；可返回未读计数与最后消息预览；
  - 结果应与 NewMessage/MessageRead 事件保持一致性。
- /chat/*/mark-read：
  - last_read_message_id 可选；若未提供，以当前服务器时间点之前的所有消息视为已读；
  - 成功后通过 WebSocket MessageRead 通知。
- /chat/*/unread_count：
  - 校验目标存在性；总未读计数需与会话未读求和一致。

**示例：发送群组消息**
（结构字段说明，示例省略）
POST /api/v1/protected/chat/group/send
{
    "group_id": 9876543210,
    "content": "大家好！欢迎来到咖啡爱好者群组",
    "message_type": "text"
}

Response:
{
    "code": 0,
    "msg": "消息发送成功",
    "data": {
        "message_id": 1111111111,
        "group_id": 9876543210,
        "sender_id": 1234567890,
        "content": "大家好！欢迎来到咖啡爱好者群组",
        "created_at": "2024-01-15T10:35:00Z"
    }
}
```

#### 7.4.2 WebSocket实时通信

**连接端点**：
- `WebSocket /ws/` - 实时通信连接

**消息类型系统** (20+种消息类型)：

**用户状态类**：
- `UserOnline` - 用户上线通知
- `UserOffline` - 用户下线通知
- `UserLocationUpdate` - 用户位置更新

**群组消息类**：
- `GroupMessage` - 群组消息
- `GroupMemberJoined` - 新成员加入
- `GroupMemberLeft` - 成员退出
- `GroupRoleChanged` - 角色变更

**私聊消息类**：
- `DirectMessage` - 私聊消息
- `MessageDelivered` - 消息送达确认
- `MessageRead` - 消息已读回执
- `TypingIndicator` - 打字指示器

**系统通知类**：
- `SystemNotification` - 系统通知
- `NewMomentNearby` - 附近新生活瞬间通知
- `GroupInvitation` - 群组邀请

**连接管理类**：
- `ConnectionEstablished` - 连接建立确认
- `HeartbeatPing` / `HeartbeatPong` - 心跳检测
- `Error` - 错误通知

客户端 → 服务器（订阅/控制消息）需求说明：
- ClientMessage.Connect：
  - 字段：token: string（必填，有效 JWT）；
  - 行为：通过后端校验后建立已认证会话；失败返回 ServerMessage.Error。
- ClientMessage.Ping：
  - 用于保活；服务端返回 ServerMessage.Pong。
- ClientMessage.JoinUserRoom / LeaveUserRoom：
  - 订阅或取消订阅当前用户私有房间；需已认证；
  - 用于接收系统通知/私有事件。
- ClientMessage.JoinGroup / LeaveGroup：
  - 字段：group_id: uuid（必填）；
  - 需用户为该群成员；加入后可接收该群消息与群事件。
- ClientMessage.JoinConversation / LeaveConversation：
  - 字段：conversation_id: uuid（必填）；
  - 加入后接收该私聊会话消息与已读/输入指示等事件。

服务器 → 客户端（推送消息）需求说明：
- ServerMessage.Connected：
  - 字段：user_id: string；session_id: string；
  - 在 Connect 校验通过后下发，标识连接就绪。
- ServerMessage.Error：
  - 字段：code: string；msg: string；
  - 用于协议或权限错误（如未认证加入房间、参数无效）。
- ServerMessage.Success：
  - 字段：msg: string；
  - 用于通用成功反馈。
- ServerMessage.NewMessage：
  - 字段：message_id: uuid；room_id: uuid；room_type: 'group'|'conversation'；sender_id: string；sender_nickname: string；content: string；message_type: 'text'|'image'|'voice'|'video'|'location'|'file'；created_at: string；latitude?: number；longitude?: number；
  - 在群聊/私聊新消息产生时向对应房间广播。
- ServerMessage.UserStartedTyping / UserStoppedTyping：
  - 字段：room_id: uuid；user_id: string；nickname: string（仅 StartedTyping）；
  - 用于输入状态展示；客户端应设置超时兜底。
- ServerMessage.MessageRead：
  - 字段：room_id: uuid；user_id: string；message_id?: uuid；
  - 标识对方已读到某条消息或整段历史已读。
- ServerMessage.GroupEvent：
  - 字段：group_id: uuid；event_type: 'member_joined'|'member_left'|'member_kicked'|'owner_transferred'|'group_updated'；actor_id: string；actor_nickname: string；target_id?: string；target_nickname?: string；created_at: string；
  - 用于成员变更与群信息变更广播。
- ServerMessage.SystemNotification：
  - 字段：notification_id: uuid；notification_type: 'friend_request'|'group_invitation'|'system'|'announcement'|'new_moment'|'group_digest'；title: string；content: string；data?: object；created_at: string；
  - 用于通用系统级通知，data 可附带业务上下文。
- ServerMessage.MessageDeleted：
  - 字段：message_id: uuid；room_id: uuid；room_type: 'group'|'conversation'；
  - 客户端应移除或标记该消息已删除。

WebSocket 一致性与可靠性建议：
- 断线重连：重连后应重新发送 Connect 与 Join* 恢复订阅，并比对最新会话历史以补齐遗漏消息；
- 心跳与空闲超时：客户端定期发送 Ping（如 30s），服务端返回 Pong；服务端可设置空闲超时（如 60-120s）清理僵尸连接；
- 速率限制：对 Join/Leave、消息发送施加轻量限流，防止抖动与刷屏；
- 消息大小：建议限制单条 Text ≤ 32KB，超限返回 ServerMessage.Error（MESSAGE_TOO_LARGE）；
- 幂等性：重复 Join*/Leave* 为幂等操作；
- 错误对齐：WS 协议错误码与 HTTP 业务码不同维度，前端适配层需统一映射与展示。

### 7.5 生活瞬间API

#### 7.5.1 生活瞬间操作 (8个端点)

**查询类接口（公开访问，无需认证，前缀：/api/v1/public/moments）**：
- `POST /api/v1/public/moments/history` - 获取生活瞬间历史
- `POST /api/v1/public/moments/search_by_location` - 按位置搜索生活瞬间
- `POST /api/v1/public/moments/search_by_tags` - 按标签搜索生活瞬间
- `POST /api/v1/public/moments/search_by_id` - 按ID搜索生活瞬间

**管理类接口（需要认证：Authorization: Bearer <jwt>，前缀：/api/v1/protected/moments）**：
- `POST /api/v1/protected/moments/create` - 创建生活瞬间
- `POST /api/v1/protected/moments/delete` - 删除生活瞬间
- `POST /api/v1/protected/moments/like` - 点赞生活瞬间
- `POST /api/v1/protected/moments/unlike` - 取消点赞

**请求类型定义：**

- **CreateMomentRequest**：
  ```typescript
  interface CreateMomentRequest {
    latitude: number;     // 纬度，-90~90
    longitude: number;    // 经度，-180~180
    description?: string; // 瞬间描述，≤500字符
    tags?: string[];      // 标签数组，单个标签1-20字符，总数≤10
    image_urls?: string[]; // 图片URL数组，总数≤9
  }
  ```

- **DeleteMomentRequest**：
  ```typescript
  interface DeleteMomentRequest {
    moment_id: string;    // 瞬间ID（UUID转字符串）
  }
  ```

- **MomentHistoryRequest**：
  ```typescript
  interface MomentHistoryRequest {
    user_id?: string;     // 用户ID（64位整数转字符串），缺省表示当前用户
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **SearchMomentsByLocationRequest**：
  ```typescript
  interface SearchMomentsByLocationRequest {
    latitude: number;     // 纬度，-90~90
    longitude: number;    // 经度，-180~180
    radius: number;       // 搜索半径，单位米，>0且≤100000
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **SearchMomentsByTagsRequest**：
  ```typescript
  interface SearchMomentsByTagsRequest {
    tags: string[];       // 标签数组，至少1个，最多10个
    cursor?: string;      // 游标分页
    limit?: number;       // 1-100，默认20
  }
  ```

- **SearchMomentByIdRequest**：
  ```typescript
  interface SearchMomentByIdRequest {
    moment_id: string;    // 瞬间ID（UUID转字符串）
  }
  ```

- **LikeMomentRequest**：
  ```typescript
  interface LikeMomentRequest {
    moment_id: string;    // 瞬间ID（UUID转字符串）
  }
  ```

- **UnlikeMomentRequest**：
  ```typescript
  interface UnlikeMomentRequest {
    moment_id: string;    // 瞬间ID（UUID转字符串）
  }
  ```

**响应类型定义：**

```typescript
// 生活瞬间查询响应
interface MomentHistoryResponse extends PaginatedResponse<Moment> {}
interface SearchMomentsByLocationResponse extends PaginatedResponse<Moment> {}
interface SearchMomentsByTagsResponse extends PaginatedResponse<Moment> {}
interface SearchMomentByIdResponse extends Moment {}

// 生活瞬间操作响应
interface CreateMomentResponse extends Moment {}
interface DeleteMomentResponse extends BasicSuccess {}

// 点赞操作响应
interface LikeMomentResponse {
  moment_id: string;        // 瞬间ID（UUID转字符串）
  likes_count: number;      // 点赞数（32位整数）
  liked_by_me: boolean;     // 当前用户是否已点赞
}
interface UnlikeMomentResponse {
  moment_id: string;        // 瞬间ID（UUID转字符串）
  likes_count: number;      // 点赞数（32位整数）
  liked_by_me: boolean;     // 当前用户是否已点赞
}
```

字段校验与业务规则（生活瞬间）：
- /moments/create：
  - latitude/longitude 必填且范围合法；description 可选，≤ 500 字符；
  - tags 可选，单个标签 1-20 字符，总数 ≤ 10；image_urls 可选，总数 ≤ 9；
  - 成功后可通过 WebSocket SystemNotification(NewMoment) 推送附近用户（遵循策略）。
- /moments/delete：
  - 仅作者或管理员可删除；moment_id 必填；
  - 成功后可广播系统通知（实现可选）。
- /moments/history：
  - user_id 缺省表示取当前用户；支持游标分页；
  - 返回按创建时间倒序。
- /moments/search_by_location：
  - 半径限制与位置校验同用户位置搜索；
  - 返回结果可带 distance 字段；
  - 空间索引必需；限制最大半径保护性能。
- /moments/search_by_tags：
  - tags 至少 1 个，最多 10 个；标签值需标准化（小写、去空格）。
- /moments/like 与 /moments/unlike：
  - 幂等性保障：重复点赞/取消返回冲突或无副作用成功（实现选其一）；
  - 返回最新 likes_count 与 liked_by_me。

**示例：创建生活瞬间**
（结构字段说明，示例省略）
POST /api/v1/protected/moments/create
{
    "longitude": 116.397128,
    "latitude": 39.916527,
    "description": "在这家咖啡厅发现了非常棒的手冲咖啡！",
    "tags": ["咖啡", "手冲", "精品咖啡"],
    "image_urls": ["https://example.com/photo1.jpg"]
}

Response:
{
    "code": 0,
    "msg": "success",
    "data": {
        "moment_id": "m_2222222222",
        "user_id": 1234567890,
        "content": "在这家咖啡厅发现了非常棒的手冲咖啡！",
        "longitude": 116.397128,
        "latitude": 39.916527,
        "address": "北京市东城区天安门广场",
        "tags": ["咖啡", "手冲", "精品咖啡"],
        "likes_count": 0,
        "created_at": "2024-01-15T10:40:00Z"
    }
}
```

**示例：按位置搜索生活瞬间**
（结构字段说明，示例省略）
POST /api/v1/public/moments/search_by_location
{
    "longitude": 116.397128,
    "latitude": 39.916527,
    "radius": 1000,
    "limit": 20,
    "cursor": 0
}

Response:
{
    "code": 0,
    "msg": "success",
    "data": {
    "items": [
      {
        "moment_id": "m_2222222222",
        "user_id": 1234567890,
        "description": "在这家咖啡厅发现了非常棒的手冲咖啡！",
                "location_name": "北京市东城区天安门广场",
                "tags": ["咖啡", "手冲", "精品咖啡"],
        "like_count": 5,
                "distance": 250.5,
                "created_at": "2024-01-15T10:40:00Z"
            }
        ],
    "has_more": false,
    "next_cursor": null
    }
}
```

### 7.6 管理接口

#### 7.6.1 用户ID分区管理 (6个端点，前缀：/api/v1/protected/admin，需管理员认证）

**权限要求**：
- 需要有效的 JWT Token（Authorization: Bearer <jwt>）
- Token 中必须包含管理员角色标识
- 管理员权限通过后端角色验证系统确认
- 所有管理操作需记录审计日志

**用户ID分区管理接口（全部为 POST）**：
- `POST /api/v1/protected/admin/user-id/stats` - 获取分区统计信息
- `POST /api/v1/protected/admin/user-id/validate` - 验证用户ID有效性
- `POST /api/v1/protected/admin/user-id/info` - 获取用户ID信息
- `POST /api/v1/protected/admin/user-id/ranges` - 获取分区范围列表
- `POST /api/v1/protected/admin/user-id/generate` - 批量生成用户ID
- `POST /api/v1/protected/admin/user-id/format` - 格式化用户ID显示

**请求类型定义：**

```typescript
// 验证用户ID请求
interface ValidateUserIdRequest {
  user_id: string;          // 用户ID（64位整数转字符串）
}

// 获取用户ID信息请求
interface GetUserIdInfoRequest {
  user_id: string;          // 用户ID（64位整数转字符串）
}

// 生成用户ID请求
interface GenerateUserIdsRequest {
  count: number;            // 生成数量，1-1000
  type?: string;            // 用户类型，默认为 'normal'
}

// 格式化用户ID请求
interface FormatUserIdRequest {
  user_id: string;          // 用户ID（64位整数转字符串）
}

// 获取分区统计和范围无需请求体
```

**响应类型定义：**

```typescript
// 分区统计响应
interface PartitionStatsResponse {
  total_partitions: number;     // 总分区数（32位整数）
  active_partitions: number;    // 活跃分区数（32位整数）
  ranges: PartitionRange[];     // 分区范围数组
}

// 分区范围
interface PartitionRange {
  start: string;                // 起始ID（64位整数转字符串）
  end: string;                  // 结束ID（64位整数转字符串）
  type: string;                 // 分区类型（如 'normal', 'test', 'admin'）
}

// 用户ID验证响应
interface ValidateUserIdResponse {
  valid: boolean;               // 是否有效
  type?: string;                // 用户类型
  reason?: string;              // 无效原因
}

// 用户ID信息响应
interface UserIdInfoResponse {
  type: string;                 // 用户类型
  partition: number;            // 分区编号（32位整数）
  created_at?: string;          // 创建时间（ISO 8601）
}

// 分区范围列表响应
interface UserTypeRangesResponse extends Array<PartitionRange> {}

// 生成用户ID响应
interface GenerateUserIdsResponse {
  user_ids: GeneratedUserId[];  // 生成的用户ID数组
}

interface GeneratedUserId {
  user_id: string;              // 用户ID（64位整数转字符串）
  type: string;                 // 用户类型
}

// 格式化用户ID响应
interface FormatUserIdResponse {
  formatted: string;            // 格式化后的显示字符串
}
```

字段校验与业务规则（管理）：
- /admin/user-id/stats：
  - 需管理员认证；用于查看用户ID分区总体状态；
  - ranges 项内字段含义：start/end 为闭区间（API传输层为string，后端内部为i64），type 为分区类型标识（如 normal/test 等）。
- /admin/user-id/validate：
  - user_id 必填 string（API传输层为字符串，后端内部转换为i64处理）；校验其是否落入有效分区，是否被占用或保留；
  - 返回 valid=false 时给出 reason（如 out_of_range/reserved/invalid_format）。
- /admin/user-id/info：
  - user_id 必填 string（API传输层为字符串，后端内部转换为i64处理）；返回其分区类型与分区编号；
  - 若未分配或非法，返回相应提示。
- /admin/user-id/ranges：
  - 无请求体；返回当前所有分区范围清单；
  - 用于离线校验与观测。
- /admin/user-id/generate：
  - count 必填，1-1000；type 可选（默认为 normal）；
  - 分配的 user_id 唯一且不与已存在冲突；批量生成需具备幂等保障（重复请求不重复分配或返回相同结果）。
- /admin/user-id/format：
  - user_id 必填 string（API传输层为字符串，后端内部转换为i64处理）；返回格式化字符串（如带分区前缀/分隔符），仅用于展示。

典型错误码（管理）：
- 1002 未授权（缺少管理员身份）；1003 权限不足；1000 参数错误；5000 服务器内部错误。

### 7.7 API性能与安全

#### 7.7.1 性能优化

**分页策略（游标分页，当前实现）**：
- 默认每页20条记录，最大100条
- 全量采用游标分页，游标字段固定为各表的自增主键 `id`（后端内部为i64，API传输层转换为string）
- 查询时请求 `limit + 1` 条以判断是否还有下一页
- 返回 `has_more` 与 `next_cursor`，其中 `next_cursor` 为本次返回最后一条记录的 `id`（转换为字符串）

（分页 SQL 模式描述，示例省略）

**缓存策略**：
- 热点数据Redis缓存，TTL 5-30分钟
- 地理位置查询结果缓存优化
- **认证 nonce 缓存**：Redis 存储认证随机数，TTL 5分钟，key 格式：`auth_nonce:{login_id}:{nonce_id}`，支持多设备并发

**限流机制**：
- 按用户限流：普通接口 100次/分钟
- 按IP限流：防止恶意请求
- WebSocket连接数限制

#### 7.7.2 安全措施

**输入验证**：
- 所有参数严格类型检查和范围验证
- SQL注入防护（使用参数化查询）
- XSS防护（输出内容转义）

**权限控制**：
- JWT Token有效性验证
- 操作权限检查（如群组管理员权限）
- 敏感操作二次验证

### 7.8 后端密码安全实现指南

#### 7.8.1 核心组件实现

**NonceManager 实现要求**：
```rust
// 伪代码示例，实际实现需要完整的错误处理
pub struct NonceManager {
    redis_client: RedisClient,
}

impl NonceManager {
    // 生成并存储 nonce（支持多设备）
    pub async fn generate_nonce(&self, login_id: &str) -> Result<(String, String, i64), Error> {
        let nonce_id = generate_uuid(); // 生成唯一标识符
        let nonce = generate_random_string(32); // 使用 rand crate
        let timestamp = current_timestamp_millis();
        let key = format!("auth_nonce:{}:{}", login_id, nonce_id);
        
        // 存储到 Redis，TTL 5分钟，支持同一用户多个nonce
        self.redis_client.setex(&key, 300, &format!("{}:{}", nonce, timestamp)).await?;
        
        Ok((nonce_id, nonce, timestamp))
    }
    
    // 验证并消费 nonce（通过 nonce_id 精确定位）
    pub async fn verify_and_consume_nonce(&self, login_id: &str, nonce_id: &str, timestamp: i64) -> Result<Option<String>, Error> {
        let key = format!("auth_nonce:{}:{}", login_id, nonce_id);
        let stored_value = self.redis_client.get(&key).await?;
        
        if let Some(value) = stored_value {
            let parts: Vec<&str> = value.split(':').collect();
            if parts.len() == 2 {
                let stored_nonce = parts[0];
                let stored_timestamp: i64 = parts[1].parse().unwrap_or(0);
                
                // 验证时间戳匹配和有效性（±30秒容差）
                if stored_timestamp == timestamp {
                    let now = current_timestamp_millis();
                    if (now - timestamp).abs() <= 30000 {
                        // 验证成功，立即删除 nonce（防重放）
                        self.redis_client.del(&key).await?;
                        return Ok(Some(stored_nonce.to_string()));
                    }
                }
            }
        }
        
        Ok(None)
    }
    
    // 清理用户的所有过期 nonce（可选功能）
    pub async fn cleanup_expired_nonces(&self, login_id: &str) -> Result<(), Error> {
        let pattern = format!("auth_nonce:{}:*", login_id);
        let keys = self.redis_client.keys(&pattern).await?;
        
        for key in keys {
            // Redis TTL 会自动清理，这里只是提供手动清理接口
            if self.redis_client.ttl(&key).await? <= 0 {
                self.redis_client.del(&key).await?;
            }
        }
        
        Ok(())
    }
}
```

**PasswordHasher 实现要求**：
```rust
// 伪代码示例
pub struct PasswordHasher;

impl PasswordHasher {
    // 用于存储的密码哈希（Argon2）
    pub fn hash_for_storage(password: &str) -> Result<String, Error> {
        // 使用 Argon2 算法哈希密码用于数据库存储
        argon2::hash_encoded(password.as_bytes(), &salt, &config)
    }
    
    // 用于认证的动态哈希（SHA-256）
    pub fn hash_for_auth(password: &str, nonce: &str, timestamp: i64, login_id: &str) -> String {
        let input = format!("{}{}{}{}", password, nonce, timestamp, login_id);
        sha256::digest(input.as_bytes())
    }
    
    // 验证存储的密码
    pub fn verify_stored_password(password: &str, hash: &str) -> Result<bool, Error> {
        argon2::verify_encoded(hash, password.as_bytes())
    }
}
```

#### 7.8.2 认证流程实现

**get_nonce 接口实现**：
1. 验证 login_id 格式
2. 调用 NonceManager::generate_nonce()
3. 返回 nonce_id、nonce、timestamp、expires_at

**register/login 接口实现**：
1. 验证请求参数格式（包括 nonce_id 和 timestamp）
2. 调用 NonceManager::verify_and_consume_nonce() 获取对应的 nonce
3. 使用获取到的 nonce 重新计算哈希：PasswordHasher::hash_for_auth(password, nonce, timestamp, login_id)
4. 对于注册：
   - 验证哈希匹配后，使用 PasswordHasher::hash_for_storage() 存储密码
5. 对于登录：
   - 从数据库获取用户存储的密码哈希
   - 使用 PasswordHasher::verify_stored_password() 验证原密码
   - 比对重新计算的哈希与请求中的哈希
6. 生成 JWT token 返回

**set_password 接口实现**：
1. 验证 JWT token 获取 user_id
2. 通过 nonce_id 从 Redis 获取对应的 nonce
3. 使用获取到的 nonce 验证新密码哈希的正确性
4. 如果提供了 old_password_hash，同样验证原密码
5. 使用 PasswordHasher::hash_for_storage() 存储新密码

#### 7.8.3 安全注意事项

**时间同步**：
- 服务器时间戳验证容差设置为 ±30 秒
- 建议使用 NTP 同步服务器时间

**错误处理**：
- nonce 验证失败统一返回 "认证失败" 错误
- 不暴露具体失败原因（nonce 过期、无效等）
- 记录详细错误日志用于调试

**性能优化**：
- Redis 连接池管理
- nonce 生成使用加密安全的随机数生成器
- 批量清理过期 nonce（可选）

**监控指标**：
- nonce 生成/验证成功率
- 认证失败次数和原因分布
- Redis 连接状态和性能指标

### 7.9 接口与 WebSocket 速查表

注：完整字段与校验规则见各自小节；此处为快捷索引。响应模型均为“复用公共模型：X”。

【认证（/api/v1/auth）】
- POST /get_nonce → 响应：NonceResponse；错误：1000, 5000
- POST /register → 响应：AuthResponse；错误：1000, 1005, 5001, 5000
- POST /login → 响应：AuthResponse；错误：1000, 1002, 5000
- POST /create_temp → 响应：AuthResponse；错误：5000
- POST /refresh_token → 响应：AuthResponse；错误：1000, 5000
- POST /check_status → 响应：AuthResponse；错误：1002, 5000
- POST /set_password → 响应：BasicSuccess/空；错误：1000, 1002, 5000
- POST /password_status → 响应：PasswordStatusResult；错误：1002, 5000

【用户（/api/v1/public/user & /api/v1/protected/user）】
- POST /public/user/search_by_name → 响应：PaginatedResponse<User>；错误：1000, 5001, 5000 **（公开）**
- POST /public/user/search_by_id → 响应：User；错误：1000, 1004, 5000 **（公开）**
- POST /public/user/search_by_location → 响应：PaginatedResponse<User>；错误：1000, 5001, 5000 **（公开）**
- POST /protected/user/update_location → 响应：BasicSuccess/空；错误：1000, 5000 **（需认证）**
- POST /protected/user/update_profile → 响应：{ nickname }/BasicSuccess；错误：1000, 5000 **（需认证）**
- POST /protected/user/upgrade_account → 响应：AuthResponse；错误：1000, 1005, 1002, 5000 **（需认证）**
- POST /protected/user/get_settings → 响应：用户设置对象；错误：1002, 5000 **（需认证）**
- POST /protected/user/update_settings → 响应：用户设置对象/BasicSuccess；错误：1000, 5000 **（需认证）**
- POST /protected/user/delete_account → 响应：BasicSuccess/空；错误：1002, 5000 **（需认证）**

【群组（/api/v1/public/group & /api/v1/protected/group）】
- POST /public/group/search_by_name → 响应：PaginatedResponse<Group>；错误：1000, 5001, 5000 **（公开）**
- POST /public/group/search_by_location → 响应：PaginatedResponse<Group>；错误：1000, 5001, 5000 **（公开）**
- POST /public/group/search_by_id → 响应：Group；错误：1000, 1004, 5000 **（公开）**
- POST /public/group/members → 响应：PaginatedResponse<User>；错误：1000, 1004, 5000 **（公开）**
- POST /protected/group/create → 响应：Group；错误：1000, 5001, 5000 **（需认证）**
- POST /protected/group/join → 响应：BasicSuccess/{ group }；错误：1000, 1004, 1003, 1002（密码错误）, 5000 **（需认证）**
- POST /protected/group/leave → 响应：BasicSuccess；错误：1000, 1004, 5000 **（需认证）**
- POST /protected/group/transfer_ownership → 响应：BasicSuccess；错误：1000, 1003, 1004, 5000 **（需认证）**
- POST /protected/group/kick → 响应：BasicSuccess；错误：1000, 1003, 1004, 5000 **（需认证）**

【消息（/api/v1/protected/chat）】
- POST /protected/chat/group/send → 响应：Message；错误：1000, 1003, 1004, 5000
- POST /protected/chat/user/send → 响应：Message；错误：1000, 1004, 5000
- POST /protected/chat/group/history → 响应：PaginatedResponse<Message>；错误：1000, 1004, 5000
- POST /protected/chat/user/history → 响应：PaginatedResponse<Message>；错误：1000, 1004, 5000
- POST /protected/chat/group/delete → 响应：BasicSuccess；错误：1000, 1003, 1004, 5000
- POST /protected/chat/user/delete → 响应：BasicSuccess；错误：1000, 1003, 1004, 5000
- POST /protected/chat/user/conversations → 响应：PaginatedResponse<ConversationSummary>；错误：1002, 5000
- POST /protected/chat/group/mark-read → 响应：BasicSuccess；错误：1000, 1004, 5000
- POST /protected/chat/user/mark-read → 响应：BasicSuccess；错误：1000, 1004, 5000
- POST /protected/chat/group/unread_count → 响应：UnreadCount（群组未读）；错误：1000, 1004, 5000
- POST /protected/chat/user/unread_count → 响应：UnreadCount；错误：1000, 5000

【生活瞬间（/api/v1/public/moments & /api/v1/protected/moments）】
- POST /public/moments/history → 响应：PaginatedResponse<Moment>；错误：1000, 5000 **（公开）**
- POST /public/moments/search_by_location → 响应：PaginatedResponse<Moment>；错误：1000, 5000 **（公开）**
- POST /public/moments/search_by_tags → 响应：PaginatedResponse<Moment>；错误：1000, 5000 **（公开）**
- POST /public/moments/search_by_id → 响应：Moment；错误：1000, 1004, 5000 **（公开）**
- POST /protected/moments/create → 响应：Moment；错误：1000, 5000 **（需认证）**
- POST /protected/moments/delete → 响应：BasicSuccess；错误：1000, 1003, 1004, 5000 **（需认证）**
- POST /protected/moments/like → 响应：{ moment_id, likes_count, liked_by_me }；错误：1000, 1004, 1005, 5000 **（需认证）**
- POST /protected/moments/unlike → 响应：{ moment_id, likes_count, liked_by_me }；错误：1000, 1004, 1005, 5000 **（需认证）**

【管理（/api/v1/protected/admin）】
- POST /protected/admin/user-id/stats → 响应：{ total_partitions, active_partitions, ranges[] }；错误：1002, 1003, 1000, 5000
- POST /protected/admin/user-id/validate → 响应：{ valid, type?, reason? }；错误：1002, 1003, 1000, 5000
- POST /protected/admin/user-id/info → 响应：{ type, partition, created_at? }；错误：1002, 1003, 1000, 5000
- POST /protected/admin/user-id/ranges → 响应：ranges[]；错误：1002, 1003, 1000, 5000
- POST /protected/admin/user-id/generate → 响应：[{ user_id, type }]；错误：1002, 1003, 1000, 5000
- POST /protected/admin/user-id/format → 响应：{ formatted }；错误：1002, 1003, 1000, 5000

【WebSocket 快速索引】
- 客户端 → 服务器：
  - Connect{ token } → 错误：INVALID_AUTH_MESSAGE, INVALID_TOKEN
  - Ping → 响应：Pong
  - JoinUserRoom/LeaveUserRoom → 错误：UNAUTHORIZED（未认证时）
  - JoinGroup{ group_id }/LeaveGroup{ group_id } → 错误：FORBIDDEN, ROOM_NOT_FOUND
  - JoinConversation{ conversation_id }/LeaveConversation{ conversation_id } → 错误：FORBIDDEN
- 服务器 → 客户端：
  - Connected{ user_id, session_id }
  - Error{ code, msg }（WS 专用错误码见 7.1.3）
  - Success{ msg }
  - NewMessage → 负载：WsMessageNotification
  - UserStartedTyping/UserStoppedTyping → { room_id, user_id, nickname? }
  - MessageRead → { room_id, user_id, message_id? }
  - GroupEvent → 负载：GroupEvent
  - SystemNotification → 负载：SystemNotification
  - MessageDeleted → { message_id, room_id, room_type }

## 8. 部署与运维

### 8.1 部署架构

#### 8.1.1 生产环境部署

**Docker容器化部署**（文字描述）：后端采用多阶段构建（编译镜像 → 运行镜像），仅保留最终可执行文件与必要运行时依赖；端口与环境变量通过容器编排传入（具体 Dockerfile 省略）。

**docker-compose 配置**（文字描述）：定义 backend/frontend/postgres/redis 四个服务，数据库使用 PostGIS 镜像并挂载迁移目录，公开访问端口映射，密钥/连接串通过环境变量传入（compose 细节省略）。

#### 8.1.2 环境配置

**环境变量配置**（文字描述）：
- 数据库/缓存：DATABASE_URL、REDIS_URL
- 认证：JWT_SECRET、JWT_EXPIRATION（秒）
- 服务器：SERVER_HOST、SERVER_PORT
- 地图服务：AMAP_API_KEY、AMAP_SECRET_KEY
- 监控：PROMETHEUS_ENABLED、METRICS_PORT

### 8.2 监控与维护

#### 8.2.1 性能监控

**Prometheus指标收集**（文字描述）：暴露 HTTP 请求计数/延迟、WebSocket 连接数/吞吐、地理查询耗时等核心指标，便于监控与告警（实现细节与示例代码省略）。

**关键监控指标**：
- API响应时间和成功率
- WebSocket连接数和消息吞吐量
- 数据库查询性能（特别是地理位置查询）
- 内存和CPU使用率
- 缓存命中率

#### 8.2.2 日志管理

**结构化日志配置**（文字描述）：使用分层日志与环境变量控制日志级别，业务日志包含关键上下文字段，便于排查（实现细节与示例代码省略）。

**日志分级策略**：
- **ERROR**: 系统错误、数据库连接失败
- **WARN**: 业务警告、性能问题
- **INFO**: 重要业务事件、用户操作
- **DEBUG**: 开发调试信息

### 8.3 数据备份与恢复

#### 8.3.1 数据库备份策略

**自动备份策略**（文字描述）：按日执行全量备份并压缩，保留至少 7 天；失败告警与恢复流程（包括索引重建与缓存重建）按标准运维手册执行。

**备份内容**：
- 完整数据库结构和数据
- PostGIS空间数据完整性
- 用户上传文件（头像、moment图片）
- 配置文件和环境变量

#### 8.3.2 灾难恢复计划

**恢复步骤**：
1. 恢复数据库：`psql -h localhost -U postgres -d hotspot < backup.sql`
2. 重建PostGIS索引：验证空间索引完整性
3. 恢复Redis缓存：重启服务自动重建缓存
4. 验证服务状态：健康检查和功能测试

**RTO/RPO目标**：
- **RTO (恢复时间目标)**: 30分钟内服务恢复
- **RPO (恢复点目标)**: 数据丢失不超过1小时

---

## 总结

HotSpot (HotSpot) 项目已经完成了核心功能的实现，具备了完整的LBS社交平台基础架构。通过Nuxt3+Vue3的现代前端技术栈和Rust+Axum的高性能后端架构，项目实现了：

**核心功能完成度**：
- ✅ **用户系统**：完整的临时用户/注册用户双轨制
- ✅ **群组管理**：基于地理位置的群组创建和管理
- ✅ **实时通讯**：WebSocket支持的即时消息系统
- ✅ **位置服务**：PostGIS支持的精确地理位置查询
- ✅ **生活瞬间系统**：基于位置的瞬间分享与互动

**技术架构优势**：
- **高性能**: Rust+Axum提供出色的并发处理能力
- **实时性**: 完整的WebSocket实时通信系统
- **扩展性**: 基于PostGIS的地理位置服务，支持复杂空间查询
- **用户体验**: 临时用户机制降低使用门槛
- **开发效率**: Nuxt3自动导入和TypeScript类型安全

**下一步发展方向**：
1. **性能优化**：数据库查询优化、缓存策略改进
2. **功能扩展**：推荐算法、内容审核、移动端APP
3. **运维完善**：监控体系、自动化部署、灾备方案
4. **社区建设**：用户反馈收集、功能迭代优化

项目已具备商业化部署的技术基础，可根据用户反馈和业务需求进行功能扩展和性能优化
