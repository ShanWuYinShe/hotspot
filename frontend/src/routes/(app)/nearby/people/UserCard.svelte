<script lang="ts">
	import { MapPin } from '@lucide/svelte';
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
		tag: string;
		mutual: number;
		note: string;
	}

	let {
		user,
		followed,
		pending = false,
		onFollow,
		onNavigate
	}: {
		user: NearbyUser;
		/** 关注请求进行中（按钮禁点防重复提交） */
		pending?: boolean;
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
	onClick={onNavigate}
	onImageError={() => (avatarFailed = true)}
	class="rounded-md aspect-[3/4]"
>
	{#snippet placeholder()}
		<span class="flex items-center justify-center rounded-full bg-accent-soft size-16">
			<span class="text-accent-default text-3xl font-semibold">{Array.from(user.name)[0] ?? '?'}</span>
		</span>
	{/snippet}

	<!-- 右上：距离（单份，消除历史重复徽章 bug） -->
	<div class="absolute top-2.5 right-2.5 z-[2] flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-surface-overlay-strong" style="backdrop-filter:blur(var(--cst-blur-sm));">
		<MapPin class="w-2.5 h-2.5 text-content-inverse" strokeWidth={1.5} />
		<span class="text-content-inverse text-xxs">{user.distance}</span>
	</div>

	{#snippet footer()}
		<div class="px-3 pt-2.5 pb-2.5">
			<div class="flex items-center justify-between gap-2">
				<div class="min-w-0">
					<p class="flex items-center gap-1.5 truncate text-text-primary text-sm font-semibold">
						<span class="truncate">{user.name}</span>
					</p>
					<p class="truncate text-text-secondary text-xxs mt-0.5">{user.note}</p>
				</div>
				{#if user.isOnline}
					<!-- 行内状态点统一 8px 档（审查 M3：9px 近邻异档归并） -->
					<span
						class="shrink-0 rounded-full size-2 ring-2 ring-surface-card-menu bg-status-online"
						title="在线"
					></span>
				{/if}
			</div>
			<div class="mt-2 flex items-center justify-between gap-2">
				{#if user.tag}
					<span class="badge bg-accent-soft text-accent-default">#{user.tag}</span>
				{:else}
					<span></span>
				{/if}
				<!-- 纯图标档 FollowButton（空间受限卡片行） -->
				<FollowButton iconOnly active={followed} pending={pending} onToggle={() => onFollow()} />
			</div>
		</div>
	{/snippet}
</NearbyImageCard>
