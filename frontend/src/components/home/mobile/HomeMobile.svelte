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

<div class="flex flex-col flex-1 min-h-full pb-36">
	<!-- 移动端顶栏：品牌 Logo + 快捷动作 + 居中胶囊分段器一体化 -->
	<div class="bg-surface-page/80 backdrop-blur-md sticky top-0 z-20 pb-2">
		<div class="flex items-center justify-between px-4 pt-3 pb-2">
			<div class="flex items-center gap-2">
				<img src={imgHotSpot} decoding="async" alt="HotSpot" class="size-6.5 rounded-full object-cover shadow-xs" />
				<span class="text-md font-bold tracking-tight text-text-primary">HotSpot</span>
			</div>
			<h1 class="sr-only">首页</h1>
			<div class="flex items-center gap-3">
				<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} variant="ghost" size={36} />
				<ActionIconButton icon={Bell} label="通知" onClick={() => void goto(ROUTES.notify)} variant="ghost" size={36} />
			</div>
		</div>
		<div class="flex justify-center px-4">
			<TabBar
				variant="segmented"
				size="sm"
				items={[
					{ id: 'discover', label: '为你推荐' },
					{ id: 'following', label: '正在关注' }
				]}
				value={data.activeTab}
				onChange={(id) => data.setTab(id as any)}
				class="w-full max-w-[13.75rem]"
			/>
		</div>
	</div>

	<div use:contentFade={data.currentLoading} class="min-h-[50vh] flex flex-col justify-between">
		{#if data.activeTab === 'following' && auth.status !== 'authenticated'}
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
				class="grid grid-cols-2 gap-2.5 px-4 pt-2"
			/>
		{:else if data.currentError && data.currentItems.length === 0}
			<LoadError message={data.currentError} onRetry={() => void data.reload()} />
		{:else if data.currentItems.length > 0}
			{#if data.currentItems.length === 1}
				<div class="mx-auto w-full max-w-md px-4 pt-2">
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
				<div class="grid grid-cols-2 gap-2.5 px-4 pt-2">
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
				<div class="px-3.5 my-3">
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

			<div class="px-3.5 mt-4">
				<button
					type="button"
					class="hover-btn group flex w-full items-center gap-3.5 rounded-lg bg-surface-island px-4 py-4 text-left focus-ring transition hover:shadow-raised-sm shadow-xs"
					onclick={() => void goto(ROUTES.nearby)}
				>
					<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft transition-transform group-hover:scale-105">
						<Compass class="size-5" strokeWidth={1.75} />
					</span>
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-1.5">
							<span class="text-sm font-semibold text-text-primary">看看附近正在发生什么</span>
						</div>
						<p class="truncate text-xs text-text-secondary mt-0.5">附近的人 · 群组 · 瞬间，5km 内实时雷达</p>
					</div>
					<ArrowRight class="size-4 shrink-0 text-text-secondary transition-transform group-hover:translate-x-0.5 group-hover:text-accent-default" strokeWidth={1.75} />
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
					<Sparkles class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>

			<div class="px-3.5 mt-4">
				<button
					type="button"
					class="hover-btn group flex w-full items-center gap-3.5 rounded-lg bg-surface-island px-4 py-4 text-left focus-ring transition hover:shadow-raised-sm shadow-xs"
					onclick={() => void goto(ROUTES.nearby)}
				>
					<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft transition-transform group-hover:scale-105">
						<Compass class="size-5" strokeWidth={1.75} />
					</span>
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-1.5">
							<span class="text-sm font-semibold text-text-primary">看看附近正在发生什么</span>
						</div>
						<p class="truncate text-xs text-text-secondary mt-0.5">附近的人 · 群组 · 瞬间，5km 内实时雷达</p>
					</div>
					<ArrowRight class="size-4 shrink-0 text-text-secondary transition-transform group-hover:translate-x-0.5 group-hover:text-accent-default" strokeWidth={1.75} />
				</button>
			</div>
		{/if}
	</div>
</div>
