<script lang="ts">
	import { page } from '$app/state';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { usersService } from '$lib/api/services';
	import { navigateToNearbyBackTarget } from '../../nearby-back';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
	import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
	import UserCard from '$lib/components/common/user-card.svelte';
	import MasonryLayout from '$lib/components/common/masonry-layout.svelte';
	import MasonryLayoutItem from '$lib/components/common/masonry-layout-item.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { UserItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Users from 'lucide-svelte/icons/users';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
	import LogIn from 'lucide-svelte/icons/log-in';

	const pagination = usePagination<UserItem>({
		uniqueKey: (item) => item.user.user_id
	});

	let hasLoadedOnce = $state(false);

	const returnTarget = $derived(page.url.searchParams.get('from') || '/search');

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function loadUsers(refresh = false) {
		await pagination.load(
			async (cursor) => {
				const response = await usersService.search({
					radius: mapAnchorStore.radius * 1000,
					reference_point: getAnchor(),
					pagination: { limit: 20, cursor }
				});
				const currentUserId = authStore.currentUserId;
				if (currentUserId && response.data?.items) {
					response.data.items = response.data.items.filter(
						(item) => item.user.user_id !== currentUserId
					);
				}
				return response;
			},
			refresh
		);
		hasLoadedOnce = true;
	}

	function handleBack() {
		navigateToNearbyBackTarget(typeof window === 'undefined' ? undefined : window, returnTarget);
	}

	function handleRetry() {
		void loadUsers(true);
	}

	useLocationAwareLoad((refresh) => loadUsers(refresh));
</script>

<svelte:head>
	<title>附近的人 - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-7xl space-y-4 px-3 py-4 sm:space-y-5 sm:px-6 sm:py-6 lg:px-8">
	<!-- Sticky header -->
	<div
		class="sticky top-0 z-10 flex items-center justify-between px-4 py-3"
		style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
	>
		<div>
			<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">附近的人</h1>
			<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">基于当前浏览位置，按距离排序展示可互动用户。</p>
		</div>
		<div class="flex items-center gap-2">
			<button
				class="flex items-center justify-center rounded-full outline-none"
				style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border);"
				onclick={handleBack}
				aria-label="返回"
			>
				<ArrowLeft size={18} color="var(--neu-text)" />
			</button>
			<div
				class="flex items-center gap-1.5 rounded-full py-1.5 px-3.5"
				style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
			>
				<MapPin size={12} strokeWidth={1.5} color="var(--neu-accent-primary)" />
				<span style="font-size: 11px; color: var(--neu-text-muted); max-width: 12ch; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{mapAnchorStore.displayName}</span>
			</div>
		</div>
	</div>

	<AnchorSummaryCard
		label="附近发现锚点"
		description={`当前按 ${mapAnchorStore.radius}km 范围加载用户结果`}
		subtle
	/>

	{#if pagination.loading}
		<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
			{#each Array(6) as _, i (i)}
				<MasonryLayoutItem>
					<div class="animate-pulse rounded-[24px]" style="height: {180 + (i % 3) * 30}px; background: var(--neu-border);"></div>
				</MasonryLayoutItem>
			{/each}
		</MasonryLayout>
	{:else if pagination.error}
		{#if !authStore.isAuthenticated}
			<EmptyState
				icon={LogIn}
				title="登录后查看"
				description="登录后即可浏览附近的人"
				actionText="立即登录"
				onAction={() => uiStore.openLoginModalWithRedirect({})}
			/>
		{:else}
			<EmptyState
				icon={AlertTriangle}
				title="加载失败"
				description={pagination.error}
				actionText="重试"
				onAction={handleRetry}
			/>
		{/if}
	{:else if pagination.items.length === 0 && hasLoadedOnce}
		<EmptyState icon={Users} title="附近暂无其他用户" description="换个浏览位置或稍后再来看看。" />
	{:else}
		<MasonryLayout
			columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }}
			gap={16}
			animated={true}
			animationDelay={50}
		>
			{#each pagination.items as user (user.user.user_id)}
				<MasonryLayoutItem>
					<UserCard
						{user}
						showDistance
						showBio
						showTags
						showOnlineStatus
						variant="square"
						class="overflow-hidden"
					/>
				</MasonryLayoutItem>
			{/each}
		</MasonryLayout>

		<LoadMoreButton
			loading={pagination.loadingMore}
			hasMore={pagination.hasMore}
			onLoadMore={() => loadUsers(false)}
		/>
	{/if}
</div>
