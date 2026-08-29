// ==================== 领域实体定义 (Domain Entities) ====================
import { AnnounceSeverity, GroupJoinMethod, GroupRole, Location, MessageMedia, MessageType, MomentImage, PaginatedData, Reaction, ReactionItem, RecommendationType, RelativeLocation, UserPresenceStatus } from './common';
import { ForbiddenReason } from './user';


// 1. 用户 (User)

export interface UserInfo {
  user_id: string;
  nickname: string;
  avatar_url?: string;
}

export interface UserStats {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
}

export interface Relationship {
  is_following: boolean;
  is_followed_by: boolean;
  is_blocked: boolean;
}

export interface SocialContext {
  mutual_friends_count: number;
  mutual_friends_preview?: UserInfo[];
  common_groups_count?: number;
  common_tags?: string[];
}

export interface UserItem {
  user: UserInfo;
  bio: string | null;
  tags: string[];

  // 隐私受控字段：恒定 nullable，null = 隐藏/不适用
  presence_status: UserPresenceStatus | null;
  last_online_at: string | null;
  location: RelativeLocation | null;
  stats: UserStats | null;
  relation: Relationship | null;
  social_context: SocialContext | null;
}

// 2. 群组 (Group)

export interface GroupInfo {
  group_id: string;
  name: string;
  avatar_url?: string;
  members_count: number;
}

export interface GroupViewerState {
  role: GroupRole;
  unread_count: number;
  is_muted: boolean;
  muted_until: string | null;
  is_pinned: boolean;
  joined_at: string;
}

// 消息摘要快照（spec types.md → MessageBrief，reply_to 与 last_message 共用）
export interface MessageBrief {
  message_id: string;
  type: MessageType;
  content: string | null;
  share_code: string | null; // 最后一条消息为 share 类型时包含
  sender: UserInfo;
  sent_at: string;
}

export interface GroupItem {
  group: GroupInfo;
  description: string | null;
  tags: string[];
  join_method: GroupJoinMethod;
  requires_approval: boolean;
  location: RelativeLocation | null;
  last_activity_at: string | null;
  online_members_count: number | null;
  next_activity_at: string | null;
  created_at: string | null;
  owner_id: string | null;
  viewer_state: GroupViewerState | null;
  friends_in_group: number | null;
  // 仅 groups/fetch_joined 返回最近消息预览；其他场景为 null
  last_message: MessageBrief | null;
}

export interface GroupSettings {
  join_method: GroupJoinMethod;   // 默认 'direct'
  requires_approval: boolean;     // 默认 false
  has_password: boolean;          // 群是否设有密码
  is_searchable: boolean;
  members_searchable: boolean;    // 默认 true（spec types.md → GroupSettings）
  max_members: number;
  location: Location;
}

export interface GroupAnnouncementItem {
  announcement_id: string;
  group_id: string;
  title: string;
  content: string;
  created_by: UserInfo;
  created_at: string;
  is_pinned: boolean;
}

export interface SystemAnnouncementItem {
  announcement_id: string;
  title: string;
  content: string;
  severity: AnnounceSeverity;
  action_url: string | null;
  created_at: string;
}

// 3. 生活瞬间 (Moment)

export interface MomentSummary {
  moment_id: string;
  cover_url: string | null;     // 首张图片缩略图 URL；null = 无图片（spec types.md → MomentSummary）
  description_preview: string;
}

export interface MomentItem {
  moment_id: string;
  description: string;
  images: MomentImage[];
  tags: string[];
  mentions: string[];
  created_at: string;
  comments_count: number;
  shares_count: number;
  reactions: Reaction[];
  user: UserInfo;
  is_deleted: boolean;
  location: RelativeLocation | null;
  is_favorited: boolean;
  recommendation: RecommendationType | null;
}

// 4. 消息 (Message)

export interface MessageSentResult {
  message_id: string;
  sent_at: string;
}

export interface PrivateMessageItem {
  message_id: string;
  type: MessageType;
  content: string | null;
  media: MessageMedia | null;
  share_code: string | null;
  sender: UserInfo;
  reply_to: MessageBrief | null;
  sent_at: string;
  edited_at: string | null;
  recalled_at: string | null;
  is_read: boolean;
  reactions: Reaction[];
}

export interface GroupMessageItem {
  message_id: string;
  type: MessageType;
  content: string | null;
  media: MessageMedia | null;
  share_code: string | null;
  sender: UserInfo;
  reply_to: MessageBrief | null;
  mentions: string[];
  recalled_by: UserInfo | null;
  sent_at: string;
  edited_at: string | null;
  recalled_at: string | null;
  read_count: number | null;
  reactions: Reaction[];
}

// 5. 评论 (Comment)

export interface CommentItem {
  comment_id: string;
  user: UserInfo;
  content: string;
  parent_id: string | null;
  mentions: string[];
  created_at: string;
  replies_count: number;
  is_deleted: boolean;
  reactions: Reaction[];
  // 顶层评论列表场景(comments/fetch)返回前 N 条回复预览；其他场景为空数组
  top_replies: CommentItem[];
}

// spec user.md → open_the_door：recent_moments 仅 include_recent_moments=true 时返回
export interface UserProfileBundle {
  user: UserItem;
  recent_moments: PaginatedData<MomentItem> | null;
}

// ReactionItem 已在通用类型区域定义; LikeItem 为历史遗留,删除以消除混淆。

// 6. 私聊 (Private Chat)



export interface PrivateChatItem {
  other_user: UserInfo;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_activity_at: string;
  last_message: MessageBrief | null;
  presence_status: UserPresenceStatus | null;
  created_at: string | null;
  can_message: boolean;
  forbidden_reason: ForbiddenReason | null;  // 仅 can_message=false 时非 null（spec chat.md → PrivateChatItem）
}

export interface PrivateChatOpenResult {
  chat: PrivateChatItem;
  recent_messages: PaginatedData<PrivateMessageItem>;
}
