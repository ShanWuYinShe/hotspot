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
} from '@/lib/types/api';

export const shareService = {
  create(request: CreateShareRequest): Promise<ApiResponse<ShareCreated>> {
    return post<ShareCreated>(SHARE.CREATE, request, { idempotencyKey: generateIdempotencyKey() });
  },

  get(request: GetShareRequest): Promise<ApiResponse<ShareDetail>> {
    return post<ShareDetail>(SHARE.GET, request);
  },

  listMyShares(params?: ShareListRequest): Promise<ApiResponse<PaginatedData<ShareListItem>>> {
    return paginate<ShareListItem>(SHARE.ME_LIST, params);
  },

  revoke(data: { share_id: string }) {
    return post<void>(SHARE.REVOKE, data);
  }
};
