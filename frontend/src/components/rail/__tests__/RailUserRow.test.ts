// RailUserRow 契约测试：头像/名称/副标渲染与点击回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import RailUserRow from '../RailUserRow.svelte';

describe('RailUserRow', () => {
	it('渲染头像 alt、名称与副标', () => {
		render(RailUserRow, {
			props: { avatar: 'https://example.com/a.jpg', name: '鹿鸣', sub: '0.4km' }
		});
		expect(screen.getByAltText('鹿鸣')).toBeTruthy();
		expect(screen.getByText('鹿鸣')).toBeTruthy();
		expect(screen.getByText('0.4km')).toBeTruthy();
	});

	it('无 sub 时不渲染副标', () => {
		render(RailUserRow, { props: { avatar: 'x', name: '鹿鸣' } });
		expect(screen.queryByText('0.4km')).toBeNull();
	});

	it('点击触发 onClick', async () => {
		const onClick = vi.fn();
		render(RailUserRow, {
			props: { avatar: 'x', name: '鹿鸣', onClick }
		});
		await fireEvent.click(screen.getByRole('button'));
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	// 回归守卫：hover 底色必须长在可点元素本体上。曾挂在行容器上，导致 trailing 动作
	//（动态详情「作者」行的「私信」pill）被高亮吞掉，形成圆角嵌套 + 同色叠加的脏效果。
	it('hover 底色归属于可点元素本体，不写在行容器上', () => {
		render(RailUserRow, { props: { avatar: 'x', name: '鹿鸣', onClick: vi.fn() } });
		const btn = screen.getByRole('button');
		expect(btn.className).toContain('hover-bg');
		expect(btn.className).toContain('cursor-pointer');
		expect(btn.parentElement?.className ?? '').not.toContain('hover-bg');
	});

	it('不可点击的行不出现任何 hover 底色', () => {
		render(RailUserRow, { props: { avatar: 'x', name: '鹿鸣' } });
		const btn = screen.getByRole('button');
		expect(btn.className).not.toContain('hover-bg');
		expect(btn.parentElement?.className ?? '').not.toContain('hover-bg');
	});
});
