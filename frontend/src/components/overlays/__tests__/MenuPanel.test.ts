// MenuPanel 契约测试：标题/元信息渲染、选项点击回调、选中态、底部动作。
// leading/footer 的 snippet 形态不在 jsdom 覆盖（snippet 传参不稳定），只断言数据驱动部分。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import MenuPanel from '../MenuPanel.svelte';

const options = [
	{ id: 'amap', title: '高德', desc: '国内细节丰富（GCJ-02）', selected: true },
	{ id: 'osm', title: 'OpenStreetMap', desc: '全球通用（WGS-84）', selected: false }
];

describe('MenuPanel', () => {
	it('渲染标题与选项行（含副标题）', () => {
		render(MenuPanel, { props: { title: '地图源', options, onSelect: () => {} } });
		expect(screen.getByText('地图源')).toBeTruthy();
		expect(screen.getByText('高德')).toBeTruthy();
		expect(screen.getByText('国内细节丰富（GCJ-02）')).toBeTruthy();
		expect(screen.getByText('OpenStreetMap')).toBeTruthy();
	});

	it('有 meta 时标题走栏形态并显示元信息', () => {
		render(MenuPanel, { props: { title: '显示类型', meta: '2/3 已选', options, onSelect: () => {} } });
		expect(screen.getByText('2/3 已选')).toBeTruthy();
	});

	it('点击选项行以 id 回调 onSelect', async () => {
		const onSelect = vi.fn();
		render(MenuPanel, { props: { title: '地图源', options, onSelect } });
		await fireEvent.click(screen.getByRole('menuitemradio', { name: /OpenStreetMap/ }));
		expect(onSelect).toHaveBeenCalledTimes(1);
		expect(onSelect).toHaveBeenCalledWith('osm');
	});

	it('multi 模式行 role 为 menuitemcheckbox，选中态经 aria-checked 表达', () => {
		render(MenuPanel, { props: { title: '显示类型', multi: true, options, onSelect: () => {} } });
		const boxes = screen.getAllByRole('menuitemcheckbox');
		expect(boxes).toHaveLength(2);
		expect(boxes[0].getAttribute('aria-checked')).toBe('true');
		expect(boxes[1].getAttribute('aria-checked')).toBe('false');
	});

	it('底部动作渲染并可点击', async () => {
		const onAction = vi.fn();
		render(MenuPanel, {
			props: {
				title: '显示类型',
				options,
				onSelect: () => {},
				footer: { actionLabel: '全部显示', onAction }
			}
		});
		await fireEvent.click(screen.getByRole('button', { name: '全部显示' }));
		expect(onAction).toHaveBeenCalledTimes(1);
	});

	it('底部动作禁用时挂 disabled（真机点击到不了 onAction）', () => {
		render(MenuPanel, {
			props: {
				title: '显示类型',
				options,
				onSelect: () => {},
				footer: { actionLabel: '全部显示', actionDisabled: true, onAction: () => {} }
			}
		});
		// jsdom 合成事件绕过 disabled 的默认拦截，这里只断言属性，真机行为由原生语义保证。
		expect((screen.getByRole('button', { name: '全部显示' }) as HTMLButtonElement).disabled).toBe(true);
	});
});
