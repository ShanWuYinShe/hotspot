<script lang="ts">
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { ArrowUp, TrendingUp, Users, LogIn, Sparkles, MapPin, Radio, ChevronRight, RotateCcw } from '@lucide/svelte';
	import { auth } from '@/stores/auth.svelte';
	import { headerIsland } from '@/stores/header-island.svelte';
	import { ROUTES } from '@/lib/routes';
	import { contentFade } from '@/lib/motion';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import HomeFeedItem from '@/components/home/HomeFeedItem.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import AdaptiveCardGrid from '@/components/layout/AdaptiveCardGrid.svelte';
	import InfiniteSentinel from '@/components/list/InfiniteSentinel.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import Button from '@/components/ui/button.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailMomentThumb from '@/components/rail/RailMomentThumb.svelte';
	import type { HomeFeedData } from '../shared/home-feed-data.svelte';

	let { data }: { data: HomeFeedData } = $props();

	function scrollToTop() {
		const scroller = document.getElementById('main-content');
		const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
		scroller?.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' });
	}

	const islandToken = Symbol('home-feed-island');
	const isDiscover = $derived(data.activeTab === 'discover');
	const islandTitle = $derived(isDiscover ? '为你推荐' : '正在关注');
	const islandMeta = $derived(
		isDiscover
			? '根据你的兴趣推荐'
			: auth.status === 'authenticated'
				? '关注好友与活跃邻居'
				: '登录后查看关注动态'
	);

	$effect(() => {
		headerIsland.register(
			islandToken,
			untrack(() => ({
				title: islandTitle,
				meta: islandMeta,
				icon: islandIcon,
				expanded: islandExpanded,
				onTitleClick: scrollToTop
			}))
		);
		return () => headerIsland.release(islandToken);
	});

	$effect(() => {
		headerIsland.update(islandToken, {
			title: islandTitle,
			meta: islandMeta,
			icon: islandIcon,
			expanded: islandExpanded,
			onTitleClick: scrollToTop
		});
	});
</script>

{#snippet islandIcon()}
	{#if isDiscover}
		<Sparkles class="size-3.5 text-accent-default" strokeWidth={1.75} />
	{:else}
		<Users class="size-3.5 text-accent-default" strokeWidth={1.75} />
	{/if}
{/snippet}

{#snippet islandExpanded()}
	<div class="flex flex-col gap-2.5">
		<div class="flex items-center justify-between text-xs pb-0.5">
			<span class="font-medium text-text-primary">信息流频道</span>
			<span class="font-mono text-xxs text-accent-default">{islandMeta}</span>
		</div>

		<!-- 分段选择器 -->
		<div class="flex justify-center w-full py-0.5">
			<TabBar
				variant="segmented"
				size="sm"
				items={[
					{ id: 'discover', label: '为你推荐' },
					{ id: 'following', label: '正在关注' }
				]}
				value={data.activeTab}
				onChange={(id) => {
					data.setTab(id as any);
				}}
				class="w-full justify-center bg-surface-subtle/80"
			/>
		</div>

		<!-- 快捷刷新操作 -->
		<div class="flex items-center justify-between pt-1 text-xxs text-text-secondary">
			<button
				type="button"
				class="group flex items-center gap-1.5 text-text-secondary hover:text-text-primary transition-colors text-xxs"
				onclick={() => void data.reload()}
			>
				<RotateCcw class="size-3 transition-transform duration-snappy group-active:-rotate-90" />
				<span>刷新动态</span>
			</button>
		</div>
	</div>
{/snippet}

<div class="flex flex-col flex-1 min-h-full">
	<SplitPageLayout contentClassName="flex flex-col pt-2 pb-8">
		{#snippet header()}
			<h1 class="sr-only">首页</h1>
		{/snippet}

		<div use:contentFade={data.currentLoading} class="min-h-[50vh] flex flex-col justify-between">
			{#if data.activeTab === 'following' && auth.status !== 'authenticated'}
				<!-- 关注流未登录态：引导登录卡片 -->
				<div class="mx-4 my-8 rounded-lg bg-surface-island p-6 text-center shadow-xs">
					<div class="mx-auto flex size-12 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft mb-3">
						<Users class="size-6" strokeWidth={1.75} />
					</div>
					<h2 class="text-md font-semibold text-text-primary">登录后查看关注动态</h2>
					<p class="mt-1 text-xs text-text-secondary max-w-sm mx-auto">
						关注同频的好友与活跃邻居，即时获取 TA 们的最新瞬间与生活记录。
					</p>
					<Button type="button" size="md" class="mt-4" onclick={() => void goto(ROUTES.login)}>
						<LogIn class="size-3.5" />
						<span>立即登录</span>
					</Button>
				</div>
			{:else if data.currentLoading}
				<SkeletonFeed
					variant="feed"
					count={4}
					label="加载动态中…"
					class="grid grid-cols-2 gap-2.5 px-2.5 pt-2 sm:grid-cols-3 sm:gap-4 sm:px-0 lg:grid-cols-[repeat(auto-fill,minmax(20rem,1fr))]"
				/>
			{:else if data.currentError && data.currentItems.length === 0}
				<LoadError message={data.currentError} onRetry={() => void data.reload()} />
			{:else if data.currentItems.length > 0}
				{#if data.currentItems.length === 1}
					<div class="mx-auto w-full max-w-md px-2.5 pt-2 sm:px-0">
						{#each data.currentItems as moment (moment.moment_id)}
							<HomeFeedItem
								{moment}
								showDistance={false}
								priority={true}
								onClick={() => void goto(ROUTES.moment(moment.moment_id))}
							/>
						{/each}
					</div>
				{:else if data.demoMode === 'a' || data.demoMode === 'b'}
					<div class="flex flex-col gap-3.5 px-3 pt-2 sm:gap-4 sm:px-0">
						{#each data.currentItems as moment, index (moment.moment_id)}
							<HomeFeedItem
								{moment}
								showDistance={false}
								priority={index < 2}
								onClick={() => void goto(ROUTES.moment(moment.moment_id))}
							/>
						{/each}
					</div>
				{:else if data.demoMode === 'c' || data.demoMode === 'd'}
					<div class="grid grid-cols-1 gap-4 px-3 pt-2 2xl:grid-cols-2 sm:px-0">
						{#each data.currentItems as moment, index (moment.moment_id)}
							<HomeFeedItem
								{moment}
								showDistance={false}
								priority={index < 2}
								onClick={() => void goto(ROUTES.moment(moment.moment_id))}
							/>
						{/each}
					</div>
				{:else}
					<AdaptiveCardGrid
						count={data.currentItems.length}
						minTrackRem={20}
						gapRem={[0.625, 1, 1]}
						class="px-2.5 pt-2 sm:px-0"
					>
						{#each data.currentItems as moment, index (moment.moment_id)}
							<HomeFeedItem
								{moment}
								showDistance={false}
								priority={index < 2}
								onClick={() => void goto(ROUTES.moment(moment.moment_id))}
							/>
						{/each}
					</AdaptiveCardGrid>
				{/if}

				{#if data.hasMore}
					<InfiniteSentinel
						hasMore={data.hasMore}
						loading={data.loadingMore}
						error={Boolean(data.currentError)}
						onLoadMore={() => void data.loadMore()}
					/>
				{:else}
					<ListEnd />
				{/if}

				{#if data.currentItems.length <= 2 && !data.hasMore}
					<div class="px-3.5 sm:px-0 my-3">
						<div class="rounded-lg bg-surface-island p-4 shadow-xs">
							<div class="flex items-center justify-between mb-2.5">
								<div class="flex items-center gap-1.5">
									<Sparkles class="size-4 text-accent-default" strokeWidth={1.75} />
									<span class="text-xs font-semibold text-text-primary">探索更多感兴趣的话题</span>
								</div>
								<span class="text-xxs text-text-secondary">社区热议</span>
							</div>
							<div class="flex flex-wrap gap-1.5">
								{#each data.trendingTopics.slice(0, 4) as topic (topic.tag)}
									<TagPill
										size="sm"
										onClick={() => void goto(`${ROUTES.search}?q=${encodeURIComponent('#' + topic.tag)}`)}
									>
										<span>#{topic.tag}</span>
										<span class="text-xxs opacity-70">({topic.count})</span>
									</TagPill>
								{/each}
							</div>
						</div>
					</div>
				{/if}
			{:else}
				<EmptyState
					title={data.activeTab === 'following' ? '暂无关注动态' : '还没有动态'}
					description={data.activeTab === 'following' ? '去「为你推荐」发现更多有趣的人，或去附近看看身边正在发生什么。' : '去附近看看，或者发布属于你的第一条瞬间。'}
					bare
					spotlight
				>
					{#snippet icon()}
						<Sparkles class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			{/if}
		</div>

		{#snippet rail()}
			{#if data.demoMode === 'd'}
				<div class="flex flex-col gap-4 py-2">
					<div class="relative overflow-hidden rounded-lg border border-border-default bg-surface-primary shadow-raised-sm">
						<div class="relative h-60 w-full bg-surface-subtle flex items-center justify-center overflow-hidden border-b border-border-subtle">
							<div class="absolute inset-0 flex items-center justify-center pointer-events-none opacity-subtle">
								<div class="size-28 rounded-full border border-accent-default/50 animate-ping"></div>
								<div class="absolute size-44 rounded-full border border-border-default/60"></div>
								<div class="absolute size-64 rounded-full border border-border-default/40"></div>
								<div class="absolute size-84 rounded-full border border-border-default/20"></div>
							</div>
							<div class="absolute top-10 left-10 flex items-center gap-1.5 rounded-full bg-surface-primary/95 backdrop-blur-sm px-3 py-1 shadow-raised-sm border border-border-subtle text-xs font-medium text-text-primary">
								<span class="size-2 rounded-full bg-status-online"></span> 云龙湖 · 8条动态
							</div>
							<div class="absolute bottom-12 right-10 flex items-center gap-1.5 rounded-full bg-surface-primary/95 backdrop-blur-sm px-3 py-1 shadow-raised-sm border border-border-subtle text-xs font-medium text-text-primary">
								<span class="size-2 rounded-full bg-accent-default"></span> 回龙窝街区 · 14人
							</div>
							<div class="relative flex flex-col items-center">
								<span class="relative flex size-6">
									<span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-accent-default opacity-soft"></span>
									<span class="relative inline-flex size-6 rounded-full bg-accent-default items-center justify-center text-text-on-accent shadow-raised-md">
										<Radio class="size-3.5" />
									</span>
								</span>
								<span class="mt-2 rounded-full bg-surface-primary px-3 py-0.5 text-xs font-bold text-text-primary shadow-xs border border-border-subtle">
									当前定位 · 5km 探索圈
								</span>
							</div>

							<button
								type="button"
								class="absolute top-3 right-3 flex items-center gap-1.5 rounded-lg bg-surface-primary/95 backdrop-blur-sm px-3 py-1.5 text-xs font-semibold text-text-primary shadow-xs border border-border-subtle hover:bg-surface-primary hover:text-accent-default transition"
								onclick={() => void goto(ROUTES.map)}
							>
								<MapPin class="size-3.5 text-accent-default" />
								<span>全景地图</span>
							</button>
						</div>

						<div class="p-4 space-y-3">
							<div class="grid grid-cols-3 gap-2.5 text-center">
								<div class="rounded-xl bg-surface-subtle p-3">
									<p class="text-xs text-text-secondary">周边动态</p>
									<p class="text-xl font-bold text-text-primary mt-0.5">38 <span class="text-xxs font-normal text-text-secondary">条</span></p>
								</div>
								<div class="rounded-xl bg-surface-subtle p-3">
									<p class="text-xs text-text-secondary">活跃邻居</p>
									<p class="text-xl font-bold text-text-primary mt-0.5">12 <span class="text-xxs font-normal text-text-secondary">人</span></p>
								</div>
								<div class="rounded-xl bg-surface-subtle p-3">
									<p class="text-xs text-text-secondary">同城圈子</p>
									<p class="text-xl font-bold text-text-primary mt-0.5">7 <span class="text-xxs font-normal text-text-secondary">个</span></p>
								</div>
							</div>

							<div class="flex items-center justify-between pt-1">
								<span class="text-xs text-text-secondary">当前状态：实时雷达已就绪</span>
								<button
									type="button"
									class="text-xs font-semibold text-accent-default hover:underline"
									onclick={() => void goto(ROUTES.nearby)}
								>
									打开周边探索中心 &rarr;
								</button>
							</div>
						</div>
					</div>

					<div class="grid grid-cols-1 xl:grid-cols-2 gap-4">
						{#if data.activeNeighbors.length > 0}
							<RailSection title="活跃邻居" variant="island">
								<div class="flex flex-col divide-y divide-border-subtle/30">
									{#each data.activeNeighbors as neighbor (neighbor.userId)}
										<div class="flex items-center gap-3 py-2.5 first:pt-0 last:pb-0">
											<Avatar src={neighbor.avatarUrl} name={neighbor.nickname} size="sm" />
											<div class="min-w-0 flex-1">
												<button
													type="button"
													class="text-left text-xs font-semibold text-text-primary hover:text-accent-default hover:underline transition-colors truncate block w-full"
													onclick={() => void goto(ROUTES.user(neighbor.userId))}
												>
													{neighbor.nickname}
												</button>
												{#if neighbor.location}
													<span class="text-xxs text-text-secondary truncate block mt-0.5">{neighbor.location}</span>
												{/if}
											</div>
										</div>
									{/each}
								</div>
							</RailSection>
						{/if}

						<RailSection title="社区热议" variant="island">
							<div class="flex flex-col divide-y divide-border-subtle/30">
								{#each data.trendingTopics.slice(0, 4) as topic (topic.tag)}
									<button
										type="button"
										class="hover-btn flex w-full items-center justify-between py-2 text-left transition hover:text-accent-default focus-ring first:pt-0 last:pb-0"
										onclick={() => void goto(`${ROUTES.search}?q=${encodeURIComponent('#' + topic.tag)}`)}
									>
										<div class="min-w-0 flex-1">
											<div class="flex items-center gap-1.5">
												<TrendingUp class="size-3 text-accent-default shrink-0" strokeWidth={1.75} />
												<span class="text-xs font-medium text-text-primary truncate">#{topic.tag}</span>
											</div>
										</div>
										<span class="text-xxs text-text-secondary tabular-nums shrink-0 ml-2">{topic.count} 条</span>
									</button>
								{/each}
							</div>
						</RailSection>
					</div>
				</div>
			{:else}
				<div class="flex flex-col gap-4 py-2">
					{#if data.activeNeighbors.length > 0}
						<RailSection
							title="活跃邻居"
							icon={Users}
							action="附近的人"
							onAction={() => void goto(ROUTES.nearbyPeople)}
							variant="island"
						>
							<div class="flex flex-col divide-y divide-border-subtle/30">
								{#each data.activeNeighbors as neighbor (neighbor.userId)}
									<button
										type="button"
										class="hover-bg focus-ring group/row -mx-2 flex min-w-0 items-center gap-2.5 rounded-sm px-2 py-2.5 text-left transition-colors first:pt-0 last:pb-0"
										aria-label={`查看 ${neighbor.nickname} 的个人主页`}
										onclick={() => void goto(ROUTES.user(neighbor.userId))}
									>
										<Avatar src={neighbor.avatarUrl} name={neighbor.nickname} size={32} />
										<span class="min-w-0 flex-1">
											<span class="block truncate text-sm font-medium text-text-primary">{neighbor.nickname}</span>
											{#if neighbor.location}
												<span class="mt-0.5 block truncate text-xxs text-text-secondary">{neighbor.location}</span>
											{/if}
										</span>
										<ChevronRight
											class="size-3.5 shrink-0 text-text-secondary transition-transform duration-rapid group-hover/row:translate-x-0.5 group-hover/row:text-accent-default"
											strokeWidth={1.5}
										/>
									</button>
								{/each}
							</div>
						</RailSection>
					{/if}

					<RailSection
						title="社区热议"
						icon={TrendingUp}
						action="查看全部"
						onAction={() => void goto(ROUTES.search)}
						variant="island"
					>
						<div class="flex flex-col divide-y divide-border-subtle/30">
							{#each data.trendingTopics.slice(0, 5) as topic, i (topic.tag)}
								<button
									type="button"
									class="hover-bg focus-ring group/row -mx-2 flex w-full items-center gap-2.5 rounded-sm px-2 py-2 text-left transition-colors first:pt-0 last:pb-0"
									aria-label={`搜索话题 #${topic.tag}`}
									onclick={() => void goto(`${ROUTES.search}?q=${encodeURIComponent('#' + topic.tag)}`)}
								>
									<span class="w-4 shrink-0 text-xxs font-medium tabular-nums text-text-secondary">{String(i + 1).padStart(2, '0')}</span>
									<span class="min-w-0 flex-1">
										<span class="block truncate text-sm font-medium text-text-primary transition-colors group-hover/row:text-accent-default">#{topic.tag}</span>
										<span class="mt-0.5 block truncate text-xxs text-text-secondary">{topic.desc}</span>
									</span>
									<span class="shrink-0 text-xs tabular-nums text-text-secondary">{topic.count} 条</span>
								</button>
							{/each}
						</div>
					</RailSection>

					{#if data.railNearbyMoments.length > 0}
						<RailSection
							title="附近活动"
							icon={MapPin}
							action="去附近"
							onAction={() => void goto(ROUTES.nearby)}
							variant="island"
						>
							<div class="flex flex-col divide-y divide-border-subtle/30">
								{#each data.railNearbyMoments as item (item.id)}
									<div class="py-2 first:pt-0 last:pb-0">
										<RailMomentThumb
											image={item.image}
											title={item.title}
											meta={item.meta}
											onClick={() => void goto(ROUTES.moment(item.id))}
										/>
									</div>
								{/each}
							</div>
						</RailSection>
					{/if}
				</div>
			{/if}
		{/snippet}
	</SplitPageLayout>
</div>
