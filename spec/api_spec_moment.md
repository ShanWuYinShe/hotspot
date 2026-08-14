# HotSpot API - 生活瞬间

> 本文档定义 HotSpot 生活瞬间 (`moment`) 相关的 RPC 接口。评论 (`comment`) 接口定义于 `api_spec_comment.md`。

## 模块信息

- **命名空间**: `moments`
- **基础路径**: `/api/v1/moments/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`
  - 搜索建议：`api_spec_discovery.md`
  - 匿名镜像版（/public/v1 同名端点）：`api_spec_public.md`

## 端点总览

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/moments/search` | 搜索瞬间（关键字/标签；双版本，匿名版在 /public/v1） | /api/v1 |
| `POST /api/v1/moments/open_the_door` | 瞬间详情（双版本） | /api/v1 |
| `POST /api/v1/moments/create` | 创建瞬间 | /api/v1 |
| `POST /api/v1/moments/update` | 编辑瞬间 | /api/v1 |
| `POST /api/v1/moments/delete` | 删除瞬间 | /api/v1 |
| `POST /api/v1/moments/set_favorite` | 设置收藏状态（两态统一） | /api/v1 |
| `POST /api/v1/moments/fetch_created` | 我创建的瞬间（拥有者视角） | /api/v1 |
| `POST /api/v1/moments/fetch_favorites` | 我的收藏 | /api/v1 |

> 表情反应（reaction）统一由 `POST /api/v1/reactions/set` 处理，反应列表统一由 `POST /api/v1/reactions/fetch` 查询（见 `api_spec_reaction.md`）。

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `MomentItem`（公开固定形状）
- `PrivacyLevel`
- `Location`
- `RelativeLocation`
- `GeoPoint`
- `PaginationParams`
- `UserInfo`

### 本模块类型

```typescript
// 拥有者视角扩展（仅拥有者端点返回：moments/fetch_created、moments/create、moments/update）
// 视角分离原则：privacy_level 属于拥有者数据，不在公开类型 MomentItem 中
interface OwnerMomentItem extends MomentItem {
  privacy_level: PrivacyLevel;   // 拥有者必见
}

// 收藏列表项：MomentItem 基础上附加收藏时间（排序键，随响应返回，客户端可展示"收藏于"时间）
interface FavoritedMomentItem extends MomentItem {
  favorited_at: string;   // 收藏时间（UTC ISO 8601）
}
```

## 端点详情

### POST /api/v1/moments/search

统一搜索生活瞬间（关键字/标签）。**不承担附近发现职责**——按位置发现瞬间的唯一入口为 `feed/nearby_moments`（见 `api_spec_discovery.md`）。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/moments/search` 见 `api_spec_public.md`）
- **排序**: 
  - `sort_by='relevance'`（默认）：按相关性优先；同分时按 `created_at` 倒序，再以 `moment_id DESC` 作为稳定 tie-breaker
  - `sort_by='newest'`：按发布时间倒序
  - `sort_by='popular'`：按互动热度排序（含时间衰减因子；具体衰减算法为服务端实现细节）
  - 一致性级别为弱一致

**Request**: `SearchMomentsRequest`
```typescript
interface SearchMomentsRequest {
  keyword: string | null;    // 关键字搜索（匹配描述等）
  tag: string | null;        // 标签筛选
  sort_by: 'relevance' | 'newest' | 'popular' | null;  // 默认: relevance
  pagination: PaginationParams | null;
}
// 至少提供 keyword、tag 中的一个；不满足时请求失败（10000）
// 搜索为全局匹配（不受位置过滤）；附近发现请使用 feed/nearby_moments
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

---

### POST /api/v1/moments/open_the_door

获取瞬间详情（登录视角 `MomentItem`，含查看者字段 `is_favorited`/`recommendation`；匿名版返回 `PublicMomentItem`）。

**隐私说明**：`privacy_level='private'` 的瞬间仅拥有者可见；非拥有者（含未登录用户）访问 private 瞬间统一返回 `10300`（按不存在处理，不通过 10200 暴露瞬间存在性）。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/moments/open_the_door` 见 `api_spec_public.md`）

**Request**: `OpenMomentDoorRequest`
```typescript
interface OpenMomentDoorRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse<MomentItem>`

**错误码**: `10300`

---

### POST /api/v1/moments/create

创建生活瞬间。返回完整的 `OwnerMomentItem`（拥有者视角，含 privacy_level），支持客户端乐观更新。

- **节点**: /api/v1
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateMomentRequest`
```typescript
interface CreateMomentRequest {
  description: string;
  location: Location;
  privacy_level: PrivacyLevel | null;  // 默认: 'public'
  tags: string[];
  image_file_ids: string[];     // 来自 media/complete_upload 返回的 file_id
  mentions: string[];
}
```

**Response**: `ApiResponse<OwnerMomentItem>`

**错误码**: `10300`（image_file_ids 中的媒体资源不存在或不属于当前用户） `10400`（tags 含被屏蔽标签）

---

### POST /api/v1/moments/update

编辑生活瞬间（仅作者）。

- **节点**: /api/v1

**Request**: `UpdateMomentRequest`
```typescript
interface UpdateMomentRequest {
  moment_id: string;
  description: string | null;
  location: Location | null;
  privacy_level: PrivacyLevel | null;
  tags: string[] | null;
  image_file_ids: string[] | null;
  mentions: string[] | null;
}
```

**Patch 语义**：未传字段保持不变；`tags: []` / `image_file_ids: []` / `mentions: []` 分别表示清空对应列表，传入非空数组表示替换。

**Response**: `ApiResponse<OwnerMomentItem>`（内容类更新返回更新后完整对象）

**错误码**: `10200`（not_resource_owner） `10300`

---

### POST /api/v1/moments/delete

删除生活瞬间（仅作者）。

**删除语义（软删除，与评论一致）**：
- 瞬间**不物理删除**，标记 `is_deleted=true`，`description` 清空（客户端展示"内容已删除"占位）
- **其下评论/反应数据保留**（不级联删除，保留对话上下文与审计）；瞬间不可见后，评论列表（`comments/fetch`）随之不可达（挂载瞬间按不存在处理）
- **可见性**：作者本人的 `moments/fetch_created`、收藏者的 `moments/fetch_favorites` 返回软删占位项（`is_deleted=true`、description 为空）；其余场景（`open_the_door`/Feed/搜索/评论挂载）按不存在处理（`10300`/过滤）
- 删除后推送 `MomentDeleted` WebSocket 事件（见 `api_spec_websocket.md`），客户端据此更新本地列表/收藏展示

- **节点**: /api/v1

**Request**: `DeleteMomentRequest`
```typescript
interface DeleteMomentRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_resource_owner） `10300`

---

### POST /api/v1/moments/set_favorite

设置瞬间收藏状态（两态统一端点）。收藏是 moment 专属的"我保存"概念（独立于 reaction 维度）。

- **节点**: /api/v1

**Request**: `FavoriteRequest`
```typescript
interface FavoriteRequest {
  moment_id: string;
  active: boolean;   // true=收藏，false=取消收藏
}
```

**Response**: `ApiResponse`

**错误码**: `10300`

---

### POST /api/v1/moments/fetch_created

我创建的瞬间列表（拥有者视角：含 `private` 瞬间与软删占位项）。

- **节点**: /api/v1
- **排序**: 默认按 `created_at` **时间倒序**（最新瞬间在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchCreatedMomentsRequest`
```typescript
interface FetchCreatedMomentsRequest {
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<OwnerMomentItem>>`

> 公开视角的「某用户发布的瞬间」使用 `users/fetch_moments`（`user_id` 必传，仅 `privacy_level='public'` 瞬间，见 `api_spec_user.md`）。

---

### POST /api/v1/moments/fetch_favorites

我的收藏列表（含被收藏后软删的瞬间占位项）。

- **节点**: /api/v1
- **排序**: 默认按 `favorited_at` **时间倒序**（最近收藏在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchFavoritesRequest`
```typescript
interface FetchFavoritesRequest {
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<FavoritedMomentItem>>`