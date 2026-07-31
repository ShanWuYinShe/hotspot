<script lang="ts">
	import { goto } from '$app/navigation';
	import { Grid3X3, Heart, MessageCircle } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import { openComposer } from '@/stores/composer.svelte';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import EmptyState from '@/components/shared/EmptyState.svelte';
	import ProfileTabs, { type Tab } from '@/components/profile/ProfileTabs.svelte';
	import { useGalleryPosts, useReactedPosts, useFavoritePosts, type GalleryPost } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 我的瞬间分区（迁移自旧版 ProfileSections 内 SectionMoments）。
	// 桌面 BSP 马赛克；移动/平板置顶卡 + 网格。

	let activeTab = $state<Tab>('posts');
	let visibleCount = $state(12);

	const myPosts = useGalleryPosts('my');
	const reacted = useReactedPosts();
	const savedP = useFavoritePosts();

	const postsMap = $derived({ posts: myPosts.posts, liked: reacted.posts, saved: savedP.posts } as Record<Tab, GalleryPost[]>);
	const activePosts = $derived(postsMap[activeTab]);

	const collectionMeta: Record<Tab, { eyebrow: string; title: string; detail: string }> = {
		posts: { eyebrow: '个人存档', title: '我的瞬间', detail: '你发布的每一次靠近与停留' },
		liked: { eyebrow: '留住灵感', title: '喜欢的瞬间', detail: '那些让你愿意再看一眼的片段' },
		saved: { eyebrow: '稍后再见', title: '收藏的瞬间', detail: '为下一次出发留下的线索' }
	};
	const current = $derived(collectionMeta[activeTab]);
	const featuredPost = $derived(activePosts[0]);
	const remainingPosts = $derived(activePosts.slice(1));
	const isDesktop = $derived(platformStore.platform === 'desktop');
</script>

<div class="flex flex-1 flex-col overflow-y-auto no-scrollbar">
	<header class="flex-none border-b border-[var(--neu-border)] px-6 pb-4 pt-5">
		<div class="flex items-start justify-between gap-5">
			<div class="min-w-0">
				<p class="neu-eyebrow">{current.eyebrow}</p>
				<div class="mt-1 flex items-baseline gap-2">
					<h1 class="text-[21px] font-medium tracking-[-0.01em] text-[var(--neu-text)]">{current.title}</h1>
					<span class="rounded-full border border-[var(--neu-accent-soft-border)] bg-[var(--neu-accent-soft)] px-2 py-0.5 text-[10px] font-medium text-[var(--neu-accent-primary)]">
						{activePosts.length} 条
					</span>
				</div>
				<p class="mt-1 text-[12px] text-[var(--neu-text-muted)]">{current.detail}</p>
			</div>
		</div>
	</header>

	<div class="flex-none px-5 pt-3">
		<ProfileTabs activeTab={activeTab} setActiveTab={(t) => (activeTab = t)} />
	</div>

	<div class="w-full px-6 pb-8 pt-5">
		{#if activePosts.length === 0}
			{#if activeTab === 'posts'}
				<EmptyState title="这里还没有瞬间" description="从一次路过、一次相遇开始记录。">
					{#snippet icon()}
						<Grid3X3 size={20} />
					{/snippet}
					{#snippet action()}
						<button type="button" onclick={openComposer} class="rounded-full bg-[var(--neu-accent-primary)] px-4 py-2 text-xs font-medium text-white neu-focus-ring">
							发布第一条
						</button>
					{/snippet}
				</EmptyState>
			{:else}
				<EmptyState title="这里还没有瞬间" description="从一次路过、一次相遇开始记录。">
					{#snippet icon()}
						<Grid3X3 size={20} />
					{/snippet}
				</EmptyState>
			{/if}
		{:else if isDesktop}
			<!-- Desktop: BSP mosaic -->
			<BspCardCanvas
				items={activePosts.slice(0, visibleCount)}
				gap={24}
				minRatio={0.68}
				maxRatio={1.48}
				minCardWidth={160}
				minCardHeight={232}
				minCanvasHeight={520}
				ariaLabel="我的瞬间的不规则满铺布局"
				hasMore={visibleCount < activePosts.length}
				onLoadMore={() => (visibleCount = Math.min(visibleCount + 8, activePosts.length))}
			>
				{#snippet renderItem(post: GalleryPost, index: number)}
					<button
						type="button"
						class="neu-hover-btn group relative h-full w-full rounded-[16px] ring-1 ring-[var(--neu-border)] bg-[var(--page-bg)] text-left neu-focus-ring focus-visible:ring-offset-[var(--neu-bg)] overflow-hidden"
						aria-label="查看瞬间详情"
						onclick={() => void goto(`${ROUTES.moment(post.id)}?from=me`)}
					>
						<div class="relative h-full w-full" style="overflow:hidden;border-radius:16px;">
							<img src={post.img} alt="" class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.035]" />
							<div class="absolute inset-x-0 bottom-0 bg-[linear-gradient(to_top,rgba(0,0,0,.62),transparent)] p-2.5">
								<div class="flex items-center gap-2 text-[10px] text-white">
									<span class="flex items-center gap-0.5"><Heart size={10} fill="currentColor" strokeWidth={0} />{post.likes}</span>
									<span class="flex items-center gap-0.5"><MessageCircle size={10} strokeWidth={1.6} />{post.comments}</span>
								</div>
							</div>
						</div>
					</button>
				{/snippet}
			</BspCardCanvas>
		{:else}
			<div class="animate-fade-in">
				{#if featuredPost}
					<button
						type="button"
						class="neu-hover-btn group relative block w-full rounded-[22px] ring-1 ring-[var(--neu-border)] bg-[var(--page-bg)] text-left neu-focus-ring focus-visible:ring-offset-[var(--neu-bg)] overflow-hidden"
						style="aspect-ratio:2.1 / 1;"
						aria-label="查看置顶瞬间详情"
						onclick={() => void goto(`${ROUTES.moment(featuredPost.id)}?from=me`)}
					>
						<div class="relative h-full w-full" style="overflow:hidden;border-radius:22px;">
							<img src={featuredPost.img} alt="" class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.025]" />
							<div class="absolute inset-0 bg-[linear-gradient(90deg,rgba(0,0,0,.62),rgba(0,0,0,.08)_65%,transparent)]"></div>
							<div class="absolute inset-x-0 bottom-0 flex items-end justify-between gap-4 p-4">
								<div>
									<p class="text-[10px] font-medium tracking-[0.12em] text-white/75">{activeTab === 'posts' ? '最近发布' : current.eyebrow}</p>
									<p class="mt-1 text-[15px] font-medium text-white">这一刻，值得被好好留下</p>
								</div>
								<div class="flex items-center gap-3 rounded-full bg-black/25 px-3 py-1.5 text-[11px] text-white backdrop-blur-sm">
									<span class="flex items-center gap-1"><Heart size={12} fill="currentColor" strokeWidth={0} />{featuredPost.likes}</span>
									<span class="flex items-center gap-1"><MessageCircle size={12} strokeWidth={1.7} />{featuredPost.comments}</span>
								</div>
							</div>
						</div>
					</button>

					<div class="mb-3 mt-6 flex items-center justify-between">
						<p class="neu-eyebrow">更多片段</p>
						<span class="text-[11px] text-[var(--neu-text-muted)]">按最近记录</span>
					</div>
					<div class="grid grid-cols-2 gap-3 sm:grid-cols-3">
						{#each remainingPosts as post (post.id)}
							<button
								type="button"
								class="neu-hover-btn group relative aspect-square rounded-[16px] ring-1 ring-[var(--neu-border)] bg-[var(--page-bg)] text-left neu-focus-ring focus-visible:ring-offset-[var(--neu-bg)] overflow-hidden"
								aria-label="查看瞬间详情"
								onclick={() => void goto(`${ROUTES.moment(post.id)}?from=me`)}
							>
								<div class="relative h-full w-full" style="overflow:hidden;border-radius:16px;">
									<img src={post.img} alt="" class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.035]" />
									<div class="absolute inset-x-0 bottom-0 bg-[linear-gradient(to_top,rgba(0,0,0,.62),transparent)] p-2.5">
										<div class="flex items-center gap-2 text-[10px] text-white">
											<span class="flex items-center gap-0.5"><Heart size={10} fill="currentColor" strokeWidth={0} />{post.likes}</span>
											<span class="flex items-center gap-0.5"><MessageCircle size={10} strokeWidth={1.6} />{post.comments}</span>
										</div>
									</div>
								</div>
							</button>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>
