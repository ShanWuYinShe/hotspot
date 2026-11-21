import { describe, expect, it } from 'vitest';
import { ANCHOR_FALLBACK_NAME, resolvePublishLocation } from './publish-location';

const fallback = {
	point: { latitude: 35.154922, longitude: 112.682524 },
	location_name: '济源市南天门'
};

describe('resolvePublishLocation', () => {
	it('浏览锚点优先于默认位置（锚点带名称）', () => {
		const loc = resolvePublishLocation({ lat: 34.2604, lng: 117.1855, name: '云龙湖东岸' }, fallback);
		expect(loc.point).toEqual({ latitude: 34.2604, longitude: 117.1855 });
		expect(loc.location_name).toBe('云龙湖东岸');
	});

	it('浏览锚点无名称时回退「自定义位置」，不使用默认位置名', () => {
		const loc = resolvePublishLocation({ lat: 34.2604, lng: 117.1855 }, fallback);
		expect(loc.point).toEqual({ latitude: 34.2604, longitude: 117.1855 });
		expect(loc.location_name).toBe(ANCHOR_FALLBACK_NAME);
	});

	it('无浏览锚点时回退 ClientConfig 默认位置', () => {
		expect(resolvePublishLocation(null, fallback)).toEqual({
			point: { latitude: 35.154922, longitude: 112.682524 },
			location_name: '济源市南天门'
		});
	});

	it('锚点与默认位置都缺失时退化为零点（仅配置不可用场景）', () => {
		expect(resolvePublishLocation(null, null)).toEqual({
			point: { latitude: 0, longitude: 0 },
			location_name: ''
		});
	});

	it('锚点名称首尾空白会被裁剪', () => {
		const loc = resolvePublishLocation({ lat: 1, lng: 2, name: '  湖边  ' }, fallback);
		expect(loc.location_name).toBe('湖边');
	});
});
