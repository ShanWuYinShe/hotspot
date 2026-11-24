// 我的举报页（/me/reports）契约测试：
//   - 列表渲染（目标类型/原因/状态胶囊/处理结果/ListEnd）
//   - 空态（EmptyState）、失败态（LoadError + 重试）
//   - status 过滤参数与 ?tab= 深链一致（all → null；其余 tab → 对应状态）
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte';

const api = vi.hoisted(() => ({ listReports: vi.fn() }));
const state = vi.hoisted(() => ({ url: new URL('http://x/me/reports') }));

vi.mock('@/api/client', () => ({ listReports: api.listReports }));
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		get url() {
			return state.url;
		}
	}
}));

import ReportsPage from '../ReportsPage.svelte';
import type { ReportItem } from '@/api/types';

function item(over: Partial<ReportItem> = {}): ReportItem {
	return {
		report_id: 'rep_1',
		target_type: 'moment',
		target_id: 'moment_1',
		reason: 'spam',
		description: '疑似营销刷屏',
		status: 'pending',
		created_at: '2026-09-08T10:00:00Z',
		handled_at: null,
		message_snapshot: null,
		resolution: null,
		...over
	};
}

const ok = (items: ReportItem[], next_cursor: string | null = null) => ({
	code: 0,
	message: 'ok',
	data: { items, next_cursor }
});

beforeEach(() => {
	vi.clearAllMocks();
	state.url = new URL('http://x/me/reports');
});

describe('我的举报页', () => {
	it('渲染记录的类型/原因/状态与处理结果，并在无更多数据时收尾', async () => {
		api.listReports.mockResolvedValue(
			ok([
				item(),
				item({
					report_id: 'rep_2',
					target_type: 'user',
					reason: 'harassment',
					description: '私信骚扰',
					status: 'resolved',
					resolution: { action: 'resolve', resolved_at: '2026-09-01T12:00:00Z', note: '确认垃圾营销' }
				})
			])
		);

		const { container } = render(ReportsPage);

		await waitFor(() => expect(screen.getByText('举报瞬间 · 垃圾信息')).toBeTruthy());
		expect(screen.getByText('举报用户 · 骚扰')).toBeTruthy();
		expect(screen.getByText('疑似营销刷屏')).toBeTruthy();
		// 状态胶囊文案与筛选 Tab 同名，故用 getAllByText
		expect(screen.getAllByText('待处理').length).toBeGreaterThan(1);
		// R13：处理结果拆成「标签 + 加粗结论 + 备注」三段（层级更清楚）；
		// 文本被拆到多个节点，testing-library 的 getByText 只读直接文本节点，
		// 故按 <p> 的 textContent 聚合断言（不绑死节点结构）。
		const resolution = [...container.querySelectorAll('p')].find((el) =>
			(el.textContent ?? '').includes('处理结果')
		);
		expect(resolution?.textContent).toContain('已确认处理');
		expect(resolution?.textContent).toContain('确认垃圾营销');
		expect(screen.getByText('没有更多了')).toBeTruthy();
	});

	it('空列表渲染空态而非空白', async () => {
		api.listReports.mockResolvedValue(ok([]));
		render(ReportsPage);
		await waitFor(() => expect(screen.getByText('暂无举报记录')).toBeTruthy());
	});

	it('接口失败渲染失败态与重试入口，不静默转空态', async () => {
		api.listReports.mockResolvedValue({ code: 10600, message: '服务暂时不可用', data: null });
		render(ReportsPage);
		await waitFor(() => expect(screen.getByText('加载失败')).toBeTruthy());
		expect(screen.getByText('服务暂时不可用')).toBeTruthy();
		expect(screen.getByText('重试')).toBeTruthy();
		expect(screen.queryByText('暂无举报记录')).toBeNull();
	});

	it('搜索时先进入加载态，完成后按关键词本地过滤并显示计数', async () => {
		api.listReports.mockResolvedValueOnce(
			ok([
				item({ report_id: 'rep_1', description: '疑似营销刷屏' }),
				item({
					report_id: 'rep_2',
					target_type: 'message',
					reason: 'harassment',
					description: '私聊骚扰',
					status: 'resolved'
				})
			])
		);
		render(ReportsPage);
		await waitFor(() => expect(screen.getByText('举报瞬间 · 垃圾信息')).toBeTruthy());

		let resolveSearch!: (value: unknown) => void;
		api.listReports.mockImplementationOnce(
			() =>
				new Promise((resolve) => {
					resolveSearch = resolve;
				})
		);

		const input = screen.getByLabelText('搜索我的举报');
		await fireEvent.input(input, { target: { value: '私聊' } });

		// 搜索尚未返回时不得提前渲染「没有匹配」空态
		await waitFor(() => expect(screen.getByText('正在搜索举报记录…')).toBeTruthy());
		expect(screen.queryByText('没有匹配的举报')).toBeNull();

		resolveSearch(
			ok([
				item({ report_id: 'rep_1', description: '疑似营销刷屏' }),
				item({
					report_id: 'rep_2',
					target_type: 'message',
					reason: 'harassment',
					description: '私聊骚扰',
					status: 'resolved'
				})
			])
		);

		await waitFor(() => expect(screen.getByText('已加载 2 条 · 匹配 1 条')).toBeTruthy());
		expect(screen.getByText('举报消息 · 骚扰')).toBeTruthy();
		expect(screen.queryByText('举报瞬间 · 垃圾信息')).toBeNull();
	});

	it('搜索失败在内容区展示失败态与重试，不显示无匹配空态', async () => {
		api.listReports.mockResolvedValueOnce(ok([item()]));
		render(ReportsPage);
		await waitFor(() => expect(screen.getByText('举报瞬间 · 垃圾信息')).toBeTruthy());

		api.listReports.mockResolvedValueOnce({ code: 10600, message: '服务暂时不可用', data: null });
		await fireEvent.input(screen.getByLabelText('搜索我的举报'), { target: { value: '私聊' } });

		await waitFor(() => expect(screen.getByText('服务暂时不可用')).toBeTruthy());
		expect(screen.getByText('加载失败')).toBeTruthy();
		expect(screen.queryByText('没有匹配的举报')).toBeNull();

		api.listReports.mockResolvedValueOnce(
			ok([item({ report_id: 'rep_2', target_type: 'message', reason: 'harassment', description: '私聊骚扰' })])
		);
		await fireEvent.click(screen.getByText('重试'));

		await waitFor(() => expect(screen.getByText('已加载 1 条 · 匹配 1 条')).toBeTruthy());
		expect(screen.getByText('举报消息 · 骚扰')).toBeTruthy();
	});

	it('搜索遇到 10100 展示登录引导而非技术错误', async () => {
		api.listReports.mockResolvedValueOnce(ok([item()]));
		render(ReportsPage);
		await waitFor(() => expect(screen.getByText('举报瞬间 · 垃圾信息')).toBeTruthy());

		api.listReports.mockResolvedValueOnce({ code: 10100, message: 'Access Token 无效', data: null });
		await fireEvent.input(screen.getByLabelText('搜索我的举报'), { target: { value: '私聊' } });

		await waitFor(() => expect(screen.getByText('登录后查看')).toBeTruthy());
		expect(screen.queryByText(/Access Token/)).toBeNull();
	});

	it('?tab= 深链决定 status 过滤参数（all → null；pending → pending）', async () => {
		api.listReports.mockResolvedValue(ok([]));
		render(ReportsPage);
		await waitFor(() => expect(api.listReports).toHaveBeenCalled());
		expect(api.listReports).toHaveBeenCalledWith(expect.objectContaining({ status: null }));

		api.listReports.mockClear();
		state.url = new URL('http://x/me/reports?tab=pending');
		render(ReportsPage);
		await waitFor(() => expect(api.listReports).toHaveBeenCalled());
		expect(api.listReports).toHaveBeenCalledWith(expect.objectContaining({ status: 'pending' }));
	});
});