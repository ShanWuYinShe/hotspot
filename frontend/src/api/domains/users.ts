// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, pick } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	AccountData,
	RelatedUserItem,
	MomentItem,
	PaginatedData,
	UserProfileBundle,
	UserSettings,
	UpdateSettingsRequest,
	DeleteAccountRequest,
	UpgradeAccountRequest,
	FetchBlockedRequest,
	BlockedUserItem,
	UpdateAccountRequest,
	UpdateLocationRequest,
	SetFollowRequest,
	SetBlockRequest,
	PaginationParams,
} from '@/api/types';

export async function getAccount()                        { return request<AccountData>(Endpoints.Users.GetAccount); }
export async function updateAccount(params: UpdateAccountRequest)       { return request<void>(Endpoints.Users.UpdateAccount, params); }
export async function getSettings()                       { return request<UserSettings>(Endpoints.Users.GetSettings); }
// spec: api_spec_user.md → users/update_settings（request 为 { settings: Partial<UserSettings> }）
export async function updateSettings(params: UpdateSettingsRequest) {
	return request<void>(Endpoints.Users.UpdateSettings, params);
}
// spec: api_spec_user.md → users/update_location（真实位置上报：用于被他人发现/展示大致位置）。
// 不在 MUST 幂等清单，不加 Idempotency-Key。
export async function updateLocation(params: UpdateLocationRequest) {
	return request<void>(Endpoints.Users.UpdateLocation, params);
}
export async function follow(params: SetFollowRequest)              { return request<void>(Endpoints.Users.Follow, params); }
export async function block(params: SetBlockRequest)               { return request<void>(Endpoints.Users.Block, params); }
// 解除拉黑 = set_block 两态统一端点的 active=false 侧（spec: api_spec_user.md → users/set_block）
export async function unblock(params: Omit<SetBlockRequest, 'active'>) { return block({ ...params, active: false }); }
// spec: api_spec_user.md → users/fetch_blocked（我的拉黑列表，blocked_at 倒序）
export async function fetchBlocked(params?: FetchBlockedRequest)   { return request<PaginatedData<BlockedUserItem>>(Endpoints.Users.ListBlocked, params ?? { pagination: null }); }
// spec: api_spec_user.md → users/upgrade（访客绑定登录凭证升级为正式账号；返回升级后完整账户数据）
export async function upgradeAccount(params: UpgradeAccountRequest) { return request<AccountData>(Endpoints.Users.Upgrade, params); }
export async function deleteAccount(params: DeleteAccountRequest) { return request<void>(Endpoints.Users.DeleteAccount, params); }
// spec: api_spec_user.md → users/open_the_door（include_recent_moments=true 时返回主页首屏聚合）
export async function openTheDoor(userId: string, includeRecentMoments = false) {
	return request<UserProfileBundle>(pick(Endpoints.Users.GetDetail, Endpoints.Public.Users.GetDetail), { user_id: userId, include_recent_moments: includeRecentMoments });
}
// spec: api_spec_user.md → users/set_remark（需已关注对方；remark 传空串清除备注）
export async function setRemark(params: { user_id: string; remark: string }) { return request<void>(Endpoints.Users.SetRemark, params); }
export async function listFriends(params: { user_id: string; pagination?: PaginationParams })        { return request<PaginatedData<RelatedUserItem>>(pick(Endpoints.Users.ListFriends, Endpoints.Public.Users.ListFriends), params); }
export async function listFollowing(params: { user_id: string; pagination?: PaginationParams })      { return request<PaginatedData<RelatedUserItem>>(pick(Endpoints.Users.ListFollowing, Endpoints.Public.Users.ListFollowing), params); }
export async function listFollowers(params: { user_id: string; pagination?: PaginationParams })      { return request<PaginatedData<RelatedUserItem>>(pick(Endpoints.Users.ListFollowers, Endpoints.Public.Users.ListFollowers), params); }

// Users extended
export async function listUserMoments(userId: string, pagination?: PaginationParams) {
  return request<PaginatedData<MomentItem>>(pick(Endpoints.Users.ListMoments, Endpoints.Public.Users.ListMoments), { user_id: userId, pagination });
}
export async function listUserReactedMoments(userId: string, pagination?: PaginationParams) {
  return request<PaginatedData<MomentItem>>(pick(Endpoints.Users.ListReactedMoments, Endpoints.Public.Users.ListReactedMoments), { user_id: userId, pagination });
}
