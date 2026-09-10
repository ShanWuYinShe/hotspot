<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		Search as SearchIcon,
		Users,
		UsersRound,
		Sparkles,
		Map as MapIcon,
		ChevronRight,
		Radar,
		MapPin,
		UserPlus,
		UserCheck
	} from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation, type DefaultLocation } from '@/lib/start-default-location';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import MomentCard from '@/components/moments/MomentCard.svelte';
	import { createFeedInteractions, type FeedItem } from '@/components/moments/moment-feed.svelte';
	import GroupCard from './groups/shared/GroupCard.svelte';
	import { mapGroup, type Group } from './groups/shared/group-data';
	import {
		searchUsers,
		searchGroups,
		fetchNearbyMoments,
		follow,
		joinGroup,
		leaveGroup,
		netError
	} from '@/api';
	import type { UserItem, MomentItem } from '@/api/types';
	import { formatDistance, formatDateTime } from '@/lib/utils';
	import { showToast } from '@/stores/toast.svelte';

	// 附近综合探索门户（Nearby Explore Portal）：
	// 告别空洞孤立按钮，聚合真实周边数据流：身边新朋友、活跃群组、即时动态图文。
	// 顶部配备轻量 4 分区快捷导航芯片与搜索入口，兼顾快速寻路与沉浸式探索。

	const SECTIONS = [
		{ id: 'people', icon: Users, title: '附近的人', desc: '新朋友', href: ROUTES.nearbyPeople },
		{ id: 'groups', icon: UsersRound, title: '附近群组', desc: '兴趣圈', href: ROUTES.nearbyGroups },
		{ id: 'moments', icon: Sparkles, title: '附近动态', desc: '新鲜事', href: ROUTES.nearbyMoments },
		{ id: 'map', icon: MapIcon, title: '地图浏览', desc: '全景看', href: ROUTES.map }
	];

	interface NearbyFriend {
		id: string;
		name: string;
		avatar: string;
		distance: string;
		tag: string;
		isOnline: boolean;
	}

	let defaultLocation = $state<DefaultLocation | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	let nearbyFriends = $state<NearbyFriend[]>([]);
	let nearbyGroups = $state<Group[]>([]);
	let feedMoments = $state<FeedItem[]>([]);
	let groupPending = $state<Set<string>>(new Set());

	// 关注状态管理（乐观更新）
	let followingIds = $state<Set<string>>(new Set());
	let followPending = $state<Set<string>>(new Set());

	const feedInteractions = createFeedInteractions(() => feedMoments);

	function mapMomentItem(m: MomentItem, index: number): FeedItem {
		const mine = (m.reactions || []).find((r) => r.reacted_by_me);
		return {
			id: m.moment_id ?? `moment-${index}`,
			user: m.user?.nickname ?? '热点用户',
			avatar: m.user?.avatar_url ?? '',
			location: m.location?.location_name ?? '',
			tag: m.tags?.[0] ?? '',
			content: m.description ?? '',
			image: m.images?.[0]?.url ?? '',
			likes: (m.reactions ?? []).reduce((sum, r) => sum + r.count, 0),
			comments: m.comments_count ?? 0,
			time: formatDateTime(m.created_at),
			liked: Boolean(mine),
			saved: Boolean(m.is_favorited),
			likeEmoji: mine?.emoji ?? '👍'
		};
	}

	$effect(() => startDefaultLocation((loc) => {
		defaultLocation = loc ?? null;
	}));

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchExploreData() {
			try {
				loading = true;
				loadError = null;
				const refPoint = defaultLocation?.point;
				const pointParam = refPoint ? { reference_point: refPoint } : {};

				const [momentsRes, usersRes, groupsRes] = await Promise.all([
					fetchNearbyMoments({
						radius: 5000,
						...pointParam,
						pagination: { limit: 8 }
					}).catch(() => netError({ items: [] })),
					searchUsers({
						radius: 5000,
						sort_by: 'distance',
						...pointParam,
						pagination: { limit: 10 }
					}).catch(() => netError({ items: [] })),
					searchGroups({
						radius: 5000,
						sort_by: 'distance',
						...pointParam,
						pagination: { limit: 4 }
					}).catch(() => netError({ items: [] }))
				]);

				if (cancelled) return;

				const hasMoments = momentsRes.code === 0 && momentsRes.data?.items;
				const hasUsers = usersRes.code === 0 && usersRes.data?.items;
				const hasGroups = groupsRes.code === 0 && groupsRes.data?.items;

				if (!hasMoments && !hasUsers && !hasGroups) {
					loadError = momentsRes.message || usersRes.message || groupsRes.message || '加载周边数据失败';
					return;
				}

				const mItems = (momentsRes.code === 0 && momentsRes.data?.items) ? momentsRes.data.items : [];
				feedMoments = mItems.map(mapMomentItem);

				const uItems = (usersRes.code === 0 && usersRes.data?.items) ? usersRes.data.items : [];
				nearbyFriends = uItems.map((u) => ({
					id: u.user?.user_id ?? '',
					name: u.user?.nickname ?? '新朋友',
					avatar: u.user?.avatar_url ?? '',
					distance: formatDistance(u.location?.distance),
					tag: u.tags?.[0] ?? '',
					isOnline: u.presence_status === 'online'
				})).filter((u) => Boolean(u.id));

				const gItems = (groupsRes.code === 0 && groupsRes.data?.items) ? groupsRes.data.items : [];
				nearbyGroups = gItems.map(mapGroup);
			} catch {
				if (!cancelled) loadError = '网络异常，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchExploreData();
		return () => {
			cancelled = true;
		};
	});

	const toggleFollow = (id: string, e?: Event) => {
		e?.stopPropagation();
		if (followPending.has(id)) return;
		const n = new Set(followingIds);
		const next = !n.has(id);
		next ? n.add(id) : n.delete(id);
		followingIds = n;
		followPending = new Set(followPending).add(id);

		follow({ user_id: id, active: next })
			.catch(() => {
				const r = new Set(followingIds);
				next ? r.delete(id) : r.add(id);
				followingIds = r;
				showToast('操作失败，请重试', 'danger');
			})
			.finally(() => {
				const rest = new Set(followPending);
				rest.delete(id);
				followPending = rest;
			});
	};

	const toggleGroupJoin = (group: Group) => {
		if (groupPending.has(group.id)) return;
		const nextJoined = !group.joined;
		groupPending = new Set(groupPending).add(group.id);
		nearbyGroups = nearbyGroups.map((g) =>
			g.id === group.id
				? { ...g, joined: nextJoined, members: nextJoined ? g.members + 1 : Math.max(0, g.members - 1) }
				: g
		);
		const req = nextJoined
			? joinGroup(group.id)
			: leaveGroup(group.id);
		req.catch(() => {
			// 回滚
			nearbyGroups = nearbyGroups.map((g) =>
				g.id === group.id
					? { ...g, joined: !nextJoined, members: nextJoined ? Math.max(0, g.members - 1) : g.members + 1 }
					: g
			);
			showToast('操作失败，请重试', 'danger');
		}).finally(() => {
			const next = new Set(groupPending);
			next.delete(group.id);
			groupPending = next;
		});
	};

	const device = $derived(platform.platform);
	const hasAnyData = $derived(nearbyFriends.length > 0 || nearbyGroups.length > 0 || feedMoments.length > 0);
</script>

<svelte:head>
	<title>附近 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentClassName="animate-fade-in flex flex-col gap-5 pt-3 pb-16">
	{#snippet header()}
		<!-- 页头不再放「地图全景」图标按钮：正文「四大分区快捷入口」里已有带文字的
		     「地图浏览」芯片跳同一目标，桌面外壳侧栏另有常驻的位置卡；三者同屏属重复入口。 -->
		<PageHeader
			title="附近"
			subtitle={defaultLocation?.location_name ? `${defaultLocation.location_name} · 5km 实时探索` : '发现周边 5km 内有趣的人和事'}
			sticky
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
			subtitleSize={device === 'desktop' ? 'xs' : 'sm'}
		/>
	{/snippet}

	<!-- 顶部快捷搜索 + 四大分区入口芯片组（紧凑、精致、不占大片视口） -->
	<section class="flex flex-col gap-3">
		<button
			type="button"
			class="hover-btn flex w-full items-center gap-3 rounded-md border border-border-default bg-surface-primary px-4 py-2.5 text-left shadow-raised-sm focus-ring transition-colors hover:border-border-strong"
			onclick={() => void goto(ROUTES.search)}
			aria-label="搜索周边内容"
		>
			<SearchIcon class="size-4 shrink-0 text-accent-default" strokeWidth={2} />
			<span class="min-w-0 flex-1 text-sm text-text-secondary truncate">搜索用户、群组、动态、地点…</span>
		</button>

		<!-- 4 大分区快捷入口芯片 -->
		<div class="grid grid-cols-4 gap-2">
			{#each SECTIONS as section (section.id)}
				<button
					type="button"
					onclick={() => void goto(section.href)}
					class="hover-btn group flex flex-col items-center justify-center gap-1.5 rounded-xl border border-border-default bg-surface-primary py-2.5 px-1 shadow-raised-sm transition-all hover:border-accent-default hover:bg-accent-soft focus-ring"
				>
					<span class="flex size-8 items-center justify-center rounded-full bg-accent-soft text-accent-default transition-transform group-hover:scale-110">
						<section.icon class="size-4" strokeWidth={1.75} />
					</span>
					<span class="text-xs font-semibold text-text-primary tracking-tight">{section.title}</span>
					<span class="text-xxs text-text-secondary truncate max-w-full px-0.5">{section.desc}</span>
				</button>
			{/each}
		</div>
	</section>

	{#if loading}
		<InlineLoading label="正在探索周边动态与邻近好友…" />
	{:else if loadError}
		<LoadError message={loadError} onRetry={() => reloadKey++} />
	{:else if !hasAnyData}
		<EmptyState
			title="周边暂时还没有动态"
			description="成为第一个点亮这片区域的人，发布瞬间或加入周边群组吧！"
		>
			{#snippet icon()}
				<Radar class="size-8 opacity-40 text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<!-- ═══ 板块一：身边的新朋友（横向滑轨，所见即所得） ═══ -->
		{#if nearbyFriends.length > 0}
			<section class="flex flex-col gap-2.5">
				<SectionHead
					title="身边的新朋友"
					action="查看全部"
					onAction={() => void goto(ROUTES.nearbyPeople)}
				/>
				<!-- 横向滑轨：自然滚动，露出卡片边缘提示可滑动 -->
				<div class="flex gap-3 overflow-x-auto pb-1.5 -mx-4 px-4 no-scrollbar snap-x">
					{#each nearbyFriends as friend (friend.id)}
						<article
							class="snap-start shrink-0 w-32 flex flex-col items-center rounded-xl border border-border-default bg-surface-primary p-3 shadow-raised-sm transition-all hover:border-border-strong hover:shadow-raised-md"
						>
							<button
								type="button"
								class="flex flex-col items-center w-full focus-ring rounded-lg cursor-pointer"
								onclick={() => void goto(ROUTES.user(friend.id))}
							>
								<div class="relative">
									<Avatar src={friend.avatar} name={friend.name} size={48} />
									{#if friend.isOnline}
										<span class="absolute bottom-0 right-0">
											<OnlineDot />
										</span>
									{/if}
								</div>
								<span class="mt-2 text-xs font-semibold text-text-primary truncate w-full text-center">
									{friend.name}
								</span>
								<div class="mt-1 flex items-center gap-0.5 text-xxs text-text-secondary">
									<MapPin class="size-2.5 shrink-0 text-accent-default" strokeWidth={2} />
									<span>{friend.distance}</span>
								</div>
								{#if friend.tag}
									<span class="badge mt-1.5 bg-accent-soft text-accent-default truncate max-w-full">
										{friend.tag}
									</span>
								{/if}
							</button>

							<button
								type="button"
								onclick={(e) => toggleFollow(friend.id, e)}
								class="mt-2.5 w-full flex items-center justify-center gap-1 py-1 rounded-md text-xxs font-medium transition-colors focus-ring {followingIds.has(friend.id) ? 'bg-surface-subtle text-text-secondary border border-border-default hover:bg-interactive-active' : 'bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md'}"
								disabled={followPending.has(friend.id)}
								aria-label={followingIds.has(friend.id) ? `取消关注 ${friend.name}` : `关注 ${friend.name}`}
							>
								{#if followingIds.has(friend.id)}
									<UserCheck class="size-3" strokeWidth={2} />
									<span>已关注</span>
								{:else}
									<UserPlus class="size-3" strokeWidth={2} />
									<span>关注</span>
								{/if}
							</button>
						</article>
					{/each}
				</div>
			</section>
		{/if}

		<!-- ═══ 板块二：身边的兴趣圈（活跃群组） ═══ -->
		{#if nearbyGroups.length > 0}
			<section class="flex flex-col gap-2.5">
				<SectionHead
					title="身边兴趣圈"
					action="探索群组"
					onAction={() => void goto(ROUTES.nearbyGroups)}
				/>
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
					{#each nearbyGroups.slice(0, 2) as group (group.id)}
						<GroupCard
							{group}
							pending={groupPending.has(group.id)}
							onOpen={() => void goto(ROUTES.group(group.id))}
							onToggle={() => toggleGroupJoin(group)}
						/>
					{/each}
				</div>
			</section>
		{/if}

		<!-- ═══ 板块三：周边新鲜事（真实图文动态流） ═══ -->
		{#if feedMoments.length > 0}
			<section class="flex flex-col gap-2.5">
				<SectionHead
					title="周边新鲜事"
					action="更多动态"
					onAction={() => void goto(ROUTES.nearbyMoments)}
				/>
				<div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
					{#each feedInteractions.items as item (item.id)}
						<MomentCard
							{item}
							liked={item.liked}
							saved={item.saved}
							onLike={() => feedInteractions.toggleLike(item.id)}
							onSave={() => feedInteractions.toggleSave(item.id)}
							onClick={() => void goto(ROUTES.moment(item.id))}
						/>
					{/each}
				</div>
			</section>
		{/if}
	{/if}
</StandardPageLayout>
