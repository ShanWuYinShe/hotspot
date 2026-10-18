// /share/[code] 落地页契约测试：解析成功跳转目标页、10300 显示过期文案。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/svelte';
import { goto } from '$app/navigation';
import SharePage from '../+page.svelte';
import { openShareDoor } from '@/api';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: { params: { code: 'ABC234567890' }, url: new URL('http://localhost/share/ABC234567890') }
}));
vi.mock('@/api', () => ({ openShareDoor: vi.fn() }));

describe('share 落地页', () => {
	beforeEach(() => {
		vi.mocked(goto).mockReset();
		vi.mocked(openShareDoor).mockReset();
	});

	it('user 分享解析成功后 replaceState 跳转到目标详情页', async () => {
		vi.mocked(openShareDoor).mockResolvedValue({
			code: 0,
			data: {
				share: { share_id: 's1', share_code: 'ABC234567890', share_type: 'user', creator: { user_id: 'u9', nickname: '张三' }, created_at: '', expires_at: null },
				payload: { shared_user_id: 'u9' },
				preview: null
			}
		} as never);
		render(SharePage, { props: { params: { code: 'ABC234567890' } } });
		await waitFor(() => {
			expect(openShareDoor).toHaveBeenCalledWith({ share_code: 'ABC234567890' });
			expect(goto).toHaveBeenCalledWith('/users/u9', { replaceState: true });
		});
	});

	it('分享过期（10300）展示过期文案且不跳转', async () => {
		vi.mocked(openShareDoor).mockResolvedValue({ code: 10300, message: '分享不存在' } as never);
		render(SharePage, { props: { params: { code: 'ABC234567890' } } });
		await waitFor(() => {
			expect(screen.getByText('分享不存在或已过期')).toBeTruthy();
		});
		expect(goto).not.toHaveBeenCalled();
	});

	// 回归守卫（走查 s3）：本路由独立于 (app) 分组，必须自备 main 地标；错误态主文案
	// 必须是 h1（原为 text-2xl 的 <p>，页面因此全站唯一「无 main 且无 h1」）。
	it('错误态具备 main#main-content 地标与 h1', async () => {
		vi.mocked(openShareDoor).mockResolvedValue({ code: 10300, message: '分享不存在' } as never);
		const { container } = render(SharePage, { props: { params: { code: 'ABC234567890' } } });
		await waitFor(() => expect(screen.getByText('分享不存在或已过期')).toBeTruthy());
		expect(container.querySelector('main#main-content')).toBeTruthy();
		expect(screen.getByRole('heading', { level: 1 }).textContent).toContain('分享不存在或已过期');
	});
});
