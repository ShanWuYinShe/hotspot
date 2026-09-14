// ==================== 私聊管理API类型 ====================
import { GeoPoint, GroupJoinMethod, GroupRole, Location, PaginationParams, SendMessageType, UserPresenceStatus } from './common';
import { GroupInfo, UserInfo } from './entities';
import { ApplicationStatus } from './user';


export interface FetchPrivateChatsRequest {
  pagination?: PaginationParams;
}

export interface OpenPrivateChatDoorRequest {
  target_user_id: string;
}

export interface UpdatePrivateChatPreferencesRequest {
  target_user_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}

export interface DeletePrivateChatRequest {
  target_user_id: string;
}

// ==================== 私聊消息API类型 ====================

export interface SendPrivateMessageRequest {
  target_user_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  share_code?: string;
  reply_to_message_id?: string;
}

export interface FetchPrivateMessagesRequest {
  target_user_id: string;
  pagination?: PaginationParams;
}

export interface UpdatePrivateMessageRequest {
  message_id: string;
  content: string;
}

export interface RecallPrivateMessageRequest {
  message_id: string;
}

export interface MarkPrivateReadRequest {
  scope: string;
  target_user_id?: string;
  up_to_message_id?: string;
}

// ==================== 群组管理API类型 ====================

export interface SearchGroupsRequest {
  keyword?: string;
  tag?: string;
  radius?: number;
  sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';
  reference_point?: GeoPoint;
  pagination?: PaginationParams;
}

export interface OpenGroupDoorRequest {
  group_id: string;
}

export interface CreateGroupRequest {
  name: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  password?: string;
  join_method?: GroupJoinMethod;
  requires_approval?: boolean;
  is_searchable?: boolean;
  members_searchable?: boolean; // 默认 true
  max_members?: number;
  location: Location;
}

export interface UpdateGroupRequest {
  group_id: string;
  name?: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  password?: string;
  clear_password?: boolean;
  join_method?: GroupJoinMethod;
  requires_approval?: boolean;
  next_activity_at?: string | null;
  is_searchable?: boolean;
  members_searchable?: boolean;
  max_members?: number;
  location?: Location;
}

export interface DismissGroupRequest {
  group_id: string;
}

export interface UpdateGroupPreferencesRequest {
  group_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}

// ==================== 群组成员管理API类型 ====================

export interface FetchGroupMembersRequest {
  group_id: string;
  role?: GroupRole;
  pagination?: PaginationParams;
}

export interface SearchGroupMembersRequest {
  group_id: string;
  keyword: string;
  role?: GroupRole;
  pagination?: PaginationParams;
}

export interface GroupMemberItem {
  user: UserInfo;
  role: GroupRole;
  joined_at: string;
  muted_until?: string;
  presence_status?: UserPresenceStatus;
  last_online_at?: string;
}

export interface CreateGroupInviteRequest {
  group_id: string;
  user_id: string;
  message?: string;
}

// invite_id/expires_at 可选（spec group.md → Invitation：no-op 已是成员/TTL=0 时省略）
export interface Invitation {
  invite_id: string | null;
  expires_at: string | null;
}

export interface EnterGroupRequest {
  group_id: string;
  password?: string;
  message?: string;
}

export interface JoinGroupResult {
  status: 'joined' | 'pending_approval';
  application?: Application;
}

export interface Application {
  application_id: string;
}

export interface SetMemberRoleRequest {
  group_id: string;
  user_id: string;
  role: 'admin' | 'member';
}

export interface SetMemberMuteRequest {
  group_id: string;
  user_id: string;
  active: boolean;
  duration_minutes?: number;
}

export interface SetMemberBanRequest {
  group_id: string;
  user_id: string;
  active?: boolean;
  reason?: string;
}

export interface FetchBannedMembersRequest {
  group_id: string;
  pagination?: PaginationParams;
}

export interface AdminBannedMemberItem {
  user: UserInfo;
  reason: string | null;
  banned_at: string;
  banned_by: UserInfo | null;
}

export interface RemoveMemberRequest {
  group_id: string;
  user_id: string;
}

export interface FetchGroupApplicationsRequest {
  group_id: string;
  status?: 'pending' | 'accepted' | 'rejected' | 'revoked' | 'all';
  pagination?: PaginationParams;
}

export interface GroupApplicationItem {
  application_id: string;
  applicant: UserInfo;
  group: GroupInfo;
  expires_at: string | null;
  message: string | null;
  status: ApplicationStatus;
  created_at: string;
  handled_at: string | null;
  handled_by: UserInfo | null;
}

export interface ApproveGroupApplicationRequest {
  group_id: string;
  application_id: string;
}

export interface RejectGroupApplicationRequest {
  group_id: string;
  application_id: string;
  reason?: string;
}

export interface TransferOwnershipRequest {
  group_id: string;
  new_owner_id: string;
}

export interface FetchJoinedGroupsRequest {
  pagination?: PaginationParams;
}

export interface FetchSubmittedApplicationsRequest {
  status?: 'pending' | 'accepted' | 'rejected' | 'revoked' | 'all';
  pagination?: PaginationParams;
}

export interface CancelApplicationRequest {
  group_id: string;
  application_id: string;
}

export interface ListSentInvitesRequest {
  group_id: string;
  status?: ApplicationStatus | 'all';
  pagination?: PaginationParams;
}

export interface GroupSentInviteItem {
  invite_id: string;
  inviter: UserInfo;
  invitee: UserInfo;
  status: ApplicationStatus;
  message: string | null;
  created_at: string;
  expires_at: string | null;
  responded_at: string | null;
}

export interface RevokeSentInviteRequest {
  group_id: string;
  invite_id: string;
  reason?: string;
}

export interface FetchReceivedInvitesRequest {
  status?: 'pending' | 'accepted' | 'rejected' | 'revoked' | 'all';
  pagination?: PaginationParams;
}

export interface GroupInviteItem {
  invite_id: string;
  group: GroupInfo;
  inviter: UserInfo;
  message?: string;
  status: ApplicationStatus;
  created_at: string;
  expires_at?: string;
}

export interface AcceptInviteRequest {
  invite_id: string;
}

/** acceptGroupInvite 响应：刚加入的群组 id（spec group:912-917） */
export interface InviteAcceptedResult {
  group_id: string;
}

export interface RejectInviteRequest {
  invite_id: string;
  reason?: string;
}

export interface ExitGroupRequest {
  group_id: string;
}

// ==================== 群公告API类型 ====================

export interface CreateGroupAnnouncementRequest {
  group_id: string;
  title: string;
  content: string;
  is_pinned?: boolean;
}

export interface FetchGroupAnnouncementsRequest {
  group_id: string;
}

export interface UpdateGroupAnnouncementRequest {
  group_id: string;
  announcement_id: string;
  title?: string;
  content?: string;
  is_pinned?: boolean;
}

export interface DeleteGroupAnnouncementRequest {
  group_id: string;
  announcement_id: string;
}

// ==================== 群聊消息API类型 ====================

export interface SendGroupMessageRequest {
  group_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  share_code?: string;
  reply_to_message_id?: string;
  mentions?: string[];
}

export interface FetchGroupMessagesRequest {
  group_id: string;
  pagination?: PaginationParams;
}

export interface UpdateGroupMessageRequest {
  message_id: string;
  content: string;
}

export interface RecallGroupMessageRequest {
  message_id: string;
}

// 群主/管理员撤回他人消息（spec group.md → groups/admin/messages/recall，无时间窗）
export interface RecallAdminGroupMessageRequest {
  group_id: string;
  message_id: string;
}

export interface MarkGroupReadRequest {
  scope: string;
  group_id?: string;
  up_to_message_id?: string;
}
