// 冷启动 bootstrap 单次请求缓存。
//
// spec api_spec_system.md → bootstrap：这是冷启动聚合端点，一次请求同时下发
// meta / client_config / profile / unread_summary，用于替代分别请求
// system/meta + system/client_config + users/account + notify/unread_summary。
//
// 前端此前只消费了 unread_summary，默认位置（lib/default-location）又单独
// 请求 /public/v1/system/client_config——首屏因此重复拉同一份公共配置。
// 这里把 bootstrap 收敛为进程内单次请求（含 in-flight 去重），auth 与
// 默认位置共用同一份响应；登录/升级等需要刷新的场景走 refreshBootstrap()。

import { bootstrap as apiBootstrap } from '@/api';
import type { BootstrapData } from '@/api/types';

let cached: BootstrapData | null = null;
let inflight: Promise<BootstrapData | null> | null = null;

function fetchBootstrap(): Promise<BootstrapData | null> {
	if (inflight) return inflight;
	inflight = apiBootstrap()
		.then((res) => {
			const data = res.code === 0 && res.data ? res.data : null;
			if (data) cached = data;
			return data;
		})
		.catch(() => null)
		.finally(() => {
			inflight = null;
		});
	return inflight;
}

/** 取冷启动 bootstrap 数据（首次发请求，之后复用缓存 / 在途请求）。失败返回 null，不抛错。 */
export function loadBootstrap(): Promise<BootstrapData | null> {
	if (cached) return Promise.resolve(cached);
	return fetchBootstrap();
}

/**
 * 强制重新拉取（登录/注册/访客成功后 unread_summary 随账号变化）。
 * 先等掉在途的旧请求，避免旧响应回填缓存。失败返回 null，不抛错。
 */
export async function refreshBootstrap(): Promise<BootstrapData | null> {
	cached = null;
	if (inflight) await inflight.catch(() => null);
	return fetchBootstrap();
}

/** 测试辅助：清空缓存（模块级单例，测试间隔离用）。 */
export function resetBootstrapCache(): void {
	cached = null;
	inflight = null;
}
