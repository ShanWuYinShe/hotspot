// Badge 原语契约测试：四档尺寸（sm/md/status/author）的几何与字号必须完整落到
// class 上——tailwind-merge 不认识项目自定义字号 `text-xxs`（10px）时会把它当颜色丢弃，
// 且 `text-*` 会移除先出现的 `leading-*`；两处配置/顺序一旦回退，徽章会继承父级字号
// （16px 时撑破 20/16px 高度）或丢掉单行行高。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Badge from '../Badge.svelte';

function renderBadge(size: 'sm' | 'md' | 'status' | 'author', tone: 'neutral' | 'accent' = 'neutral') {
	const { container } = render(Badge, { props: { size, tone } });
	return container.firstElementChild as HTMLElement;
}

describe('Badge 尺寸契约', () => {
	it('sm = 20px / text-xxs（自定义 10px 字号不被合并器丢弃）', () => {
		const el = renderBadge('sm');
		expect(el.className).toContain('h-5');
		expect(el.className).toContain('text-xxs');
		expect(el.className).toContain('leading-none');
	});

	it('md = 24px / text-xs', () => {
		const el = renderBadge('md');
		expect(el.className).toContain('h-6');
		expect(el.className).toContain('text-xs');
		expect(el.className).toContain('leading-none');
	});

	it('status = 20px 药丸 + 11px（举报状态/类型徽标）', () => {
		const el = renderBadge('status', 'accent');
		expect(el.className).toContain('h-5');
		expect(el.className).toContain('text-xs');
		expect(el.className).toContain('rounded-full');
		expect(el.className).toContain('leading-none');
		expect(el.className).toContain('bg-accent-soft');
	});

	it('author = 28×16 微徽章（h-4 / px-1 / r4 / 10px）', () => {
		const el = renderBadge('author', 'accent');
		expect(el.className).toContain('h-4');
		expect(el.className).toContain('px-1');
		expect(el.className).toContain('rounded-[4px]');
		expect(el.className).toContain('text-xxs');
		expect(el.className).toContain('leading-none');
		expect(el.className).not.toContain('rounded-full');
	});
});
