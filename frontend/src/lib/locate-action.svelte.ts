// 「定位到我」动作（地图页桌面/移动共用）：浏览器 GPS → 回调交出 WGS-84 坐标。
// 状态与错误由调用方消费（按钮 loading 态 + toast/行内提示），本模块只做
// 单次定位的进行中管理，不缓存 fix（每次点击都是新鲜定位）。

import { GeoError, requestGeoPosition, type GeoFix } from '@/lib/geolocation.svelte';

export interface LocateAction {
	/** 定位请求进行中（按钮转圈/禁用） */
	readonly locating: boolean;
	/** 触发一次定位；成功/失败经回调交出（错误为 GeoError，含中文提示）。 */
	locate(): Promise<void>;
}

export function createLocateAction(handlers: {
	onSuccess: (fix: GeoFix) => void;
	onError?: (err: GeoError) => void;
}): LocateAction {
	let locating = $state(false);

	async function locate(): Promise<void> {
		if (locating) return;
		locating = true;
		try {
			handlers.onSuccess(await requestGeoPosition());
		} catch (err) {
			handlers.onError?.(
				err instanceof GeoError ? err : new GeoError('failed', '定位失败，请稍后重试')
			);
		} finally {
			locating = false;
		}
	}

	return {
		get locating() {
			return locating;
		},
		locate
	};
}
