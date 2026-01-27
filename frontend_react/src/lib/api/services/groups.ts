import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import {
  GroupCard,
  GroupSettingsResponse,
  GroupAnnouncementItem,
  GroupMemberItem,
  MyGroupItem,
  PaginatedData,
  PaginationParams,
} from '@/types/api';

export const GroupService = {
  myGroups: (params?: PaginationParams) =>
    apiClient.post<PaginatedData<MyGroupItem>>(ENDPOINTS.GROUPS.MY_GROUPS, params ?? {}),

  openTheDoor: (groupId: string) =>
    apiClient.post<GroupCard>(ENDPOINTS.GROUPS.OPEN_THE_DOOR, { group_chat_id: groupId }),

  settings: (groupId: string) =>
    apiClient.post<GroupSettingsResponse>(ENDPOINTS.GROUPS.SETTINGS, { group_chat_id: groupId }),

  announcements: (groupId: string, params?: PaginationParams) =>
    apiClient.post<PaginatedData<GroupAnnouncementItem>>(ENDPOINTS.GROUPS.ANNOUNCEMENTS, {
      group_chat_id: groupId,
      ...params,
    }),

  apply: (data: { group_chat_id: string; message?: string }) =>
    apiClient.post(ENDPOINTS.GROUPS.APPLY, data),

  join: (data: { group_chat_id: string; password?: string }) =>
    apiClient.post(ENDPOINTS.GROUPS.JOIN, data),

  leave: (data: { group_chat_id: string }) =>
    apiClient.post(ENDPOINTS.GROUPS.LEAVE, data),

  create: (data: unknown) =>
    apiClient.post<GroupCard>(ENDPOINTS.GROUPS.CREATE, data),

  members: (groupId: string, params?: PaginationParams) =>
    apiClient.post<PaginatedData<GroupMemberItem>>(ENDPOINTS.GROUPS.MEMBERS, {
      group_chat_id: groupId,
      ...params,
    }),

  update: (data: unknown) =>
    apiClient.post<GroupCard>(ENDPOINTS.GROUPS.UPDATE, data),
};
