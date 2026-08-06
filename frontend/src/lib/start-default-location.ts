// 默认位置加载 helper —— 收敛「$effect + cancelled + loadDefaultLocation」样板。
// 使用方在 $effect 中调用，回调里写入页面自己的 $state：
//
//   $effect(() => startDefaultLocation((loc) => {
//     referencePoint = loc?.point ?? null;
//     locationName = loc?.location_name ?? '';
//   }));
//
// 返回 cleanup 供 $effect 的 return 使用（卸载后不再落地），与原有 cancelled 模式等价。
import { loadDefaultLocation, type DefaultLocation } from '@/lib/default-location';
export type { DefaultLocation } from '@/lib/default-location';

export function startDefaultLocation(onLoad: (loc: DefaultLocation | null) => void): () => void {
	let cancelled = false;
	void loadDefaultLocation().then((loc) => {
		if (!cancelled) onLoad(loc);
	});
	return () => {
		cancelled = true;
	};
}
