<script lang="ts">
	import { Check } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';
	import type { Group } from './group-data';
	import { popOn } from '@/lib/motion';

	// 加入/已加入/审核中三态切换按钮（对齐群详情页 toggleJoin 三态语义，
	// 与 3cbb2bec 同类：需审核群申请后不得冒充「已加入」）。
	// 统一走 ui/button 原语（审查 H1：第四种手写主按钮收敛）：
	//   未加入：primary（accent 实底 + 光晕抬档）胶囊 sm 档；
	//   已加入/审核中：soft（中性软填，R13 次级动作统一语言，不再用描边胶囊）+
	//   压图时换成深色遮罩底白字（onMedia），两种形态各自有 hover 反馈。
	// 审核中点击 = 取消申请（onToggle 由页面分流），与详情页「审核中 · 点击取消」一致。
	let {
		group,
		pending = false,
		pendingApproval = false,
		onMedia = false,
		onToggle
	}: {
		group: Group;
		pending?: boolean;
		pendingApproval?: boolean;
		/** 压在图片/遮罩上的形态：白色描边+白字（已加入/审核中），加入仍用 accent 实底 */
		onMedia?: boolean;
		onToggle: () => void;
	} = $props();
	// 点击驱动的 pop：仅未加入→加入/申请时播；初始 joined 同步不播。
	let joinPop = $state(0);
</script>

<Button
	type="button"
	variant={group.joined || pendingApproval ? 'soft' : 'primary'}
	size="sm"
	shape="pill"
	disabled={pending}
	class="hit-slop px-3 gap-1
		{group.joined || pendingApproval
			? onMedia
				? 'bg-surface-media-scrim-strong text-content-inverse backdrop-blur-sm hover:bg-surface-media-scrim-strong/80'
				: 'text-text-secondary hover:text-text-primary'
			: 'font-semibold'}
		{pending ? 'pointer-events-none opacity-soft' : ''}"
	aria-pressed={group.joined}
	onclick={(e) => {
		e.stopPropagation();
		if (!group.joined && !pendingApproval) joinPop += 1;
		onToggle();
	}}
>
	<span class="inline-flex items-center gap-1" use:popOn={joinPop}>
		{#if pendingApproval}
			<Check size={12} />
			审核中 · 点击取消
		{:else if group.joined}
			<Check size={12} />
			已加入
		{:else}
			加入
		{/if}
	</span>
</Button>
