// 四端外壳分派契约：平台/方向 → 端壳种类（与 MainLayout 分派规则同口径）。
import { describe, expect, it } from 'vitest';
import { shellKindFor } from '../shell-loader';

describe('shellKindFor', () => {
	it('桌面与平板按平台分派（方向无关）', () => {
		expect(shellKindFor('desktop', 'portrait')).toBe('desktop');
		expect(shellKindFor('desktop', 'landscape')).toBe('desktop');
		expect(shellKindFor('tablet', 'portrait')).toBe('tablet');
		expect(shellKindFor('tablet', 'landscape')).toBe('tablet');
	});

	it('移动端按方向分派竖屏/横屏壳', () => {
		expect(shellKindFor('mobile', 'portrait')).toBe('portrait');
		expect(shellKindFor('mobile', 'landscape')).toBe('landscape');
	});
});
