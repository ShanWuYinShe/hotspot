import '@testing-library/jest-dom/vitest';

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
