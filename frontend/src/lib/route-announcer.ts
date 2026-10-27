// SPA 路由播报（Round 9）：导航后更新 #route-announcer（aria-live=polite/atomic），
// 让屏幕阅读器确认页面已切换。首屏由调用方（afterNavigate type==='enter'）跳过；
// 相同标题在短时间内的重复更新会被去重，避免连续触发噪音。
let lastMessage = '';
let lastAt = 0;

export function announceRoute(message: string): void {
	if (typeof document === 'undefined') return;
	const text = message.trim();
	if (!text) return;
	const now = Date.now();
	if (text === lastMessage && now - lastAt < 1500) return;
	lastMessage = text;
	lastAt = now;
	const el = document.getElementById('route-announcer');
	if (!el) return;
	// 先清空再写入，确保 live region 文本变化可被读屏捕获
	el.textContent = '';
	requestAnimationFrame(() => {
		el.textContent = text;
	});
}

/** 重置去重状态（测试用）。 */
export function resetRouteAnnouncer(): void {
	lastMessage = '';
	lastAt = 0;
}
