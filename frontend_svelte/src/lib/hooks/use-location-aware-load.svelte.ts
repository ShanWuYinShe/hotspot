/**
 * 监听 mapAnchorStore.anchor 变化的复用 hook
 *
 * 首次立即加载，后续变化防抖刷新。自动清理定时器。
 *
 * 使用示例：
 * ```typescript
 * useLocationAwareLoad((refresh) => loadMoments(refresh));
 * ```
 */

import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';

/**
 * @param loadFn 接受 refresh 参数的异步加载函数
 * @param options.debounceMs 防抖时间（默认 500ms）
 */
export function useLocationAwareLoad(
	loadFn: (refresh: boolean) => Promise<void>,
	options?: { debounceMs?: number }
) {
	const debounceMs = options?.debounceMs ?? 500;
	// 非响应式：仅用于 effect 内部判断，不能是 $state，否则写入会导致 effect 重跑
	let prevUpdatedAt: number | undefined;
	let timer: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		const now = mapAnchorStore.anchor.updatedAt;

		if (prevUpdatedAt === undefined) {
			// 首次：记录时间戳并立即加载
			prevUpdatedAt = now;
			void loadFn(false);
			return;
		}

		// 时间戳未变化时跳过（如依赖重算但值未变）
		if (prevUpdatedAt === now) return;

		// 后续：防抖刷新
		prevUpdatedAt = now;
		if (timer) clearTimeout(timer);
		timer = setTimeout(() => {
			void loadFn(true);
		}, debounceMs);

		return () => {
			if (timer) {
				clearTimeout(timer);
				timer = null;
			}
			// 重置，以便下次 mount 时走首次加载路径（prevUpdatedAt === undefined）
			prevUpdatedAt = undefined;
		};
	});
}
