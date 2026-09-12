// 浏览器定位封装（Geolocation API）。应用数据口径 WGS-84 —— GPS 原始
// 坐标即为 WGS-84，无需换算；高德瓦片渲染的 GCJ 纠偏由 MapCanvas 承担。
// 错误统一映射为可操作的中文提示（拒绝授权/不可用/超时）。

export interface GeoFix {
	lat: number;
	lng: number;
	/** 定位精度半径（米），可向下取整展示 */
	accuracy: number;
}

export type GeoErrorCode = 'denied' | 'unavailable' | 'timeout' | 'unsupported' | 'failed';

export class GeoError extends Error {
	code: GeoErrorCode;
	constructor(code: GeoErrorCode, message: string) {
		super(message);
		this.code = code;
	}
}

/** GeolocationPositionError.code → 可操作的中文提示。 */
function mapNativeError(err: GeolocationPositionError): GeoError {
	switch (err.code) {
		case err.PERMISSION_DENIED:
			return new GeoError('denied', '定位被拒绝：请在浏览器地址栏授权位置权限后重试');
		case err.POSITION_UNAVAILABLE:
			return new GeoError('unavailable', '暂时无法获取位置信息');
		case err.TIMEOUT:
			return new GeoError('timeout', '定位超时，请稍后重试');
		default:
			return new GeoError('failed', '定位失败，请稍后重试');
	}
}

/**
 * 请求一次当前位置。enableHighAccuracy 优先 GPS；30s 内的缓存坐标可直接复用。
 * @throws GeoError 定位失败时（UI 捕获后以 toast/行内提示呈现）。
 */
export function requestGeoPosition(timeoutMs = 10_000): Promise<GeoFix> {
	return new Promise((resolve, reject) => {
		if (typeof navigator === 'undefined' || !navigator.geolocation) {
			reject(new GeoError('unsupported', '当前浏览器不支持定位'));
			return;
		}
		navigator.geolocation.getCurrentPosition(
			(pos) =>
				resolve({
					lat: pos.coords.latitude,
					lng: pos.coords.longitude,
					accuracy: pos.coords.accuracy ?? 0
				}),
			(err) => reject(mapNativeError(err)),
			{ enableHighAccuracy: true, timeout: timeoutMs, maximumAge: 30_000 }
		);
	});
}
