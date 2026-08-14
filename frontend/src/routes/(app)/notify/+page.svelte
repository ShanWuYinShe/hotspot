<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, CheckCheck } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import Button from '@/components/ui/button.svelte';
	import NotifyRow from './shared/NotifyRow.svelte';
	import { TYPE_CONFIG, normalizeNotifType, type NotifyViewItem } from './shared/notify-data';
	import { listNotifies, getUnreadCount, markRead, listMyInvites, acceptGroupInvite, rejectGroupInvite } from '@/api';
	import { WS_NOTIFY_EVENT } from '@/lib/ws/client';
	import { formatDateTime } from '@/lib/utils';
	import { ROUTES, ME_SECTION_PATHS } from '@/lib/routes';
	import { follow } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import type { GroupInviteItem } from '@/api/types';

	// 通知列表：类型过滤、单条/全部已读、桌面右侧概览 Rail。
	// 入群邀请：通知行内联接受/拒绝（实时路径）+ 顶部「入群邀请」待办区块
	// （listMyInvites 补拉路径，覆盖通知被清理/未落通知的 pending 邀请）。

const FILTER_TABS = [
	{ id: 'all', label: '全部' },
	// 点赞由 reaction_added 表达(契约已删 moment_liked),tab 语义对齐 reaction。
	{ id: 'reaction', label: '点赞' },
	{ id: 'follow', label: '关注' },
	{ id: 'comment', label: '评论' },
	{ id: 'nearby', label: '附近' }
];

	let notifies = $state<NotifyViewItem[]>([]);
	/** 我收到的待处理入群邀请（listMyInvites，默认仅 pending） */
	let invites = $state<GroupInviteItem[]>([]);
	let filter = $state('all');

	const device = $derived(platform.platform);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	// WS 推送（ws:notify）到来时 +1，触发下方加载 effect 重新拉取通知与未读。
	let reloadKey = $state(0);

	// WS 实时通知：收到推送后重新拉取列表（复用下方同一加载逻辑）。
	$effect(() => {
		const onNotify = () => {
			reloadKey++;
		};
		window.addEventListener(WS_NOTIFY_EVENT, onNotify);
		return () => window.removeEventListener(WS_NOTIFY_EVENT, onNotify);
	});

	$effect(() => {
		if (reloadKey) { /* 依赖占位：WS 推送触发重新加载 */ }
		let cancelled = false;
		async function fetchNotifs() {
			try {
				loading = true;
				loadError = null;
				const [notifRes, unreadRes, invitesRes] = await Promise.all([
					listNotifies().catch(() => ({ code: -1, message: '网络异常', data: { items: [] } })),
					getUnreadCount().catch(() => ({ code: -1, message: '网络异常', data: { total: 0 } })),
					listMyInvites().catch(() => ({ code: -1, message: '网络异常', data: { items: [] } }))
				]);
				if (cancelled) return;

				const notifData = notifRes.code === 0 ? notifRes.data : null;
				if (notifRes.code !== 0) {
					if (!cancelled) loadError = notifRes.message ?? '加载失败';
					return;
				}

				if (invitesRes.code === 0) invites = invitesRes.data?.items ?? [];

				if (notifData) {
					const items: NotifyViewItem[] = (notifData.items || []).map((n, i) => {
						const actor = n.payload?.actor ?? { user_id: '', nickname: '' };
						const groupName = n.payload?.group_name ?? '群组';
						let text = '';
						if (n.type === 'moment_commented') text = n.payload?.comment_preview ?? `${actor.nickname ?? ''} 评论了你的动态`;
						else if (n.type === 'comment_reply') text = `${actor.nickname ?? ''} 回复了你的评论:${n.payload?.reply_content_preview ?? ''}`;
						else if (n.type === 'new_follower') text = `${actor.nickname ?? ''} 关注了你`;
						else if (n.type === 'mutual_follow') text = `${actor.nickname ?? ''} 和你互相关注了`;
						else if (n.type === 'group_invite_received') text = `${actor.nickname ?? ''} 邀请你加入 ${groupName}`;
						else if (n.type === 'group_application_accepted') text = `你加入「${groupName}」的申请已通过`;
						else if (n.type === 'group_application_rejected') text = `${n.payload?.rejected_by?.nickname ?? '管理员'} 拒绝了你的「${groupName}」入群申请`;
						else if (n.type === 'reaction_added') text = `${actor.nickname ?? ''} 做出了反应 ${n.payload?.emoji ?? ''}`;
						else if (n.type === 'group_mentioned') text = `${actor.nickname ?? ''} 在「${groupName}」群聊中 @ 了你${n.payload?.content_preview ? `:${n.payload.content_preview}` : ''}`;
						else if (n.type === 'report_resolved') text = `你举报的内容已处理${n.payload?.note ? `:${n.payload.note}` : ''}`;
						else if (n.type === 'report_dismissed') text = `你的举报未通过审核${n.payload?.note ? `:${n.payload.note}` : ''}`;
						else if (n.type === 'nearby_new_moment') text = `${actor.nickname ?? ''} 在附近发布了新瞬间`;
						else if (n.type === 'system_announcement') text = n.payload?.title ?? '系统公告';
						else text = n.payload?.message ?? '';
						return {
							id: n.notify_id ?? String(i + 1),
							// 归一化为语义键，使图标/计数/分区过滤都能命中 TYPE_CONFIG。
							type: normalizeNotifType(n.type ?? 'reaction_added'),
							read: n.is_read ?? false,
							time: formatDateTime(n.created_at),
							user: { id: actor.user_id ?? '', name: actor.nickname ?? '', avatar: actor.avatar_url ?? '' },
							text,
							momentImage: n.payload?.moment_summary?.cover_url ?? undefined,
							// 入群邀请通知携带 invite_id，供行内接受/拒绝
							inviteId: n.type === 'group_invite_received' ? (n.payload as { invite_id?: string })?.invite_id : undefined
						};
					});
					notifies = items;
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchNotifs();
		return () => {
			cancelled = true;
		};
	});

	const unread = $derived(notifies.filter((n) => !n.read).length);
	let followedBack = $state<Set<string>>(new Set());

	// 桌面 Rail「回关」：直接关注通知中的 actor（spec: users/set_follow）
	const followBack = async (n: NotifyViewItem) => {
		if (!n.user.id || followedBack.has(n.user.id)) return;
		followedBack = new Set([...followedBack, n.user.id]);
		try {
			await follow({ user_id: n.user.id, active: true });
			showToast(`已关注 ${n.user.name}`, 'success');
		} catch {
			followedBack = new Set([...followedBack].filter((id) => id !== n.user.id));
			showToast('操作失败，请重试', 'danger');
		}
	};

	const markAllRead = () => {
		// spec: notify.md → mark_read（参数必显式）：scope='all' 全部已读。乐观更新本地状态。
		notifies = notifies.map((n) => ({ ...n, read: true }));
		void markRead({ scope: 'all' }).catch(() => {});
	};
	const markSingleRead = (id: string) => {
		notifies = notifies.map((n) => (n.id === id ? { ...n, read: true } : n));
		// spec: notify.md → mark_read（参数必显式）：scope='ids' 指定 1-50 条。
		void markRead({ scope: 'ids', notify_ids: [id] }).catch(() => {});
	};
	const remove = (id: string) => {
		notifies = notifies.filter((n) => n.id !== id);
	};

	// 入群邀请：接受/拒绝成功后同步移除待办区块与该邀请的通知行
	const removeInvite = (inviteId: string) => {
		invites = invites.filter((i) => i.invite_id !== inviteId);
		notifies = notifies.filter((n) => n.inviteId !== inviteId);
	};
	const handleAcceptInvite = async (inviteId: string) => {
		try {
			const res = await acceptGroupInvite({ invite_id: inviteId });
			if (res.code === 0) { showToast('已加入群组', 'success'); removeInvite(inviteId); }
			else showToast(res.message || '操作失败', 'danger');
		} catch {
			showToast('操作失败，请重试', 'danger');
		}
	};
	const handleRejectInvite = async (inviteId: string) => {
		try {
			const res = await rejectGroupInvite({ invite_id: inviteId });
			if (res.code === 0) { showToast('已拒绝邀请', 'success'); removeInvite(inviteId); }
			else showToast(res.message || '操作失败', 'danger');
		} catch {
			showToast('操作失败，请重试', 'danger');
		}
	};

	// 归一化后 n.type 已是语义键（reaction/follow/comment/nearby/event），直接与 tab id 比较。
	const filtered = $derived(notifies.filter((n) => filter === 'all' || n.type === filter));
	const unreadGroup = $derived(filtered.filter((n) => !n.read));
	const readGroup = $derived(filtered.filter((n) => n.read));
	const typeStats = $derived(
		(Object.keys(TYPE_CONFIG) as string[]).map((t) => ({ type: t, count: notifies.filter((n) => n.type === t).length }))
	);
</script>
<svelte:head>
	<title>通知 · HotSpot</title>
</svelte:head>


<SplitPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			title="通知"
			sticky
			titleSize={device === 'desktop' ? 20 : device === 'tablet' ? 24 : 22}
			contentClassName="pt-4 pb-0"
			onBack={() => history.back()}
		>
			{#snippet trailing()}
				<!-- 未读计数统一在标题右侧（对齐首页/附近动态样板），与「全部已读」同排 -->
				<div class="flex items-center gap-2">
					{#if unread > 0}
						<span class="text-text-secondary text-xs">{unread} 条未读</span>
						<Button
							type="button"
							variant="secondary"
							size="md"
							class="rounded-full px-3 py-1.5 text-accent-default"
							onclick={markAllRead}
						>
							<CheckCheck class="size-3.5"  strokeWidth={2} />
							全部已读
						</Button>
					{/if}
				</div>
			{/snippet}
			{#snippet below()}
				<!-- 分类过滤统一走 TabBar（下划线） -->
				<div class="mt-3">
					<TabBar items={FILTER_TABS.map((t) => ({ id: t.id, label: t.label }))} value={filter} onChange={(id) => (filter = id)} />
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	<!-- List：每个分组一个连接列表表面（List bordered 契约；heading 在列表外） -->
	<div class="flex flex-col">
				{#if invites.length > 0}
					<!-- 入群邀请待办区块（listMyInvites 补拉路径；接受/拒绝后移除） -->
					<div class="mb-3 flex h-5 items-center gap-2">
						<span class="eyebrow font-semibold text-text-primary">入群邀请</span>
						<span class="badge text-accent-default bg-accent-soft">{invites.length}</span>
					</div>
					<ListContainer bordered class="mb-5">
						{#each invites as inv (inv.invite_id)}
							<div class="flex items-center gap-3 px-3 py-3">
								<img src={inv.group.avatar_url} alt="" loading="lazy" class="h-10 w-10 rounded-full object-cover" />
								<span class="min-w-0 flex-1">
									<span class="block truncate text-sm text-text-primary">{inv.group.name}</span>
									<span class="text-xs text-text-secondary">{inv.inviter.nickname} 邀请你 · {formatDateTime(inv.created_at)}</span>
								</span>
								<button
									type="button"
									onclick={() => void handleAcceptInvite(inv.invite_id)}
									class="hover-btn rounded-full bg-accent-default px-3 py-1.5 text-xs font-medium text-text-on-accent focus-ring"
								>接受</button>
								<button
									type="button"
									onclick={() => void handleRejectInvite(inv.invite_id)}
									class="hover-btn rounded-full border border-border-default px-3 py-1.5 text-xs text-text-primary focus-ring"
								>拒绝</button>
							</div>
						{/each}
					</ListContainer>
				{/if}

				{#if unreadGroup.length > 0}
					<!-- 新消息分组标题（emphasis：accent 点 + 计数 pill，对齐 Figma Make） -->
					<div class="mb-3 flex h-5 items-center gap-2">
						<span aria-hidden="true" class="h-1.5 w-1.5 rounded-full bg-accent-default" ></span>
						<p class="eyebrow font-semibold text-text-primary" >新消息</p>
						<span class="badge text-accent-default bg-accent-soft">{unreadGroup.length}</span>
					</div>
					<ListContainer bordered>
						{#each unreadGroup as n (n.id)}
							<NotifyRow notif={n} onRead={markSingleRead} onRemove={remove} onAccept={(id) => void handleAcceptInvite(id)} onReject={(id) => void handleRejectInvite(id)} />
						{/each}
					</ListContainer>
				{/if}

				{#if readGroup.length > 0}
					<!-- 较早分组标题（非 emphasis：仅 label + flex 分隔线，对齐 Figma Make） -->
					<div class="mb-3 flex h-5 items-center gap-2 {unreadGroup.length > 0 ? 'mt-5' : ''}">
						<p class="eyebrow font-medium text-text-secondary" >较早</p>
						<span aria-hidden="true" class="h-px flex-1 bg-border-default"></span>
					</div>
					<ListContainer bordered>
						{#each readGroup as n (n.id)}
							<NotifyRow notif={n} onRead={markSingleRead} onRemove={remove} onAccept={(id) => void handleAcceptInvite(id)} onReject={(id) => void handleRejectInvite(id)} />
						{/each}
					</ListContainer>
				{/if}

				{#if loadError}
					<LoadError message={loadError} onRetry={() => reloadKey++} />
				{:else if filtered.length === 0}
					<EmptyState title="暂无通知">
						{#snippet icon()}
							<Bell class="size-6 opacity-muted text-text-secondary"  strokeWidth={1.5} />
						{/snippet}
					</EmptyState>
				{/if}
	</div>

	{#snippet rail()}

		<div>
			<RailSection title="今日概览">
				<div class="flex items-baseline gap-1.5 mb-3">
					<span class="text-text-primary text-3xl font-semibold leading-none">{unread}</span>
					<span class="text-text-secondary text-xs">条未读 · 共 {notifies.length} 条</span>
				</div>
				<div class="flex flex-col gap-2">
					{#each typeStats as { type, count }}
						{@const cfg = TYPE_CONFIG[type] ?? TYPE_CONFIG.reaction}
						<button
							type="button"
							class="hover-btn flex items-center gap-2.5 rounded-sm px-2 py-1.5 focus-ring text-left"
							style="background:{filter === type ? cfg.bg : 'transparent'};border:var(--cst-border-width-hairline) solid {filter === type ? cfg.border : 'transparent'};transition:background var(--cst-motion-rapid) var(--cst-ease-standard);"
							onclick={() => (filter = type)}
						>
							<div class="flex items-center justify-center rounded-full shrink-0 size-6" style="background:{cfg.bg};">
								<cfg.icon class="size-3" style="color:{cfg.color};" strokeWidth={1.75} />
							</div>
							<span class="text-text-primary text-sm flex-1">{cfg.label}</span>
							<span class="text-text-secondary text-xs">{count}</span>
						</button>
					{/each}
				</div>
			</RailSection>

			<RailSection title="新关注你的人" action="全部" onAction={() => void goto(`${ME_SECTION_PATHS.relationships}?tab=fans`)}>
				<div class="flex flex-col gap-3">
					{#each notifies.filter((n) => n.type === 'follow').slice(0, 3) as n (n.id)}
						<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.time}>
							{#snippet trailing()}
								<Button
									type="button"
									variant={followedBack.has(n.user.id) ? 'outline' : 'secondary'}
									size="sm"
									class="rounded-full px-2.5 py-1 text-accent-default"
									disabled={followedBack.has(n.user.id)}
									onclick={() => void followBack(n)}
								>
									{followedBack.has(n.user.id) ? '已关注' : '回关'}
								</Button>
							{/snippet}
						</RailUserRow>
					{/each}
				</div>
			</RailSection>

			<RailSection title="此刻附近" action="查看" onAction={() => void goto(ROUTES.nearbyPeople)}>
				<div class="flex flex-col gap-3">
					{#each notifies.filter((n) => n.type === 'nearby').slice(0, 3) as n (n.id)}
						<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.text} />
					{/each}
				</div>
			</RailSection>
		</div>
	{/snippet}
</SplitPageLayout>
