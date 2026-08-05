<script lang="ts">
	// TabBar — 下划线分类 Tab 栏（统一 nearby/moments 与关系聚合页等的手写重复实现）。
	// 容器 border-bottom + 激活项底部 2px 指示条 + aria-pressed，随 PageHeader below 吸顶。

	export type TabItem = { id: string; label: string; count?: number };

	let {
		items,
		value,
		onchange,
		class: className = ''
	}: {
		items: TabItem[];
		value: string;
		onchange: (id: string) => void;
		class?: string;
	} = $props();
</script>

<div class="no-scrollbar flex overflow-x-auto border-b border-border-default {className}">
	{#each items as tab}
		{@const isActive = value === tab.id}
		<button
			type="button"
			onclick={() => onchange(tab.id)}
			aria-pressed={isActive}
			class="relative shrink-0 rounded-t-small px-4.5 pb-3 pt-1 text-base outline-none transition-colors duration-200 {isActive ? 'font-medium text-accent-default' : 'font-normal text-text-secondary'}"
		>
			{tab.label}
			{#if tab.count !== undefined}
				<span class="ml-0.5 text-xxs opacity-75">{tab.count}</span>
			{/if}
			{#if isActive}
				<span class="absolute bottom-[-1px] left-[14%] right-[14%] h-0.5 rounded-[1px] bg-accent-default"></span>
			{/if}
		</button>
	{/each}
</div>
