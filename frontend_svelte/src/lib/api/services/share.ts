/**
 * 分享服务
 */

import { post, generateIdempotencyKey } from '../client';
import { SHARE } from '../endpoints';
import type {
	ApiResponse,
	CreateShareRequest,
	GetShareRequest,
	ShareDetail,
	ShareCreated
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

	// getMyShares deferred: share/me/list

	/**
	 * 撤销分享
	 */
	revoke(data: { share_id: string }) {
		return post<void>(SHARE.REVOKE, data);
	}
};
