// 导航意图预取（R16 性能整改）：把「点击后才发现要下载/编译路由模块」提前到
// hover / focus / touchstart 的意图阶段，真实导航只剩 DOM 提交。
//
// 为什么不用 SvelteKit 内置的 <a data-sveltekit-preload-data="hover">：
// 内置预取只挂在 `<a href>` 上；本项目桌面侧栏/顶栏导航是 `<button>` + `goto()`
//（SidebarNavItem / ActionIconButton / LocationCard），所以 app.html 上已有的
// data-sveltekit-preload-data 覆盖不到它们，必须在 DesktopLayout 里做事件委托。
//
// 为什么用 preloadCode 而不是 preloadData：本应用 ssr=false 且没有会取数的路由
// load 函数（仅旧链接 308 重定向的 +page.ts），预取数据没有收益；preloadCode
// 不执行 load 函数，因此对真实后端是零请求副作用（settings、notify 等
// 页面的数据仍在真实挂载后的 $effect 里拉取，时序不变）。
import { preloadCode } from '$app/navigation';

/** 已发起预取的路径集合：并发/重复意图只触发一次，避免重复网络与 transform。 */
const preloaded = new Set<string>();

/** 规范化导航路径：去掉 query/hash，要求以 / 开头；不可用的 href 返回 null。 */
export function normalizeNavPath(href: string | null | undefined): string | null {
	const raw = (href ?? '').trim();
	if (!raw) return null;
	const path = raw.split('?')[0].split('#')[0];
	if (!path.startsWith('/')) return null;
	return path;
}

/** 测试辅助：清空去重集合。 */
export function resetPreloadedRoutes(): void {
	preloaded.clear();
}

/**
 * 预取单个路由的代码。
 * - Set 去重：同一路径只调用一次 runner（历史/并发意图不会重复请求）；
 * - 当前所在路径跳过（无需预取，SvelteKit 也会直接命中内存模块）；
 * - 失败静默：预取只是加速手段，绝不能影响真实导航（goto 会自行加载）。
 * runner 可注入，单测无需触碰真实 SvelteKit 运行时。
 */
export function preloadRoute(
	pathname: string,
	runner: (path: string) => Promise<void> = preloadCode
): Promise<void> {
	const path = normalizeNavPath(pathname);
	if (!path) return Promise.resolve();
	if (typeof window !== 'undefined' && window.location.pathname === path) return Promise.resolve();
	if (preloaded.has(path)) return Promise.resolve();
	preloaded.add(path);
	return Promise.resolve(runner(path)).catch(() => {
		/* 预取失败静默：真实点击导航会重新加载，不影响可用性 */
	});
}

/**
 * 从事件 target 解析导航入口路径：向上找到最近的按钮/链接，按
 * aria-label → title → 可见文本 依次查「标签 → 路径」映射表。
 * 侧栏项用 aria-label（SidebarNavItem），顶栏图标按钮用 aria-label（ActionIconButton），
 * 位置卡用 aria-label=位置选择——都不需要改被委托的子组件。
 */
export function resolvePreloadTarget(
	target: EventTarget | null,
	labels: ReadonlyMap<string, string>
): string | null {
	const el = target as HTMLElement | null;
	if (!el || typeof el.closest !== 'function') return null;
	const entry = el.closest<HTMLElement>('button, a[href], [role="button"]');
	if (!entry) return null;
	const candidates = [entry.getAttribute('aria-label'), entry.getAttribute('title'), entry.textContent];
	for (const candidate of candidates) {
		const key = candidate?.trim();
		if (!key) continue;
		const path = labels.get(key);
		if (path) return path;
	}
	return null;
}

export interface IdlePreloadOptions {
	/** 页面加载后延迟多久再挑空闲时间预热（默认 5000ms：先让首屏与数据请求占用带宽）。 */
	delayMs?: number;
	/** requestIdleCallback 的兜底超时（默认 3000ms）。 */
	timeoutMs?: number;
}

/**
 * requestIdleCallback 兜底预热常用路由：给没有 hover 意图的用户（触屏点按、键盘、
 * 或直接点导航但鼠标未经过目标）也留一条加速路径。失败静默由 preloadRoute 保证。
 * - 省流量（saveData）或页面已隐藏时跳过；
 * - 返回取消函数（组件卸载时清理定时器）。
 */
export function scheduleIdlePreload(
	routes: readonly string[],
	options: IdlePreloadOptions = {}
): () => void {
	if (typeof window === 'undefined') return () => {};
	const { delayMs = 5000, timeoutMs = 3000 } = options;
	let cancelled = false;
	const run = () => {
		if (cancelled || document.visibilityState === 'hidden') return;
		const connection = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection;
		if (connection?.saveData) return;
		for (const route of routes) void preloadRoute(route);
	};
	const timer = window.setTimeout(() => {
		const idle = (
			window as Window & {
				requestIdleCallback?: (cb: () => void, opts?: { timeout?: number }) => number;
			}
		).requestIdleCallback;
		if (typeof idle === 'function') idle(run, { timeout: timeoutMs });
		else run();
	}, delayMs);
	return () => {
		cancelled = true;
		window.clearTimeout(timer);
	};
}

/** 当前是否已在浏览器运行（SSR/jsdom 保护）。 */
export function canPreload(): boolean {
	return typeof window !== 'undefined';
}
