import { post, paginate, generateIdempotencyKey, type RequestOptions } from '../client';
import { MOMENTS } from '../endpoints';
import type {
  MomentItem,
  LikeItem,
  Location,
  PrivacyLevel,
  PaginationParams,
  GeoPoint
} from '@/lib/types/api';

export const momentsService = {
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
    return paginate<MomentItem>(MOMENTS.SEARCH, params, options);
  },

  getMoment(data: { moment_id: string }) {
    return post<MomentItem>(MOMENTS.OPEN_THE_DOOR, data);
  },

  getLikes(params: { moment_id: string; pagination?: PaginationParams }) {
    return paginate<LikeItem>(MOMENTS.LIST_LIKES, params);
  },

  create(data: {
    description: string;
    location: Location;
    tags?: string[];
    image_file_ids?: string[];
    privacy_level?: PrivacyLevel;
    mentions?: string[];
  }) {
    return post<MomentItem>(MOMENTS.CREATE, data, { idempotencyKey: generateIdempotencyKey() });
  },

  update(data: {
    moment_id: string;
    description?: string;
    location?: Location;
    tags?: string[];
    image_file_ids?: string[];
    privacy_level?: PrivacyLevel;
    mentions?: string[];
  }) {
    return post<MomentItem>(MOMENTS.UPDATE, data);
  },

  delete(data: { moment_id: string }) {
    return post<void>(MOMENTS.DELETE, data);
  },

  like(data: { moment_id: string }) {
    return post<void>(MOMENTS.ME.LIKE, data);
  },

  unlike(data: { moment_id: string }) {
    return post<void>(MOMENTS.ME.UNLIKE, data);
  },

  favorite(data: { moment_id: string }) {
    return post<void>(MOMENTS.ME.FAVORITE, data);
  },

  unfavorite(data: { moment_id: string }) {
    return post<void>(MOMENTS.ME.UNFAVORITE, data);
  },

  getMyMoments(params?: { pagination?: PaginationParams }) {
    return paginate<MomentItem>(MOMENTS.ME.LIST, params);
  },

  getMyFavorites(params?: { pagination?: PaginationParams }) {
    return paginate<MomentItem>(MOMENTS.ME.LIST_FAVORITES, params);
  }
};
