---
alwaysApply: false
---
# HotSpot API - 搜索与发现

> 本文档定义 HotSpot 全局搜索 (`search`) 和标签 (`tags`) 相关的 RPC 接口。
> 旨在提供基于地理位置和语义的综合搜索与发现能力。
> 管理员标签管理接口定义于 `@spec/api_spec_admin_moderation.md`。

## 模块信息

- **命名空间**: `search`
- **基础路径**: `/api/v1/search/`
- **引用**: 
  - 基础规范：`@spec/api_spec_base.md`
  - 通用类型：`@spec/api_spec_types.md`
  - 错误码：`@spec/api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/search/suggest` | 综合搜索建议 | 可选 | 无 |

## 类型定义

以下类型定义于 `@spec/api_spec_types.md`：
- `GeoPoint`
- `RelativeLocation`
- `UserInfo`

## 端点详情

### POST /api/v1/search/suggest

综合搜索建议（标签、用户、群组、瞬间）。

- **认证**: 可选
- **幂等**: 无
- **排序**: 
  1. **完全匹配**优先。
  2. **距离优先**（在匹配度相近的情况下）。
  3. **热度/活跃度**作为辅助排序信号。
  - 各类型结果内部以对应资源 ID（`user_id`/`group_id`/`moment_id`/`tag name`）作为稳定 tie-breaker；一致性级别为弱一致（搜索场景允许短暂延迟）

**说明**：
- Token 可选；未登录时仅返回公开内容。
- 建议用于搜索框输入提示与全局发现入口。
- **LBS 优化**：必须提供 `reference_point`，用于优先返回附近的匹配结果。
- **类型过滤**：通过 `types` 参数可限定返回特定类型的结果，替代各模块独立的 suggest 端点。

**Request**: `SearchSuggestRequest`
```typescript
interface SearchSuggestRequest {
  prefix: string;
  reference_point: GeoPoint; // 必填，用于 LBS 优先排序（用户拒绝定位时使用 ClientConfig.defaults.default_location.point 作为 fallback）
  types?: SearchSuggestType[];  // 可选：限定返回的类型（如 ['user', 'group']）。未提供或空数组均表示返回所有类型。
  radius?: number;  // 搜索半径（米），约束见 @spec/api_spec_base.md 通用字段约束
}
```

**半径与定位说明**：
- `radius` 未提供时默认使用 `10_000` 米（10 km），并会被 `AppConfig.limits.search_radius_max` 上限裁剪。搜索建议使用较大半径（10 km）以提供更丰富的结果多样性，区别于 Feed 的 5 km 默认值（Feed 侧重附近内容密度）。
- 用户拒绝定位或无法获取实时位置时，客户端必须使用 `ClientConfig.defaults.default_location.point` 作为 `reference_point` fallback（此规则在 base 中已定义，此处再次强调）。

**Response**: `ApiResponse<SearchSuggestResponse>`

```typescript
// 标签建议项
interface TagSuggestItem {
  name: string;
  usage_count: number;
}

// 用户建议项
interface UserSuggestItem {
  user_id: string;
  nickname: string;
  avatar_url?: string;
  bio?: string;
  location?: RelativeLocation;
}

// 群组建议项
interface GroupSuggestItem {
  group_id: string;
  name: string;
  avatar_url?: string;
  description?: string;
  members_count: number;
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

## 标签相关类型

`TagItem`、`TagTrend` 已收归至 `@spec/api_spec_types.md`，本模块和其他模块统一引用。
