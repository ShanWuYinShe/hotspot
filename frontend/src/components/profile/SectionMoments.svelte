<script lang="ts">
	import { goto } from '$app/navigation';
	import { Heart, MessageCircle, CalendarDays, ArrowUpRight, Grid3X3 } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import MediaPlaceholder from '@/components/data-display/MediaPlaceholder.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { openComposer } from '@/stores/composer.svelte';
	import { useGalleryMoments, useReactedMoments, useFavoriteMoments, type GalleryMoment } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 我的瞬间分区（对齐 Figma Make ProfileSections SectionMoments）。
	// 编辑式档案布局：mono eyebrow + 大标题 + 描述 + 归档统计 + pill tab +
	// 大特色精选卡（16/10 + 标题段落 + ArrowUpRight）+ 近期条目 4-col grid（item0 span 2）。

	type Tab = 'posts' | 'liked' | 'saved';

	let activeTab = $state<Tab>('posts');

	const myPosts = useGalleryMoments('my');
	const reacted = useReactedMoments();
	const savedP = useFavoriteMoments();

	const postsMap = $derived({ posts: myPosts.posts, liked: reacted.posts, saved: savedP.posts } as Record<Tab, GalleryMoment[]>);
	const archive = $derived(postsMap[activeTab]);
	const loading = $derived(activeTab === 'posts' ? myPosts.loading : activeTab === 'liked' ? reacted.loading : savedP.loading);
	const error = $derived(activeTab === 'posts' ? myPosts.error : activeTab === 'liked' ? reacted.error : savedP.error);
	const retryActive = () => {
		if (activeTab === 'posts') myPosts.retry();
		else if (activeTab === 'liked') reacted.retry();
		else savedP.retry();
	};
	const featured = $derived(archive[0]);
	const remaining = $derived(archive.slice(1));
	// 年份取档案内最新一条的创建年份，避免硬编码过期
	const archiveYear = $derived(
		featured?.created_at ? new Date(featured.created_at).getFullYear() : new Date().getFullYear()
	);

	const activeMeta: Record<Tab, { label: string; description: string; note: string }> = {
		posts: { label: '已发布', description: '把正在发生的事，留成可回看的片段。', note: '已发布记录' },
		liked: { label: '喜欢', description: '被光线、路途和陌生叙事打动的瞬间。', note: '最近收藏的灵感' },
		saved: { label: '收藏', description: '留给以后慢慢打开的城市线索。', note: '可随时继续整理' }
	};
	const tabItems: Array<{ id: Tab; label: string; count: number }> = $derived([
		{ id: 'posts', label: '发布', count: myPosts.posts.length },
		{ id: 'liked', label: '喜欢', count: reacted.posts.length },
		{ id: 'saved', label: '收藏', count: savedP.posts.length }
	]);
</script>

<!-- 我的瞬间分区（对齐 Figma Make ProfileSections SectionMoments）。
     纯内容组件：滚动与 gutter 由外层 StandardPageLayout/ContentColumn 承担，不自建容器。
     顶部/底部间距均由外层 contentClassName（pt-4 pb-6）承担——分区内不叠 pb，
     否则各分区尾部留白随自身 pb 各异（曾出现 48/56/64px 三种值）。 -->
<div>
		{#if loading}
			<div class="flex flex-col items-center gap-3 py-20">
				<div class="spinner size-7"></div>
				<p class="text-text-secondary text-sm">加载我的瞬间…</p>
			</div>
		{:else if error}
			<LoadError message={error} onRetry={retryActive} />
		{:else if archive.length === 0}
			<!-- 空态 -->
			<EmptyState title="这里还没有瞬间" description="从一次路过、一次相遇开始记录。">
				{#snippet icon()}
					<Grid3X3 size={20} />
				{/snippet}
				{#snippet action()}
					{#if activeTab === 'posts'}
						<button type="button" onclick={openComposer} class="hover-btn rounded-full bg-accent-default px-4 py-2 text-sm font-medium text-text-on-accent focus-ring">
							发布第一条
						</button>
					{/if}
				{/snippet}
			</EmptyState>
		{:else}
			<!-- 编辑式档案头部（对齐 Figma SectionMoments header） -->
			<header class="border-b border-border-default pb-6">
				<div class="flex flex-wrap items-end justify-between gap-5">
					<div>
						<!-- 年份取档案内最新一条的创建年份，避免硬编码过期 -->
						<p class="eyebrow font-semibold text-accent-default font-mono">Personal archive / {archiveYear}</p>
						<h2 class="mt-2 text-3xl font-medium leading-none tracking-title text-text-primary">个人影像档案</h2>
						<p class="mt-3 max-w-115 text-sm leading-6 text-text-secondary">{activeMeta[activeTab].description}</p>
					</div>
					<div class="flex items-center gap-5 border-l border-border-default pl-5">
						<div>
							<p class="font-mono text-xl font-semibold leading-none text-text-primary">{archive.length}</p>
							<p class="mt-1 text-xs text-text-secondary">{activeMeta[activeTab].label}</p>
						</div>
					</div>
				</div>

				<!-- 分类 Tab 统一走 TabBar（胶囊） -->
				<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
					<div class="min-w-0 flex-1" aria-label="瞬间分类">
						<TabBar items={tabItems.map((t) => ({ id: t.id, label: t.label, count: t.count }))} value={activeTab} onChange={(id) => (activeTab = id as Tab)} />
					</div>
					<div class="flex items-center gap-1.5 text-xs text-text-secondary">
						<CalendarDays class="size-3.5 text-accent-default" strokeWidth={1.75} />
						<span>{activeTab === 'posts' ? `本年 ${archive.length} 则记录` : activeMeta[activeTab].note}</span>
					</div>
				</div>
			</header>

			<div class="pt-7">
				<!-- 特色精选卡（对齐 Figma Featured record：16/10 图 + 标题段落 + ArrowUpRight）。
				     hover 反馈（外边缘 accent 边框）绑定整卡按钮而非图片容器：
				     鼠标在卡片任意位置（含文本区）反馈持续。 -->
				{#if featured}
					<!-- 整卡跳转动态详情（大图查看在详情页内），原生 button 承载键盘/焦点语义 -->
					<button
						type="button"
						onclick={() => void goto(`${ROUTES.moment(featured.id)}?from=me`)}
						class="relative block w-full overflow-hidden rounded-lg border border-border-default bg-surface-primary shadow-raised-md text-left transition-colors hover:border-accent-default focus-ring"
						aria-label="查看精选瞬间"
					>
						<div class="lg:grid lg:grid-cols-[minmax(0,1.38fr)_minmax(280px,0.62fr)]">
							<div class="tap-97 relative aspect-[16/10] overflow-hidden lg:aspect-auto lg:min-h-97.5 bg-surface-page" >
								{#if featured.img}
									<img src={featured.img} alt="本期精选瞬间" loading="lazy" class="h-full w-full object-cover" />
								{:else}
									<!-- 无图瞬间占位（禁止 <img src=""> 破图） -->
									<MediaPlaceholder iconSize={28} label="无图瞬间" />
								{/if}
							</div>
							<div class="flex flex-col justify-between border-t border-border-default p-6 lg:border-l lg:border-t-0">
								<div>
									<div class="flex items-center justify-between">
										<span class="eyebrow font-semibold text-accent-default font-mono">Featured record</span>
									</div>
									<h3 class="mt-5 text-2xl font-medium leading-snug text-text-primary">{featured.title || '精选瞬间'}</h3>
								</div>
									<div class="mt-8 flex items-center justify-between border-t border-border-default pt-4">
										<div class="flex items-center gap-4 text-xs text-text-secondary">
											<span class="flex items-center gap-1"><Heart class="size-3 text-accent-default" strokeWidth={1.75} />{featured.likes}</span>
											<span class="flex items-center gap-1"><MessageCircle class="size-3 text-accent-default" strokeWidth={1.75} />{featured.comments}</span>
										</div>
										<span class="flex size-8 items-center justify-center rounded-full text-accent-default bg-accent-soft"  aria-hidden="true">
											<ArrowUpRight class="size-4" strokeWidth={1.75} />
										</span>
									</div>
								</div>
							</div>
						</button>
					{/if}

				<!-- 近期条目（对齐 Figma Recent entries：4-col grid，item0 span 2） -->
				{#if remaining.length > 0}
						<div class="mt-10 flex items-end justify-between border-b border-border-default pb-3">
							<div>
								<p class="eyebrow font-semibold text-accent-default font-mono">Recent entries</p>
								<h3 class="mt-1 text-sm font-medium text-text-primary">近期记录</h3>
							</div>
							<p class="text-xs text-text-secondary">{remaining.length} 个片段</p>
						</div>

					<div class="mt-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
				{#each remaining as moment, index (moment.id)}
					<!-- 同 featured：整卡跳转动态详情 -->
					<button
						type="button"
						onclick={() => void goto(`${ROUTES.moment(moment.id)}?from=me`)}
						class="relative overflow-hidden rounded-md border border-border-default bg-surface-primary shadow-raised-md text-left transition-colors hover:border-accent-default focus-ring {index === 0 ? 'col-span-2' : ''}"
						aria-label="查看瞬间详情"
					>
						<div class="tap-97 relative overflow-hidden {index === 0 ? 'aspect-[16/8]' : 'aspect-square'} bg-surface-page" >
							{#if moment.img}
								<img src={moment.img} alt={moment.title || '瞬间图片'} loading="lazy" class="h-full w-full object-cover" />
							{:else}
								<MediaPlaceholder iconSize={18} />
							{/if}
						</div>
						<div class="flex items-center justify-between gap-2 px-3 py-2.5">
							<p class="truncate text-xs text-text-primary">{moment.title || '瞬间'}</p>
							<div class="flex shrink-0 items-center gap-2 text-xxs text-text-secondary">
								<span class="flex items-center gap-0.5"><Heart class="size-3" strokeWidth={1.75} />{moment.likes}</span>
								<span class="flex items-center gap-0.5"><MessageCircle class="size-3" strokeWidth={1.75} />{moment.comments}</span>
							</div>
						</div>
					</button>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

