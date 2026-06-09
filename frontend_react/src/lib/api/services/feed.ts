import { post, paginate } from "../client";
import { FEED } from "../endpoints";
import type {
  MomentItem,
  GroupItem,
  UserItem,
  TrendingBundle,
  FeedSource,
  DismissTargetType,
  GeoPoint,
  PaginationParams,
} from "@/lib/types/api";

export const feedService = {
  getMoments(params: {
    reference_point: GeoPoint;
    source: FeedSource;
    radius?: number;
    pagination?: PaginationParams;
  }) {
    return paginate<MomentItem>(FEED.LIST_MOMENTS, params);
  },

  getGroups(params: {
    reference_point: GeoPoint;
    radius?: number;
    pagination?: PaginationParams;
  }) {
    return paginate<GroupItem>(FEED.LIST_GROUPS, params);
  },

  getUsers(params: {
    reference_point: GeoPoint;
    radius?: number;
    pagination?: PaginationParams;
  }) {
    return paginate<UserItem>(FEED.LIST_USERS, params);
  },

  getTrending() {
    return post<TrendingBundle>(FEED.LIST_TRENDING, {});
  },

  dismiss(data: { target_type: DismissTargetType; target_id: string }) {
    return post<void>(FEED.ME.DISMISS, data);
  },
};
