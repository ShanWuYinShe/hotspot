// 客户端配置默认位置（迁移自 React useDefaultLocation 的纯 TS 部分）。
//
// reference_point 是每个 feed/search 端点的必填参数，但前端尚无浏览器定位
// 来源。按跨端边界（根 AGENTS.md：距离/位置由后端计算），回退到服务端
// ClientConfig.defaults.default_location.point。全应用缓存一次（配置近静态）。

import { getClientConfig } from '@/api';

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
			const res = await getClientConfig();
			if (res.code === 0) {
				const loc = (res.data as any)?.defaults?.default_location;
				if (loc?.point?.latitude != null && loc?.point?.longitude != null) {
					cached = {
						point: { latitude: loc.point.latitude, longitude: loc.point.longitude },
						location_name: loc.location_name ?? ''
					};
					return cached;
				}
			}
		} catch {
			/* network/mock unavailable — caller handles null */
		} finally {
			inflight = null;
		}
		return null;
	})();
	return inflight;
}
