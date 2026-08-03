<script lang="ts">
	import type { Component } from 'svelte';
	import { goto } from '$app/navigation';
	import { Coffee, Music, Users, Star, ChevronRight, MapPin, Search } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import { getTrending, getNearbyUsers, getNearbyGroups } from '@/api';
	import SectionHead from '@/routes/(app)/_home/SectionHead.svelte';
	import HeroStats from '@/routes/(app)/_home/HeroStats.svelte';
	import { type NearbyUser } from '@/routes/(app)/_home/PeopleRadarStrip.svelte';
	import ActivityList, { type Activity } from '@/routes/(app)/_home/ActivityList.svelte';
	import GroupCarousel, { type NearbyGroup } from '@/routes/(app)/_home/GroupCarousel.svelte';
	import NearbyUsersFlatList from '@/routes/(app)/_home/NearbyUsersFlatList.svelte';
	import {
		TabletRail,
		NearbyPulseCard,
		StreakCard,
		DailyPickCard,
		SuggestedFollowsCard,
		VibeCard
	} from '@/components/layout/tablet';
	import { ROUTES } from '@/lib/routes';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';

	// 首页「附近」：Hero 统计 + 话题活动 + 群组轮播 + 推荐列表。
	// 三端共用同一套业务组件，仅外层 wrapper 按平台区分布局。
	// 桌面：左侧 340px 摘要栏 + 右侧完整用户列表（2 栏 grid）。
	// 移动/平板：单栏堆叠 + 宽平板右栏陪伴卡。

	// ── 数据获取（原 useEffect + Promise.all + cancelled 竞态）────────
	let nearbyUsers = $state<NearbyUser[]>([]);
	let activities = $state<Activity[]>([]);
	let nearbyGroups = $state<NearbyGroup[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
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
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				error = null;
				const searchParams: Record<string, unknown> = { radius: 5000, pagination: { limit: 12 } };
				if (referencePoint) searchParams.reference_point = referencePoint;
				const [trendingRes, nearbyUsersRes, nearbyGroupsRes] = await Promise.all([
					getTrending().catch(() => ({ code: -1, data: null })),
					getNearbyUsers(searchParams).catch(() => ({ code: -1, data: { items: [] } })),
					getNearbyGroups(searchParams).catch(() => ({ code: -1, data: { items: [] } }))
				]);
				if (cancelled) return;

				const trendingData = (trendingRes.code === 0 ? trendingRes.data : null) as any;
				const nearbyUsersData = (nearbyUsersRes.code === 0 ? nearbyUsersRes.data : null) as any;
				const nearbyGroupsData = (nearbyGroupsRes.code === 0 ? nearbyGroupsRes.data : null) as any;

				if (trendingData) {
					const acts: Activity[] = (trendingData.hot_moments || []).map((a: any, i: number) => ({
						id: i + 1,
						icon: [Coffee, Music, Users, Star][i % 4],
						title: a.description ?? '',
						sub: a.location?.location_name ?? '',
						color: ['var(--semantic-warning)', 'var(--neu-accent-primary)', 'var(--map-group)', 'var(--semantic-hot)'][i % 4],
						bg: [
							'color-mix(in srgb, var(--semantic-warning) 10%, transparent)',
							'var(--neu-accent-soft)',
							'color-mix(in srgb, var(--map-group) 10%, transparent)',
							'color-mix(in srgb, var(--semantic-hot) 10%, transparent)'
						][i % 4]
					}));
					activities = acts;
				}

				if (nearbyUsersData) {
					const users: NearbyUser[] = nearbyUsersData.items.map((u: any) => ({
						id: u.user?.user_id ?? '',
						name: u.user?.nickname ?? '',
						distance: u.location?.distance ?? '',
						avatar: u.user?.avatar_url ?? '',
						status: u.presence_status === 'online' ? '在线' : '离线',
						tag: u.tags?.[0] ?? ''
					}));
					nearbyUsers = users;
				}

				if (nearbyGroupsData) {
					const groups: NearbyGroup[] = nearbyGroupsData.items.map((g: any) => ({
						id: g.group?.group_id ?? '',
						name: g.group?.name ?? '',
						members: g.group?.members_count ?? 0,
						banner: g.group?.avatar_url ?? '',
						distance: g.location?.distance ?? '',
						privacy: g.join_method === 'invite_only' || g.join_method === 'password' ? 'private' : 'public'
					}));
					nearbyGroups = groups;
				}
			} catch (e: any) {
				if (!cancelled) error = e.message ?? '加载失败';
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

	const isTablet = $derived(platformStore.platform === 'tablet');
	const isWideTablet = $derived(isTablet && platformStore.isWideTablet);
</script>

{#snippet homeContent()}
	<!-- 三端共用的业务组件（只写一次） -->
	<HeroStats onExplore={() => void goto(ROUTES.nearbyPeople)} stats={heroStats} />
	<div>
		<SectionHead title="附近活动" />
		<ActivityList items={activities} onClick={(id) => void goto(ROUTES.event(id))} />
	</div>
	<GroupCarousel groups={nearbyGroups} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
	<div>
		<SectionHead title="人物推荐" action="查看全部" onAction={() => void goto(ROUTES.nearbyPeople)} />
		<NearbyUsersFlatList users={nearbyUsers} followingIds={followingIds} onToggle={toggleFollow} />
	</div>
{/snippet}

{#if platformStore.platform !== 'mobile'}
	<!-- ═══ Desktop/Tablet Home — header + 2-column ═══ -->
	<div class="content-mode-wide flex flex-col h-full">
		<div
			class="flex-shrink-0 flex items-center justify-between px-5 sm:px-7 py-4"
			style="background-color:var(--neu-bg);border-bottom:1px solid var(--neu-border);"
		>
			<div>
				<h1 style="font-size:24px;color:var(--neu-text);">首页</h1>
				<p style="font-size:12px;color:var(--neu-text-muted);margin-top:2px;">发现周边有趣的人和事</p>
			</div>
			<div class="flex items-center gap-2">
				<div class="flex items-center gap-1.5 px-3 py-1.5 rounded-full neu-surface-flat">
					<span class="rounded-full" style="width:6px;height:6px;background:var(--semantic-online);"></span>
					<span class="text-xs" style="color:var(--neu-text-muted);">{onlineCount} 在线</span>
				</div>
			</div>
		</div>

		<div class="flex flex-1 min-h-0 gap-0 overflow-hidden">
			<!-- Left: hero + activity + groups (340px) -->
			<div class="flex flex-col overflow-y-auto no-scrollbar gap-6 shrink-0 p-5 sm:p-6" style="width:340px;border-right:1px solid var(--neu-border);">
				<HeroStats onExplore={() => void goto(ROUTES.nearbyPeople)} stats={heroStats} />
				<div>
					<SectionHead title="附近活动" />
					<ActivityList items={activities} onClick={(id) => void goto(ROUTES.event(id))} />
				</div>
				<GroupCarousel groups={nearbyGroups} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
			</div>

			<!-- Right: full nearby-users list -->
			<div class="flex-1 overflow-y-auto no-scrollbar p-5 sm:p-6">
				<div class="flex items-end justify-between mb-5">
					<div>
						<span class="neu-eyebrow">附近的人</span>
						<p class="text-xs" style="color:var(--neu-text-muted);margin-top:4px;">
							5km 范围内 · 实时更新
						</p>
					</div>
					<button
						type="button"
						class="neu-hover-btn flex items-center gap-0.5 outline-none neu-surface-ghost rounded-full px-2 py-0.5 text-[var(--neu-accent-primary)]"
						onclick={() => void goto(ROUTES.nearbyPeople)}
					>
						<span class="text-xs">查看全部</span>
						<ChevronRight style="width:13px;height:13px;" strokeWidth={1.5} />
					</button>
				</div>

				<div class="mt-6">
					<NearbyUsersFlatList users={nearbyUsers} followingIds={followingIds} onToggle={toggleFollow} />
				</div>
			</div>
		</div>
	</div>
{:else}
	<!-- ═══ Mobile/Tablet Home — single column ═══ -->
	<div class="sticky top-0 z-10 py-3 bg-[var(--neu-bg)]" style="border-bottom:1px solid var(--neu-border);">
		<ContentColumn maxWidth={isTablet ? '924px' : undefined} class={isTablet ? 'flex gap-6' : 'flex'}>
			<div
				class="{isTablet ? 'px-5 flex items-center justify-between min-w-0' : 'w-full flex items-center justify-between'}"
				style={isTablet ? 'max-width:600px;margin-left:auto;margin-right:auto;width:100%;' : ''}
			>
				<div>
					<h1 class="text-[22px]" style="color:var(--neu-text);">附近</h1>
					<p class="text-xs" style="color:var(--neu-text-muted);margin-top:2px;">发现周边有趣的人和事</p>
				</div>
				<div class="flex items-center gap-2">
					<div class="flex items-center gap-1.5 py-1.5 px-3 rounded-full neu-surface-flat">
						<span class="rounded-full" style="width:6px;height:6px;background:var(--semantic-online);"></span>
						<span style="font-size:11px;color:var(--neu-text-muted);">徐州</span>
					</div>
					<button
						type="button"
						class="neu-hover-btn flex size-8 items-center justify-center rounded-full border cursor-pointer transition-colors hover:bg-[var(--neu-accent-soft)]"
						style="border-color:var(--neu-border);background:var(--neu-bg);"
						onclick={() => void goto(ROUTES.search)}
						aria-label="搜索"
					>
						<Search class="size-4" color="var(--neu-text-muted)" strokeWidth={1.8} />
					</button>
				</div>
			</div>
			{#if isWideTablet}
				<div class="hidden lg:block flex-shrink-0" style="width:var(--context-rail-width);"></div>
			{/if}
		</ContentColumn>

		<ContentColumn
			maxWidth={isTablet ? '924px' : undefined}
			class="animate-fade-in {isTablet ? 'flex gap-6 pt-5 pb-8' : 'flex flex-col pt-5 pb-8 gap-7'}"
		>
			<div
				class="{isTablet ? 'flex flex-col gap-7 px-5 min-w-0' : 'flex flex-col gap-7 w-full'}"
				style={isTablet ? 'max-width:600px;margin-left:auto;margin-right:auto;width:100%;' : ''}
			>
				{@render homeContent()}
			</div>
			{#if isWideTablet}
				<div class="hidden lg:block flex-shrink-0" style="width:var(--context-rail-width);">
					<TabletRail>
						<NearbyPulseCard />
						<StreakCard />
						<DailyPickCard />
						<SuggestedFollowsCard />
						<VibeCard />
					</TabletRail>
				</div>
			{/if}
		</ContentColumn>
	</div>
{/if}
