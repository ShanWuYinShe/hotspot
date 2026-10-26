<script lang="ts">
	import { Check, UserCheck, UserPlus } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';

	// FollowButton — 关注/已关注切换的唯一实现（审计 P1-6：此前 6 种手写写法）。
	// 复用 ui/button 原语：未关注 = primary（accent 实底），已关注 = outline 变体
	// 覆写 accent 文字（JoinButton 同款两态范式）。pending 禁点防重复提交。
	// size（两派契约：胶囊只属于行内 chip 形态，页面级动作条归 md 派，见 style skill）：
	//   pill     列表行动作位胶囊 chip（rounded-full px-3.5 py-1.5 text-xs）；
	//   wide     详情页宽胶囊 chip（rounded-full px-4 py-2 text-xs，桌面/平板动作区）；
	//   block    移动端动作条（h-11 flex-1 text-sm，rounded-md——原 rounded-xl 破档已归并）。
	// iconOnly  空间受限行的纯图标圆钮（nearby 卡片）。
	// labelActive/labelInactive 允许语义定制（互关/回关），默认 关注/已关注。
	let {
		active,
		pending = false,
		disabled = false,
		size = 'pill',
		iconOnly = false,
		onMedia = false,
		labelActive = '已关注',
		labelInactive = '关注',
		onToggle
	}: {
		active: boolean;
		/** 乐观更新/请求中：禁点 + 半透明 */
		pending?: boolean;
		disabled?: boolean;
		size?: 'pill' | 'wide' | 'block';
		iconOnly?: boolean;
		/** 压在图片/遮罩上的形态：改用白色描边+白字（不依赖主题实底/浅玻璃底） */
		onMedia?: boolean;
		labelActive?: string;
		labelInactive?: string;
		onToggle: (e: MouseEvent) => void;
	} = $props();
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
				? 'border border-border-strong bg-transparent text-text-secondary hover:bg-interactive-hover'
				: 'bg-accent-bg text-accent-default border border-transparent hover:border-accent-border'}"
		onclick={(e) => {
			e.stopPropagation();
			onToggle(e);
		}}
	>
		{#if active}
			<Check class="size-4" strokeWidth={2.5} />
		{:else}
			<UserPlus class="size-4" strokeWidth={2} />
		{/if}
	</button>
{:else}
	<Button
		type="button"
		variant={active ? 'outline' : 'surface-accent'}
		size={size === 'block' ? 'lg' : 'sm'}
		disabled={pending || disabled}
		class="hit-slop tap-92 gap-1
			{size === 'pill' ? 'rounded-full px-3.5 py-1.5 text-xs' : ''}
			{size === 'wide' ? 'rounded-full px-4 py-2 text-xs' : ''}
			{size === 'block' ? 'h-11 flex-1' : ''}
			{active
			? 'border-accent-default bg-surface-primary text-accent-default hover:bg-accent-soft'
			: 'font-semibold'}
			{pending ? 'pointer-events-none opacity-soft' : ''}"
		aria-pressed={active}
		onclick={(e) => {
			e.stopPropagation();
			onToggle(e);
		}}
	>
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
	</Button>
{/if}
