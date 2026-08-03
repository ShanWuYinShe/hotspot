<script lang="ts">
	import { Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platformStore } from '@/stores/platform.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import UserCard, { type NearbyUser } from './UserCard.svelte';
	import { getNearbyUsers } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 附近的人：排序/标签过滤 + BSP 卡片墙（桌面）/ 网格（移动/平板）。
	// 迁移自旧版 NearbyPeoplePage。

	const SORT_OPTIONS = ['距离', '在线', '共同好友'];
	const FILTER_TAGS = ['全部', '摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术'];

	let search = $state('');
	let sort = $state('距离');
	let tagFilter = $state('全部');
	let followedIds = $state<Set<string>>(new Set());
	let visibleCount = $state(12);
	let allUsers = $state<NearbyUser[]>([]);
	let loading = $state(true);
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);

	$effect(() => {
		let cancelled = false;
		void loadDefaultLocation().then((loc) => {
			if (!cancelled) referencePoint = loc?.point ?? null;
		});
		return () => {
			cancelled = true;
		};
	});

	$effect(() => {
		// reference_point 可选:有显式锚点就传,否则由服务端回退链解析。
		let cancelled = false;
		async function fetchUsers() {
			try {
				loading = true;
				const res = await getNearbyUsers(referencePoint ? { reference_point: referencePoint } : {}).catch(() => ({ code: -1, data: { items: [] } }));
				if (cancelled) return;
				const data = (res.code === 0 ? res.data : null) as any;
				if (data) {
					const items: NearbyUser[] = (data.items || []).map((u: any) => ({
						// spec: feed/nearby_users → UserItem（user 嵌套 + presence_status + location）
						id: u.user?.user_id ?? '',
						name: u.user?.nickname ?? '',
						distance: u.location?.distance ?? '',
						avatar: u.user?.avatar_url ?? '',
						status: u.presence_status === 'online' ? '在线' : '离线',
						tag: u.tags?.[0] ?? '',
						mutual: u.social_context?.mutual_friends_count ?? 0,
						note: u.bio ?? ''
					}));
					allUsers = items;
				}
			} catch {
				if (!cancelled) allUsers = [];
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchUsers();
		return () => {
			cancelled = true;
		};
	});

	// 过滤 + 排序（原 useMemo）
	const list = $derived.by(() => {
		const next = allUsers.filter(
			(user) =>
				(tagFilter === '全部' || user.tag === tagFilter) &&
				(!search || `${user.name}${user.tag}${user.note}`.toLowerCase().includes(search.toLowerCase()))
		);
		if (sort === '在线') return [...next].sort((a, b) => Number(b.status === '在线') - Number(a.status === '在线'));
		if (sort === '共同好友') return [...next].sort((a, b) => b.mutual - a.mutual);
		return next;
	});

	const isDesktop = $derived(platformStore.platform === 'desktop');
	const isTablet = $derived(platformStore.platform === 'tablet');
	const gridColumns = $derived(platformStore.platform === 'mobile' ? 2 : isDesktop ? 3 : 2);

	const toggleFollow = (id: string) => {
		const next = new Set(followedIds);
		next.has(id) ? next.delete(id) : next.add(id);
		followedIds = next;
	};
</script>

<StandardPageLayout
	contentClassName="py-5 sm:py-6"
	contentMaxWidth={isTablet ? '964px' : isDesktop ? '1152px' : '680px'}
>
	{#snippet header()}
		<PageHeader title="附近的人" subtitle="徐州 · 5km 内 · 每 3 分钟刷新" sticky glass contentClassName="pt-4 pb-0" onBack={() => history.back()}>
			{#snippet trailing()}
				<div class="hidden sm:flex flex-1 max-w-sm ml-3">
					<SearchInput value={search} onChange={(v) => (search = v)} placeholder="搜索昵称、兴趣或此刻状态" class="px-3 py-2" />
				</div>
			{/snippet}
			{#snippet below()}
				<!-- 移动端搜索框：与标题同处 sticky header 内（对齐 Figma NearbyPeoplePage） -->
				<div class="sm:hidden mt-3">
					<SearchInput value={search} onChange={(v) => (search = v)} placeholder="搜索昵称、兴趣或状态" class="px-3 py-2" />
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	<div class={isTablet ? 'grid gap-6' : ''}>
		<div class="min-w-0">
			<!-- 排序 + 标签过滤 -->
			<div class="flex flex-col xl:flex-row xl:items-center justify-between gap-3 mb-5">
				<div class="flex items-center gap-1 p-1 rounded-full neu-input-surface w-fit">
					{#each SORT_OPTIONS as option}
						<button
							type="button"
							onclick={() => (sort = option)}
							aria-pressed={sort === option}
							class="px-3.5 py-1 rounded-full outline-none transition-colors text-xs focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] {sort === option ? 'bg-[var(--neu-accent-primary)] text-white font-semibold' : 'text-[var(--neu-text-muted)]'}"
						>
							{option}
						</button>
					{/each}
				</div>
				<div class="flex gap-1.5 overflow-x-auto no-scrollbar">
					{#each FILTER_TAGS as tag}
						<TagPill active={tagFilter === tag} onClick={() => (tagFilter = tag)} size="md">{tag}</TagPill>
					{/each}
				</div>
			</div>

			{#if list.length}
				{#if isDesktop || isTablet}
					<BspCardCanvas
						items={list.slice(0, visibleCount)}
						gap={14}
						minRatio={0.68}
						maxRatio={1.48}
						minCardWidth={160}
						minCardHeight={250}
						minCanvasHeight={560}
						ariaLabel="附近用户的不规则满铺布局"
						hasMore={visibleCount < list.length}
						onLoadMore={() => (visibleCount = Math.min(visibleCount + 8, list.length))}
					>
						{#snippet renderItem(user: NearbyUser, index: number)}
							<UserCard
								user={user}
								followed={followedIds.has(user.id)}
								onFollow={() => toggleFollow(user.id)}
								onNavigate={() => void goto(ROUTES.user(user.id))}
							/>
						{/snippet}
					</BspCardCanvas>
				{:else}
					<div class="grid gap-3" style="grid-template-columns:repeat({gridColumns}, minmax(0, 1fr));">
						{#each list as user, index (user.id)}
							<UserCard
								user={user}
								followed={followedIds.has(user.id)}
								onFollow={() => toggleFollow(user.id)}
								onNavigate={() => void goto(ROUTES.user(user.id))}
							/>
						{/each}
					</div>
				{/if}
			{:else}
				<EmptyState bare title="没有找到匹配的附近用户">
					{#snippet icon()}
						<Users size={40} color="var(--neu-text-muted)" opacity={0.4} />
					{/snippet}
				</EmptyState>
			{/if}
		</div>
	</div>
</StandardPageLayout>
