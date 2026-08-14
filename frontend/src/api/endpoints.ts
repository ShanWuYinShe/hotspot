// API endpoint registry — MUST match spec/api_spec_*.md（v1 重设计后）。
// 三位置：/api/v1（客户端）、/api/v1/groups/admin（群管理守卫子树）、/admin/v1（平台管理节点）。

const PREFIX = '/api/v1';
const ADMIN_PREFIX = '/admin/v1';

export const Endpoints = {
  Auth: {
    Register: PREFIX + '/auth/register',
    Login: PREFIX + '/auth/login',
    Logout: PREFIX + '/auth/logout',
    Guest: PREFIX + '/auth/guest',
    RefreshToken: PREFIX + '/auth/refresh_token',
    GetStatus: PREFIX + '/auth/get_status',
    RequestPasswordReset: PREFIX + '/auth/request_password_reset',
    ResetPassword: PREFIX + '/auth/reset_password',
    ChangePassword: PREFIX + '/auth/change_password',
    ListSessions: PREFIX + '/auth/list_sessions',
    RevokeSession: PREFIX + '/auth/revoke_session',
    RevokeAllSessions: PREFIX + '/auth/revoke_all_sessions',
  },

  Users: {
    Search: PREFIX + '/users/search',
    GetDetail: PREFIX + '/users/open_the_door',
    BatchGet: PREFIX + '/users/batch_get',
    ListFriends: PREFIX + '/users/list_friends',
    ListFollowing: PREFIX + '/users/list_following',
    ListFollowers: PREFIX + '/users/list_followers',
    ListGroups: PREFIX + '/users/list_groups',
    ListMoments: PREFIX + '/users/list_moments',
    ListReactedMoments: PREFIX + '/users/list_reacted_moments',
    GetAccount: PREFIX + '/users/get_account',
    UpdateAccount: PREFIX + '/users/update_account',
    GetSettings: PREFIX + '/users/get_settings',
    UpdateSettings: PREFIX + '/users/update_settings',
    UpdateLocation: PREFIX + '/users/update_location',
    Upgrade: PREFIX + '/users/upgrade',
    DeleteAccount: PREFIX + '/users/delete_account',
    Follow: PREFIX + '/users/set_follow',
    Block: PREFIX + '/users/set_block',
    ListBlocked: PREFIX + '/users/list_blocked',
    SetRemark: PREFIX + '/users/set_remark',
  },

  Groups: {
    // 公开/成员路径
    Search: PREFIX + '/groups/search',
    GetDetail: PREFIX + '/groups/open_the_door',
    Create: PREFIX + '/groups/create',
    Join: PREFIX + '/groups/join',
    Leave: PREFIX + '/groups/leave',
    ListJoined: PREFIX + '/groups/list_joined',
    UpdatePreferences: PREFIX + '/groups/update_preferences',
    ListSubmittedApplications: PREFIX + '/groups/applications/list_submitted',
    CancelApplication: PREFIX + '/groups/applications/cancel',
    ListReceivedInvites: PREFIX + '/groups/invites/list_received',
    AcceptInvite: PREFIX + '/groups/invites/accept',
    RejectInvite: PREFIX + '/groups/invites/reject',
    Members: {
      List: PREFIX + '/groups/members/list',
      Search: PREFIX + '/groups/members/search',
    },
    Announcements: {
      List: PREFIX + '/groups/announcements/list',
    },
    // 群管理守卫子树（/api/v1/groups/admin/*，调用者须为群主/群管理员）
    Admin: {
      Update: PREFIX + '/groups/admin/update',
      GetSettings: PREFIX + '/groups/admin/get_settings',
      Dismiss: PREFIX + '/groups/admin/dismiss',
      TransferOwnership: PREFIX + '/groups/admin/transfer_ownership',
      Invites: {
        Create: PREFIX + '/groups/admin/invites/create',
        ListSent: PREFIX + '/groups/admin/invites/list_sent',
        Revoke: PREFIX + '/groups/admin/invites/revoke',
      },
      Members: {
        List: PREFIX + '/groups/admin/members/list',
        SetRole: PREFIX + '/groups/admin/members/set_role',
        SetMute: PREFIX + '/groups/admin/members/set_mute',
        SetBan: PREFIX + '/groups/admin/members/set_ban',
        ListBanned: PREFIX + '/groups/admin/members/list_banned',
        Remove: PREFIX + '/groups/admin/members/remove',
      },
      Applications: {
        List: PREFIX + '/groups/admin/applications/list',
        Approve: PREFIX + '/groups/admin/applications/approve',
        Reject: PREFIX + '/groups/admin/applications/reject',
      },
      Announcements: {
        Create: PREFIX + '/groups/admin/announcements/create',
        Update: PREFIX + '/groups/admin/announcements/update',
        Delete: PREFIX + '/groups/admin/announcements/delete',
      },
      Messages: {
        Recall: PREFIX + '/groups/admin/messages/recall',
      },
    },
  },

  Moments: {
    Search: PREFIX + '/moments/search',
    GetDetail: PREFIX + '/moments/open_the_door',
    Create: PREFIX + '/moments/create',
    Update: PREFIX + '/moments/update',
    Delete: PREFIX + '/moments/delete',
    ListCreated: PREFIX + '/moments/list_created',
    Favorite: PREFIX + '/moments/set_favorite',
    ListFavorites: PREFIX + '/moments/list_favorites',
  },

  Comments: {
    Create: PREFIX + '/comments/create',
    Delete: PREFIX + '/comments/delete',
    List: PREFIX + '/comments/list',
    ListReplies: PREFIX + '/comments/list_replies',
    GetDetail: PREFIX + '/comments/open_the_door',
  },

  Reactions: {
    Set: PREFIX + '/reactions/set',
    List: PREFIX + '/reactions/list',
  },

  Media: {
    PresignUpload: PREFIX + '/media/presign_upload',
    CompleteUpload: PREFIX + '/media/complete_upload',
    BatchPresignUpload: PREFIX + '/media/batch_presign_upload',
    BatchCompleteUpload: PREFIX + '/media/batch_complete_upload',
  },

  PrivateChats: {
    List: PREFIX + '/private_chats/list',
    GetDetail: PREFIX + '/private_chats/open_the_door',
    UpdatePreferences: PREFIX + '/private_chats/update_preferences',
    Delete: PREFIX + '/private_chats/delete',
  },
  PrivateMessages: {
    Send: PREFIX + '/private_messages/send',
    List: PREFIX + '/private_messages/list',
    Update: PREFIX + '/private_messages/update',
    Recall: PREFIX + '/private_messages/recall',
    MarkRead: PREFIX + '/private_messages/mark_read',
  },
  GroupMessages: {
    Send: PREFIX + '/group_messages/send',
    List: PREFIX + '/group_messages/list',
    Update: PREFIX + '/group_messages/update',
    Recall: PREFIX + '/group_messages/recall',
    MarkRead: PREFIX + '/group_messages/mark_read',
  },

  Feed: {
    Following: PREFIX + '/feed/following',
    Nearby: PREFIX + '/feed/nearby',
    NearbyUsers: PREFIX + '/feed/nearby_users',
    NearbyGroups: PREFIX + '/feed/nearby_groups',
    Trending: PREFIX + '/feed/trending',
  },

  Search: {
    Suggest: PREFIX + '/search/suggest',
  },

  System: {
    GetMeta: PREFIX + '/system/get_meta',
    Health: PREFIX + '/system/health',
    ListAnnouncements: PREFIX + '/system/announcements/list',
    GetClientConfig: PREFIX + '/system/get_client_config',
    Bootstrap: PREFIX + '/system/bootstrap',
  },

  Map: {
    Canvas: PREFIX + '/map/canvas',
  },

  Notify: {
    List: PREFIX + '/notify/list',
    GetUnreadCount: PREFIX + '/notify/get_unread_count',
    MarkRead: PREFIX + '/notify/mark_read',
  },

  Share: {
    Create: PREFIX + '/share/create',
    Get: PREFIX + '/share/get',
    List: PREFIX + '/share/list',
    Revoke: PREFIX + '/share/revoke',
  },

  Reports: {
    Create: PREFIX + '/reports/create',
    List: PREFIX + '/reports/list',
  },

  // 平台管理节点（/admin/v1/*，仅平台管理员）
  Admin: {
    Rbac: {
      GetMe: ADMIN_PREFIX + '/rbac/get_me',
    },
  },
} as const;
