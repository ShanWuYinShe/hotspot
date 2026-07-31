<script lang="ts">
	import { Grid3X3, Heart, Bookmark } from '@lucide/svelte';

	// 个人中心 Tab 栏（迁移自 React ProfileSections.tsx 内 ProfileTabs）。

	export type Tab = 'posts' | 'liked' | 'saved';

	let { activeTab, setActiveTab }: { activeTab: Tab; setActiveTab: (t: Tab) => void } = $props();

	const tabs = [
		{ id: 'posts', icon: Grid3X3, label: '帖子' },
		{ id: 'liked', icon: Heart, label: '喜欢' },
		{ id: 'saved', icon: Bookmark, label: '收藏' }
	] as const;
</script>

<div class="flex" style="border-bottom:1px solid var(--neu-border);padding-left:16px;padding-right:16px;">
	{#each tabs as { id, icon: Icon, label }}
		{@const isActive = activeTab === id}
		<button
			type="button"
			class="neu-hover-btn relative flex items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none text-[13px] {isActive ? 'text-[var(--neu-accent-primary)] font-medium' : 'text-[var(--neu-text-muted)] font-normal'}"
			style="flex:1;transition:color 0.2s;"
			onclick={() => setActiveTab(id)}
		>
			<Icon style="width:14px;height:14px;" strokeWidth={isActive ? 2 : 1.5} />
			<span>{label}</span>
			{#if isActive}
				<span class="bg-[var(--neu-accent-primary)]" style="position:absolute;bottom:-1px;left:18%;right:18%;height:2px;border-radius:1px;"></span>
			{/if}
		</button>
	{/each}
</div>
