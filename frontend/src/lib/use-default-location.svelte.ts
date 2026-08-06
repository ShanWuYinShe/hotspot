// 默认位置 rune helper —— 收敛「$effect + cancelled + loadDefaultLocation」样板。
// 使用方只需声明两个 $state 并在 $effect 中调用：
//
//   let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);
//   let locationName = $state('');
//   $effect(() => useDefaultLocation(referencePoint, locationName));
//
// helper 内部处理取消清理（组件卸载后不再落地），返回清理函数。
import { loadDefaultLocation } from '@/lib/default-location';

/**
 * 在 $effect 内调用：加载默认位置并写入传入的 state 引用。
 * 返回 cleanup 供 $effect 的 return 使用（卸载时取消落地）。
 */
export function useDefaultLocation(
	referencePoint: { value: { latitude: number; longitude: number } | null },
	locationName: { value: string }
): () => void {
	let cancelled = false;
	void loadDefaultLocation().then((loc) => {
		if (!cancelled) {
			referencePoint.value = loc?.point ?? null;
			locationName.value = loc?.location_name ?? '';
		}
	});
	return () => {
		cancelled = true;
	};
}
