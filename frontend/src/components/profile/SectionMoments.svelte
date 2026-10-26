<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { CalendarDays, Grid3X3 } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import MomentCard from '@/components/moments/MomentCard.svelte';
	import { createFeedInteractions, type FeedItem } from '@/components/moments/moment-feed.svelte';
	import { openComposer } from '@/stores/composer.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { useAccountData, useGalleryMoments, useReactedMoments, useFavoriteMoments, type GalleryMoment } from '@/lib/profile-data.svelte';
	import { formatCardTime } from '@/lib/utils';
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
	// list_reacted 的 user_id 必传（spec，不回退当前用户）：显式传登录用户 id；
	// getter 形式传入，auth 就绪前后 effect 自动重跑。
	const reacted = useReactedMoments(() => auth.user?.user_id);
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

	// 分区工具的计数/尾注文案；页面标题与描述由外层 PageHeader 承担，不在内容区重复。
	const activeMeta: Record<Tab, { label: string; note: string }> = {
		posts: { label: '已发布', note: '已发布记录' },
		liked: { label: '喜欢', note: '最近收藏的灵感' },
		saved: { label: '收藏', note: '可随时继续整理' }
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
			is_deleted: m.is_deleted ?? false,
			user: m.user || account.nickname || '我',
			avatar: m.avatar || account.avatar || '',
			location: m.location || (activeTab === 'posts' ? account.locationName : '') || '',
			tag: m.tag || '',
			tags: m.tag ? [m.tag] : [],
			content: m.title || '',
			image: m.img || '',
			likes: m.likes || 0,
			comments: m.comments || 0,
			time: m.created_at ? formatCardTime(m.created_at) : '',
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
		<!-- 首屏骨架屏：与真实 MomentCard 网格同列数、同卡比例（骨架→内容零位移） -->
		<SkeletonFeed
			variant="card"
			count={6}
			class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5"
			label="加载我的瞬间…"
		/>
	{:else if needsAuth}
		<NeedsAuthNotice spotlight title="登录后查看" description="登录后即可查看你的瞬间与收藏" />
	{:else if error}
		<LoadError message={error} onRetry={retryActive} />
	{:else if archive.length === 0}
		<!-- 空态 -->
		<EmptyState spotlight title="这里还没有瞬间" description="从一次路过、一次相遇开始记录。" bare>
			{#snippet icon()}
				<Grid3X3 size={24} />
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
		<!-- 分区工具行：分类 Tab + 记录计数。页面标题由外层 PageHeader 唯一提供
		     （同族 /me/groups 无内容区大标题；此处曾重复渲染同名 h2 并吃掉首屏 1/3）。 -->
		<div class="flex flex-wrap items-center justify-between gap-3 border-b border-border-default pb-4">
			<div class="min-w-0 flex-1" aria-label="瞬间分类">
				<TabBar items={tabItems.map((t) => ({ id: t.id, label: t.label, count: t.count }))} value={activeTab} onChange={(id) => setActiveTab(id as Tab)} />
			</div>
			<div class="flex items-center gap-1.5 text-xs text-text-secondary">
				<CalendarDays class="size-3.5 text-accent-default" strokeWidth={1.75} />
				<span>{activeTab === 'posts' ? `共 ${archive.length} 则记录` : activeMeta[activeTab].note}</span>
			</div>
		</div>

		<!-- 规整动态网格：与 nearby/moments 完全对齐的 1~3 列对称网格，所有卡片统一复用 MomentCard。
		     工具行 pb-4（分隔线上侧 16）与此处 pt-4（分隔线下侧 16）分居 border-b 两侧，
		     是对称留白而非 mt+mb 双倍叠加，格子距卡片内容保持 16。 -->
		<div class="pt-4">
			<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-4">
				{#each interactions.items as item (item.id)}
					{#if item.is_deleted}
						<!-- 软删除墓碑（契约：客户端展示占位）：不可点、无操作，仅说明状态 -->
						<div class="flex aspect-[4/3] flex-col items-center justify-center gap-2 rounded-md border border-border-default bg-surface-primary px-4 text-center">
							<p class="text-sm text-text-secondary">该瞬间已删除</p>
						</div>
					{:else}
						<MomentCard
							{item}
							liked={item.liked}
							saved={item.saved}
							onLike={() => void interactions.toggleLike(item.id)}
							onSave={() => void interactions.toggleSave(item.id)}
							onClick={() => openMoment(item.id)}
							onCommentClick={() => openMoment(item.id, true)}
						/>
					{/if}
				{/each}
			</div>
		</div>
	{/if}
</div>

