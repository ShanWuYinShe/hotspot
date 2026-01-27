import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import {
  CreateMomentRequest,
  EditMomentRequest,
  FavoritedStatus,
  LikedStatus,
  MomentCard,
  MomentLikeItem,
  PaginatedData,
  UserMomentListItem,
} from '@/types/api';

export const MomentService = {
  create: (data: CreateMomentRequest) =>
    apiClient.post<UserMomentListItem>(ENDPOINTS.MOMENTS.CREATE, data),

  update: (data: EditMomentRequest) =>
    apiClient.post(ENDPOINTS.MOMENTS.UPDATE, data),

  getDetail: (momentId: string) =>
    apiClient.post<MomentCard>(ENDPOINTS.MOMENTS.OPEN_THE_DOOR, { moment_id: momentId }),

  interactionState: (momentId: string) =>
    apiClient.post(ENDPOINTS.MOMENTS.INTERACTION_STATE, { moment_id: momentId }),

  listMyMoments: (params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<UserMomentListItem>>(ENDPOINTS.MOMENTS.ME, params ?? {}),

  listUserMoments: (userId: number, params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<UserMomentListItem>>(ENDPOINTS.MOMENTS.USER_MOMENTS, { user_id: userId, ...params }),

  listMyFavorites: (params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<MomentCard>>(ENDPOINTS.MOMENTS.ME_FAVORITES, params ?? {}),

  checkLiked: (momentId: string) =>
    apiClient.post<LikedStatus>(ENDPOINTS.MOMENTS.ME_LIKED, { moment_id: momentId }),

  checkFavorited: (momentId: string) =>
    apiClient.post<FavoritedStatus>(ENDPOINTS.MOMENTS.ME_FAVORITED, { moment_id: momentId }),

  likes: (momentId: string, params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<MomentLikeItem>>(ENDPOINTS.MOMENTS.LIKES, { moment_id: momentId, ...params }),

  like: (momentId: string) =>
    apiClient.post(ENDPOINTS.MOMENTS.LIKE, { moment_id: momentId }),

  unlike: (momentId: string) =>
    apiClient.post(ENDPOINTS.MOMENTS.UNLIKE, { moment_id: momentId }),

  favorite: (momentId: string) =>
    apiClient.post(ENDPOINTS.MOMENTS.FAVORITE, { moment_id: momentId }),

  unfavorite: (momentId: string) =>
    apiClient.post(ENDPOINTS.MOMENTS.UNFAVORITE, { moment_id: momentId }),

  delete: (momentId: string) =>
    apiClient.post(ENDPOINTS.MOMENTS.DELETE, { moment_id: momentId }),
};
