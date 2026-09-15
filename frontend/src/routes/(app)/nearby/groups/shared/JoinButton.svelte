<script lang="ts">
	import { Check } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';
	import type { Group } from './group-data';

	// 加入/已加入切换按钮（对齐 Figma Make NearbyGroupsPage JoinButton）。
	// 统一走 ui/button 原语（审查 H1：第四种手写主按钮收敛）：
	//   未加入：primary（accent 实底 + 光晕抬档）胶囊 sm 档；
	//   已加入：outline 变体 + 描边加深/文字降级覆写，hover 淡底填充——两态
	//   hover 反馈不同（四态契约）。pending：禁点防重复提交（乐观更新期间）。
	let {
		group,
		pending = false,
		onToggle
	}: { group: Group; pending?: boolean; onToggle: () => void } = $props();
</script>

<Button
	type="button"
	variant={group.joined ? 'outline' : 'primary'}
	size="sm"
	disabled={pending}
	class="hit-slop rounded-full px-3 gap-1
		{group.joined
		? 'border-border-strong text-text-secondary hover:bg-interactive-hover hover:text-text-primary'
		: 'font-semibold'}
		{pending ? 'pointer-events-none opacity-soft' : ''}"
	aria-pressed={group.joined}
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
</Button>
