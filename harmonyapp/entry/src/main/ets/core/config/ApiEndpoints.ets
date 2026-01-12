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

const API_VERSION = 'v1' as const;
const API_PREFIX = `/api/${API_VERSION}` as const;

const PublicPrefix = `${API_PREFIX}/public` as const;
const ProtectedPrefix = `${API_PREFIX}/protected` as const;

export const ApiEndpoints = {
  public: {
    auth: {
      register: `${PublicPrefix}/auth/register`,
      login: `${PublicPrefix}/auth/login`,
      guest: `${PublicPrefix}/auth/guest`,
      refreshToken: `${PublicPrefix}/auth/refresh_token`,
      requestPasswordReset: `${PublicPrefix}/auth/request_password_reset`,
      verifyResetCode: `${PublicPrefix}/auth/verify_reset_code`,
      resetPassword: `${PublicPrefix}/auth/reset_password`
    },

    user: {
      // Search / public info
      searchByKeyword: `${PublicPrefix}/user/search_by_keyword`,
      searchByTag: `${PublicPrefix}/user/search_by_tag`,
      searchByLocation: `${PublicPrefix}/user/search_by_location`,
      openTheDoor: `${PublicPrefix}/user/open_the_door`,

      // Supplemental
      stats: `${PublicPrefix}/user/stats`,

      // Relationship visibility controlled endpoints
      friends: `${PublicPrefix}/user/friends`,
      following: `${PublicPrefix}/user/following`,
      followers: `${PublicPrefix}/user/followers`,

      // Other public lists
      groups: `${PublicPrefix}/user/groups`,
      likedMoments: `${PublicPrefix}/user/liked_moments`,

      // Moments list for a specific user (public)
      moments: `${PublicPrefix}/user/moments`
    },

    moments: {
      searchByKeyword: `${PublicPrefix}/moments/search_by_keyword`,
      searchByTag: `${PublicPrefix}/moments/search_by_tag`,
      searchByLocation: `${PublicPrefix}/moments/search_by_location`,
      openTheDoor: `${PublicPrefix}/moments/open_the_door`,
      comments: `${PublicPrefix}/moments/comments`,
      likes: `${PublicPrefix}/moments/likes`
    },

    comments: {
      list: `${PublicPrefix}/comments`,
      replies: `${PublicPrefix}/comments/replies`,
      likes: `${PublicPrefix}/comments/likes`
    },

    groupChats: {
      searchByKeyword: `${PublicPrefix}/group_chats/search_by_keyword`,
      searchByTag: `${PublicPrefix}/group_chats/search_by_tag`,
      searchByLocation: `${PublicPrefix}/group_chats/search_by_location`,
      openTheDoor: `${PublicPrefix}/group_chats/open_the_door`,
      settings: `${PublicPrefix}/group_chats/settings`,
      members: `${PublicPrefix}/group_chats/members`
    },

    tags: {
      trending: `${PublicPrefix}/tags/trending`,
      suggest: `${PublicPrefix}/tags/suggest`
    },

    system: {
      health: `${PublicPrefix}/system/health`
    }
  },

  protected: {
    auth: {
      logout: `${ProtectedPrefix}/auth/logout`,
      checkStatus: `${ProtectedPrefix}/auth/check_status`,
      me: {
        changePassword: `${ProtectedPrefix}/auth/me/change_password`,
        sessions: `${ProtectedPrefix}/auth/me/sessions`,
        revokeSession: `${ProtectedPrefix}/auth/me/sessions/revoke`
      }
    },

    user: {
      relationship: `${ProtectedPrefix}/user/relationship`,
      me: {
        profile: `${ProtectedPrefix}/user/me/profile`,
        updateProfile: `${ProtectedPrefix}/user/me/profile/update`,
        location: `${ProtectedPrefix}/user/me/location`,
        upgrade: `${ProtectedPrefix}/user/me/upgrade`,
        stats: `${ProtectedPrefix}/user/me/stats`,
        settings: `${ProtectedPrefix}/user/me/settings`,
        updateSettings: `${ProtectedPrefix}/user/me/settings/update`,
        deleteAccount: `${ProtectedPrefix}/user/me/delete`,
        follow: `${ProtectedPrefix}/user/me/follow`,
        unfollow: `${ProtectedPrefix}/user/me/unfollow`,
        block: `${ProtectedPrefix}/user/me/block`,
        unblock: `${ProtectedPrefix}/user/me/unblock`,
        friends: `${ProtectedPrefix}/user/me/friends`,
        following: `${ProtectedPrefix}/user/me/following`,
        followers: `${ProtectedPrefix}/user/me/followers`,
        blocked: `${ProtectedPrefix}/user/me/blocked`,
        remark: `${ProtectedPrefix}/user/me/remark`
      }
    },

    moments: {
      create: `${ProtectedPrefix}/moments/create`,
      update: `${ProtectedPrefix}/moments/update`,
      delete: `${ProtectedPrefix}/moments/delete`,
      like: `${ProtectedPrefix}/moments/like`,
      unlike: `${ProtectedPrefix}/moments/unlike`,
      favorite: `${ProtectedPrefix}/moments/favorite`,
      unfavorite: `${ProtectedPrefix}/moments/unfavorite`,
      me: `${ProtectedPrefix}/moments/me`,
      myFavorites: `${ProtectedPrefix}/moments/me/favorites`,
      checkLiked: `${ProtectedPrefix}/moments/me/liked`,
      checkFavorited: `${ProtectedPrefix}/moments/me/favorited`
    },

    comments: {
      create: `${ProtectedPrefix}/comments/create`,
      delete: `${ProtectedPrefix}/comments/delete`,
      like: `${ProtectedPrefix}/comments/like`,
      unlike: `${ProtectedPrefix}/comments/unlike`
    },

    groupChats: {
      create: `${ProtectedPrefix}/group_chats/create`,
      update: `${ProtectedPrefix}/group_chats/update`,
      delete: `${ProtectedPrefix}/group_chats/delete`,
      members: {
        invite: `${ProtectedPrefix}/group_chats/members/invite`,
        join: `${ProtectedPrefix}/group_chats/members/join`,
        apply: `${ProtectedPrefix}/group_chats/members/apply`,
        setRole: `${ProtectedPrefix}/group_chats/members/set_role`,
        mute: `${ProtectedPrefix}/group_chats/members/mute`,
        unmute: `${ProtectedPrefix}/group_chats/members/unmute`,
        ban: `${ProtectedPrefix}/group_chats/members/ban`,
        unban: `${ProtectedPrefix}/group_chats/members/unban`,
        banned: `${ProtectedPrefix}/group_chats/members/banned`,
        removeMember: `${ProtectedPrefix}/group_chats/members/remove_member`
      },
      applications: {
        list: `${ProtectedPrefix}/group_chats/applications`,
        approve: `${ProtectedPrefix}/group_chats/applications/approve`,
        reject: `${ProtectedPrefix}/group_chats/applications/reject`,
        cancel: `${ProtectedPrefix}/group_chats/me/applications/cancel`
      },
      transferOwnership: `${ProtectedPrefix}/group_chats/transfer_ownership`,
      me: {
        list: `${ProtectedPrefix}/group_chats/me`,
        invites: `${ProtectedPrefix}/group_chats/me/invites`,
        acceptInvite: `${ProtectedPrefix}/group_chats/me/invites/accept`,
        rejectInvite: `${ProtectedPrefix}/group_chats/me/invites/reject`,
        leave: `${ProtectedPrefix}/group_chats/me/leave`
      }
    },

    privateChats: {
      me: `${ProtectedPrefix}/private_chats/me`,
      openTheDoor: `${ProtectedPrefix}/private_chats/open_the_door`,
      updateSettings: `${ProtectedPrefix}/private_chats/update_settings`,
      delete: `${ProtectedPrefix}/private_chats/delete`
    },

    privateMessages: {
      send: `${ProtectedPrefix}/private_messages/send`,
      history: `${ProtectedPrefix}/private_messages/history`,
      update: `${ProtectedPrefix}/private_messages/update`,
      recall: `${ProtectedPrefix}/private_messages/recall`,
      delete: `${ProtectedPrefix}/private_messages/delete`,
      markAsRead: `${ProtectedPrefix}/private_messages/mark_as_read`,
      addReaction: `${ProtectedPrefix}/private_messages/add_reaction`,
      removeReaction: `${ProtectedPrefix}/private_messages/remove_reaction`,
      reactions: `${ProtectedPrefix}/private_messages/reactions`
    },

    groupMessages: {
      send: `${ProtectedPrefix}/group_messages/send`,
      history: `${ProtectedPrefix}/group_messages/history`,
      update: `${ProtectedPrefix}/group_messages/update`,
      recall: `${ProtectedPrefix}/group_messages/recall`,
      delete: `${ProtectedPrefix}/group_messages/delete`,
      markAsRead: `${ProtectedPrefix}/group_messages/mark_as_read`,
      readStatus: `${ProtectedPrefix}/group_messages/read_status`,
      addReaction: `${ProtectedPrefix}/group_messages/add_reaction`,
      removeReaction: `${ProtectedPrefix}/group_messages/remove_reaction`,
      reactions: `${ProtectedPrefix}/group_messages/reactions`
    },

    messages: {
      markAllAsRead: `${ProtectedPrefix}/messages/mark_all_as_read`
    },

    reports: {
      create: `${ProtectedPrefix}/reports/create`,
      me: `${ProtectedPrefix}/reports/me`
    },

    subscriptions: {
      me: `${ProtectedPrefix}/subscriptions/me`,
      update: `${ProtectedPrefix}/subscriptions/me/update`
    },

    deviceTokens: {
      me: {
        register: `${ProtectedPrefix}/device_tokens/me/register`,
        unregister: `${ProtectedPrefix}/device_tokens/me/unregister`,
        list: `${ProtectedPrefix}/device_tokens/me`
      }
    },

    feed: {
      moments: `${ProtectedPrefix}/feed/moments`,
      groups: `${ProtectedPrefix}/feed/groups`,
      users: `${ProtectedPrefix}/feed/users`,
      trending: `${ProtectedPrefix}/feed/trending`,
      feedback: `${ProtectedPrefix}/feed/feedback`
    },

    media: {
      presignUpload: `${ProtectedPrefix}/media/presign_upload`,
      completeUpload: `${ProtectedPrefix}/media/complete_upload`,
      chunkedInit: `${ProtectedPrefix}/media/chunked/init`,
      chunkedFinish: `${ProtectedPrefix}/media/chunked/finish`,
      chunkedCheck: `${ProtectedPrefix}/media/chunked/check`
    },

    map: {
      render: `${PublicPrefix}/map/render`
    },

    globalSettings: {
      fetch: `${PublicPrefix}/global_settings/fetch`
    }
  }
} as const;

// Optional helper types for stronger typing at call-sites.
export type ApiEndpointsType = typeof ApiEndpoints;
export type ApiEndpointPath =
  | ApiEndpointsType['public']['auth'][keyof ApiEndpointsType['public']['auth']]
  | ApiEndpointsType['public']['user'][keyof ApiEndpointsType['public']['user']]
  | ApiEndpointsType['public']['moments'][keyof ApiEndpointsType['public']['moments']]
  | ApiEndpointsType['public']['comments'][keyof ApiEndpointsType['public']['comments']]
  | ApiEndpointsType['public']['groupChats'][keyof ApiEndpointsType['public']['groupChats']]
  | ApiEndpointsType['public']['tags'][keyof ApiEndpointsType['public']['tags']]
  | ApiEndpointsType['public']['system'][keyof ApiEndpointsType['public']['system']]
  | ApiEndpointsType['protected']['auth'][keyof ApiEndpointsType['protected']['auth']]
  | ApiEndpointsType['protected']['auth']['me'][keyof ApiEndpointsType['protected']['auth']['me']]
  | ApiEndpointsType['protected']['user'][keyof ApiEndpointsType['protected']['user']]
  | ApiEndpointsType['protected']['user']['me'][keyof ApiEndpointsType['protected']['user']['me']]
  | ApiEndpointsType['protected']['moments'][keyof ApiEndpointsType['protected']['moments']]
  | ApiEndpointsType['protected']['comments'][keyof ApiEndpointsType['protected']['comments']]
  | ApiEndpointsType['protected']['groupChats'][keyof ApiEndpointsType['protected']['groupChats']]
  | ApiEndpointsType['protected']['groupChats']['members'][keyof ApiEndpointsType['protected']['groupChats']['members']]
  | ApiEndpointsType['protected']['groupChats']['applications'][keyof ApiEndpointsType['protected']['groupChats']['applications']]
  | ApiEndpointsType['protected']['groupChats']['me'][keyof ApiEndpointsType['protected']['groupChats']['me']]
  | ApiEndpointsType['protected']['privateChats'][keyof ApiEndpointsType['protected']['privateChats']]
  | ApiEndpointsType['protected']['privateMessages'][keyof ApiEndpointsType['protected']['privateMessages']]
  | ApiEndpointsType['protected']['groupMessages'][keyof ApiEndpointsType['protected']['groupMessages']]
  | ApiEndpointsType['protected']['messages'][keyof ApiEndpointsType['protected']['messages']]
  | ApiEndpointsType['protected']['reports'][keyof ApiEndpointsType['protected']['reports']]
  | ApiEndpointsType['protected']['subscriptions'][keyof ApiEndpointsType['protected']['subscriptions']]
  | ApiEndpointsType['protected']['deviceTokens']['me'][keyof ApiEndpointsType['protected']['deviceTokens']['me']]
  | ApiEndpointsType['protected']['feed'][keyof ApiEndpointsType['protected']['feed']]
  | ApiEndpointsType['protected']['media'][keyof ApiEndpointsType['protected']['media']]
  | ApiEndpointsType['public']['map'][keyof ApiEndpointsType['public']['map']]
  | ApiEndpointsType['public']['globalSettings'][keyof ApiEndpointsType['public']['globalSettings']];

// Runtime assertion helper (optional): useful during development to ensure paths are absolute API paths.
export function assertApiEndpointPath(path: string): void {
  // Keep this very lightweight: no throw in production if you prefer; adjust as needed.
  if (!path.startsWith('/api/')) {
    throw new Error(`ApiEndpoints path must start with "/api/": ${path}`);
  }
}
