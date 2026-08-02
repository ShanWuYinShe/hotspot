<script lang="ts">
	import { goto } from '$app/navigation';
	import { Heart, MessageCircle, CalendarDays, ArrowUpRight, Grid3X3 } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import { openComposer } from '@/stores/composer.svelte';
	import { useGalleryPosts, useReactedPosts, useFavoritePosts, type GalleryPost } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 我的瞬间分区（对齐 Figma Make ProfileSections SectionMoments）。
	// 编辑式档案布局：mono eyebrow + 大标题 + 描述 + 归档统计 + pill tab +
	// 大特色精选卡（16/10 + 标题段落 + ArrowUpRight）+ 近期条目 4-col grid（item0 span 2）。

	type Tab = 'posts' | 'liked' | 'saved';

	let activeTab = $state<Tab>('posts');

	const myPosts = useGalleryPosts('my');
	const reacted = useReactedPosts();
	const savedP = useFavoritePosts();

	const postsMap = $derived({ posts: myPosts.posts, liked: reacted.posts, saved: savedP.posts } as Record<Tab, GalleryPost[]>);
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
	const captions = [
		'湖边的慢光', '回龙窝的午后', '留给傍晚的颜色', '骑过旧城区', '雨后小巷', '一场没有终点的散步', '相机里的朋友', '夜晚仍在继续'
	];
</script>

<div class="no-scrollbar flex min-h-0 flex-1 flex-col overflow-y-auto">
	<div class="mx-auto w-full max-w-[1180px] px-5 pb-10 pt-7 sm:px-7 lg:px-10">
		{#if archive.length === 0}
			<!-- 空态 -->
			<EmptyState title="这里还没有瞬间" description="从一次路过、一次相遇开始记录。">
				{#snippet icon()}
					<Grid3X3 size={20} />
				{/snippet}
				{#if activeTab === 'posts'}
					{#snippet action()}
						<button type="button" onclick={openComposer} class="rounded-full bg-[var(--neu-accent-primary)] px-4 py-2 text-xs font-medium text-[var(--text-on-accent)] neu-focus-ring">
							发布第一条
						</button>
					{/snippet}
				{/if}
			</EmptyState>
		{:else}
			<!-- 编辑式档案头部（对齐 Figma SectionMoments header） -->
			<header class="border-b pb-6" style="border-color:var(--neu-border);">
				<div class="flex flex-wrap items-end justify-between gap-5">
					<div>
						<p class="font-mono text-[10px] font-semibold uppercase tracking-[0.16em] text-[var(--neu-accent-primary)]">Personal archive / 2024</p>
						<h2 class="mt-2 text-[30px] font-medium leading-none tracking-[-0.02em] text-[var(--neu-text)]">个人影像档案</h2>
						<p class="mt-3 max-w-[460px] text-sm leading-6 text-[var(--neu-text-muted)]">{activeMeta[activeTab].description}</p>
					</div>
					<div class="flex items-center gap-5 border-l pl-5" style="border-color:var(--neu-border);">
						<div>
							<p class="font-mono text-xl font-semibold leading-none text-[var(--neu-text)]">{archive.length}</p>
							<p class="mt-1 text-[11px] text-[var(--neu-text-muted)]">{activeMeta[activeTab].label}</p>
						</div>
						<div>
							<p class="font-mono text-xl font-semibold leading-none text-[var(--neu-text)]">27</p>
							<p class="mt-1 text-[11px] text-[var(--neu-text-muted)]">到过的地方</p>
						</div>
					</div>
				</div>

				<!-- pill tab（对齐 Figma：rounded-full border bg-secondary，active=bg-card+shadow） -->
				<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
					<div class="flex rounded-full border p-1" style="border-color:var(--neu-border);background:var(--neu-accent-soft);" role="tablist" aria-label="瞬间分类">
						{#each tabItems as item (item.id)}
							{@const isActive = item.id === activeTab}
							<button
								type="button"
								role="tab"
								aria-selected={isActive}
								onclick={() => (activeTab = item.id)}
								class="rounded-full px-3.5 py-2 text-xs transition-colors {isActive ? 'bg-[var(--neu-bg)] text-[var(--neu-text)]' : 'text-[var(--neu-text-muted)] hover:text-[var(--neu-text)]'}"
								style={isActive ? 'box-shadow:0 2px 8px var(--neu-shadow-dark);' : ''}
							>
								{item.label}<span class="ml-1.5 font-mono text-[10px] opacity-70">{item.count}</span>
							</button>
						{/each}
					</div>
					<div class="flex items-center gap-1.5 text-[11px] text-[var(--neu-text-muted)]">
						<CalendarDays class="size-3.5 text-[var(--neu-accent-primary)]" strokeWidth={1.7} />
						<span>{activeMeta[activeTab].note}</span>
					</div>
				</div>
			</header>

			<div class="pt-7">
				<!-- 特色精选卡（对齐 Figma Featured record：16/10 图 + 标题段落 + ArrowUpRight） -->
				{#if featured}
					<button
						type="button"
						onclick={() => void goto(`${ROUTES.moment(featured.id)}?from=me`)}
						class="group block w-full overflow-hidden rounded-feature border text-left transition-colors hover:border-[var(--neu-accent-soft-border)]"
						style="border-color:var(--neu-border);background:var(--neu-bg);"
						aria-label="查看精选瞬间"
					>
						<div class="lg:grid lg:grid-cols-[minmax(0,1.38fr)_minmax(280px,0.62fr)]">
							<div class="image-hover-frame relative aspect-[16/10] overflow-hidden lg:aspect-auto lg:min-h-[390px]" style="background:var(--page-bg);">
								<img src={featured.img} alt="本期精选瞬间" class="image-hover-frame__image h-full w-full object-cover" />
							</div>
							<div class="flex flex-col justify-between border-t p-6 lg:border-l lg:border-t-0" style="border-color:var(--neu-border);">
								<div>
									<div class="flex items-center justify-between">
										<span class="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--neu-accent-primary)]">Featured record</span>
										<span class="text-[11px] text-[var(--neu-text-muted)]">06.18</span>
									</div>
									<h3 class="mt-5 text-[24px] font-medium leading-snug text-[var(--neu-text)]">在云龙湖边，<br />把傍晚收进相册。</h3>
									<p class="mt-4 text-sm leading-6 text-[var(--neu-text-muted)]">风停下来以后，水面留下了比语言更完整的回答。今天没有目的地，只是沿着湖岸慢慢走。</p>
								</div>
								<div class="mt-8 flex items-center justify-between border-t pt-4" style="border-color:var(--neu-border);">
									<div class="flex items-center gap-4 text-[11px] text-[var(--neu-text-muted)]">
										<span class="flex items-center gap-1"><Heart class="size-3 text-[var(--neu-accent-primary)]" strokeWidth={1.8} />{featured.likes}</span>
										<span class="flex items-center gap-1"><MessageCircle class="size-3 text-[var(--neu-accent-primary)]" strokeWidth={1.8} />{featured.comments}</span>
									</div>
									<span class="flex size-8 items-center justify-center rounded-full text-[var(--neu-accent-primary)]" style="background:var(--neu-accent-soft);" aria-hidden="true">
										<ArrowUpRight class="size-4" strokeWidth={1.8} />
									</span>
								</div>
							</div>
						</div>
					</button>
				{/if}

				<!-- 近期条目（对齐 Figma Recent entries：4-col grid，item0 span 2） -->
				{#if remaining.length > 0}
					<div class="mt-10 flex items-end justify-between border-b pb-3" style="border-color:var(--neu-border);">
						<div>
							<p class="font-mono text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--neu-accent-primary)]">Recent entries</p>
							<h3 class="mt-1 text-base font-medium text-[var(--neu-text)]">六月的城市笔记</h3>
						</div>
						<p class="text-[11px] text-[var(--neu-text-muted)]">{remaining.length} 个片段</p>
					</div>

					<div class="mt-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
						{#each remaining as post, index (post.id)}
							<button
								type="button"
								onclick={() => void goto(`${ROUTES.moment(post.id)}?from=me`)}
								class="group overflow-hidden rounded-control border text-left transition-colors hover:border-[var(--neu-accent-soft-border)] {index === 0 ? 'col-span-2' : ''}"
								style="border-color:var(--neu-border);background:var(--neu-bg);"
								aria-label="查看瞬间详情"
							>
								<div class="image-hover-frame overflow-hidden {index === 0 ? 'aspect-[16/8]' : 'aspect-square'}" style="background:var(--page-bg);">
									<img src={post.img} alt={captions[index % captions.length]} class="image-hover-frame__image h-full w-full object-cover" />
								</div>
								<div class="flex items-center justify-between gap-2 px-3 py-2.5">
									<p class="truncate text-xs text-[var(--neu-text)]">{captions[index % captions.length]}</p>
									<div class="flex shrink-0 items-center gap-2 text-[10px] text-[var(--neu-text-muted)]">
										<span class="flex items-center gap-0.5"><Heart class="size-3" strokeWidth={1.7} />{post.likes}</span>
										<span class="flex items-center gap-0.5"><MessageCircle class="size-3" strokeWidth={1.7} />{post.comments}</span>
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
