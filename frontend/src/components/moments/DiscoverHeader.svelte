<script lang="ts">
	import { Search, X } from '@lucide/svelte';

	// 发现流共享页头（搜索条 + Tab 栏）。迁移自 React MomentsFeedPage.tsx。

	const TABS = ['热门', '附近', '活动', '关注'];

	let {
		searchVal,
		setSearchVal,
		activeTab,
		setActiveTab
	}: {
		searchVal: string;
		setSearchVal: (v: string) => void;
		activeTab: string;
		setActiveTab: (t: string) => void;
	} = $props();
</script>

<!-- Search bar -->
<div class="flex items-center gap-3 px-4 py-2.5 rounded-[18px] neu-input-surface mb-2.5">
	<Search style="width:15px;height:15px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
	<input
		bind:value={searchVal}
		oninput={(e) => setSearchVal(e.currentTarget.value)}
		placeholder="地点、人物、关键词…"
		class="flex-1 bg-transparent outline-none text-[var(--neu-text)] text-[13px]"
	/>
	{#if searchVal.length > 0}
		<button type="button" class="neu-hover-btn flex-shrink-0 outline-none" onclick={() => setSearchVal('')}>
			<X style="width:13px;height:13px;color:var(--neu-text-muted);" strokeWidth={2} />
		</button>
	{/if}
</div>

<!-- Tab bar -->
<div class="flex overflow-x-auto no-scrollbar" style="border-bottom:1px solid var(--neu-border);">
	{#each TABS as tab}
		{@const isActive = activeTab === tab}
		<button
			type="button"
			class="neu-hover-btn relative flex-shrink-0 pb-3 pt-1 outline-none text-[13px] px-[18px] transition-colors duration-200 {isActive ? 'text-[var(--neu-accent-primary)] font-medium' : 'text-[var(--neu-text-muted)] font-normal'}"
			onclick={() => setActiveTab(tab)}
		>
			{tab}
			{#if isActive}
				<div class="bg-[var(--neu-accent-primary)]" style="position:absolute;bottom:-1px;left:14%;right:14%;height:2px;border-radius:1px;"></div>
			{/if}
		</button>
	{/each}
</div>
