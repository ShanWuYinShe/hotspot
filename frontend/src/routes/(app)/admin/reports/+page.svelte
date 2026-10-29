<script lang="ts">
	import { adminListReports, adminJudgeReport } from '@/api';
	import type { AdminReportItem, ReportStatus, ReportTargetType } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import type { Component } from 'svelte';
	import { Camera, Flag, MessageCircle, MessageSquare, UserRound, UsersRound } from '@lucide/svelte';
	import { ROUTES } from '@/lib/routes';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import Input from '@/components/forms/Input.svelte';
	import Select from '@/components/forms/Select.svelte';
	import Textarea from '@/components/forms/Textarea.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import { META_MONO } from '../shared/table';
	import { REPORT_REASON_LABELS, REPORT_STATUS_LABELS, REPORT_TARGET_LABELS } from '../shared/labels';
	import { formatDateTime } from '@/lib/utils';

	// 举报审核：状态/类型过滤 + 裁决（resolve 确认 / dismiss 驳回）。
	// resolve 携带封禁天数时需要额外具备 admin.users.write（权限蕴含，见 spec）。
	// R13：表格改「举报卡」——类型/对象/状态/时间 → 理由（中文）→ 说明/证据 → 举报人 + 裁决动作；
	// 卡内一条主链路（谁报了谁、为什么、怎么处理），密度靠留白与发丝线，不再用 7 列表格硬塞。

	const canRead = $derived(hasAdminPermission('admin.reports.read'));
	const canWrite = $derived(hasAdminPermission('admin.reports.write'));
	const canBan = $derived(hasAdminPermission('admin.users.write'));

	const STATUS_TABS = [
		{ id: '', label: '全部' },
		{ id: 'pending', label: '待处理' },
		{ id: 'resolved', label: '已确认' },
		{ id: 'dismissed', label: '已驳回' }
	];
	const TARGET_TYPES = [
		{ value: '', label: '全部目标' },
		...Object.entries(REPORT_TARGET_LABELS).map(([value, label]) => ({ value, label }))
	];
	let status = $state('');
	let targetType = $state('');

	const hasFilter = $derived(Boolean(status || targetType));
	const clearFilters = () => {
		status = '';
		targetType = '';
	};

	const list = createAdminList<AdminReportItem>(
		(cursor, limit) =>
			adminListReports({ status: (status || null) as ReportStatus | null, target_type: (targetType || null) as ReportTargetType | null, pagination: { limit, cursor } }),
		() => `${status}|${targetType}`
	);

	// 状态徽章 tone：待处理=accent（唯一强调）、已确认=online、已驳回=中性；
	// 三档均走 Badge 软底语义档（§20.13），不使用实底抢视觉。
	const statusTone: Record<ReportStatus, 'accent' | 'online' | 'neutral'> = {
		pending: 'accent',
		resolved: 'online',
		dismissed: 'neutral'
	};

	let judgeTarget = $state<AdminReportItem | null>(null);
	let action = $state<'resolve' | 'dismiss'>('resolve');
	let note = $state('');
	let banDays = $state<number | null>(null);
	let pending = $state(false);

	const openJudge = (r: AdminReportItem) => {
		judgeTarget = r;
		action = 'resolve';
		note = '';
		banDays = null;
	};

	const submitJudge = async () => {
		if (!judgeTarget || pending) return;
		// 天数填了就必须是正整数：负数静默转永久封禁会放大操作后果（后端对负数本就拒绝）。
		if (action === 'resolve' && banDays !== null && (!Number.isFinite(banDays) || banDays < 1)) {
			showToast('封禁天数需为正整数，留空表示不封禁', 'danger');
			return;
		}
		pending = true;
		try {
			const res = await adminJudgeReport({
				report_id: judgeTarget.report_id,
				action,
				note: note.trim() || null,
				ban_duration_days: action === 'resolve' && banDays !== null && banDays > 0 ? Math.floor(banDays) : null
			});
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast(action === 'resolve' ? '已确认处理' : '已驳回', 'success');
			judgeTarget = null;
			list.reload();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			pending = false;
		}
	};

	const snapshotText = (r: AdminReportItem): string =>
		r.description ?? r.message_snapshot?.snapshot_content ?? '';

	// 被举报对象：类型图标 + 中文类型 + 可跳转 ID（用户/瞬间/群组有详情页；
	// 评论/消息暂无独立详情页，保留等宽 ID + title 说明，不做假链接）
	const TARGET_META: Record<ReportTargetType, { icon: Component; href?: (id: string) => string }> = {
		user: { icon: UserRound, href: (id) => ROUTES.user(id) },
		moment: { icon: Camera, href: (id) => ROUTES.moment(id) },
		group: { icon: UsersRound, href: (id) => ROUTES.group(id) },
		comment: { icon: MessageCircle },
		message: { icon: MessageSquare }
	};
</script>

<svelte:head>
	<title>举报裁决 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.reports.read 权限" iconTone="accent">
		{#snippet icon()}<Flag class="size-7" strokeWidth={1.5} />{/snippet}
	</EmptyState>
{:else}
	<div class="flex flex-col gap-4">
		<!-- 筛选行：状态 TabBar（左）+ 目标类型下拉（右）；移动端堆叠 -->
		<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<TabBar items={STATUS_TABS} value={status} onChange={(id) => (status = id)} />
			<Select value={targetType} onChange={(v) => (targetType = v)} ariaLabel="目标类型" density="compact" class="w-full sm:w-auto">
				{#each TARGET_TYPES as t (t.value)}
					<option value={t.value}>{t.label}</option>
				{/each}
			</Select>
		</div>

		{#if list.loading}
			<InlineLoading label="加载举报列表…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的举报" description="当前筛选条件下暂无举报，切换状态或目标类型再试。" iconTone="neutral">
				{#snippet icon()}<Flag class="size-7" strokeWidth={1.5} />{/snippet}
				{#snippet action()}
					{#if hasFilter}
						<Button type="button" variant="secondary" size="sm" onclick={clearFilters}>清除筛选</Button>
					{/if}
				{/snippet}
			</EmptyState>
		{:else}
			<!-- 桌面两列卡片（内容链路较长，2 列兼顾密度与可读；≥md=960 起） -->
			<ul class="grid grid-cols-1 gap-3 md:grid-cols-2">
				{#each list.items as r (r.report_id)}
					{@const target = TARGET_META[r.target_type]}
					{@const TargetIcon = target.icon}
					<li>
						<article class="rounded-lg bg-surface-island p-4 shadow-xs">
							<!-- 元信息行：对象类型图标 + 中文类型 + 可跳转 ID + 状态 + 时间 -->
							<div class="flex flex-wrap items-center gap-x-2 gap-y-1.5">
								<span class="flex items-center gap-1.5">
									<TargetIcon class="size-3.5 shrink-0 text-text-secondary" strokeWidth={1.75} aria-hidden="true" />
									<Badge>{REPORT_TARGET_LABELS[r.target_type]}</Badge>
								</span>
								{#if target.href}
									<a
										href={target.href(r.target_id)}
										class="link-btn hit-slop-44 py-0.5 text-xs font-mono focus-ring"
										title="查看被举报对象"
									>{r.target_id}</a>
								{:else}
									<span class={`${META_MONO}`} title="该类型暂无独立详情页">{r.target_id}</span>
								{/if}
								<Badge tone={statusTone[r.status]}>{REPORT_STATUS_LABELS[r.status]}</Badge>
								<time datetime={r.created_at} class="ml-auto text-xs tabular-nums text-text-secondary">
									{formatDateTime(r.created_at)}
								</time>
							</div>
							<!-- 主链路：举报理由 → 补充说明 / 消息快照（证据） -->
							<p class="mt-2.5 text-md font-semibold text-text-primary">{REPORT_REASON_LABELS[r.reason]}</p>
							{#if snapshotText(r)}
								<p class="mt-1 text-xs leading-card text-text-secondary">{snapshotText(r)}</p>
							{:else}
								<p class="mt-1 text-xs leading-card text-text-secondary">举报人未补充说明。</p>
							{/if}
							{#if r.message_snapshot?.snapshot_media}
								<p class="mt-1.5 text-xxs text-text-secondary">
									证据附件：{r.message_snapshot.snapshot_media.mime_type ?? '媒体消息'}
								</p>
							{/if}
							<!-- 底部动作行：举报人/处理人（安静）→ 裁决（主操作） -->
							<div class="mt-3 flex items-center justify-between gap-3 border-t border-border-subtle pt-3">
								<p class="min-w-0 truncate text-xs text-text-secondary">
									举报人 {r.reporter?.nickname ?? '匿名'}
									{#if r.handled_by}
										<span aria-hidden="true">·</span>
										处理人 {r.handled_by.nickname}
									{/if}
									{#if r.handle_note}
										<!-- 裁决备注是处理结论，悬停可看全文 -->
										<span class="cursor-help" title={r.handle_note}>· 有备注</span>
									{/if}
								</p>
								{#if canWrite && r.status === 'pending'}
									<Button type="button" variant="primary" size="sm" class="hit-slop-44" onclick={() => openJudge(r)}>裁决</Button>
								{/if}
							</div>
						</article>
					</li>
				{/each}
			</ul>

			{#if list.hasMore}
				<LoadMoreButton loading={list.loadingMore} onLoad={list.loadMore} />
			{:else}
				<ListEnd />
			{/if}
		{/if}
	</div>

	<Dialog.Root open={judgeTarget !== null} onOpenChange={(next) => (next ? undefined : (judgeTarget = null))}>
		<Dialog.Content class="max-w-90 p-0">
			<div class="px-5 pt-6 pb-4">
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">裁决举报</Dialog.Title>
				<p class="mt-1 text-xs text-text-secondary">
					{judgeTarget ? `${REPORT_TARGET_LABELS[judgeTarget.target_type]} · ${judgeTarget.target_id} · ${REPORT_REASON_LABELS[judgeTarget.reason]} · ${formatDateTime(judgeTarget.created_at)}` : ''}
				</p>
				{#if judgeTarget?.target_type === 'message' && judgeTarget?.message_snapshot}
					{@const snap = judgeTarget.message_snapshot}
					<div class="mt-3 rounded-md bg-surface-subtle px-3 py-2.5">
						<p class="text-xs text-text-secondary">
							被举报消息（{snap.chat_type === 'group' ? '群聊' : '私聊'} · {formatDateTime(snap.sent_at)}）
						</p>
						{#if snap.snapshot_content}
							<p class="mt-1 text-sm break-all text-text-primary">{snap.snapshot_content}</p>
						{:else if snap.snapshot_media}
							<p class="mt-1 text-sm text-text-primary">[媒体消息：{snap.snapshot_media.mime_type ?? '附件'}]</p>
						{:else}
							<p class="mt-1 text-sm text-text-secondary">（内容为空）</p>
						{/if}
					</div>
				{/if}
				<div class="mt-4 flex flex-col gap-3">
					<!-- 裁决方向用统一分段控件（与全站 TabBar 语言一致；选中态带 ✓ 不只靠颜色） -->
					<TabBar
						variant="segmented"
						items={[
							{ id: 'resolve', label: '确认有效并处理' },
							{ id: 'dismiss', label: '驳回举报' }
						]}
						value={action}
						onChange={(id) => (action = id as 'resolve' | 'dismiss')}
					/>
					<Textarea
						label="处理备注（写入 handle_note 与审计日志，可留空）"
						value={note}
						onChange={(v) => (note = v)}
						rows={2}
						maxlength={500}
					/>
					{#if action === 'resolve'}
						<Input
							label={`连带封禁天数${canBan ? '（留空 = 仅标记处理不封禁；0 = 永久）' : '（需要 admin.users.write 权限）'}`}
							type="number"
							min={0}
							value={banDays === null ? '' : String(banDays)}
							onChange={(v) => (banDays = v.trim() === '' ? null : Number(v))}
							disabled={!canBan}
							density="compact"
						/>
					{/if}
				</div>
			</div>
			<div class="flex justify-end gap-2 px-5 pb-5">
				<Button type="button" variant="secondary" size="md" onclick={() => (judgeTarget = null)}>取消</Button>
				<Button type="button" variant={action === 'resolve' ? 'danger' : 'primary'} size="md" loading={pending} onclick={() => void submitJudge()}>
					{action === 'resolve' ? '确认处理' : '确认驳回'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>
{/if}