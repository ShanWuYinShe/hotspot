// CreateGroupSheet 契约测试：位置缺失阻止提交、成功调用 createGroup 并回调。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import CreateGroupSheet from '../CreateGroupSheet.svelte';
import { createGroup } from '@/api';
import { loadDefaultLocation } from '@/lib/default-location';
import { startDefaultLocation } from '@/lib/start-default-location';

vi.mock('@/api', () => ({ createGroup: vi.fn() }));
vi.mock('@/lib/default-location', () => ({ loadDefaultLocation: vi.fn() }));
vi.mock('@/lib/start-default-location', () => ({ startDefaultLocation: vi.fn(() => () => {}) }));

const LOCATION = { point: { latitude: 31.2, longitude: 121.5 }, location_name: '上海' };

describe('CreateGroupSheet', () => {
	beforeEach(() => {
		vi.mocked(createGroup).mockReset();
		vi.mocked(loadDefaultLocation).mockReset();
		vi.mocked(startDefaultLocation).mockReturnValue(() => {});
		vi.mocked(createGroup).mockResolvedValue({ code: 0, data: { group: { group_id: 'g1' } } } as never);
	});

	it('拿不到位置时阻止提交并提示', async () => {
		vi.mocked(loadDefaultLocation).mockRejectedValue(new Error('no location'));
		render(CreateGroupSheet, { props: { onClose: () => {} } });
		await fireEvent.input(screen.getByPlaceholderText('给你的群组起个名字…'), { target: { value: '周末骑行' } });
		await fireEvent.click(screen.getByRole('button', { name: '创建群组' }));
		expect(await screen.findByText('无法获取当前位置，请稍后重试')).toBeTruthy();
		expect(createGroup).not.toHaveBeenCalled();
	});

	it('创建成功调用 createGroup 并回调 groupId', async () => {
		vi.mocked(loadDefaultLocation).mockResolvedValue(LOCATION);
		const onCreated = vi.fn();
		render(CreateGroupSheet, { props: { onClose: () => {}, onCreated } });
		await fireEvent.input(screen.getByPlaceholderText('给你的群组起个名字…'), { target: { value: '周末骑行' } });
		await fireEvent.click(screen.getByRole('button', { name: '创建群组' }));
		await waitFor(() => {
			expect(createGroup).toHaveBeenCalledWith(
				expect.objectContaining({ name: '周末骑行', location: LOCATION })
			);
		});
		await waitFor(() => expect(onCreated).toHaveBeenCalledWith('g1'));
	});
});
