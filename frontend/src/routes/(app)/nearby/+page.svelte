<script lang="ts">
	import { MapPin, Users, Store, Camera, Map as MapIcon, Search } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import NearbyImageCard from '@/components/data-display/NearbyImageCard.svelte';
	import UserCard, { type NearbyUser } from './people/UserCard.svelte';
	import { searchUsers, searchGroups, netError } from '@/api';
	import { formatDistance } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';

	// 附近聚合门户（恢复为一级 Tab，对齐鸿蒙端 NearbyPage）：
	// 搜索入口 + 人/群/瞬间/地图四入口 + 身边的新朋友横滑 + 身边兴趣圈。
	// 首页保留「实时附近」信息流聚合，本页承担「发现入口」导航语义，两者互补。

	const ENTRIES = [
		{ path: '/nearby/people', icon: Users, title: '附近的人', desc: '新朋友' },
		{ path: '/nearby/groups', icon: Store, title: '附近群组', desc: '兴趣圈' },
		{ path: '/nearby/moments', icon: Camera, title: '附近动态', desc: '新鲜事' },
		{ path: '/map', icon: MapIcon, title: '地图浏览', desc: '全景看' }
	] as const;

	let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);
	let locationName = $state('');
	let users = $state<NearbyUser[]>([]);
	let groups = $state<{ id: string; name: string; banner: string; category: string; members: number; online: number; distance: string }[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	$effect(() =>
		startDefaultLocation((loc) => {
			referencePoint = loc?.point ?? null;
			locationName = loc?.location_name ?? '';
		})
	);

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchAll() {
			loading = true;
			loadError = null;
			try {
				const ref = referencePoint ? { reference_point: referencePoint } : {};
				const [usersRes, groupsRes] = await Promise.all([
					searchUsers({ radius: 5000, sort_by: 'distance' as const, ...ref }).catch(() => netError({ items: [] })),
					searchGroups({ radius: 5000, sort_by: 'distance' as const, ...ref, pagination: { limit: 8 } }).catch(() =>
						netError({ items: [] })
					)
				]);
				if (cancelled) return;
				if (usersRes.code !== 0 || groupsRes.code !== 0) {
					loadError = usersRes.code !== 0 ? (usersRes.message ?? '加载失败') : (groupsRes.message ?? '加载失败');
					return;
				}
				users = (usersRes.data?.items ?? []).slice(0, 8).map((u) => ({
					id: u.user?.user_id ?? '',
					name: u.user?.nickname ?? '',
					distance: formatDistance(u.location?.distance),
					avatar: u.user?.avatar_url ?? '',
					isOnline: u.presence_status === 'online',
					tag: u.tags?.[0] ?? '',
					mutual: u.social_context?.mutual_friends_count ?? 0,
					note: u.bio ?? ''
				}));
				groups = (groupsRes.data?.items ?? []).slice(0, 6).map((item) => {
					const g = item.group;
					return {
						id: g.group_id,
						name: g.name,
						banner: g.avatar_url ?? '',
						category: item.tags?.[0] ?? '同城',
						members: g.members_count,
						online: item.online_members_count ?? 0,
						distance: formatDistance(item.location?.distance)
					};
				});
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchAll();
		return () => {
			cancelled = true;
		};
	});

	const device = $derived(platform.platform);
</script>
<svelte:head>
	<title>附近 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			title="附近"
			subtitle={locationName ? `${locationName} · 5km 内` : '5km 实时探索'}
			sticky
			titleSize={device === 'desktop' ? 'xl' : '3xl'}
		>
			{#snippet subtitleIcon()}
				<MapPin class="size-2.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
		</PageHeader>
	{/snippet}

	<!-- 搜索入口（对齐鸿蒙附近页：假输入样式，点击进搜索页） -->
	<button
		type="button"
		class="mb-5 flex w-full items-center gap-2.5 rounded-md border border-border-default bg-surface-primary px-4 py-2.5 text-sm text-text-secondary transition-colors hover:bg-interactive-hover focus-ring"
		onclick={() => goto(ROUTES.search)}
	>
		<Search class="size-4 shrink-0" strokeWidth={1.75} />
		<span class="truncate">搜索用户、群组、动态、地点…</span>
	</button>

	<!-- 四入口（人/群/瞬间/地图） -->
	<div class="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
		{#each ENTRIES as entry (entry.path)}
			<button
				type="button"
				class="group flex flex-col items-center gap-1.5 rounded-md border border-border-default bg-surface-primary px-3 py-4 transition-colors hover:bg-interactive-hover focus-ring"
				onclick={() => goto(entry.path)}
			>
				<entry.icon class="size-6 text-accent-default" strokeWidth={1.75} />
				<span class="text-sm font-medium text-text-primary">{entry.title}</span>
				<span class="text-xxs text-text-secondary">{entry.desc}</span>
			</button>
		{/each}
	</div>

	{#if loadError}
		<LoadError message={loadError} onRetry={() => reloadKey++} />
	{:else if loading}
		<InlineLoading label="加载附近内容…" />
	{:else}
		{#if users.length}
			<div class="mb-6">
				<SectionHead title="身边的新朋友" action="查看全部" onAction={() => goto('/nearby/people')} />
				<div class="no-scrollbar -mx-4 flex gap-3 overflow-x-auto px-4 pb-1 sm:mx-0 sm:px-0">
					{#each users as user (user.id)}
						<div class="w-36 shrink-0">
							<UserCard {user} followed={false} pending={false} onFollow={() => {}} onNavigate={() => void goto(ROUTES.user(user.id))} />
						</div>
					{/each}
				</div>
			</div>
		{/if}

		{#if groups.length}
			<div>
				<SectionHead title="身边兴趣圈" action="探索群组" onAction={() => goto('/nearby/groups')} />
				<div class="grid grid-cols-2 gap-4 lg:grid-cols-3">
					{#each groups as group (group.id)}
						<NearbyImageCard imageSrc={group.banner} imageAlt={group.name} onClick={() => void goto(ROUTES.group(group.id))} class="rounded-md aspect-[3/4]">
							{#snippet placeholder()}
								<span class="text-accent-default text-3xl font-semibold">{Array.from(group.name)[0] ?? '群'}</span>
							{/snippet}
							<span class="absolute top-2.5 left-2.5 z-[2] inline-flex h-5 items-center justify-center rounded-full bg-accent-default px-2.5 text-xxs font-medium leading-none text-text-on-accent shadow-raised-sm">
								{group.category}
							</span>
							{#snippet footer()}
								<div class="p-3">
									<p class="line-clamp-1 text-sm font-semibold text-text-primary">{group.name}</p>
									<div class="mt-1 flex items-center justify-between gap-2 text-xxs text-text-secondary">
										<span class="flex items-center gap-1">{group.members} 人</span>
										<span>{group.distance}</span>
									</div>
								</div>
							{/snippet}
						</NearbyImageCard>
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</StandardPageLayout>
