# HotSpot API - 搜索与发现

> 本文档定义 HotSpot 全局搜索 (`search`) 和标签 (`tags`) 相关的 RPC 接口。
> 旨在提供基于地理位置和语义的综合搜索与发现能力。
> 管理员标签管理接口定义于 `api_spec_admin_moderation.md`。

## 模块信息

- **命名空间**: `search`
- **基础路径**: `/api/v1/search/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/search/suggest` | 综合搜索建议 | 可选 | 无 |
| `POST /api/v1/search/all` | 综合搜索（用户+群组+瞬间首屏聚合） | 可选 | 无 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `GeoPoint`
- `RelativeLocation`
- `UserInfo`
- `GroupInfo`

## 端点详情

### POST /api/v1/search/suggest

综合搜索建议（标签、用户、群组、瞬间）。

- **认证**: 可选
- **幂等**: 无
- **排序**:
  - 完全匹配优先
  - 在匹配度相近时按距离优先，热度/活跃度作为辅助信号
  - 各类型结果内部以对应资源 ID（`user_id`/`group_id`/`moment_id`/`tag name`）作为稳定 tie-breaker；一致性级别为弱一致

**说明**：
- Token 可选；未登录时仅返回公开内容。
- 建议用于搜索框输入提示与全局发现入口。
- **LBS 优化**：必须提供 `reference_point`，用于优先返回附近的匹配结果。
- **类型过滤**：通过 `types` 参数可限定返回特定类型的结果，替代各模块独立的 suggest 端点。

**Request**: `SearchSuggestRequest`
```typescript
interface SearchSuggestRequest {
  prefix: string;
  reference_point: GeoPoint; // 浏览锚点位置（必填；拒绝定位时使用 ClientConfig.defaults.default_location.point）
  types?: SearchSuggestType[];  // 可选：限定返回的类型（如 ['user', 'group']）。未提供或空数组均表示返回所有类型。
  radius?: number;  // 搜索半径（米），约束见 api_spec_base.md 通用字段约束
}
```

**半径与定位说明**：
- `radius` 未提供时默认使用 `5_000` 米（5 km），并会被 `AppConfig.limits.search_radius_max` 上限裁剪。
- 用户拒绝定位或无法获取实时位置时，客户端必须使用 `ClientConfig.defaults.default_location.point` 作为 `reference_point` fallback（参见 `api_spec_types.md` 请求位置要求）。

**Response**: `ApiResponse<SearchSuggestResponse>`

```typescript
// 标签建议项
interface TagSuggestItem {
  name: string;
  usage_count: number;
}

// 用户建议项（统一使用 UserInfo 嵌套，与 spec 全局类型约定一致）
interface UserSuggestItem {
  user: UserInfo;
  bio?: string;
  location?: RelativeLocation;
}

// 群组建议项（统一使用 GroupInfo 嵌套）
interface GroupSuggestItem {
  group: GroupInfo;
  description?: string;
  location?: RelativeLocation;
}

// 瞬间建议项
interface MomentSuggestItem {
  moment_id: string;
  description_preview: string;
  author: UserInfo;
  location?: RelativeLocation;
}

// 搜索建议类型（支持按类型过滤）
type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';

type SearchSuggestItem =
  | { type: 'tag'; payload: TagSuggestItem }
  | { type: 'user'; payload: UserSuggestItem }
  | { type: 'group'; payload: GroupSuggestItem }
  | { type: 'moment'; payload: MomentSuggestItem };

interface SearchSuggestResponse {
  items: SearchSuggestItem[];
}
```

**错误码**: `10000`

---

### POST /api/v1/search/all

综合搜索：一次请求返回用户、群组、瞬间三个分类的首屏结果。用于搜索结果页的 tab 切换 / 合并展示，消除分别调用 `users/search`、`groups/search`、`moments/search` 的三次往返。

- **认证**: 可选
- **幂等**: 无
- **排序**: 各分类内部排序与对应资源 `search` 接口一致（默认按相关性）

**设计要点**：
- 聚焦 `keyword` 搜索，不支持 `tag` / `radius` 组合（那是各资源 `search` 的能力，综合搜索负责首屏聚合）
- 各分类条数受 `AppConfig.response_defaults.search_all_count_per_type` 控制（默认 10）
- `pagination` 三类共用同一请求参数；响应中各分类的 `next_cursor` **各自独立**。客户端翻页某个分类时，应切换到该分类的独立 `search` 端点继续分页
- `reference_point` 可选——不传时服务端按回退链解析（见 `api_spec_types.md`「`reference_point` 解析规则」）

**Request**: `SearchAllRequest`
```typescript
interface SearchAllRequest {
  keyword: string;
  reference_point?: GeoPoint;   // 可选；不传时服务端按回退链解析（见 api_spec_types.md）
  pagination?: PaginationParams; // 三类共用；响应的 next_cursor 各自独立
}
```

**Response**: `ApiResponse<SearchAllResult>`
```typescript
interface SearchAllResult {
  users: PaginatedData<UserItem>;
  groups: PaginatedData<GroupItem>;
  moments: PaginatedData<MomentItem>;
}
```

**错误码**: `10000`

---

## 标签相关类型

`TagItem`、`TagTrend` 已收归至 `api_spec_types.md`，本模块和其他模块统一引用。
