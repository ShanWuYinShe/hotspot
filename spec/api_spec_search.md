# HotSpot API - 搜索与发现

> 本文档定义 HotSpot 全局搜索 (`search`) 相关的 RPC 接口。
> 管理员标签管理接口定义于 `api_spec_admin_moderation.md`。

## 模块信息

- **命名空间**: `search`
- **基础路径**: `/api/v1/search/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 |
|-----|------|------|

| `POST /api/v1/search/suggest` | 综合搜索建议 | 可选 |
## 类型定义

以下类型定义于 `api_spec_types.md`：
- `GeoPoint`
- `RelativeLocation`
- `UserInfo`
- `GroupInfo`
- `UserItem`
- `GroupItem`
- `TagItem`

## 端点详情

### POST /api/v1/search/suggest

综合搜索建议（标签、用户、群组、瞬间）。

- **认证**: 可选
- **排序**:
  - 完全匹配优先
  - 在匹配度相近时按距离优先，热度/活跃度作为辅助信号
  - 各类型结果内部以对应资源 ID（`user_id`/`group_id`/`moment_id`/`tag name`）作为稳定 tie-breaker；一致性级别为弱一致

**说明**：
- Token 可选；未登录时仅返回公开内容。
- 建议用于搜索框输入提示与全局发现入口。
- **LBS 优化**：`reference_point` 可选，不传时服务端按 `api_spec_types.md`「reference_point 解析规则」回退链解析，用于优先返回附近的匹配结果。
- **类型过滤**：通过 `types` 参数可限定返回特定类型的结果，替代各模块独立的 suggest 端点。

**Request**: `SearchSuggestRequest`
```typescript
interface SearchSuggestRequest {
  prefix: string;
  reference_point?: GeoPoint; // 浏览锚点位置（可选；不传时服务端按 api_spec_types.md「reference_point 解析规则」回退链解析）
  types?: SearchSuggestType[];  // 可选：限定返回的类型（如 ['user', 'group']）。未提供或空数组均表示返回所有类型。
  radius?: number;  // 搜索半径（米），约束见 api_spec_base.md 通用字段约束
}
```

**半径与定位说明**：
- `radius` 未提供时默认使用 `5_000` 米（5 km），并会被 `AppConfig.limits.search_radius_max` 上限裁剪。
- `reference_point` 为可选：不传时服务端按 `api_spec_types.md`「reference_point 解析规则」回退链解析（已登录用户最近位置 → `ClientConfig.defaults.default_location.point`）。

**Response**: `ApiResponse<SearchSuggestResponse>`

```typescript
// 建议项统一复用通用类型（TagItem/UserItem/GroupItem 定义于 api_spec_types.md），不重复定义同义结构
// - tag：TagItem（name + usage_count + trend?）
// - user：UserItem（@auth-only 字段按登录态返回，与全规范一致）
// - group：GroupItem
// - moment：MomentSuggestItem（基于 MomentSummary 的轻量扩展，仅用于建议场景）

// 瞬间建议项（基于 MomentSummary 的轻量扩展，仅用于建议场景；author 为发布者，location 为相对浏览锚点的位置）
// MomentSummary 定义于 api_spec_types.md（含 moment_id + description_preview + cover_url?）
interface MomentSuggestItem extends MomentSummary {
  author: UserInfo;
  location?: RelativeLocation;
}

// 搜索建议类型（支持按类型过滤）
type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';

type SearchSuggestItem =
  | { type: 'tag'; payload: TagItem }
  | { type: 'user'; payload: UserItem }
  | { type: 'group'; payload: GroupItem }
  | { type: 'moment'; payload: MomentSuggestItem };

interface SearchSuggestResponse {
  items: SearchSuggestItem[];
}
```

---

## 综合搜索

标签相关类型（`TagItem`/`TagTrend`）定义于 `api_spec_types.md`。综合搜索结果（用户 + 群组 + 瞬间）由客户端分别调用 `users/search`、`groups/search`、`moments/search` 拼装，各自独立分页，不提供聚合端点。
