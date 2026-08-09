<script lang="ts">
	import { goto } from '$app/navigation';
	import { ArrowLeft, Bell, Check, MapPin, MessageCircle, Users } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
import TabBar from '@/components/navigation/TabBar.svelte';
		import { ROUTES } from '@/lib/routes';
	import { formatDistance, formatDateTime } from '@/lib/utils';
	import { getGroupDetail, getGroupMembers, getGroupAnnouncements, joinGroup, leaveGroup } from '@/api';
	import { wsClient } from '@/lib/ws/client';
	import { presence, watchPresence } from '@/stores/presence.svelte';

	// 群组详情：封面 + 简介/成员/公告三 Tab + 加入按钮 + 桌面侧栏。
	// WS 实时：订阅 group 频道，成员/详情/公告变更事件触发对应刷新；解散弹提示并跳转。

	let { params }: { params: { groupId: string } } = $props();

	type GroupTab = 'overview' | 'members' | 'announcements';

	interface Group {
		name: string;
		category: string;
		members: number;
		online: number;
		distance: string;
		description: string;
		banner: string;
	}
	interface Member {
		id: number;
		userId: string;
		name: string;
		role: string;
		online: boolean;
		avatar: string;
	}
	interface Announcement {
		title: string;
		body: string;
		time: string;
	}

	let joined = $state(false);
	/** 加入/退出请求进行中——防连点 */
	let joinPending = $state(false);
	let tab = $state<GroupTab>('overview');
	let group = $state<Group>({
		name: '',
		category: '',
		members: 0,
		online: 0,
		distance: '',
		description: '',
		banner: ''
	});
	let members = $state<Member[]>([]);
	let announcements = $state<Announcement[]>([]);
	let loading = $state(true);
	let unwatchMembers: (() => void) | null = null;
	/** 竞态防护：仅最后一次发起的加载允许落地。 */
	let loadToken = 0;

	// --- 加载函数（初始加载与 WS 事件刷新复用） ---

	// 加入/退出：乐观更新 + 失败回滚（spec: groups/join、groups/leave）。
	// join 对需审核群组返回 pending_approval——UI 仍标记 joined 简化（真实后端
	// 以 my_state.is_joined 为准，审核通过前详情不会把它标为已加入）。
	function toggleJoin() {
		if (joinPending) return;
		joinPending = true;
		const next = !joined;
		joined = next;
		const done = (res: { code: number }) => {
			joinPending = false;
			if (res.code !== 0) joined = !next;
		};
		if (next) {
			void joinGroup(params.groupId).then(done).catch(() => done({ code: -1 }));
		} else {
			void leaveGroup(params.groupId).then(done).catch(() => done({ code: -1 }));
		}
	}

	async function loadDetail(gid: string) {
		try {
			const detailRes = await getGroupDetail(gid).catch(() => ({ code: -1, data: null }));
			// 竞态防护：页面已切到别的群组则丢弃本次结果
			if (params.groupId !== gid) return;
			const detailData = detailRes.code === 0 ? detailRes.data : null;
			if (detailData) {
				const g = detailData.group;
				group = {
					name: g.name,
					category: (detailData.tags?.[0] ?? group.category) || '同城',
					members: g.members_count,
					online: detailData.online_members_count ?? group.online,
					distance: formatDistance(detailData.location?.distance) || group.distance,
					description: detailData.description ?? group.description,
					banner: g.avatar_url ?? group.banner
				};
				if (detailData.my_state?.is_joined) joined = true;
			}
		} catch {
			// empty data is normal
		}
	}

	async function loadMembers(gid: string) {
		try {
			const membersRes = await getGroupMembers(gid).catch(() => ({ code: -1, data: { items: [] } }));
			// 竞态防护：页面已切到别的群组则丢弃本次结果
			if (params.groupId !== gid) return;
			const membersData = membersRes.code === 0 ? membersRes.data : null;
			if (membersData) {
				const items: Member[] = (membersData.items || []).map((m, i) => ({
					id: i + 1,
					userId: m.user?.user_id ?? '',
					name: m.user?.nickname ?? '',
					role: m.role ?? '成员',
					online: m.presence_status === 'online',
					avatar: m.user?.avatar_url ?? ''
				}));
				members = items;
				// 成员在线状态实时化：订阅全部成员（先释放旧订阅再建新订阅）
				unwatchMembers?.();
				unwatchMembers = watchPresence(items.map((m) => m.userId).filter(Boolean));
			}
		} catch {
			// empty data is normal
		}
	}

	async function loadAnnouncements(gid: string) {
		try {
			const announcementsRes = await getGroupAnnouncements(gid).catch(() => ({ code: -1, data: { items: [] } }));
			// 竞态防护：页面已切到别的群组则丢弃本次结果
			if (params.groupId !== gid) return;
			const announcementsData = announcementsRes.code === 0 ? announcementsRes.data : null;
			if (announcementsData) {
				const items: Announcement[] = (announcementsData.items || []).map((a) => ({
					title: a.title ?? '',
					body: a.content ?? '',
					time: formatDateTime(a.created_at)
				}));
				announcements = items;
			}
		} catch {
			// empty data is normal
		}
	}

	async function loadAll(gid: string) {
		const token = ++loadToken;
		loading = true;
		await Promise.all([loadDetail(gid), loadMembers(gid), loadAnnouncements(gid)]);
		if (token !== loadToken) return; // 已被更新的请求取代
		loading = false;
	}

	// 初始加载
	$effect(() => {
		const gid = params.groupId ?? '';
		if (!gid) return;
		void loadAll(gid);
	});

	// WS 群组订阅：成员/详情/公告变更实时刷新；解散跳转列表页。
	$effect(() => {
		const gid = params.groupId ?? '';
		if (!gid) return;

		wsClient.setSubscriptions('subscribe', [{ type: 'group', ids: [gid] }]);

		const isForThisGroup = (p: unknown) => (p as { group_id?: string })?.group_id === gid;
		const offs = [
			wsClient.on('GroupMemberJoined', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberLeft', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberRemoved', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberMuted', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberUnmuted', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupMemberRoleChanged', (p) => {
				if (isForThisGroup(p)) void loadMembers(gid);
			}),
			wsClient.on('GroupUpdated', (p) => {
				if (isForThisGroup(p)) void loadDetail(gid);
			}),
			wsClient.on('GroupAnnouncementCreated', (p) => {
				if (isForThisGroup(p)) void loadAnnouncements(gid);
			}),
			wsClient.on('GroupDismissed', (p) => {
				if (!isForThisGroup(p)) return;
				window.alert('该群组已被解散');
				void goto(ROUTES.nearbyGroups);
			})
		];

		return () => {
			for (const off of offs) off();
			wsClient.setSubscriptions('unsubscribe', [{ type: 'group', ids: [gid] }]);
			unwatchMembers?.();
			unwatchMembers = null;
		};
	});

	// 成员在线实时派生：绿点与"X 人在线"计数均以 presence 为准（HTTP 快照仅初始兜底）。
	const onlineMemberIds = $derived(
		new Set(members.filter((m) => presence.isOnline(m.userId)).map((m) => m.userId))
	);
	const onlineMemberCount = $derived(onlineMemberIds.size);
</script>
<svelte:head>
	<title>群组主页 · HotSpot</title>
</svelte:head>


<StandardPageLayout contentClassName="animate-fade-in pb-6">
	<!-- 全幅封面（负 margin 抵消 ContentColumn gutter） -->
	<div class="relative h-[var(--cst-cover-height)] overflow-hidden -mx-[var(--cst-page-gutter)]">
		<img src={group.banner} alt={`${group.name} 的群组封面`} class="h-full w-full object-cover" />
		<!-- gradient scrim 走 inline background（gradient 不能作 background-color 值，
		     与 NearbyImageCard 的 --cst-surface-media-scrim-card 用法一致） -->
		<div class="absolute inset-0" style="background:var(--cst-surface-cover-scrim);"></div>
		<div class="absolute left-5 top-4">
			<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} variant="overlay" tone="inverse" />
		</div>
		<div class="absolute bottom-5 left-6 right-6">
			<span class="rounded-full bg-surface-glass-badge px-2.5 py-1 text-xs text-content-inverse backdrop-blur-sm">{group.category}</span>
			<h1 class="mt-3 text-2xl md:text-3xl font-medium tracking-title text-content-inverse">{group.name}</h1>
		</div>
	</div>

	<div class="grid gap-8 pt-6 lg:grid-cols-[minmax(0,1fr)_80]">
			<main class="min-w-0">
			<div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-text-secondary">
				<span class="flex items-center gap-1.5">
					<Users size={14} />
					{group.members} 位成员
				</span>
				<span class="flex items-center gap-1.5 text-status-online">
					<span class="h-2 w-2 rounded-full bg-status-online"></span>
					{onlineMemberCount} 人在线
				</span>
				<span class="flex items-center gap-1.5">
					<MapPin size={14} />
					距你 {group.distance}
				</span>
			</div>

			<!-- 分类 Tab（统一 TabBar 下划线样式） -->
			<div class="mt-4 border-b border-border-default">
				<TabBar
					items={[
						{ id: 'overview', label: '简介' },
						{ id: 'members', label: '成员' },
						{ id: 'announcements', label: '公告' }
					]}
					value={tab}
					onChange={(id) => (tab = id as GroupTab)}
				/>
			</div>

			{#if tab === 'overview'}
				<div class="py-6">
					<p class="max-w-3xl text-sm leading-7 text-text-secondary">{group.description}</p>
				</div>
				{:else if tab === 'members'}
					<!-- 空旷内容区 → List bordered（有边框列表契约）；列表行不设独立边框 -->
					<ListContainer bordered class="mt-6">
						{#each members as member (member.id)}
							<ListItem
								onClick={() => {
									if (member.userId) void goto(ROUTES.user(member.userId));
								}}
								ariaLabel={`查看 ${member.name} 的资料`}
								class="app-list-row app-list-row--standard"
							>
								<div class="relative">
									<img src={member.avatar} alt="" class="h-10 w-10 rounded-full object-cover" />
									{#if onlineMemberIds.has(member.userId)}
										<span class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-surface-primary bg-status-online"></span>
									{/if}
								</div>
								<span class="min-w-0 flex-1">
									<span class="block text-sm text-text-primary">{member.name}</span>
									<span class="text-xs text-text-secondary">{member.role}</span>
								</span>
								<span class="text-xs text-text-secondary">查看资料</span>
							</ListItem>
						{/each}
					</ListContainer>
			{:else}
				<!-- 空旷内容区 → List bordered（有边框列表契约） -->
				<ListContainer bordered class="mt-6">
					{#each announcements as item (item.title)}
						<article class="px-4 py-4">
							<p class="text-sm font-medium text-text-primary">{item.title}</p>
							<p class="mt-1 text-sm leading-5 text-text-secondary">{item.body}</p>
							<p class="mt-2 text-xs text-text-secondary">{item.time}</p>
						</article>
					{/each}
				</ListContainer>
			{/if}
		</main>

		<aside class="flex h-fit flex-col gap-3 rounded-lg ring-1 ring-border-default bg-surface-primary p-4">
			<button
				type="button"
				class="hover-btn w-full rounded-full px-4 py-3 text-sm font-medium focus-ring {joined ? 'border border-accent-default text-accent-default' : 'bg-accent-default text-text-on-accent'}"
				onclick={toggleJoin}
			>
				{#if joined}
					<span class="flex items-center justify-center gap-1.5">
						<Check size={15} />
						已加入 · 点击退出
					</span>
				{:else}
					加入群组
				{/if}
			</button>
			{#if joined}
				<button type="button" onclick={() => void goto(ROUTES.groupMessage(params.groupId))} class="hover-btn flex w-full items-center justify-center gap-2 rounded-full border border-border-default px-4 py-3 text-sm text-text-primary focus-ring">
					<MessageCircle size={15} />
					打开群聊
				</button>
			{:else}
				<p class="px-2 text-center text-xs leading-5 text-text-secondary">
					<Bell size={13} class="mr-1 inline" />
					加入后可管理群组通知并进入群聊
				</p>
			{/if}
			</aside>
	</div>
</StandardPageLayout>
