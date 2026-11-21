// centerOnPage — 桌面端「页面级空态」水平中心对齐整页（视口）中心。
//
// 为什么需要它：桌面外壳有左侧栏 + 分栏页右 rail，路由正文列在「侧栏右侧的可用宽」
// 内居中，其中心比整页中心右移（侧栏宽/2 左右）。空态按正文列居中时，视觉上会偏离
// 顶部灵动岛的整页中轴，左侧留白明显小于右侧。
//
// 为什么用测量而不是固定侧栏宽度：侧栏可折叠、分栏页有右 rail、断点与滚动条都会
// 改变正文列中心；用 getBoundingClientRect 实测当前偏移量，任何外壳形态都成立。
// 只做水平平移（transform），不改变布局、不吞点击：动画/滚动/吸顶行为不变。
//
// 使用范围：仅页面级空态（整页无内容）。分栏面板/卡片内空态不要使用。
// 测试环境（jsdom）可能没有 matchMedia / ResizeObserver：一律降级为无操作。

const DESKTOP_QUERY = '(min-width: 1024px)';

export function centerOnPage(node: HTMLElement, enabled: boolean = true) {
	if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
		return {};
	}

	let active = enabled;
	let raf = 0;
	const media = window.matchMedia(DESKTOP_QUERY);

	const apply = () => {
		if (!active || !media.matches) {
			node.style.transform = '';
			return;
		}
		// 先清空上一次平移再测量，避免 getBoundingClientRect 把旧 transform 叠进新偏移。
		node.style.transform = '';
		const rect = node.getBoundingClientRect();
		if (rect.width === 0) return;
		const shift = Math.round(window.innerWidth / 2 - (rect.left + rect.width / 2));
		node.style.transform = shift === 0 ? '' : `translateX(${shift}px)`;
	};

	const schedule = () => {
		if (raf) return;
		raf = requestAnimationFrame(() => {
			raf = 0;
			apply();
		});
	};

	schedule();
	const ResizeObserverImpl = typeof ResizeObserver === 'undefined' ? null : ResizeObserver;
	const resizeObserver = ResizeObserverImpl ? new ResizeObserverImpl(schedule) : null;
	resizeObserver?.observe(node);
	window.addEventListener('resize', schedule);
	media.addEventListener?.('change', schedule);

	return {
		update(next: boolean) {
			active = next;
			schedule();
		},
		destroy() {
			cancelAnimationFrame(raf);
			resizeObserver?.disconnect();
			window.removeEventListener('resize', schedule);
			media.removeEventListener?.('change', schedule);
			node.style.transform = '';
		}
	};
}
