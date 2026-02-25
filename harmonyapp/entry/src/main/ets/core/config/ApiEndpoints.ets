// Centralized API endpoint registry for HarmonyOS client.
// Endpoint values MUST match spec/api_spec_*.md.
// Do NOT add endpoints here unless they exist in the spec.

// Auth
class AuthSessionsEndpoints {
  readonly LIST: string = '/api/v1/auth/me/sessions/list';
  readonly REVOKE: string = '/api/v1/auth/me/sessions/revoke';
  readonly REVOKE_ALL: string = '/api/v1/auth/me/sessions/revoke_all';
}

class AuthMeEndpoints {
  readonly CHANGE_PASSWORD: string = '/api/v1/auth/me/change_password';
  readonly SESSIONS: AuthSessionsEndpoints = new AuthSessionsEndpoints();
}

class AuthEndpoints {
  readonly LOGIN: string = '/api/v1/auth/login';
  readonly REGISTER: string = '/api/v1/auth/register';
  readonly LOGOUT: string = '/api/v1/auth/logout';
  readonly GUEST: string = '/api/v1/auth/guest';
  readonly REFRESH_TOKEN: string = '/api/v1/auth/refresh_token';
  readonly CHECK_STATUS: string = '/api/v1/auth/check_status';
  readonly REQUEST_PASSWORD_RESET: string = '/api/v1/auth/request_password_reset';
  readonly RESET_PASSWORD: string = '/api/v1/auth/reset_password';
  readonly ME: AuthMeEndpoints = new AuthMeEndpoints();
}

// User
class UserMeProfileEndpoints {
  readonly GET: string = '/api/v1/users/me/profile/get';
  readonly UPDATE: string = '/api/v1/users/me/profile/update';
}

class UserMeLocationEndpoints {
  readonly UPDATE: string = '/api/v1/users/me/location/update';
}

class UserMeSettingsEndpoints {
  readonly GET: string = '/api/v1/users/me/settings/get';
  readonly UPDATE: string = '/api/v1/users/me/settings/update';
}

class UserMeEndpoints {
  readonly PROFILE: UserMeProfileEndpoints = new UserMeProfileEndpoints();
  readonly LOCATION: UserMeLocationEndpoints = new UserMeLocationEndpoints();
  readonly SETTINGS: UserMeSettingsEndpoints = new UserMeSettingsEndpoints();
  readonly UPGRADE: string = '/api/v1/users/me/upgrade';
  readonly DELETE: string = '/api/v1/users/me/delete';
  readonly FOLLOW: string = '/api/v1/users/me/follow';
  readonly UNFOLLOW: string = '/api/v1/users/me/unfollow';
  readonly BLOCK: string = '/api/v1/users/me/block';
  readonly UNBLOCK: string = '/api/v1/users/me/unblock';
  readonly LIST_BLOCKED: string = '/api/v1/users/me/list_blocked';
  readonly SET_REMARK: string = '/api/v1/users/me/set_remark';
  readonly EMAIL_REQUEST_CHANGE: string = '/api/v1/users/me/email/request_change';
  readonly EMAIL_CONFIRM_CHANGE: string = '/api/v1/users/me/email/confirm_change';
  readonly PHONE_REQUEST_CHANGE: string = '/api/v1/users/me/phone/request_change';
  readonly PHONE_CONFIRM_CHANGE: string = '/api/v1/users/me/phone/confirm_change';
}

class UserEndpoints {
  readonly SEARCH: string = '/api/v1/users/search';
  readonly OPEN_THE_DOOR: string = '/api/v1/users/open_the_door';
  readonly LIST_FRIENDS: string = '/api/v1/users/list_friends';
  readonly LIST_FOLLOWING: string = '/api/v1/users/list_following';
  readonly LIST_FOLLOWERS: string = '/api/v1/users/list_followers';
  readonly LIST_GROUPS: string = '/api/v1/users/list_groups';
  readonly LIST_MOMENTS: string = '/api/v1/users/list_moments';
  readonly LIST_LIKED_MOMENTS: string = '/api/v1/users/list_liked_moments';
  readonly ME: UserMeEndpoints = new UserMeEndpoints();
}

// Group
class GroupMembersInvitesEndpoints {
  readonly LIST: string = '/api/v1/groups/members/invites/list';
  readonly REVOKE: string = '/api/v1/groups/members/invites/revoke';
}

class GroupMembersEndpoints {
  readonly LIST: string = '/api/v1/groups/members/list';
  readonly SEARCH: string = '/api/v1/groups/members/search';
  readonly INVITE: string = '/api/v1/groups/members/invite';
  readonly INVITES: GroupMembersInvitesEndpoints = new GroupMembersInvitesEndpoints();
  readonly SET_ROLE: string = '/api/v1/groups/members/set_role';
  readonly MUTE: string = '/api/v1/groups/members/mute';
  readonly UNMUTE: string = '/api/v1/groups/members/unmute';
  readonly BAN: string = '/api/v1/groups/members/ban';
  readonly UNBAN: string = '/api/v1/groups/members/unban';
  readonly LIST_BANNED: string = '/api/v1/groups/members/list_banned';
  readonly REMOVE_MEMBER: string = '/api/v1/groups/members/remove_member';
}

class GroupApplicationsEndpoints {
  readonly LIST: string = '/api/v1/groups/applications/list';
  readonly APPROVE: string = '/api/v1/groups/applications/approve';
  readonly REJECT: string = '/api/v1/groups/applications/reject';
}

class GroupAnnouncementsEndpoints {
  readonly CREATE: string = '/api/v1/groups/announcements/create';
  readonly LIST: string = '/api/v1/groups/announcements/list';
  readonly UPDATE: string = '/api/v1/groups/announcements/update';
  readonly DELETE: string = '/api/v1/groups/announcements/delete';
}

class GroupMeApplicationsEndpoints {
  readonly LIST: string = '/api/v1/groups/me/applications/list';
  readonly CANCEL: string = '/api/v1/groups/me/applications/cancel';
}

class GroupMeInvitesEndpoints {
  readonly LIST: string = '/api/v1/groups/me/invites/list';
  readonly ACCEPT: string = '/api/v1/groups/me/invites/accept';
  readonly REJECT: string = '/api/v1/groups/me/invites/reject';
}

class GroupMeEndpoints {
  readonly JOIN: string = '/api/v1/groups/me/join';
  readonly LEAVE: string = '/api/v1/groups/me/leave';
  readonly LIST: string = '/api/v1/groups/me/list';
  readonly UPDATE_SETTINGS: string = '/api/v1/groups/me/update_settings';
  readonly TRANSFER_OWNERSHIP: string = '/api/v1/groups/me/transfer_ownership';
  readonly APPLICATIONS: GroupMeApplicationsEndpoints = new GroupMeApplicationsEndpoints();
  readonly INVITES: GroupMeInvitesEndpoints = new GroupMeInvitesEndpoints();
}

class GroupEndpoints {
  readonly SEARCH: string = '/api/v1/groups/search';
  readonly OPEN_THE_DOOR: string = '/api/v1/groups/open_the_door';
  readonly CREATE: string = '/api/v1/groups/create';
  readonly UPDATE: string = '/api/v1/groups/update';
  readonly DISMISS: string = '/api/v1/groups/dismiss';
  readonly MEMBERS: GroupMembersEndpoints = new GroupMembersEndpoints();
  readonly APPLICATIONS: GroupApplicationsEndpoints = new GroupApplicationsEndpoints();
  readonly ANNOUNCEMENTS: GroupAnnouncementsEndpoints = new GroupAnnouncementsEndpoints();
  readonly ME: GroupMeEndpoints = new GroupMeEndpoints();
}

// Moment
class MomentMeEndpoints {
  readonly LIST: string = '/api/v1/moments/me/list';
  readonly LIST_FAVORITES: string = '/api/v1/moments/me/list_favorites';
  readonly LIKE: string = '/api/v1/moments/me/like';
  readonly UNLIKE: string = '/api/v1/moments/me/unlike';
  readonly FAVORITE: string = '/api/v1/moments/me/favorite';
  readonly UNFAVORITE: string = '/api/v1/moments/me/unfavorite';
}

class MomentEndpoints {
  readonly SEARCH: string = '/api/v1/moments/search';
  readonly OPEN_THE_DOOR: string = '/api/v1/moments/open_the_door';
  readonly CREATE: string = '/api/v1/moments/create';
  readonly UPDATE: string = '/api/v1/moments/update';
  readonly DELETE: string = '/api/v1/moments/delete';
  readonly LIST_LIKES: string = '/api/v1/moments/list_likes';
  readonly ME: MomentMeEndpoints = new MomentMeEndpoints();
}

// Chat
class PrivateChatsMeEndpoints {
  readonly LIST: string = '/api/v1/private_chats/me/list';
  readonly UPDATE_SETTINGS: string = '/api/v1/private_chats/me/update_settings';
  // ⏳ 延期实现: readonly DELETE: string = '/api/v1/private_chats/me/delete';
}

class PrivateChatsEndpoints {
  readonly OPEN_THE_DOOR: string = '/api/v1/private_chats/open_the_door';
  readonly ME: PrivateChatsMeEndpoints = new PrivateChatsMeEndpoints();
}

class PrivateMessagesMeEndpoints {
  readonly MARK_AS_READ: string = '/api/v1/private_messages/me/mark_as_read';
  readonly MARK_ALL_READ: string = '/api/v1/private_messages/me/mark_all_read';
}

class PrivateMessagesEndpoints {
  readonly SEND: string = '/api/v1/private_messages/send';
  readonly GET_HISTORY: string = '/api/v1/private_messages/get_history';
  readonly UPDATE: string = '/api/v1/private_messages/update';
  readonly RECALL: string = '/api/v1/private_messages/recall';
  readonly ME: PrivateMessagesMeEndpoints = new PrivateMessagesMeEndpoints();
}

class GroupMessagesMeEndpoints {
  readonly MARK_AS_READ: string = '/api/v1/group_messages/me/mark_as_read';
  readonly MARK_ALL_READ: string = '/api/v1/group_messages/me/mark_all_read';
}

class GroupMessagesEndpoints {
  readonly SEND: string = '/api/v1/group_messages/send';
  readonly GET_HISTORY: string = '/api/v1/group_messages/get_history';
  readonly UPDATE: string = '/api/v1/group_messages/update';
  readonly RECALL: string = '/api/v1/group_messages/recall';
  readonly ME: GroupMessagesMeEndpoints = new GroupMessagesMeEndpoints();
}

class ChatEndpoints {
  readonly PRIVATE_CHATS: PrivateChatsEndpoints = new PrivateChatsEndpoints();
  readonly PRIVATE_MESSAGES: PrivateMessagesEndpoints = new PrivateMessagesEndpoints();
  readonly GROUP_MESSAGES: GroupMessagesEndpoints = new GroupMessagesEndpoints();
}

// Search
class SearchEndpoints {
  readonly SUGGEST: string = '/api/v1/search/suggest';
}

// Feed
class FeedEndpoints {
  readonly LIST_MOMENTS: string = '/api/v1/feed/list_moments';
  readonly LIST_GROUPS: string = '/api/v1/feed/list_groups';
  readonly LIST_USERS: string = '/api/v1/feed/list_users';
  readonly LIST_TRENDING: string = '/api/v1/feed/list_trending';
  readonly ME_DISMISS: string = '/api/v1/feed/me/dismiss';
}

// Upload
class UploadEndpoints {
  readonly PRESIGN_UPLOAD: string = '/api/v1/media/presign_upload';
  readonly BATCH_PRESIGN_UPLOAD: string = '/api/v1/media/batch_presign_upload';
  readonly COMPLETE_UPLOAD: string = '/api/v1/media/complete_upload';
}

// System
class SystemEndpoints {
  readonly GET_META: string = '/api/v1/system/get_meta';
  readonly HEALTH: string = '/api/v1/system/health';
  readonly LIST_ANNOUNCEMENTS: string = '/api/v1/system/list_announcements';
  readonly GET_CLIENT_CONFIG: string = '/api/v1/system/get_client_config';
}

// Comment
class CommentMeEndpoints {
  readonly LIKE: string = '/api/v1/comments/me/like';
  readonly UNLIKE: string = '/api/v1/comments/me/unlike';
}

class CommentEndpoints {
  readonly CREATE: string = '/api/v1/comments/create';
  readonly DELETE: string = '/api/v1/comments/delete';
  readonly LIST: string = '/api/v1/comments/list';
  readonly LIST_REPLIES: string = '/api/v1/comments/list_replies';
  readonly LIST_LIKES: string = '/api/v1/comments/list_likes';
  readonly ME: CommentMeEndpoints = new CommentMeEndpoints();
}

// Map
class MapEndpoints {
  readonly RENDER: string = '/api/v1/map/render';
}

// Notification
class NotificationsEndpoints {
  readonly LIST: string = '/api/v1/notifications/list';
  readonly UNREAD_COUNT: string = '/api/v1/notifications/unread_count';
  readonly MARK_AS_READ: string = '/api/v1/notifications/mark_as_read';
  readonly MARK_ALL_READ: string = '/api/v1/notifications/mark_all_read';
}

// Report
class ReportMeEndpoints {
  readonly LIST: string = '/api/v1/reports/me/list';
}

class ReportEndpoints {
  readonly CREATE: string = '/api/v1/reports/create';
  readonly ME: ReportMeEndpoints = new ReportMeEndpoints();
}

// Share
class ShareMeEndpoints {
  readonly LIST: string = '/api/v1/share/me/list';
}

class ShareEndpoints {
  readonly CREATE: string = '/api/v1/share/create';
  readonly GET: string = '/api/v1/share/get';
  readonly REVOKE: string = '/api/v1/share/revoke';
  readonly ME: ShareMeEndpoints = new ShareMeEndpoints();
}

// Main entry
export class ApiEndpoints {
  static readonly AUTH: AuthEndpoints = new AuthEndpoints();
  static readonly USER: UserEndpoints = new UserEndpoints();
  static readonly GROUP: GroupEndpoints = new GroupEndpoints();
  static readonly MOMENT: MomentEndpoints = new MomentEndpoints();
  static readonly CHAT: ChatEndpoints = new ChatEndpoints();
  static readonly SEARCH: SearchEndpoints = new SearchEndpoints();
  static readonly FEED: FeedEndpoints = new FeedEndpoints();
  static readonly UPLOAD: UploadEndpoints = new UploadEndpoints();
  static readonly SYSTEM: SystemEndpoints = new SystemEndpoints();
  static readonly COMMENT: CommentEndpoints = new CommentEndpoints();
  static readonly MAP: MapEndpoints = new MapEndpoints();
  static readonly NOTIFICATIONS: NotificationsEndpoints = new NotificationsEndpoints();
  static readonly REPORT: ReportEndpoints = new ReportEndpoints();
  static readonly SHARE: ShareEndpoints = new ShareEndpoints();
}

export function assertApiEndpointPath(path: string): void {
  if (!path.startsWith('/api/')) {
    throw new Error(`ApiEndpoints path must start with "/api/": ${path}`);
  }
}
