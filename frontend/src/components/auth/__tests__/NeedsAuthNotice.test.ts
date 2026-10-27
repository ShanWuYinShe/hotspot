// NeedsAuthNotice 契约测试：访客引导面板的标题/描述/价值点/主次 CTA 与登录跳转。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { MapPin } from '@lucide/svelte';
import { goto } from '$app/navigation';
import NeedsAuthNotice from '../NeedsAuthNotice.svelte';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

describe('NeedsAuthNotice', () => {
	it('渲染标题、描述与价值点', () => {
		render(NeedsAuthNotice, {
			props: {
				title: '登录后查看消息',
				description: '登录后即可继续聊天',
				highlights: [{ icon: MapPin, text: '查看附近的人' }]
			}
		});
		expect(screen.getByText('登录后查看消息')).toBeTruthy();
		expect(screen.getByText('登录后即可继续聊天')).toBeTruthy();
		expect(screen.getByText('查看附近的人')).toBeTruthy();
	});

	it('主 CTA 跳转登录页', async () => {
		render(NeedsAuthNotice, { props: { title: '登录后查看' } });
		await fireEvent.click(screen.getByRole('button', { name: /登录 \/ 注册/ }));
		expect(vi.mocked(goto)).toHaveBeenCalledWith('/login');
	});

	it('次 CTA 默认回首页，可被 onSecondary 覆盖', async () => {
		const onSecondary = vi.fn();
		render(NeedsAuthNotice, { props: { title: '登录后查看', onSecondary } });
		await fireEvent.click(screen.getByRole('button', { name: /先随便逛逛/ }));
		expect(onSecondary).toHaveBeenCalledTimes(1);
	});
});
