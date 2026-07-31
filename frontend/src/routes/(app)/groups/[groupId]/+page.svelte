<script lang="ts">
	import { goto } from '$app/navigation';
	import { ArrowLeft, Bell, CalendarDays, Check, MapPin, MessageCircle, Users } from '@lucide/svelte';
	import ActionIconButton from '@/components/shared/ActionIconButton.svelte';
	import { ROUTES } from '@/lib/routes';
	import { getGroupDetail, getGroupMembers, getGroupAnnouncements } from '@/api';

	// 群组详情：封面 + 简介/成员/公告三 Tab + 加入按钮 + 桌面侧栏。
	// 迁移自 React GroupDetailPage.tsx。

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

	$effect(() => {
		let cancelled = false;
		const gid = params.groupId ?? '';
		async function fetchGroup() {
			try {
				loading = true;
				const [detailRes, membersRes, announcementsRes] = await Promise.all([
					getGroupDetail(gid).catch(() => ({ code: -1, data: null })),
					getGroupMembers(gid).catch(() => ({ code: -1, data: { items: [] } })),
					getGroupAnnouncements(gid).catch(() => ({ code: -1, data: { items: [] } }))
				]);
				if (cancelled) return;

				const detailData = (detailRes.code === 0 ? detailRes.data : null) as any;
				const membersData = (membersRes.code === 0 ? membersRes.data : null) as any;
				const announcementsData = (announcementsRes.code === 0 ? announcementsRes.data : null) as any;

				if (detailData) {
					const g = detailData.group ?? detailData;
					group = {
						name: g.name ?? group.name,
						category: (detailData.tags?.[0] ?? g.category ?? group.category) || '同城',
						members: g.members_count ?? group.members,
						online: g.online_count ?? group.online,
						distance: detailData.location?.distance ?? group.distance,
						description: detailData.description ?? group.description,
						banner: g.avatar_url ?? group.banner
					};
					if (detailData.my_state?.is_joined) joined = true;
				}

				if (membersData) {
					const items: Member[] = (membersData.items || []).map((m: any, i: number) => ({
						id: m.user?.user_id ?? i + 1,
						name: m.user?.nickname ?? m.name ?? '',
						role: m.role ?? '成员',
						online: m.user?.presence_status === 'online' || m.online || false,
						avatar: m.user?.avatar_url ?? m.avatar ?? ''
					}));
					members = items;
				}

				if (announcementsData) {
					const items: Announcement[] = (announcementsData.items || []).map((a: any) => ({
						title: a.title ?? a.name ?? '',
						body: a.body ?? a.content ?? a.text ?? '',
						time: a.time ?? a.created_at ?? ''
					}));
					announcements = items;
				}
			} catch {
				// empty data is normal
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchGroup();
		return () => {
			cancelled = true;
		};
	});
</script>

<div class="content-mode-wide min-h-full bg-[var(--neu-bg)]">
	<div class="relative h-[clamp(220px,28vw,340px)] overflow-hidden">
		<img src={group.banner} alt={`${group.name} 的群组封面`} class="h-full w-full object-cover" />
		<div class="absolute inset-0 bg-[linear-gradient(to_top,rgba(0,0,0,.7),transparent_65%)]"></div>
		<div class="absolute left-5 top-4">
			<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} variant="overlay" tone="inverse" />
		</div>
		<div class="absolute bottom-5 left-6 right-6">
			<span class="rounded-full bg-white/18 px-2.5 py-1 text-[11px] text-white backdrop-blur-sm">{group.category}</span>
			<h1 class="mt-3 text-[clamp(24px,3vw,34px)] font-medium tracking-[-0.02em] text-white">{group.name}</h1>
		</div>
	</div>

	<div class="grid gap-8 px-6 py-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:px-8">
		<main class="min-w-0">
			<div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-[13px] text-[var(--neu-text-muted)]">
				<span class="flex items-center gap-1.5">
					<Users size={14} />
					{group.members} 位成员
				</span>
				<span class="flex items-center gap-1.5 text-[var(--semantic-online)]">
					<span class="h-2 w-2 rounded-full bg-[var(--semantic-online)]"></span>
					{group.online} 人在线
				</span>
				<span class="flex items-center gap-1.5">
					<MapPin size={14} />
					距你 {group.distance}
				</span>
			</div>

			<!-- Tabs -->
			<div class="mt-6 flex gap-1 border-b border-[var(--neu-border)]">
				{#each ['overview', 'members', 'announcements'] as item (item)}
					<button
						type="button"
						onclick={() => (tab = item as GroupTab)}
						class="neu-hover-btn relative px-3 py-2.5 text-[13px] neu-focus-ring {tab === item ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}"
					>
						{item === 'overview' ? '简介' : item === 'members' ? '成员' : '公告'}
						{#if tab === item}
							<span class="absolute inset-x-3 bottom-0 h-0.5 rounded-full bg-[var(--neu-accent-primary)]"></span>
						{/if}
					</button>
				{/each}
			</div>

			{#if tab === 'overview'}
				<div class="py-6">
					<p class="max-w-3xl text-[15px] leading-7 text-[var(--neu-text-muted)]">{group.description}</p>
					<div class="mt-7 rounded-[18px] ring-1 ring-[var(--neu-border)] p-4">
						<p class="flex items-center gap-2 text-[14px] font-medium text-[var(--neu-text)]">
							<CalendarDays size={15} class="text-[var(--neu-accent-primary)]" />
							群组节奏
						</p>
						<p class="mt-2 text-[12px] leading-5 text-[var(--neu-text-muted)]">每周三、周末晚间有主题讨论。加入后可接收新公告与群聊提醒。</p>
					</div>
				</div>
			{:else if tab === 'members'}
				<div class="divide-y divide-[var(--neu-border)]">
					{#each members as member (member.id)}
						<button type="button" class="neu-hover-row app-list-row app-list-row--standard neu-row-hover rounded-[12px] outline-none" onclick={() => void goto(ROUTES.user(member.id))}>
							<div class="relative">
								<img src={member.avatar} alt="" class="h-10 w-10 rounded-full object-cover" />
								{#if member.online}
									<span class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-[var(--neu-bg)] bg-[var(--semantic-online)]"></span>
								{/if}
							</div>
							<span class="min-w-0 flex-1">
								<span class="block text-[14px] text-[var(--neu-text)]">{member.name}</span>
								<span class="text-[11px] text-[var(--neu-text-muted)]">{member.role}</span>
							</span>
							<span class="text-[11px] text-[var(--neu-text-muted)]">查看资料</span>
						</button>
					{/each}
				</div>
			{:else}
				<div class="divide-y divide-[var(--neu-border)]">
					{#each announcements as item (item.title)}
						<article class="py-4">
							<p class="text-[14px] font-medium text-[var(--neu-text)]">{item.title}</p>
							<p class="mt-1 text-[12px] leading-5 text-[var(--neu-text-muted)]">{item.body}</p>
							<p class="mt-2 text-[11px] text-[var(--neu-text-muted)]">{item.time}</p>
						</article>
					{/each}
				</div>
			{/if}
		</main>

		<aside class="flex h-fit flex-col gap-3 rounded-[20px] ring-1 ring-[var(--neu-border)] bg-[var(--neu-bg)] p-4">
			<button
				type="button"
				class="neu-hover-btn w-full rounded-full px-4 py-3 text-[13px] font-medium neu-focus-ring {joined ? 'border border-[var(--neu-accent-primary)] text-[var(--neu-accent-primary)]' : 'bg-[var(--neu-accent-primary)] text-white'}"
				onclick={() => (joined = !joined)}
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
				<button type="button" onclick={() => void goto(ROUTES.messages)} class="neu-hover-btn flex w-full items-center justify-center gap-2 rounded-full border border-[var(--neu-border)] px-4 py-3 text-[13px] text-[var(--neu-text)] neu-focus-ring">
					<MessageCircle size={15} />
					打开群聊
				</button>
			{:else}
				<p class="px-2 text-center text-[11px] leading-5 text-[var(--neu-text-muted)]">
					<Bell size={13} class="mr-1 inline" />
					加入后可管理群组通知并进入群聊
				</p>
			{/if}
		</aside>
	</div>
</div>
