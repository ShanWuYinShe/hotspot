// Centralized API endpoint registry for HarmonyOS client.
// Endpoint values MUST match spec/api_spec_*.md.
// Do NOT add endpoints here unless they exist in the spec.
//
// v2: All /me/ path segments removed, action names normalised per migration doc.

// Auth
class AuthEndpoints {
  readonly LOGIN: string = '/public/v1/auth/signin';
  readonly REGISTER: string = '/public/v1/auth/register';
  readonly LOGOUT: string = '/api/v1/auth/signout';
  readonly GUEST: string = '/public/v1/auth/guest';
  readonly REFRESH_TOKEN: string = '/public/v1/auth/refresh_token';
  readonly GET_STATUS: string = '/api/v1/auth/session';
  readonly REQUEST_PASSWORD_RESET: string = '/public/v1/auth/forgot_password';
  readonly RESET_PASSWORD: string = '/public/v1/auth/reset_password';
  readonly CHANGE_PASSWORD: string = '/api/v1/auth/change_password';
  readonly LIST_SESSIONS: string = '/api/v1/auth/fetch_sessions';
  readonly REVOKE_SESSION: string = '/api/v1/auth/revoke_session';
  readonly REVOKE_ALL_SESSIONS: string = '/api/v1/auth/revoke_other_sessions';
}

// User
class UserEndpoints {
  readonly SEARCH: string = '/api/v1/users/search';
  readonly GET_DETAIL: string = '/api/v1/users/open_the_door';
  readonly BATCH_GET: string = '/api/v1/users/batch_fetch';
  readonly LIST_FRIENDS: string = '/api/v1/users/fetch_friends';
  readonly LIST_FOLLOWING: string = '/api/v1/users/fetch_following';
  readonly LIST_FOLLOWERS: string = '/api/v1/users/fetch_followers';
  readonly LIST_GROUPS: string = '/api/v1/users/fetch_groups';
  readonly LIST_MOMENTS: string = '/api/v1/users/fetch_moments';
  readonly LIST_REACTED_MOMENTS: string = '/api/v1/users/fetch_reacted_moments';
  readonly GET_ACCOUNT: string = '/api/v1/users/account';
  readonly UPDATE_ACCOUNT: string = '/api/v1/users/update_account';
  readonly GET_SETTINGS: string = '/api/v1/users/settings';
  readonly UPDATE_SETTINGS: string = '/api/v1/users/update_settings';
  readonly UPDATE_LOCATION: string = '/api/v1/users/update_location';
  readonly UPGRADE: string = '/api/v1/users/upgrade';
  readonly DELETE_ACCOUNT: string = '/api/v1/users/delete_account';
  readonly FOLLOW: string = '/api/v1/users/set_follow';
  readonly BLOCK: string = '/api/v1/users/set_block';
  readonly LIST_BLOCKED: string = '/api/v1/users/fetch_blocked';
  readonly SET_REMARK: string = '/api/v1/users/set_remark';
  readonly EMAIL_REQUEST_CHANGE: string = '/api/v1/users/email/request_change';
  readonly EMAIL_CONFIRM_CHANGE: string = '/api/v1/users/email/confirm_change';
  readonly PHONE_REQUEST_CHANGE: string = '/api/v1/users/phone/request_change';
  readonly PHONE_CONFIRM_CHANGE: string = '/api/v1/users/phone/confirm_change';
}

// Group
class GroupMembersEndpoints {
  readonly LIST: string = '/api/v1/groups/members/fetch';
  readonly SEARCH: string = '/api/v1/groups/members/search';
  readonly INVITE: string = '/api/v1/groups/admin/invites/create';
  readonly INVITES_LIST: string = '/api/v1/groups/admin/invites/fetch_sent';
  readonly INVITES_REVOKE: string = '/api/v1/groups/admin/invites/revoke';
  readonly SET_ROLE: string = '/api/v1/groups/admin/members/set_role';
  readonly SET_MUTE: string = '/api/v1/groups/admin/members/set_mute';
  readonly BAN: string = '/api/v1/groups/admin/members/set_ban';
  readonly UNBAN: string = '/api/v1/groups/admin/members/set_ban';
  readonly LIST_BANNED: string = '/api/v1/groups/admin/members/fetch_banned';
  readonly REMOVE: string = '/api/v1/groups/admin/members/remove';
}

class GroupApplicationsEndpoints {
  readonly LIST: string = '/api/v1/groups/admin/applications/fetch';
  readonly APPROVE: string = '/api/v1/groups/admin/applications/approve';
  readonly REJECT: string = '/api/v1/groups/admin/applications/reject';
}

class GroupAnnouncementsEndpoints {
  readonly CREATE: string = '/api/v1/groups/admin/announcements/create';
  readonly LIST: string = '/public/v1/groups/announcements/fetch';
  readonly UPDATE: string = '/api/v1/groups/admin/announcements/update';
  readonly DELETE: string = '/api/v1/groups/admin/announcements/delete';
}

class GroupEndpoints {
  readonly SEARCH: string = '/api/v1/groups/search';
  readonly GET_DETAIL: string = '/api/v1/groups/open_the_door';
  readonly CREATE: string = '/api/v1/groups/create';
  readonly UPDATE: string = '/api/v1/groups/admin/update';
  readonly DISMISS: string = '/api/v1/groups/admin/dismiss';
  readonly LIST_JOINED: string = '/api/v1/groups/fetch_joined';
  readonly JOIN: string = '/api/v1/groups/enter';
  readonly LEAVE: string = '/api/v1/groups/exit';
  readonly TRANSFER_OWNERSHIP: string = '/api/v1/groups/admin/transfer_ownership';
  readonly UPDATE_PREFERENCES: string = '/api/v1/groups/update_preferences';
  readonly LIST_SUBMITTED_APPLICATIONS: string = '/api/v1/groups/applications/fetch_submitted';
  readonly CANCEL_APPLICATION: string = '/api/v1/groups/applications/cancel';
  readonly LIST_INVITES: string = '/api/v1/groups/invites/fetch_received';
  readonly ACCEPT_INVITE: string = '/api/v1/groups/invites/accept';
  readonly REJECT_INVITE: string = '/api/v1/groups/invites/reject';
  readonly ADMIN_RECALL_MESSAGE: string = '/api/v1/groups/admin/messages/recall';
  readonly MEMBERS: GroupMembersEndpoints = new GroupMembersEndpoints();
  readonly APPLICATIONS: GroupApplicationsEndpoints = new GroupApplicationsEndpoints();
  readonly ANNOUNCEMENTS: GroupAnnouncementsEndpoints = new GroupAnnouncementsEndpoints();
}

// Moment
class MomentEndpoints {
  readonly SEARCH: string = '/api/v1/moments/search';
  readonly GET_DETAIL: string = '/api/v1/moments/open_the_door';
  readonly CREATE: string = '/api/v1/moments/create';
  readonly UPDATE: string = '/api/v1/moments/update';
  readonly DELETE: string = '/api/v1/moments/delete';
  readonly LIST_CREATED: string = '/api/v1/moments/fetch_created';
  readonly FAVORITE: string = '/api/v1/moments/set_favorite';
  readonly LIST_FAVORITES: string = '/api/v1/moments/fetch_favorites';
}

// Comment
class CommentEndpoints {
  readonly CREATE: string = '/api/v1/comments/create';
  readonly DELETE: string = '/api/v1/comments/delete';
  readonly LIST: string = '/api/v1/comments/fetch';
  readonly LIST_REPLIES: string = '/api/v1/comments/fetch_replies';
  readonly GET_DETAIL: string = '/api/v1/comments/open_the_door';
}

// Reactions (new namespace)
class ReactionsEndpoints {
  readonly SET: string = '/api/v1/reactions/set';
  readonly LIST: string = '/api/v1/reactions/fetch';
}

// Chat
class PrivateChatsEndpoints {
  readonly LIST: string = '/api/v1/private_chats/fetch';
  readonly GET_DETAIL: string = '/api/v1/private_chats/open_the_door';
  readonly UPDATE_PREFERENCES: string = '/api/v1/private_chats/update_preferences';
  readonly DELETE: string = '/api/v1/private_chats/delete';
}

class PrivateMessagesEndpoints {
  readonly SEND: string = '/api/v1/private_messages/send';
  readonly LIST: string = '/api/v1/private_messages/fetch';
  readonly UPDATE: string = '/api/v1/private_messages/update';
  readonly RECALL: string = '/api/v1/private_messages/recall';
  readonly MARK_READ: string = '/api/v1/private_messages/mark_read';
}

class GroupMessagesEndpoints {
  readonly SEND: string = '/api/v1/group_messages/send';
  readonly LIST: string = '/api/v1/group_messages/fetch';
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
  readonly NEARBY: string = '/api/v1/feed/nearby_moments_moments';
  readonly NEARBY_USERS: string = '/api/v1/feed/nearby_moments_users';
  readonly NEARBY_GROUPS: string = '/api/v1/feed/nearby_moments_groups';
  readonly TRENDING: string = '/api/v1/feed/trending';
}

// Upload
class UploadEndpoints {
  readonly PRESIGN_UPLOAD: string = '/api/v1/media/presign_upload';
  readonly BATCH_PRESIGN_UPLOAD: string = '/api/v1/media/presign_upload';
  readonly COMPLETE_UPLOAD: string = '/api/v1/media/complete_upload';
  readonly BATCH_COMPLETE_UPLOAD: string = '/api/v1/media/complete_upload';
}

// System
class SystemEndpoints {
  readonly GET_META: string = '/public/v1/system/meta';
  readonly HEALTH: string = '/public/v1/system/health';
  readonly LIST_ANNOUNCEMENTS: string = '/public/v1/system/announcements/fetch';
  readonly GET_CLIENT_CONFIG: string = '/public/v1/system/client_config';
  readonly BOOTSTRAP: string = '/api/v1/system/bootstrap';
}

// Map
class MapEndpoints {
  readonly CANVAS: string = '/public/v1/map/canvas';
}

// Notify (renamed from notifications)
class NotifyEndpoints {
  readonly LIST: string = '/api/v1/notify/fetch';
  readonly GET_UNREAD_COUNT: string = '/api/v1/notify/unread_summary';
  readonly MARK_READ: string = '/api/v1/notify/mark_read';
}

// Report
class ReportEndpoints {
  readonly CREATE: string = '/api/v1/reports/create';
  readonly LIST: string = '/api/v1/reports/list';
}

// Share
class ShareEndpoints {
  readonly CREATE: string = '/api/v1/share/create';
  readonly GET: string = '/public/v1/share/open_the_door';
  readonly REVOKE: string = '/api/v1/share/revoke';
  readonly LIST: string = '/api/v1/share/fetch_created';
}

// Admin
class AdminRbacEndpoints {
  readonly GET_ME: string = '/admin/v1/rbac/me';
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
  if (!path.startsWith('/api/') && !path.startsWith('/admin/') && !path.startsWith('/public/')) {
    throw new Error(`ApiEndpoints path must start with "/api/", "/admin/" or "/public/": ${path}`);
  }
}
