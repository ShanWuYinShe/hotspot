---
alwaysApply: false
---
# HotSpot API - 生活瞬间与评论

> 本文档定义 HotSpot 生活瞬间 (`moment`) 和评论 (`comment`) 相关的 RPC 接口。

## 模块信息

- **命名空间**:
  - `moment`
  - `comment`
- **基础路径**:
  - `/api/v1/moments/`
  - `/api/v1/comments/`
- **引用**:
  - 基础规范：`@spec/api_spec_base.md`
  - 通用类型：`@spec/api_spec_types.md`
  - 错误码：`@spec/api_spec_errors.md`
  - 搜索建议：`@spec/api_spec_search.md`（per-module suggest 已移除，使用全局 `search/suggest`）

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/moments/search` | 搜索瞬间（支持关键字/标签/位置组合） | 可选 | 无 |
| `POST /api/v1/moments/open_the_door` | 瞬间详情 | 可选 | 无 |
| `POST /api/v1/moments/create` | 创建瞬间 | 必须 | MUST |
| `POST /api/v1/moments/update` | 编辑瞬间 | 必须 | 幂等语义 |
| `POST /api/v1/moments/delete` | 删除瞬间 | 必须 | 幂等语义 |
| `POST /api/v1/moments/me/list` | 我发布的瞬间 | 必须 | 无 |
| `POST /api/v1/moments/me/list_favorites` | 我的收藏 | 必须 | 无 |
| `POST /api/v1/moments/me/like` | 点赞 | 必须 | 幂等语义 |
| `POST /api/v1/moments/me/unlike` | 取消点赞 | 必须 | 幂等语义 |
| `POST /api/v1/moments/me/favorite` | 收藏 | 必须 | 幂等语义 |
| `POST /api/v1/moments/me/unfavorite` | 取消收藏 | 必须 | 幂等语义 |
| `POST /api/v1/moments/list_likes` | 瞬间点赞列表 | 可选 | 无 |
| `POST /api/v1/comments/create` | 创建评论/回复 | 必须 | MUST |
| `POST /api/v1/comments/delete` | 删除评论 | 必须 | 幂等语义 |
| `POST /api/v1/comments/list` | 顶层评论列表 | 可选 | 无 |
| `POST /api/v1/comments/list_replies` | 子评论列表 | 可选 | 无 |
| `POST /api/v1/comments/me/like` | 点赞评论 | 必须 | 幂等语义 |
| `POST /api/v1/comments/me/unlike` | 取消点赞评论 | 必须 | 幂等语义 |
| `POST /api/v1/comments/list_likes` | 评论点赞列表 | 可选 | 无 |

## 类型定义

以下类型定义于 `@spec/api_spec_types.md`：
- `MomentItem`
- `PrivacyLevel`
- `Location`
- `RelativeLocation`
- `GeoPoint`
- `PaginationParams`
- `UserInfo`

## 生活瞬间 API

### 搜索与发现

说明：生活瞬间支持 `privacy_level` 控制可见性（`public` / `private`）。未登录用户仅能访问 `public` 的瞬间；登录态可访问自己的所有瞬间（含 `private`）+ 所有 `public` 瞬间。认证可选，字段可见性规则见 `@spec/api_spec_base.md`。

### POST /api/v1/moments/search

统一搜索生活瞬间。支持关键字、标签、位置范围三种条件的自由组合。

- **认证**: 可选
- **幂等**: 无
- **排序**: 默认 `sort_by='relevance'`：按相关性优先排序，同分时以 `created_at` 倒序作为第二排序，再以 `moment_id ASC` 作为最终稳定 tie-breaker；`sort_by='distance'`：按距离由近到远排序（需要 `radius`）；`sort_by='newest'`：按发布时间倒序；`sort_by='popular'`：按点赞/评论热度排序。一致性级别为弱一致（搜索场景允许短暂延迟）

**Request**: `SearchMomentsRequest`
```typescript
interface SearchMomentsRequest {
  keyword?: string;    // 关键字搜索（匹配描述等）
  tag?: string;        // 标签筛选
  radius?: number;     // 位置范围（米），100-50000，配合 reference_point 使用；未提供时不限定地理范围（仅按其他条件匹配）
  sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';  // 默认: relevance
  reference_point: GeoPoint; // 浏览锚点位置（客户端必须传入有效坐标，拒绝定位时使用 fallback）
  pagination?: PaginationParams;
}
// 至少提供 keyword、tag、radius 中的一个；不满足时返回 10000（参数错误）
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10000`

### POST /api/v1/moments/open_the_door

获取瞬间详情。认证可选，字段可见性规则见 `@spec/api_spec_base.md`。

- **认证**: 可选
- **幂等**: 无

**Request**: `MomentOpenTheDoorRequest`
```typescript
interface MomentOpenTheDoorRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse<MomentItem>`

**错误码**: `10000` `10300`

### 瞬间操作

### POST /api/v1/moments/create

创建生活瞬间。返回完整的 `MomentItem`，支持客户端乐观更新（立即在 Feed/列表中展示）。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateMomentRequest`
```typescript
interface CreateMomentRequest {
  description: string;
  location: Location;
  privacy_level?: PrivacyLevel;  // 默认: 'public'
  tags?: string[];
  image_file_ids?: string[];
  mentions?: string[];
}
```

**Response**: `ApiResponse<MomentItem>`

**错误码**: `10000` `10300`

### POST /api/v1/moments/update

编辑生活瞬间。

- **认证**: 必须
- **幂等**: 幂等语义

**Request**: `EditMomentRequest`
```typescript
interface EditMomentRequest {
  moment_id: string;
  description?: string;
  location?: Location;
  privacy_level?: PrivacyLevel;
  tags?: string[];
  image_file_ids?: string[];
  mentions?: string[];
}
```

**Response**: `ApiResponse<MomentItem>`

**错误码**: `10000` `10200` `10300`

### POST /api/v1/moments/delete

删除生活瞬间。

- **认证**: 必须
- **幂等**: 幂等语义

**Request**: `DeleteMomentRequest`
```typescript
interface DeleteMomentRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`

### 当前用户瞬间互动 API（/me/ 路径）

### POST /api/v1/moments/me/like

点赞生活瞬间。重复点赞返回成功（不报错），客户端无需担心重复操作。

- **认证**: 必须
- **幂等**: 幂等语义

**Request**: `LikeMomentRequest`
```typescript
interface LikeMomentRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

### POST /api/v1/moments/me/unlike

取消点赞生活瞬间。重复取消点赞返回成功（不报错），客户端无需担心重复操作。

- **认证**: 必须
- **幂等**: 幂等语义

**Request**: `UnlikeMomentRequest`
```typescript
interface UnlikeMomentRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

### POST /api/v1/moments/me/favorite

收藏瞬间。重复收藏返回成功（不报错），客户端无需担心重复操作。

- **认证**: 必须
- **幂等**: 幂等语义

**Request**: `FavoriteMomentRequest`
```typescript
interface FavoriteMomentRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

### POST /api/v1/moments/me/unfavorite

取消收藏瞬间。重复取消收藏返回成功（不报错），客户端无需担心重复操作。

- **认证**: 必须
- **幂等**: 幂等语义

**Request**: `UnfavoriteMomentRequest`
```typescript
interface UnfavoriteMomentRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

### POST /api/v1/moments/me/list

我发布的瞬间列表。

- **认证**: 必须
- **幂等**: 无
- **排序**: 默认按 `created_at` **时间倒序**（最新瞬间在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MyMomentsRequest`
```typescript
interface MyMomentsRequest {
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10000`

### POST /api/v1/moments/me/list_favorites

我的收藏列表。

- **认证**: 必须
- **幂等**: 无
- **排序**: 默认按 `favorited_at` **时间倒序**（最近收藏在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MyFavoritesRequest`
```typescript
interface MyFavoritesRequest {
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10000`

### POST /api/v1/moments/list_likes

获取生活瞬间点赞列表。

- **认证**: 可选
- **幂等**: 无
- **排序**: 默认按 `liked_at` **时间倒序**（最近点赞在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MomentLikesRequest`
```typescript
interface MomentLikesRequest {
  moment_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<LikeItem>>`（`LikeItem` 定义于 `@spec/api_spec_types.md`）

**错误码**: `10300`

---

## 评论与点赞 API

### 评论层级模型

评论采用**两级结构**（顶层评论 + 一级回复），不支持多层嵌套：

- **顶层评论**：`parent_id` 为空，直接挂载在瞬间下
- **回复**：`parent_id` 指向一条顶层评论，回复平铺在该顶层评论下
- **回复的回复**：若 `parent_id` 指向一条已有 `parent_id` 的评论（即指向另一条回复），服务端**自动将其挂载到对应的顶层评论下**（`parent_id` 重写为顶层评论 ID），保证数据库中回复始终只有一层
- `list_replies` 返回指定顶层评论下的所有回复（扁平列表），不存在递归嵌套

### POST /api/v1/comments/create

添加评论/回复。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateCommentRequest`
```typescript
interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id?: string;    // 回复的目标评论 ID（不传则为顶层评论；若指向二级回复，服务端自动挂载到其所属顶层评论）
  mentions?: string[];
}
```

**Response**: `ApiResponse<CommentItem>`

```typescript
interface CommentItem {
  comment_id: string;
  user: UserInfo;
  content: string;
  parent_id?: string;       // 父评论ID（回复时存在）
  mentions?: string[];      // @ 提及的用户 ID 列表
  created_at: string;
  likes_count: number;
  replies_count: number;    // 子评论数量
  
  // === @auth-only 以下字段仅登录态返回 ===
  is_liked?: boolean;       // 当前用户是否已点赞该评论
}
```

**错误码**: `10000` `10200` `10300`

### POST /api/v1/comments/delete

删除评论。

- **认证**: 必须
- **幂等**: 幂等语义

**Request**: `DeleteCommentRequest`
```typescript
interface DeleteCommentRequest {
  comment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`

### POST /api/v1/comments/list

获取指定瞬间的顶层评论列表。认证可选，字段可见性规则见 `@spec/api_spec_base.md`。

- **认证**: 可选
- **幂等**: 无
- **排序**: 评论列表默认按 `created_at` **时间倒序**（最新在前），`comment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `CommentsRequest`
```typescript
interface CommentsRequest {
  moment_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<CommentItem>>`

**错误码**: `10300`

### POST /api/v1/comments/list_replies

获取指定评论的子评论列表。认证可选，字段可见性规则见 `@spec/api_spec_base.md`。

- **认证**: 可选
- **幂等**: 无
- **排序**: 回复列表默认按 `created_at` **时间正序**（最早在前，便于对话阅读），`comment_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `CommentRepliesRequest`
```typescript
interface CommentRepliesRequest {
  comment_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<CommentItem>>`

**错误码**: `10300`

### POST /api/v1/comments/me/like

点赞评论。重复点赞返回成功（不报错），客户端无需担心重复操作。

- **认证**: 必须
- **幂等**: 幂等语义

**Request**: `LikeCommentRequest`
```typescript
interface LikeCommentRequest {
  comment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

### POST /api/v1/comments/me/unlike

取消点赞评论。重复取消点赞返回成功（不报错），客户端无需担心重复操作。

- **认证**: 必须
- **幂等**: 幂等语义

**Request**: `UnlikeCommentRequest`
```typescript
interface UnlikeCommentRequest {
  comment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

### POST /api/v1/comments/list_likes

获取评论点赞列表。

- **认证**: 可选
- **幂等**: 无
- **排序**: 点赞列表默认按 `liked_at` **时间倒序**（最近点赞在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `CommentLikesRequest`
```typescript
interface CommentLikesRequest {
  comment_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<LikeItem>>`（`LikeItem` 定义于 `@spec/api_spec_types.md`）

**错误码**: `10300`
