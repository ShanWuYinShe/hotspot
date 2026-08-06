<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, CheckCheck } from '@lucide/svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import NotificationRow from './NotificationRow.svelte';
	import { TYPE_CONFIG, normalizeNotifType, type NotificationItem } from './notification-data';
	import { listNotifications, getUnreadCount, markRead } from '@/api';
	import { WS_NOTIFY_EVENT } from '@/lib/ws/client';
	import { formatDateTime } from '@/lib/utils';
	import { ROUTES, ME_SECTION_PATHS } from '@/lib/routes';

	// 通知列表：类型过滤、单条/全部已读、桌面右侧概览 Rail。

const FILTER_TABS = [
	{ id: 'all', label: '全部' },
	// 点赞由 reaction_added 表达(契约已删 moment_liked),tab 语义对齐 reaction。
	{ id: 'reaction', label: '点赞' },
	{ id: 'follow', label: '关注' },
	{ id: 'comment', label: '评论' },
	{ id: 'nearby', label: '附近' }
];

	let notifications = $state<NotificationItem[]>([]);
	let filter = $state('all');
	let loading = $state(true);
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
				const [notifRes, unreadRes] = await Promise.all([
					listNotifications().catch(() => ({ code: -1, data: { items: [] } })),
					getUnreadCount().catch(() => ({ code: -1, data: { total: 0 } }))
				]);
				if (cancelled) return;

				const notifData = notifRes.code === 0 ? notifRes.data : null;

				if (notifData) {
					const items: NotificationItem[] = (notifData.items || []).map((n, i) => {
						const actor = n.payload?.actor ?? { user_id: '', nickname: '' };
						const groupName = n.payload?.group_name ?? '群组';
						let text = '';
						if (n.type === 'moment_commented') text = n.payload?.comment_preview ?? `${actor.nickname ?? ''} 评论了你的动态`;
						else if (n.type === 'comment_reply') text = `${actor.nickname ?? ''} 回复了你的评论:${n.payload?.reply_content_preview ?? ''}`;
						else if (n.type === 'new_follower') text = `${actor.nickname ?? ''} 关注了你`;
						else if (n.type === 'mutual_follow') text = `${actor.nickname ?? ''} 和你互相关注了`;
						else if (n.type === 'group_invite_received') text = `${actor.nickname ?? ''} 邀请你加入 ${groupName}`;
						else if (n.type === 'group_invite_revoked') text = `${n.payload?.revoked_by?.nickname ?? '对方'} 撤销了「${groupName}」的入群邀请`;
						else if (n.type === 'group_invite_accepted') text = `${n.payload?.user?.nickname ?? ''} 接受了你的「${groupName}」入群邀请`;
						else if (n.type === 'group_invite_rejected') text = `${n.payload?.rejected_by?.nickname ?? '对方'} 拒绝了你的「${groupName}」入群邀请${n.payload?.reason ? `:${n.payload.reason}` : ''}`;
						else if (n.type === 'group_application_cancelled') text = `${n.payload?.applicant?.nickname ?? '申请人'} 取消了「${groupName}」的入群申请`;
						else if (n.type === 'group_application_accepted') text = `你加入「${groupName}」的申请已通过`;
						else if (n.type === 'group_application_rejected') text = `${n.payload?.rejected_by?.nickname ?? '管理员'} 拒绝了你的「${groupName}」入群申请`;
						else if (n.type === 'reaction_added') text = `${actor.nickname ?? ''} 做出了反应 ${n.payload?.emoji ?? ''}`;
						else if (n.type === 'report_resolved') text = `你举报的内容已处理${n.payload?.note ? `:${n.payload.note}` : ''}`;
						else if (n.type === 'report_dismissed') text = `你的举报未通过审核${n.payload?.note ? `:${n.payload.note}` : ''}`;
						else if (n.type === 'nearby_new_moment') text = `${actor.nickname ?? ''} 在附近发布了新瞬间`;
						else if (n.type === 'system_announcement') text = '系统公告';
						else text = n.payload?.message ?? '';
						return {
							id: n.notification_id ?? String(i + 1),
							// 归一化为语义键，使图标/计数/分区过滤都能命中 TYPE_CONFIG。
							type: normalizeNotifType(n.type ?? 'reaction_added'),
							read: n.is_read ?? false,
							time: formatDateTime(n.created_at),
							user: { name: actor.nickname ?? '', avatar: actor.avatar_url ?? '' },
							text,
							momentImage: n.payload?.moment_summary?.cover_url ?? undefined
						};
					});
					notifications = items;
				}
			} catch {
				if (!cancelled) notifications = [];
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchNotifs();
		return () => {
			cancelled = true;
		};
	});

	const unread = $derived(notifications.filter((n) => !n.read).length);

	const markAllRead = () => {
		// spec: scope='all' clears all unread. Update local state optimistically.
		notifications = notifications.map((n) => ({ ...n, read: true }));
		void markRead({ scope: 'all' }).catch(() => {});
	};
	const markSingleRead = (id: string) => {
		notifications = notifications.map((n) => (n.id === id ? { ...n, read: true } : n));
		// spec: scope='selected' with notification_ids (one of notification_ids|type).
		void markRead({ scope: 'selected', notification_ids: [id] }).catch(() => {});
	};
	const remove = (id: string) => {
		notifications = notifications.filter((n) => n.id !== id);
	};

	// 归一化后 n.type 已是语义键（reaction/follow/comment/nearby/event），直接与 tab id 比较。
	const filtered = $derived(notifications.filter((n) => filter === 'all' || n.type === filter));
	const unreadGroup = $derived(filtered.filter((n) => !n.read));
	const readGroup = $derived(filtered.filter((n) => n.read));
	const typeStats = $derived(
		(Object.keys(TYPE_CONFIG) as string[]).map((t) => ({ type: t, count: notifications.filter((n) => n.type === t).length }))
	);
</script>

<SplitPageLayout contentClassName="pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			title="通知"
			sticky
			titleSize={20}
			subtitleSize={12}
			contentClassName="pt-4 pb-0"
			onBack={() => history.back()}
		>
			{#snippet trailing()}
				<!-- 未读计数统一在标题右侧（对齐首页/附近动态样板），与「全部已读」同排 -->
				<div class="flex items-center gap-2">
					{#if unread > 0}
						<span class="text-text-secondary text-xs">{unread} 条未读</span>
						<button
							type="button"
							class="hover-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none bg-surface-primary text-accent-default text-sm"
							style="border:1px solid var(--border-default);box-shadow:var(--shadow-neu-xs);"
							onclick={markAllRead}
						>
							<CheckCheck class="size-3.25"  strokeWidth={2} />
							全部已读
						</button>
					{/if}
				</div>
			{/snippet}
			{#snippet below()}
				<!-- Filter tabs（对齐 Figma Make：与标题同处 sticky header 内，滚动时吸顶） -->
				<div class="mt-3 flex overflow-x-auto no-scrollbar gap-2 pb-3">
					{#each FILTER_TABS as t}
						<button
							type="button"
							aria-pressed={filter === t.id}
							data-active={filter === t.id ? 'true' : undefined}
							onclick={() => (filter = t.id)}
							class="tap-92 shrink-0 px-3.5 py-1.5 rounded-full outline-none transition-all focus-visible:ring-2 focus-visible:ring-accent-default"
							style={filter === t.id
								? 'background:var(--accent-default);color:white;font-size:12px;box-shadow:0 3px 10px var(--glow-accent);border:none;'
								: 'background:var(--surface-primary);color:var(--text-secondary);font-size:12px;border:1px solid var(--border-default);box-shadow:var(--shadow-neu-xs);'}
						>
							{t.label}
						</button>
					{/each}
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	<!-- List：每个分组一个连接列表表面（List bordered 契约；heading 在列表外） -->
	<div class="flex flex-col">
				{#if unreadGroup.length > 0}
					<!-- 新消息分组标题（emphasis：accent 点 + 计数 pill，对齐 Figma Make） -->
					<div class="mb-3 flex h-5 items-center gap-2">
						<span aria-hidden="true" class="h-1.5 w-1.5 rounded-full" style="background:var(--accent-default);"></span>
						<p class="uppercase text-xxs"  style="font-weight:600;letter-spacing:0.14em;color:var(--text-primary)">新消息</p>
						<span class="rounded-full px-1.5 py-0.5 text-xxs"  style="line-height:1;color:var(--accent-default);background:var(--accent-soft)">{unreadGroup.length}</span>
					</div>
					<ListContainer bordered>
						{#each unreadGroup as n (n.id)}
							<NotificationRow notif={n} onRead={markSingleRead} onRemove={remove} />
						{/each}
					</ListContainer>
				{/if}

				{#if readGroup.length > 0}
					<!-- 较早分组标题（非 emphasis：仅 label + flex 分隔线，对齐 Figma Make） -->
					<div class="mb-3 flex h-5 items-center gap-2" style="margin-top:{unreadGroup.length > 0 ? '20px' : '0'};">
						<p class="uppercase text-xxs"  style="font-weight:500;letter-spacing:0.14em;color:var(--text-secondary)">较早</p>
						<span aria-hidden="true" class="h-px flex-1" style="background:var(--border-default);"></span>
					</div>
					<ListContainer bordered>
						{#each readGroup as n (n.id)}
							<NotificationRow notif={n} onRead={markSingleRead} onRemove={remove} />
						{/each}
					</ListContainer>
				{/if}

				{#if filtered.length === 0}
					<EmptyState title="暂无通知">
						{#snippet icon()}
							<Bell class="size-6"  style="color:var(--text-secondary);opacity:0.5" strokeWidth={1.25} />
						{/snippet}
					</EmptyState>
				{/if}
	</div>

	{#snippet rail()}

		<div>
			<RailSection title="今日概览">
				<div class="flex items-baseline gap-1.5 mb-3">
					<span class="text-text-primary text-3xl font-semibold leading-none">{unread}</span>
					<span class="text-text-secondary text-xs">条未读 · 共 {notifications.length} 条</span>
				</div>
				<div class="flex flex-col gap-2">
					{#each typeStats as { type, count }}
						{@const cfg = TYPE_CONFIG[type] ?? TYPE_CONFIG.reaction}
						<button
							type="button"
							class="hover-btn flex items-center gap-2.5 rounded-tight px-2 py-1.5 outline-none text-left"
							style="background:{filter === type ? `color-mix(in srgb, ${cfg.color} 10%, transparent)` : 'transparent'};border:1px solid {filter === type ? `color-mix(in srgb, ${cfg.color} 22%, transparent)` : 'transparent'};transition:background 0.15s;"
							onclick={() => (filter = type)}
						>
							<div class="flex items-center justify-center rounded-full shrink-0" style="width:24px;height:24px;background:{`color-mix(in srgb, ${cfg.color} 16%, transparent)`};">
								<cfg.icon style="width:11px;height:11px;color:{cfg.color};" strokeWidth={1.75} />
							</div>
							<span class="text-text-primary text-sm flex-1">{cfg.label}</span>
							<span class="text-text-secondary text-xs">{count}</span>
						</button>
					{/each}
				</div>
			</RailSection>

			<RailSection title="新关注你的人" action="全部" onAction={() => void goto(`${ME_SECTION_PATHS.relationships}?tab=fans`)}>
				<div class="flex flex-col gap-3">
					{#each notifications.filter((n) => n.type === 'follow').slice(0, 3) as n (n.id)}
						<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.time}>
							{#snippet trailing()}
								<button type="button" class="hover-btn px-2.5 py-1 rounded-full outline-none shrink-0 bg-accent-soft text-accent-default text-xs border border-accent-border" >
									回关
								</button>
							{/snippet}
						</RailUserRow>
					{/each}
				</div>
			</RailSection>

			<RailSection title="此刻附近" action="查看" onAction={() => void goto(ROUTES.nearbyPeople)}>
				<div class="flex flex-col gap-3">
					{#each notifications.filter((n) => n.type === 'nearby').slice(0, 3) as n (n.id)}
						<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.text} />
					{/each}
				</div>
			</RailSection>
		</div>
	{/snippet}
</SplitPageLayout>
