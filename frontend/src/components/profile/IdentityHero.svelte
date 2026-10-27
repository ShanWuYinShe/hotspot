<script lang="ts">
	import type { Snippet } from 'svelte';
	import { MapPin } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import MetricRow, { type MetricItem } from '@/components/data-display/MetricRow.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import { fadeImage, presetImageUrl, presetSrcSet } from '@/lib/images';
	import { platform } from '@/stores/platform.svelte';
	import { cn } from '@/lib/utils';

	// IdentityHero — 资料页身份封面卡（R13 母题 B「叙事封面」）。
	// /me（ProfilePage / ProfileMobile）与 /users/[userId] 共用同一实现：
	//   封面（桌面 132 / 移动 92：主题预览渐变 + 一层柔光；有 coverUrl 时铺图 + 底部遮罩）
	//   → 头像压封面（桌面 88 / 移动 72 + ring-4 画布色环）→ 昵称 / handle / 在线状态
	//   → 简介 / 位置 / 兴趣标签
	//   → 指标（MetricRow strip）与动作位：桌面右列（左身份 / 右指标+动作，避免卡片右半留空岛），
	//     平板/移动通栏纵排（动作整行落在大拇指区）。
	//
	// 契约：
	// - 不改资料数据结构：所有内容由调用方传入，动作/额外内容走 snippet 注入；
	// - 封面图走 lib/images 统一管线（尺寸档 + srcset + 渐显），无图时只用主题 token；
	// - 颜色/阴影/字号/圆角一律走 token 类，只有布局几何（封面/头像高度、负边距）用 px 刻度；
	// - 同一份 DOM 承载三端（不复制移动端/桌面端两套结构），避免重复交互节点。

	let {
		/** self=自己的主页（不显示在线状态）；other=他人主页（按接口真值显示在线状态） */
		mode = 'self',
		name,
		/** 账号标识行（如 @user_lumos）；他人主页可传备注等关系信息 */
		handle = '',
		bio = '',
		avatar = '',
		/** 可选封面图；缺省用主题封面预览渐变（当前资料接口无封面字段） */
		coverUrl = '',
		location = '',
		tags = [],
		/** 仅 other 模式消费：undefined = 数据未知/对方隐藏，不渲染在线状态 */
		online,
		onlineLabel = '在线',
		offlineLabel = '离线',
		metrics = [],
		/** 昵称标题层级：页面已有 PageHeader(h1) 时传 h2（默认），整页唯一标题时传 h1 */
		titleTag = 'h2',
		ariaLabel,
		/** 动作位：主操作 + 次操作（编辑资料/关注/私信等） */
		actions,
		/** 压在封面上的浮层动作（返回/分享/更多），桌面/移动一致 */
		coverActions,
		children,
		class: className = ''
	}: {
		mode?: 'self' | 'other';
		name: string;
		handle?: string;
		bio?: string;
		avatar?: string;
		coverUrl?: string;
		location?: string;
		tags?: string[];
		online?: boolean;
		onlineLabel?: string;
		offlineLabel?: string;
		metrics?: MetricItem[];
		titleTag?: 'h1' | 'h2';
		ariaLabel?: string;
		actions?: Snippet;
		coverActions?: Snippet;
		children?: Snippet;
		class?: string;
	} = $props();

	const isMobile = $derived(platform.platform === 'mobile');
	const avatarPx = $derived(isMobile ? 72 : 88);
	// 在线状态只在「他人主页 + 数据已知」时出现：自己的在线状态没有信息量
	const showPresence = $derived(mode === 'other' && online !== undefined);
	// 头像角标只在「他人主页 + 确实在线」时出现：数据未知/离线时不假装在线
	const showOnlineDot = $derived(mode === 'other' && online === true);
	const regionLabel = $derived(ariaLabel ?? (mode === 'self' ? '我的资料' : `${name} 的资料`));
</script>

<section class={cn('overflow-hidden rounded-lg bg-surface-island shadow-xs', className)} aria-label={regionLabel}>
	<!-- 封面层：主题预览渐变（或封面图 + 遮罩）+ 一层柔光；两种视觉叠加已是上限，不再加描边 -->
	<div class="relative h-23 overflow-hidden sm:h-33">
		{#if coverUrl}
			<img
				src={presetImageUrl(coverUrl, 'cover', 1600)}
				srcset={presetSrcSet(coverUrl, 'cover')}
				sizes="(min-width: 640px) 58rem, 100vw"
				alt=""
				aria-hidden="true"
				loading="lazy"
				decoding="async"
				class="absolute inset-0 h-full w-full object-cover"
				use:fadeImage
			/>
			<!-- 封面图上的底部压暗：保证压图浮层动作/文字的对比度（只压底部，不整块变暗） -->
			<div class="absolute inset-0" style="background:var(--cst-surface-cover-scrim);"></div>
		{:else}
			<div class="absolute inset-0" style="background:var(--cst-surface-cover-placeholder);"></div>
		{/if}
		<!-- 柔光：主题环境光斜向叠一层，让封面不是一条平渐变（色值全部来自 --cst-ambient-*） -->
		<div
			aria-hidden="true"
			class="pointer-events-none absolute inset-0"
			style="background:radial-gradient(52% 88% at 16% 0%, var(--cst-ambient-glow-1) 0%, transparent 72%);"
		></div>
		{#if coverActions}
			<div class="absolute inset-x-0 top-0 flex items-start justify-between gap-2 p-3 sm:p-4">
				{@render coverActions()}
			</div>
		{/if}
	</div>

	<!-- 身份区：relative 让头像的负边距压到封面上（封面是 positioned 元素，不提升层级会盖住身份文字）。
	     桌面两栏（左：身份/简介/标签，右：指标 + 动作），避免 1024 宽卡片右半整片空；平板/移动单列。 -->
	<div class="relative grid gap-x-8 px-4 pb-5 sm:px-6 lg:grid-cols-[minmax(0,1fr)_15rem] lg:items-start">
		<div class="min-w-0">
			<div class="-mt-9 flex items-end gap-4 sm:-mt-11 sm:gap-5">
				<div class="relative shrink-0">
					<Avatar src={avatar} name={name} size={avatarPx} decorative class="ring-4 ring-surface-page shadow-raised-lg">
						{#snippet badge()}
							{#if showOnlineDot}
								<OnlineDot size="lg" class="bottom-1 right-1" />
							{/if}
						{/snippet}
					</Avatar>
				</div>
				<div class="min-w-0 flex-1 pb-0.5 sm:pb-1">
					<div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1.5">
						<svelte:element
							this={titleTag}
							class="min-w-0 truncate text-xl font-semibold tracking-title text-text-primary sm:text-2xl"
						>
							{name || '未设置昵称'}
						</svelte:element>
						{#if showPresence}
							<span
								class="inline-flex h-5 shrink-0 items-center justify-center gap-1 rounded-full px-2 text-xxs font-medium leading-none {online
									? 'bg-status-online-bg text-text-status-online-on-soft'
									: 'bg-surface-subtle text-text-secondary'}"
							>
								{#if online}
									<span class="size-1.5 rounded-full bg-status-online" aria-hidden="true"></span>
								{/if}
								{online ? onlineLabel : offlineLabel}
							</span>
						{/if}
					</div>
					{#if handle}
						<p class="mt-1 truncate text-xs text-text-secondary">{handle}</p>
					{/if}
				</div>
			</div>

			{#if bio}
				<p class="mt-3 max-w-115 text-sm leading-relaxed text-text-secondary sm:mt-4">{bio}</p>
			{/if}

			{#if location || tags.length > 0}
				<div class="mt-2.5 flex flex-wrap items-center gap-x-3.5 gap-y-2 sm:mt-3">
					{#if location}
						<span class="inline-flex min-w-0 items-center gap-1.5 text-xs text-text-secondary">
							<MapPin class="size-3.5 shrink-0 text-accent-default" strokeWidth={1.75} />
							<span class="truncate">{location}</span>
						</span>
					{/if}
					{#each tags as tag (tag)}
						<span
							class="inline-flex h-6 items-center justify-center rounded-full bg-accent-soft px-2.5 text-xs font-medium leading-none text-text-accent-on-soft"
						>
							#{tag}
						</span>
					{/each}
				</div>
			{/if}

			{#if children}
				<div class="mt-3">{@render children()}</div>
			{/if}
		</div>

		{#if metrics.length > 0 || actions}
			<!-- 桌面：右列（指标做节奏锚点 + 动作收敛成一组）；移动/平板：通栏发丝线分隔后纵排。
			     发丝线是母题 A 允许的区块顶线，非页面级横线。 -->
			<div class="mt-4 flex flex-col gap-4 border-t border-border-subtle pt-4 lg:mt-8 lg:border-t-0 lg:pt-0">
				{#if metrics.length > 0}
					<MetricRow items={metrics} variant="strip" />
				{/if}
				{#if actions}
					<div class="flex flex-wrap items-center gap-2 lg:flex-col lg:items-stretch lg:[&>*]:w-full">
						{@render actions()}
					</div>
				{/if}
			</div>
		{/if}
	</div>
</section>
