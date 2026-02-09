// hotspot/harmonyapp/entry/src/main/ets/core/config/ApiEndpoints.ets
//
// Centralized API endpoint registry for HarmonyOS client.
//
// Purpose:
// - Prevent hardcoded '/api/v1/...' strings from being scattered around the codebase.
// - Keep endpoint usage consistent and discoverable.
// - Make it easy to audit, update, and refactor endpoints.
//
// Authority:
// - Endpoint values MUST match `@api_spec.mdc` ("API 端点与类型定义").
// - Do NOT add endpoints here unless they exist in the spec.

export class ApiEndpoints {
  static readonly AUTH = new class {
    readonly LOGIN = '/api/v1/auth/login';
    readonly REGISTER = '/api/v1/auth/register';
    readonly LOGOUT = '/api/v1/auth/logout';
    readonly GUEST = '/api/v1/auth/guest';
    readonly REFRESH_TOKEN = '/api/v1/auth/refresh_token';
    readonly CHECK_STATUS = '/api/v1/auth/check_status';
    readonly ME = new class {
      readonly CHANGE_PASSWORD = '/api/v1/auth/me/change_password';
      readonly SESSIONS = new class {
        readonly LIST = '/api/v1/auth/me/sessions/list';
        readonly REVOKE = '/api/v1/auth/me/sessions/revoke';
        readonly REVOKE_ALL = '/api/v1/auth/me/sessions/revoke_all';
      }();
    }();
  }();

  static readonly USER = new class {
    readonly SEARCH = '/api/v1/users/search';
    readonly OPEN_THE_DOOR = '/api/v1/users/open_the_door';
    readonly LIST_FRIENDS = '/api/v1/users/list_friends';
    readonly LIST_FOLLOWING = '/api/v1/users/list_following';
    readonly LIST_FOLLOWERS = '/api/v1/users/list_followers';
    readonly LIST_GROUPS = '/api/v1/users/list_groups';
    readonly LIST_MOMENTS = '/api/v1/users/list_moments';
    readonly LIST_LIKED_MOMENTS = '/api/v1/users/list_liked_moments';
    readonly ME = new class {
      readonly PROFILE = new class {
        readonly GET = '/api/v1/users/me/profile/get';
        readonly UPDATE = '/api/v1/users/me/profile/update';
      }();
      readonly LOCATION = new class {
        readonly UPDATE = '/api/v1/users/me/location/update';
      }();
      readonly UPGRADE = '/api/v1/users/me/upgrade';
      readonly SETTINGS = new class {
        readonly GET = '/api/v1/users/me/settings/get';
        readonly UPDATE = '/api/v1/users/me/settings/update';
      }();
      readonly DELETE = '/api/v1/users/me/delete';
      readonly FOLLOW = '/api/v1/users/me/follow';
      readonly UNFOLLOW = '/api/v1/users/me/unfollow';
      readonly BLOCK = '/api/v1/users/me/block';
      readonly UNBLOCK = '/api/v1/users/me/unblock';
      readonly LIST_FRIENDS = '/api/v1/users/me/list_friends';
      readonly LIST_FOLLOWING = '/api/v1/users/me/list_following';
      readonly LIST_FOLLOWERS = '/api/v1/users/me/list_followers';
      readonly LIST_BLOCKED = '/api/v1/users/me/list_blocked';
      readonly SET_REMARK = '/api/v1/users/me/set_remark';
    }();
  }();

  static readonly GROUP = new class {
    readonly SEARCH = '/api/v1/groups/search';
    readonly OPEN_THE_DOOR = '/api/v1/groups/open_the_door';
    readonly CREATE = '/api/v1/groups/create';
    readonly UPDATE = '/api/v1/groups/update';
    readonly DISMISS = '/api/v1/groups/dismiss';
    readonly MEMBERS = new class {
      readonly LIST = '/api/v1/groups/members/list';
      readonly SEARCH = '/api/v1/groups/members/search';
      readonly INVITE = '/api/v1/groups/members/invite';
      readonly SET_ROLE = '/api/v1/groups/members/set_role';
      readonly MUTE = '/api/v1/groups/members/mute';
      readonly UNMUTE = '/api/v1/groups/members/unmute';
      readonly BAN = '/api/v1/groups/members/ban';
      readonly UNBAN = '/api/v1/groups/members/unban';
      readonly LIST_BANNED = '/api/v1/groups/members/list_banned';
      readonly REMOVE_MEMBER = '/api/v1/groups/members/remove_member';
    }();
    readonly APPLICATIONS = new class {
      readonly LIST = '/api/v1/groups/applications/list';
      readonly APPROVE = '/api/v1/groups/applications/approve';
      readonly REJECT = '/api/v1/groups/applications/reject';
    }();
    readonly ANNOUNCEMENTS = new class {
      readonly CREATE = '/api/v1/groups/announcements/create';
      readonly LIST = '/api/v1/groups/announcements/list';
      readonly UPDATE = '/api/v1/groups/announcements/update';
      readonly DELETE = '/api/v1/groups/announcements/delete';
    }();
    readonly ME = new class {
      readonly JOIN = '/api/v1/groups/me/join';
      readonly LEAVE = '/api/v1/groups/me/leave';
      readonly LIST = '/api/v1/groups/me/list';
      readonly UPDATE_SETTINGS = '/api/v1/groups/me/update_settings';
      readonly TRANSFER_OWNERSHIP = '/api/v1/groups/me/transfer_ownership';
      readonly APPLICATIONS = new class {
        readonly LIST = '/api/v1/groups/me/applications/list';
        readonly CANCEL = '/api/v1/groups/me/applications/cancel';
      }();
      readonly INVITES = new class {
        readonly LIST = '/api/v1/groups/me/invites/list';
        readonly ACCEPT = '/api/v1/groups/me/invites/accept';
        readonly REJECT = '/api/v1/groups/me/invites/reject';
      }();
    }();
  }();

  static readonly MOMENT = new class {
    readonly SEARCH = '/api/v1/moments/search';
    readonly OPEN_THE_DOOR = '/api/v1/moments/open_the_door';
    readonly CREATE = '/api/v1/moments/create';
    readonly UPDATE = '/api/v1/moments/update';
    readonly DELETE = '/api/v1/moments/delete';
    readonly LIST_LIKES = '/api/v1/moments/list_likes';
    readonly ME = new class {
      readonly LIST = '/api/v1/moments/me/list';
      readonly LIST_FAVORITES = '/api/v1/moments/me/list_favorites';
      readonly LIKE = '/api/v1/moments/me/like';
      readonly UNLIKE = '/api/v1/moments/me/unlike';
      readonly FAVORITE = '/api/v1/moments/me/favorite';
      readonly UNFAVORITE = '/api/v1/moments/me/unfavorite';
    }();
  }();

  static readonly CHAT = new class {
    readonly PRIVATE_CHATS = new class {
      readonly OPEN_THE_DOOR = '/api/v1/private_chats/open_the_door';
      readonly ME = new class {
        readonly LIST = '/api/v1/private_chats/me/list';
        readonly UPDATE_SETTINGS = '/api/v1/private_chats/me/update_settings';
        readonly DELETE = '/api/v1/private_chats/me/delete';
      }();
    }();
    readonly PRIVATE_MESSAGES = new class {
      readonly SEND = '/api/v1/private_messages/send';
      readonly GET_HISTORY = '/api/v1/private_messages/get_history';
      readonly UPDATE = '/api/v1/private_messages/update';
      readonly RECALL = '/api/v1/private_messages/recall';
      readonly ME = new class {
        readonly MARK_AS_READ = '/api/v1/private_messages/me/mark_as_read';
      }();
    }();
    readonly GROUP_MESSAGES = new class {
      readonly SEND = '/api/v1/group_messages/send';
      readonly GET_HISTORY = '/api/v1/group_messages/get_history';
      readonly UPDATE = '/api/v1/group_messages/update';
      readonly RECALL = '/api/v1/group_messages/recall';
      readonly ME = new class {
        readonly MARK_AS_READ = '/api/v1/group_messages/me/mark_as_read';
      }();
    }();
  }();

  static readonly SEARCH = new class {
    readonly SUGGEST = '/api/v1/search/suggest';
  }();

  static readonly FEED = new class {
    readonly LIST_MOMENTS = '/api/v1/feed/list_moments';
    readonly LIST_GROUPS = '/api/v1/feed/list_groups';
    readonly LIST_USERS = '/api/v1/feed/list_users';
    readonly LIST_TRENDING = '/api/v1/feed/list_trending';
    readonly ME = new class {
      readonly FEEDBACK = '/api/v1/feed/me/feedback';
    }();
  }();

  static readonly UPLOAD = new class {
    readonly PRESIGN_UPLOAD = '/api/v1/media/presign_upload';
    readonly BATCH_PRESIGN_UPLOAD = '/api/v1/media/batch_presign_upload';
    readonly COMPLETE_UPLOAD = '/api/v1/media/complete_upload';
  }();

  static readonly SYSTEM = new class {
    readonly GET_META = '/api/v1/system/get_meta';
    readonly HEALTH = '/api/v1/system/health';
    readonly LIST_ANNOUNCEMENTS = '/api/v1/system/list_announcements';
    readonly GET_CLIENT_CONFIG = '/api/v1/system/get_client_config';
  }();

  static readonly COMMENT = new class {
    readonly CREATE = '/api/v1/comments/create';
    readonly DELETE = '/api/v1/comments/delete';
    readonly LIST = '/api/v1/comments/list';
    readonly LIST_REPLIES = '/api/v1/comments/list_replies';
    readonly LIST_LIKES = '/api/v1/comments/list_likes';
    readonly ME = new class {
      readonly LIKE = '/api/v1/comments/me/like';
      readonly UNLIKE = '/api/v1/comments/me/unlike';
    }();
  }();

  static readonly MAP = new class {
    readonly RENDER = '/api/v1/map/render';
  }();

  static readonly TAG = new class {
    readonly LIST_TRENDING = '/api/v1/tags/list_trending';
  }();

  static readonly REPORT = new class {
    readonly CREATE = '/api/v1/reports/create';
    readonly ME = new class {
      readonly LIST = '/api/v1/reports/me/list';
    }();
  }();

  static readonly SHARE = new class {
    readonly CREATE = '/api/v1/share/create';
    readonly GET = '/api/v1/share/get';
    readonly APPLY = '/api/v1/share/apply';
    readonly REVOKE = '/api/v1/share/revoke';
    readonly ME = new class {
      readonly LIST = '/api/v1/share/me/list';
    }();
  }();
}

// Runtime assertion helper (optional): useful during development to ensure paths are absolute API paths.
export function assertApiEndpointPath(path: string): void {
  // Keep this very lightweight: no throw in production if you prefer; adjust as needed.
  if (!path.startsWith('/api/')) {
    throw new Error(`ApiEndpoints path must start with "/api/": ${path}`);
  }
}
