// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, newIdempotencyKey, pick, type RequestOptions } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	MomentItem,
	OwnerMomentItem,
	FavoritedMomentItem,
	PaginatedData,
	CreateMomentRequest,
	UpdateMomentRequest,
	SearchMomentsRequest,
	FavoriteMomentRequest,
	FetchFollowingMomentsRequest,
	FetchNearbyMomentsRequest,
	PaginationParams,
} from '@/api/types';

// Moments
export async function searchMoments(params: SearchMomentsRequest, opts?: RequestOptions) {
	return request<PaginatedData<MomentItem>>(pick(Endpoints.Moments.Search, Endpoints.Public.Moments.Search), params, {}, 0, undefined, opts);
}
export async function getMomentDetail(momentId: string)   { return request<MomentItem>(pick(Endpoints.Moments.GetDetail, Endpoints.Public.Moments.GetDetail), { moment_id: momentId }); }
export async function createMoment(params: CreateMomentRequest)        { return request<OwnerMomentItem>(Endpoints.Moments.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function updateMoment(params: UpdateMomentRequest)        { return request<OwnerMomentItem>(Endpoints.Moments.Update, params); }
export async function deleteMoment(params: { moment_id: string }) { return request<void>(Endpoints.Moments.Delete, params); }
// spec: api_spec_moment.md → moments/fetch_created（拥有者视角，含 private 与软删占位）
export async function listMyMoments(params?: { pagination?: PaginationParams })      { return request<PaginatedData<OwnerMomentItem>>(Endpoints.Moments.ListCreated, params ?? {}); }
export async function getFavoriteMoments(params?: { pagination?: PaginationParams })   { return request<PaginatedData<FavoritedMomentItem>>(Endpoints.Moments.ListFavorites, params ?? {}); }
// spec: api_spec_moment.md → moments/fetch_following（关注动态；依赖查看者关注关系，无匿名版）
export async function fetchFollowingMoments(params?: FetchFollowingMomentsRequest)  { return request<PaginatedData<MomentItem>>(Endpoints.Moments.FetchFollowing, params ?? {}); }
// spec: api_spec_moment.md → moments/fetch_nearby（附近瞬间，原 feed/nearby_moments 迁移；双版本）
export async function fetchNearbyMoments(params?: FetchNearbyMomentsRequest, opts?: RequestOptions) {
	return request<PaginatedData<MomentItem>>(pick(Endpoints.Moments.FetchNearby, Endpoints.Public.Moments.FetchNearby), params ?? {}, {}, 0, undefined, opts);
}

// Moments — favorite (two-state via active per spec)
export async function favoriteMoment(params: FavoriteMomentRequest)      { return request<void>(Endpoints.Moments.Favorite, params); }
