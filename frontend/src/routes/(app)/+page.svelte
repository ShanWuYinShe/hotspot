<script lang="ts">
	import { goto } from '$app/navigation';
	import { ChevronRight, Search } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import { getTrending, getNearbyUsers, getNearbyGroups } from '@/api';
	import SectionHead from '@/routes/(app)/_home/SectionHead.svelte';
	import HeroStats from '@/routes/(app)/_home/HeroStats.svelte';
	import { type NearbyUser } from '@/routes/(app)/_home/nearby-users';
	import { toActivities, type Activity } from '@/routes/(app)/_home/activity-data';
	import GroupCarousel, { type NearbyGroup } from '@/routes/(app)/_home/GroupCarousel.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import { ROUTES } from '@/lib/routes';
	import { formatDistance } from '@/lib/utils';
	import { loadDefaultLocation, type DefaultLocation } from '@/lib/default-location';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';

	// 首页「附近」：Hero 统计 + 话题活动 + 群组轮播 + 推荐列表。
	// 三端统一 PageHeader（title + 额外信息 + trailing 工具区）+ 内容区：
	// 桌面/平板：左侧 340px 摘要栏 + 右侧完整用户列表（2 栏 grid）。
	// 移动：单栏堆叠。

	// ── 数据获取（Promise.all + cancelled 竞态）────────
	let nearbyUsers = $state<NearbyUser[]>([]);
	let activities = $state<Activity[]>([]);
	let nearbyGroups = $state<NearbyGroup[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let defaultLocation = $state<DefaultLocation | null>(null);

	$effect(() => {
		let cancelled = false;
		void loadDefaultLocation().then((loc) => {
			if (!cancelled) defaultLocation = loc ?? null;
		});
		return () => {
			cancelled = true;
		};
	});

	$effect(() => {
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				error = null;
				const searchParams: Record<string, unknown> = { radius: 5000, pagination: { limit: 12 } };
				if (defaultLocation) searchParams.reference_point = defaultLocation.point;
				const [trendingRes, nearbyUsersRes, nearbyGroupsRes] = await Promise.all([
					getTrending().catch(() => ({ code: -1, data: null })),
					getNearbyUsers(searchParams).catch(() => ({ code: -1, data: { items: [] } })),
					getNearbyGroups(searchParams).catch(() => ({ code: -1, data: { items: [] } }))
				]);
				if (cancelled) return;

				const trendingData = trendingRes.code === 0 ? trendingRes.data : null;
				const nearbyUsersData = nearbyUsersRes.code === 0 ? nearbyUsersRes.data : null;
				const nearbyGroupsData = nearbyGroupsRes.code === 0 ? nearbyGroupsRes.data : null;

				activities = toActivities(trendingData);

				if (nearbyUsersData) {
					const users: NearbyUser[] = nearbyUsersData.items.map((u) => ({
						id: u.user?.user_id ?? '',
						name: u.user?.nickname ?? '',
						distance: formatDistance(u.location?.distance),
						avatar: u.user?.avatar_url ?? '',
						status: u.presence_status === 'online' ? '在线' : '离线',
						tag: u.tags?.[0] ?? ''
					}));
					nearbyUsers = users;
				}

				if (nearbyGroupsData) {
					const groups: NearbyGroup[] = nearbyGroupsData.items.map((g) => ({
						id: g.group?.group_id ?? '',
						name: g.group?.name ?? '',
						members: g.group?.members_count ?? 0,
						banner: g.group?.avatar_url ?? '',
						distance: formatDistance(g.location?.distance),
						privacy: g.join_method === 'invite_only' || g.join_method === 'password' ? 'private' : 'public'
					}));
					nearbyGroups = groups;
				}
			} catch (e) {
				if (!cancelled) error = e instanceof Error ? e.message : '加载失败';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	let followingIds = $state<Set<string>>(new Set());
	const toggleFollow = (id: string) => {
		const n = new Set(followingIds);
		n.has(id) ? n.delete(id) : n.add(id);
		followingIds = n;
	};
	const onlineCount = $derived(nearbyUsers.filter((u) => u.status === '在线').length);
	const heroStats = $derived({
		nearby: String(nearbyUsers.length),
		active: String(activities.length),
		online: String(onlineCount)
	});

</script>

{#snippet activityRows(items: Activity[], onClick: (id: number) => void)}
	<ListContainer bordered>
		{#each items as item (item.id)}
			<ListItem
				onClick={() => onClick(item.id)}
				ariaLabel={item.title}
				class="flex items-center gap-3 px-4 py-3"
			>
				<div
					class="rounded-tight flex items-center justify-center shrink-0"
					style="width:34px;height:34px;background:{item.bg};"
				>
					<item.icon style="width:16px;height:16px;color:{item.color};" strokeWidth={1.75} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-[13px]"  style="color:var(--text-primary)">{item.title}</p>
					<p class="text-[11px] mt-px"  style="color:var(--text-secondary)">{item.sub}</p>
				</div>
				<ChevronRight class="size-3.25"  style="color:var(--text-secondary)" strokeWidth={1.5} />
			</ListItem>
		{/each}
	</ListContainer>
{/snippet}

{#snippet userRows(users: NearbyUser[], following: Set<string>, onToggle: (id: string) => void)}
	<ListContainer bordered>
		{#each users as u (u.id)}
			{@const followed = following.has(u.id)}
			<UserCardRow name={u.name} avatar={u.avatar}>
				{#snippet avatarBadge()}
					{#if u.status === '在线'}
						<span
							class="absolute bottom-0 right-0 rounded-full border-2"
							style="width:10px;height:10px;background:var(--status-online);border-color:var(--surface-primary);"
						></span>
					{/if}
				{/snippet}
				{#snippet subtitle()}
					<p class="mt-1 flex items-center gap-1.5">
						<span class="text-[10px]"  style="color:var(--accent-default)">#{u.tag}</span>
						<span class="text-[10px]"  style="color:var(--text-secondary);opacity:0.5">·</span>
						<span class="text-[10px]"  style="color:var(--text-secondary)">{u.distance}</span>
					</p>
				{/snippet}
				{#snippet actions()}
					<button
						type="button"
						class="hover-btn tap-93 px-3 py-1 rounded-full outline-none transition-all"
						style={followed
							? 'background:transparent;color:var(--text-secondary);border:1px solid var(--border-strong);font-size:11px;'
							: 'background:var(--accent-default);color:white;border:1px solid transparent;font-size:11px;'}
						onclick={() => onToggle(u.id)}
					>
						{followed ? '已关注' : '关注'}
					</button>
				{/snippet}
			</UserCardRow>
		{/each}
	</ListContainer>
{/snippet}

{#snippet homeContent()}
	<!-- 三端共用的业务组件（只写一次）。首页为大纲页：列表限长 + 「更多」链接导航到专区 -->
	<HeroStats onExplore={() => void goto(ROUTES.nearbyPeople)} stats={heroStats} locationName={defaultLocation?.location_name ?? ''} />
	<div>
		<SectionHead title="附近活动" action="更多" onAction={() => void goto(ROUTES.nearbyMoments)} />
		{@render activityRows(activities.slice(0, 3), (id) => void goto(ROUTES.event(id)))}
	</div>
	<GroupCarousel groups={nearbyGroups} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
	<div>
		<SectionHead title="人物推荐" action="查看全部" onAction={() => void goto(ROUTES.nearbyPeople)} />
		<!-- 对齐 Figma HomePage：人物推荐仅展示前 4 位，其余引导到附近的人专区 -->
		{@render userRows(nearbyUsers.slice(0, 4), followingIds, toggleFollow)}
	</div>
{/snippet}

<StandardPageLayout
	contentClassName={platform.platform === 'mobile' ? 'animate-fade-in flex flex-col gap-7 pt-5 pb-8' : 'py-6'}
>
	{#snippet header()}
		{#if platform.platform !== 'mobile'}
			<!-- ═══ Desktop/Tablet Home — 统一 PageHeader + 2-column ═══ -->
			<PageHeader title="首页" titleSize={24} subtitle="发现周边有趣的人和事" sticky>
				{#snippet trailing()}
					<div class="flex items-center gap-2">
						<div class="flex items-center gap-1.5 px-3 py-1.5 rounded-full surface-flat">
							<span class="rounded-full size-1.5"  style="background:var(--status-online)"></span>
							<span class="text-xs" style="color:var(--text-secondary);">{onlineCount} 在线</span>
						</div>
						{#if platform.platform === 'tablet'}
							<button
								type="button"
								aria-label="搜索"
								onclick={() => void goto(ROUTES.search)}
								class="hover-btn flex size-8 items-center justify-center rounded-full outline-none bg-surface-primary text-text-secondary"
								style="border:1px solid var(--border-default);"
							>
								<Search class="size-4"  strokeWidth={1.8} />
							</button>
						{/if}
					</div>
				{/snippet}
			</PageHeader>
		{:else}
			<!-- ═══ Mobile Home — 统一 PageHeader + single column ═══ -->
			<PageHeader title="附近" titleSize={22} subtitle="发现周边有趣的人和事" sticky>
				{#snippet trailing()}
					<div class="flex items-center gap-2">
						<div class="flex items-center gap-1.5 py-1.5 px-3 rounded-full surface-flat">
							<span class="rounded-full size-1.5"  style="background:var(--status-online)"></span>
							<span class="text-[11px]"  style="color:var(--text-secondary)">{defaultLocation?.location_name || '附近'}</span>
						</div>
						<button
							type="button"
							class="hover-btn flex size-8 items-center justify-center rounded-full border cursor-pointer transition-colors hover:bg-accent-soft"
							style="border-color:var(--border-default);background:var(--surface-primary);"
							onclick={() => void goto(ROUTES.search)}
							aria-label="搜索"
						>
							<Search class="size-4" color="var(--text-secondary)" strokeWidth={1.8} />
						</button>
					</div>
				{/snippet}
			</PageHeader>
		{/if}
	{/snippet}

	{#if platform.platform !== 'mobile'}
		<!-- Desktop/Tablet：左摘要栏 + 右用户列表（正文多列 grid，随页面统一滚动） -->
		<div class="grid grid-cols-[340px_minmax(0,1fr)] gap-6">
			<!-- Left: hero + activity + groups (340px) -->
			<div class="flex flex-col gap-6 border-r border-border-default pr-6">
				<HeroStats onExplore={() => void goto(ROUTES.nearbyPeople)} stats={heroStats} locationName={defaultLocation?.location_name ?? ''} />
				<div>
					<SectionHead title="附近活动" />
					{@render activityRows(activities, (id) => void goto(ROUTES.event(id)))}
				</div>
				<GroupCarousel groups={nearbyGroups} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
			</div>

			<!-- Right: full nearby-users list -->
			<div>
				<div class="flex items-end justify-between mb-5">
					<div>
						<span class="eyebrow">附近的人</span>
						<p class="text-xs mt-1"  style="color:var(--text-secondary)">
							5km 范围内 · 实时更新
						</p>
					</div>
					<button
						type="button"
						class="hover-btn tap-93 flex items-center gap-0.5 outline-none surface-ghost rounded-full px-2 py-0.5 text-accent-default"
						onclick={() => void goto(ROUTES.nearbyPeople)}
					>
						<span class="text-xs">查看全部</span>
						<ChevronRight class="size-3.25"  strokeWidth={1.5} />
					</button>
				</div>

				<div class="mt-6">
					{@render userRows(nearbyUsers, followingIds, toggleFollow)}
				</div>
			</div>
		</div>
	{:else}
		{@render homeContent()}
	{/if}
</StandardPageLayout>
