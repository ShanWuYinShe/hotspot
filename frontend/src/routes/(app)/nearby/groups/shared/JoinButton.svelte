<script lang="ts">
	import { Check } from '@lucide/svelte';
	import type { Group } from './group-data';

	// 加入/已加入切换按钮（对齐 Figma Make NearbyGroupsPage JoinButton）。
	// 双态样式全部类化——inline 背景会压制 hover 反馈，导致两态 hover 相同：
	//   未加入：accent 实底 + glow，hover 光晕抬档（hover:shadow-glow-md）；
	//   已加入：中性描边，hover 淡底填充 + 文字加深——两态 hover 反馈不同（四态契约）。
	// pending：提交中禁点防重复请求（乐观更新期间）。
	let {
		group,
		pending = false,
		onToggle
	}: { group: Group; pending?: boolean; onToggle: () => void } = $props();
</script>

<button
	type="button"
	class="hover-btn hit-slop focus-ring flex shrink-0 items-center gap-1 rounded-full px-3.25 py-1.5 text-sm
		{group.joined
		? 'border border-border-strong text-text-secondary hover:bg-interactive-hover hover:text-text-primary'
		: 'bg-accent-default font-semibold text-text-on-accent shadow-glow-sm hover:shadow-glow-md'}
		{pending ? 'pointer-events-none opacity-soft' : ''}"
	aria-pressed={group.joined}
	disabled={pending}
	onclick={(e) => {
		e.stopPropagation();
		onToggle();
	}}
>
	{#if group.joined}
		<Check size={12} />
		已加入
	{:else}
		加入
	{/if}
</button>
