import { afterAll, vi } from 'vitest';
import '@testing-library/jest-dom/vitest';

// bits-ui 的 dialog/dropdown-menu 等在释放 body-scroll-lock 时会安排约 24ms 的延迟
// 清理定时器；若它在本测试文件结束、jsdom 环境拆除之后才触发，回调访问 document
// 会抛 ReferenceError: document is not defined，并让整个 vitest 运行失败（v1.7.0
// 发布时 CI 连续两次复现，触发文件为 MomentActionMenu 用例）。统一在每个测试文件
// 收尾前等一拍：先恢复真实定时器（避免用例遗留的假定时器让等待永不返回），60ms
// 远大于 24ms，等待期间 jsdom 仍存活，清理可正常落地。
afterAll(async () => {
	vi.useRealTimers();
	await new Promise((resolve) => setTimeout(resolve, 60));
});

// Node 25+ 的原生 Web Storage 全局会遮蔽 jsdom 的 localStorage/sessionStorage
//（vitest-dev/vitest#8757）。vite.config 通过 worker execArgv 传
// --no-experimental-webstorage 让 jsdom 拥有 storage；但 bun 等运行时下该参数
// 可能不生效，此处守卫式兜底：jsdom 未提供 storage 时安装内存实现。
if (typeof window !== 'undefined' && !window.localStorage) {
	function createMemoryStorage(): Storage {
		const map = new Map<string, string>();
		return {
			get length() {
				return map.size;
			},
			clear() {
				map.clear();
			},
			getItem(key) {
				return map.has(key) ? map.get(key)! : null;
			},
			key(index) {
				return [...map.keys()][index] ?? null;
			},
			removeItem(key) {
				map.delete(key);
			},
			setItem(key, value) {
				map.set(String(key), String(value));
			}
		} as Storage;
	}

	Object.defineProperty(window, 'localStorage', {
		value: createMemoryStorage(),
		configurable: true
	});
	Object.defineProperty(window, 'sessionStorage', {
		value: createMemoryStorage(),
		configurable: true
	});
	Object.defineProperty(globalThis, 'localStorage', {
		value: window.localStorage,
		configurable: true
	});
	Object.defineProperty(globalThis, 'sessionStorage', {
		value: window.sessionStorage,
		configurable: true
	});
}

// jsdom 未实现 Web Animations API：Svelte 5 的 transition 只要返回了 css 配置就会走
// element.animate（transform/opacity 交给合成器），缺它时任何带 transition 的组件
//（如顶栏灵动岛 IslandCapsule）在测试里一挂载就抛 TypeError。这里补一个「立即完成」的
// 空实现：测试只看最终 DOM/交互，不逐帧断言动画。
if (typeof Element !== 'undefined' && typeof Element.prototype.animate !== 'function') {
	Element.prototype.animate = function animateStub() {
		return {
			finished: Promise.resolve(),
			cancel() {},
			finish() {},
			play() {},
			pause() {},
			reverse() {},
			commitStyles() {},
			currentTime: 0,
			playState: 'finished',
			onfinish: null
		} as unknown as Animation;
	};
}

// jsdom 未实现 ResizeObserver：Svelte 的 bind:clientWidth（AdaptiveCardGrid）与卡片画布
// 在组件测试里会实例化它。测试环境只保证构造/方法存在，不触发回调（宽高由各用例自行断言）。
if (!('ResizeObserver' in globalThis)) {
	class TestResizeObserver {
		observe() {}
		unobserve() {}
		disconnect() {}
	}
	Object.defineProperty(globalThis, 'ResizeObserver', { value: TestResizeObserver, writable: true, configurable: true });
}
