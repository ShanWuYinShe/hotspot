/**
 * 分享服务
 */

import { post, paginate, generateIdempotencyKey } from '../client';
import { SHARE } from '../endpoints';
import type {
  ApiResponse,
  CreateShareRequest,
  GetShareRequest,
  PaginatedData,
  ShareDetail,
  ShareCreated,
  ShareListItem,
  ShareListRequest
} from '$lib/types/api';

export const shareService = {
  /**
   * 创建分享
   */
  create(request: CreateShareRequest): Promise<ApiResponse<ShareCreated>> {
    return post<ShareCreated>(SHARE.CREATE, request, { idempotencyKey: generateIdempotencyKey() });
  },

  /**
   * 获取分享详情
   */
  get(request: GetShareRequest): Promise<ApiResponse<ShareDetail>> {
    return post<ShareDetail>(SHARE.GET, request);
  },

  /**
   * 我的分享列表
   */
  listMyShares(params?: ShareListRequest): Promise<ApiResponse<PaginatedData<ShareListItem>>> {
    return paginate<ShareListItem>(SHARE.ME_LIST, params);
  },

  /**
   * 撤销分享
   */
  revoke(data: { share_id: string }) {
    return post<void>(SHARE.REVOKE, data);
  }
};
