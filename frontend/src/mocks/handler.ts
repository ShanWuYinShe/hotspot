import {
  mockClientConfig,
  mockMyProfile,
  mockOtherUser,
  MOCK_CURRENT_USER_ID,
  mockMomentsStore,
  createMockMomentId,
  momentToItem
} from './data';

function ok(data: unknown) {
  return new Response(JSON.stringify({ code: 0, data }), {
    headers: { 'Content-Type': 'application/json' }
  });
}

function err(code: number, message: string, status = 400) {
  return new Response(JSON.stringify({ code, message }), {
    status,
    headers: { 'Content-Type': 'application/json' }
  });
}

function readBody(request: Request): Promise<Record<string, unknown>> {
  return request.text().then((text) => {
    try {
      return text ? JSON.parse(text) : {};
    } catch {
      return {};
    }
  });
}

const MOCK_GROUPS = [
  {
    group: {
      group_id: 'group_mock_001',
      name: '测试群组',
      avatar_url: 'https://api.dicebear.com/7.x/identicon/svg?seed=group1',
      members_count: 42
    },
    description: '这是一个测试群组',
    tags: ['测试', '社交'],
    has_password: false,
    location: {
      point: { latitude: 39.9042, longitude: 116.4074 },
      location_name: '北京市朝阳区',
      distance: 1200
    },
    last_activity_at: new Date(Date.now() - 600 * 1000).toISOString(),
    created_at: '2025-01-01T00:00:00Z',
    owner_id: MOCK_CURRENT_USER_ID,
    my_state: {
      role: 'owner' as const,
      is_joined: true,
      unread_count: 3,
      is_muted: false,
      is_pinned: false
    }
  },
  {
    group: {
      group_id: 'group_mock_002',
      name: '附近的群组',
      avatar_url: 'https://api.dicebear.com/7.x/identicon/svg?seed=group2',
      members_count: 128
    },
    description: '附近活跃的社交群组',
    tags: ['附近', '活动'],
    has_password: false,
    location: {
      point: { latitude: 39.9142, longitude: 116.4174 },
      location_name: '北京市东城区',
      distance: 3500
    },
    last_activity_at: new Date(Date.now() - 1800 * 1000).toISOString(),
    created_at: '2025-02-01T00:00:00Z'
  }
];

const MOCK_NOTIFICATIONS = [
  {
    notification_id: 'notif_001',
    type: 'moment_liked',
    is_read: false,
    created_at: new Date(Date.now() - 300 * 1000).toISOString(),
    payload: {
      type: 'moment_liked',
      actor: {
        user_id: 'user_mock_002',
        nickname: '其他用户',
        avatar_url: mockOtherUser.avatar_url
      },
      moment_id: 'moment_mock_001',
      moment_summary: { moment_id: 'moment_mock_001', description_preview: '这是我的一条测试动态' }
    }
  },
  {
    notification_id: 'notif_002',
    type: 'new_follower',
    is_read: false,
    created_at: new Date(Date.now() - 3600 * 1000).toISOString(),
    payload: {
      type: 'new_follower',
      actor: {
        user_id: 'user_mock_002',
        nickname: '其他用户',
        avatar_url: mockOtherUser.avatar_url
      },
      source: 'moment'
    }
  },
  {
    notification_id: 'notif_003',
    type: 'moment_commented',
    is_read: true,
    created_at: new Date(Date.now() - 7200 * 1000).toISOString(),
    payload: {
      type: 'moment_commented',
      actor: {
        user_id: 'user_mock_002',
        nickname: '其他用户',
        avatar_url: mockOtherUser.avatar_url
      },
      moment_id: 'moment_mock_001',
      moment_summary: { moment_id: 'moment_mock_001', description_preview: '这是我的一条测试动态' },
      comment_id: 'comment_001',
      comment_preview: '这是一条测试评论'
    }
  }
];

const MOCK_SETTINGS = {
  privacy_level: 'public',
  location_visible: true,
  presence_visible: true,
  social_lists_visible: true,
  groups_visible: true,
  liked_moments_visible: true,
  stats_visible: true,
  allow_follow: true,
  allow_private_chat: 'everyone',
  searchable: true,
  phone_searchable: false,
  nickname_searchable: true,
  location_searchable: true
};

const MOCK_PRIVATE_CHATS = [
  {
    other_user: {
      user_id: 'user_mock_002',
      nickname: '其他用户',
      avatar_url: mockOtherUser.avatar_url
    },
    unread_count: 2,
    is_muted: false,
    is_pinned: false,
    last_activity_at: new Date(Date.now() - 600 * 1000).toISOString(),
    can_message: true,
    presence_status: 'online',
    last_message: {
      message_id: 'msg_001',
      content: '你好！',
      type: 'text',
      sender: {
        user_id: 'user_mock_002',
        nickname: '其他用户',
        avatar_url: mockOtherUser.avatar_url
      },
      sent_at: new Date(Date.now() - 600 * 1000).toISOString()
    }
  }
];

const MOCK_SESSIONS = [
  {
    session_id: 'session_mock_001',
    device_type: 'web',
    device_name: 'Chrome on macOS',
    ip_address: '127.0.0.1',
    ip_location: '北京',
    created_at: new Date(Date.now() - 86400 * 1000).toISOString(),
    last_active_at: new Date().toISOString(),
    is_current: true
  }
];

async function handleApi(path: string, body: Record<string, unknown>): Promise<Response | null> {
  switch (path) {
    // ---- System ----
    case '/api/v1/system/get_client_config':
      return ok(mockClientConfig);
    case '/api/v1/system/get_meta':
      return ok({ api_version: 'v1', server_version: 'mock-0.0.1' });
    case '/api/v1/system/health':
      return ok({ status: 'healthy', timestamp: new Date().toISOString() });
    case '/api/v1/system/list_announcements':
      return ok({ items: [] });

    // ---- Auth ----
    case '/api/v1/auth/login': {
      const b = body as { login_id?: string; password?: string };
      if (!b.login_id || !b.password) return err(40000, 'Missing login_id or password');
      return ok({});
    }
    case '/api/v1/auth/register': {
      const b = body as { login_id?: string; nickname?: string; password?: string };
      if (!b.login_id || !b.password) return err(40000, 'Missing required fields');
      return ok({});
    }
    case '/api/v1/auth/guest':
      return ok({});
    case '/api/v1/auth/logout':
      return ok({});
    case '/api/v1/auth/refresh_token':
      return ok({});
    case '/api/v1/auth/check_status':
      return ok({ is_valid: true, expires_at: new Date(Date.now() + 3600 * 1000).toISOString() });
    case '/api/v1/auth/request_password_reset':
      return ok({});
    case '/api/v1/auth/reset_password':
      return ok({});
    case '/api/v1/auth/me/change_password':
      return ok({});
    case '/api/v1/auth/me/sessions/list':
      return ok({ items: MOCK_SESSIONS });
    case '/api/v1/auth/me/sessions/revoke':
      return ok({});
    case '/api/v1/auth/me/sessions/revoke_all':
      return ok({ revoked_count: 0 });

    // ---- Users / me ----
    case '/api/v1/users/me/profile/get':
      return ok(mockMyProfile);
    case '/api/v1/users/me/profile/update': {
      const b = body as Record<string, unknown>;
      Object.assign(mockMyProfile, b);
      return ok(mockMyProfile);
    }
    case '/api/v1/users/me/location/update':
      return ok({});
    case '/api/v1/users/me/upgrade':
      return ok({});
    case '/api/v1/users/me/settings/get':
      return ok(MOCK_SETTINGS);
    case '/api/v1/users/me/settings/update':
      return ok({});
    case '/api/v1/users/me/delete':
      return ok({});
    case '/api/v1/users/me/list_blocked':
      return ok({ items: [] });
    case '/api/v1/users/me/follow':
      return ok({});
    case '/api/v1/users/me/unfollow':
      return ok({});
    case '/api/v1/users/me/block':
      return ok({});
    case '/api/v1/users/me/unblock':
      return ok({});
    case '/api/v1/users/me/set_remark':
      return ok({});

    // ---- Users / public ----
    case '/api/v1/users/search':
      return ok({
        items: [
          {
            user: {
              user_id: 'user_mock_002',
              nickname: '其他用户',
              avatar_url: mockOtherUser.avatar_url
            },
            bio: '其他用户的动态',
            tags: ['摄影', '旅行'],
            presence_status: 'online',
            location: {
              point: { latitude: 39.9142, longitude: 116.4174 },
              location_name: '北京市东城区',
              distance: 3500
            },
            stats: {
              moments_count: 25,
              friends_count: 30,
              followers_count: 200,
              following_count: 80,
              groups_count: 5
            },
            relation: { is_following: false, is_followed_by: false, is_blocked: false },
            social_context: { mutual_friends_count: 3 }
          }
        ]
      });
    case '/api/v1/users/open_the_door': {
      const uid = (body as { user_id?: string }).user_id ?? 'user_mock_002';
      if (uid === MOCK_CURRENT_USER_ID) {
        return ok({
          user: {
            user_id: MOCK_CURRENT_USER_ID,
            nickname: mockMyProfile.nickname,
            avatar_url: mockMyProfile.avatar_url
          },
          bio: mockMyProfile.bio,
          tags: mockMyProfile.tags,
          presence_status: 'online',
          location: {
            point: mockMyProfile.location.point,
            location_name: mockMyProfile.location.location_name
          },
          stats: mockMyProfile.stats,
          relation: { is_following: false, is_followed_by: false, is_blocked: false }
        });
      }
      return ok({
        user: { user_id: uid, nickname: '其他用户', avatar_url: mockOtherUser.avatar_url },
        bio: '其他用户的动态',
        tags: ['摄影', '旅行'],
        presence_status: 'online',
        location: {
          point: { latitude: 39.9142, longitude: 116.4174 },
          location_name: '北京市东城区',
          distance: 3500
        },
        stats: {
          moments_count: 25,
          friends_count: 30,
          followers_count: 200,
          following_count: 80,
          groups_count: 5
        },
        relation: { is_following: false, is_followed_by: false, is_blocked: false }
      });
    }
    case '/api/v1/users/list_friends':
      return ok({ items: [] });
    case '/api/v1/users/list_following':
      return ok({ items: [] });
    case '/api/v1/users/list_followers':
      return ok({ items: [] });
    case '/api/v1/users/list_groups':
      return ok({ items: MOCK_GROUPS.filter((g) => g.my_state?.is_joined) });
    case '/api/v1/users/list_moments':
      return ok({
        items: Array.from(mockMomentsStore.values()).map((m) =>
          momentToItem(m, MOCK_CURRENT_USER_ID)
        )
      });
    case '/api/v1/users/list_liked_moments':
      return ok({ items: [] });

    // ---- Moments ----
    case '/api/v1/moments/search':
      return ok({
        items: Array.from(mockMomentsStore.values()).map((m) =>
          momentToItem(m, MOCK_CURRENT_USER_ID)
        )
      });
    case '/api/v1/moments/open_the_door': {
      const moment = mockMomentsStore.get((body as { moment_id: string }).moment_id);
      if (!moment) return err(40401, 'Moment not found');
      return ok(momentToItem(moment, MOCK_CURRENT_USER_ID));
    }
    case '/api/v1/moments/create': {
      const b = body as {
        description: string;
        location: { point: { latitude: number; longitude: number }; location_name: string };
        tags?: string[];
        image_file_ids?: string[];
        privacy_level?: string;
      };
      const momentId = createMockMomentId();
      const now = new Date().toISOString();
      const newMoment = {
        moment_id: momentId,
        user_id: MOCK_CURRENT_USER_ID,
        description: b.description,
        location: b.location,
        tags: b.tags ?? [],
        images: (b.image_file_ids ?? []).map((id: string) => ({
          file_id: id,
          url: `https://picsum.photos/seed/${id}/400/300`
        })),
        privacy_level: (b.privacy_level ?? 'public') as 'public' | 'private',
        created_at: now,
        updated_at: now,
        likes_count: 0,
        comments_count: 0
      };
      mockMomentsStore.set(momentId, newMoment);
      return ok(momentToItem(newMoment, MOCK_CURRENT_USER_ID));
    }
    case '/api/v1/moments/update': {
      const b = body as {
        moment_id: string;
        description?: string;
        location?: { point: { latitude: number; longitude: number }; location_name: string };
        tags?: string[];
        image_file_ids?: string[];
        privacy_level?: string;
      };
      const existing = mockMomentsStore.get(b.moment_id);
      if (!existing) return err(40401, 'Moment not found');
      const updated = {
        ...existing,
        description: b.description ?? existing.description,
        location: b.location ?? existing.location,
        tags: b.tags !== undefined ? b.tags : existing.tags,
        images:
          b.image_file_ids !== undefined
            ? b.image_file_ids.map((id: string) => ({
                file_id: id,
                url: `https://picsum.photos/seed/${id}/400/300`
              }))
            : existing.images,
        privacy_level: (b.privacy_level ?? existing.privacy_level) as 'public' | 'private',
        updated_at: new Date().toISOString()
      };
      mockMomentsStore.set(b.moment_id, updated);
      return ok(momentToItem(updated, MOCK_CURRENT_USER_ID));
    }
    case '/api/v1/moments/delete':
      mockMomentsStore.delete((body as { moment_id: string }).moment_id);
      return ok(undefined);
    case '/api/v1/moments/list_likes':
      return ok({ items: [] });

    // ---- Moments / me ----
    case '/api/v1/moments/me/like': {
      const m = mockMomentsStore.get((body as { moment_id: string }).moment_id);
      if (m) {
        m.likes_count += 1;
        m.my_state = { ...(m.my_state ?? {}), is_liked: true };
      }
      return ok(undefined);
    }
    case '/api/v1/moments/me/unlike': {
      const m = mockMomentsStore.get((body as { moment_id: string }).moment_id);
      if (m) {
        m.likes_count = Math.max(0, m.likes_count - 1);
        m.my_state = { ...(m.my_state ?? {}), is_liked: false };
      }
      return ok(undefined);
    }
    case '/api/v1/moments/me/favorite': {
      const m = mockMomentsStore.get((body as { moment_id: string }).moment_id);
      if (m) {
        m.my_state = { ...(m.my_state ?? {}), is_favorited: true };
      }
      return ok(undefined);
    }
    case '/api/v1/moments/me/unfavorite': {
      const m = mockMomentsStore.get((body as { moment_id: string }).moment_id);
      if (m) {
        m.my_state = { ...(m.my_state ?? {}), is_favorited: false };
      }
      return ok(undefined);
    }
    case '/api/v1/moments/me/list':
      return ok({
        items: Array.from(mockMomentsStore.values())
          .filter((m) => m.user_id === MOCK_CURRENT_USER_ID)
          .map((m) => momentToItem(m, MOCK_CURRENT_USER_ID))
      });
    case '/api/v1/moments/me/list_favorites':
      return ok({
        items: Array.from(mockMomentsStore.values())
          .filter((m) => m.my_state?.is_favorited)
          .map((m) => momentToItem(m, MOCK_CURRENT_USER_ID))
      });

    // ---- Comments ----
    case '/api/v1/comments/list':
      return ok({
        items: [
          {
            comment_id: 'comment_001',
            user: {
              user_id: 'user_mock_003',
              nickname: '评论者A',
              avatar_url: 'https://api.dicebear.com/7.x/avataaars/svg?seed=commenter1'
            },
            content: '这是一条测试评论',
            created_at: new Date(Date.now() - 1800 * 1000).toISOString(),
            likes_count: 0,
            replies_count: 0,
            is_liked: false
          }
        ]
      });
    case '/api/v1/comments/list_replies':
      return ok({ items: [] });
    case '/api/v1/comments/list_likes':
      return ok({ items: [] });
    case '/api/v1/comments/create': {
      const b = body as { moment_id: string; content: string };
      const m = mockMomentsStore.get(b.moment_id);
      if (m) m.comments_count += 1;
      return ok({
        comment_id: `comment_${Date.now()}`,
        user: {
          user_id: MOCK_CURRENT_USER_ID,
          nickname: mockMyProfile.nickname,
          avatar_url: mockMyProfile.avatar_url
        },
        content: b.content,
        created_at: new Date().toISOString(),
        likes_count: 0,
        replies_count: 0,
        is_liked: false
      });
    }
    case '/api/v1/comments/delete':
      return ok({});
    case '/api/v1/comments/me/like':
      return ok({});
    case '/api/v1/comments/me/unlike':
      return ok({});

    // ---- Feed ----
    case '/api/v1/feed/list_moments':
      return ok({
        items: Array.from(mockMomentsStore.values()).map((m) =>
          momentToItem(m, MOCK_CURRENT_USER_ID)
        )
      });
    case '/api/v1/feed/list_groups':
      return ok({ items: MOCK_GROUPS });
    case '/api/v1/feed/list_users':
      return ok({
        items: [
          {
            user: {
              user_id: 'user_mock_002',
              nickname: '其他用户',
              avatar_url: mockOtherUser.avatar_url
            },
            bio: '其他用户的动态',
            tags: ['摄影', '旅行'],
            presence_status: 'online',
            location: {
              point: { latitude: 39.9142, longitude: 116.4174 },
              location_name: '北京市东城区',
              distance: 3500
            },
            stats: {
              moments_count: 25,
              friends_count: 30,
              followers_count: 200,
              following_count: 80,
              groups_count: 5
            },
            relation: { is_following: false, is_followed_by: false, is_blocked: false }
          }
        ]
      });
    case '/api/v1/feed/list_trending':
      return ok({
        trending_tags: [
          { name: '测试', usage_count: 100, trend: 'rising' as const },
          { name: '美食', usage_count: 80, trend: 'stable' as const }
        ],
        hot_moments: Array.from(mockMomentsStore.values()).map((m) =>
          momentToItem(m, MOCK_CURRENT_USER_ID)
        )
      });
    case '/api/v1/feed/me/dismiss':
      return ok({});

    // ---- Search ----
    case '/api/v1/search/suggest':
      return ok({
        items: [
          { type: 'tag', payload: { name: '测试', usage_count: 100 } },
          { type: 'tag', payload: { name: '美食', usage_count: 80 } },
          {
            type: 'user',
            payload: {
              user_id: 'user_mock_002',
              nickname: '其他用户',
              avatar_url: mockOtherUser.avatar_url,
              bio: '其他用户的动态'
            }
          },
          {
            type: 'group',
            payload: { group_id: 'group_mock_001', name: '测试群组', members_count: 42 }
          }
        ]
      });

    // ---- Groups ----
    case '/api/v1/groups/search':
      return ok({ items: MOCK_GROUPS });
    case '/api/v1/groups/open_the_door': {
      const gid = (body as { group_id?: string }).group_id ?? 'group_mock_001';
      const group = MOCK_GROUPS.find((g) => g.group.group_id === gid) ?? MOCK_GROUPS[0];
      return ok(group);
    }
    case '/api/v1/groups/create':
      return ok({ group_id: `group_${Date.now()}` });
    case '/api/v1/groups/update':
      return ok({});
    case '/api/v1/groups/dismiss':
      return ok({});
    case '/api/v1/groups/members/list':
      return ok({
        items: [
          {
            user: {
              user_id: MOCK_CURRENT_USER_ID,
              nickname: mockMyProfile.nickname,
              avatar_url: mockMyProfile.avatar_url
            },
            role: 'owner',
            joined_at: '2025-01-01T00:00:00Z',
            presence_status: 'online'
          }
        ]
      });
    case '/api/v1/groups/me/list':
      return ok({ items: MOCK_GROUPS.filter((g) => g.my_state?.is_joined) });
    case '/api/v1/groups/me/join':
      return ok({ status: 'joined' });
    case '/api/v1/groups/me/leave':
      return ok({});
    case '/api/v1/groups/me/update_settings':
      return ok({});

    // ---- Group messages ----
    case '/api/v1/group_messages/send':
      return ok({ message_id: `gmsg_${Date.now()}`, sent_at: new Date().toISOString() });
    case '/api/v1/group_messages/get_history':
      return ok({ items: [] });

    // ---- Private chats ----
    case '/api/v1/private_chats/open_the_door':
      return ok({
        chat: MOCK_PRIVATE_CHATS[0],
        recent_messages: { items: [] }
      });
    case '/api/v1/private_chats/me/list':
      return ok({ items: MOCK_PRIVATE_CHATS });

    // ---- Private messages ----
    case '/api/v1/private_messages/send':
      return ok({ message_id: `pmsg_${Date.now()}`, sent_at: new Date().toISOString() });
    case '/api/v1/private_messages/get_history':
      return ok({ items: [] });
    case '/api/v1/private_messages/me/mark_as_read':
      return ok({});
    case '/api/v1/private_messages/me/mark_all_read':
      return ok({});

    // ---- Map ----
    case '/api/v1/map/canvas':
      return ok({
        heatmap: {
          items: [
            {
              location: { latitude: 39.9042, longitude: 116.4074 },
              total_count: 42,
              breakdown: { user: 10, group: 5, moment: 27 }
            }
          ]
        },
        hotspots: {
          items: Array.from(mockMomentsStore.values()).map((m) => ({
            object_type: 'moment' as const,
            object_id: m.moment_id,
            title: m.description.slice(0, 30),
            location: m.location.point,
            distance_meters: 500 + Math.floor(Math.random() * 2000)
          })),
          next_cursor: undefined
        }
      });

    // ---- Media ----
    case '/api/v1/media/presign_upload':
      return ok({
        upload_id: `upload_mock_${Date.now()}`,
        upload_url: `/api/v1/mock_upload/${Date.now()}`,
        headers: {},
        expires_at: new Date(Date.now() + 3600 * 1000).toISOString()
      });
    case '/api/v1/media/batch_presign_upload':
      return ok({
        items: [
          {
            upload_id: `upload_mock_${Date.now()}`,
            upload_url: `/api/v1/mock_upload/${Date.now()}`,
            headers: {},
            expires_at: new Date(Date.now() + 3600 * 1000).toISOString()
          }
        ]
      });
    case '/api/v1/media/complete_upload':
      return ok({
        file_id: `file_mock_${Date.now()}`,
        url: `https://picsum.photos/seed/${Date.now()}/400/300`,
        thumbnail_url: `https://picsum.photos/seed/${Date.now()}/100/100`,
        file_size: 102400,
        mime_type: 'image/jpeg'
      });

    // ---- Notifications ----
    case '/api/v1/notifications/list':
      return ok({ items: MOCK_NOTIFICATIONS });
    case '/api/v1/notifications/unread_count':
      return ok({ total: 2, by_type: { social: 1, interaction: 1, group: 0, system: 0 } });
    case '/api/v1/notifications/mark_as_read':
      return ok({});
    case '/api/v1/notifications/mark_all_read':
      return ok({});

    // ---- Reports ----
    case '/api/v1/reports/create':
      return ok({ report_id: `report_${Date.now()}` });
    case '/api/v1/reports/me/list':
      return ok({ items: [] });

    // ---- Share ----
    case '/api/v1/users/me/email/request_change':
      return ok({
        verification_sent: true,
        expires_at: new Date(Date.now() + 600 * 1000).toISOString()
      });
    case '/api/v1/users/me/email/confirm_change':
      return ok({});
    case '/api/v1/users/me/phone/request_change':
      return ok({
        verification_sent: true,
        expires_at: new Date(Date.now() + 600 * 1000).toISOString()
      });
    case '/api/v1/users/me/phone/confirm_change':
      return ok({});

    case '/api/v1/groups/members/search':
      return ok({
        items: [
          {
            user: {
              user_id: MOCK_CURRENT_USER_ID,
              nickname: mockMyProfile.nickname,
              avatar_url: mockMyProfile.avatar_url
            },
            role: 'owner' as const,
            joined_at: '2025-01-01T00:00:00Z',
            presence_status: 'online' as const
          }
        ]
      });
    case '/api/v1/groups/members/invite':
      return ok({ invite_id: `invite_${Date.now()}` });
    case '/api/v1/groups/members/invites/list':
      return ok({ items: [] });
    case '/api/v1/groups/members/invites/revoke':
      return ok({});
    case '/api/v1/groups/members/set_role':
      return ok({});
    case '/api/v1/groups/members/mute':
      return ok({});
    case '/api/v1/groups/members/unmute':
      return ok({});
    case '/api/v1/groups/members/ban':
      return ok({});
    case '/api/v1/groups/members/unban':
      return ok({});
    case '/api/v1/groups/members/list_banned':
      return ok({ items: [] });
    case '/api/v1/groups/members/remove_member':
      return ok({});
    case '/api/v1/groups/applications/list':
      return ok({ items: [] });
    case '/api/v1/groups/applications/approve':
      return ok({});
    case '/api/v1/groups/applications/reject':
      return ok({});
    case '/api/v1/groups/announcements/list':
      return ok({ items: [] });
    case '/api/v1/groups/announcements/create':
      return ok({ announcement_id: `ann_${Date.now()}` });
    case '/api/v1/groups/announcements/update':
      return ok({});
    case '/api/v1/groups/announcements/delete':
      return ok({});
    case '/api/v1/groups/me/applications/list':
      return ok({ items: [] });
    case '/api/v1/groups/me/applications/cancel':
      return ok({});
    case '/api/v1/groups/me/invites/list':
      return ok({ items: [] });
    case '/api/v1/groups/me/invites/accept':
      return ok({});
    case '/api/v1/groups/me/invites/reject':
      return ok({});
    case '/api/v1/groups/me/transfer_ownership':
      return ok({});

    case '/api/v1/group_messages/update':
      return ok({});
    case '/api/v1/group_messages/recall':
      return ok({});
    case '/api/v1/group_messages/me/mark_as_read':
      return ok({});
    case '/api/v1/group_messages/me/mark_all_read':
      return ok({});

    case '/api/v1/private_chats/me/update_settings':
      return ok({});
    case '/api/v1/private_chats/me/delete':
      return ok({});
    case '/api/v1/private_messages/update':
      return ok({});
    case '/api/v1/private_messages/recall':
      return ok({});

    case '/api/v1/admin/rbac/get_me':
      return ok({
        user: {
          user_id: MOCK_CURRENT_USER_ID,
          nickname: mockMyProfile.nickname,
          avatar_url: mockMyProfile.avatar_url
        },
        permissions: ['*'],
        roles: [{ role_id: 'role_admin', name: 'Administrator' }]
      });
    case '/api/v1/admin/rbac/list_permissions':
      return ok({ items: [{ key: '*', description: 'All permissions' }] });
    case '/api/v1/admin/rbac/roles/list':
      return ok({
        items: [
          {
            role_id: 'role_admin',
            name: 'Administrator',
            description: 'Full access',
            permissions: ['*']
          }
        ]
      });
    case '/api/v1/admin/rbac/roles/create':
      return ok({ role_id: `role_${Date.now()}` });
    case '/api/v1/admin/rbac/roles/update':
      return ok({});
    case '/api/v1/admin/rbac/roles/delete':
      return ok({});
    case '/api/v1/admin/rbac/roles/set_permissions':
      return ok({});
    case '/api/v1/admin/rbac/admins/list':
      return ok({
        items: [
          {
            user: {
              user_id: MOCK_CURRENT_USER_ID,
              nickname: mockMyProfile.nickname,
              avatar_url: mockMyProfile.avatar_url
            },
            permissions: ['*'],
            roles: [{ role_id: 'role_admin', name: 'Administrator' }]
          }
        ]
      });
    case '/api/v1/admin/rbac/admins/grant':
      return ok({});
    case '/api/v1/admin/rbac/admins/revoke':
      return ok({});

    case '/api/v1/admin/users/search':
      return ok({
        items: [
          {
            user: {
              user_id: 'user_mock_002',
              nickname: '其他用户',
              avatar_url: mockOtherUser.avatar_url
            },
            status: 'normal' as const,
            created_at: '2025-01-01T00:00:00Z',
            login_id: 'mock002'
          }
        ]
      });
    case '/api/v1/admin/users/open_the_door':
      return ok({
        user: {
          user_id: 'user_mock_002',
          nickname: '其他用户',
          avatar_url: mockOtherUser.avatar_url,
          bio: '其他用户的动态',
          tags: [],
          gender: 'female' as const,
          location: {
            point: { latitude: 39.9142, longitude: 116.4174 },
            location_name: '北京市东城区'
          },
          presence_status: 'online' as const,
          last_online: new Date().toISOString(),
          stats: mockOtherUser.stats,
          account_status: 'normal' as const,
          login_id: 'mock002',
          created_at: '2025-01-01T00:00:00Z'
        }
      });
    case '/api/v1/admin/users/ban':
      return ok({});
    case '/api/v1/admin/users/unban':
      return ok({});

    case '/api/v1/admin/groups/search':
      return ok({
        items: MOCK_GROUPS.map((g) => ({
          ...g,
          owner: { user_id: MOCK_CURRENT_USER_ID, nickname: mockMyProfile.nickname },
          admins: [],
          stats: { messages_count: 0, active_members_today: 1, reports_count: 0 }
        }))
      });
    case '/api/v1/admin/groups/open_the_door':
      return ok({
        ...MOCK_GROUPS[0],
        owner: { user_id: MOCK_CURRENT_USER_ID, nickname: mockMyProfile.nickname },
        admins: [],
        stats: { messages_count: 0, active_members_today: 1, reports_count: 0 }
      });
    case '/api/v1/admin/groups/dismiss':
      return ok({});

    case '/api/v1/admin/announcements/search':
      return ok({ items: [] });
    case '/api/v1/admin/announcements/create':
      return ok({ announcement_id: `ann_${Date.now()}` });
    case '/api/v1/admin/announcements/update':
      return ok({});
    case '/api/v1/admin/announcements/delete':
      return ok({});

    case '/api/v1/admin/audit/list':
      return ok({ items: [] });
    case '/api/v1/admin/audit/open_the_door':
      return err(40401, 'Audit log not found');

    case '/api/v1/admin/reports/search':
      return ok({ items: [] });
    case '/api/v1/admin/reports/handle':
      return ok({});

    case '/api/v1/admin/tags/search':
      return ok({
        items: [
          {
            name: '测试',
            usage_count: 100,
            trend: 'rising' as const,
            created_at: '2025-01-01T00:00:00Z'
          }
        ]
      });
    case '/api/v1/admin/tags/delete':
      return ok({});

    case '/api/v1/admin/app_config/get':
      return ok({
        ...mockClientConfig,
        upload: {
          image_max_size: 10485760,
          video_max_size: 104857600,
          audio_max_size: 52428800,
          file_max_size: 104857600,
          allowed_image_types: ['image/jpeg', 'image/png', 'image/webp'],
          allowed_video_types: ['video/mp4', 'video/quicktime']
        },
        defaults: {
          default_location: mockClientConfig.defaults.default_location,
          default_group_avatar: mockClientConfig.defaults.default_group_avatar,
          default_user_avatar: mockClientConfig.defaults.default_user_avatar,
          default_group_settings: {
            join_method: 'open' as const,
            is_searchable: true,
            max_members: 500,
            location: mockClientConfig.defaults.default_location
          }
        },
        rate_limits: {
          global_per_hour: 10000,
          user_per_hour: 1000,
          auth_register_per_10minutes: 10,
          auth_login_per_10minutes: 20,
          auth_guest_per_10minutes: 10,
          auth_refresh_per_10minutes: 100,
          auth_reset_per_10minutes: 5,
          search_per_minute: 60,
          write_per_minute: 60,
          moments_create_per_minute: 10,
          comments_create_per_minute: 30,
          reports_create_per_hour: 10,
          upload_presign_per_hour: 100,
          ws_subscribe_per_second: 50,
          admin_read_per_minute: 100,
          admin_write_per_minute: 50
        },
        response_defaults: {
          search_suggest_count: 10,
          trending_tags_count: 10,
          hot_moments_count: 20,
          chat_recent_messages_count: 30,
          group_announcements_count: 3,
          system_announcements_count: 5
        },
        updated_at: new Date().toISOString()
      });
    case '/api/v1/admin/app_config/update_upload':
    case '/api/v1/admin/app_config/update_defaults':
    case '/api/v1/admin/app_config/update_rate_limits':
    case '/api/v1/admin/app_config/update_response_defaults':
    case '/api/v1/admin/app_config/update_features':
    case '/api/v1/admin/app_config/update_limits':
    case '/api/v1/admin/app_config/update_contact':
      return ok({});

    case '/api/v1/share/create':
      return ok({ share_id: `share_${Date.now()}`, share_code: 'mock_share_code' });
    case '/api/v1/share/get':
      return ok({
        share: {
          share_id: 'share_mock_001',
          share_code: 'mock_share_code',
          share_type: 'moment',
          creator: { user_id: MOCK_CURRENT_USER_ID, nickname: mockMyProfile.nickname },
          created_at: new Date().toISOString()
        },
        payload: { shared_moment_id: 'moment_mock_001' },
        preview: { title: '测试动态', subtitle: '这是一条测试动态' }
      });
    case '/api/v1/share/me/list':
      return ok({ items: [] });
    case '/api/v1/share/revoke':
      return ok({});

    default:
      return null;
  }
}

export async function handleMockRequest(request: Request): Promise<Response | null> {
  const url = new URL(request.url);
  if (!url.pathname.startsWith('/api/v1/')) return null;

  console.log('[MockAPI]', request.method, url.pathname);
  const body = await readBody(request);
  return handleApi(url.pathname, body);
}
