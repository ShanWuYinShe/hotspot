<script lang="ts">
	import { Check, UserCheck, UserPlus } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';
	import { popOn } from '@/lib/motion';

	// FollowButton — 关注/已关注切换的唯一实现（审计 P1-6：此前 6 种手写写法）。
	// 复用 ui/button 原语，两态共用同一"安静次级动作"语言（R13 统一口径）：
	//   未关注 = surface-accent（accent 淡底 Tonal，引导点击）
	//   已关注 = soft（中性软填、无描边，回到信息态；与 JoinButton/私信/拒绝同族）
	// pending 禁点防重复提交。
	// size（两派契约：胶囊只属于行内 chip 形态，页面级动作条归 md 派，见 style skill）：
	//   chip     卡片作者行内的最小 chip（h-5 px-2 text-xxs + 图标，2026-10-29 为信息流卡片新增）；
	//   pill     列表行动作位胶囊 chip（rounded-full px-3.5 py-1.5 text-xs）；
	//   wide     详情页宽胶囊 chip（rounded-full px-4 py-2 text-xs，桌面/平板动作区）；
	//   block    移动端动作条（h-11 flex-1 text-sm，rounded-md——原 rounded-xl 破档已归并）。
	// iconOnly  空间受限行的纯图标圆钮（nearby 卡片）。
	// onMedia  压图形态：chip/pill 改用「玻璃底 + 反色字」；未关注态用 accent 实底提供强 affordance。
	// labelActive/labelInactive 允许语义定制（互关/回关），默认 关注/已关注。
	// 卡片契约（2026-10-29）：卡片作者行的关注状态必须是可点按钮（取消关注），不能是不可点的徽章——
	// 徽章没有 pointer-events，点击会穿透到整卡覆盖层打开详情（用户反馈的真实 bug）。
	let {
		active,
		pending = false,
		disabled = false,
		size = 'pill',
		iconOnly = false,
		onMedia = false,
		labelActive = '已关注',
		labelInactive = '关注',
		ariaLabel,
		onToggle
	}: {
		active: boolean;
		/** 乐观更新/请求中：禁点 + 半透明 */
		pending?: boolean;
		disabled?: boolean;
		size?: 'chip' | 'pill' | 'wide' | 'block';
		/** 无可见文案语义补充（如「取消关注 鹿鸣」）；有可见文案时不必传 */
		ariaLabel?: string;
		iconOnly?: boolean;
		/** 压在图片/遮罩上的形态：改用白色描边+白字（不依赖主题实底/浅玻璃底） */
		onMedia?: boolean;
		labelActive?: string;
		labelInactive?: string;
		onToggle: (e: MouseEvent) => void;
	} = $props();

	// 点击驱动的 pop：仅「未关注→关注」点击时递增；已关注/初始同步不播。
	let followPop = $state(0);
	function handleClick(e: MouseEvent) {
		if (!active && !pending && !disabled) followPop += 1;
		onToggle(e);
	}
</script>

{#if iconOnly}
	<button
		type="button"
		aria-pressed={active}
		aria-label={active ? labelActive : labelInactive}
		disabled={pending || disabled}
		class="hover-btn hit-slop tap-92 flex size-9 shrink-0 items-center justify-center rounded-full focus-ring disabled:pointer-events-none disabled:opacity-faint {onMedia
			? active
				? 'border border-content-inverse/70 bg-surface-media-scrim-strong text-content-inverse hover:bg-surface-media-scrim-strong/80'
				: 'border border-content-inverse/40 bg-surface-media-scrim-strong text-content-inverse hover:bg-surface-media-scrim-strong/80'
			: active
				? 'bg-surface-subtle text-text-secondary hover:bg-interactive-active'
				: 'bg-accent-bg text-accent-default border border-transparent hover:border-accent-border'}"
		onclick={(e) => {
			e.stopPropagation();
			handleClick(e);
		}}
	>
		<span class="inline-flex" use:popOn={followPop}>
			{#if active}
				<Check class="size-4" strokeWidth={2.5} />
			{:else}
				<UserPlus class="size-4" strokeWidth={2} />
			{/if}
		</span>
	</button>
{:else}
	<Button
		type="button"
		variant={active ? 'soft' : 'surface-accent'}
		shape={size === 'block' ? 'rect' : 'pill'}
		size={size === 'block' ? 'lg' : 'sm'}
		disabled={pending || disabled}
		class="pointer-events-auto hit-slop-44 tap-92 gap-1
			{size === 'chip' ? 'h-5 px-2 text-xxs' : ''}
			{size === 'pill' ? 'px-3.5 py-1.5 text-xs' : ''}
			{size === 'wide' ? 'px-4 py-2 text-xs' : ''}
			{size === 'block' ? 'h-11 flex-1' : ''}
			{onMedia && !active
				? 'border border-content-inverse/70 bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md'
				: ''}
			{onMedia && active
				? 'border border-content-inverse/40 bg-surface-media-scrim-strong text-content-inverse shadow-none hover:bg-surface-media-scrim-strong/85'
				: ''}
			{!onMedia && (active ? 'text-text-primary' : 'font-semibold')}
			{pending ? 'pointer-events-none opacity-soft' : ''}"
		aria-pressed={active}
		aria-label={ariaLabel}
		title={active ? '取消关注' : '关注'}
		onclick={(e) => {
			e.stopPropagation();
			handleClick(e);
		}}
	>
		<span class="inline-flex items-center gap-1" use:popOn={followPop}>
		{#if active}
			{#if labelActive === '已关注'}
				<UserCheck class="size-2.5" strokeWidth={1.5} />
			{:else}
				<Check class="size-2.5" strokeWidth={2.5} />
			{/if}
			{labelActive}
		{:else}
			<UserPlus class="size-2.5" strokeWidth={2} />
			{labelInactive}
		{/if}
		</span>
	</Button>
{/if}