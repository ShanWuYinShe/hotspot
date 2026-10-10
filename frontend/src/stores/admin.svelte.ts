// Admin store — 平台管理员权限状态（rbac/me 结果缓存）。
// 供桌面侧栏条件渲染「管理后台」入口与 /admin 路由守卫共用，避免各处重复请求。
// AdminContext（runes 模块单例）。非管理员（10200 not_admin）与未登录均收敛为
// isAdmin=false；ensureLoaded 在登录态变化时可安全重入（in-flight 去重）。

import { adminRbacMe } from '@/api';
import type { AdminPermissions } from '@/api/types';

let permissions = $state<string[]>([]);
let roles = $state<{ role_id: string; name: string }[]>([]);
let isAdmin = $state(false);
let loaded = $state(false);
let inflight: Promise<void> | null = null;

async function fetchMe(): Promise<void> {
	try {
		const res = await adminRbacMe();
		if (res.code === 0 && res.data) {
			permissions = res.data.permissions ?? [];
			roles = res.data.roles ?? [];
			isAdmin = true;
		} else {
			// 10200 not_admin / 10100 未登录等：统一视为非管理员
			permissions = [];
			roles = [];
			isAdmin = false;
		}
	} catch {
		// 传输失败（网络抖动/后端滚动）不等同「非管理员」：不置 loaded=true，
		// 保留下次 ensureAdminLoaded 的重查机会，避免管理入口被一次瞬时故障
		// 永久锁死（与 auth store 区分 transport-error/invalid 的纪律一致）。
		isAdmin = false;
		return;
	}
	loaded = true;
}

/** 登录后调用一次；重复调用在 in-flight 或已成功加载时直接复用。 */
export function ensureAdminLoaded(): void {
	if (loaded || inflight) return;
	inflight = fetchMe().finally(() => {
		inflight = null;
	});
}

/** 登出时清空（auth.logout 处调用）。 */
export function resetAdmin(): void {
	permissions = [];
	roles = [];
	isAdmin = false;
	loaded = false;
}

export function hasAdminPermission(key: string): boolean {
	return isAdmin && (permissions.includes('*') || permissions.includes(key));
}

/** '*'（superadmin）判定；'*' 不隐含 PII 键（read_ip），PII 一律用 hasAdminPermission 显式判。 */
export function isSuperAdmin(): boolean {
	return isAdmin && permissions.includes('*');
}

export const admin = {
	get isAdmin() { return isAdmin; },
	get loaded() { return loaded; },
	get permissions(): readonly string[] { return permissions; },
	get roles(): readonly { role_id: string; name: string }[] { return roles; }
};

export type { AdminPermissions };
