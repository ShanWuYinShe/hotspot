---
globs: backend-go/**
alwaysApply: false
---

# HotSpot 后端文档（What）

> 适用范围：本文件针对 `backend-go/`（Go 主后端）。历史 `backend/`（Rust/Axum）已暂停，不在本文档描述范围内。

## 文档定位（权威边界）

本文件只回答"后端提供什么能力"：
- 业务能力与模块边界
- 对外行为与关键用户可感知流程（说明性）
- Out of scope

本文件不包含：
- 任何工程实现细节、数据库结构、缓存策略
- API 端点/字段/错误码/WS 事件定义

权威引用：
- **API 契约**：`@spec/api_spec_index.md`
- **基础规范**：`@spec/api_spec_base.md`
- **通用类型**：`@spec/api_spec_types.md`
- **错误码**：`@spec/api_spec_errors.md`
- **实现规范**：`@spec/backend_rule.md`

---

## 系统能力概览（按域）

### 认证与会话（auth）
- 注册/登录/访客/刷新/登出/状态检查
- Web 端 Cookie、App 端 Header 的统一兼容
- 会话撤销后立即失效（对用户可感知）

### 用户（user）
- 用户搜索与资料详情
- 关系管理：关注/取消关注、拉黑/取消拉黑、关系查询
- 当前用户：资料、位置、设置、统计、注销

### 群组（group）
- 群组创建/更新/解散
- 成员管理：加入/退出、邀请/申请与审核
- 角色与权限、群公告管理

### 私聊与群聊（chat）
- 会话列表/详情、消息发送与历史
- 消息编辑/撤回/删除
- 未读计数以 WebSocket 增量维护为主，服务端仅用于冷启动校准

### 生活瞬间与评论（moment/comment）
- 发布/编辑/删除
- 点赞/收藏与评论/回复
- 搜索与详情

### 信息流与搜索（feed/search/tags）
- 信息流推荐、热门内容
- 搜索建议与标签发现

### 地图（map）
- 视口驱动的地图渲染与聚合

### 举报（report）
- 创建举报与我的举报记录

### 分享（share）
- 创建/解析/应用分享

### 媒体上传（media）
- 预签名上传流程
- 对象存储为 S3 兼容；开发环境可使用 MinIO

### 系统与配置（system/server_config/client_config/announcements）
- 健康检查与版本元信息
- 全局配置读取与系统公告

### 管理后台（admin）
- RBAC 与管理员授权
- 用户/群组/举报管理
- 标签管理
- 系统公告与系统配置
- 审计日志与导出

---

## 实时能力（WebSocket）

- 连接、鉴权、订阅与事件推送
- 事件与订阅模型以 `@spec/api_spec_websocket.md` 为准

---

## 非目标（Out of scope）

- 支付/交易能力
- 端侧离线同步算法
- 消息全文搜索
