<script lang="ts">
	import { ArrowLeftRight, Sparkles, Users } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import IslandCapsule from '@/components/layout/page/IslandCapsule.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { homeFeed } from '@/stores/home-feed.svelte';
	import type { IslandRect } from '@/lib/island-morph';

	// 首页灵动岛（桌面端，滚动 handoff）：
	// 页面顶部显示内容区分段器；分段器滚出视口后，顶栏中央浮现灵动岛，
	// 展示当前信息流 + 语境说明 + 切换动作——与详情页锚点岛同一形态与交互语言（见 IslandCapsule）。
	// 首页不注册 headerIsland（没有页面标题），进度环同样消费 headerIsland.progress。
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

	let {
		progress = 0,
		getMorphSource
	}: { progress?: number; getMorphSource?: () => IslandRect | null } = $props();

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

<IslandCapsule
	ariaLabel="信息流锚点与切换"
	title={label}
	{meta}
	titleAria="回到信息流顶部"
	onTitleClick={scrollToTop}
	{progress}
	{getMorphSource}
>
	{#snippet leading()}
		<span
			class="flex size-7 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent-default"
			aria-hidden="true"
		>
			<ActiveIcon class="size-3.5" strokeWidth={2} />
		</span>
	{/snippet}
	{#snippet trailing()}
		<ActionIconButton
			icon={ArrowLeftRight}
			label={`切换到${otherLabel}`}
			onClick={switchFeed}
			variant="ghost"
			tone="muted"
			size={32}
		/>
	{/snippet}
</IslandCapsule>
