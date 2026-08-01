<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, CheckCheck } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import DesktopDetailSplit from '@/components/layout/DesktopDetailSplit.svelte';
	import RailSection from '@/components/layout/RailSection.svelte';
	import RailUserRow from '@/components/layout/RailUserRow.svelte';
	import PageHeader from '@/components/shared/PageHeader.svelte';
	import TagPill from '@/components/shared/TagPill.svelte';
	import EmptyState from '@/components/shared/EmptyState.svelte';
	import NotifRow from './NotifRow.svelte';
	import { TYPE_CONFIG, type Notif } from './notif-data';
	import { listNotifications, getUnreadCount, markRead } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 通知列表：类型过滤、单条/全部已读、桌面右侧概览 Rail。
	// 迁移自旧版 NotificationsPage。

	const FILTER_TABS = [
		{ id: 'all', label: '全部' },
		// 点赞由 reaction_added 表达(契约已删 moment_liked),tab 语义对齐 reaction。
		{ id: 'reaction', label: '点赞' },
		{ id: 'follow', label: '关注' },
		{ id: 'comment', label: '评论' },
		{ id: 'nearby', label: '附近' }
	];

	let notifs = $state<Notif[]>([]);
	let filter = $state('all');
	let loading = $state(true);

	const isDesktop = $derived(platformStore.platform === 'desktop');
	const shellStyle = $derived(`max-width:${isDesktop ? 'var(--content-reading-max)' : 640}px;margin-left:auto;margin-right:auto;width:100%;`);

	$effect(() => {
		let cancelled = false;
		async function fetchNotifs() {
			try {
				loading = true;
				const [notifRes, unreadRes] = await Promise.all([
					listNotifications().catch(() => ({ code: -1, data: { items: [] } })),
					getUnreadCount().catch(() => ({ code: -1, data: { total: 0 } }))
				]);
				if (cancelled) return;

				const notifData = (notifRes.code === 0 ? notifRes.data : null) as any;

				if (notifData) {
					const items: Notif[] = (notifData.items || []).map((n: any, i: number) => {
						const actor = n.payload?.actor ?? n.actor ?? {};
						const groupName = n.payload?.group_name ?? '群组';
						let text = '';
						if (n.type === 'moment_commented') text = n.payload?.comment_preview ?? `${actor.nickname ?? ''} 评论了你的动态`;
						else if (n.type === 'comment_reply') text = `${actor.nickname ?? ''} 回复了你的评论:${n.payload?.reply_content_preview ?? ''}`;
						else if (n.type === 'new_follower') text = `${actor.nickname ?? ''} 关注了你`;
						else if (n.type === 'mutual_follow') text = `${actor.nickname ?? ''} 和你互相关注了`;
						else if (n.type === 'group_invite_received') text = `${actor.nickname ?? ''} 邀请你加入 ${groupName}`;
						else if (n.type === 'group_invite_revoked') text = `${n.payload?.revoked_by?.nickname ?? '对方'} 撤销了「${groupName}」的入群邀请`;
						else if (n.type === 'group_invite_accepted') text = `${n.payload?.user?.nickname ?? ''} 接受了你的「${groupName}」入群邀请`;
						else if (n.type === 'group_application_cancelled') text = `${n.payload?.applicant?.nickname ?? '申请人'} 取消了「${groupName}」的入群申请`;
						else if (n.type === 'group_application_accepted') text = `你加入「${groupName}」的申请已通过`;
						else if (n.type === 'group_application_rejected') text = `${n.payload?.rejected_by?.nickname ?? '管理员'} 拒绝了你的「${groupName}」入群申请`;
						else if (n.type === 'reaction_added') text = `${actor.nickname ?? ''} 做出了反应 ${n.payload?.emoji ?? ''}`;
						else if (n.type === 'report_resolved') text = `你举报的内容已处理${n.payload?.note ? `:${n.payload.note}` : ''}`;
						else if (n.type === 'report_dismissed') text = `你的举报未通过审核${n.payload?.note ? `:${n.payload.note}` : ''}`;
						else if (n.type === 'nearby_new_moment') text = `${actor.nickname ?? ''} 在附近发布了新瞬间`;
						else if (n.type === 'system_announcement') text = '系统公告';
						else text = n.payload?.title ?? n.message ?? '';
						return {
							id: n.notification_id ?? String(i + 1),
							type: n.type ?? 'reaction',
							read: n.is_read ?? false,
							time: n.created_at ?? '',
							user: { name: actor.nickname ?? '', avatar: actor.avatar_url ?? '' },
							text,
							postImage: n.payload?.moment_summary?.cover_url ?? n.payload?.cover_url ?? undefined
						};
					});
					notifs = items;
				}
			} catch {
				if (!cancelled) notifs = [];
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchNotifs();
		return () => {
			cancelled = true;
		};
	});

	const unread = $derived(notifs.filter((n) => !n.read).length);

	const markAllRead = () => {
		// spec: scope='all' clears all unread. Update local state optimistically.
		notifs = notifs.map((n) => ({ ...n, read: true }));
		void markRead({ scope: 'all' }).catch(() => {});
	};
	const markSingleRead = (id: string) => {
		notifs = notifs.map((n) => (n.id === id ? { ...n, read: true } : n));
		// spec: scope='selected' with notification_ids (one of notification_ids|type).
		void markRead({ scope: 'selected', notification_ids: [id] }).catch(() => {});
	};
	const remove = (id: string) => {
		notifs = notifs.filter((n) => n.id !== id);
	};

	const filtered = $derived(notifs.filter((n) => filter === 'all' || n.type === filter));
	const unreadGroup = $derived(filtered.filter((n) => !n.read));
	const readGroup = $derived(filtered.filter((n) => n.read));
	const typeStats = $derived(
		(Object.keys(TYPE_CONFIG) as string[]).map((t) => ({ type: t, count: notifs.filter((n) => n.type === t).length }))
	);
</script>

{#snippet main()}

		<div class="h-full overflow-y-auto no-scrollbar">
			<PageHeader title="通知" sticky onBack={() => history.back()} subtitle={unread > 0 ? `${unread} 条未读` : undefined}>
				{#snippet actions()}
					{#if unread > 0}
						<button
							type="button"
							class="neu-hover-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none bg-[var(--neu-bg)] text-[var(--neu-accent-primary)] text-[12px]"
							style="border:1px solid var(--neu-border);box-shadow:3px 3px 8px var(--neu-shadow-dark), -3px -3px 8px var(--neu-shadow-light);"
							onclick={markAllRead}
						>
							<CheckCheck style="width:13px;height:13px;" strokeWidth={2} />
							全部已读
						</button>
					{/if}
				{/snippet}
			</PageHeader>

			<!-- Filter tabs -->
			<div class="flex overflow-x-auto no-scrollbar pb-3 gap-2 px-5" style={shellStyle}>
				{#each FILTER_TABS as t}
					<TagPill active={filter === t.id} onClick={() => (filter = t.id)} size="sm">{t.label}</TagPill>
				{/each}
			</div>

			<!-- List -->
			<div class="flex flex-col px-5 pt-3 pb-6 gap-0" style={shellStyle}>
				{#if unreadGroup.length > 0}
					<p class="uppercase mb-2 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em]">新消息</p>
					{#each unreadGroup as n (n.id)}
						<NotifRow notif={n} onRead={markSingleRead} onRemove={remove} />
					{/each}
					{#if readGroup.length > 0}
						<p class="uppercase mt-4 mb-2 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em]">较早</p>
					{/if}
				{/if}

				{#each readGroup as n (n.id)}
					<NotifRow notif={n} onRead={markSingleRead} onRemove={remove} />
				{/each}

				{#if filtered.length === 0}
					<EmptyState title="暂无通知">
						{#snippet icon()}
							<Bell style="width:24px;height:24px;color:var(--neu-text-muted);opacity:0.5;" strokeWidth={1.25} />
						{/snippet}
					</EmptyState>
				{/if}
			</div>
		</div>
{/snippet}

{#snippet rail()}

		<div>
			<RailSection title="今日概览">
				<div class="flex items-baseline gap-1.5 mb-3">
					<span class="text-[var(--neu-text)] text-[30px] font-semibold leading-none">{unread}</span>
					<span class="text-[var(--neu-text-muted)] text-[11px]">条未读 · 共 {notifs.length} 条</span>
				</div>
				<div class="flex flex-col gap-2">
					{#each typeStats as { type, count }}
						{@const cfg = TYPE_CONFIG[type] ?? TYPE_CONFIG.reaction}
						<button
							type="button"
							class="neu-hover-btn flex items-center gap-2.5 rounded-[10px] px-2 py-1.5 outline-none text-left"
							style="background:{filter === type ? `color-mix(in srgb, ${cfg.color} 10%, transparent)` : 'transparent'};border:1px solid {filter === type ? `color-mix(in srgb, ${cfg.color} 22%, transparent)` : 'transparent'};transition:background 0.15s;"
							onclick={() => (filter = type)}
						>
							<div class="flex items-center justify-center rounded-full flex-shrink-0" style="width:24px;height:24px;background:{`color-mix(in srgb, ${cfg.color} 16%, transparent)`};">
								<cfg.icon style="width:11px;height:11px;color:{cfg.color};" strokeWidth={1.75} />
							</div>
							<span class="text-[var(--neu-text)] text-[12px] flex-1">{cfg.label}</span>
							<span class="text-[var(--neu-text-muted)] text-[11px]">{count}</span>
						</button>
					{/each}
				</div>
			</RailSection>

			<RailSection title="新关注你的人" action="全部" onAction={() => void goto(`${ROUTES.meSection('relationships')}?tab=followers`)}>
				<div class="flex flex-col gap-3">
					{#each notifs.filter((n) => n.type === 'follow').slice(0, 3) as n (n.id)}
						<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.time}>
							{#snippet trailing()}
								<button type="button" class="neu-hover-btn px-2.5 py-1 rounded-full outline-none flex-shrink-0 bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] text-[11px]" style="border:1px solid var(--neu-accent-soft-border);">
									回关
								</button>
							{/snippet}
						</RailUserRow>
					{/each}
				</div>
			</RailSection>

			<RailSection title="此刻附近" action="查看" onAction={() => void goto(ROUTES.nearbyPeople)}>
				<div class="flex flex-col gap-3">
					{#each notifs.filter((n) => n.type === 'nearby').slice(0, 3) as n (n.id)}
						<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.text} />
					{/each}
				</div>
			</RailSection>
		</div>
{/snippet}

<DesktopDetailSplit {main} {rail} />
