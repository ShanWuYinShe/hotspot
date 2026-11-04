<script lang="ts">
	// InfiniteSentinel — 内容型列表「滚到底自动续载」的唯一实现（components/list）。
	//
	// 用法：在每个内容列表尾部渲染本组件（配合页面的 hasMore / loadingMore / loadMore）。
	//   - 哨兵进入视口（rootMargin 预载 320px）且空闲时调用 onLoadMore；
	//   - 自动补屏：加载完成后哨兵仍在视口内（首屏内容不足一屏）→ 继续拉起下一页，
	//     直到填满视口或到底；由 loading/hasMore 状态变化驱动，不写轮询；
	//   - error 为真时停止自动加载，避免失败后反复重试打后端；页面渲染 LoadError 让用户手动重试；
	//   - 到底由页面渲染 ListEnd，本组件不输出任何可见文案；
	//   - 管理后台六模块等「显式分页」场景仍用 LoadMoreButton，不挂本组件。
	//
	// 无障碍：哨兵本身 aria-hidden（纯加载触发点），键盘 End/PageDown 滚到列表底部同样触发；
	// 不改变 Tab 顺序。

	let {
		hasMore,
		loading = false,
		error = false,
		onLoadMore,
		rootMargin = '320px'
	}: {
		/** 是否还有下一页（无下一页时组件不参与观察） */
		hasMore: boolean;
		/** 是否正在加载（加载中不重复触发） */
		loading?: boolean;
		/** 是否处于错误态（错误后停止自动加载，等用户手动重试） */
		error?: boolean;
		onLoadMore: () => void;
		/** 预载距离，缺省 320px（约一屏 1/3，滚动到接近底部前开始取数） */
		rootMargin?: string;
	} = $props();

	let sentinelEl = $state<HTMLElement | null>(null);
	/** 哨兵当前是否在视口（含 rootMargin 预载区）内 */
	let visible = $state(false);
	/** 节流：两次自动触发至少间隔 300ms，避免页面状态更新滞后时紧循环 */
	let lastTriggeredAt = Number.NEGATIVE_INFINITY;
	/** 上一帧 loading 值：一次加载结束（true→false）后解除节流，允许继续补屏 */
	let wasLoading = false;
	/** 安全上限：同一段可见期内最多连续补载 30 页（防御页面不更新 loading 的异常） */
	let firedInView = 0;

	$effect(() => {
		const node = sentinelEl;
		const more = hasMore;
		if (!node || !more) {
			visible = false;
			return;
		}
		firedInView = 0;
		const io = new IntersectionObserver(
			([entry]) => {
				visible = Boolean(entry?.isIntersecting);
				if (!visible) firedInView = 0;
			},
			{ rootMargin }
		);
		io.observe(node);
		return () => {
			io.disconnect();
			visible = false;
			firedInView = 0;
		};
	});

	$effect(() => {
		// 一次加载结束（loading true→false）后解除节流：即使两页间隔 <300ms 也允许继续补屏；
		// 节流只兜底「页面从不进入 loading」的异常循环。
		const justFinished = wasLoading && !loading;
		wasLoading = loading;
		if (justFinished) lastTriggeredAt = Number.NEGATIVE_INFINITY;

		if (!visible || loading || error || !hasMore) return;
		if (firedInView >= 30) return;
		const now = Date.now();
		if (now - lastTriggeredAt < 300) return;
		lastTriggeredAt = now;
		firedInView += 1;
		const id = setTimeout(() => onLoadMore(), 0);
		return () => clearTimeout(id);
	});
</script>

<div bind:this={sentinelEl} data-infinite-sentinel aria-hidden="true" class="h-px w-full"></div>
