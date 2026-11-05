import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { headerIsland } from './header-island.svelte';
import { subscribeHandoff } from '@/lib/island-handoff';

// PageIsland（原 header-island）契约：
//   - token 注册：只有当前持有者能 update / setHandoff / release（路由切换不串台）
//   - 交接进度 p 驱动 visible（p>0 挂载）与 settled（p>0.55 页头 inert）两个响应式状态
//   - register / release 都把交接归零：路由切换时旧岛直接消失，不再飞回上一页标题
//   - 内容模型：title + meta + 可选 author（头像/作者名片）+ trailing（行动区）

const A = Symbol('page-a');
const B = Symbol('page-b');

describe('headerIsland store', () => {
	beforeEach(() => {
		headerIsland.release(A);
		headerIsland.release(B);
	});

	it('初始为未注册、不可见', () => {
		expect(headerIsland.active).toBe(false);
		expect(headerIsland.visible).toBe(false);
		expect(headerIsland.settled).toBe(false);
		expect(headerIsland.title).toBe('');
		expect(headerIsland.meta).toBe('');
	});

	it('注册后写入内容并置 active', () => {
		headerIsland.register(A, { title: '附近的人', meta: '20 人 · 5km 内' });
		expect(headerIsland.active).toBe(true);
		expect(headerIsland.title).toBe('附近的人');
		expect(headerIsland.meta).toBe('20 人 · 5km 内');
	});

	it('支持作者名片字段（详情页）', () => {
		headerIsland.register(A, {
			title: '鹿鸣',
			author: { name: '鹿鸣', location: '云龙湖东岸', time: '10月29日 21:59' }
		});
		expect(headerIsland.author?.name).toBe('鹿鸣');
		expect(headerIsland.author?.location).toBe('云龙湖东岸');
	});

	it('setHandoff 只对当前持有者生效，并按 p 更新 visible / settled', () => {
		headerIsland.register(A, { title: 'A' });
		headerIsland.setHandoff(A, 0.3, null);
		expect(headerIsland.visible).toBe(true);
		expect(headerIsland.settled).toBe(false);

		// 非当前持有者的帧被忽略（旧页面的滚动不能推动新页面的岛）
		headerIsland.setHandoff(B, 1, null);
		expect(headerIsland.settled).toBe(false);

		headerIsland.setHandoff(A, 1, null);
		expect(headerIsland.settled).toBe(true);

		headerIsland.setHandoff(A, 0, null);
		expect(headerIsland.visible).toBe(false);
		expect(headerIsland.settled).toBe(false);
	});

	it('交接进度收敛到 0–1（越界与非法值归零/归一）', () => {
		headerIsland.register(A, { title: 'A' });
		headerIsland.setHandoff(A, 2.5, null);
		expect(headerIsland.settled).toBe(true);
		headerIsland.setHandoff(A, Number.NaN, null);
		expect(headerIsland.visible).toBe(false);
		headerIsland.setHandoff(A, -1, null);
		expect(headerIsland.visible).toBe(false);
	});

	it('update 只对当前持有者生效', () => {
		headerIsland.register(A, { title: '旧标题' });
		headerIsland.update(A, { title: '新标题', meta: '新 meta' });
		expect(headerIsland.title).toBe('新标题');
		expect(headerIsland.meta).toBe('新 meta');

		headerIsland.update(B, { title: '别人的标题' });
		expect(headerIsland.title).toBe('新标题');
	});

	it('新 token 注册后接管；旧 token release 不得清空新内容', () => {
		headerIsland.register(A, { title: 'A 页' });
		headerIsland.register(B, { title: 'B 页' });
		headerIsland.setHandoff(B, 1, null);
		expect(headerIsland.title).toBe('B 页');

		headerIsland.release(A);
		expect(headerIsland.active).toBe(true);
		expect(headerIsland.title).toBe('B 页');
		expect(headerIsland.visible).toBe(true);

		headerIsland.release(B);
		expect(headerIsland.active).toBe(false);
		expect(headerIsland.title).toBe('');
		expect(headerIsland.visible).toBe(false);
	});

	it('注册/释放会把交接归零（订阅者收到归零帧，路由切换不残留）', () => {
		const frames: number[] = [];
		const off = subscribeHandoff((frame) => frames.push(frame.p));
		try {
			headerIsland.register(A, { title: 'A' });
			headerIsland.setHandoff(A, 0.8, null);
			headerIsland.release(A);
		} finally {
			off();
		}
		expect(frames[0]).toBe(0);
		expect(frames).toContain(0.8);
		expect(frames[frames.length - 1]).toBe(0);
	});

	it('注册会重置可见性（新页面从未见状态开始）', () => {
		headerIsland.register(A, { title: 'A' });
		headerIsland.setHandoff(A, 1, null);
		headerIsland.release(A);
		headerIsland.register(B, { title: 'B' });
		expect(headerIsland.visible).toBe(false);
		expect(headerIsland.settled).toBe(false);
	});

	it('阅读进度：与 token 无关、收敛到 0–100、非法输入归零', () => {
		headerIsland.register(A, { title: 'A' });
		headerIsland.setProgress(42);
		expect(headerIsland.progress).toBe(42);

		// 首页不注册岛，但同样需要进度：不依赖 token
		headerIsland.release(A);
		headerIsland.setProgress(80);
		expect(headerIsland.progress).toBe(80);

		headerIsland.setProgress(180);
		expect(headerIsland.progress).toBe(100);
		headerIsland.setProgress(-20);
		expect(headerIsland.progress).toBe(0);
		headerIsland.setProgress(Number.NaN);
		expect(headerIsland.progress).toBe(0);
	});
});
