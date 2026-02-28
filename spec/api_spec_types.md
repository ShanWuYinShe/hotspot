---
alwaysApply: false
---
# HotSpot API - 通用类型定义

> 定义 HotSpot API 跨模块复用的基础值对象（Domain Primitives）和业务核心对象（Domain Entities）。

## 契约引用

- 基础规范与约束：`@spec/api_spec_base.md`
- 错误码：`@spec/api_spec_errors.md`

---

## 通用类型定义 (Domain Primitives)

本节定义跨模块复用的基础值对象。

### 基础标识与枚举

```typescript
// 消息类型枚举
type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share' | 'system';
type SendMessageType = Exclude<MessageType, 'system'>;

// 用户类型枚举（仅表达账号生命周期，不含权限角色）
// - 'temp': 临时用户（访客，未注册）
// - 'normal': 正式用户（已注册）
// 管理员身份由 RBAC 系统管理（见 @spec/api_spec_admin_core.md），不体现在 UserType 中
// 系统通知通过 MessageType = 'system' 和公告 API 处理，不需要"系统用户"类型
type UserType = 'temp' | 'normal';

// 隐私级别枚举（跨模块复用：用户设置、瞬间等）
// - 'public': 所有人可见（包括未登录用户）
// - 'private': 仅自己可见
type PrivacyLevel = 'public' | 'private';

// 角色枚举
type GroupRole = 'owner' | 'admin' | 'member';

// === 客户端 ===
type ClientType = 'web' | 'app';

// === 状态枚举 ===
// 用户在线状态（当前仅支持 online/offline，后续可扩展更多状态）
type UserPresenceStatus = 'online' | 'offline';
type ReportStatus = 'pending' | 'reviewing' | 'resolved' | 'dismissed';

// === 业务操作类型 ===
type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';
type ReportHandleAction = 'resolve' | 'dismiss' | 'escalate';

// 举报原因枚举（用户从中选取）
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

type GroupJoinMethod = 'open' | 'approval' | 'invite_only';

// === 公告 ===
type AnnouncementSeverity = 'info' | 'warning' | 'error';

// === 关注来源（用于来源归因展示）===
type FollowSource = 'moment' | 'search' | 'recommendation' | 'profile' | 'group';

// === 推荐理由类型（用于解释推荐来源）===
type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';

interface RecommendationContext {
  type: RecommendationType;
}

// === 以下类型从各模块收归至此，作为跨模块共享的权威定义 ===

// 性别枚举（原定义于 api_spec_user.md）
type Gender = 'male' | 'female' | 'apache_helicopter';

// 私聊权限（原定义于 api_spec_user.md）
// 注：'friends' 表示"仅好友"，好友 = 双方互相关注的用户
type PrivateChatPermission = 'everyone' | 'friends' | 'nobody';

// 申请状态（原定义于 api_spec_group.md）
// 'revoked': 邀请/申请发起者主动撤销（与被拒绝区分，便于客户端展示不同状态）
type ApplicationStatus = 'pending' | 'accepted' | 'rejected' | 'revoked';

// 分享类型（原定义于 api_spec_share.md）
type ShareType = 'user' | 'group' | 'moment' | 'location';

// 通知类型（原定义于 api_spec_notification.md）
type NotificationType =
  | 'new_follower'
  | 'mutual_follow'
  | 'moment_liked'
  | 'moment_commented'
  | 'comment_liked'
  | 'comment_reply'
  | 'mentioned'
  | 'group_invite_received'
  | 'group_invite_accepted'
  | 'group_application_received'
  | 'group_application_accepted'
  | 'system_announcement';

// 不可发消息原因（原定义于 api_spec_chat.md）
// 设计说明：被拉黑场景统一返回 'privacy'，不暴露拉黑状态（隐私保护）
type MessageBlockedReason =
  | 'privacy'                  // 对方隐私设置不允许（含被拉黑场景）
  | 'stranger_quota_exceeded'  // 陌生人消息限额已用完
  | 'account_suspended';       // 账号被限制
```

### 位置与地理信息

**重要概念：用户真实位置 vs 浏览锚点位置**

本系统区分两种位置：
- **用户真实位置**：通过「更新我的位置」接口更新；属于隐私数据，用于“被其他人发现我/展示我的大致位置”。
- **浏览锚点位置（reference_point）**：作为请求参数传入；用于“当前在看哪里/距离计算基准”。

核心原则：`distance` / `bearing_degrees` 必须由服务端基于 `reference_point` 计算并返回，客户端不得本地计算用于排序。

**隐私与精度策略（契约级）**：
- 服务端可基于隐私设置、关系与登录态对位置/距离做降精度或匿名化处理
- 客户端只用于展示，不得将距离/方位作为精确导航或本地排序依据
- 为防止位置反推，服务端可对短时间内的查询结果做稳定或随机化处理

**请求位置要求（契约级）**：
- 创建类 Request 中出现 `Location` 字段时，必须为必填字段（如 `groups/create`、`moments/create`、`users/me/location/update`）
- 更新/Patch 类 Request 中 `Location` 可为可选字段（如 `groups/update`、`moments/update`），允许仅更新其他字段而不修改位置；管理员配置中的 `Partial<Location>` 同理
- 若用户拒绝定位或无法获取精确位置，客户端必须使用 `ClientConfig.defaults.default_location` 填充 `Location`

```typescript
// 基础坐标点 (经纬度)
interface GeoPoint {
  latitude: number;  // [-90, 90]
  longitude: number; // [-180, 180]
}

// 位置信息 (包含坐标和位置名称)
// 用途：仅用于 request 请求参数
interface Location {
  point: GeoPoint;
  location_name: string; // 必填，0-100字符，无名称时使用 ClientConfig.defaults.default_location.location_name
}

// 相对位置 (用于所有 response 响应数据)
// 用途：所有 response 中的位置字段统一使用此类型
// distance 和 bearing_degrees 基于 reference_point 计算，且可能被服务端降精度
interface RelativeLocation {
  point: GeoPoint;
  location_name: string;       // 位置名称
  distance?: number;           // 距离浏览锚点的距离 (米)，有参考点时有值，否则为 undefined
  bearing_degrees?: number;    // 相对浏览锚点的方位角 [0, 360)，有参考点时有值，否则为 undefined
}

// 使用说明：
// - Request：使用 Location（仅坐标+名称）
// - Response：经过距离/角度计算时使用 RelativeLocation（搜索/列表/Feed 有 distance/bearing；详情/Admin/当前用户视角通常为 undefined）
// - 未经距离/角度计算的响应字段仍使用 Location（如 GroupSettings.location、MyProfile.location）
```

### 分页与元数据

```typescript
// 分页请求参数
interface PaginationParams {
  limit?: number;  // default: 20, max: 100
  cursor?: string; // 不透明字符串，客户端只透传
}

// 通用分页容器
interface PaginatedData<T> {
  items: T[];
  next_cursor?: string;  // 下一页游标；为 undefined 时表示没有更多数据
}
```

### 媒体资源

```typescript
// 消息附件/媒体信息
interface MessageMedia {
  url: string;
  thumbnail_url?: string;
  file_name?: string;
  file_size?: number;
  duration?: number;
  width?: number;
  height?: number;
  mime_type?: string;
}
```

---

## 领域实体定义 (Domain Entities)

本节定义业务核心对象。遵循 **Info (嵌入用) -> Item (统一数据类型)** 的分层模式。通过 `@auth-only`/`@group-admin-only` 等标注区分不同上下文的字段可见性。

### 用户 (User)

说明：类型定义以显式字段为准；禁止使用类型层级表达字段复用，统一使用组合（字段嵌入）。

**系统管理员可见性规则**：
- 系统管理员使用普通 API 时与普通用户**无区别**：使用相同端点、获得相同响应形态
- 其他用户可正常搜索到管理员用户、查看其资料；`UserInfo` 不包含管理员标识，普通用户无法通过任何普通 API 判断对方是否为管理员
- 管理员身份**不通过 `UserType` 体现**（`UserType` 仅表达账号生命周期）；管理员通过 `admin/rbac/get_me`（见 `@spec/api_spec_admin_core.md`）获知自己的管理员身份和权限
- 管理员专用字段（如 `status`、`login_id`、`register_ip` 等）不在本类型中定义，**仅在 `/admin/*` 路径下通过独立的 Admin 类型返回**
- 设计专用接口用于用户联系系统管理员（如举报反馈、申诉等），此类接口中管理员身份匿名化，用户仅知道"对方是官方客服/管理员"

**UserInfo**: 用户基础信息（用于嵌套在其他对象中）
```typescript
interface UserInfo {
  user_id: string;
  nickname: string;
  avatar_url?: string;
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

// 账号状态（系统管理员视角）
type UserAccountStatus = 'normal' | 'banned';
```

**UserItem**: 用户统一数据类型（用于列表、搜索、详情、Feed 等所有场景）
```typescript
interface UserItem {
  user: UserInfo;
  bio?: string;
  tags?: string[];
  
  // 隐私受控字段（隐私限制时不返回，不暴露原因）
  presence_status?: UserPresenceStatus;  // 用户状态（online/offline）
  last_online?: string;                  // 最后在线时间（offline 时有值），精度截断到分钟（隐私保护，防止行为模式分析）
  
  // 位置信息（可见性规则见 RelativeLocation 类型定义）
  location?: RelativeLocation;
  
  // 统计信息：详情页（open_the_door）始终返回；搜索/列表由服务端根据接口语义决定是否返回（客户端不应依赖其在列表中的存在性）
  stats?: UserStats;
  
  // === @auth-only 以下字段仅登录态返回 ===
  relation?: Relationship;
  
  // @auth-only
  // 社交上下文（展示与当前用户的共同点）
  // 详情页/Feed 返回，搜索/列表不返回
  social_context?: {
    mutual_friends_count: number;       // 共同好友数
    mutual_friends_preview?: UserInfo[];  // 前2-3个共同好友
    common_groups_count?: number;       // 共同群组数
    common_tags?: string[];             // 共同兴趣标签
  };
}
```

**Relationship**: 用户关系信息（@auth-only）
```typescript
interface Relationship {
  is_following: boolean;       // 我是否关注了对方
  is_followed_by: boolean;     // 对方是否关注了我
  is_blocked: boolean;         // 我是否拉黑了对方
  
  // 互相关注可由客户端计算：is_following && is_followed_by
  
  // 设计说明：不暴露 is_blocked_by（对方是否拉黑了我）
  // 被拉黑的用户不应感知到自己被拉黑（业界隐私最佳实践）
  // 被拉黑后的体验由服务端隐式处理：搜索不到对方、看不到动态、无法发消息等
  // 客户端看到的效果与"对方设置了严格隐私"一致
}
```

### 群组 (Group)

**GroupInfo**: 群组基础信息（用于嵌套在其他对象中）
```typescript
interface GroupInfo {
  group_id: string;
  name: string;
  avatar_url?: string;
  members_count: number;
}
```

**GroupItem**: 群组统一数据类型（用于列表、搜索、详情、Feed 等所有场景）
```typescript
interface GroupItem {
  group: GroupInfo;
  description?: string;
  tags?: string[];
  has_password: boolean;
  
  // 位置信息：统一使用 RelativeLocation
  // - 列表/搜索/Feed 场景：distance / bearing_degrees 由服务端基于 reference_point 计算
  // - 详情场景（open_the_door）：distance / bearing_degrees 为 undefined
  location: RelativeLocation;
  
  // 列表/详情页返回
  last_activity_at?: string;     // 最近活跃时间（最后消息/公告/成员变动等）
  
  // 详情页返回，搜索/列表不返回
  created_at?: string;
  owner_id?: string;
  
  // === @auth-only 以下字段仅登录态返回 ===
  my_state?: MyGroupState;
  
  // Feed 场景返回（好友中有多少在该群组内）
  friends_in_group?: number;

  // === @group-admin-only 以下字段仅群组管理员返回 ===
  join_method?: GroupJoinMethod;
  is_searchable?: boolean;
  max_members?: number;
}
```

**GroupSettings**: 群组设置（可修改的配置项）
```typescript
interface GroupSettings {
  join_method: GroupJoinMethod;
  is_searchable: boolean;
  max_members: number;
  location: Location;  // 直接映射自请求，不经过距离/角度计算
}
```

**MyGroupState**: 我在群内的状态（@auth-only）
```typescript
interface MyGroupState {
  role: GroupRole;
  is_joined: boolean;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  joined_at?: string;
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
  severity: AnnouncementSeverity;
  action_url?: string;
  created_at: string;
}
```

### 生活瞬间 (Moment)

**MomentItem**: 瞬间统一数据类型（用于列表、搜索、详情、Feed、用户瞬间历史等所有场景）
```typescript
interface MomentItem {
  moment_id: string;
  description: string;
  images?: {
    url: string;
    thumbnail_url?: string;   // 缩略图 URL（列表/Feed 场景使用）
    width?: number;
    height?: number;
    mime_type?: string;
  }[];
  tags: string[];
  mentions?: string[];
  created_at: string;
  likes_count: number;
  comments_count: number;
  shares_count: number;
  user: UserInfo;
  
  // 位置信息（可见性规则见 RelativeLocation 类型定义）
  location: RelativeLocation;
  
  // === @owner-only 以下字段仅资源拥有者返回 ===
  privacy_level?: PrivacyLevel;   // 仅拥有者可见，服务端过滤控制可见性
  
  // === @auth-only 以下字段仅登录态返回 ===
  my_state?: {
    is_liked: boolean;
    is_favorited: boolean;
  };
  
  // Feed 场景返回（推荐理由，让用户了解为什么看到这条内容）
  recommendation?: RecommendationContext;
}
```

注意：`privacy_level` 不对非拥有者暴露，生活瞬间的可见性由服务端过滤控制。仅资源拥有者在详情/我的瞬间列表中可见 `privacy_level`。

**MomentSummary**: 瞬间摘要（用于通知、WebSocket 推送等需要轻量展示瞬间上下文的场景）
```typescript
interface MomentSummary {
  moment_id: string;
  description_preview: string;  // 截断的描述文本（最多 100 字符）
  cover_url?: string;           // 首张图片缩略图 URL（无图片时省略）
}
```

### 点赞 (Like)

**LikeItem**: 点赞列表项（瞬间点赞 `moments/list_likes` 和评论点赞 `comments/list_likes` 共用）
```typescript
interface LikeItem {
  user: UserInfo;
  liked_at: string;
}
```

### 消息 (Message)

**MessageReplyTo**: 回复引用快照
```typescript
interface MessageReplyTo {
  message_id: string;
  user_id: string;
  nickname: string;
  content: string; // 文本预览
  share_code?: string; // 若被回复的消息为 share 类型，则包含该 share_code
}
```

**MessageSent**: 消息发送成功响应（私聊/群聊通用）
```typescript
interface MessageSent {
  message_id: string;
  sent_at: string;
}
```

私聊/群聊消息项类型定义在模块文档中：`@spec/api_spec_chat.md`。

### 标签 (Tag)

```typescript
// 标签趋势（原定义于 api_spec_search.md，跨模块复用故收归至此）
type TagTrend = 'rising' | 'stable' | 'falling';

// 标签项（搜索、Feed 等模块共用）
interface TagItem {
  name: string;
  usage_count: number;
  trend?: TagTrend;
}
```

### 举报 (Report)

```typescript
// 举报项（原定义于 api_spec_report.md，因 admin_moderation 模块 AdminReportItem extends ReportItem，跨 2 模块复用故收归至此）
interface ReportItem {
  report_id: string;
  target_type: ReportTargetType;
  target_id: string;
  reason: ReportReason;
  description?: string;
  status: ReportStatus;
  created_at: string;
  handled_at?: string;
}
```

### 用户设置 (UserSettings)

```typescript
// 用户设置（原定义于 api_spec_user.md，因 WebSocket 模块 SelfSettingsUpdatedPayload 引用，跨 2 模块复用故收归至此）
// PrivateChatPermission 定义于本文件

interface UserSettings {
  privacy_level: PrivacyLevel;
  location_visible: boolean;          // false 时，普通用户视角的 UserItem.location 字段不返回
  presence_visible: boolean;         // 在线状态和最后在线时间的可见性（统一控制），false 时 `presence_status`/`last_online` 均不返回
  social_lists_visible: boolean;     // 好友/关注/粉丝列表是否公开（统一控制）
  groups_visible: boolean;           // 群组列表是否公开（false 时 users/list_groups 直接过滤）
  liked_moments_visible: boolean;    // 点赞瞬间列表是否公开
  stats_visible: boolean;            // 统计信息是否公开
  allow_follow: boolean;
  allow_private_chat: PrivateChatPermission;
  searchable: boolean;
  phone_searchable: boolean;
  nickname_searchable: boolean;
  location_searchable: boolean;
}
```
