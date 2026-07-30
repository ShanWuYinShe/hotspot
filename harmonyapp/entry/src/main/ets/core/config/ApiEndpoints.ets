// Centralized API endpoint registry for HarmonyOS client.
// Endpoint values MUST match spec/api_spec_*.md.
// Do NOT add endpoints here unless they exist in the spec.
//
// v2: All /me/ path segments removed, action names normalised per migration doc.

// Auth
class AuthEndpoints {
  readonly LOGIN: string = '/api/v1/auth/login';
  readonly REGISTER: string = '/api/v1/auth/register';
  readonly LOGOUT: string = '/api/v1/auth/logout';
  readonly GUEST: string = '/api/v1/auth/guest';
  readonly REFRESH_TOKEN: string = '/api/v1/auth/refresh_token';
  readonly CHECK_STATUS: string = '/api/v1/auth/check_status';
  readonly REQUEST_PASSWORD_RESET: string = '/api/v1/auth/request_password_reset';
  readonly RESET_PASSWORD: string = '/api/v1/auth/reset_password';
  readonly CHANGE_PASSWORD: string = '/api/v1/auth/change_password';
  readonly LIST_SESSIONS: string = '/api/v1/auth/list_sessions';
  readonly REVOKE_SESSION: string = '/api/v1/auth/revoke_session';
  readonly REVOKE_ALL_SESSIONS: string = '/api/v1/auth/revoke_all_sessions';
}

// User
class UserEndpoints {
  readonly SEARCH: string = '/api/v1/users/search';
  readonly GET_DETAIL: string = '/api/v1/users/get_detail';
  readonly GET_PROFILE: string = '/api/v1/users/get_profile';
  readonly BATCH_GET: string = '/api/v1/users/batch_get';
  readonly LIST_FRIENDS: string = '/api/v1/users/list_friends';
  readonly LIST_FOLLOWING: string = '/api/v1/users/list_following';
  readonly LIST_FOLLOWERS: string = '/api/v1/users/list_followers';
  readonly LIST_GROUPS: string = '/api/v1/users/list_groups';
  readonly LIST_MOMENTS: string = '/api/v1/users/list_moments';
  readonly LIST_REACTED_MOMENTS: string = '/api/v1/users/list_reacted_moments';
  readonly GET_ACCOUNT: string = '/api/v1/users/get_account';
  readonly UPDATE_ACCOUNT: string = '/api/v1/users/update_account';
  readonly GET_SETTINGS: string = '/api/v1/users/get_settings';
  readonly UPDATE_SETTINGS: string = '/api/v1/users/update_settings';
  readonly UPDATE_LOCATION: string = '/api/v1/users/update_location';
  readonly UPGRADE: string = '/api/v1/users/upgrade';
  readonly DELETE_ACCOUNT: string = '/api/v1/users/delete_account';
  readonly FOLLOW: string = '/api/v1/users/follow';
  readonly BLOCK: string = '/api/v1/users/block';
  readonly LIST_BLOCKED: string = '/api/v1/users/list_blocked';
  readonly SET_REMARK: string = '/api/v1/users/set_remark';
  readonly EMAIL_REQUEST_CHANGE: string = '/api/v1/users/email/request_change';
  readonly EMAIL_CONFIRM_CHANGE: string = '/api/v1/users/email/confirm_change';
  readonly PHONE_REQUEST_CHANGE: string = '/api/v1/users/phone/request_change';
  readonly PHONE_CONFIRM_CHANGE: string = '/api/v1/users/phone/confirm_change';
}

// Group
class GroupMembersEndpoints {
  readonly LIST: string = '/api/v1/groups/members/list';
  readonly SEARCH: string = '/api/v1/groups/members/search';
  readonly INVITE: string = '/api/v1/groups/members/invite';
  readonly INVITES_LIST: string = '/api/v1/groups/members/invites/list';
  readonly INVITES_REVOKE: string = '/api/v1/groups/members/invites/revoke';
  readonly SET_ROLE: string = '/api/v1/groups/members/set_role';
  readonly SET_MUTE: string = '/api/v1/groups/members/set_mute';
  readonly BAN: string = '/api/v1/groups/members/ban';
  readonly UNBAN: string = '/api/v1/groups/members/unban';
  readonly LIST_BANNED: string = '/api/v1/groups/members/list_banned';
  readonly REMOVE: string = '/api/v1/groups/members/remove';
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

class GroupEndpoints {
  readonly SEARCH: string = '/api/v1/groups/search';
  readonly GET_DETAIL: string = '/api/v1/groups/get_detail';
  readonly CREATE: string = '/api/v1/groups/create';
  readonly UPDATE: string = '/api/v1/groups/update';
  readonly DISMISS: string = '/api/v1/groups/dismiss';
  readonly LIST_MY: string = '/api/v1/groups/list_my';
  readonly JOIN: string = '/api/v1/groups/join';
  readonly LEAVE: string = '/api/v1/groups/leave';
  readonly TRANSFER_OWNERSHIP: string = '/api/v1/groups/transfer_ownership';
  readonly UPDATE_PREFERENCES: string = '/api/v1/groups/update_preferences';
  readonly LIST_APPLICATIONS: string = '/api/v1/groups/list_applications';
  readonly CANCEL_APPLICATION: string = '/api/v1/groups/cancel_application';
  readonly LIST_INVITES: string = '/api/v1/groups/list_invites';
  readonly ACCEPT_INVITE: string = '/api/v1/groups/accept_invite';
  readonly REJECT_INVITE: string = '/api/v1/groups/reject_invite';
  readonly MEMBERS: GroupMembersEndpoints = new GroupMembersEndpoints();
  readonly APPLICATIONS: GroupApplicationsEndpoints = new GroupApplicationsEndpoints();
  readonly ANNOUNCEMENTS: GroupAnnouncementsEndpoints = new GroupAnnouncementsEndpoints();
}

// Moment
class MomentEndpoints {
  readonly SEARCH: string = '/api/v1/moments/search';
  readonly GET_DETAIL: string = '/api/v1/moments/get_detail';
  readonly CREATE: string = '/api/v1/moments/create';
  readonly UPDATE: string = '/api/v1/moments/update';
  readonly DELETE: string = '/api/v1/moments/delete';
  readonly LIST_MY: string = '/api/v1/moments/list_my';
  readonly FAVORITE: string = '/api/v1/moments/favorite';
  readonly LIST_FAVORITES: string = '/api/v1/moments/list_favorites';
}

// Comment
class CommentEndpoints {
  readonly CREATE: string = '/api/v1/comments/create';
  readonly DELETE: string = '/api/v1/comments/delete';
  readonly LIST: string = '/api/v1/comments/list';
  readonly LIST_REPLIES: string = '/api/v1/comments/list_replies';
  readonly GET_DETAIL: string = '/api/v1/comments/get_detail';
}

// Reactions (new namespace)
class ReactionsEndpoints {
  readonly SET: string = '/api/v1/reactions/set';
  readonly LIST: string = '/api/v1/reactions/list';
}

// Chat
class PrivateChatsEndpoints {
  readonly LIST: string = '/api/v1/private_chats/list';
  readonly GET_DETAIL: string = '/api/v1/private_chats/get_detail';
  readonly UPDATE_PREFERENCES: string = '/api/v1/private_chats/update_preferences';
  readonly DELETE: string = '/api/v1/private_chats/delete';
}

class PrivateMessagesEndpoints {
  readonly SEND: string = '/api/v1/private_messages/send';
  readonly GET_HISTORY: string = '/api/v1/private_messages/get_history';
  readonly UPDATE: string = '/api/v1/private_messages/update';
  readonly RECALL: string = '/api/v1/private_messages/recall';
  readonly MARK_READ: string = '/api/v1/private_messages/mark_read';
}

class GroupMessagesEndpoints {
  readonly SEND: string = '/api/v1/group_messages/send';
  readonly GET_HISTORY: string = '/api/v1/group_messages/get_history';
  readonly UPDATE: string = '/api/v1/group_messages/update';
  readonly RECALL: string = '/api/v1/group_messages/recall';
  readonly MARK_READ: string = '/api/v1/group_messages/mark_read';
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
  readonly FOLLOWING: string = '/api/v1/feed/following';
  readonly NEARBY: string = '/api/v1/feed/nearby';
  readonly NEARBY_USERS: string = '/api/v1/feed/nearby_users';
  readonly NEARBY_GROUPS: string = '/api/v1/feed/nearby_groups';
  readonly TRENDING: string = '/api/v1/feed/trending';
}

// Upload
class UploadEndpoints {
  readonly PRESIGN_UPLOAD: string = '/api/v1/media/presign_upload';
  readonly BATCH_PRESIGN_UPLOAD: string = '/api/v1/media/batch_presign_upload';
  readonly COMPLETE_UPLOAD: string = '/api/v1/media/complete_upload';
  readonly BATCH_COMPLETE_UPLOAD: string = '/api/v1/media/batch_complete_upload';
}

// System
class SystemEndpoints {
  readonly GET_META: string = '/api/v1/system/get_meta';
  readonly HEALTH: string = '/api/v1/system/health';
  readonly LIST_ANNOUNCEMENTS: string = '/api/v1/system/list_announcements';
  readonly GET_CLIENT_CONFIG: string = '/api/v1/system/get_client_config';
  readonly BOOTSTRAP: string = '/api/v1/system/bootstrap';
}

// Map
class MapEndpoints {
  readonly CANVAS: string = '/api/v1/map/canvas';
}

// Notify (renamed from notifications)
class NotifyEndpoints {
  readonly LIST: string = '/api/v1/notify/list';
  readonly GET_UNREAD_COUNT: string = '/api/v1/notify/get_unread_count';
  readonly MARK_READ: string = '/api/v1/notify/mark_read';
}

// Report
class ReportEndpoints {
  readonly CREATE: string = '/api/v1/reports/create';
  readonly LIST_MY: string = '/api/v1/reports/list_my';
}

// Share
class ShareEndpoints {
  readonly CREATE: string = '/api/v1/share/create';
  readonly GET: string = '/api/v1/share/get';
  readonly REVOKE: string = '/api/v1/share/revoke';
  readonly LIST_MY: string = '/api/v1/share/list_my';
}

// Admin
class AdminRbacEndpoints {
  readonly GET_ME: string = '/api/v1/admin/rbac/get_me';
}

class AdminEndpoints {
  readonly RBAC: AdminRbacEndpoints = new AdminRbacEndpoints();
}

// Main entry
export class ApiEndpoints {
  static readonly AUTH: AuthEndpoints = new AuthEndpoints();
  static readonly USER: UserEndpoints = new UserEndpoints();
  static readonly GROUP: GroupEndpoints = new GroupEndpoints();
  static readonly MOMENT: MomentEndpoints = new MomentEndpoints();
  static readonly COMMENT: CommentEndpoints = new CommentEndpoints();
  static readonly REACTIONS: ReactionsEndpoints = new ReactionsEndpoints();
  static readonly CHAT: ChatEndpoints = new ChatEndpoints();
  static readonly SEARCH: SearchEndpoints = new SearchEndpoints();
  static readonly FEED: FeedEndpoints = new FeedEndpoints();
  static readonly UPLOAD: UploadEndpoints = new UploadEndpoints();
  static readonly SYSTEM: SystemEndpoints = new SystemEndpoints();
  static readonly MAP: MapEndpoints = new MapEndpoints();
  static readonly NOTIFY: NotifyEndpoints = new NotifyEndpoints();
  static readonly REPORT: ReportEndpoints = new ReportEndpoints();
  static readonly SHARE: ShareEndpoints = new ShareEndpoints();
  static readonly ADMIN: AdminEndpoints = new AdminEndpoints();
}

export function assertApiEndpointPath(path: string): void {
  if (!path.startsWith('/api/')) {
    throw new Error(`ApiEndpoints path must start with "/api/": ${path}`);
  }
}
