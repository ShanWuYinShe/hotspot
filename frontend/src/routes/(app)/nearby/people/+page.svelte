<script lang="ts">
	import { Map, MapPin, Radar, SlidersHorizontal, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platformStore } from '@/stores/platform.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import ActionIconButton from '@/components/shared/ActionIconButton.svelte';
	import BspCardCanvas from '@/components/shared/BspCardCanvas.svelte';
	import PageHeader from '@/components/shared/PageHeader.svelte';
	import SearchInput from '@/components/shared/SearchInput.svelte';
	import OnlineDot from '@/components/shared/OnlineDot.svelte';
	import TagPill from '@/components/shared/TagPill.svelte';
	import EmptyState from '@/components/shared/EmptyState.svelte';
	import UserCard, { type NearbyUser } from './UserCard.svelte';
	import { getNearbyUsers } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 附近的人：在线条 + 排序/标签过滤 + BSP 卡片墙（桌面）/ 网格（移动/平板）。
	// 迁移自旧版 NearbyPeoplePage。

	const SORT_OPTIONS = ['距离', '在线', '共同好友'];
	const FILTER_TAGS = ['全部', '摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术'];

	let search = $state('');
	let sort = $state('距离');
	let tagFilter = $state('全部');
	let followedIds = $state<Set<number>>(new Set());
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
		const point = referencePoint;
		if (!point) return;
		let cancelled = false;
		async function fetchUsers() {
			try {
				loading = true;
				const res = await getNearbyUsers({ reference_point: point }).catch(() => ({ code: -1, data: { items: [] } }));
				if (cancelled) return;
				const data = (res.code === 0 ? res.data : null) as any;
				if (data) {
					const items: NearbyUser[] = (data.items || []).map((u: any, i: number) => ({
						id: u.user_id ?? i + 1,
						name: u.nickname ?? '',
						distance: u.distance ?? '',
						avatar: u.avatar_url ?? '',
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

	const onlineUsers = $derived(allUsers.filter((user) => user.status === '在线'));
	const onlineCount = $derived(list.filter((user) => user.status === '在线').length);

	const isDesktop = $derived(platformStore.platform === 'desktop');
	const isTablet = $derived(platformStore.platform === 'tablet');
	const contentStyle = $derived(
		isDesktop ? 'width:100%;' : isTablet ? 'max-width:964px;width:100%;' : 'max-width:680px;width:100%;'
	);
	const gridColumns = $derived(platformStore.platform === 'mobile' ? 2 : isDesktop ? 3 : 2);

	const toggleFollow = (id: number) => {
		const next = new Set(followedIds);
		next.has(id) ? next.delete(id) : next.add(id);
		followedIds = next;
	};
</script>

<PageHeader
	title="附近的人"
	subtitle="徐州 · 5km 内 · 每 3 分钟刷新"
	sticky
	onBack={() => history.back()}
>
	{#snippet actions()}
		<ActionIconButton icon={Map} label="在地图中查看" onClick={() => void goto(ROUTES.map)} variant="ghost" tone="accent" size={34} />
	{/snippet}
	{#snippet children()}
		<div class="hidden sm:flex flex-1 max-w-sm ml-3">
			<SearchInput value={search} onChange={(v) => (search = v)} placeholder="搜索昵称、兴趣或此刻状态" compact />
		</div>
	{/snippet}
</PageHeader>

{#if !isDesktop}
	<div class="sm:hidden px-5 mt-0 -mt-2 pb-2">
		<SearchInput value={search} onChange={(v) => (search = v)} placeholder="搜索昵称、兴趣或状态" compact />
	</div>
{/if}

<div class={isDesktop ? 'px-7 py-6' : 'px-5 py-5 mx-auto'} style={contentStyle}>
	<div class={isTablet ? 'grid gap-6' : ''}>
		<div class="min-w-0">
			<!-- 在线条 -->
			<section class="mb-5" aria-label="正在附近">
				<div class="flex items-center gap-2 mb-3">
					<span class="animate-pulse w-[7px] h-[7px] rounded-full" style="background:var(--semantic-online);"></span>
					<span class="neu-eyebrow">正在附近</span>
					<span class="px-1.5 py-0.5 rounded-full bg-[color-mix(in_srgb,var(--semantic-online)_14%,transparent)] text-[var(--semantic-online)] text-[10px]">{onlineUsers.length} 人在线</span>
				</div>
				<div class="flex gap-3 overflow-x-auto no-scrollbar pb-1">
					{#each onlineUsers as u (u.id)}
						<button
							type="button"
							class="neu-hover-card flex-shrink-0 flex flex-col items-center gap-1.5 outline-none"
							onclick={() => void goto(ROUTES.user(u.id))}
						>
							<div class="relative">
								<img src={u.avatar} alt={u.name} class="w-14 h-14 rounded-full object-cover border-2 border-[var(--semantic-online)]" style="box-shadow:0 0 0 2px var(--neu-bg);" />
								<OnlineDot size="lg" />
							</div>
							<span class="max-w-14 truncate text-[var(--neu-text-muted)] text-[11px]">{u.name}</span>
						</button>
					{/each}
				</div>
			</section>

			<!-- Hero 统计条 -->
			<section class="neu-surface-hero rounded-[22px] p-4 mb-5 flex items-center gap-4">
				<div class="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--neu-accent-primary)] w-[46px] h-[46px]" style="box-shadow:0 4px 14px var(--neu-glow);">
					<Radar size={20} color="white" />
				</div>
				<div class="min-w-0 flex-1">
					<p class="text-[var(--neu-text)] text-sm font-semibold">此刻的徐州，正有 {onlineCount} 位新朋友在附近</p>
					<p class="text-[var(--neu-text-muted)] text-[11px] mt-1">已按你的位置与兴趣筛出 {list.length} 人；打个招呼，从共同话题开始。</p>
				</div>
				<button
					type="button"
					onclick={() => void goto(ROUTES.map)}
					class="hidden md:flex flex-shrink-0 items-center gap-1.5 px-3 py-2 rounded-full neu-surface-ghost neu-focus-ring text-[var(--neu-accent-primary)] text-[11px]"
				>
					<Map size={13} />
					查看地图
				</button>
			</section>

			<!-- 排序 + 标签过滤 -->
			<div class="flex flex-col xl:flex-row xl:items-center justify-between gap-3 mb-5">
				<div class="flex items-center gap-1 p-1 rounded-full neu-input-surface w-fit">
					{#each SORT_OPTIONS as option}
						<button
							type="button"
							onclick={() => (sort = option)}
							aria-pressed={sort === option}
							class="px-3.5 py-1 rounded-full outline-none transition-colors text-xs focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] {sort === option ? 'bg-[var(--neu-accent-primary)] text-white font-semibold' : 'neu-surface-ghost text-[var(--neu-text-muted)]'}"
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

			<div class="flex items-center justify-between mb-3">
				<span class="neu-eyebrow">为你排序</span>
				<span class="flex items-center gap-1 text-[var(--neu-text-muted)] text-[11px]">
					<SlidersHorizontal size={12} />
					{list.length} 位可认识
				</span>
			</div>

			{#if list.length}
				{#if isDesktop}
					<BspCardCanvas
						items={list.slice(0, visibleCount)}
						gap={24}
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
				<EmptyState title="没有找到匹配的附近用户">
					{#snippet icon()}
						<Users size={24} color="var(--neu-text-muted)" opacity={0.4} />
					{/snippet}
				</EmptyState>
			{/if}
		</div>
	</div>
</div>
