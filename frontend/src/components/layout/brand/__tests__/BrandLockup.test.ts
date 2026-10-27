// BrandLockup 契约：三档尺寸都渲染品牌字名，Logo 为装饰（不重复播报）。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import BrandLockup from '../BrandLockup.svelte';

describe('BrandLockup', () => {
	it('渲染 HotSpot 字名与装饰性 Logo', () => {
		const { container } = render(BrandLockup);
		expect(screen.getByText('HotSpot')).toBeTruthy();
		const img = container.querySelector('img');
		expect(img).not.toBeNull();
		expect(img?.getAttribute('alt')).toBe('');
		expect(img?.getAttribute('aria-hidden')).toBe('true');
	});

	it('尺寸档映射到不同字标字号', () => {
		const { container } = render(BrandLockup, { props: { size: 'lg' } });
		expect(container.querySelector('.text-2xl')).not.toBeNull();
	});
});
