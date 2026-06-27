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
| `POST /api/v1/reports/me/list` | 我的举报记录 | 必须 | 无 |

## 举报审核流程说明

举报处理的状态流转为：`pending` 进入 `reviewing`，最终为 `resolved` 或 `dismissed`。

**状态流转**：

| 状态 | 说明 | 可执行操作 |
|-----|------|-----------|
| `pending` | 待审核 | 管理员可开始审核 |
| `reviewing` | 审核中 | 管理员可处理或升级 |
| `resolved` | 已处理（确认违规） | 已采取措施（封禁/删除内容） |
| `dismissed` | 已驳回（无违规） | 审核结束，无需处理 |

管理员审核流程（处理动作、内容快照机制等）详见 `api_spec_admin_moderation.md`。

**不同 `ReportTargetType` 的处理逻辑**：

| 目标类型 | 可执行措施 |
|---------|-----------|
| `user` | 封禁用户 |
| `moment` | 删除瞬间 + 警告/封禁发布者 |
| `comment` | 删除评论 + 警告发布者 |
| `message` | 删除消息 + 警告/封禁发送者 |
| `group` | 解散群组 + 封禁群主 |

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
- `target_id`：统一为 string 类型；举报用户时将 `user_id` 转为字符串格式

**业务规则**：
- 禁止自举报：当 `target_type='user'` 且 `target_id` 为当前用户时，返回 `10400`
- 当 `target_type='moment'`/`'comment'`/`'message'` 且目标资源属于当前用户时，返回 `10400`
- 重复举报：同一 `target_type + target_id` 已存在 `pending` 或 `reviewing` 记录时，返回 `10400`
- 若该目标历史举报均已进入终态（`resolved` / `dismissed`），允许再次举报（视为新举报单）

**Request**: `CreateReportRequest`
```typescript
interface CreateReportRequest {
  target_type: ReportTargetType;
  target_id: string;            // 统一为 string（user_id 需转为字符串格式）
  reason: ReportReason;         // 举报原因（从枚举中选取）
  description?: string;         // 举报详细说明（0-1000 字符），reason 为 'other' 时建议填写
}
```

**Response**: `ApiResponse<ReportCreated>`
```typescript
interface ReportCreated {
  report_id: string;
}
```

**错误码**: `10000` `10300` `10400`（自举报、重复举报等）

### POST /api/v1/reports/me/list

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

**错误码**: `10000`
