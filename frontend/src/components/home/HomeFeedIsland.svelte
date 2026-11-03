<script lang="ts">
	import { ArrowLeftRight, Sparkles, Users } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { homeFeed } from '@/stores/home-feed.svelte';

	// 首页灵动岛（桌面端，滚动 handoff）：
	// 页面顶部显示内容区分段器；分段器滚出视口后，顶栏中央浮现灵动岛，
	// 展示当前信息流 + 语境说明 + 切换动作——与详情页锚点岛同一种「信息交接」节奏，
	// 而不是始终挂着一排两个标签。
	const isDiscover = $derived(homeFeed.activeTab === 'discover');
	const ActiveIcon = $derived(isDiscover ? Sparkles : Users);
	const label = $derived(isDiscover ? '为你推荐' : '正在关注');
	const meta = $derived(
		isDiscover
			? '根据你的兴趣推荐'
			: auth.status === 'authenticated'
				? '关注的人的最新动态'
				: '登录后查看关注动态'
	);
	const otherLabel = $derived(isDiscover ? '正在关注' : '为你推荐');

	function scrollToTop() {
		const scroller = document.getElementById('main-content');
		const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
		scroller?.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' });
	}

	function switchFeed() {
		homeFeed.setTab(isDiscover ? 'following' : 'discover');
		// 切换后回到信息流顶部：内容分区同步、岛随即交还分段器（自然 handoff）。
		requestAnimationFrame(scrollToTop);
	}
</script>

<div
	data-desktop-dynamic-island
	class="pointer-events-auto flex h-9 min-w-0 max-w-md items-center gap-1.5 rounded-full border border-border-default/50 bg-surface-primary/90 px-2 text-text-primary shadow-raised-sm backdrop-blur-xl animate-in fade-in zoom-in-95"
	style="contain: layout paint;"
>
	<span class="flex size-7 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent-default" aria-hidden="true">
		<ActiveIcon class="size-3.5" strokeWidth={2} />
	</span>
	<button
		type="button"
		class="hover-btn focus-ring flex min-w-0 flex-1 items-center rounded-full px-1.5 py-0.5 text-left"
		onclick={scrollToTop}
		aria-label="回到信息流顶部"
	>
		<span class="flex min-w-0 flex-col justify-center">
			<span class="truncate text-xs font-semibold leading-tight text-text-primary">{label}</span>
			<span class="truncate text-xxs leading-tight text-text-secondary">{meta}</span>
		</span>
	</button>
	<div class="h-3.5 w-px shrink-0 bg-border-default/40" aria-hidden="true"></div>
	<ActionIconButton
		icon={ArrowLeftRight}
		label={`切换到${otherLabel}`}
		onClick={switchFeed}
		variant="ghost"
		tone="muted"
		size={32}
		class="shrink-0"
	/>
</div>
