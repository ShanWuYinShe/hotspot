// AnnouncementCards 契约测试：公告渲染、severity 标签、action_url 协议白名单。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import AnnouncementCards from '../AnnouncementCards.svelte';
import type { SystemAnnouncementItem } from '@/api/types';

const base: SystemAnnouncementItem = {
	announcement_id: 'ann_1',
	title: '社区公约更新',
	content: '分享链接统一改为可撤销形式。',
	severity: 'warning',
	action_url: 'https://example.com/rules',
	created_at: new Date().toISOString()
};

describe('AnnouncementCards', () => {
	it('渲染公告标题/正文与 severity 标签', () => {
		render(AnnouncementCards, { props: { announcements: [base] } });
		expect(screen.getByText('平台公告')).toBeTruthy();
		expect(screen.getByText('社区公约更新')).toBeTruthy();
		expect(screen.getByText('分享链接统一改为可撤销形式。')).toBeTruthy();
		expect(screen.getByText('提醒')).toBeTruthy();
	});

	it('https action_url 渲染「查看详情」，非 http(s) 协议不渲染', () => {
		const { unmount } = render(AnnouncementCards, { props: { announcements: [base] } });
		expect(screen.getByText('查看详情')).toBeTruthy();
		unmount();

		render(AnnouncementCards, {
			props: { announcements: [{ ...base, action_url: 'javascript:alert(1)' }] }
		});
		expect(screen.queryByText('查看详情')).toBeNull();
	});

	it('无公告时不渲染分区', () => {
		const { container } = render(AnnouncementCards, { props: { announcements: [] } });
		expect(container.textContent?.trim()).toBe('');
	});
});
