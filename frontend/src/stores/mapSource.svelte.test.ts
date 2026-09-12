import { describe, expect, it } from 'vitest';
import {
	MAP_SOURCE_OPTIONS,
	setMapSource,
	mapSourceStore,
	type MapSource
} from './mapSource.svelte';

// mapSource store：纯函数部分（选项常量 + setMapSource 持久化写回）。
// SSR 无 localStorage 环境 store 初值恒为 'amap'，写入持久化被 try/catch 吞掉。

describe('MAP_SOURCE_OPTIONS', () => {
	it('固定提供高德与 OSM 两源', () => {
		expect(MAP_SOURCE_OPTIONS.map((o) => o.id)).toEqual(['amap', 'osm']);
	});
});

describe('setMapSource', () => {
	it('更新 store 当前值（非法值由调用侧类型约束）', () => {
		for (const id of MAP_SOURCE_OPTIONS.map((o) => o.id) as MapSource[]) {
			setMapSource(id);
			expect(mapSourceStore.source).toBe(id);
		}
	});
});
