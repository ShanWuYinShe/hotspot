// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, newIdempotencyKey } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	CreateShareRequest,
	FetchCreatedSharesRequest,
	PaginatedData,
	RevokeShareRequest,
	ShareCreated,
	ShareDetail,
	ShareListItem,
} from '@/api/types';

// 分享码落地（spec api_spec_public.md share/open_the_door；仅 public 版，匿名可调）
export async function openShareDoor(params: { share_code: string }) { return request<ShareDetail>(Endpoints.Share.Get, params); }
// 分享码创建（spec api_spec_share.md share/create；契约幂等端点：必须携带 Idempotency-Key）
export async function createShare(params: CreateShareRequest) { return request<ShareCreated>(Endpoints.Share.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
// 我的分享列表（spec api_spec_share.md share/fetch_created；登录端点，按 created_at DESC）
export async function fetchCreatedShares(params: FetchCreatedSharesRequest) { return request<PaginatedData<ShareListItem>>(Endpoints.Share.List, params); }
// 撤销分享（spec api_spec_share.md share/revoke；share_id 与 share_code 二选一，重复撤销 no-op）
export async function revokeShare(params: RevokeShareRequest) { return request<null>(Endpoints.Share.Revoke, params); }
