import { post, paginate, generateIdempotencyKey, type RequestOptions } from '../client';
import { GROUPS, GROUP_MESSAGES } from '../endpoints';
import type {
  ApplicationStatus,
  GroupItem,
  GroupMemberItem,
  BannedMemberItem,
  GroupAnnouncementItem,
  GroupApplicationItem,
  GroupInviteItem,
  GroupMessageItem,
  GroupSentInviteItem,
  MessageSent,
  JoinGroupResult,
  Invitation,
  Location,
  SendMessageType,
  SettableGroupRole,
  PaginationParams,
  GeoPoint,
  GroupRole
} from '@/lib/types/api';

export const groupsService = {
  search(
    params: {
      keyword?: string;
      tag?: string;
      radius?: number;
      sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';
      reference_point: GeoPoint;
      pagination?: PaginationParams;
    },
    options?: RequestOptions
  ) {
    return paginate<GroupItem>(GROUPS.SEARCH, params, options);
  },

  getGroup(data: { group_id: string }) {
    return post<GroupItem>(GROUPS.OPEN_THE_DOOR, data);
  },

  create(data: {
    name: string;
    location: Location;
    description?: string;
    avatar_file_id?: string;
    tags?: string[];
    password?: string;
    join_method?: 'open' | 'approval' | 'invite_only';
    is_searchable?: boolean;
    max_members?: number;
  }) {
    return post<GroupItem>(GROUPS.CREATE, data, { idempotencyKey: generateIdempotencyKey() });
  },

  update(data: {
    group_id: string;
    name?: string;
    description?: string;
    avatar_file_id?: string;
    tags?: string[];
    password?: string;
    clear_password?: boolean;
    join_method?: 'open' | 'approval' | 'invite_only';
    is_searchable?: boolean;
    max_members?: number;
    location?: Location;
  }) {
    return post<void>(GROUPS.UPDATE, data);
  },

  updateMySettings(data: { group_id: string; is_muted?: boolean; is_pinned?: boolean }) {
    return post<void>(GROUPS.ME.UPDATE_SETTINGS, data);
  },

  dismiss(data: { group_id: string }) {
    return post<void>(GROUPS.DISMISS, data, { idempotencyKey: generateIdempotencyKey() });
  },

  transferOwnership(data: { group_id: string; new_owner_id: string }) {
    return post<void>(GROUPS.ME.TRANSFER_OWNERSHIP, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  getMembers(params: { group_id: string; role?: GroupRole; pagination?: PaginationParams }) {
    return paginate<GroupMemberItem>(GROUPS.MEMBERS.LIST, params);
  },

  searchMembers(params: {
    group_id: string;
    keyword: string;
    role?: GroupRole;
    pagination?: PaginationParams;
  }) {
    return paginate<GroupMemberItem>(GROUPS.MEMBERS.SEARCH, params);
  },

  invite(data: { group_id: string; user_id: string; message?: string }) {
    return post<Invitation>(GROUPS.MEMBERS.INVITE, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  getSentInvites(params: {
    group_id: string;
    status?: ApplicationStatus | 'all';
    pagination?: PaginationParams;
  }) {
    return paginate<GroupSentInviteItem>(GROUPS.MEMBERS.INVITES_LIST, params);
  },

  revokeSentInvite(data: { group_id: string; invite_id: string; reason?: string }) {
    return post<void>(GROUPS.MEMBERS.INVITES_REVOKE, data);
  },

  join(data: { group_id: string; password?: string; message?: string }) {
    return post<JoinGroupResult>(GROUPS.ME.JOIN, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  setRole(data: { group_id: string; user_id: string; role: SettableGroupRole }) {
    return post<void>(GROUPS.MEMBERS.SET_ROLE, data);
  },

  mute(data: { group_id: string; user_id: string; duration_minutes: number }) {
    return post<void>(GROUPS.MEMBERS.MUTE, data);
  },

  unmute(data: { group_id: string; user_id: string }) {
    return post<void>(GROUPS.MEMBERS.UNMUTE, data);
  },

  ban(data: { group_id: string; user_id: string; reason?: string }) {
    return post<void>(GROUPS.MEMBERS.BAN, data, { idempotencyKey: generateIdempotencyKey() });
  },

  unban(data: { group_id: string; user_id: string }) {
    return post<void>(GROUPS.MEMBERS.UNBAN, data);
  },

  getBanned(params: { group_id: string; pagination?: PaginationParams }) {
    return paginate<BannedMemberItem>(GROUPS.MEMBERS.LIST_BANNED, params);
  },

  removeMember(data: { group_id: string; user_id: string }) {
    return post<void>(GROUPS.MEMBERS.REMOVE_MEMBER, data);
  },

  getApplications(params: {
    group_id: string;
    status?: ApplicationStatus | 'all';
    pagination?: PaginationParams;
  }) {
    return paginate<GroupApplicationItem>(GROUPS.APPLICATIONS.LIST, params);
  },

  approveApplication(data: { application_id: string }) {
    return post<void>(GROUPS.APPLICATIONS.APPROVE, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  rejectApplication(data: { application_id: string; reason?: string }) {
    return post<void>(GROUPS.APPLICATIONS.REJECT, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  getAnnouncements(params: { group_id: string }) {
    return post<{ items: GroupAnnouncementItem[] }>(GROUPS.ANNOUNCEMENTS.LIST, params);
  },

  createAnnouncement(data: {
    group_id: string;
    title: string;
    content: string;
    is_pinned?: boolean;
  }) {
    return post<GroupAnnouncementItem>(GROUPS.ANNOUNCEMENTS.CREATE, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  updateAnnouncement(data: {
    group_id: string;
    announcement_id: string;
    title?: string;
    content?: string;
    is_pinned?: boolean;
  }) {
    return post<void>(GROUPS.ANNOUNCEMENTS.UPDATE, data);
  },

  deleteAnnouncement(data: { group_id: string; announcement_id: string }) {
    return post<void>(GROUPS.ANNOUNCEMENTS.DELETE, data);
  },

  getMyGroups(params?: { pagination?: PaginationParams }) {
    return paginate<GroupItem>(GROUPS.ME.LIST, params);
  },

  getMyApplications(params?: {
    status?: ApplicationStatus | 'all';
    pagination?: PaginationParams;
  }) {
    return paginate<GroupApplicationItem>(GROUPS.ME.APPLICATIONS_LIST, params);
  },

  cancelApplication(data: { application_id: string }) {
    return post<void>(GROUPS.ME.CANCEL_APPLICATION, data);
  },

  getInvites(params?: { status?: ApplicationStatus | 'all'; pagination?: PaginationParams }) {
    return paginate<GroupInviteItem>(GROUPS.ME.INVITES_LIST, params);
  },

  acceptInvite(data: { invite_id: string }) {
    return post<void>(GROUPS.ME.ACCEPT_INVITE, data, { idempotencyKey: generateIdempotencyKey() });
  },

  rejectInvite(data: { invite_id: string }) {
    return post<void>(GROUPS.ME.REJECT_INVITE, data, { idempotencyKey: generateIdempotencyKey() });
  },

  leave(data: { group_id: string }) {
    return post<void>(GROUPS.ME.LEAVE, data, { idempotencyKey: generateIdempotencyKey() });
  }
};

export const groupMessagesService = {
  send(data: {
    group_id: string;
    type: SendMessageType;
    content?: string;
    file_id?: string;
    share_code?: string;
    reply_to_message_id?: string;
    mentions?: string[];
  }) {
    return post<MessageSent>(GROUP_MESSAGES.SEND, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  getHistory(params: { group_id: string; pagination?: PaginationParams }) {
    return paginate<GroupMessageItem>(GROUP_MESSAGES.GET_HISTORY, params);
  },

  update(data: { message_id: string; content: string }) {
    return post<void>(GROUP_MESSAGES.UPDATE, data);
  },

  recall(data: { message_id: string }) {
    return post<void>(GROUP_MESSAGES.RECALL, data);
  },

  markAsRead(data: { group_id: string; up_to_message_id?: string }) {
    return post<void>(GROUP_MESSAGES.ME.MARK_AS_READ, data);
  },

  markAllRead() {
    return post<void>(GROUP_MESSAGES.ME.MARK_ALL_READ, {});
  }
};
