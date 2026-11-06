<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, Search, Compass, ArrowRight, Sparkles, Users, LogIn } from '@lucide/svelte';
	import { auth } from '@/stores/auth.svelte';
	import { ROUTES } from '@/lib/routes';
	import { contentFade } from '@/lib/motion';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import HomeFeedItem from '@/components/home/HomeFeedItem.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InfiniteSentinel from '@/components/list/InfiniteSentinel.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import Button from '@/components/ui/button.svelte';
	import imgHotSpot from '@/assets/hotspot-logo.png';
	import type { HomeFeedData } from '../shared/home-feed-data.svelte';

	let { data }: { data: HomeFeedData } = $props();
</script>

<div class="flex flex-col flex-1 min-h-full pb-36 px-4 max-w-4xl mx-auto w-full">
	<!-- 平板端顶栏：适度放大触控热区，宽屏居中，不占全宽大留白 -->
	<div class="bg-surface-page/80 backdrop-blur-md sticky top-0 z-20 pb-3 pt-2">
		<div class="flex items-center justify-between py-2">
			<div class="flex items-center gap-2.5">
				<img src={imgHotSpot} decoding="async" alt="HotSpot" class="size-7 rounded-full object-cover shadow-xs" />
				<span class="text-lg font-bold tracking-tight text-text-primary">HotSpot</span>
			</div>
			<h1 class="sr-only">首页</h1>
			<div class="flex items-center gap-2">
				<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} variant="ghost" size={36} />
				<ActionIconButton icon={Bell} label="通知" onClick={() => void goto(ROUTES.notify)} variant="ghost" size={36} />
			</div>
		</div>
		<div class="flex justify-center mt-1">
			<TabBar
				variant="segmented"
				size="md"
				items={[
					{ id: 'discover', label: '为你推荐' },
					{ id: 'following', label: '正在关注' }
				]}
				value={data.activeTab}
				onChange={(id) => data.setTab(id as any)}
				class="w-full max-w-sm"
			/>
		</div>
	</div>

	<div use:contentFade={data.currentLoading} class="min-h-[50vh] flex flex-col justify-between">
		{#if data.activeTab === 'following' && auth.status !== 'authenticated'}
			<div class="my-8 rounded-lg bg-surface-island p-8 text-center shadow-xs">
				<div class="mx-auto flex size-14 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft mb-3.5">
					<Users class="size-7" strokeWidth={1.75} />
				</div>
				<h2 class="text-lg font-semibold text-text-primary">登录后查看关注动态</h2>
				<p class="mt-1 text-sm text-text-secondary max-w-md mx-auto">
					关注同频的好友与活跃邻居，即时获取 TA 们的最新瞬间与生活记录。
				</p>
				<Button type="button" size="lg" class="mt-5" onclick={() => void goto(ROUTES.login)}>
					<LogIn class="size-4" />
					<span>立即登录</span>
				</Button>
			</div>
		{:else if data.currentLoading}
			<SkeletonFeed
				variant="feed"
				count={4}
				label="加载动态中…"
				class="grid grid-cols-2 gap-4 pt-3"
			/>
		{:else if data.currentError && data.currentItems.length === 0}
			<LoadError message={data.currentError} onRetry={() => void data.reload()} />
		{:else if data.currentItems.length > 0}
			{#if data.currentItems.length === 1}
				<div class="mx-auto w-full max-w-md pt-3">
					{#each data.currentItems as moment (moment.moment_id)}
						<HomeFeedItem
							{moment}
							showDistance={false}
							priority={true}
							onClick={() => void goto(ROUTES.moment(moment.moment_id))}
						/>
					{/each}
				</div>
			{:else}
				<!-- 平板端稳定 2 列（宽平板 3 列）规整排列，保证卡片图幅视觉张力 -->
				<div class="grid grid-cols-2 gap-4 pt-3">
					{#each data.currentItems as moment, index (moment.moment_id)}
						<HomeFeedItem
							{moment}
							showDistance={false}
							priority={index < 2}
							onClick={() => void goto(ROUTES.moment(moment.moment_id))}
						/>
					{/each}
				</div>
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
				<div class="my-4">
					<div class="rounded-lg bg-surface-island p-5 shadow-xs">
						<div class="flex items-center justify-between mb-3">
							<div class="flex items-center gap-2">
								<Sparkles class="size-4.5 text-accent-default" strokeWidth={1.75} />
								<span class="text-sm font-semibold text-text-primary">探索更多感兴趣的话题</span>
							</div>
							<span class="text-xs text-text-secondary">社区热议</span>
						</div>
						<div class="flex flex-wrap gap-2">
							{#each data.trendingTopics.slice(0, 4) as topic (topic.tag)}
								<TagPill
									size="md"
									onClick={() => void goto(`${ROUTES.search}?q=${encodeURIComponent('#' + topic.tag)}`)}
								>
									<span>#{topic.tag}</span>
									<span class="text-xs opacity-70">({topic.count})</span>
								</TagPill>
							{/each}
						</div>
					</div>
				</div>
			{/if}

			<div class="mt-5">
				<button
					type="button"
					class="hover-btn group flex w-full items-center gap-4 rounded-lg bg-surface-island p-5 text-left focus-ring transition hover:shadow-raised-sm shadow-xs"
					onclick={() => void goto(ROUTES.nearby)}
				>
					<span class="flex size-12 shrink-0 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft transition-transform group-hover:scale-105">
						<Compass class="size-6" strokeWidth={1.75} />
					</span>
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-1.5">
							<span class="text-md font-semibold text-text-primary">看看附近正在发生什么</span>
						</div>
						<p class="truncate text-xs text-text-secondary mt-1">附近的人 · 群组 · 瞬间，5km 内实时雷达</p>
					</div>
					<ArrowRight class="size-5 shrink-0 text-text-secondary transition-transform group-hover:translate-x-0.5 group-hover:text-accent-default" strokeWidth={1.75} />
				</button>
			</div>
		{:else}
			<EmptyState
				title={data.activeTab === 'following' ? '暂无关注动态' : '还没有动态'}
				description={data.activeTab === 'following' ? '去「为你推荐」发现更多有趣的人，或去附近看看身边正在发生什么。' : '去附近看看，或者发布属于你的第一条瞬间。'}
				bare
				spotlight
			>
				{#snippet icon()}
					<Sparkles class="size-7 opacity-subtle text-text-secondary" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>

			<div class="mt-5">
				<button
					type="button"
					class="hover-btn group flex w-full items-center gap-4 rounded-lg bg-surface-island p-5 text-left focus-ring transition hover:shadow-raised-sm shadow-xs"
					onclick={() => void goto(ROUTES.nearby)}
				>
					<span class="flex size-12 shrink-0 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft transition-transform group-hover:scale-105">
						<Compass class="size-6" strokeWidth={1.75} />
					</span>
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-1.5">
							<span class="text-md font-semibold text-text-primary">看看附近正在发生什么</span>
						</div>
						<p class="truncate text-xs text-text-secondary mt-1">附近的人 · 群组 · 瞬间，5km 内实时雷达</p>
					</div>
					<ArrowRight class="size-5 shrink-0 text-text-secondary transition-transform group-hover:translate-x-0.5 group-hover:text-accent-default" strokeWidth={1.75} />
				</button>
			</div>
		{/if}
	</div>
</div>
