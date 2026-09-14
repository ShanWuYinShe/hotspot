/**
 * 乐观关注状态共享模块（关注/粉丝分区与首页「附近的人」三处共用）。
 *
 * 三处原先各持一份 overrides + pendingIds + try/catch 回滚的复制实现，且语义有
 * 漂移：分区版检查响应 code!==0 并回滚，首页版只捕获网络异常（HTTP 200 但
 * code!==0 被当成功）。本模块取最完整语义：code!==0 视为失败并回滚。
 *
 * 状态模型：服务端列表数据是基准（base），本地 overrides 承载乐观切换——
 * isFollowed(id, base) = overrides[id] ?? base；toggle 乐观翻转后调用
 * users/set_follow，失败回滚并 toast，结束后清除 pending（按钮防重入）。
 * 成功侧 toast 文案三处各不相同，由调用方按返回的 followed 自行决定。
 */
import { follow } from '@/api';
import { showToast } from '@/stores/toast.svelte';

export interface OptimisticFollow {
	/** 该 id 当前关注态：本地 override 优先，否则回落调用方给定的服务端基准。 */
	isFollowed: (id: string, base: boolean) => boolean;
	/** 该 id 是否有进行中的切换请求（按钮 disabled 防重入）。 */
	isPending: (id: string) => boolean;
	/**
	 * 乐观翻转关注态并请求 users/set_follow（active = 目标态）。
	 * 返回 ok=false 表示请求失败或被防重入拦截；成功时 followed 为切换后的关注态。
	 */
	toggle: (id: string, base: boolean) => Promise<{ ok: boolean; followed: boolean }>;
}

export function createOptimisticFollow(): OptimisticFollow {
	let overrides = $state<Record<string, boolean>>({});
	let pendingIds = $state<Set<string>>(new Set());

	const isFollowed = (id: string, base: boolean) => overrides[id] ?? base;
	const isPending = (id: string) => pendingIds.has(id);

	const toggle = async (id: string, base: boolean) => {
		if (pendingIds.has(id)) return { ok: false, followed: base };
		const prev = isFollowed(id, base);
		const next = !prev;
		overrides = { ...overrides, [id]: next };
		pendingIds = new Set(pendingIds).add(id);
		try {
			// spec: users/set_follow——active:true 关注，active:false 取消关注
			const res = await follow({ user_id: id, active: next });
			if (res.code !== 0) throw new Error(res.message ?? '操作失败');
			return { ok: true, followed: next };
		} catch {
			overrides = { ...overrides, [id]: prev };
			showToast('操作失败，请重试', 'danger');
			return { ok: false, followed: prev };
		} finally {
			const rest = new Set(pendingIds);
			rest.delete(id);
			pendingIds = rest;
		}
	};

	return { isFollowed, isPending, toggle };
}
