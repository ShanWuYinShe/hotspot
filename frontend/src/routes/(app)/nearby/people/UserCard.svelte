<script lang="ts">
	import { MapPin, UserRound } from '@lucide/svelte';
	import FollowButton from '@/components/ui/FollowButton.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';

	// 附近用户卡片：基于 NearbyImageCard 统一图片优先原语。
	// 标签/距离/关注等菜单以覆盖层显示在图片上。
	// 桌面/平板 BSP 布局中高度由布局单元决定；移动网格中由 aspect-[3/4] 撑起。

	export interface NearbyUser {
		id: string;
		name: string;
		distance: string;
		avatar: string;
		isOnline: boolean;
		/** 展示用首标签（卡片 #tag）；筛选走全量 tags（首标签之外同样可命中） */
		tag: string;
		tags?: string[];
		mutual?: number;
		note?: string;
		/** 查看者是否已关注（users/search relation.is_following；进度缓存沿用） */
		isFollowing?: boolean;
	}

	let {
		user,
		followed,
		pending = false,
		priority = false,
		onFollow,
		onNavigate
	}: {
		user: NearbyUser;
		/** 关注请求进行中（按钮禁点防重复提交） */
		pending?: boolean;
		/** 首屏前 2 张（N1）：透传 NearbyImageCard 的 eager + fetchpriority=high */
		priority?: boolean;
		followed: boolean;
		onFollow: () => void;
		onNavigate: () => void;
	} = $props();

	// 头像缺失或加载失败时的占位状态；头像地址变化后重置
	let avatarFailed = $state(false);
	$effect(() => {
		void user.avatar;
		avatarFailed = false;
	});
</script>

<NearbyImageCard
	imageSrc={avatarFailed ? '' : user.avatar}
	imageAlt={user.name}
	{priority}
	onClick={onNavigate}
	onImageError={() => (avatarFailed = true)}
	class="aspect-[3/4] h-auto rounded-lg shadow-xs hover:shadow-raised-sm transition-all duration-standard"
>
	{#snippet watermark()}
		<UserRound size={72} strokeWidth={1} />
	{/snippet}

	{#snippet placeholder()}
		<span class="flex items-center justify-center rounded-full bg-accent-soft size-16">
			<span class="text-accent-default text-3xl font-semibold">{Array.from(user.name)[0] ?? '?'}</span>
		</span>
	{/snippet}

	<!-- 右上：距离（单份，消除历史重复徽章 bug；无距离时整 pill 隐藏，免得剩个光杆图钉） -->
	{#if user.distance}
	<div class="absolute top-2.5 right-2.5 z-[2] flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-surface-overlay-strong" style="backdrop-filter:blur(var(--cst-blur-sm));">
		<MapPin class="w-2.5 h-2.5 text-content-inverse" strokeWidth={1.5} />
		<span class="text-content-inverse text-xxs">{user.distance}</span>
	</div>
	{/if}

	{#snippet footer({ onImage, horizontal }: { onImage: boolean; horizontal: boolean })}
	{@const onCover = onImage && !horizontal}
	<div class="p-4">
		<p class="flex items-center gap-1.5 truncate text-sm font-semibold {onCover ? 'text-content-inverse' : 'text-text-primary'}">
			<span class="truncate">{user.name}</span>
			{#if user.isOnline}
				<span
					class="shrink-0 rounded-full size-2 bg-status-online {onCover
						? 'ring-2 ring-content-inverse opacity-emphasis'
						: 'ring-2 ring-surface-card-menu'}"
					title="在线"
				></span>
			{/if}
		</p>
		{#if user.note}
			<p class="mt-0.5 truncate text-xxs {onCover ? 'text-content-inverse opacity-emphasis' : 'text-text-secondary'}">{user.note}</p>
		{/if}
		<div class="mt-2 flex items-center justify-between gap-2">
			{#if user.tag}
				<span class="badge {onCover ? 'bg-surface-media-scrim-strong text-content-inverse' : 'bg-accent-soft text-accent-default'}">#{user.tag}</span>
			{:else}
				<span></span>
			{/if}
			<FollowButton iconOnly active={followed} pending={pending} onMedia={onCover} onToggle={() => onFollow()} />
		</div>
	</div>
{/snippet}
</NearbyImageCard>
