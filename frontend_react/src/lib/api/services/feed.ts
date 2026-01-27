import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import {
  FeedGroupItem,
  FeedMomentItem,
  FeedUserItem,
  GetFeedMomentsRequest,
  PaginatedData,
  TrendingBundle,
} from '@/types/api';

export const FeedService = {
  getMoments: (params: GetFeedMomentsRequest) =>
    apiClient.post<PaginatedData<FeedMomentItem>>(ENDPOINTS.FEED.MOMENTS, params),

  getGroups: (params: { reference_point: { latitude: number; longitude: number }; radius?: number; cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<FeedGroupItem>>(ENDPOINTS.FEED.GROUPS, params),

  getUsers: (params: { reference_point: { latitude: number; longitude: number }; radius?: number; cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<FeedUserItem>>(ENDPOINTS.FEED.USERS, params),

  getTrending: (params?: { reference_point?: { latitude: number; longitude: number } }) =>
    apiClient.post<TrendingBundle>(ENDPOINTS.FEED.TRENDING, params ?? {}),

  feedback: (data: { target_type: 'moment' | 'group' | 'user'; target_id: string; action: 'hide' | 'not_interested' }) =>
    apiClient.post(ENDPOINTS.FEED.FEEDBACK, data),
};
