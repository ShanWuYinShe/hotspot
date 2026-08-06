<script lang="ts">
	import { goto } from '$app/navigation';
	import { Heart, MessageCircle, CalendarDays, ArrowUpRight, Grid3X3 } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
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
	const featured = $derived(archive[0]);
	const remaining = $derived(archive.slice(1));

	const activeMeta: Record<Tab, { label: string; description: string; note: string }> = {
		posts: { label: '已发布', description: '把正在发生的事，留成可回看的片段。', note: '本年 48 则记录' },
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
     纯内容组件：滚动与 gutter 由外层 StandardPageLayout/ContentColumn 承担，不自建容器。 -->
<div class="pb-10 pt-7">
		{#if archive.length === 0}
			<!-- 空态 -->
			<EmptyState title="这里还没有瞬间" description="从一次路过、一次相遇开始记录。">
				{#snippet icon()}
					<Grid3X3 size={20} />
				{/snippet}
				{#snippet action()}
					{#if activeTab === 'posts'}
						<button type="button" onclick={openComposer} class="rounded-full bg-accent-default px-4 py-2 text-xs font-medium text-text-on-accent focus-ring">
							发布第一条
						</button>
					{/if}
				{/snippet}
			</EmptyState>
		{:else}
			<!-- 编辑式档案头部（对齐 Figma SectionMoments header） -->
			<header class="border-b pb-6" style="border-color:var(--cst-border-default);">
				<div class="flex flex-wrap items-end justify-between gap-5">
					<div>
						<p class="font-mono text-xxs font-semibold uppercase tracking-[0.16em] text-accent-default">Personal archive / 2024</p>
						<h2 class="mt-2 text-3xl font-medium leading-none tracking-[-0.02em] text-text-primary">个人影像档案</h2>
						<p class="mt-3 max-w-115 text-sm leading-6 text-text-secondary">{activeMeta[activeTab].description}</p>
					</div>
					<div class="flex items-center gap-5 border-l pl-5" style="border-color:var(--cst-border-default);">
						<div>
							<p class="font-mono text-xl font-semibold leading-none text-text-primary">{archive.length}</p>
							<p class="mt-1 text-xs text-text-secondary">{activeMeta[activeTab].label}</p>
						</div>
						<div>
							<p class="font-mono text-xl font-semibold leading-none text-text-primary">27</p>
							<p class="mt-1 text-xs text-text-secondary">到过的地方</p>
						</div>
					</div>
				</div>

				<!-- pill tab（对齐 Figma：rounded-full border bg-secondary，active=bg-card+shadow） -->
				<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
					<div class="flex rounded-full border p-1" style="border-color:var(--cst-border-default);background:var(--cst-accent-soft);" role="tablist" aria-label="瞬间分类">
						{#each tabItems as item (item.id)}
							{@const isActive = item.id === activeTab}
							<button
								type="button"
								role="tab"
								aria-selected={isActive}
								onclick={() => (activeTab = item.id)}
								class="rounded-full px-3.5 py-2 text-xs transition-colors {isActive ? 'bg-surface-primary text-text-primary' : 'text-text-secondary hover:text-text-primary'}"
								style={isActive ? 'box-shadow:0 2px 8px var(--cst-elevation-shadow-dark);' : ''}
							>
								{item.label}<span class="ml-1.5 font-mono text-xxs opacity-70">{item.count}</span>
							</button>
						{/each}
					</div>
					<div class="flex items-center gap-1.5 text-xs text-text-secondary">
						<CalendarDays class="size-3.5 text-accent-default" strokeWidth={1.7} />
						<span>{activeMeta[activeTab].note}</span>
					</div>
				</div>
			</header>

			<div class="pt-7">
				<!-- 特色精选卡（对齐 Figma Featured record：16/10 图 + 标题段落 + ArrowUpRight）。
				     hover 反馈（image-hover-frame 边框 + 图片缩放）绑定整卡按钮而非图片容器：
				     鼠标在卡片任意位置（含文本区）反馈持续。 -->
				{#if featured}
					<button
						type="button"
						onclick={() => void goto(`${ROUTES.moment(featured.id)}?from=me`)}
						class="group image-hover-frame relative block w-full overflow-hidden rounded-feature border text-left transition-colors"
						style="border-color:var(--cst-border-default);background:var(--cst-surface-primary);"
						aria-label="查看精选瞬间"
					>
						<div class="lg:grid lg:grid-cols-[minmax(0,1.38fr)_minmax(280px,0.62fr)]">
							<div class="tap-97 relative aspect-[16/10] overflow-hidden lg:aspect-auto lg:min-h-97.5" style="background:var(--cst-surface-page);">
								<img src={featured.img} alt="本期精选瞬间" class="image-hover-frame__image h-full w-full object-cover" />
							</div>
							<div class="flex flex-col justify-between border-t p-6 lg:border-l lg:border-t-0" style="border-color:var(--cst-border-default);">
								<div>
									<div class="flex items-center justify-between">
										<span class="font-mono text-xxs font-semibold uppercase tracking-[0.14em] text-accent-default">Featured record</span>
										<span class="text-xs text-text-secondary">06.18</span>
									</div>
									<h3 class="mt-5 text-2xl font-medium leading-snug text-text-primary">在云龙湖边，<br />把傍晚收进相册。</h3>
									<p class="mt-4 text-sm leading-6 text-text-secondary">风停下来以后，水面留下了比语言更完整的回答。今天没有目的地，只是沿着湖岸慢慢走。</p>
								</div>
								<div class="mt-8 flex items-center justify-between border-t pt-4" style="border-color:var(--cst-border-default);">
									<div class="flex items-center gap-4 text-xs text-text-secondary">
										<span class="flex items-center gap-1"><Heart class="size-3 text-accent-default" strokeWidth={1.8} />{featured.likes}</span>
										<span class="flex items-center gap-1"><MessageCircle class="size-3 text-accent-default" strokeWidth={1.8} />{featured.comments}</span>
									</div>
									<span class="flex size-8 items-center justify-center rounded-full text-accent-default" style="background:var(--cst-accent-soft);" aria-hidden="true">
										<ArrowUpRight class="size-4" strokeWidth={1.8} />
									</span>
								</div>
							</div>
						</div>
					</button>
				{/if}

				<!-- 近期条目（对齐 Figma Recent entries：4-col grid，item0 span 2） -->
				{#if remaining.length > 0}
					<div class="mt-10 flex items-end justify-between border-b pb-3" style="border-color:var(--cst-border-default);">
						<div>
							<p class="font-mono text-xxs font-semibold uppercase tracking-[0.14em] text-accent-default">Recent entries</p>
							<h3 class="mt-1 text-base font-medium text-text-primary">六月的城市笔记</h3>
						</div>
						<p class="text-xs text-text-secondary">{remaining.length} 个片段</p>
					</div>

					<div class="mt-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
				{#each remaining as moment, index (moment.id)}
					<button
						type="button"
						onclick={() => void goto(`${ROUTES.moment(moment.id)}?from=me`)}
						class="group image-hover-frame relative overflow-hidden rounded-control border text-left transition-colors {index === 0 ? 'col-span-2' : ''}"
						style="border-color:var(--cst-border-default);background:var(--cst-surface-primary);"
						aria-label="查看瞬间详情"
					>
						<div class="tap-97 relative overflow-hidden {index === 0 ? 'aspect-[16/8]' : 'aspect-square'}" style="background:var(--cst-surface-page);">
									<img src={moment.img} alt={moment.title || '瞬间图片'} class="image-hover-frame__image h-full w-full object-cover" />
								</div>
								<div class="flex items-center justify-between gap-2 px-3 py-2.5">
									<p class="truncate text-xs text-text-primary">{moment.title || '瞬间'}</p>
									<div class="flex shrink-0 items-center gap-2 text-xxs text-text-secondary">
										<span class="flex items-center gap-0.5"><Heart class="size-3" strokeWidth={1.7} />{moment.likes}</span>
										<span class="flex items-center gap-0.5"><MessageCircle class="size-3" strokeWidth={1.7} />{moment.comments}</span>
									</div>
								</div>
							</button>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
</div>
