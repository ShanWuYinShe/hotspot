// API endpoint registry — MUST match spec/api_spec_*.md.
// 节点：/public/v1（纯匿名）、/api/v1（注册用户）、/api/v1/groups/admin（群管理守卫子树）、/admin/api/v1（平台管理节点）。

const PUBLIC_PREFIX = '/public/v1';
const PREFIX = '/api/v1';
const ADMIN_PREFIX = '/admin/api/v1';

export const Endpoints = {
  Auth: {
    Register: PUBLIC_PREFIX + '/auth/register',
    SendRegisterCode: PUBLIC_PREFIX + '/auth/send_register_code',
    Login: PUBLIC_PREFIX + '/auth/signin',
    Guest: PUBLIC_PREFIX + '/auth/guest',
    RefreshToken: PUBLIC_PREFIX + '/auth/refresh_token',
    RequestPasswordReset: PUBLIC_PREFIX + '/auth/forgot_password',
    ResetPassword: PUBLIC_PREFIX + '/auth/reset_password',
    Session: PREFIX + '/auth/session',
    Signout: PREFIX + '/auth/signout',
    ChangePassword: PREFIX + '/auth/change_password',
    FetchSessions: PREFIX + '/auth/fetch_sessions',
    RevokeSession: PREFIX + '/auth/revoke_session',
    RevokeOtherSessions: PREFIX + '/auth/revoke_other_sessions',
  },

  Users: {
    Search: PREFIX + '/users/search',
    GetDetail: PREFIX + '/users/open_the_door',
    BatchFetch: PREFIX + '/users/batch_fetch',
    ListFriends: PREFIX + '/users/fetch_friends',
    ListFollowing: PREFIX + '/users/fetch_following',
    ListFollowers: PREFIX + '/users/fetch_followers',
    ListGroups: PREFIX + '/users/fetch_groups',
    ListMoments: PREFIX + '/users/fetch_moments',
    ListReactedMoments: PREFIX + '/users/fetch_reacted_moments',
    GetAccount: PREFIX + '/users/account',
    UpdateAccount: PREFIX + '/users/update_account',
    GetSettings: PREFIX + '/users/settings',
    UpdateSettings: PREFIX + '/users/update_settings',
    UpdateLocation: PREFIX + '/users/update_location',
    Upgrade: PREFIX + '/users/upgrade',
    DeleteAccount: PREFIX + '/users/delete_account',
    Follow: PREFIX + '/users/set_follow',
    Block: PREFIX + '/users/set_block',
    ListBlocked: PREFIX + '/users/fetch_blocked',
    SetRemark: PREFIX + '/users/set_remark',
  },

  Groups: {
    Search: PREFIX + '/groups/search',
    GetDetail: PREFIX + '/groups/open_the_door',
    Create: PREFIX + '/groups/create',
    Join: PREFIX + '/groups/enter',
    Leave: PREFIX + '/groups/exit',
    ListJoined: PREFIX + '/groups/fetch_joined',
    UpdatePreferences: PREFIX + '/groups/update_preferences',
    ListSubmittedApplications: PREFIX + '/groups/applications/fetch_submitted',
    CancelApplication: PREFIX + '/groups/applications/cancel',
    ListReceivedInvites: PREFIX + '/groups/invites/fetch_received',
    AcceptInvite: PREFIX + '/groups/invites/accept',
    RejectInvite: PREFIX + '/groups/invites/reject',
    Members: {
      List: PREFIX + '/groups/members/fetch',
      Search: PREFIX + '/groups/members/search',
    },
    Announcements: {
      List: PUBLIC_PREFIX + '/groups/announcements/fetch',
    },
    Admin: {
      Update: PREFIX + '/groups/admin/update',
      GetSettings: PREFIX + '/groups/admin/settings',
      Dismiss: PREFIX + '/groups/admin/dismiss',
      TransferOwnership: PREFIX + '/groups/admin/transfer_ownership',
      Invites: {
        Create: PREFIX + '/groups/admin/invites/create',
        ListSent: PREFIX + '/groups/admin/invites/fetch_sent',
        Revoke: PREFIX + '/groups/admin/invites/revoke',
      },
      Members: {
        List: PREFIX + '/groups/admin/members/fetch',
        SetRole: PREFIX + '/groups/admin/members/set_role',
        SetMute: PREFIX + '/groups/admin/members/set_mute',
        SetBan: PREFIX + '/groups/admin/members/set_ban',
        ListBanned: PREFIX + '/groups/admin/members/fetch_banned',
        Remove: PREFIX + '/groups/admin/members/remove',
      },
      Applications: {
        List: PREFIX + '/groups/admin/applications/fetch',
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
    ListCreated: PREFIX + '/moments/fetch_created',
    FetchFollowing: PREFIX + '/moments/fetch_following',
    FetchNearby: PREFIX + '/moments/fetch_nearby',
    Favorite: PREFIX + '/moments/set_favorite',
    ListFavorites: PREFIX + '/moments/fetch_favorites',
  },

  Comments: {
    Create: PREFIX + '/comments/create',
    Delete: PREFIX + '/comments/delete',
    List: PREFIX + '/comments/fetch',
    ListReplies: PREFIX + '/comments/fetch_replies',
    GetDetail: PREFIX + '/comments/open_the_door',
  },

  Reactions: {
    Set: PREFIX + '/reactions/set',
    List: PREFIX + '/reactions/fetch',
  },

  Media: {
    PresignUpload: PREFIX + '/media/presign_upload',
    CompleteUpload: PREFIX + '/media/complete_upload',
  },

  PrivateChats: {
    List: PREFIX + '/private_chats/fetch',
    GetDetail: PREFIX + '/private_chats/open_the_door',
    UpdatePreferences: PREFIX + '/private_chats/update_preferences',
    Delete: PREFIX + '/private_chats/delete',
  },
  PrivateMessages: {
    Send: PREFIX + '/private_messages/send',
    List: PREFIX + '/private_messages/fetch',
    Update: PREFIX + '/private_messages/update',
    Recall: PREFIX + '/private_messages/recall',
    MarkRead: PREFIX + '/private_messages/mark_read',
  },
  GroupMessages: {
    Send: PREFIX + '/group_messages/send',
    List: PREFIX + '/group_messages/fetch',
    Update: PREFIX + '/group_messages/update',
    Recall: PREFIX + '/group_messages/recall',
    MarkRead: PREFIX + '/group_messages/mark_read',
  },

  Search: {
    Suggest: PREFIX + '/search/suggest',
  },

  System: {
    GetMeta: PUBLIC_PREFIX + '/system/meta',
    Health: PUBLIC_PREFIX + '/system/health',
    ListAnnouncements: PUBLIC_PREFIX + '/system/announcements/fetch',
    GetClientConfig: PUBLIC_PREFIX + '/system/client_config',
    Bootstrap: PREFIX + '/system/bootstrap',
  },

  Map: {
    Canvas: PUBLIC_PREFIX + '/map/canvas',
  },

  Notify: {
    List: PREFIX + '/notify/fetch',
    GetUnreadCount: PREFIX + '/notify/unread_summary',
    MarkRead: PREFIX + '/notify/mark_read',
  },

  Share: {
    Create: PREFIX + '/share/create',
    Get: PUBLIC_PREFIX + '/share/open_the_door',
    List: PREFIX + '/share/fetch_created',
    Revoke: PREFIX + '/share/revoke',
  },

  Reports: {
    Create: PREFIX + '/reports/create',
    List: PREFIX + '/reports/fetch',
  },

  // 平台管理节点（/admin/api/v1/*，仅平台管理员）
  Admin: {
    Rbac: {
      GetMe: ADMIN_PREFIX + '/rbac/me',
      PermissionsFetch: ADMIN_PREFIX + '/rbac/permissions/fetch',
      RolesFetch: ADMIN_PREFIX + '/rbac/roles/fetch',
      RolesCreate: ADMIN_PREFIX + '/rbac/roles/create',
      RolesUpdate: ADMIN_PREFIX + '/rbac/roles/update',
      RolesDelete: ADMIN_PREFIX + '/rbac/roles/delete',
      RolesSetPermissions: ADMIN_PREFIX + '/rbac/roles/set_permissions',
      AdminsFetch: ADMIN_PREFIX + '/rbac/admins/fetch',
      AdminsAward: ADMIN_PREFIX + '/rbac/admins/award',
      AdminsRevoke: ADMIN_PREFIX + '/rbac/admins/revoke',
    },
    Audit: {
      Fetch: ADMIN_PREFIX + '/audit/fetch',
    },
    AppConfig: {
      Get: ADMIN_PREFIX + '/app_config',
      Update: ADMIN_PREFIX + '/app_config/update',
    },
    Users: {
      Search: ADMIN_PREFIX + '/users/search',
      OpenTheDoor: ADMIN_PREFIX + '/users/open_the_door',
      SetBan: ADMIN_PREFIX + '/users/set_ban',
    },
    Groups: {
      Search: ADMIN_PREFIX + '/groups/search',
      OpenTheDoor: ADMIN_PREFIX + '/groups/open_the_door',
      Dismiss: ADMIN_PREFIX + '/groups/dismiss',
    },
    Reports: {
      Fetch: ADMIN_PREFIX + '/reports/fetch',
      Judge: ADMIN_PREFIX + '/reports/judge',
    },
    Tags: {
      Search: ADMIN_PREFIX + '/tags/search',
      SetBan: ADMIN_PREFIX + '/tags/set_ban',
    },
    Announcements: {
      Fetch: ADMIN_PREFIX + '/announcements/fetch',
      Create: ADMIN_PREFIX + '/announcements/create',
      Update: ADMIN_PREFIX + '/announcements/update',
      Delete: ADMIN_PREFIX + '/announcements/delete',
    },
  },

  // 匿名双版本端点：未登录时走 /public/v1，登录后走 /api/v1（由 client 层切换）
  Public: {
    Users: {
      Search: PUBLIC_PREFIX + '/users/search',
      GetDetail: PUBLIC_PREFIX + '/users/open_the_door',
      BatchFetch: PUBLIC_PREFIX + '/users/batch_fetch',
      ListFriends: PUBLIC_PREFIX + '/users/fetch_friends',
      ListFollowing: PUBLIC_PREFIX + '/users/fetch_following',
      ListFollowers: PUBLIC_PREFIX + '/users/fetch_followers',
      ListGroups: PUBLIC_PREFIX + '/users/fetch_groups',
      ListMoments: PUBLIC_PREFIX + '/users/fetch_moments',
      ListReactedMoments: PUBLIC_PREFIX + '/users/fetch_reacted_moments',
    },
    Groups: {
      Search: PUBLIC_PREFIX + '/groups/search',
      GetDetail: PUBLIC_PREFIX + '/groups/open_the_door',
      Members: {
        List: PUBLIC_PREFIX + '/groups/members/fetch',
        Search: PUBLIC_PREFIX + '/groups/members/search',
      },
      Announcements: {
        List: PUBLIC_PREFIX + '/groups/announcements/fetch',
      },
    },
    Moments: {
      Search: PUBLIC_PREFIX + '/moments/search',
      GetDetail: PUBLIC_PREFIX + '/moments/open_the_door',
      FetchNearby: PUBLIC_PREFIX + '/moments/fetch_nearby',
    },
    Comments: {
      List: PUBLIC_PREFIX + '/comments/fetch',
      ListReplies: PUBLIC_PREFIX + '/comments/fetch_replies',
      GetDetail: PUBLIC_PREFIX + '/comments/open_the_door',
    },
    Reactions: {
      List: PUBLIC_PREFIX + '/reactions/fetch',
    },
    Search: {
      Suggest: PUBLIC_PREFIX + '/search/suggest',
    },
    System: {
      GetMeta: PUBLIC_PREFIX + '/system/meta',
      Health: PUBLIC_PREFIX + '/system/health',
      ListAnnouncements: PUBLIC_PREFIX + '/system/announcements/fetch',
      GetClientConfig: PUBLIC_PREFIX + '/system/client_config',
      Bootstrap: PUBLIC_PREFIX + '/system/bootstrap',
    },
    Map: {
      Canvas: PUBLIC_PREFIX + '/map/canvas',
    },
    Share: {
      Get: PUBLIC_PREFIX + '/share/open_the_door',
    },
  },
} as const;
