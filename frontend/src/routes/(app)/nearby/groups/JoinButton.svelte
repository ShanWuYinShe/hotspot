<script lang="ts">
	import { Check } from '@lucide/svelte';
	import type { Group } from './group-data';

	// 加入/已加入切换按钮。卡片改为封面整卡后，按钮覆盖在图片上：
	// 未加入用主题主色实心；已加入用白色玻璃底保证深色图片上的可读性。

	let { group, onToggle }: { group: Group; onToggle: () => void } = $props();
</script>

<button
	type="button"
	class="neu-hover-btn rounded-full outline-none text-[11px] px-[13px] py-1.5 flex-shrink-0 {group.joined ? 'bg-white/15 border border-white/45 text-white backdrop-blur-sm' : 'bg-[var(--neu-accent-primary)] text-white font-semibold'}"
	style={group.joined ? '' : 'box-shadow:0 2px 8px var(--neu-glow);'}
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
