<script lang="ts">
	import { untrack } from 'svelte';
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
import FollowButton from '@/components/ui/FollowButton.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
import ListEnd from '@/components/data-display/ListEnd.svelte';
	import Button from '@/components/ui/button.svelte';
	import NotifyRow from './shared/NotifyRow.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import { TYPE_CONFIG, normalizeNotifType, type NotifyViewItem } from './shared/notify-data';
	import { listNotifies, getUnreadCount, markRead, listMyInvites, acceptGroupInvite, rejectGroupInvite, listFollowing, netError } from '@/api';
	import { setNotifyUnread, unread as globalUnread } from '@/stores/unread.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { WS_NOTIFY_EVENT } from '@/lib/ws/client';
	import { formatDateTime } from '@/lib/utils';
	import { ROUTES } from '@/lib/routes';
	import { relationshipsHref } from '@/lib/relationships';
	import { follow } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import type { GroupInviteItem, NotifyItem } from '@/api/types';
	import { smartBack } from '@/lib/smart-back';

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
	/** invites 是否成功加载：用于判定通知行邀请动作的显隐（加载失败时
	保持旧行为——按钮照常显示，点击走后端诚实报错）。 */
	let invitesLoaded = false;
	let filter = $state('all');
	// 分页（审查 M5）：长期积累的通知此前一次全量拉取；改 cursor 分页 + 手动加载更多。
	const NOTIFY_PAGE_SIZE = 30;
	let nextCursor = $state<string | null>(null);
	let loadingMore = $state(false);

	const device = $derived(platform.platform);
	let needsAuth = $state(false);
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
				needsAuth = false;
				// 仅首刷（无数据）展示整页 loading；WS 触发的后台刷新静默换数据，
				// 避免用户浏览列表时每来一条新通知整表闪一次 spinner。
				// notifies 读取必须 untrack：数组回写会成为 effect 依赖 → 无限请求循环（同 messages）
				if (untrack(() => !notifies.length)) loading = true;
				loadError = null;
				const [notifRes, unreadRes, invitesRes, followingRes] = await Promise.all([
					listNotifies({ pagination: { limit: NOTIFY_PAGE_SIZE } }).catch(() => netError({ items: [], next_cursor: null })),
					getUnreadCount().catch(() => netError({ total: 0 })),
					listMyInvites().catch(() => netError({ items: [] })),
					// rail 回关钮诚实化：已关注的 actor 直接显示「已关注」（后端 set_follow
					// 幂等 no-op 不会出错，但按钮不应显示与真实关系相反的动作）。
					listFollowing({ user_id: auth.user?.user_id ?? '', pagination: { limit: 100 } }).catch(() => netError({ items: [], next_cursor: null }))
				]);
				if (cancelled) return;

				const notifData = notifRes.code === 0 ? notifRes.data : null;
				if (notifRes.code === 10100) {
					// 未登录：给登录引导而非裸技术报错
					if (!cancelled) needsAuth = true;
					return;
				}
				if (notifRes.code !== 0) {
					// 后台刷新失败不整页替换为错误态：已有列表时降级为 toast 提示
					if (!cancelled) {
						if (notifies.length) showToast(notifRes.message || '通知刷新失败', 'danger');
						else loadError = notifRes.message ?? '加载失败';
					}
					return;
				}
				if (!cancelled) nextCursor = notifData?.next_cursor ?? null;

				if (invitesRes.code === 0) {
					invites = invitesRes.data?.items ?? [];
					invitesLoaded = true;
				}

				if (followingRes.code === 0) {
					const ids = (followingRes.data?.items ?? [])
						.map((u) => u.user?.user?.user_id ?? '')
						.filter(Boolean);
					followedBack = new Set([...followedBack, ...ids]);
				}

				if (notifData) {
					notifies = stripDecidedInviteActions(toViewItems(notifData.items || [], 0));
				}
				// 服务端未读真相回写全局徽标（markRead/全部已读后同样经此收敛）
				if (!cancelled && unreadRes.code === 0 && unreadRes.data) {
					setNotifyUnread(unreadRes.data.total);
				}
			} catch {
				if (!cancelled) {
					if (notifies.length) showToast('通知刷新失败，请稍后重试', 'danger');
					else loadError = '加载失败，请稍后重试';
				}
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchNotifs();
		return () => {
			cancelled = true;
		};
	});

	/** 后端通知 → 视图模型（首页与加载更多共用；offset 供 fallback id 去重）。 */
	function toViewItems(raw: NotifyItem[], offset: number): NotifyViewItem[] {
		const sorted = raw.slice().sort((a, b) =>
			String(b.created_at ?? '').localeCompare(String(a.created_at ?? ''))
		);
		return sorted.map((n, i) => {
			const actor = n.payload?.actor ?? { user_id: '', nickname: '', avatar_url: null };
			const groupName = n.payload?.group_name ?? '群组';
			// 行内已渲染 actor 昵称（NotifyRow user.name），text 只承载动作语义，
			// 避免昵称出现两遍。
			let text = '';
			if (n.type === 'moment_commented') text = n.payload?.comment_preview ?? '评论了你的动态';
			else if (n.type === 'comment_reply') text = `回复了你的评论：${n.payload?.reply_content_preview ?? ''}`;
			else if (n.type === 'new_follower') text = '关注了你';
			else if (n.type === 'mutual_follow') text = '和你互相关注了';
			else if (n.type === 'group_invite_received') text = `邀请你加入 ${groupName}`;
			else if (n.type === 'group_application_accepted') text = `你加入「${groupName}」的申请已通过`;
			else if (n.type === 'group_application_rejected') text = `${n.payload?.rejected_by?.nickname ?? '管理员'} 拒绝了你的「${groupName}」入群申请`;
			else if (n.type === 'reaction_added') text = `做出了反应 ${n.payload?.emoji ?? ''}`;
			else if (n.type === 'group_mentioned') text = `在「${groupName}」群聊中 @ 了你${n.payload?.content_preview ? `：${n.payload.content_preview}` : ''}`;
			else if (n.type === 'report_resolved') text = `你举报的内容已处理${n.payload?.note ? `：${n.payload.note}` : ''}`;
			else if (n.type === 'report_dismissed') text = `你的举报未通过审核${n.payload?.note ? `：${n.payload.note}` : ''}`;
			else if (n.type === 'nearby_new_moment') text = '在附近发布了新瞬间';
			else if (n.type === 'system_announcement') text = n.payload?.title ?? '系统公告';
			else text = n.payload?.message ?? '';
			return {
				id: n.notify_id ?? String(offset + i + 1),
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
	}

	/** 已决邀请的通知行不再提供接受/拒绝（后端 listMyInvites 默认 pending；
	行残留时点之只会收到“已过期”错误）。invites 未成功加载时保持原样。 */
	function stripDecidedInviteActions(items: NotifyViewItem[]): NotifyViewItem[] {
		if (!invitesLoaded) return items;
		const pending = new Set(invites.map((i) => i.invite_id));
		return items.map((n) => (n.inviteId && !pending.has(n.inviteId) ? { ...n, inviteId: undefined } : n));
	}

	/** 加载下一页（cursor keyset，服务端按时间倒序，追加即保持全局有序）。 */
	async function loadMoreNotifs() {
		if (nextCursor === null || loadingMore) return;
		loadingMore = true;
		try {
			const res = await listNotifies({ pagination: { limit: NOTIFY_PAGE_SIZE, cursor: nextCursor } }).catch(() => netError({ items: [], next_cursor: null }));
			if (res.code === 0 && res.data) {
				notifies = stripDecidedInviteActions([...notifies, ...toViewItems(res.data.items || [], notifies.length)]);
				nextCursor = res.data.next_cursor ?? null;
			}
		} finally {
			loadingMore = false;
		}
	}

	const unread = $derived(notifies.filter((n) => !n.read).length);
	let followedBack = $state<Set<string>>(new Set());
	// A 关注 B 成互关时双方都会收到 mutual_follow，actor 都是 B——A 自己 rail 里的
	// 行 actor 就是自己，回关自己永远失败（诚实报错）。这类行不再提供回关按钮。
	const meId = $derived(auth.user?.user_id ?? '');

	// 桌面 Rail「回关」：直接关注通知中的 actor（spec: users/set_follow）。
	// code!==0 视为失败并回滚（此前只捕获网络异常：业务失败会误报成功 toast
	// 且按钮永久卡死在已关注；与用户详情/附近的人/共享模块同口径）。
	const followBack = async (n: NotifyViewItem) => {
		if (!n.user.id || followedBack.has(n.user.id)) return;
		followedBack = new Set([...followedBack, n.user.id]);
		try {
			const res = await follow({ user_id: n.user.id, active: true });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast(`已关注 ${n.user.name}`, 'success');
		} catch (err) {
			followedBack = new Set([...followedBack].filter((id) => id !== n.user.id));
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		}
	};

	const markAllRead = () => {
		// spec: notify.md → mark_read（参数必显式）：scope='all' 全部已读。乐观更新 + 失败回滚。
	const snapshot = notifies;
	notifies = notifies.map((n) => ({ ...n, read: true }));
	setNotifyUnread(0);
	void markRead({ scope: 'all' }).catch(() => {
		notifies = snapshot;
		// 回滚：向服务端复核未读真相（徽标与列表一并还原）
		void getUnreadCount().then((r) => {
			if (r.code === 0 && r.data) setNotifyUnread(r.data.total);
		});
		showToast('全部已读失败，请重试', 'danger');
	});
};
const markSingleRead = (id: string) => {
	// 先取原状态再变异：仅原本未读的项才递减全局徽标
	const wasUnread = notifies.find((n) => n.id === id)?.read === false;
	notifies = notifies.map((n) => (n.id === id ? { ...n, read: true } : n));
	// spec: notify.md → mark_read（参数必显式）：scope='ids' 指定 1-50 条。
	if (wasUnread) setNotifyUnread(Math.max(0, globalUnread.notify - 1));
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
	// 邀请处理 pending（响应式）：接受/拒绝进行中禁点防重复提交
	let invitePending = $state(new Set<string>());
	const beginInvite = (inviteId: string) => {
		invitePending = new Set(invitePending).add(inviteId);
	};
	const endInvite = (inviteId: string) => {
		const rest = new Set(invitePending);
		rest.delete(inviteId);
		invitePending = rest;
	};
	const handleAcceptInvite = async (inviteId: string) => {
		if (invitePending.has(inviteId)) return;
		beginInvite(inviteId);
		try {
			const res = await acceptGroupInvite({ invite_id: inviteId });
			if (res.code === 0) { showToast('已加入群组', 'success'); removeInvite(inviteId); }
			else showToast(res.message || '操作失败', 'danger');
		} catch {
			showToast('操作失败，请重试', 'danger');
		} finally {
			endInvite(inviteId);
		}
	};
	const handleRejectInvite = async (inviteId: string) => {
		if (invitePending.has(inviteId)) return;
		beginInvite(inviteId);
		try {
			const res = await rejectGroupInvite({ invite_id: inviteId });
			if (res.code === 0) { showToast('已拒绝邀请', 'success'); removeInvite(inviteId); }
			else showToast(res.message || '操作失败', 'danger');
		} catch {
			showToast('操作失败，请重试', 'danger');
		} finally {
			endInvite(inviteId);
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
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
			onBack={() => smartBack()}
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
				<!-- 分类过滤统一走 TabBar（胶囊） -->
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
						<span class="eyebrow">入群邀请</span>
						<span class="badge text-accent-default bg-accent-soft">{invites.length}</span>
					</div>
					<ListContainer bordered class="mb-5">
						{#each invites as inv (inv.invite_id)}
							<div class="flex items-center gap-3.5 px-3 py-3.5">
								<Avatar src={inv.group.avatar_url} name={inv.group.name} size="lg" shape="rounded" />
								<span class="min-w-0 flex-1">
									<span class="block truncate text-sm text-text-primary">{inv.group.name}</span>
									<span class="text-xs text-text-secondary">{inv.inviter.nickname} 邀请你 · {formatDateTime(inv.created_at)}</span>
								</span>
								<button
									type="button"
									onclick={() => void handleAcceptInvite(inv.invite_id)}
									disabled={invitePending.has(inv.invite_id)}
									class="hover-btn rounded-full bg-accent-default px-3.5 py-1.5 text-sm font-medium text-text-on-accent shadow-raised-sm focus-ring hover:shadow-raised-md disabled:pointer-events-none disabled:opacity-faint"
								>接受</button>
								<button
									type="button"
									onclick={() => void handleRejectInvite(inv.invite_id)}
									disabled={invitePending.has(inv.invite_id)}
									class="hover-btn rounded-full border border-border-default px-3.5 py-1.5 text-sm text-text-primary focus-ring hover:border-border-strong disabled:pointer-events-none disabled:opacity-faint"
								>拒绝</button>
							</div>
						{/each}
					</ListContainer>
				{/if}

				{#if unreadGroup.length > 0}
					<!-- 新消息分组标题（emphasis：accent 点 + 计数 pill，对齐 Figma Make） -->
					<div class="mb-3 flex h-5 items-center gap-2">
						<span aria-hidden="true" class="h-1.5 w-1.5 rounded-full bg-accent-default" ></span>
						<p class="eyebrow">新消息</p>
						<span class="badge text-accent-default bg-accent-soft">{unreadGroup.length}</span>
					</div>
					<ListContainer bordered class="mb-5">
						{#each unreadGroup as n (n.id)}
							<NotifyRow notif={n} actionPending={n.inviteId ? invitePending.has(n.inviteId) : false} onRead={markSingleRead} onRemove={remove} onAccept={(id) => void handleAcceptInvite(id)} onReject={(id) => void handleRejectInvite(id)} />
						{/each}
					</ListContainer>
				{/if}

				{#if readGroup.length > 0}
					<!-- 较早分组标题（非 emphasis：仅 label + flex 分隔线，对齐 Figma Make） -->
					<div class="mb-3 flex h-5 items-center gap-2 {unreadGroup.length > 0 ? 'mt-5' : ''}">
						<p class="eyebrow">较早</p>
						<span aria-hidden="true" class="h-px flex-1 bg-border-default"></span>
					</div>
					<ListContainer bordered>
						{#each readGroup as n (n.id)}
							<NotifyRow notif={n} actionPending={n.inviteId ? invitePending.has(n.inviteId) : false} onRead={markSingleRead} onRemove={remove} onAccept={(id) => void handleAcceptInvite(id)} onReject={(id) => void handleRejectInvite(id)} />
						{/each}
					</ListContainer>
				{/if}

				{#if !loading && nextCursor !== null}
					<div class="mt-5">
						<LoadMoreButton label="加载更早通知" loading={loadingMore} onLoad={() => void loadMoreNotifs()} />
					</div>
				{:else if filtered.length > 0 && !loadError && nextCursor === null}
					<!-- 分页耗尽契约（frontend/AGENTS.md）：到底必须渲染 ListEnd -->
					<ListEnd />
				{/if}

				{#if loading}
					<InlineLoading label="加载通知…" />
				{:else if loadError}
					<LoadError message={loadError} onRetry={() => reloadKey++} />
				{:else if needsAuth}
					<div class="flex min-h-[50vh] items-center justify-center">
						<NeedsAuthNotice title="登录后查看" description="收到的点赞、评论与关注会出现在这里" />
					</div>
				{:else if filtered.length === 0}
					<EmptyState
						title={filter === 'all' ? '暂无通知' : '该分类下暂无通知'}
						description="收到的点赞、评论与关注会出现在这里"
					>
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
						<!-- 纯展示：类型概览数字（过滤只走标题下 TabBar，不在此写 filter） -->
						<div class="flex items-center gap-2.5 px-2 py-1.5">
							<div class="flex items-center justify-center rounded-full shrink-0 size-6" style="background:{cfg.bg};">
								<cfg.icon class="size-3" style="color:{cfg.color};" strokeWidth={1.75} />
							</div>
							<span class="text-text-primary text-sm flex-1">{cfg.label}</span>
							<span class="text-text-secondary text-xs">{count}</span>
						</div>
					{/each}
				</div>
			</RailSection>

			{#if notifies.filter((n) => n.type === 'follow').length > 0}
			<RailSection title="新关注你的人" action="全部" onAction={() => void goto(relationshipsHref('fans'))}>
				<div class="flex flex-col gap-3">
					{#each notifies.filter((n) => n.type === 'follow' && n.user.id !== meId).slice(0, 3) as n (n.id)}
						<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.time}>
							{#snippet trailing()}
								<FollowButton
									active={followedBack.has(n.user.id)}
									disabled={followedBack.has(n.user.id)}
									labelActive="已关注"
									labelInactive="回关"
									onToggle={() => void followBack(n)}
								/>
							{/snippet}
						</RailUserRow>
					{/each}
				</div>
			</RailSection>
			{/if}

			{#if notifies.filter((n) => n.type === 'nearby').length > 0}
			<RailSection title="此刻附近" action="查看" onAction={() => void goto(ROUTES.nearbyPeople)}>
				<div class="flex flex-col gap-3">
					{#each notifies.filter((n) => n.type === 'nearby').slice(0, 3) as n (n.id)}
						<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.text} />
					{/each}
				</div>
			</RailSection>
			{/if}
		</div>
	{/snippet}
</SplitPageLayout>
