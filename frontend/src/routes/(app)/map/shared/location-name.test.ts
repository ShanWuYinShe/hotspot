// 地图「浏览位置」地名解析单测：锁死优先级、距离门槛与核心回归——
// 中心远离默认位置点时绝不能显示服务端默认名。
import { describe, it, expect } from 'vitest';
import {
	NAME_MATCH_RADIUS_M,
	MY_LOCATION_RADIUS_M,
	approxDistanceMeters,
	locationGridKey,
	resolveLocationName,
	type LatLng,
	type NamedPoint
} from './location-name';

// dev 报障真实场景：GPS 在郑州，服务端默认位置在济源（约 100km 外）。
const ZHENGZHOU_GPS: LatLng = { lat: 34.7466, lng: 113.6254 };
const JIYUAN_DEFAULT: NamedPoint = { lat: 35.0904, lng: 112.5908, name: '济源市南天门' };

const CENTER: LatLng = { lat: 34.2604, lng: 117.1855 };
const METERS_PER_DEGREE_LAT = 111194.9266; // 2πR/360（R=6371000，与实现同口径）

/** 从 center 向北偏移 meters 米的点：用于构造半径内/外的判定点。 */
function northOf(center: LatLng, meters: number): LatLng {
	return { lat: center.lat + meters / METERS_PER_DEGREE_LAT, lng: center.lng };
}

describe('半径常量', () => {
	it('命名点匹配半径 500m、我的位置半径 300m（Lead 接线依赖该契约）', () => {
		expect(NAME_MATCH_RADIUS_M).toBe(500);
		expect(MY_LOCATION_RADIUS_M).toBe(300);
	});
});

describe('approxDistanceMeters', () => {
	it('同一点距离为 0', () => {
		expect(approxDistanceMeters(CENTER, { ...CENTER })).toBe(0);
	});

	it('0.001° 纬度 ≈ 111m（±2m）', () => {
		const d = approxDistanceMeters({ lat: 34, lng: 113 }, { lat: 34.001, lng: 113 });
		expect(d).toBeGreaterThan(109);
		expect(d).toBeLessThan(113);
	});

	it('对称：a→b 与 b→a 等距', () => {
		const a = { lat: 34.7466, lng: 113.6254 };
		const b = { lat: 34.751, lng: 113.63 };
		expect(approxDistanceMeters(a, b)).toBeCloseTo(approxDistanceMeters(b, a), 6);
	});

	it('经度方向随纬度按 cos 收缩（北纬 60° 的 0.001° ≈ 55.6m）', () => {
		const d = approxDistanceMeters({ lat: 60, lng: 113 }, { lat: 60, lng: 113.001 });
		expect(d).toBeGreaterThan(53.6);
		expect(d).toBeLessThan(57.6);
	});
});

describe('locationGridKey', () => {
	it('四位小数格式化，与地图页逆地理缓存 key 同口径', () => {
		expect(locationGridKey({ lat: 34.260449, lng: 117.185512 })).toBe('34.2604,117.1855');
	});

	it('同格内不同点得到相同 key', () => {
		expect(locationGridKey({ lat: 34.26041, lng: 117.18551 })).toBe(
			locationGridKey({ lat: 34.26044, lng: 117.18554 })
		);
	});

	it('负坐标与补零保持 toFixed(4) 形态', () => {
		expect(locationGridKey({ lat: -34.26041, lng: -117.1855 })).toBe('-34.2604,-117.1855');
		expect(locationGridKey({ lat: 35.1, lng: 112.6 })).toBe('35.1000,112.6000');
	});
});

describe('resolveLocationName 优先级', () => {
	const everything = {
		center: CENTER,
		selectedName: '云龙湖',
		pinName: '图钉名',
		reverseName: '逆地理名',
		anchor: { ...CENTER, name: '锚点名' },
		myLocation: { ...CENTER },
		defaultLocation: { ...CENTER, name: '服务端默认名' }
	};

	it('center 为空时返回「未命名区域」，即使其它名字齐全', () => {
		expect(resolveLocationName({ ...everything, center: null })).toBe('未命名区域');
	});

	it('选中地点优先于图钉/逆地理/锚点/我的位置/默认名', () => {
		expect(resolveLocationName(everything)).toBe('云龙湖');
	});

	it('无选中地点时用选中图钉名', () => {
		expect(resolveLocationName({ ...everything, selectedName: null })).toBe('图钉名');
	});

	it('无选中地点/图钉时用同格逆地理名', () => {
		expect(resolveLocationName({ ...everything, selectedName: null, pinName: null })).toBe('逆地理名');
	});

	it('逆地理缺席时用附近锚点名', () => {
		expect(
			resolveLocationName({ ...everything, selectedName: null, pinName: null, reverseName: null })
		).toBe('锚点名');
	});

	it('锚点与我的位置都在半径内时，锚点名优先（优先级在 myLocation 之前）', () => {
		expect(
			resolveLocationName({
				center: CENTER,
				anchor: { ...northOf(CENTER, 200), name: '锚点名' },
				myLocation: northOf(CENTER, 100)
			})
		).toBe('锚点名');
	});

	it('锚点不在半径内时让位给「我的位置」，而不是显示远处的锚点名', () => {
		expect(
			resolveLocationName({
				center: CENTER,
				anchor: { ...northOf(CENTER, 600), name: '锚点名' },
				myLocation: northOf(CENTER, 100),
				defaultLocation: { ...CENTER, name: '服务端默认名' }
			})
		).toBe('我的位置');
	});

	it('我的位置优先于服务端默认名', () => {
		expect(
			resolveLocationName({
				center: CENTER,
				myLocation: northOf(CENTER, 100),
				defaultLocation: { ...northOf(CENTER, 200), name: '服务端默认名' }
			})
		).toBe('我的位置');
	});

	it('以上都缺席时用附近的服务端默认名', () => {
		expect(
			resolveLocationName({ center: northOf(JIYUAN_DEFAULT, 200), defaultLocation: JIYUAN_DEFAULT })
		).toBe('济源市南天门');
	});

	it('空串/空白串视为没有名字，继续向下兜底', () => {
		expect(
			resolveLocationName({
				center: CENTER,
				selectedName: '',
				pinName: '   ',
				reverseName: null,
				anchor: { ...CENTER, name: '锚点名' }
			})
		).toBe('锚点名');
	});

	it('名字两端空白 trim 后返回', () => {
		expect(resolveLocationName({ center: CENTER, selectedName: ' 云龙湖 ' })).toBe('云龙湖');
	});

	it('选中名字不受距离门槛限制（选中什么就显示什么）', () => {
		expect(resolveLocationName({ center: ZHENGZHOU_GPS, selectedName: '云龙湖' })).toBe('云龙湖');
	});
});

describe('resolveLocationName 半径门槛', () => {
	it('锚点在 500m 内采用其名字，超出则忽略', () => {
		expect(
			resolveLocationName({ center: CENTER, anchor: { ...northOf(CENTER, 450), name: '锚点名' } })
		).toBe('锚点名');
		expect(
			resolveLocationName({ center: CENTER, anchor: { ...northOf(CENTER, 550), name: '锚点名' } })
		).toBe('未命名区域');
	});

	it('半径边界：499m 采用、501m 忽略', () => {
		expect(
			resolveLocationName({ center: CENTER, anchor: { ...northOf(CENTER, 499), name: '锚点名' } })
		).toBe('锚点名');
		expect(
			resolveLocationName({ center: CENTER, anchor: { ...northOf(CENTER, 501), name: '锚点名' } })
		).toBe('未命名区域');
	});

	it('GPS 在 300m 内视为「我的位置」，超出不显示', () => {
		expect(resolveLocationName({ center: CENTER, myLocation: northOf(CENTER, 250) })).toBe('我的位置');
		expect(resolveLocationName({ center: CENTER, myLocation: northOf(CENTER, 350) })).toBe('未命名区域');
	});

	it('默认位置点在 500m 内显示默认名，超出显示「未命名区域」', () => {
		expect(
			resolveLocationName({ center: northOf(JIYUAN_DEFAULT, 200), defaultLocation: JIYUAN_DEFAULT })
		).toBe('济源市南天门');
		expect(
			resolveLocationName({ center: northOf(JIYUAN_DEFAULT, 600), defaultLocation: JIYUAN_DEFAULT })
		).toBe('未命名区域');
	});
});

describe('核心回归：中心远离默认位置点', () => {
	it('郑州 GPS + 服务端默认济源：不得显示「济源市南天门」', () => {
		const name = resolveLocationName({
			center: ZHENGZHOU_GPS,
			reverseName: null,
			defaultLocation: JIYUAN_DEFAULT
		});
		expect(name).not.toBe('济源市南天门');
		expect(name).toBe('未命名区域');
	});

	it('郑州 GPS + 我的位置：显示「我的位置」，仍不显示默认名', () => {
		const name = resolveLocationName({
			center: ZHENGZHOU_GPS,
			myLocation: ZHENGZHOU_GPS,
			defaultLocation: JIYUAN_DEFAULT
		});
		expect(name).toBe('我的位置');
		expect(name).not.toBe('济源市南天门');
	});

	it('中心落在默认点附近时仍可显示默认名（门槛不是禁用）', () => {
		expect(
			resolveLocationName({ center: northOf(JIYUAN_DEFAULT, 300), defaultLocation: JIYUAN_DEFAULT })
		).toBe('济源市南天门');
	});
});

describe('null 边界', () => {
	it('只给 center 时返回「未命名区域」', () => {
		expect(resolveLocationName({ center: CENTER })).toBe('未命名区域');
	});

	it('可选字段显式 null 时不崩且继续兜底', () => {
		expect(
			resolveLocationName({
				center: CENTER,
				selectedName: null,
				pinName: null,
				reverseName: null,
				anchor: null,
				myLocation: null,
				defaultLocation: null
			})
		).toBe('未命名区域');
	});

	it('锚点/默认点没有名字时不采用（即使中心就在其位置上）', () => {
		expect(
			resolveLocationName({
				center: CENTER,
				anchor: { ...CENTER, name: null },
				defaultLocation: { ...CENTER, name: null }
			})
		).toBe('未命名区域');
	});

	it('锚点/默认点缺省 name 字段时不采用', () => {
		expect(
			resolveLocationName({
				center: CENTER,
				anchor: { ...CENTER },
				defaultLocation: { ...CENTER }
			})
		).toBe('未命名区域');
	});
});
