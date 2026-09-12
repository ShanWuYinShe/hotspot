// Map source store — 底图源偏好（高德 GCJ-02 / OpenStreetMap WGS-84）。
// ThemeContext（runes 模块单例），与 theme.svelte.ts 同模式：localStorage
// 持久化 + 非法值回退；设置页与地图页快速切换控件共用，二者实时同步。

export type MapSource = 'amap' | 'osm';

export const MAP_SOURCE_OPTIONS = [
	{ id: 'amap', label: '高德', note: '国内细节丰富（GCJ-02）' },
	{ id: 'osm', label: 'OpenStreetMap', note: '全球通用（WGS-84）' }
] as const;

const MAP_SOURCE_KEY = 'mapSource';

let source = $state<MapSource>('amap');
let initialized = $state(false);

function readSaved(): MapSource {
	try {
		const saved = localStorage.getItem(MAP_SOURCE_KEY);
		if (saved === 'amap' || saved === 'osm') return saved;
	} catch {}
	return 'amap';
}

/** 由根布局/地图页调用一次：读取持久化偏好。幂等，SSR 下安全跳过。 */
export function initMapSource(): void {
	if (initialized) return;
	initialized = true;
	source = readSaved();
}

/** 切换底图源（持久化）。 */
export function setMapSource(next: MapSource): void {
	source = next;
	try {
		localStorage.setItem(MAP_SOURCE_KEY, next);
	} catch {}
}

export const mapSourceStore = {
	get source() {
		return source;
	}
};
