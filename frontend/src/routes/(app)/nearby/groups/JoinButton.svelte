<script lang="ts">
	import { Check } from '@lucide/svelte';
	import type { Group } from './group-data';

	// 加入/已加入切换按钮（对齐 Figma Make NearbyGroupsPage JoinButton）。
	// 按钮位于底部 card-menu tray（非深色图片上），已加入用中性描边样式。

	let { group, onToggle }: { group: Group; onToggle: () => void } = $props();
</script>

<button
	type="button"
	class="neu-hover-btn rounded-full outline-none text-[11px] px-[13px] py-1.5 flex-shrink-0"
	style={group.joined
		? 'border:1px solid var(--neu-border-pressed);color:var(--neu-text-muted);'
		: 'background:var(--neu-accent-primary);color:var(--text-on-accent);font-weight:600;box-shadow:0 2px 8px var(--neu-glow);'}
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
