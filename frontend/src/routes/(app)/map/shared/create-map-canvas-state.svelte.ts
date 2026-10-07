/**
 * map/canvas 数据状态工厂（桌面/移动共用）：图钉/热力桶/分类计数 + 加载/错误/重试。
 * 视口中心、稳定边界或筛选变化时重新拉取；依赖 stableBounds（手势/缩放结束才回写），
 * 拖拽中的逐帧 bounds 变化只驱动图钉定位、不触发请求。
 * 须在组件初始化期调用（内部使用 $effect，与 createLocateAction 同款工厂先例）。
 */
import {
	fetchMapCanvas, fetchMapCanvasByViewport,
	type MapBounds, type MapHeatBucket
} from './location-shared';
import type { Pin, PinType } from './location-data';

export interface MapCanvasStateOptions {
	/** 视口中心（null = 尚未判定，不请求）。 */
	getCenter: () => { lat: number; lng: number } | null;
	/** 当前启用的图钉类型筛选。 */
	getFilters: () => Set<PinType>;
	/** 稳定边界（手势/缩放结束才回写的那份）。 */
	getStableBounds: () => MapBounds;
	/** 中心来源是否已判定完毕（判定后仍无中心才落「无法获取位置」错误）。 */
	isCenterResolved: () => boolean;
	/** 重试连带动作（组件侧 bump 中心配置 reload key），可选。 */
	onRetry?: () => void;
}

export function createMapCanvasState(options: MapCanvasStateOptions) {
	let pins = $state<Pin[]>([]);
	let heatBuckets = $state<MapHeatBucket[]>([]);
	let counts = $state<Record<PinType, number>>({ user: 0, group: 0, moment: 0 });
	let loading = $state(true);
	let error = $state<string | null>(null);
	let reloadKey = $state(0);

	function retryLoad() {
		// 重试：既重拉地图数据，也重拉中心配置——中心拿不到时只 bump 本 key
		// 会原地踏步（effect 因 !center 直接返回），重试按钮形同死的。
		options.onRetry?.();
		reloadKey++;
	}

	$effect(() => {
		void reloadKey;
		const center = options.getCenter();
		const types = [...options.getFilters()];
		const bounds = options.getStableBounds();
		if (!center) {
			// 中心未定不请求；判定完毕仍无中心（本地无锚点且配置拿不到）就结束 loading
			// 并给可重试提示，否则「加载地图数据…」永久空转、用户只见空白。
			if (options.isCenterResolved()) {
				loading = false;
				error = '暂时无法获取位置信息';
			}
			return;
		}
		let cancelled = false;
		loading = true;
		error = null;
		// 首屏无视口时用固定 delta 兜底；mapBounds 就绪后（hasViewport）走真实视口。
		const hasViewport = bounds.maxLat !== bounds.minLat || bounds.maxLng !== bounds.minLng;
		const request = hasViewport
			? fetchMapCanvasByViewport({ bounds: { north: bounds.maxLat, south: bounds.minLat, east: bounds.maxLng, west: bounds.minLng } }, types)
			: fetchMapCanvas(center, 0.045, types);
		void request
			.then((r) => {
				if (cancelled) return;
				pins = r.pins;
				heatBuckets = r.heatBuckets;
				counts = r.counts;
				loading = false;
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				// 无 catch 时请求失败会让 loading 永久卡在 true、地图一片空白
				error = err instanceof Error ? err.message : '地图数据加载失败';
				loading = false;
			});
		return () => {
			cancelled = true;
		};
	});

	return {
		get pins() { return pins; },
		get heatBuckets() { return heatBuckets; },
		get counts() { return counts; },
		get loading() { return loading; },
		get error() { return error; },
		retryLoad
	};
}
