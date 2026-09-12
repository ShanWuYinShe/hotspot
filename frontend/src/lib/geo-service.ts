// 地理编码服务（按地图源路由，全部经 nginx 同源代理，前端零密钥）。
//   高德：/amap-service/ → restapi.amap.com（key/jscode 由服务器侧注入；
//         未配置时 nginx 返回 503，本层转为 GeoServiceUnavailable）。
//   OSM：/osm-service/ → nominatim.openstreetmap.org（代理侧限流 1r/s）。
//
// 坐标口径：应用数据 WGS-84。高德 Web 服务出入参均为 GCJ-02，故搜索结果
// 的坐标经 gcj02ToWgs84 反演回 WGS-84，"near" 中心先 wgs84ToGcj02 再传参；
// OSM 全程 WGS-84 直通。逆地理只取文字地址，无坐标系歧义。

import { gcj02ToWgs84, wgs84ToGcj02 } from '@/routes/(app)/map/shared/map-projection';
import type { MapSource } from '@/stores/mapSource.svelte';

export interface GeoPlace {
	displayName: string;
	lat: number;
	lng: number;
	/** 补充说明（地址/类别），展示用 */
	desc?: string;
}

export class GeoServiceUnavailable extends Error {
	constructor() {
		super('地理编码服务不可用');
		this.name = 'GeoServiceUnavailable';
	}
}

interface FetchOpts {
	signal?: AbortSignal;
}

/** 同源 JSON GET；非 2xx 一律按服务不可用处理（含 nginx 503 占位）。 */
async function getJson(url: string, opts?: FetchOpts): Promise<unknown> {
	const res = await fetch(url, { signal: opts?.signal, credentials: 'omit' });
	if (!res.ok) throw new GeoServiceUnavailable();
	return res.json();
}

/** 高德关键字搜索（place/text），near 提供时按距离排序。 */
async function amapSearch(query: string, near: { lat: number; lng: number } | null, opts?: FetchOpts): Promise<GeoPlace[]> {
	const params = new URLSearchParams({ keywords: query, offset: '8', page: '1' });
	if (near) {
		const g = wgs84ToGcj02(near.lat, near.lng);
		// 高德 location 口径为「经度,纬度」
		params.set('location', `${g.lng.toFixed(6)},${g.lat.toFixed(6)}`);
	}
	const data = (await getJson(`/amap-service/v3/place/text?${params}`, opts)) as {
		status?: string;
		pois?: Array<{ name?: string; address?: string; pname?: string; cityname?: string; adname?: string; location?: string }>;
	};
	if (data.status !== '1' || !Array.isArray(data.pois)) return [];
	return data.pois
		.filter((poi) => poi.location && poi.name)
		.map((poi) => {
			const [lng, lat] = poi.location!.split(',').map(Number);
			const wgs = gcj02ToWgs84(lat, lng);
			return {
				displayName: poi.name!,
				lat: wgs.lat,
				lng: wgs.lng,
				desc: [poi.pname, poi.cityname, poi.adname, poi.address].filter(Boolean).join(' · ')
			};
		});
}

/** OSM 关键字搜索（Nominatim search）。 */
async function osmSearch(query: string, opts?: FetchOpts): Promise<GeoPlace[]> {
	const params = new URLSearchParams({ q: query, format: 'jsonv2', limit: '8', 'accept-language': 'zh-CN' });
	const data = (await getJson(`/osm-service/search?${params}`, opts)) as Array<{
		display_name?: string;
		lat?: string;
		lon?: string;
		type?: string;
	}>;
	return (Array.isArray(data) ? data : [])
		.filter((r) => r.lat && r.lon && r.display_name)
		.map((r) => ({
			displayName: r.display_name!.split(',').slice(0, 2).map((s) => s.trim()).join(' · '),
			lat: Number(r.lat),
			lng: Number(r.lon),
			desc: [r.type, r.display_name].filter(Boolean).join(' · ')
		}));
}

/** 关键字搜索地点（WGS-84 输出）。失败抛 GeoServiceUnavailable，由调用方降级。 */
export function searchPlaces(
	source: MapSource,
	query: string,
	near: { lat: number; lng: number } | null,
	opts?: FetchOpts
): Promise<GeoPlace[]> {
	return source === 'amap' ? amapSearch(query, near, opts) : osmSearch(query, opts);
}

/** 逆地理编码：坐标 → 文字地址（无结果返回 null，服务不可用抛错）。 */
export async function reverseGeocode(
	source: MapSource,
	lat: number,
	lng: number,
	opts?: FetchOpts
): Promise<string | null> {
	if (source === 'amap') {
		const g = wgs84ToGcj02(lat, lng);
		const data = (await getJson(
			`/amap-service/v3/geocode/regeo?location=${g.lng.toFixed(6)},${g.lat.toFixed(6)}`,
			opts
		)) as { status?: string; regeocode?: { formatted_address?: string } };
		const addr = data.regeocode?.formatted_address?.trim();
		return addr && addr !== '[]' ? addr : null;
	}
	const params = new URLSearchParams({
		lat: String(lat),
		lon: String(lng),
		format: 'jsonv2',
		zoom: '17',
		'accept-language': 'zh-CN'
	});
	const data = (await getJson(`/osm-service/reverse?${params}`, opts)) as {
		display_name?: string;
		error?: string;
	};
	return data.display_name?.trim() ?? null;
}
