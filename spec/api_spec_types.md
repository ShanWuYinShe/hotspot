# HotSpot API - 通用类型定义

> 定义 HotSpot API 跨模块复用的基础值对象（Domain Primitives）和业务核心对象（Domain Entities）。
>
> 空值表达与视角分离遵循 `api_spec_base.md`「类型稳定性」（恒定 nullable 通则；视角差异由不同端点 + 扩展类型承载，注解已废除）。

## 契约引用

- 基础规范与约束：`api_spec_base.md`
- 错误码：`api_spec_errors.md`

---

## 通用类型定义 (Domain Primitives)

本节定义跨模块复用的基础值对象。

### 基础标识与枚举

```typescript
// 消息类型枚举
type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share' | 'system';

// 用户类型枚举（仅表达账号生命周期，不含权限角色）
// - 'temp': 临时用户（访客，未注册）
// - 'normal': 正式用户（已注册）
// 平台管理员身份由 RBAC 系统管理（见 api_spec_admin.md），不体现在 UserType 中
type UserType = 'temp' | 'normal';

// 隐私级别枚举（跨模块复用：用户设置、瞬间等）
// - 'public': 所有人可见（包括未登录用户）
// - 'private': 仅自己可见
type PrivacyLevel = 'public' | 'private';

// 群组角色（仅表达群组内角色；与平台管理员是完全不同的两套体系，见 api_spec_authz.md）
type GroupRole = 'owner' | 'admin' | 'member';

// === 客户端 ===
type ClientType = 'web' | 'app';

// === 设备标识（单设备单会话模型的基础原语，详见 api_spec_public.md「单设备单会话模型」）===
// 设备身份标识（UUID v4 字符串）。语义：一台逻辑设备 = 一个 device_id。
// - App：客户端首次启动生成并存入 Keychain（iOS）/ Keystore（Android），卸载重装后视为新设备；一经生成即稳定，禁止轮换
// - Web：由服务端通过独立 device_id Cookie 下发（非 HttpOnly，登出不清除；不同浏览器档案 = 不同 device_id）
type DeviceId = string;

// === Reaction（emoji 表情回应，统一用于 message/moment/comment 三类资源）===
// emoji 为 Unicode 码点字符串（如 "👍"），服务端按 AppConfig.limits.reaction_emojis 白名单校验
// 可用 emoji 集合对客户端透明，通过 system/client_config 下发
type ReactTarget = 'message' | 'moment' | 'comment';

interface Reaction {
  emoji: string;          // 表情符号
  count: number;          // 该表情的反应总数
  reacted_by_me: boolean; // 查看者是否已用此表情反应（恒定布尔：未反应时为 false）
}

// === 状态枚举 ===
type UserPresenceStatus = 'online' | 'offline';
type ReportStatus = 'pending' | 'resolved' | 'dismissed';

// === 业务操作类型 ===
type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';
type ReportAction = 'resolve' | 'dismiss';

// 举报原因枚举（用户从中选取；跨 report + admin 两模块）
type ReportReason =
  | 'spam'              // 垃圾信息/广告
  | 'harassment'        // 骚扰/欺凌
  | 'violence'          // 暴力/威胁
  | 'hate_speech'       // 仇恨言论
  | 'sexual_content'    // 色情/裸露内容
  | 'misinformation'    // 虚假信息
  | 'impersonation'     // 冒充他人
  | 'self_harm'         // 自伤/自杀相关
  | 'other';            // 其他（需填写 description）

// 群组准入方式（对加入者可见；叠加 requires_approval 决定是否需要审核）
type GroupJoinMethod = 'direct' | 'password' | 'invite_only';

// === 公告 ===
type AnnounceSeverity = 'info' | 'warning' | 'error';

// === 关注来源（用于来源归因展示）===
type FollowSource = 'moment' | 'search' | 'recommendation' | 'profile' | 'group';

// === 推荐理由类型（用于解释推荐来源）===
type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';

// 私聊权限（跨 user + chat 模块）
// 注：'friends' 表示"仅好友"，好友 = 双方互相关注的用户
type ChatPerm = 'everyone' | 'friends' | 'nobody';

// 通知类型（跨 notify + websocket + admin 模块）
type NotifyType =
  | 'new_follower'                // 被关注
  | 'mutual_follow'               // 形成互相关注
  | 'moment_commented'            // 瞬间被评论
  | 'comment_reply'               // 评论被回复
  | 'mentioned'                   // 被 @ 提及（瞬间/评论）
  | 'group_mentioned'             // 群聊消息中被 @
  | 'reaction_added'              // 内容被添加表情反应（自反应不落库）
  | 'group_invite_received'       // 收到入群邀请
  | 'group_application_received'  // 收到入群申请（群管理员视角）
  | 'group_application_accepted'  // 入群申请被通过
  | 'group_application_rejected'  // 入群申请被拒绝
  | 'report_resolved'             // 举报被处理
  | 'report_dismissed'            // 举报被驳回
  | 'system_announcement'         // 系统公告
  | 'nearby_new_moment';          // 附近有人发布新瞬间

// 通知分类（用于未读计数分组展示，见 api_spec_notify.md unread_count 返回的 by_type）
// 映射为契约级，新增 NotifyType 时必须同步归类：
// - social:        new_follower, mutual_follow
// - interaction:   moment_commented, comment_reply, mentioned, group_mentioned, reaction_added
// - group:         group_invite_received, group_application_received, group_application_accepted, group_application_rejected
// - report:        report_resolved, report_dismissed
// - nearby:        nearby_new_moment
// - system:        system_announcement
type NotifyCategory = 'social' | 'interaction' | 'group' | 'report' | 'nearby' | 'system';
```

### 位置与地理信息

**重要概念：用户真实位置 vs 浏览锚点位置**

- **用户真实位置**：通过「更新我的位置」接口更新；属于隐私数据，用于"被其他人发现我/展示我的大致位置"。
- **浏览锚点位置（reference_point）**：作为请求参数传入；用于"当前在看哪里/距离计算基准"。

核心原则：`distance` / `bearing_degrees` 必须由服务端基于 `reference_point` 计算并返回，客户端不得本地计算用于排序。

**隐私与精度策略（契约级）**：服务端可基于隐私设置与关系对位置/距离做降精度或匿名化处理；客户端只用于展示，不得将距离/方位作为精确导航或本地排序依据。

**请求位置要求（契约级）**：
- 创建类 Request 中出现 `Location` 字段时，必须为必填字段（如 `groups/create`、`moments/create`、`users/update_location`）
- 更新/Patch 类 Request 中 `Location` 可为可选字段（如 `groups/admin/update`、`moments/update`）
- 若用户拒绝定位或无法获取精确位置，客户端必须使用 `ClientConfig.defaults.default_location` 填充 `Location`

**`reference_point` 解析规则（契约级）**：

发现类与搜索类接口（`moments/fetch_nearby`、`search/*` 及各资源 `search` 端点）的 `reference_point` 字段统一为**可选**（`reference_point?: GeoPoint`）。当客户端不传时，服务端按以下回退链解析浏览锚点位置：

1. 请求显式传入的 `reference_point`（如地图浏览场景传地图中心点）
2. 最近一次 `users/update_location` 的位置（仅 `/api/v1` 登录请求可命中本环；未更新过位置时跳过，`/public/v1` 请求无登录身份恒跳过）
3. `ClientConfig.defaults.default_location.point`

不适用：`map/canvas` 用 `viewport.bounds` 锚定视口，无 `reference_point` 字段，不经此规则。

```typescript
// 基础坐标点 (经纬度)
interface GeoPoint {
  latitude: number;  // [-90, 90]
  longitude: number; // [-180, 180]
}

// 位置信息 (包含坐标和位置名称)
// 原始位置：不含 distance/bearing（距离/方位仅存在于 RelativeLocation）
// 用途：请求参数（如 groups/create、users/update_location）与配置/存储回显（GroupSettings、ClientDefaults、AccountData）
interface Location {
  point: GeoPoint;
  location_name: string; // 必填，0-100字符，无名称时使用 ClientConfig.defaults.default_location.location_name
}

// 相对位置 (用于所有 response 响应数据)
// distance 和 bearing_degrees 基于 reference_point 计算，且可能被服务端降精度
interface RelativeLocation {
  point: GeoPoint;
  location_name: string;       // 位置名称
  distance: number | null;     // 距离浏览锚点的距离 (米)；null = 无参考点/不可见
  bearing_degrees: number | null;  // 相对浏览锚点的方位角 [0, 360)；null = 无参考点/不可见
}
```

### 分页与元数据

```typescript
// 分页请求参数（请求侧可选字段保持 ?，语义为「未传使用默认」）
interface PaginationParams {
  limit?: number;  // default: 20, max: 100
  cursor?: string; // 不透明字符串，客户端只透传
}

// 通用分页容器（响应侧恒定 nullable：next_cursor 恒在，null = 没有更多数据）
interface PaginatedData<T> {
  items: T[];
  next_cursor: string | null;
}
```

### 媒体资源

```typescript
// 媒体资源基础信息（图片/视频等资源的展示与尺寸元数据）
// 用途：MomentItem.images、MessageMedia 等媒体字段的共享基础类型
interface MediaAsset {
  url: string;
  thumbnail_url: string | null;   // null = 无缩略图
  width: number | null;           // null = 未知
  height: number | null;          // null = 未知
  mime_type: string | null;       // MIME 类型字符串，如 "image/jpeg"；null = 未知
}

// 消息附件/媒体信息（在 MediaAsset 基础上补充文件元数据）
interface MessageMedia extends MediaAsset {
  file_name: string | null;
  file_size: number | null;
  duration: number | null;        // 音视频时长（秒）；null = 不适用
}
```

---

## 领域实体定义 (Domain Entities)

本节定义业务核心对象。遵循 **Info (嵌入用) -> Item (统一数据类型)** 的分层模式。查看者相关字段一律「恒定存在 + null 表达不适用」，不同视角由不同端点承载（`OwnerXxxItem`/`AdminXxxItem`/`XxxViewerState`）。

> **节点归属**：本节 Item 类型内嵌查看者字段（`relation` / `viewer_state` / `is_favorited` / `Reaction.reacted_by_me` 等），仅服务 `/api/v1`（含 groups/admin 子树）与 `/admin/api/v1` 登录视角；`/public/v1` 使用 `api_spec_public.md` 的 `PublicXxxItem` 对偶类型。

### 用户 (User)

**平台管理员可见性规则**：
- 平台管理员使用 /api/v1 时与普通用户**无区别**：使用相同端点、获得相同响应形态
- `UserInfo`/`UserItem` 不包含管理员标识，普通用户无法通过任何普通 API 判断对方是否为平台管理员
- 平台管理员专用字段仅在 `/admin/api/v1` 通过独立 Admin 类型返回（见 `api_spec_admin.md`）

**UserInfo**: 用户基础信息（用于嵌套在其他对象中）
```typescript
interface UserInfo {
  user_id: string;
  nickname: string;
  avatar_url: string | null;  // null = 未设置头像（客户端使用默认头像）；否则为可直接展示的 URL（头像类为 /public/v1/media/avatars/<file_id> 代理相对路径，见 api_spec_media.md「媒体读取」）
}
```

**UserStats**: 用户统计信息
```typescript
interface UserStats {
  moments_count: number;
  friends_count: number;  // 好友数量（好友 = 互相关注）
  followers_count: number;
  following_count: number;
  groups_count: number;
}
```

**UserItem**: 用户统一数据类型（用于列表、搜索、详情、Feed 等所有场景；形状恒定，不随查看者变化）
```typescript
interface UserItem {
  user: UserInfo;
  bio: string | null;
  tags: string[];

  // === 隐私受控字段（恒定 nullable：null = 对方隐私设置隐藏） ===
  presence_status: UserPresenceStatus | null;  // null = presence_visible=false 或未知
  last_online_at: string | null;               // 最后在线时间（UTC ISO 8601，精度截断到分钟）；null = 隐藏
  location: RelativeLocation | null;           // null = location_visible=false 或未设置
  stats: UserStats | null;                     // null = stats_visible=false 或端点不返回统计

  // === 查看者状态（恒定 nullable：null = 无关系/不适用；仅存在于登录视角类型，public 版无此字段） ===
  relation: Relationship | null;               // 当前用户与该用户的关系；无关系时为 null

  // 社交上下文（展示与当前用户的共同点）；详情/Feed 返回，搜索/列表为 null
  social_context: SocialContext | null;
}

// 社交上下文（查看者相关，恒定 nullable）
interface SocialContext {
  mutual_friends_count: number;                // 共同好友数
  mutual_friends_preview: UserInfo[];          // 前2-3个共同好友
  common_groups_count: number | null;
  common_tags: string[];
}
```

**Relationship**: 用户关系信息（查看者状态，恒定 nullable）
```typescript
interface Relationship {
  is_following: boolean;       // 我是否关注了对方
  is_followed_by: boolean;     // 对方是否关注了我
  is_blocked: boolean;         // 我是否拉黑了对方

  // 互相关注可由客户端计算：is_following && is_followed_by
  // 设计说明：不暴露 is_blocked_by（对方是否拉黑了我）——资料/关系字段不透露拉黑状态
  // 被拉黑后的被动体验由服务端隐式处理；主动交互明确告知：私聊发送失败返回 blocked_by_user
}
```

### 群组 (Group)

**GroupInfo**: 群组基础信息（用于嵌套在其他对象中）
```typescript
interface GroupInfo {
  group_id: string;
  name: string;
  avatar_url: string | null;
  members_count: number;
}
```

**GroupItem**: 群组统一公开类型（用于搜索、详情、Feed 等场景；形状恒定，对所有人一致）
```typescript
interface GroupItem {
  group: GroupInfo;
  description: string | null;
  tags: string[];

  // 加入门槛（对所有人可见——用户加入前必须知晓加入方式与是否需审核）
  join_method: GroupJoinMethod;      // 'direct' / 'password' / 'invite_only'
  requires_approval: boolean;        // 是否需管理员审核

  // 位置信息：恒定 nullable（null = 群组未设置位置）
  location: RelativeLocation | null;

  last_activity_at: string | null;   // 最近活跃时间；详情/列表返回，其他场景可为 null
  online_members_count: number | null; // 在线成员数（聚合计数；不暴露具体成员在线状态）
  next_activity_at: string | null;      // 下次活动预告（群主设置）；null = 无预告
  created_at: string | null;         // 详情返回；搜索/列表为 null
  owner_id: string | null;           // 详情返回；搜索/列表为 null

  // === 查看者状态（恒定 nullable：null = 非成员；仅存在于登录视角类型，public 版无此字段） ===
  viewer_state: GroupViewerState | null;   // 当前用户在该群的成员状态；非成员为 null
  friends_in_group: number | null;         // 好友中有多少在该群组内（Feed 场景返回，其余为 null）

  // 群管理字段（is_searchable / max_members / has_password）不在本类型——见 GroupSettings（groups/admin/settings）
}

// 查看者的群内状态（GroupItem.viewer_state 的取值类型；viewer_state 为 null 即非成员）
interface GroupViewerState {
  role: GroupRole;                 // 当前用户在群内的角色
  unread_count: number;            // 群聊未读数
  is_muted: boolean;
  muted_until: string | null;      // 禁言截止时间（UTC ISO 8601）；null = 未被禁言（被禁言者本人可见自己的禁言状态）
  is_pinned: boolean;
  joined_at: string;
}

// 群列表成员视角扩展（仅 groups/fetch_joined 返回；viewer_state 恒有值）
interface GroupJoinedItem extends GroupItem {
  viewer_state: GroupViewerState;          // 本端点必非 null
  last_message: MessageBrief | null;     // 最近消息预览；null = 无消息
}

// 群组设置（groups/admin/settings 响应；groups/create 与 groups/admin/update 请求中的同名设置字段与此同构）
interface GroupSettings {
  join_method: GroupJoinMethod;   // 默认 'direct'
  requires_approval: boolean;     // 默认 false
  has_password: boolean;          // 群是否设有密码
  is_searchable: boolean;
  members_searchable: boolean;    // 默认 true；是否允许非成员检索群成员列表
  max_members: number;
  location: Location;             // 直接映射，不经过距离/角度计算
}
```

**GroupAnnouncementItem**: 群公告列表项
```typescript
interface GroupAnnouncementItem {
  announcement_id: string;
  group_id: string;
  title: string;
  content: string;
  created_by: UserInfo;
  created_at: string;
  is_pinned: boolean;
}
```

**SystemAnnouncementItem**: 系统公告列表项
```typescript
interface SystemAnnouncementItem {
  announcement_id: string;
  title: string;
  content: string;
  severity: AnnounceSeverity;
  action_url: string | null;
  created_at: string;
}
```

### 生活瞬间 (Moment)

**MomentItem**: 瞬间统一公开类型（用于列表、搜索、详情、Feed 等所有场景；形状恒定）
```typescript
interface MomentItem {
  moment_id: string;
  description: string;         // 软删除时为空字符串，客户端展示"内容已删除"占位
  images: MediaAsset[];        // 空数组 = 无图片
  tags: string[];
  mentions: string[];
  created_at: string;
  comments_count: number;
  shares_count: number;
  reactions: Reaction[];       // 表情反应聚合（空数组 = 无反应）
  user: UserInfo;
  is_deleted: boolean;         // 软删除标记

  // 位置信息（恒定 nullable：null = 隐私隐藏/未设置）
  location: RelativeLocation | null;

  // === 查看者布尔（恒定布尔） ===
  is_favorited: boolean;       // 当前用户是否收藏；未收藏为 false

  // 查看者→作者的双向关注关系（**查看者字段，恒定 nullable**）：
  // null = 匿名/未登录，或作者 id 缺失。登录视角下反馈"这张卡是不是我关注的人发的"。
  // 注意：不内嵌到 user（UserInfo 是跨模块共享形状，含查看者字段会污染访客版与其它模块响应）。
  author_relation: { is_following: boolean; is_followed_by: boolean } | null;

  // Feed 场景返回（推荐理由）；非 Feed 场景为 null
  recommendation: RecommendationType | null;

  // privacy_level 不在本类型——拥有者视角见 OwnerMomentItem（moments/fetch_created 等拥有者端点）
}
```

**MomentSummary**: 瞬间摘要（用于通知、WebSocket 推送等轻量展示场景）
```typescript
interface MomentSummary {
  moment_id: string;
  description_preview: string;  // 截断的描述文本（最多 100 字符）
  cover_url: string | null;     // 首张图片缩略图 URL；null = 无图片
}
```

### 评论 (Comment)

**CommentItem**: 评论统一类型（跨 comment + reaction + notify 模块，定义于此）
```typescript
interface CommentItem {
  comment_id: string;
  user: UserInfo;
  content: string;             // 软删除时为空，客户端展示"该评论已删除"占位
  parent_id: string | null;    // 父评论 ID；null = 顶层评论
  mentions: string[];
  created_at: string;
  replies_count: number;       // 子评论数量
  reactions: Reaction[];       // 表情反应聚合（空数组 = 无反应）
  is_deleted: boolean;         // 软删除标记

  // 顶层评论列表场景（comments/fetch）返回前 N 条回复预览；其他端点为空数组
  // 条数由 AppConfig.response_defaults.comment_top_replies_count 控制（默认 3）
  top_replies: CommentItem[];
}
```

### 反应 (Reaction)

**ReactionItem**: 反应者列表项（用于 `reactions/fetch`）
```typescript
interface ReactionItem {
  user: UserInfo;
  emoji: string;        // 该用户使用的表情
  reacted_at: string;   // UTC ISO 8601
}
```

```typescript
// 批量用户解析结果（users/batch_fetch 响应；跨节点共享——无查看者字段的纯 UserInfo 映射，
// api 版与 public 版共用，见 api_spec_user.md / api_spec_public.md）
interface UsersBatchResult {
  users: Record<string, UserInfo>;  // user_id → UserInfo；不存在的 user_id 不出现在 map 中
}
```

### 消息 (Message)

**MessageBrief**: 消息摘要快照（`reply_to` 回复引用与 `last_message` 会话最近消息预览共用；完整消息项见 `api_spec_chat.md`）
```typescript
interface MessageBrief {
  message_id: string;
  type: MessageType;          // 消息类型（决定前端渲染方式）
  sender: UserInfo;
  content: string | null;     // 文本预览（仅 text 类型有意义）
  share_code: string | null;  // 若为 share 类型消息，包含该 share_code
  sent_at: string;
}
```

私聊/群聊消息项类型定义在模块文档中：`api_spec_chat.md`。

### 标签 (Tag)

```typescript
// 标签项（搜索建议等模块共用；usage_count 足以支撑热度排序，不设趋势字段）
interface TagItem {
  name: string;
  usage_count: number;
}
```

### 举报 (Report)

```typescript
// 举报项（report + admin 两模块共用：AdminReportItem extends ReportItem）
interface ReportItem {
  report_id: string;
  target_type: ReportTargetType;
  target_id: string;
  reason: ReportReason;
  description: string | null;
  status: ReportStatus;
  created_at: string;
  handled_at: string | null;
  message_snapshot: ReportSnapshot | null;  // 仅 target_type='message'

  // 处理结果摘要（恒定 nullable：null = 尚未处理）。reports/fetch 天生只有举报人本人调用，
  // 故 resolution 在客户端类型中恒可返回；不暴露处理平台管理员身份（隐私）
  resolution: ReportResolution | null;
}

// 处理结果摘要（脱敏，不含处理者身份）
interface ReportResolution {
  action: ReportAction;
  resolved_at: string;
  note: string | null;   // 可选的处理说明（脱敏后）
}

// 消息举报快照（隐私边界设计：平台管理员基于快照审核，不直接查询私聊消息库）
// 生成方（契约级）：由服务端在举报受理时基于消息真值生成并冻结——客户端不提交快照（防伪造/篡改证据链），
// 审核方信任的是服务端快照而非举报人自述
interface ReportSnapshot {
  chat_type: 'private' | 'group';   // 会话类型
  target_chat_id: string;           // 会话目标 ID：private 会话 = 对方 user_id，group 会话 = group_id
  message_type: MessageType;
  snapshot_content: string | null;  // 文本内容（text 类型）；非文本类型为 null
  snapshot_media: MessageMedia | null; // 媒体引用（图片/视频等）；文本类型为 null
  sent_at: string;
}
```

### 用户设置 (UserSettings)

```typescript
// 用户设置（user + websocket 两模块共用；ChatPerm 定义于本文件）
interface UserSettings {
  privacy_level: PrivacyLevel;
  location_visible: boolean;          // false 时，UserItem.location 恒为 null
  presence_visible: boolean;          // false 时 presence_status/last_online_at 恒为 null
  social_lists_visible: boolean;      // 好友/关注/粉丝列表是否公开（false 时列表端点 10200+privacy_setting）
  groups_visible: boolean;            // 群组列表是否公开
  reacted_moments_visible: boolean;   // 反应过的瞬间列表是否公开
  stats_visible: boolean;             // 统计信息是否公开（false 时 UserItem.stats 恒为 null）
  allow_follow: boolean;
  allow_private_chat: ChatPerm;
  searchable: boolean;
  phone_searchable: boolean;
  nickname_searchable: boolean;
  location_searchable: boolean;
  nearby_moment_notify: boolean;      // 是否接收"附近新瞬间"推送通知；默认 false
}
```

### 应用配置 (AppConfig 相关)

> 以下配置类型由 `system`（客户端可见投影 `ClientConfig`）与 `admin`（完整服务端配置 `AppConfig`）共用。

```typescript
// 功能开关（客户端可见，由平台管理员配置）
interface FeatureSwitches {
  enable_public_registration: boolean;
  enable_guest_register: boolean;
  enable_moments: boolean;
  enable_nearby_people: boolean;
}

// 业务限额（客户端可见，用于前端校验/提示）
interface AppLimits {
  max_group_members: number;
  max_moment_tags: number;
  max_moment_images: number;
  moment_description_max_length: number;
  comment_max_length: number;
  search_radius_max: number;
  max_chat_files: number;
  message_recall_window_seconds: number;  // 消息撤回时间窗（秒）；0 表示不允许撤回
  stranger_message_daily_quota: number;   // 每日可向陌生人发送的私聊消息上限；0 表示禁止
  reaction_emojis: string[];              // 允许的表情反应 emoji 白名单
  group_invite_ttl_hours: number;         // 群邀请有效期（小时）；0 表示永不过期
  group_application_ttl_hours: number;    // 群申请有效期（小时）；0 表示永不过期
}

// 联系/合规信息（客户端可见）
interface ContactInfo {
  support_email: string;
  privacy_policy_url: string;
  terms_of_service_url: string;
  official_website_url: string;
}
```

### 客户端配置与版本元信息（system 域，跨节点共享）

> 以下类型无查看者字段，被 `/public/v1/system/*` 与 `/api/v1/system/bootstrap` 共同引用，故定义于共享层。

```typescript
// API 版本状态
type ApiVersionStatus =
  | 'active'      // 正常使用中
  | 'deprecated'  // 已废弃，但仍可用
  | 'sunset';     // 即将下线或已停止

// API 版本元信息
interface ApiVersionMeta {
  version: string;                    // 当前 API 版本号，如 "v1"
  status: ApiVersionStatus;
  deprecated_at: string | null;       // 进入废弃状态时间（UTC ISO 8601）
  sunset_at: string | null;           // 计划下线时间（UTC ISO 8601）
  message: string | null;             // 面向客户端的提示信息
  recommended_version: string | null; // 推荐升级的版本号（如 "v2"）
}

// 系统健康状态
type SystemHealthStatus = 'healthy' | 'degraded' | 'unhealthy';

interface HealthStatus {
  status: SystemHealthStatus;
  timestamp: string;
  // 构建期注入的版本指纹（-ldflags 注入；本地裸构建为 version="dev"、其余空串）。
  // 用途：线上排障时核对该实例运行的源码版本，commit 为精确回溯锚点。
  version: string;      // git describe --tags --always，如 "v1.0.0" / "v1.0.0-2-g1a2b3c4"
  commit: string;       // 完整 git commit sha
  build_time: string;   // 镜像构建时刻（UTC RFC3339）
}

// 客户端公开配置（AppConfig 的可见投影；客户端启动时获取并缓存）
interface ClientConfig {
  features: FeatureSwitches;
  limits: AppLimits;
  contact: ContactInfo;
  defaults: ClientDefaults;
}

// 客户端可见的默认值（由平台管理员在后台配置，见 api_spec_admin.md 中 ContentDefaults）
interface ClientDefaults {
  default_location: Location;        // 默认位置（必须满足 Location 约束）
  default_user_avatar_url: string;   // 默认用户头像 URL
  default_group_avatar_url: string;  // 默认群组头像 URL
}
```

### 分享载荷与摘要（share 域，跨节点共享）

> 以下类型无查看者字段，被 `/public/v1/share/open_the_door` 与 `/api/v1/share/*` 共同引用，故定义于共享层。`ShareCreated`/`ShareListItem` 仅登录端点使用，定义于 `api_spec_share.md`。

```typescript
// 分享目标类型
type ShareType = 'user' | 'group' | 'moment' | 'location';

interface UserSharePayload {
  shared_user_id: string;
}

interface GroupSharePayload {
  shared_group_id: string;
}

interface MomentSharePayload {
  shared_moment_id: string;
}

interface LocationSharePayload {
  shared_location: Location;
}

type SharePayload =
  | UserSharePayload
  | GroupSharePayload
  | MomentSharePayload
  | LocationSharePayload;

interface ShareBase {
  share_id: string;
  share_code: string;
  share_type: ShareType;
  creator: UserInfo;
  created_at: string;
  expires_at: string | null;
}

// 展示用摘要：分享存在但目标资源不可访问时（如目标被删除/隐私变更）不返回 preview
interface SharePreview {
  title: string;
  subtitle: string | null;
  cover_url: string | null;
}
```

### 搜索建议（discovery 域，跨节点共享）

> 以下类型无查看者字段。search 为双版本端点（双方主体类型 `SearchSuggestResponse`/`PublicSearchSuggestResponse` 分别定义于 `api_spec_discovery.md`/`api_spec_public.md`）。

```typescript
// === 搜索建议 ===

// 搜索建议类型（支持按类型过滤）
type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';

// 瞬间建议项（基于 MomentSummary 的轻量扩展，仅用于建议场景）
interface MomentSuggestItem extends MomentSummary {
  author: UserInfo;
  location: RelativeLocation | null;
}
```