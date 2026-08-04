// API endpoint registry — MUST match spec/api_spec_*.md.

const PREFIX = '/api/v1';

export const Endpoints = {
  // Auth
  Auth: {
    Register: `${PREFIX}/auth/register`,
    Login: `${PREFIX}/auth/login`,
    Logout: `${PREFIX}/auth/logout`,
    Guest: `${PREFIX}/auth/guest`,
    RefreshToken: `${PREFIX}/auth/refresh_token`,
    CheckStatus: `${PREFIX}/auth/check_status`,
    RequestPasswordReset: `${PREFIX}/auth/request_password_reset`,
    ResetPassword: `${PREFIX}/auth/reset_password`,
    ChangePassword: `${PREFIX}/auth/change_password`,
    ListSessions: `${PREFIX}/auth/list_sessions`,
    RevokeSession: `${PREFIX}/auth/revoke_session`,
    RevokeAllSessions: `${PREFIX}/auth/revoke_all_sessions`,
  },

  // Users
  Users: {
    Search: `${PREFIX}/users/search`,
    GetDetail: `${PREFIX}/users/get_detail`,
    GetProfile: `${PREFIX}/users/get_profile`,
    BatchGet: `${PREFIX}/users/batch_get`,
    ListFriends: `${PREFIX}/users/list_friends`,
    ListFollowing: `${PREFIX}/users/list_following`,
    ListFollowers: `${PREFIX}/users/list_followers`,
    ListGroups: `${PREFIX}/users/list_groups`,
    ListMoments: `${PREFIX}/users/list_moments`,
    ListReactedMoments: `${PREFIX}/users/list_reacted_moments`,
    GetAccount: `${PREFIX}/users/get_account`,
    UpdateAccount: `${PREFIX}/users/update_account`,
    GetSettings: `${PREFIX}/users/get_settings`,
    UpdateSettings: `${PREFIX}/users/update_settings`,
    UpdateLocation: `${PREFIX}/users/update_location`,
    Upgrade: `${PREFIX}/users/upgrade`,
    DeleteAccount: `${PREFIX}/users/delete_account`,
    Follow: `${PREFIX}/users/follow`,
    Block: `${PREFIX}/users/block`,
    ListBlocked: `${PREFIX}/users/list_blocked`,
    SetRemark: `${PREFIX}/users/set_remark`,
  },

  // Groups
  Groups: {
    Search: `${PREFIX}/groups/search`,
    GetDetail: `${PREFIX}/groups/get_detail`,
    Create: `${PREFIX}/groups/create`,
    Update: `${PREFIX}/groups/update`,
    Dismiss: `${PREFIX}/groups/dismiss`,
    ListMy: `${PREFIX}/groups/list_my`,
    Join: `${PREFIX}/groups/join`,
    Leave: `${PREFIX}/groups/leave`,
    TransferOwnership: `${PREFIX}/groups/transfer_ownership`,
    UpdatePreferences: `${PREFIX}/groups/update_preferences`,
    ListApplications: `${PREFIX}/groups/list_applications`,
    CancelApplication: `${PREFIX}/groups/cancel_application`,
    ListInvites: `${PREFIX}/groups/list_invites`,
    AcceptInvite: `${PREFIX}/groups/accept_invite`,
    RejectInvite: `${PREFIX}/groups/reject_invite`,
    Members: {
      List: `${PREFIX}/groups/members/list`,
      Search: `${PREFIX}/groups/members/search`,
      Invite: `${PREFIX}/groups/members/invite`,
      SetRole: `${PREFIX}/groups/members/set_role`,
      SetMute: `${PREFIX}/groups/members/set_mute`,
      Ban: `${PREFIX}/groups/members/ban`,
      Unban: `${PREFIX}/groups/members/unban`,
      ListBanned: `${PREFIX}/groups/members/list_banned`,
      Remove: `${PREFIX}/groups/members/remove`,
      Invites: {
        List: `${PREFIX}/groups/members/invites/list`,
        Revoke: `${PREFIX}/groups/members/invites/revoke`,
      },
    },
    Applications: {
      List: `${PREFIX}/groups/applications/list`,
      Approve: `${PREFIX}/groups/applications/approve`,
      Reject: `${PREFIX}/groups/applications/reject`,
    },
    Announcements: {
      Create: `${PREFIX}/groups/announcements/create`,
      List: `${PREFIX}/groups/announcements/list`,
      Update: `${PREFIX}/groups/announcements/update`,
      Delete: `${PREFIX}/groups/announcements/delete`,
    },
  },

  // Moments
  Moments: {
    Search: `${PREFIX}/moments/search`,
    GetDetail: `${PREFIX}/moments/get_detail`,
    Create: `${PREFIX}/moments/create`,
    Update: `${PREFIX}/moments/update`,
    Delete: `${PREFIX}/moments/delete`,
    ListMy: `${PREFIX}/moments/list_my`,
    Favorite: `${PREFIX}/moments/favorite`,
    ListFavorites: `${PREFIX}/moments/list_favorites`,
  },

  // Comments
  Comments: {
    Create: `${PREFIX}/comments/create`,
    Delete: `${PREFIX}/comments/delete`,
    List: `${PREFIX}/comments/list`,
    ListReplies: `${PREFIX}/comments/list_replies`,
    GetDetail: `${PREFIX}/comments/get_detail`,
  },

  // Reactions
  Reactions: {
    Set: `${PREFIX}/reactions/set`,
    List: `${PREFIX}/reactions/list`,
  },

  // Chat
  PrivateChats: {
    List: `${PREFIX}/private_chats/list`,
    GetDetail: `${PREFIX}/private_chats/get_detail`,
    UpdatePreferences: `${PREFIX}/private_chats/update_preferences`,
    Delete: `${PREFIX}/private_chats/delete`,
  },
  PrivateMessages: {
    Send: `${PREFIX}/private_messages/send`,
    GetHistory: `${PREFIX}/private_messages/get_history`,
    Update: `${PREFIX}/private_messages/update`,
    Recall: `${PREFIX}/private_messages/recall`,
    MarkRead: `${PREFIX}/private_messages/mark_read`,
  },
  GroupMessages: {
    Send: `${PREFIX}/group_messages/send`,
    GetHistory: `${PREFIX}/group_messages/get_history`,
    Update: `${PREFIX}/group_messages/update`,
    Recall: `${PREFIX}/group_messages/recall`,
    MarkRead: `${PREFIX}/group_messages/mark_read`,
  },

  // Feed
  Feed: {
    Following: `${PREFIX}/feed/following`,
    Nearby: `${PREFIX}/feed/nearby`,
    NearbyUsers: `${PREFIX}/feed/nearby_users`,
    NearbyGroups: `${PREFIX}/feed/nearby_groups`,
    Trending: `${PREFIX}/feed/trending`,
  },

  // Search
  Search: {
    Suggest: `${PREFIX}/search/suggest`,
  },

  // Media
  Media: {
    PresignUpload: `${PREFIX}/media/presign_upload`,
    BatchPresignUpload: `${PREFIX}/media/batch_presign_upload`,
    CompleteUpload: `${PREFIX}/media/complete_upload`,
    BatchCompleteUpload: `${PREFIX}/media/batch_complete_upload`,
  },

  // System
  System: {
    GetMeta: `${PREFIX}/system/get_meta`,
    Health: `${PREFIX}/system/health`,
    ListAnnouncements: `${PREFIX}/system/list_announcements`,
    GetClientConfig: `${PREFIX}/system/get_client_config`,
    Bootstrap: `${PREFIX}/system/bootstrap`,
  },

  // Map
  Map: {
    Canvas: `${PREFIX}/map/canvas`,
  },

  // Notify
  Notify: {
    List: `${PREFIX}/notify/list`,
    GetUnreadCount: `${PREFIX}/notify/get_unread_count`,
    MarkRead: `${PREFIX}/notify/mark_read`,
  },

  // Reports
  Reports: {
    Create: `${PREFIX}/reports/create`,
    ListMy: `${PREFIX}/reports/list_my`,
  },

  // Share
  Share: {
    Create: `${PREFIX}/share/create`,
    Get: `${PREFIX}/share/get`,
    Revoke: `${PREFIX}/share/revoke`,
    ListMy: `${PREFIX}/share/list_my`,
  },

  // Admin
  Admin: {
    Rbac: {
      GetMe: `${PREFIX}/admin/rbac/get_me`,
    },
  },

  // Events（spec 尚未定义，仅 MSW mock 提供；后端契约就绪后按 spec 收编）
  Events: {
    GetDetail: `${PREFIX}/events/get_detail`,
    ListSimilar: `${PREFIX}/events/list_similar`,
  },
} as const;
