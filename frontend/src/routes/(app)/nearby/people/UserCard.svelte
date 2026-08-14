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
	class="rounded-md aspect-[3/4]"
>
	{#snippet placeholder()}
		<span class="text-accent-default text-3xl font-semibold">{Array.from(user.name)[0] ?? '?'}</span>
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
					<span
						class="shrink-0 rounded-full size-2.25 ring-2 ring-surface-card-menu bg-status-online"
						title="在线"
					></span>
				{/if}
			</div>
			<div class="mt-2 flex items-center justify-between gap-2">
				{#if user.tag}
					<span class="badge border border-accent-border bg-accent-soft text-accent-default">#{user.tag}</span>
				{:else}
					<span></span>
				{/if}
				<button
					type="button"
					class="hover-btn scale-btn tap-92 hit-slop flex items-center justify-center rounded-full focus-ring shrink-0 size-7"
					style="{followed
						? 'background:var(--cst-accent-soft);border:var(--cst-border-width-hairline) solid var(--cst-accent-border);color:var(--cst-accent-default);'
						: 'background:var(--cst-accent-default);color:var(--cst-text-on-accent);box-shadow:var(--shadow-glow-sm);'}"
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
