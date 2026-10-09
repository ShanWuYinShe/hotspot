// GCJ-02 反演精度契约：滚轮/捏合每缩放一级都要经过一次 WGS↔GCJ 往返，
// 一次反演残留 ~1m 系统性偏差会逐级同向累积（14→18 级实测漂移约 6m）。
// gcj02ToWgs84 须迭代到厘米级，否则高 zoom 下“我的位置”肉眼可见滑 off。
import { describe, it, expect } from 'vitest';
import { wgs84ToGcj02, gcj02ToWgs84, outOfChina } from '../map-projection';

const toM = (dLat: number, dLng: number, lat: number) =>
	Math.hypot(dLat * 111320, dLng * 111320 * Math.cos((lat * Math.PI) / 180));

// 国内典型点：北京 / 上海 / 广州 / 成都 / 中国边陲（拉萨）
const DOMESTIC: [number, number][] = [
	[39.9042, 116.4074],
	[31.2304, 121.4737],
	[23.1291, 113.2644],
	[30.5728, 104.0668],
	[29.6525, 91.1006]
];

describe('GCJ-02 反演精度', () => {
	it.each(DOMESTIC)('国内点 (%f, %f) 往返误差 < 0.05m', (lat, lng) => {
		const g = wgs84ToGcj02(lat, lng);
		// 确认偏移量非零（否则用例在测空路径）
		expect(toM(g.lat - lat, g.lng - lng, lat)).toBeGreaterThan(10);
		const back = gcj02ToWgs84(g.lat, g.lng);
		expect(toM(back.lat - lat, back.lng - lng, lat)).toBeLessThan(0.05);
	});

	it('境外直通：反演恒等（不引入偏移）', () => {
		// 出了中国范围（如东京），正/反变换均为恒等映射
		expect(outOfChina(35.6762, 139.6503)).toBe(true);
		expect(wgs84ToGcj02(35.6762, 139.6503)).toEqual({ lat: 35.6762, lng: 139.6503 });
		expect(gcj02ToWgs84(35.6762, 139.6503)).toEqual({ lat: 35.6762, lng: 139.6503 });
	});
});
