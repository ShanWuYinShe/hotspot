// 客户端配置默认位置。
//
// reference_point 是每个 feed/search 端点的可选参数:不传时服务端按回退链解析
// (请求显式传参 → 最近一次 users/update_location → ClientConfig.defaults.default_location.point)。
// 前端尚无浏览器定位来源,故仅在拿得到 default_location 时显式传参;拿不到就不传,
// 交给服务端回退链。全应用缓存一次(配置近静态)。

import { getClientConfig } from '@/api';
import { loadBootstrap } from '@/lib/bootstrap-cache';

export interface GeoPoint {
	latitude: number;
	longitude: number;
}

export interface DefaultLocation {
	point: GeoPoint;
	location_name: string;
}

let cached: DefaultLocation | null = null;
let inflight: Promise<DefaultLocation | null> | null = null;

export async function loadDefaultLocation(): Promise<DefaultLocation | null> {
	if (cached) return cached;
	if (inflight) return inflight;
	inflight = (async () => {
		try {
			// 优先用冷启动 bootstrap 聚合响应里的 client_config（同一次请求已下发，
			// spec api_spec_system.md → bootstrap），避免首屏再单独请求一次公共配置。
			// 仅当该段被省略（bootstrap 缓存命中精简响应）时才回退独立端点。
			let config = (await loadBootstrap())?.client_config ?? null;
			if (!config) {
				const res = await getClientConfig();
				if (res.code === 0) config = res.data ?? null;
			}
			const loc = config?.defaults?.default_location;
			if (loc?.point?.latitude != null && loc?.point?.longitude != null) {
				cached = {
					point: { latitude: loc.point.latitude, longitude: loc.point.longitude },
					location_name: loc.location_name ?? ''
				};
				return cached;
			}
		} catch {
			/* 网络不可用（后端未就绪等）——调用方按 null 处理 */
		} finally {
			inflight = null;
		}
		return null;
	})();
	return inflight;
}
