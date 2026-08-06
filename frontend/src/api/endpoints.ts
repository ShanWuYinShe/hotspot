// API endpoint registry — MUST match spec/api_spec_*.md.
// 块注释标注 client.ts 已包装的端点；未标注的为「待接线」——有 UI 需求时
// 补 client 包装（spec 是契约权威，登记表必须完整，不删无消费端点）。

const PREFIX = '/api/v1';

export const Endpoints = {
  // Auth — 已包装: Register/Login/Logout/Guest/RefreshToken/CheckStatus/
  //   RequestPasswordReset/ResetPassword；待接线: ChangePassword/ListSessions/
  //   RevokeSession/RevokeAllSessions
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

  // Users — 已包装: Search/GetProfile/ListFriends/ListFollowing/ListFollowers/
  //   ListMoments/ListReactedMoments/GetAccount/UpdateAccount/GetSettings/Follow/Block；
  //   待接线: GetDetail/BatchGet/ListGroups/UpdateSettings/UpdateLocation/Upgrade/
  //   DeleteAccount/ListBlocked/SetRemark
  Users: {
    Search: `${PREFIX}/users/search`,
    GetDetail: `${PREFIX}/users/open_the_door`,
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
    Follow: `${PREFIX}/users/set_follow`,
    Block: `${PREFIX}/users/set_block`,
    ListBlocked: `${PREFIX}/users/list_blocked`,
    SetRemark: `${PREFIX}/users/set_remark`,
  },

  // Groups — 已包装: GetDetail/Create/ListMy/Join/Leave/Members.List/Announcements.List；
  //   待接线: Search/Update/Dismiss/TransferOwnership/UpdatePreferences/ListApplications/
  //   CancelApplication/ListInvites/AcceptInvite/RejectInvite/Members.*/Applications.*/
  //   Announcements.Create|Update|Delete
  Groups: {
    Search: `${PREFIX}/groups/search`,
    GetDetail: `${PREFIX}/groups/open_the_door`,
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
    ListInvites: `${PREFIX}/groups/invites/received/list`,
    AcceptInvite: `${PREFIX}/groups/invites/received/accept`,
    RejectInvite: `${PREFIX}/groups/invites/received/reject`,
    Members: {
      List: `${PREFIX}/groups/members/list`,
      Search: `${PREFIX}/groups/members/search`,
      Invite: `${PREFIX}/groups/members/invite`,
      SetRole: `${PREFIX}/groups/members/set_role`,
      SetMute: `${PREFIX}/groups/members/set_mute`,
      Ban: `${PREFIX}/groups/members/set_ban`,
      Unban: `${PREFIX}/groups/members/set_ban`,
      ListBanned: `${PREFIX}/groups/members/list_banned`,
      Remove: `${PREFIX}/groups/members/remove`,
      Invites: {
        List: `${PREFIX}/groups/invites/sent/list`,
        Revoke: `${PREFIX}/groups/invites/sent/revoke`,
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

  // Moments — 已包装: Search/GetDetail/Create/ListMy/Favorite/ListFavorites；
  //   待接线: Update/Delete
  Moments: {
    Search: `${PREFIX}/moments/search`,
    GetDetail: `${PREFIX}/moments/open_the_door`,
    Create: `${PREFIX}/moments/create`,
    Update: `${PREFIX}/moments/update`,
    Delete: `${PREFIX}/moments/delete`,
    ListMy: `${PREFIX}/moments/list_my`,
    Favorite: `${PREFIX}/moments/set_favorite`,
    ListFavorites: `${PREFIX}/moments/list_favorites`,
  },

  // Comments — 已包装: List/Create；待接线: Delete/ListReplies/GetDetail
  Comments: {
    Create: `${PREFIX}/comments/create`,
    Delete: `${PREFIX}/comments/delete`,
    List: `${PREFIX}/comments/list`,
    ListReplies: `${PREFIX}/comments/list_replies`,
    GetDetail: `${PREFIX}/comments/open_the_door`,
  },

  // Reactions — 已包装: Set；待接线: List
  Reactions: {
    Set: `${PREFIX}/reactions/set`,
    List: `${PREFIX}/reactions/list`,
  },

  // Chat — 已包装: PrivateChats.List/PrivateMessages.Send|GetHistory|MarkRead/
  //   GroupMessages.Send|GetHistory|MarkRead；
  //   待接线: PrivateChats.GetDetail|UpdatePreferences|Delete、消息 Update|Recall
  PrivateChats: {
    List: `${PREFIX}/private_chats/list`,
    GetDetail: `${PREFIX}/private_chats/open_the_door`,
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

  // Feed — 已包装: Nearby/NearbyUsers/NearbyGroups/Trending；待接线: Following
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

  // System — 已包装: GetClientConfig/Bootstrap；待接线: GetMeta/Health/ListAnnouncements
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

  // Notify — 已包装: List/GetUnreadCount/MarkRead
  Notify: {
    List: `${PREFIX}/notify/list`,
    GetUnreadCount: `${PREFIX}/notify/get_unread_count`,
    MarkRead: `${PREFIX}/notify/mark_read`,
  },

  // Reports — 已包装: Create；待接线: ListMy
  Reports: {
    Create: `${PREFIX}/reports/create`,
    ListMy: `${PREFIX}/reports/list_my`,
  },

  // Admin
  Admin: {
    Rbac: {
      GetMe: `${PREFIX}/admin/rbac/get_me`,
    },
  },
} as const;
