# HotSpot API - 通用类型定义

> 定义 HotSpot API 跨模块复用的基础值对象（Domain Primitives）和业务核心对象（Domain Entities）。

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
// 注：SendMessageType 仅 chat 单模块使用，已下放至 api_spec_chat.md，此处不重复

// 用户类型枚举（仅表达账号生命周期，不含权限角色）
// - 'temp': 临时用户（访客，未注册）
// - 'normal': 正式用户（已注册）
// 管理员身份由 RBAC 系统管理（见 api_spec_admin_core.md），不体现在 UserType 中
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

// === 设备标识（单设备单会话模型的基础原语，详见 api_spec_auth.md「单设备单会话模型」）===
// 设备身份标识（UUID v4 字符串）。语义：一台逻辑设备 = 一个 device_id。
// - App：客户端首次启动生成并存入 Keychain（iOS）/ Keystore（Android），卸载重装后视为新设备；一经生成即稳定，禁止轮换
// - Web：由服务端通过独立 `device_id` Cookie 下发（非 HttpOnly，前端可读以便与 App 内嵌 WebView 共享；见 base.md「Web 端 device_id cookie」），登出不清除以保持设备身份连续；不同浏览器档案 = 不同 device_id
// 客户端不得自造或预测 device_id（Web 端由服务端下发；App 端本地生成一次后即稳定）
// 服务端用作会话主键的一部分：同一 user 在同一 device_id 上只允许一个活跃 session（新登录驱逐旧 session，详见 auth.md login）
type DeviceId = string;

// === Reaction（emoji 表情回应，统一用于 message/moment/comment 三类资源）===
// emoji 为 Unicode 码点字符串（如 "👍"），服务端按 AppConfig.limits.reaction_emojis 白名单校验
// 可用 emoji 集合对客户端透明，通过 system/get_client_config 下发
type ReactTarget = 'message' | 'moment' | 'comment';

interface Reaction {
  emoji: string;          // 表情符号
  count: number;          // 该表情的反应总数
  reacted_by_me: boolean; // 当前用户是否已用此表情反应（@auth-only 语义：匿名时为 false）
}

// === 状态枚举 ===
// 用户在线状态（当前仅支持 online/offline，后续可扩展更多状态）
type UserPresenceStatus = 'online' | 'offline';
type ReportStatus = 'pending' | 'reviewing' | 'escalated' | 'resolved' | 'dismissed';

// === 业务操作类型 ===
type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';
type ReportAction = 'resolve' | 'dismiss' | 'escalate';

// 举报原因枚举（用户从中选取；跨 report + admin_moderation 两模块——AdminReportItem extends ReportItem 继承此字段）
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
type AnnounceLevel = 'info' | 'warning' | 'error';

// === 关注来源（用于来源归因展示）===
type FollowSource = 'moment' | 'search' | 'recommendation' | 'profile' | 'group';

// === 推荐理由类型（用于解释推荐来源）===
// 注：原 RecommendationContext 为单字段 { type } 包装，过度设计，已折叠。
// MomentItem.recommendation 现直接使用 RecommendationType。
type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';

// === 以下类型从各模块收归至此，作为跨模块共享的权威定义 ===

// 私聊权限（跨 user + chat + admin 三模块，故保留在通用类型）
// 注：'friends' 表示"仅好友"，好友 = 双方互相关注的用户
type ChatPerm = 'everyone' | 'friends' | 'nobody';

// 通知类型（跨 notification + websocket + admin 三模块，故保留在通用类型）
// 注：原单模块类型 Gender/ApplicationStatus/ShareType/BlockReason/SendMessageType/MessageSent
// 经核查仅单模块使用，已下放回各自模块文件（符合「单模块归属」规则）。
// 通知类型与分类本身确实跨 notification + websocket + admin_moderation 三个模块（未读计数、推送事件、管理端），故保留于此。
type NotifyType =
  | 'new_follower'
  | 'mutual_follow'
  | 'moment_commented'
  | 'comment_reply'
  | 'mentioned'
  | 'reaction_added'              // 新增：表情反应通知（@auth-only，仅 message/moment/comment 三类目标的 reaction）
  | 'group_invite_received'
  | 'group_invite_revoked'        // 邀请被发起方撤销或过期自动失效（@auth-only，通知受邀人）
  | 'group_invite_accepted'
  | 'group_application_received'
  | 'group_application_cancelled' // 申请被发起方取消或过期自动失效（@auth-only，通知群管理员）
  | 'group_application_accepted'
  | 'group_application_rejected'  // 新增：申请被拒绝（@auth-only，通知申请人，对应 #18 返回处理结果）
  | 'report_resolved'             // 新增：举报被处理（@auth-only，通知举报人处理结果）
  | 'report_dismissed'            // 新增：举报被驳回（@auth-only，通知举报人处理结果）
  | 'system_announcement'
  | 'nearby_new_moment';          // 新增：附近有人发布了新瞬间（基于位置的发现通知，用户可配置开关，服务端节流推送）

// 通知分类（用于未读计数分组展示，见 api_spec_notify.md get_unread_count 返回的 by_type）
// 映射为契约级，新增 NotifyType 时必须同步归类：
// - social:        new_follower, mutual_follow
// - interaction:   moment_commented, comment_reply, mentioned, reaction_added
// - group:         group_invite_received, group_invite_revoked, group_invite_accepted, group_application_received, group_application_cancelled, group_application_accepted, group_application_rejected
// - report:        report_resolved, report_dismissed
// - nearby:        nearby_new_moment
// - system:        system_announcement
type NotifyCategory = 'social' | 'interaction' | 'group' | 'report' | 'nearby' | 'system';
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
- 创建类 Request 中出现 `Location` 字段时，必须为必填字段（如 `groups/create`、`moments/create`、`users/update_location`）
- 更新/Patch 类 Request 中 `Location` 可为可选字段（如 `groups/update`、`moments/update`），允许仅更新其他字段而不修改位置；管理员配置中的 `Partial<Location>` 同理
- 若用户拒绝定位或无法获取精确位置，客户端必须使用 `ClientConfig.defaults.default_location` 填充 `Location`

**`reference_point` 解析规则（契约级）**：

发现类与搜索类接口（所有 `feed/*`、`search/*` 及各资源 `search` 端点）的 `reference_point` 字段统一为**可选**（`reference_point?: GeoPoint`）。当客户端不传时，服务端按以下回退链解析浏览锚点位置：

1. 请求显式传入的 `reference_point`（如地图浏览场景传地图中心点）
2. 已登录用户最近一次 `users/update_location` 的位置（匿名或未更新过位置时跳过）
3. `ClientConfig.defaults.default_location.point`

设计目的：让 App 端绝大多数请求无需自行管理定位状态机——服务端自动复用用户已知位置；只有"地图浏览"等需要显式锚点的场景，客户端才传入 `reference_point`。

不适用：`map/canvas` 等 map 模块端点用 `viewport.bounds` 锚定视口，无 `reference_point` 字段，不经此规则。

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
// - 未经距离/角度计算的响应字段仍使用 Location（如 GroupSettings.location、AccountData.location）
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
// 媒体资源基础信息（图片/视频等资源的展示与尺寸元数据）
// 用途：MomentItem.images、MessageMedia 等媒体字段的共享基础类型
interface MediaAsset {
  url: string;
  thumbnail_url?: string;
  width?: number;
  height?: number;
  mime_type?: string;  // MIME 类型字符串，如 "image/jpeg"
}

// 消息附件/媒体信息（在 MediaAsset 基础上补充文件元数据）
interface MessageMedia extends MediaAsset {
  file_name?: string;
  file_size?: number;
  duration?: number;
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
- 管理员身份**不通过 `UserType` 体现**（`UserType` 仅表达账号生命周期）；管理员通过 `admin/rbac/get_me`（见 `api_spec_admin_core.md`）获知自己的管理员身份和权限
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
```


**UserItem**: 用户统一数据类型（用于列表、搜索、详情、Feed 等所有场景）
```typescript
interface UserItem {
  user: UserInfo;
  bio?: string;
  tags?: string[];

  // 隐私受控字段（隐私限制时不返回，不暴露原因）
  presence_status?: UserPresenceStatus;  // 用户状态（online/offline）
  last_online_at?: string;               // 最后在线时间（offline 时有值），UTC ISO 8601，精度截断到分钟（隐私保护，防止行为模式分析）

  // 位置信息（可见性规则见 RelativeLocation 类型定义）
  location?: RelativeLocation;

  // 统计信息：详情页（get_detail）始终返回；搜索/列表由服务端根据接口语义决定是否返回（客户端不应依赖其在列表中的存在性）
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

  // === @member-only 以下字段仅群成员视角返回 ===
  // 设计说明：密码状态不对未加入的匿名/非成员暴露（避免向爬虫/爆破者提供精准线索）。
  // 成员可见以便了解群组门槛；未加入用户通过 join 流程时才感知密码要求。
  has_password?: boolean;

  // 位置信息：统一使用 RelativeLocation
  // - 列表/搜索/Feed 场景：distance / bearing_degrees 由服务端基于 reference_point 计算
  // - 详情场景（get_detail）：distance / bearing_degrees 为 undefined
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
  muted_until?: string;           // 禁言截止时间（UTC ISO 8601）；is_muted=true 时有值，客户端据此展示剩余时间并预判输入框可用性
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
  severity: AnnounceLevel;
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
  images?: MediaAsset[];
  tags?: string[];                 // 标签；与 user/group 对齐改为可选
  mentions?: string[];
  created_at: string;
  likes_count: number;
  comments_count: number;
  shares_count: number;
  reactions?: Reaction[];          // 表情反应聚合（无反应时省略）；reaction.reacted_by_me 为 @auth-only 语义
  user: UserInfo;

  // 位置信息（可见性规则见 RelativeLocation 类型定义）
  location: RelativeLocation;

  // === @owner-only 以下字段仅资源拥有者返回 ===
  privacy_level?: PrivacyLevel;   // 仅拥有者可见，服务端过滤控制可见性

  // === @auth-only 以下字段仅登录态返回 ===
  my_state?: {
    is_favorited: boolean;         // 收藏状态（点赞/反应态改由 reactions[].reacted_by_me 表达）
  };

  // Feed 场景返回（推荐理由，让用户了解为什么看到这条内容）
  recommendation?: RecommendationType;  // 原 RecommendationContext 已折叠为单 type
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

### 反应 (Reaction)

**ReactionItem**: 反应者列表项（用于 `reactions/list` 查询某个资源的反应详情，替代原 `moments/list_likes`/`comments/list_likes`）
```typescript
interface ReactionItem {
  user: UserInfo;
  emoji: string;        // 该用户使用的表情
  reacted_at: string;   // UTC ISO 8601
}
```

> 设计说明：原 `LikeItem`（仅 liked_at）随点赞/取消点赞二元接口被统一 `set_reaction` 取代而废弃。
> 反应列表统一返回带 emoji 的 `ReactionItem`，客户端可按 emoji 分组渲染，也可聚合为 `MomentItem.reactions` 的 count。

### 消息 (Message)

**MessageReplyTo**: 回复引用快照（用于消息回复时引用被回复消息的上下文）
```typescript
interface MessageReplyTo {
  message_id: string;
  type: MessageType;       // 被回复消息的类型（决定前端渲染方式：text 显示文字预览，image/video 等显示对应占位，share 显示分享卡片）
  sender: UserInfo;        // 被回复消息的发送者（嵌套 UserInfo，与 sender/actor 等用户引用保持一致）
  content?: string;        // 文本预览（仅 text 类型有意义；非文本类型可省略）
  share_code?: string;     // 若被回复的消息为 share 类型，则包含该 share_code
  sent_at: string;         // 被回复消息的发送时间
}
```

> 设计说明：`sender` 采用嵌套 `UserInfo` 而非扁平 `user_id`/`nickname`，与 spec 中所有"对用户的引用"（`sender`/`actor`/`creator`/`applicant` 等）保持一致；`type` 字段是渲染回复引用的必需信息（区分文本/图片/分享等），扁平结构无法表达非文本消息的引用。

> 注：`MessageSent`（消息发送成功响应）经核查仅 chat 模块使用，已下放至 `api_spec_chat.md`，此处不再定义。

私聊/群聊消息项类型定义在模块文档中：`api_spec_chat.md`。

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
  message_snapshot?: ReportSnapshot;  // 仅 target_type='message'：举报时的消息内容快照，供管理员审核

  // === @owner-only 仅举报人自己可见（reports/list_my 返回，他人不可见）===
  // 处理结果摘要：让举报人了解处理进展与结论，但不暴露处理管理员身份（隐私）
  resolution?: {
    action: ReportAction;   // 最终处理动作
    resolved_at: string;          // UTC ISO 8601，处理完成时间
    note?: string;                // 可选的处理说明（脱敏后，不含内部细节）
  };
}

// 消息举报快照（隐私边界设计：管理员基于快照审核，不直接查询私聊消息库）
// 举报人在 reports/create 时提交，服务端原样存储，审核方据此判断举报是否成立
// 跨 report + admin_moderation 两模块（随 ReportItem 被 AdminReportItem 继承），故保留于通用类型
interface ReportSnapshot {
  chat_type: 'private' | 'group';   // 会话类型
  target_chat_id: string;           // private: target_user_id；group: group_id
  message_type: MessageType;        // 消息类型（text/image 等，决定 snapshot_content 的含义）
  snapshot_content?: string;        // 文本内容（text 类型）；非文本类型为空
  snapshot_media?: MessageMedia;    // 媒体引用（图片/视频等）；文本类型为空
  sent_at: string;                  // 消息发送时间（快照基准）
}
```

### 用户设置 (UserSettings)

```typescript
// 用户设置（原定义于 api_spec_user.md，因 WebSocket 模块 SelfSettingsUpdatedPayload 引用，跨 2 模块复用故收归至此）
// ChatPerm 定义于本文件

interface UserSettings {
  privacy_level: PrivacyLevel;
  location_visible: boolean;          // false 时，普通用户视角的 UserItem.location 字段不返回
  presence_visible: boolean;         // 在线状态和最后在线时间的可见性（统一控制），false 时 `presence_status`/`last_online_at` 均不返回
  social_lists_visible: boolean;     // 好友/关注/粉丝列表是否公开（统一控制）
  groups_visible: boolean;           // 群组列表是否公开（false 时 users/list_groups 直接过滤）
  reacted_moments_visible: boolean;  // 反应过的瞬间列表是否公开（原 liked_moments_visible，随 reaction 统一更名）
  stats_visible: boolean;            // 统计信息是否公开
  allow_follow: boolean;
  allow_private_chat: ChatPerm;
  searchable: boolean;
  phone_searchable: boolean;
  nickname_searchable: boolean;
  location_searchable: boolean;
  nearby_moment_notify: boolean;     // 是否接收"附近新瞬间"推送通知；默认 false（避免打扰，用户主动开启）
}
```

### 应用配置 (AppConfig 相关)

> 以下配置类型由 `system`（客户端可见投影 `ClientConfig`）与 `admin_core`（完整服务端配置 `AppConfig`）共用，故收归至此。

```typescript
// 功能开关（客户端可见，由管理员配置）
interface FeatureSwitches {
  enable_public_registration: boolean;
  enable_guest_login: boolean;
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
  search_radius_max: number;  // 搜索半径上限（米），用于搜索和 Feed 相关接口
  message_recall_window_seconds: number;  // 消息撤回时间窗（秒），发送者仅可撤回该窗口内的消息；0 表示不允许撤回
  stranger_message_daily_quota: number;   // 每日可向非互相关注（陌生人）用户发送的私聊消息上限；超过返回 BlockReason='stranger_quota_exceeded'，0 表示禁止任何陌生人消息
  reaction_emojis: string[];              // 允许的表情反应 emoji 白名单（Unicode 码点字符串数组），客户端据此渲染反应面板
  group_invite_ttl_hours: number;         // 群邀请有效期（小时），过期后状态置 'revoked'；0 表示永不过期
  group_application_ttl_hours: number;    // 群申请有效期（小时），过期后状态置 'revoked'；0 表示永不过期
}

// 联系/合规信息（客户端可见）
interface ContactInfo {
  support_email: string;
  privacy_policy_url: string;
  terms_of_service_url: string;
  official_website_url: string;  // 原 official_website，加 _url 后缀与同接口其他 URL 字段统一
}
```
