import { post } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
	ApiResponse,
	ApplyShareRequest,
	CreateShareRequest,
	GetShareRequest,
	Share
} from '$lib/types/api';

/**
 * 创建分享
 */
export async function createShare(
	request: CreateShareRequest
): Promise<ApiResponse<Share>> {
	return post<Share>(ENDPOINTS.SHARE.CREATE, request);
}

/**
 * 获取分享详情
 */
export async function getShare(
	request: GetShareRequest
): Promise<ApiResponse<Share>> {
	return post<Share>(ENDPOINTS.SHARE.GET, request);
}

/**
 * 应用分享
 */
export async function applyShare(
	request: ApplyShareRequest
): Promise<ApiResponse<void>> {
	return post<void>(ENDPOINTS.SHARE.APPLY, request);
}
