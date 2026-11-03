<script lang="ts">
	import { Check } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';
	import { popOn } from '@/lib/motion';

	// JoinButton —— 群组加入/已加入/审核中三态切换按钮（对齐 FollowButton 关系原语）：
	// - 未加入（非压图）：surface-accent（accent 淡底 Tonal + accent 字）——与 FollowButton
	//   未关注态同一安静档（R13 quiet 口径），卡片同级 CTA 不再一处实底一处淡底；
	// - 未加入（onMedia 压图）：仍用 primary（accent 实底 + on-accent 字）——半透明淡底压在
	//   照片上对比度不可控，压图形态按 design-system §17.2「未加入主按钮可用 accent 实底」；
	// - 已加入 / 审核中：soft（中性软填，次级动作统一语言，不再用描边胶囊）；
	// - 压图时已加入换深色遮罩底白字（onMedia），两种形态均有明确 hover 反馈；
	// - 审核中点击 = 取消申请（onToggle 由页面分流），与详情页一致。
	let {
		group,
		pending = false,
		pendingApproval = false,
		onMedia = false,
		onToggle
	}: {
		group: { joined?: boolean };
		pending?: boolean;
		pendingApproval?: boolean;
		/** 压在图片/遮罩上的形态：深色遮罩+白字（已加入/审核中）；未加入时用 accent 实底保证压图对比度 */
		onMedia?: boolean;
		onToggle: () => void;
	} = $props();

	// 点击驱动的 pop：仅未加入→加入/申请时播；初始 joined 同步不播。
	let joinPop = $state(0);
</script>

<Button
	type="button"
	variant={group.joined || pendingApproval ? 'soft' : onMedia ? 'primary' : 'surface-accent'}
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
