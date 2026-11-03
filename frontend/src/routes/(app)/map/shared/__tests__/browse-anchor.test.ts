import { describe, it, expect } from 'vitest';
import {
	ANCHOR_SAME_TOLERANCE_METERS,
	anchorActionLabel,
	anchorDisplayName,
	anchorDistanceMeters,
	anchorPurposeHint,
	isSameAnchorPoint,
	resolveAnchorName
} from '../browse-anchor';

const A = { lat: 34.6645, lng: 112.4912 };
// 约 111m（纬度 0.001°）
const B = { lat: 34.6655, lng: 112.4912 };

describe('browse-anchor', () => {
	it('锚点距离：同点为 0，纬度差 0.001° 约 111m', () => {
		expect(anchorDistanceMeters(A, A)).toBe(0);
		expect(anchorDistanceMeters(A, B)).toBeGreaterThan(100);
		expect(anchorDistanceMeters(A, B)).toBeLessThan(120);
	});

	it('isSameAnchorPoint：容差内视为同一锚点，超出不是', () => {
		expect(isSameAnchorPoint(A, A)).toBe(true);
		expect(isSameAnchorPoint(A, B)).toBe(false);
		const nearby = { lat: A.lat + 0.0002, lng: A.lng }; // ≈22m
		expect(isSameAnchorPoint(nearby, A)).toBe(true);
		expect(isSameAnchorPoint(null, A)).toBe(false);
		expect(isSameAnchorPoint(A, null)).toBe(false);
	});

	it('anchorDisplayName：无名称回退「自定义位置」', () => {
		expect(anchorDisplayName(null)).toBe('自定义位置');
		expect(anchorDisplayName({ ...A, name: '  ' })).toBe('自定义位置');
		expect(anchorDisplayName({ ...A, name: '云龙湖东岸' })).toBe('云龙湖东岸');
	});

	it('anchorActionLabel：当前锚点 / 选中地点 / 地图中心三种文案', () => {
		expect(anchorActionLabel({ hasAnchor: true, sameAsCenter: true })).toBe('当前浏览锚点');
		expect(anchorActionLabel({ hasAnchor: false, sameAsCenter: false, selectedLocationName: '云龙湖' })).toBe(
			'以此地点为浏览锚点'
		);
		expect(anchorActionLabel({ hasAnchor: true, sameAsCenter: false })).toBe('将地图中心设为浏览锚点');
	});

	it('resolveAnchorName：选中地点优先；未命名/我的位置不落库', () => {
		expect(resolveAnchorName('云龙湖东岸', '济源市南天门')).toBe('云龙湖东岸');
		expect(resolveAnchorName('', '济源市南天门')).toBe('济源市南天门');
		expect(resolveAnchorName(null, '未命名区域')).toBeUndefined();
		expect(resolveAnchorName(null, '我的位置')).toBeUndefined();
		expect(resolveAnchorName(null, '')).toBeUndefined();
	});

	it('anchorPurposeHint 按是否已有锚点给用途说明', () => {
		expect(anchorPurposeHint(true)).toContain('附近页');
		expect(anchorPurposeHint(false)).toContain('设置后');
	});

	it('容差常量为 50m（拖拽/逆地理抖动不触发文案跳变）', () => {
		expect(ANCHOR_SAME_TOLERANCE_METERS).toBe(50);
	});
});
