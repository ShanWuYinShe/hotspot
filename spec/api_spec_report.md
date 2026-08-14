# HotSpot API - 举报

> 本文档定义 HotSpot 举报 (`reports`) 相关的 RPC 接口。平台管理员审核接口定义于 `api_spec_admin.md`。

## 模块信息

- **命名空间**: `reports`
- **基础路径**: `/api/v1/reports/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`（`ReportTargetType`、`ReportStatus`、`PaginationParams`、`PaginatedData` 等）
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 |
|-----|------|------|
| `POST /api/v1/reports/create` | 创建举报 | required |
| `POST /api/v1/reports/list` | 我的举报记录（命名空间天生私有，直接 list） | required |

## 举报审核流程说明

**状态流转**：

| 状态 | 说明 |
|-----|------|
| `pending` | 待审核 |
| `resolved` | 已处理（确认违规，已采取措施：封禁/删除内容） |
| `dismissed` | 已驳回（无违规，审核结束） |

平台管理员审核时将举报从 `pending` 直接流转至 `resolved` 或 `dismissed`（处理流程、内容快照机制等详见 `api_spec_admin.md`）。

**不同 `ReportTargetType` 的处理逻辑**：

| 目标类型 | 可执行措施 |
|---------|-----------|
| `user` | 封禁用户 |
| `moment` | 删除瞬间 + 警告/封禁发布者 |
| `comment` | 删除评论 + 警告发布者 |
| `message` | 删除消息 + 警告/封禁发送者 |
| `group` | 解散群组（群违规统一走解散，不连坐封禁群主；若群主本人违规，由平台管理员单独对其发起封禁） |

## 类型定义

以下类型定义于 `api_spec_types.md`：`ReportTargetType`、`ReportReason`、`ReportStatus`、`ReportItem`、`PaginationParams`、`PaginatedData`。

## 端点详情

### POST /api/v1/reports/create

创建举报。

- **认证**: required

**字段说明**：
- `reason`：举报原因，从 `ReportReason` 枚举中选取
- `description`：举报详细说明，0-1000 字符，`reason` 为 `'other'` 时建议填写
- `target_id`：统一为 string 类型（与全规范 `xxx_id` 一致）；不同 `target_type` 下语义为对应的 `user_id`/`moment_id`/`comment_id`/`message_id`/`group_id`

**业务规则**：
- 禁止自举报：当 `target_type='user'` 且 `target_id` 为当前用户时，返回 `10400`
- 当 `target_type='moment'`/`'comment'`/`'message'` 且目标资源属于当前用户时，返回 `10400`
- 当 `target_type='group'` 且调用方为该群群主时，返回 `10400`（群主举报自己的群无审核意义）
- 重复举报：同一 `target_type + target_id` 已存在 `pending` 记录时，**返回成功**（`code=0`），`data` 中包含已有举报的 `report_id` 及 `is_duplicate: true`，不创建新举报单。这本质是天然幂等——`(reporter_id, target_type, target_id, pending_status)` 构成自然去重键
- 若该目标历史举报均已进入终态（`resolved` / `dismissed`），允许再次举报（视为新举报单）
- **消息举报快照（隐私边界）**：当 `target_type='message'` 时，`message_snapshot` 为**必填**——举报人必须提交举报时的消息内容快照。服务端原样存储，平台管理员审核**基于快照**判断，不直接查询私聊消息库。缺少 `message_snapshot` 或快照不完整时请求失败。

**Request**: `CreateReportRequest`
```typescript
interface CreateReportRequest {
  target_type: ReportTargetType;
  target_id: string;            // 统一 string；语义随 target_type 为对应资源 ID
  reason: ReportReason;         // 举报原因（从枚举中选取）
  description: string | null;   // 举报详细说明（0-1000 字符）
  message_snapshot: ReportSnapshot | null;  // target_type='message' 时必填：消息内容快照（见 ReportSnapshot，定义于 api_spec_types.md）
}
```

**Response**: `ApiResponse<ReportCreated>`
```typescript
interface ReportCreated {
  report_id: string;
  is_duplicate: boolean;  // true 表示该举报为已有举报的幂等回放（未创建新举报单），report_id 为已有举报的 ID
}
```

**错误码**: `10300` `10400`（自举报等）

---

### POST /api/v1/reports/list

我的举报记录。

- **认证**: required
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `report_id DESC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `ListReportsRequest`
```typescript
interface ListReportsRequest {
  status: ReportStatus | null;
  target_type: ReportTargetType | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<ReportItem>>`

**可见性说明（契约级）**：举报人在 `reports/list` 可见自己举报的完整状态机（`pending`/终态）及 `ReportItem.resolution`（null = 尚未处理）。**不可见**处理平台管理员的身份（隐私保护）——`resolution` 不含 `handled_by`，仅含 `action`/`resolved_at`/`note`(脱敏)。