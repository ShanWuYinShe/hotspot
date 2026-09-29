// useGroupDetail 加入/退出刷新契约：成功后重拉详情（人数）与名单。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';
import GroupDetailHarness from './GroupDetailHarness.svelte';
import type { useGroupDetail } from '../group-data.svelte';

vi.mock('@/api', () => ({
	getGroupDetail: vi.fn(),
	getGroupMembers: vi.fn(),
	getGroupAnnouncements: vi.fn(),
	joinGroup: vi.fn(),
	leaveGroup: vi.fn(),
	createGroupAnnouncement: vi.fn(),
	updateGroupAnnouncement: vi.fn(),
	deleteGroupAnnouncement: vi.fn(),
	listSentGroupInvites: vi.fn(),
	revokeSentGroupInvite: vi.fn(),
	listBannedGroupMembers: vi.fn(),
	unbanGroupMember: vi.fn(),
	updateGroup: vi.fn(),
	listGroupApplications: vi.fn(),
	approveGroupApplication: vi.fn(),
	rejectGroupApplication: vi.fn(),
	transferGroupOwnership: vi.fn(),
	dismissGroup: vi.fn(),
	listMyApplications: vi.fn(),
	cancelGroupApplication: vi.fn(),
	listAdminGroupMembers: vi.fn(),
	netError: vi.fn((data?: unknown) => ({ code: -1, data: data ?? null }))
}));

vi.mock('@/lib/ws/client', () => ({
	wsClient: { setSubscriptions: vi.fn(), on: vi.fn(() => () => {}) }
}));

import * as api from '@/api';

type Gd = ReturnType<typeof useGroupDetail>;

// 有状态 mock：服务端成员态随 join/leave 翻转，详情快照如实返回最新值。
let isMember = false;

function mockServer() {
	vi.mocked(api.getGroupDetail).mockImplementation(async () => ({
		code: 0,
		data: {
			group: { group_id: 'g1', name: '测试群', members_count: isMember ? 3 : 2 },
			viewer_state: isMember ? { role: 'member' } : null,
			tags: [],
			online_members_count: 0
		}
	} as never));
	vi.mocked(api.getGroupMembers).mockResolvedValue({ code: 0, data: { items: [], next_cursor: null } } as never);
	vi.mocked(api.getGroupAnnouncements).mockResolvedValue({ code: 0, data: { items: [] } } as never);
	vi.mocked(api.listSentGroupInvites).mockResolvedValue({ code: 0, data: { items: [] } } as never);
	vi.mocked(api.listBannedGroupMembers).mockResolvedValue({ code: 0, data: { items: [] } } as never);
	vi.mocked(api.listGroupApplications).mockResolvedValue({ code: 0, data: { items: [] } } as never);
	vi.mocked(api.joinGroup).mockImplementation(async (gid: string) => {
		if (gid === 'g1') isMember = true;
		return { code: 0, data: {} } as never;
	});
	vi.mocked(api.leaveGroup).mockImplementation(async (gid: string) => {
		if (gid === 'g1') isMember = false;
		return { code: 0, data: {} } as never;
	});
}

async function mount(): Promise<Gd> {
	let gd: Gd | null = null;
	render(GroupDetailHarness, { props: { onReady: (g: Gd) => {
		gd = g;
	} } });
	await waitFor(() => expect(gd).not.toBeNull());
	// 等初始 loadAll 落地。
	await waitFor(() => expect(vi.mocked(api.getGroupDetail)).toHaveBeenCalled());
	await waitFor(() => expect(gd as unknown as Gd).toBeTruthy());
	return gd as unknown as Gd;
}

beforeEach(() => {
	vi.clearAllMocks();
	isMember = false;
	mockServer();
});

describe('useGroupDetail toggleJoin', () => {
	it('加入成功后人数即时更新（免手动刷新）', async () => {
		const gd = await mount();
		await waitFor(() => expect(gd.detail.members).toBe(2));
		const detailCalls = vi.mocked(api.getGroupDetail).mock.calls.length;
		const memberCalls = vi.mocked(api.getGroupMembers).mock.calls.length;
		gd.toggleJoin();
		// 乐观更新先行，服务端刷新随后落地并保持一致。
		await waitFor(() => expect(gd.joined).toBe(true));
		await waitFor(() => expect(gd.detail.members).toBe(3));
		expect(vi.mocked(api.getGroupDetail).mock.calls.length).toBeGreaterThan(detailCalls);
		expect(vi.mocked(api.getGroupMembers).mock.calls.length).toBeGreaterThan(memberCalls);
	});

	it('退出成功后人数即时回落', async () => {
		isMember = true;
		const gd = await mount();
		await waitFor(() => expect(gd.joined).toBe(true));
		await waitFor(() => expect(gd.detail.members).toBe(3));
		gd.toggleJoin();
		await waitFor(() => expect(gd.joined).toBe(false));
		await waitFor(() => expect(gd.detail.members).toBe(2));
	});

	it('待审核群不刷新（尚未成为成员，人数无变化）', async () => {
		vi.mocked(api.joinGroup).mockResolvedValue({ code: 0, data: { status: 'pending_approval', application_id: 'a1' } });
		const gd = await mount();
		await waitFor(() => expect(gd.detail.members).toBe(2));
		const detailCalls = vi.mocked(api.getGroupDetail).mock.calls.length;
		gd.toggleJoin();
		await waitFor(() => expect(gd.joinPendingApproval).toBe(true));
		// 让出事件循环：若实现误刷新，调用数会在此期间增长。
		await new Promise((r) => setTimeout(r, 30));
		expect(vi.mocked(api.getGroupDetail).mock.calls.length).toBe(detailCalls);
		expect(gd.detail.members).toBe(2);
	});
});
