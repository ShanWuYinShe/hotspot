<script lang="ts">
	import { MapPin, UserCheck, UserPlus } from '@lucide/svelte';
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
		onFollow,
		onNavigate
	}: {
		user: NearbyUser;
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
	class="rounded-panel aspect-[3/4] surface-flat"
>
	{#snippet placeholder()}
		<span class="text-accent-default text-3xl font-semibold">{Array.from(user.name)[0] ?? '?'}</span>
	{/snippet}

	<!-- 右上：距离（单份，消除历史重复徽章 bug） -->
	<div class="absolute top-2.5 right-2.5 z-[2] flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-surface-overlay-strong" style="backdrop-filter:blur(8px);">
		<MapPin class="w-2.25 h-2.25 text-white" strokeWidth={1.5} />
		<span class="text-white text-xxs">{user.distance}</span>
	</div>

	{#snippet footer()}
		<div class="px-3 pt-2.5 pb-2.5">
			<div class="flex items-center justify-between gap-2">
				<div class="min-w-0">
					<p class="flex items-center gap-1.5 truncate text-text-primary text-base" style="font-weight:600;">
						<span class="truncate">{user.name}</span>
					</p>
					<p class="truncate text-text-secondary text-xxs mt-0.5">{user.note}</p>
				</div>
				{#if user.isOnline}
					<span
						class="shrink-0 rounded-full"
						title="在线"
						style="width:9px;height:9px;background:var(--status-online);box-shadow:0 0 0 2px var(--surface-card-menu);"
					></span>
				{/if}
			</div>
			<div class="mt-2 flex items-center justify-between gap-2">
				{#if user.tag}
					<span class="rounded-full px-1.5 py-0.5 text-xxs border border-accent-border"  style="background:var(--accent-soft);color:var(--accent-default)">#{user.tag}</span>
				{:else}
					<span></span>
				{/if}
				<button
					type="button"
					class="hover-btn tap-90 flex items-center justify-center rounded-full outline-none shrink-0"
					style="width:28px;height:28px;{followed
						? 'background:var(--accent-soft);border:1px solid var(--accent-border);color:var(--accent-default);'
						: 'background:var(--accent-default);color:var(--text-on-accent);box-shadow:0 2px 8px var(--glow-accent);'}"
					aria-label={followed ? `取消关注 ${user.name}` : `关注 ${user.name}`}
					onclick={(e) => {
						e.stopPropagation();
						onFollow();
					}}
				>
					{#if followed}
						<UserCheck size={14} />
					{:else}
						<UserPlus size={14} />
					{/if}
				</button>
			</div>
		</div>
	{/snippet}
</NearbyImageCard>
