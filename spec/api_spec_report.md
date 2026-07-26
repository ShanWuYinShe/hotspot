# HotSpot API - 举报

> 本文档定义 HotSpot 举报审核 (`reports`) 相关的 RPC 接口。管理员审核接口定义于 `api_spec_admin_moderation.md`。

## 模块信息

- **命名空间**: `reports`
- **基础路径**: `/api/v1/reports/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`（`ReportTargetType`、`ReportStatus`、`PaginationParams`、`PaginatedData` 等枚举/工具类型）
  - 错误码：`api_spec_errors.md`
  - 管理员举报审核：`api_spec_admin_moderation.md`

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/reports/create` | 创建举报 | 必须 | MUST |
| `POST /api/v1/reports/list_my` | 我的举报记录 | 必须 | 无 |

## 举报审核流程说明

举报处理的状态流转为：`pending` 进入 `reviewing`，可 `escalate` 升级为 `escalated` 等待更高级管理员接手，最终为 `resolved` 或 `dismissed`。

**状态流转**：

| 状态 | 说明 | 可执行操作 |
|-----|------|-----------|
| `pending` | 待审核 | 管理员可开始审核（→ `reviewing`），或直接处理/升级 |
| `reviewing` | 审核中（某管理员已认领） | 管理员可处理（→ `resolved`/`dismissed`）或升级（→ `escalated`） |
| `escalated` | 已升级，等待更高级管理员接手 | 更高级权限管理员可认领（→ `reviewing`）后处理；避免举报卡死，应配置升级超时回退机制（见下方说明） |
| `resolved` | 已处理（确认违规） | 已采取措施（封禁/删除内容） |
| `dismissed` | 已驳回（无违规） | 审核结束，无需处理 |

**`escalated` 防卡死设计**：
- `escalated` 与 `reviewing` 语义区分：`reviewing` = 某管理员正在审核；`escalated` = 已升级但尚未被更高级管理员认领
- 升级后由更高级权限管理员通过 `handle`（action=`escalate` 之外的动作，或新增认领动作）接手，状态回到 `reviewing`
- **超时回退**（契约级要求）：升级后若超过服务端配置时长（建议 72 小时，可由 `AppConfig` 调整）无人认领，服务端自动回退为 `pending` 重新进入待审队列，避免升级后无更高级管理员时举报永久卡死

管理员审核流程（处理动作、内容快照机制等）详见 `api_spec_admin_moderation.md`。

**不同 `ReportTargetType` 的处理逻辑**：

| 目标类型 | 可执行措施 |
|---------|-----------|
| `user` | 封禁用户 |
| `moment` | 删除瞬间 + 警告/封禁发布者 |
| `comment` | 删除评论 + 警告发布者 |
| `message` | 删除消息 + 警告/封禁发送者 |
| `group` | 解散群组（群违规统一走解散，不连坐封禁群主；若群主本人违规，由管理员单独对其发起封禁） |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `ReportTargetType`
- `ReportReason`
- `ReportStatus`
- `ReportItem`
- `PaginationParams`
- `PaginatedData`

## 端点详情

### POST /api/v1/reports/create

创建举报。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**字段说明**：
- `reason`：举报原因，从 `ReportReason` 枚举中选取
- `description`：举报详细说明，0-1000 字符，`reason` 为 `'other'` 时建议填写
- `target_id`：统一为 string 类型（与全规范 `xxx_id` 一致）；不同 `target_type` 下语义为对应的 `user_id`/`moment_id`/`comment_id`/`message_id`/`group_id`

**业务规则**：
- 禁止自举报：当 `target_type='user'` 且 `target_id` 为当前用户时，返回 `10400`
- 当 `target_type='moment'`/`'comment'`/`'message'` 且目标资源属于当前用户时，返回 `10400`
- 重复举报：同一 `target_type + target_id` 已存在 `pending`、`reviewing` 或 `escalated` 记录时，返回 `10400`
- 若该目标历史举报均已进入终态（`resolved` / `dismissed`），允许再次举报（视为新举报单）
- **消息举报快照**（隐私边界）：当 `target_type='message'` 时，`message_snapshot` 为**必填**——举报人必须提交举报时的消息内容快照。服务端原样存储，管理员审核**基于快照**判断，不直接查询私聊消息库。这既保证管理员有足够信息审核，又避免管理员随意翻阅用户私聊内容的隐私风险。缺少 `message_snapshot` 或快照不完整时返回 `10000`。

**Request**: `CreateReportRequest`
```typescript
interface CreateReportRequest {
  target_type: ReportTargetType;
  target_id: string;            // 统一 string；语义随 target_type 为对应资源 ID
  reason: ReportReason;         // 举报原因（从枚举中选取）
  description?: string;         // 举报详细说明（0-1000 字符），reason 为 'other' 时建议填写
  message_snapshot?: ReportSnapshot;  // target_type='message' 时必填：消息内容快照（见 ReportSnapshot，定义于 api_spec_types.md）
}
```

**Response**: `ApiResponse<ReportCreated>`
```typescript
interface ReportCreated {
  report_id: string;
}
```

**错误码**: `10000` `10300` `10400`（自举报、重复举报等）

### POST /api/v1/reports/list_my

我的举报记录。

- **认证**: 必须
- **幂等**: 无
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `report_id DESC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `MyReportsRequest`
```typescript
interface MyReportsRequest {
  status?: ReportStatus;
  target_type?: ReportTargetType;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<ReportItem>>`

`ReportItem` 定义于 `api_spec_types.md`。

**可见性说明（契约级）**：举报人在 `reports/list_my` 可见自己举报的完整状态机（`pending`/`reviewing`/`escalated`/终态）及 `ReportItem.resolution`（`@owner-only`，仅举报人自己可见处理结果摘要）。**不可见**处理管理员的身份（隐私保护，避免举报人针对性报复）——`resolution` 不含 `handled_by`，仅含 `action`/`resolved_at`/`note`(脱敏)。

**错误码**: `10000`
