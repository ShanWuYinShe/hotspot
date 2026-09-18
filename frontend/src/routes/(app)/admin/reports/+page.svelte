<script lang="ts">
	import { adminListReports, adminJudgeReport } from '@/api';
	import type { AdminReportItem, ReportStatus, ReportTargetType } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import { TABLE_WRAP, TABLE, TH, TR, TD, TD_ACTIONS, ROW_BTN_PRIMARY } from '../shared/table';
	import { formatDateTime } from '@/lib/utils';

	// 举报审核：状态/类型过滤 + 裁决（resolve 确认 / dismiss 驳回）。
	// resolve 携带封禁天数时需要额外具备 admin.users.write（权限蕴含，见 spec）。

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
		{ value: 'user', label: '用户' },
		{ value: 'moment', label: '瞬间' },
		{ value: 'comment', label: '评论' },
		{ value: 'message', label: '消息' },
		{ value: 'group', label: '群组' }
	];
	let status = $state('');
	let targetType = $state('');

	const list = createAdminList<AdminReportItem>(
		(cursor, limit) =>
			adminListReports({ status: (status || null) as ReportStatus | null, target_type: (targetType || null) as ReportTargetType | null, pagination: { limit, cursor } }),
		() => `${status}|${targetType}`
	);

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
</script>

<svelte:head>
	<title>举报裁决 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.reports.read 权限" />
{:else}
	<div class="flex flex-col gap-4">
		<div class="flex flex-col gap-3 lg:flex-row lg:items-center">
			<TabBar items={STATUS_TABS} value={status} onChange={(id) => (status = id)} />
			<select bind:value={targetType} class="input-surface shrink-0 rounded-md px-3 py-2 text-xs text-text-primary" aria-label="目标类型">
				{#each TARGET_TYPES as t (t.value)}
					<option value={t.value}>{t.label}</option>
				{/each}
			</select>
		</div>

		{#if list.loading}
			<InlineLoading label="加载举报列表…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的举报" />
		{:else}
			<div class={TABLE_WRAP}>
				<table class={TABLE}>
					<thead>
						<tr>
							<th scope="col" class={TH}>目标</th>
							<th scope="col" class={TH}>原因</th>
							<th scope="col" class={TH}>描述</th>
							<th scope="col" class={TH}>举报人</th>
							<th scope="col" class={TH}>状态</th>
							<th scope="col" class={TH}>时间</th>
							<th scope="col" class={TH}></th>
						</tr>
					</thead>
					<tbody>
						{#each list.items as r (r.report_id)}
							<tr class={TR}>
								<td class={TD}>
									<span class="badge bg-surface-subtle text-text-secondary">{r.target_type}</span>
									<span class="ml-1.5 text-xs text-text-secondary font-mono">{r.target_id}</span>
								</td>
								<td class={TD}>{r.reason}</td>
								<td class={`${TD} max-w-56 truncate`} title={r.description ?? r.message_snapshot?.snapshot_content ?? ''}>{r.description ?? r.message_snapshot?.snapshot_content ?? '—'}</td>
								<td class={TD}>{r.reporter?.nickname ?? '—'}</td>
								<td class={TD}>
									<span class="badge {r.status === 'pending' ? 'bg-accent-soft text-accent-default' : r.status === 'resolved' ? 'bg-surface-subtle text-text-secondary' : 'bg-surface-subtle text-text-secondary'}">
										{r.status === 'pending' ? '待处理' : r.status === 'resolved' ? '已确认' : '已驳回'}
									</span>
								</td>
								<td class={TD}>{formatDateTime(r.created_at)}</td>
								<td class={TD_ACTIONS}>
									{#if canWrite && r.status === 'pending'}
										<button type="button" class={ROW_BTN_PRIMARY} onclick={() => openJudge(r)}>裁决</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

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
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">裁决举报 {judgeTarget?.report_id}</Dialog.Title>
				<Dialog.Description class="mt-1 text-sm text-text-secondary leading-relaxed">
					目标 {judgeTarget?.target_type}（<span class="font-mono">{judgeTarget?.target_id}</span>），原因：{judgeTarget?.reason}
				</Dialog.Description>
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
					<div class="flex gap-2">
						<button
							type="button"
							onclick={() => (action = 'resolve')}
							class="flex-1 rounded-md border px-3 py-2 text-sm transition-colors {action === 'resolve' ? 'border-accent-default bg-accent-soft text-accent-default font-medium' : 'border-border-default text-text-secondary hover:bg-interactive-hover'}"
						>确认有效并处理</button>
						<button
							type="button"
							onclick={() => (action = 'dismiss')}
							class="flex-1 rounded-md border px-3 py-2 text-sm transition-colors {action === 'dismiss' ? 'border-accent-default bg-accent-soft text-accent-default font-medium' : 'border-border-default text-text-secondary hover:bg-interactive-hover'}"
						>驳回举报</button>
					</div>
					<label class="flex flex-col gap-1.5">
						<span class="text-xs text-text-secondary">处理备注（写入 handle_note 与审计日志，可留空）</span>
						<textarea bind:value={note} rows={2} maxlength={500} class="input-surface resize-none rounded-md px-3 py-2 text-sm text-text-primary"></textarea>
					</label>
					{#if action === 'resolve'}
						<label class="flex flex-col gap-1.5">
							<span class="text-xs text-text-secondary">
								连带封禁天数{canBan ? '（留空 = 仅标记处理不封禁；0 = 永久）' : '（需要 admin.users.write 权限）'}
							</span>
							<input type="number" min={0} bind:value={banDays} disabled={!canBan} class="input-surface rounded-md px-3 py-2 text-sm text-text-primary disabled:opacity-faint" />
						</label>
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
