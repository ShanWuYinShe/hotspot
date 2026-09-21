# HotSpot API - 评论

> 本文档定义 HotSpot 评论 (`comment`) 相关的 RPC 接口。

## 模块信息

- **命名空间**: `comments`
- **基础路径**: `/api/v1/comments/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`
  - 匿名镜像版（/public/v1 同名端点）：`api_spec_public.md`

## 端点总览

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/comments/create` | 创建评论/回复 | /api/v1 |
| `POST /api/v1/comments/delete` | 删除评论 | /api/v1 |
| `POST /api/v1/comments/fetch` | 顶层评论列表（双版本，匿名版在 /public/v1） | /api/v1 |
| `POST /api/v1/comments/fetch_replies` | 子评论列表（双版本） | /api/v1 |
| `POST /api/v1/comments/open_the_door` | 评论详情（双版本） | /api/v1 |

> 表情反应（reaction）统一由 `POST /api/v1/reactions/set` 处理，反应列表统一由 `POST /api/v1/reactions/fetch` 查询（见 `api_spec_reaction.md`）。

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `CommentItem`
- `PaginationParams`
- `UserInfo`

## 评论层级模型

评论采用**两级结构**（顶层评论 + 一级回复），不支持多层嵌套：

- **顶层评论**：`parent_id` 为 null，直接挂载在瞬间下
- **回复**：`parent_id` 指向一条顶层评论，回复平铺在该顶层评论下
- **回复的回复**：若 `parent_id` 指向一条已有 `parent_id` 的评论（即指向另一条回复），服务端**自动将其挂载到对应的顶层评论下**（`parent_id` 重写为顶层评论 ID），保证数据库中回复始终只有一层
- `fetch_replies` 返回指定顶层评论下的所有回复（扁平列表），不存在递归嵌套

## 端点详情

### POST /api/v1/comments/create

添加评论/回复。

- **节点**: /api/v1
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateCommentRequest`
```typescript
interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id: string | null;    // 回复的目标评论 ID（不传则为顶层评论；若指向二级回复，服务端自动挂载到其所属顶层评论）
  mentions: string[];
}
```

**Response**: `ApiResponse<CommentItem>`

**错误码**: `10200`（privacy_setting 瞬间作者限制评论 / blocked_by_user 被对方拉黑） `10300`（瞬间不存在或不可见）

---

### POST /api/v1/comments/delete

删除评论（**仅评论作者**）。平台审核删除评论统一走 `/admin/api/v1/reports/judge`（见 `api_spec_admin.md`），本端点不设平台管理员分支。

**删除语义（软删除，与主流社交产品一致）**：
- 评论**不物理删除**，标记为 `is_deleted=true`，`content` 清空（客户端展示"该评论已删除"占位）
- **删除顶层评论**：该评论软删，但其下**子评论保留**（仍挂载在该顶层评论下，`fetch_replies` 继续返回；顶层显示为已删除占位，回复正常展示）
- **删除回复**：仅该回复软删，不影响同层其它回复与顶层评论
- 设计目的：保留对话上下文完整性，避免"删一条顶层评论导致整串回复消失"

- **节点**: /api/v1

**Request**: `DeleteCommentRequest`
```typescript
interface DeleteCommentRequest {
  comment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_resource_owner） `10300`

---

### POST /api/v1/comments/fetch

获取指定瞬间的顶层评论列表。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/comments/fetch` 见 `api_spec_public.md`）
- **排序**: 评论列表默认按 `created_at` **时间倒序**（最新在前），`comment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchCommentsRequest`
```typescript
interface FetchCommentsRequest {
  moment_id: string;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<CommentItem>>`

**错误码**: `10300`

---

### POST /api/v1/comments/fetch_replies

获取指定评论的子评论列表。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/comments/fetch_replies` 见 `api_spec_public.md`）
- **排序**: 回复列表默认按 `created_at` **时间正序**（最早在前，便于对话阅读），`comment_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchCommentRepliesRequest`
```typescript
interface FetchCommentRepliesRequest {
  comment_id: string;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<CommentItem>>`

**错误码**: `10300`

---

### POST /api/v1/comments/open_the_door

获取单条评论详情（用于通知/举报等场景的评论定位）。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/comments/open_the_door` 见 `api_spec_public.md`）

**Request**: `OpenCommentDoorRequest`
```typescript
interface OpenCommentDoorRequest {
  comment_id: string;
}
```

**Response**: `ApiResponse<CommentItem>`

**错误码**: `10300`