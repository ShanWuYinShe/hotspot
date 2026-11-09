import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// 地图锚点 helper：读写的格式兼容与非法输入校验（进入地图不再自动定位）。
const { readLastLocation, writeLastLocation } = await import('./map-auto-locate');

const KEY = 'hotspot_last_location';

describe('readLastLocation', () => {
	beforeEach(() => localStorage.clear());
	it('历史数组格式不再支持，按无锚点处理', () => {
		localStorage.setItem(KEY, '[39.9,116.4]');
		expect(readLastLocation()).toBeNull();
	});

	it('对象格式带名称时保留名称', () => {
		localStorage.setItem(KEY, JSON.stringify({ lat: 30.66, lng: 104.06, name: '成都' }));
		expect(readLastLocation()).toEqual({ lat: 30.66, lng: 104.06, name: '成都' });
	});

	it('名称为空串时按无名称处理', () => {
		localStorage.setItem(KEY, JSON.stringify({ lat: 30.66, lng: 104.06, name: '  ' }));
		expect(readLastLocation()).toEqual({ lat: 30.66, lng: 104.06 });
	});

	const invalid: Array<[string, string | null]> = [
		['缺失', null],
		['非法 JSON', 'xxx'],
		['非坐标对象', '{"a":1}'],
		['非数字数组', '["a","b"]'],
		['数组越界', '[91,116.4]'],
		['对象越界', '{"lat":91,"lng":116.4}'],
		['JSON null', 'null']
	];

	it.each(invalid)('%s 时返回 null', (_label, raw) => {
		if (raw === null) localStorage.removeItem(KEY);
		else localStorage.setItem(KEY, raw);
		expect(readLastLocation()).toBeNull();
	});
});

describe('writeLastLocation', () => {
	beforeEach(() => localStorage.clear());

	it('写入坐标与名称，供附近页/侧栏同一口径读取', () => {
		writeLastLocation({ lat: 30.66, lng: 104.06, name: '成都' });
		expect(readLastLocation()).toEqual({ lat: 30.66, lng: 104.06, name: '成都' });
	});

	it('名称为空白时不写入该字段（避免显示空串）', () => {
		writeLastLocation({ lat: 30.66, lng: 104.06, name: '   ' });
		expect(readLastLocation()).toEqual({ lat: 30.66, lng: 104.06 });
	});

	it('越界坐标拒绝写入（后端 S2 索引会拒，不应固化到本地）', () => {
		writeLastLocation({ lat: 999, lng: 104.06 });
		expect(localStorage.getItem(KEY)).toBeNull();
	});
});
