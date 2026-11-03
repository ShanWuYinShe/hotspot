<script lang="ts">
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { homeFeed, type HomeFeedTab } from '@/stores/home-feed.svelte';

	// 首页灵动岛（桌面端）：把信息流分段器（为你推荐 / 正在关注）固定到 52px 顶栏中央。
	// - 常驻、不占内容高度：滚动时不再出现第二条吸顶分段器；
	// - 视觉与页面锚点岛同源（玻璃胶囊 + 阴影），内容只有分段器一件事；
	// - 移动/平板不渲染本组件，继续用页内品牌头下的分段器。
	const items = [
		{ id: 'discover', label: '为你推荐' },
		{ id: 'following', label: '正在关注' }
	];
</script>

<div
	data-desktop-dynamic-island
	class="pointer-events-auto flex h-9 items-center rounded-full border border-border-default/50 bg-surface-primary/90 px-1 text-text-primary shadow-raised-sm backdrop-blur-xl animate-in fade-in zoom-in-95"
	style="contain: layout paint;"
>
	<TabBar
		variant="segmented"
		size="sm"
		{items}
		value={homeFeed.activeTab}
		onChange={(id) => homeFeed.setTab(id as HomeFeedTab)}
		class="min-w-[15rem] bg-transparent ring-0"
	/>
</div>
