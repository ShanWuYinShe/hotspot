/**
 * 瞬间服务
 * 瞬间相关 API
 */

import { apiRequest, apiRequestPublic, apiRequestIdempotent } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type {
  ApiResponse,
  PaginatedData,
  MomentCard,
  MyMomentState,
  UserMomentListItem,
  Location,
  MomentImage,
} from '@/types/api';

interface OpenTheDoorParams {
  moment_id: string;
}

export async function getMoment(
  momentId: string
): Promise<ApiResponse<MomentCard>> {
  return apiRequestPublic<MomentCard>(ENDPOINTS.MOMENTS.OPEN_THE_DOOR, {
    moment_id: momentId,
  } as OpenTheDoorParams);
}

export async function getMomentState(
  momentId: string
): Promise<ApiResponse<MyMomentState>> {
  return apiRequest<MyMomentState>(ENDPOINTS.MOMENTS.INTERACTION_STATE, {
    moment_id: momentId,
  });
}

interface MyMomentsParams {
  limit?: number;
  cursor?: string;
}

export async function getMyMoments(
  params: MyMomentsParams = {}
): Promise<ApiResponse<PaginatedData<UserMomentListItem>>> {
  return apiRequest<PaginatedData<UserMomentListItem>>(
    ENDPOINTS.MOMENTS.ME,
    params
  );
}

export async function getMyFavorites(
  params: MyMomentsParams = {}
): Promise<ApiResponse<PaginatedData<UserMomentListItem>>> {
  return apiRequest<PaginatedData<UserMomentListItem>>(
    ENDPOINTS.MOMENTS.ME_FAVORITES,
    params
  );
}

export async function getMyLikedMoments(
  params: MyMomentsParams = {}
): Promise<ApiResponse<PaginatedData<UserMomentListItem>>> {
  return apiRequest<PaginatedData<UserMomentListItem>>(
    ENDPOINTS.MOMENTS.ME_LIKED,
    params
  );
}

interface CreateMomentParams {
  description: string;
  images?: MomentImage[];
  tags?: string[];
  mentions?: number[];
  location: Location;
}

export async function createMoment(
  params: CreateMomentParams
): Promise<ApiResponse<{ moment_id: string }>> {
  return apiRequestIdempotent<{ moment_id: string }>(
    ENDPOINTS.MOMENTS.CREATE,
    params
  );
}

interface UpdateMomentParams {
  moment_id: string;
  description?: string;
  tags?: string[];
  mentions?: number[];
}

export async function updateMoment(
  params: UpdateMomentParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.MOMENTS.UPDATE, params);
}

export async function deleteMoment(
  momentId: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.MOMENTS.DELETE, { moment_id: momentId });
}

export async function likeMoment(
  momentId: string
): Promise<ApiResponse<void>> {
  return apiRequestIdempotent<void>(ENDPOINTS.MOMENTS.LIKE, {
    moment_id: momentId,
  });
}

export async function unlikeMoment(
  momentId: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.MOMENTS.UNLIKE, { moment_id: momentId });
}

export async function favoriteMoment(
  momentId: string
): Promise<ApiResponse<void>> {
  return apiRequestIdempotent<void>(ENDPOINTS.MOMENTS.FAVORITE, {
    moment_id: momentId,
  });
}

export async function unfavoriteMoment(
  momentId: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.MOMENTS.UNFAVORITE, {
    moment_id: momentId,
  });
}
