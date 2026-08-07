<script lang="ts">
	import { Check } from '@lucide/svelte';
	import type { Group } from './group-data';

	// 加入/已加入切换按钮（对齐 Figma Make NearbyGroupsPage JoinButton）。
	// 按钮位于底部 card-menu tray（非深色图片上），已加入用中性描边样式。

	let { group, onToggle }: { group: Group; onToggle: () => void } = $props();
</script>

<button
	type="button"
	class="hover-btn rounded-full outline-none text-xs px-3.25 py-1.5 shrink-0 {group.joined ? '' : 'font-semibold'}"
	style={group.joined
		? 'border:1px solid var(--cst-border-strong);color:var(--cst-text-secondary);'
		: 'background:var(--cst-accent-default);color:var(--cst-text-on-accent);box-shadow:var(--shadow-glow-sm);'}
	onclick={(e) => {
		e.stopPropagation();
		onToggle();
	}}
>
	{#if group.joined}
		<span class="flex items-center gap-1">
			<Check size={12} />
			已加入
		</span>
	{:else}
		加入
	{/if}
</button>
