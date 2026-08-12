<script lang="ts">
	import { goto } from '$app/navigation';
	import { ArrowLeft, Bell, Check, MapPin, Megaphone, MessageCircle, MoreHorizontal, PencilLine, Pin, Settings, Trash2, Users } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import GroupAnnouncementDialog from '@/components/groups/GroupAnnouncementDialog.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';
	import { useGroupDetail } from '@/lib/group-data.svelte';
	import { presence, watchPresence } from '@/stores/presence.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { setGroupMemberRole, banGroupMember, removeGroupMember } from '@/api';
	import MemberActionSheet from '@/components/groups/MemberActionSheet.svelte';
	import type { MemberAction } from '@/components/groups/MemberActionSheet.svelte';

	// 群组详情：封面 + 简介/成员/公告三 Tab + 加入按钮 + 桌面侧栏。
	// 状态/加载/WS/操作收敛到 useGroupDetail 工厂；本页只保留 UI + 角色门控。
	// WS 实时：工厂订阅 group 频道，成员/详情/公告变更触发对应刷新；解散由 onDismissed 处理。

	let { params }: { params: { groupId: string } } = $props();

	type GroupTab = 'overview' | 'members' | 'announcements';
	let tab = $state<GroupTab>('overview');

	const gd = useGroupDetail({
		groupId: () => params.groupId,
		onDismissed: () => {
			window.alert('该群组已被解散');
			void goto(ROUTES.nearbyGroups);
		},
		watchPresence,
		isOnline: (id) => presence.isOnline(id)
	});

	// 公告管理 UI 状态（owner/admin 门控）
	let announceOpen = $state(false);
	let announceMode = $state<'create' | 'edit'>('create');
	let announceEditing = $state<{ id: string; title: string; content: string; isPinned: boolean } | null>(null);
	let deleteAnnounceId = $state<string | null>(null);
	const canManageAnnouncements = $derived(gd.joined && (gd.myRole === 'owner' || gd.myRole === 'admin'));

	const openCreateAnnouncement = () => {
		announceMode = 'create';
		announceEditing = null;
		announceOpen = true;
	};
	const openEditAnnouncement = (item: { id: string; title: string; body: string; isPinned: boolean }) => {
		announceMode = 'edit';
		announceEditing = { id: item.id, title: item.title, content: item.body, isPinned: item.isPinned };
		announceOpen = true;
	};
	const handleAnnouncementSubmit = async (title: string, content: string, isPinned: boolean): Promise<boolean> => {
		if (announceMode === 'create') {
			const res = await gd.createAnnouncement(title, content, isPinned);
			if (res.code === 0) { showToast('公告已发布', 'success'); return true; }
			showToast(res.message || '发布失败', 'danger'); return false;
		} else if (announceEditing) {
			const res = await gd.updateAnnouncement(announceEditing.id, { title, content, is_pinned: isPinned });
			if (res.code === 0) { showToast('公告已更新', 'success'); return true; }
			showToast(res.message || '保存失败', 'danger'); return false;
		}
		return false;
	};
	const handleDeleteAnnouncement = async () => {
		if (!deleteAnnounceId) return;
		const res = await gd.deleteAnnouncement(deleteAnnounceId);
		if (res.code === 0) showToast('公告已删除', 'success');
		else showToast(res.message || '删除失败', 'danger');
		deleteAnnounceId = null;
	};

	// 成员管理（owner/admin；不能操作群主/自己）
	const meId = $derived(auth.user?.user_id ?? '');
	const canManageMembers = $derived(gd.joined && (gd.myRole === 'owner' || gd.myRole === 'admin'));
	const canManageMember = (m: { userId: string; role: string }) =>
		canManageMembers && m.role !== 'owner' && m.userId !== meId;
	let actionMember = $state<{ userId: string; name: string; role: string } | null>(null);

	const handleMemberAction = async (action: MemberAction) => {
		if (!actionMember) return;
		const m = actionMember;
		const gid = params.groupId;
		try {
			let res: { code: number; message?: string } = { code: -1 };
			if (action === 'set_admin') res = await setGroupMemberRole({ group_id: gid, user_id: m.userId, role: 'admin' });
			else if (action === 'unset_admin') res = await setGroupMemberRole({ group_id: gid, user_id: m.userId, role: 'member' });
			else if (action === 'ban') res = await banGroupMember({ group_id: gid, user_id: m.userId });
			else if (action === 'remove') res = await removeGroupMember({ group_id: gid, user_id: m.userId });
			if (res.code === 0) {
				showToast('操作成功', 'success');
				void gd.loadMembers(gid);
				actionMember = null;
			} else {
				showToast(res.message || '操作失败', 'danger');
			}
		} catch {
			showToast('操作失败，请重试', 'danger');
		}
	};
</script>
<svelte:head>
	<title>群组主页 · HotSpot</title>
</svelte:head>


{#if gd.loadError}
	<LoadError message={gd.loadError} onRetry={() => void gd.loadAll(params.groupId ?? '')} />
{:else}
<StandardPageLayout contentClassName="animate-fade-in pb-6">
	<!-- 全幅封面（负 margin 抵消 ContentColumn gutter） -->
	<div class="relative h-[var(--cst-cover-height)] overflow-hidden -mx-[var(--cst-page-gutter)]">
		<img src={gd.detail.banner} alt={`${gd.detail.name} 的群组封面`} class="h-full w-full object-cover" />
		<!-- gradient scrim 走 inline background（gradient 不能作 background-color 值，
		     与 NearbyImageCard 的 --cst-surface-media-scrim-card 用法一致） -->
		<div class="absolute inset-0" style="background:var(--cst-surface-cover-scrim);"></div>
		<div class="absolute left-5 top-4">
			<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} variant="overlay" tone="inverse" />
		</div>
		<div class="absolute bottom-5 left-6 right-6">
			<span class="rounded-full bg-surface-glass-badge px-2.5 py-1 text-xs text-content-inverse backdrop-blur-sm">{gd.detail.category}</span>
			<h1 class="mt-3 text-2xl md:text-3xl font-medium tracking-title text-content-inverse">{gd.detail.name}</h1>
		</div>
	</div>

	<div class="grid gap-8 pt-6 lg:grid-cols-[minmax(0,1fr)_80]">
		<main class="min-w-0">
			<div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-text-secondary">
				<span class="flex items-center gap-1.5">
					<Users size={14} />
					{gd.detail.members} 位成员
				</span>
				<span class="flex items-center gap-1.5 text-status-online">
					<span class="h-2 w-2 rounded-full bg-status-online"></span>
					{gd.onlineMemberCount} 人在线
				</span>
				<span class="flex items-center gap-1.5">
					<MapPin size={14} />
					距你 {gd.detail.distance}
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
					<p class="max-w-3xl text-sm leading-7 text-text-secondary">{gd.detail.description}</p>
				</div>
			{:else if tab === 'members'}
				<!-- 空旷内容区 → List bordered（有边框列表契约）；列表行不设独立边框 -->
				<ListContainer bordered class="mt-6">
					{#each gd.members as member (member.id)}
						<ListItem
							onClick={() => {
								if (member.userId) void goto(ROUTES.user(member.userId));
							}}
							ariaLabel={`查看 ${member.name} 的资料`}
							class="app-list-row app-list-row--standard"
						>
							<div class="relative">
								<img src={member.avatar} alt="" loading="lazy" class="h-10 w-10 rounded-full object-cover" />
								{#if gd.onlineMemberIds.has(member.userId)}
									<span class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-surface-primary bg-status-online"></span>
								{/if}
							</div>
							<span class="min-w-0 flex-1">
								<span class="block text-sm text-text-primary">{member.name}</span>
								<span class="text-xs text-text-secondary">{member.role}</span>
							</span>
							<span class="text-xs text-text-secondary">查看资料</span>
							{#if canManageMember(member)}
								<button
									type="button"
									onclick={(e) => { e.stopPropagation(); actionMember = member; }}
									class="flex h-7 w-7 items-center justify-center rounded-full text-text-tertiary outline-none hover:bg-interactive-hover hover-btn scale-btn focus-visible:ring-2 focus-visible:ring-accent-default"
									aria-label={`管理 ${member.name}`}
								>
									<MoreHorizontal size={15} strokeWidth={2} />
								</button>
							{/if}
						</ListItem>
					{/each}
				</ListContainer>
			{:else}
				<div class="mt-6 flex items-center justify-between">
					<p class="text-sm text-text-secondary">群公告</p>
					{#if canManageAnnouncements}
						<button type="button" onclick={openCreateAnnouncement} class="hover-btn flex items-center gap-1 rounded-full border border-border-default px-3 py-1.5 text-xs text-text-primary focus-ring">
							<Megaphone size={12} strokeWidth={1.75} />
							发公告
						</button>
					{/if}
				</div>
				<!-- 空旷内容区 → List bordered（有边框列表契约） -->
				<ListContainer bordered class="mt-3">
					{#each gd.announcements as item (item.id)}
						<article class="relative px-4 py-4">
							<div class="flex items-center gap-1.5 pr-16">
								{#if item.isPinned}
									<Pin size={12} class="shrink-0 text-accent-default" strokeWidth={1.75} />
								{/if}
								<p class="text-sm font-medium text-text-primary">{item.title}</p>
							</div>
							<p class="mt-1 text-sm leading-5 text-text-secondary">{item.body}</p>
							<p class="mt-2 text-xs text-text-secondary">{item.time}</p>
							{#if canManageAnnouncements}
								<div class="absolute right-3 top-3 flex gap-1">
									<button type="button" onclick={() => openEditAnnouncement(item)} class="flex h-6 w-6 items-center justify-center rounded-full text-text-tertiary outline-none hover:bg-interactive-hover hover-btn scale-btn focus-visible:ring-2 focus-visible:ring-accent-default" aria-label="编辑公告">
										<PencilLine size={12} strokeWidth={1.75} />
									</button>
									<button type="button" onclick={() => (deleteAnnounceId = item.id)} class="flex h-6 w-6 items-center justify-center rounded-full text-status-danger outline-none hover:bg-interactive-hover hover-btn scale-btn focus-visible:ring-2 focus-visible:ring-status-danger" aria-label="删除公告">
										<Trash2 size={12} strokeWidth={1.75} />
									</button>
								</div>
							{/if}
						</article>
					{:else}
						<p class="px-4 py-6 text-center text-sm text-text-secondary">暂无公告</p>
					{/each}
				</ListContainer>
			{/if}
		</main>

		<aside class="flex h-fit flex-col gap-3 rounded-lg ring-1 ring-border-default bg-surface-primary p-4">
			<button
				type="button"
				class="hover-btn w-full rounded-full px-4 py-3 text-sm font-medium focus-ring {gd.joined ? 'border border-accent-default text-accent-default' : 'bg-accent-default text-text-on-accent'}"
				onclick={gd.toggleJoin}
			>
				{#if gd.joined}
					<span class="flex items-center justify-center gap-1.5">
						<Check size={15} />
						已加入 · 点击退出
					</span>
				{:else}
					加入群组
				{/if}
			</button>
			{#if gd.joined && (gd.myRole === 'owner' || gd.myRole === 'admin')}
				<!-- 角色门控：仅群主/管理员可见的管理入口（占位，实际管理 UI 由后续批次填充） -->
				<button
					type="button"
					onclick={() => { /* TODO 群管理批次 N：打开群管理抽屉 */ }}
					class="hover-btn flex w-full items-center justify-center gap-2 rounded-full border border-border-default px-4 py-3 text-sm text-text-primary focus-ring"
				>
					<Settings size={15} />
					群管理
				</button>
			{/if}
			{#if gd.joined}
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
{/if}

<Dialog.Root open={deleteAnnounceId !== null} onOpenChange={(next) => (next ? undefined : (deleteAnnounceId = null))}>
	<Dialog.Content side="center" class="inset-x-4 mx-auto max-w-sm rounded-xl bg-surface-primary p-5" aria-label="确认删除公告" style="box-shadow:var(--cst-elevation-modal);">
		<p class="text-md text-text-primary">删除这条公告？</p>
		<p class="mt-1 text-sm text-text-secondary">删除后无法恢复。</p>
		<div class="mt-4 flex items-center justify-end gap-3">
			<button type="button" onclick={() => (deleteAnnounceId = null)} class="hover-btn rounded-full px-4 py-2 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">取消</button>
			<button type="button" onclick={handleDeleteAnnouncement} class="hover-btn rounded-full px-5 py-2 text-sm font-medium text-text-on-accent outline-none focus-visible:ring-2 focus-visible:ring-status-danger enabled:bg-status-danger">确认删除</button>
		</div>
	</Dialog.Content>
</Dialog.Root>

<GroupAnnouncementDialog
	open={announceOpen}
	mode={announceMode}
	initial={announceEditing ?? undefined}
	onClose={() => (announceOpen = false)}
	onSubmit={handleAnnouncementSubmit}
/>

<MemberActionSheet
	open={actionMember !== null}
	member={actionMember}
	onClose={() => (actionMember = null)}
	onSelect={handleMemberAction}
/>
