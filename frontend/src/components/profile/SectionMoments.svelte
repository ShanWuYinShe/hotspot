<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { CalendarDays, Grid3X3, Sparkles } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import MomentCard from '@/components/moments/MomentCard.svelte';
	import { createFeedInteractions, type FeedItem } from '@/components/moments/moment-feed.svelte';
	import { openComposer } from '@/stores/composer.svelte';
	import { useAccountData, useGalleryMoments, useReactedMoments, useFavoriteMoments, type GalleryMoment } from '@/lib/profile-data.svelte';
	import { formatDateTime } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';

	// 我的瞬间分区：与 nearby/moments 全面统一视觉语言。
	// 共享 MomentCard 原语（封面 + 标签 + 作者 + 位置 + 正文 + 赞评藏操作栏），
	// 采用标准响应式网格（桌面 3 列 / 平板 2 列 / 移动 1~2 列），告别生硬纯英文与割裂版面。

	type Tab = 'posts' | 'liked' | 'saved';
	const TABS: Tab[] = ['posts', 'liked', 'saved'];

	// tab 状态入 URL ?tab=（刷新/分享/返回保留所选分区）
	const initTab = (): Tab => {
		const t = page.url.searchParams.get('tab');
		return TABS.includes(t as Tab) ? (t as Tab) : 'posts';
	};
	let activeTab = $state<Tab>(initTab());
	const setActiveTab = (tab: Tab) => {
		activeTab = tab;
		void goto(`?tab=${tab}`, { keepFocus: true, noScroll: true });
	};

	const account = useAccountData();
	const myPosts = useGalleryMoments('my');
	const reacted = useReactedMoments();
	const savedP = useFavoriteMoments();

	const postsMap = $derived({
		posts: myPosts.posts || [],
		liked: reacted.posts || [],
		saved: savedP.posts || []
	} as Record<Tab, GalleryMoment[]>);
	const archive = $derived(postsMap[activeTab] || []);
	const loading = $derived(activeTab === 'posts' ? myPosts.loading : activeTab === 'liked' ? reacted.loading : savedP.loading);
	const error = $derived(activeTab === 'posts' ? myPosts.error : activeTab === 'liked' ? reacted.error : savedP.error);
	const needsAuth = $derived(
		activeTab === 'posts' ? myPosts.needsAuth : activeTab === 'liked' ? reacted.needsAuth : savedP.needsAuth
	);
	const retryActive = () => {
		if (activeTab === 'posts') myPosts.retry();
		else if (activeTab === 'liked') reacted.retry();
		else savedP.retry();
	};

	const activeMeta: Record<Tab, { label: string; description: string; note: string }> = {
		posts: { label: '已发布', description: '把正在发生的事，留成可回看的片段。', note: '已发布记录' },
		liked: { label: '喜欢', description: '被光线、路途和陌生叙事打动的瞬间。', note: '最近收藏的灵感' },
		saved: { label: '收藏', description: '留给以后慢慢打开的城市线索。', note: '可随时继续整理' }
	};
	const tabItems: Array<{ id: Tab; label: string; count: number }> = $derived([
		{ id: 'posts', label: '发布', count: (myPosts.posts || []).length },
		{ id: 'liked', label: '喜欢', count: (reacted.posts || []).length },
		{ id: 'saved', label: '收藏', count: (savedP.posts || []).length }
	]);

	// 将 GalleryMoment 转换为标准的 FeedItem，以完全适配并复用 MomentCard
	const feedItems = $derived<FeedItem[]>(
		(archive || []).map((m) => ({
			id: m.id,
			user: m.user || account.nickname || '我',
			avatar: m.avatar || account.avatar || '',
			location: m.location || (activeTab === 'posts' ? account.locationName : '') || '',
			tag: m.tag || '',
			content: m.title || '',
			image: m.img || '',
			likes: m.likes || 0,
			comments: m.comments || 0,
			time: m.created_at ? formatDateTime(m.created_at) : '',
			liked: m.liked ?? (activeTab === 'liked'),
			saved: m.saved ?? (activeTab === 'saved'),
			likeEmoji: '👍'
		}))
	);

	const interactions = createFeedInteractions(() => feedItems);

	function openMoment(id: string, focusComments = false) {
		void goto(focusComments ? `${ROUTES.moment(id)}?from=me&focus=comments` : `${ROUTES.moment(id)}?from=me`);
	}
</script>

<div>
	{#if loading}
		<InlineLoading label="加载我的瞬间…" />
	{:else if needsAuth}
		<NeedsAuthNotice title="登录后查看" description="登录后即可查看你的瞬间与收藏" />
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
					<button type="button" onclick={openComposer} class="hover-btn rounded-full bg-accent-default px-4 py-2 text-sm font-medium text-text-on-accent shadow-raised-sm hover:shadow-raised-md focus-ring">
						发布第一条
					</button>
				{/if}
			{/snippet}
		</EmptyState>
	{:else}
		<!-- 统一的页面头部：纯中文、层级分明、呼吸感充足 -->
		<header class="border-b border-border-default pb-6">
			<div class="flex flex-wrap items-end justify-between gap-5">
				<div>
					<div class="flex items-center gap-1.5 text-xs font-medium text-accent-default">
						<Sparkles class="size-3.5" />
						<span>影像记录</span>
					</div>
					<h2 class="mt-2 text-2xl lg:text-3xl font-semibold tracking-tight text-text-primary">我的瞬间</h2>
					<p class="mt-2 max-w-xl text-sm leading-relaxed text-text-secondary">{activeMeta[activeTab].description}</p>
				</div>
				<div class="flex items-center gap-5 border-l border-border-default pl-5">
					<div>
						<p class="font-mono text-2xl font-semibold leading-none text-text-primary">{archive.length}</p>
						<p class="mt-1.5 text-xs text-text-secondary">{activeMeta[activeTab].label}</p>
					</div>
				</div>
			</div>

			<!-- 分类 Tab 统一走 TabBar（胶囊） -->
			<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
				<div class="min-w-0 flex-1" aria-label="瞬间分类">
					<TabBar items={tabItems.map((t) => ({ id: t.id, label: t.label, count: t.count }))} value={activeTab} onChange={(id) => setActiveTab(id as Tab)} />
				</div>
				<div class="flex items-center gap-1.5 text-xs text-text-secondary">
					<CalendarDays class="size-3.5 text-accent-default" strokeWidth={1.75} />
					<span>{activeTab === 'posts' ? `共 ${archive.length} 则记录` : activeMeta[activeTab].note}</span>
				</div>
			</div>
		</header>

		<!-- 规整动态网格：与 nearby/moments 完全对齐的 1~3 列对称网格，所有卡片统一复用 MomentCard -->
		<div class="pt-6">
			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
				{#each interactions.items as item (item.id)}
					<MomentCard
						{item}
						liked={item.liked}
						saved={item.saved}
						onLike={() => void interactions.toggleLike(item.id)}
						onSave={() => void interactions.toggleSave(item.id)}
						onClick={() => openMoment(item.id)}
						onCommentClick={() => openMoment(item.id, true)}
					/>
				{/each}
			</div>
		</div>
	{/if}
</div>

