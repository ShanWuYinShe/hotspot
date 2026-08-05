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
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`
  - 搜索建议：`api_spec_search.md`

## 端点总览

| 端点 | 说明 | 认证 |
|-----|------|------|

| `POST /api/v1/moments/search` | 搜索瞬间（支持关键字/标签/位置组合） | 可选 |
| `POST /api/v1/moments/get_detail` | 瞬间详情 | 可选 |
| `POST /api/v1/moments/create` | 创建瞬间 | 必须 |
| `POST /api/v1/moments/update` | 编辑瞬间 | 必须 |
| `POST /api/v1/moments/delete` | 删除瞬间 | 必须 |
| `POST /api/v1/moments/list_my` | 我发布的瞬间 | 必须 |
| `POST /api/v1/moments/list_favorites` | 我的收藏 | 必须 |
| `POST /api/v1/moments/favorite` | 设置收藏状态（两态统一） | 必须 |
| `POST /api/v1/comments/create` | 创建评论/回复 | 必须 |
| `POST /api/v1/comments/delete` | 删除评论 | 必须 |
| `POST /api/v1/comments/list` | 顶层评论列表 | 可选 |
| `POST /api/v1/comments/list_replies` | 子评论列表 | 可选 |
| `POST /api/v1/comments/get_detail` | 评论详情 | 可选 |
> 表情反应（reaction）统一由 `POST /api/v1/reactions/set` 处理，反应列表统一由 `POST /api/v1/reactions/list` 查询（见 `api_spec_reaction.md`）。

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `MomentItem`
- `PrivacyLevel`
- `Location`
- `RelativeLocation`
- `GeoPoint`
- `PaginationParams`
- `UserInfo`

## 生活瞬间 API

### 搜索与发现

说明：生活瞬间支持 `privacy_level` 控制可见性（`public` / `private`）。未登录用户仅能访问 `public` 的瞬间；登录态可访问自己的所有瞬间（含 `private`）+ 所有 `public` 瞬间。认证可选，字段可见性规则见 `api_spec_base.md`。

### POST /api/v1/moments/search

统一搜索生活瞬间。支持关键字、标签、位置范围三种条件的自由组合。

- **认证**: 可选
- **排序**:
  - `sort_by='relevance'`（默认）：按相关性优先；同分时按 `created_at` 倒序，再以 `moment_id ASC` 作为稳定 tie-breaker
  - `sort_by='distance'`：按距离由近到远（未提供 `radius` 时按默认 `5_000` 米计算）
  - `sort_by='newest'`：按发布时间倒序
  - `sort_by='popular'`：按互动热度排序（评论/表情反应等，含时间衰减因子，避免老内容长期占榜；具体衰减算法为服务端实现细节）
  - 一致性级别为弱一致（搜索场景允许短暂延迟）

**Request**: `SearchMomentsRequest`
```typescript
interface SearchMomentsRequest {
  keyword?: string;    // 关键字搜索（匹配描述等）
  tag?: string;        // 标签筛选
  radius?: number;     // 搜索半径（米），约束见 api_spec_base.md 字段校验约束
  sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';  // 默认: relevance
  reference_point?: GeoPoint; // 浏览锚点位置（可选；不传时服务端按 api_spec_types.md「reference_point 解析规则」回退链解析）
  pagination?: PaginationParams;
}
// 至少提供 keyword、tag、radius 中的一个；不满足时请求失败
// reference_point 不算搜索条件，仅用于距离计算和排序，不能单独作为搜索条件
// 提供 keyword/tag 但缺省 radius 时，服务端按默认半径计算
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

### POST /api/v1/moments/get_detail

获取瞬间详情。认证可选，字段可见性规则见 `api_spec_base.md`。

**隐私说明**：`privacy_level='private'` 的瞬间仅拥有者可见；非拥有者（含未登录用户）访问 private 瞬间统一返回 `10300`（按不存在处理，不通过 10200 暴露瞬间存在性，与 base.md 隐私规则一致）。

- **认证**: 可选

**Request**: `MomentOpenTheDoorRequest`
```typescript
interface MomentOpenTheDoorRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse<MomentItem>`

**错误码**: `10300`

### 瞬间操作

### POST /api/v1/moments/create

创建生活瞬间。返回完整的 `MomentItem`，支持客户端乐观更新（立即在 Feed/列表中展示）。

- **认证**: 必须
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateMomentRequest`
```typescript
interface CreateMomentRequest {
  description: string;
  location: Location;
  privacy_level?: PrivacyLevel;  // 默认: 'public'
  tags?: string[];
  image_file_ids?: string[];     // 来自 media/batch_complete_upload 返回的 file_id
  mentions?: string[];
}
```

**Response**: `ApiResponse<MomentItem>`

**错误码**: `10300`

### POST /api/v1/moments/update

编辑生活瞬间。

- **认证**: 必须

**Request**: `EditMomentRequest`
```typescript
interface EditMomentRequest {
  moment_id: string;
  description?: string;
  location?: Location;
  privacy_level?: PrivacyLevel;
  tags?: string[];
  image_file_ids?: string[];     // 来自 media/batch_complete_upload 返回的 file_id
  mentions?: string[];
}
```

**Patch 语义**：未传字段保持不变；`tags: []` / `image_file_ids: []` / `mentions: []` 分别表示清空对应列表，传入非空数组表示替换。

**Response**: `ApiResponse<MomentItem>`

**错误码**: `10200` `10300`

### POST /api/v1/moments/delete

删除生活瞬间。

- **认证**: 必须

**Request**: `DeleteMomentRequest`
```typescript
interface DeleteMomentRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

### 当前用户瞬间互动 API

> 表情反应（reaction）统一由 `POST /api/v1/reactions/set` 处理（见 `api_spec_reaction.md`）。本模块不再单独定义 like/unlike 端点。
> 收藏是 moment 专属的"我保存"概念（独立于 reaction 维度），保留为下方 `set_favorite` 单端点。

### POST /api/v1/moments/favorite

设置瞬间收藏状态（两态统一端点）。

- **认证**: 必须

**Request**: `FavoriteRequest`
```typescript
interface FavoriteRequest {
  moment_id: string;
  active: boolean;   // true=收藏，false=取消收藏
}
```

**Response**: `ApiResponse`

**错误码**: `10300`

### POST /api/v1/moments/list_my

我发布的瞬间列表。

- **认证**: 必须
- **排序**: 默认按 `created_at` **时间倒序**（最新瞬间在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MyMomentsRequest`
```typescript
interface MyMomentsRequest {
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

### POST /api/v1/moments/list_favorites

我的收藏列表。

- **认证**: 必须
- **排序**: 默认按 `favorited_at` **时间倒序**（最近收藏在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MyFavoritesRequest`
```typescript
interface MyFavoritesRequest {
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

> 瞬间反应列表由 `POST /api/v1/reactions/list` 查询（见 `api_spec_reaction.md`），传入 `{ target_type: 'moment', target_id: ... }` 即可。

---

## 评论 API

### 评论层级模型

评论采用**两级结构**（顶层评论 + 一级回复），不支持多层嵌套：

- **顶层评论**：`parent_id` 为空，直接挂载在瞬间下
- **回复**：`parent_id` 指向一条顶层评论，回复平铺在该顶层评论下
- **回复的回复**：若 `parent_id` 指向一条已有 `parent_id` 的评论（即指向另一条回复），服务端**自动将其挂载到对应的顶层评论下**（`parent_id` 重写为顶层评论 ID），保证数据库中回复始终只有一层
- `list_replies` 返回指定顶层评论下的所有回复（扁平列表），不存在递归嵌套

### POST /api/v1/comments/create

添加评论/回复。

- **认证**: 必须
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

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
  parent_id?: string;       // 父评论 ID（回复时存在）
  mentions?: string[];      // @ 提及的用户 ID 列表
  created_at: string;
  replies_count: number;    // 子评论数量
  reactions?: Reaction[];   // 表情反应聚合（含 reacted_by_me）
  is_deleted: boolean;      // 软删除标记；true 时 content 为空，客户端展示"该评论已删除"占位

  // 顶层评论列表场景（comments/list）返回前 N 条回复预览，消除展开回复的 N+1；
  // comments/list_replies 返回的回复项不返回此字段（避免递归嵌套）。
  // 条数由 AppConfig.response_defaults.comment_top_replies_count 控制（默认 3）
  top_replies?: CommentItem[];
}
```

**错误码**: `10200` `10300`

### POST /api/v1/comments/delete

删除评论。

**删除语义（软删除，与微博/掘金等主流社交产品一致）**：
- 评论**不物理删除**，标记为 `is_deleted=true`，`content` 清空（客户端展示"该评论已删除"占位）
- **删除顶层评论**：该评论软删，但其下**子评论保留**（仍挂载在该顶层评论下，`list_replies` 继续返回；顶层显示为已删除占位，回复正常展示）
- **删除回复**：仅该回复软删，不影响同层其它回复与顶层评论
- 设计目的：保留对话上下文完整性，避免"删一条顶层评论导致整串回复消失"
- 仅评论作者或系统管理员可删除（管理员删除记录审计日志）

- **认证**: 必须

**Request**: `DeleteCommentRequest`
```typescript
interface DeleteCommentRequest {
  comment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

### POST /api/v1/comments/list

获取指定瞬间的顶层评论列表。认证可选，字段可见性规则见 `api_spec_base.md`。

- **认证**: 可选
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

获取指定评论的子评论列表。认证可选，字段可见性规则见 `api_spec_base.md`。

- **认证**: 可选
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

> 评论的表情反应统一由 `POST /api/v1/reactions/set { target_type: 'comment', target_id: comment_id, ... }` 处理，本模块不再单独定义 like/unlike 端点。

> 评论反应列表由 `POST /api/v1/reactions/list` 查询（见 `api_spec_reaction.md`），传入 `{ target_type: 'comment', target_id: ... }` 即可。

### POST /api/v1/comments/get_detail

获取单条评论详情。

- **认证**: 可选

**Request**: `CommentOpenTheDoorRequest`
```typescript
interface CommentOpenTheDoorRequest {
  comment_id: string;
}
```

**Response**: `ApiResponse<CommentItem>`

**错误码**: `10300`
