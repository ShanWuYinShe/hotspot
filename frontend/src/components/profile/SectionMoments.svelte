<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { CalendarDays, Grid3X3 } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import Button from '@/components/ui/button.svelte';
	import ProfileMomentTile from '@/components/profile/ProfileMomentTile.svelte';
	import { createFeedInteractions, type FeedItem } from '@/components/moments/moment-feed.svelte';
	import { openComposer } from '@/stores/composer.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { useAccountData, useGalleryMoments, useReactedMoments, useFavoriteMoments, type GalleryMoment } from '@/lib/profile-data.svelte';
	import { formatCardTime } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';

	// 我的瞬间分区：与 nearby/moments 统一视觉语言（共享 MomentCard 原语），
	// 并给每张卡加管理入口（R13 评审 P1-7 批准：编辑/删除，复用详情页同一实现）。
	// 删除/编辑结果只改本地列表（不做整页重拉），避免操作后滚动位置丢失。

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

	// 本地删除/编辑覆盖（不重拉列表，保留滚动位置）
	let removedIds = $state(new Set<string>());
	let editedMap = $state<Record<string, { description: string; tags: string[] }>>({});

	const postsMap = $derived({
		posts: myPosts.posts || [],
		liked: reacted.posts || [],
		saved: savedP.posts || []
	} as Record<Tab, GalleryMoment[]>);
	const archive = $derived((postsMap[activeTab] || []).filter((m) => !removedIds.has(m.id)));
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
		{ id: 'posts', label: '发布', count: myPosts.posts?.length ?? 0 },
		{ id: 'liked', label: '喜欢', count: reacted.posts?.length ?? 0 },
		{ id: 'saved', label: '收藏', count: savedP.posts?.length ?? 0 }
	]);

	// 将 GalleryMoment 转换为标准的 FeedItem，以完全适配并复用 MomentCard
	const feedItems = $derived<FeedItem[]>(
		(archive || []).map((m) => {
			const edited = editedMap[m.id];
			const tags = edited ? edited.tags : m.tag ? [m.tag] : [];
			return {
				id: m.id,
				is_deleted: m.is_deleted ?? false,
				user: m.user || account.nickname || '我',
				avatar: m.avatar || account.avatar || '',
				location: m.location || (activeTab === 'posts' ? account.locationName : '') || '',
				tag: tags[0] ?? '',
				tags,
				content: edited ? edited.description : m.title || '',
				image: m.img || '',
				likes: m.likes || 0,
				comments: m.comments || 0,
				time: m.created_at ? formatCardTime(m.created_at) : '',
				liked: m.liked ?? (activeTab === 'liked'),
				saved: m.saved ?? (activeTab === 'saved'),
				likeEmoji: '👍'
			};
		})
	);

	const interactions = createFeedInteractions(() => feedItems);

	function openMoment(id: string, focusComments = false) {
		void goto(focusComments ? `${ROUTES.moment(id)}?from=me&focus=comments` : `${ROUTES.moment(id)}?from=me`);
	}

	function handleDeleted(id: string) {
		removedIds = new Set([...removedIds, id]);
	}

	function handleEdited(id: string, updated: { description: string; tags: string[] }) {
		editedMap = { ...editedMap, [id]: updated };
	}
</script>

<div>
	{#if loading}
		<!-- 首屏骨架屏：与真实 MomentCard 网格同列数、同卡比例（骨架→内容零位移） -->
		<SkeletonFeed
			variant="card"
			count={6}
			class="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4"
			label="加载我的瞬间…"
		/>
	{:else if needsAuth}
		<NeedsAuthNotice spotlight title="登录后查看" description="登录后即可查看你的瞬间与收藏" />
	{:else if error}
		<LoadError message={error} onRetry={retryActive} />
	{:else if archive.length === 0}
		<!-- 空态：页面级，用带图标砖的空态（非 bare）+ 场景化装饰 -->
		<EmptyState spotlight iconTone="accent" title="这里还没有瞬间" description="从一次路过、一次相遇开始记录。">
			{#snippet icon()}
				<Grid3X3 class="size-7" strokeWidth={1.5} />
			{/snippet}
			{#snippet action()}
				{#if activeTab === 'posts'}
					<Button type="button" size="md" onclick={openComposer}>发布第一条</Button>
				{/if}
			{/snippet}
		</EmptyState>
	{:else}
		<!-- 分区工具行：分类 Tab + 记录计数。页面标题由外层 PageHeader 唯一提供。 -->
		<div class="flex flex-wrap items-center justify-between gap-3 pb-4">
			<div class="min-w-0 flex-1" aria-label="瞬间分类">
				<TabBar items={tabItems.map((t) => ({ id: t.id, label: t.label, count: t.count }))} value={activeTab} onChange={(id) => setActiveTab(id as Tab)} />
			</div>
			<div class="flex items-center gap-1.5 text-xs text-text-secondary">
				<CalendarDays class="size-3.5 text-accent-default" strokeWidth={1.75} />
				<span>{activeTab === 'posts' ? `共 ${archive.length} 则记录` : activeMeta[activeTab].note}</span>
			</div>
		</div>

		<!-- 规整动态网格：与 nearby/moments 对齐（移动 2 列竖版卡 / 桌面 3–4 列） -->
		<div class="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4">
			{#each interactions.items as item (item.id)}
				{#if item.is_deleted}
					<!-- 软删除墓碑（契约：客户端展示占位）：不可点、无操作，仅说明状态 -->
					<div class="flex aspect-[3/4] flex-col items-center justify-center gap-2 rounded-lg bg-surface-subtle/60 px-4 text-center">
						<p class="text-sm text-text-secondary">该瞬间已删除</p>
					</div>
				{:else}
					<ProfileMomentTile
						{item}
						liked={item.liked}
						saved={item.saved}
						onLike={() => void interactions.toggleLike(item.id)}
						onSave={() => void interactions.toggleSave(item.id)}
						onClick={() => openMoment(item.id)}
						onCommentClick={() => openMoment(item.id, true)}
						onDeleted={() => handleDeleted(item.id)}
						onEdited={(updated) => handleEdited(item.id, updated)}
					/>
				{/if}
			{/each}
		</div>
	{/if}
</div>