<script lang="ts">
	import { Check } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';
	import type { Group } from './group-data';

	// 加入/已加入/审核中三态切换按钮（对齐群详情页 toggleJoin 三态语义，
	// 与 3cbb2bec 同类：需审核群申请后不得冒充「已加入」）。
	// 统一走 ui/button 原语（审查 H1：第四种手写主按钮收敛）：
	//   未加入：primary（accent 实底 + 光晕抬档）胶囊 sm 档；
	//   已加入/审核中：outline 变体 + 描边加深/文字降级覆写，hover 淡底填充——
	//   两态 hover 反馈不同（四态契约）。pending：禁点防重复提交（乐观更新期间）。
	// 审核中点击 = 取消申请（onToggle 由页面分流），与详情页「审核中 · 点击取消」一致。
	let {
		group,
		pending = false,
		pendingApproval = false,
		onToggle
	}: { group: Group; pending?: boolean; pendingApproval?: boolean; onToggle: () => void } = $props();
</script>

<Button
	type="button"
	variant={group.joined || pendingApproval ? 'outline' : 'primary'}
	size="sm"
	disabled={pending}
	class="hit-slop rounded-full px-3 gap-1
		{group.joined || pendingApproval
		? 'border-border-strong text-text-secondary hover:bg-interactive-hover hover:text-text-primary'
		: 'font-semibold'}
		{pending ? 'pointer-events-none opacity-soft' : ''}"
	aria-pressed={group.joined}
	onclick={(e) => {
		e.stopPropagation();
		onToggle();
	}}
>
	{#if pendingApproval}
		<Check size={12} />
		审核中 · 点击取消
	{:else if group.joined}
		<Check size={12} />
		已加入
	{:else}
		加入
	{/if}
</Button>
