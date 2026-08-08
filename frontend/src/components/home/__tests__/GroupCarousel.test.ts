// GroupCarousel 契约测试：群组卡渲染与「更多」回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import GroupCarousel from '../GroupCarousel.svelte';

const groups = [
	{ id: 'g1', name: '云龙湖晨跑团', members: 89, banner: 'https://example.com/b.jpg', distance: '0.8km', privacy: 'public' as const },
	{ id: 'g2', name: '私密小组', members: 12, banner: 'https://example.com/c.jpg', distance: '1.2km', privacy: 'private' as const }
];

describe('GroupCarousel', () => {
	it('渲染群组卡信息', () => {
		render(GroupCarousel, { props: { groups, onSeeAll: () => {} } });
		expect(screen.getByText('云龙湖晨跑团')).toBeTruthy();
		expect(screen.getByText('89 成员')).toBeTruthy();
		expect(screen.getByText('0.8km')).toBeTruthy();
		expect(screen.getByAltText('云龙湖晨跑团')).toBeTruthy();
	});

	it('点击「更多」触发 onSeeAll', async () => {
		const onSeeAll = vi.fn();
		render(GroupCarousel, { props: { groups, onSeeAll } });
		await fireEvent.click(screen.getByRole('button', { name: /更多/ }));
		expect(onSeeAll).toHaveBeenCalledTimes(1);
	});
});
