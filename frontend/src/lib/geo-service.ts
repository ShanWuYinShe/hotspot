// 地理编码服务（按地图源路由）。
//   高德：浏览器直连 restapi.amap.com（实测响应带 Access-Control-Allow-Origin:*，
//         支持跨域直调）。key 经构建期注入（VITE_AMAP_KEY）；Web 服务 key 在前端
//         可被提取盗刷，请在高德控制台为该 key 单独设置低配额与用量告警。
//         key 未配置时本层直接抛 GeoServiceUnavailable 降级 OSM，不发无效请求。
//   OSM：/osm-service/ → nginx 同源代理 → nominatim.openstreetmap.org
//         （公共 Nominatim 政策要求服务端限流/缓存，且国内直连不可达，保留代理）。
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

/** 高德 Web 服务 key（构建期注入）。空值表示未配置，高德源直接降级 OSM。 */
function amapKey(): string {
	return ((import.meta.env?.VITE_AMAP_KEY as string | undefined) ?? '').trim();
}

/** 高德 restapi 直连 URL（key 注入 query；协议 https，国内可直连无需代理）。 */
function amapUrl(path: string, params: URLSearchParams): string {
	params.set('key', amapKey());
	return `https://restapi.amap.com${path}?${params.toString()}`;
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
	if (!amapKey()) throw new GeoServiceUnavailable();
	const data = (await getJson(amapUrl('/v3/place/text', params), opts)) as {
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

/** 逆地理编码（OSM Nominatim reverse）。 */
async function osmReverse(lat: number, lng: number, opts?: FetchOpts): Promise<string | null> {
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

/** 高德逆地理编码：WGS-84 转 GCJ-02 后查询，返回格式化地址。 */
async function amapReverse(lat: number, lng: number, opts?: FetchOpts): Promise<string | null> {
	if (!amapKey()) throw new GeoServiceUnavailable();
	const g = wgs84ToGcj02(lat, lng);
	const params = new URLSearchParams({ location: `${g.lng.toFixed(6)},${g.lat.toFixed(6)}` });
	const data = (await getJson(amapUrl('/v3/geocode/regeo', params), opts)) as {
		status?: string;
		regeocode?: { formatted_address?: string }
	};
	const addr = data.regeocode?.formatted_address?.trim();
	return addr && addr !== '[]' ? addr : null;
}

/** 关键字搜索地点（WGS-84 输出）。失败抛 GeoServiceUnavailable，由调用方降级。 */
export async function searchPlaces(
	source: MapSource,
	query: string,
	near: { lat: number; lng: number } | null,
	opts?: FetchOpts
): Promise<GeoPlace[]> {
	if (source !== 'amap') return osmSearch(query, opts);
	try {
		return await amapSearch(query, near, opts);
	} catch (err) {
		// 高德编码服务不可用（如 Web 服务 key 未配置/限流）时降级到 OSM
		// 编码：底图仍是高德瓦片，结果来自 OSM（坐标同为 WGS-84）。
		if (err instanceof GeoServiceUnavailable) return osmSearch(query, opts);
		throw err;
	}
}

/** 逆地理编码：坐标 → 文字地址（无结果返回 null，服务不可用抛错）。 */
export async function reverseGeocode(
	source: MapSource,
	lat: number,
	lng: number,
	opts?: FetchOpts
): Promise<string | null> {
	if (source !== 'amap') return osmReverse(lat, lng, opts);
	try {
		return await amapReverse(lat, lng, opts);
	} catch (err) {
		if (err instanceof GeoServiceUnavailable) return osmReverse(lat, lng, opts);
		throw err;
	}
}
