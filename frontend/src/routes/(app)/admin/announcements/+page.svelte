<script lang="ts">
	import { adminListAnnouncements, adminCreateAnnouncement, adminUpdateAnnouncement, adminDeleteAnnouncement } from '@/api';
	import type { AdminAnnouncementItem, AnnounceSeverity } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { Megaphone, Plus } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import Toggle from '@/components/forms/Toggle.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import {
		TABLE_WRAP, TABLE, TH, TH_SECONDARY, TH_SECONDARY_NUM, TR, TD, TD_SECONDARY, TD_SECONDARY_NUM,
		TD_ACTIONS, ROW_BTN, ROW_BTN_DANGER
	} from '../shared/table';
	import { scrollFade } from '@/lib/scroll-fade';
	import { formatDateTime } from '@/lib/utils';

	// 系统公告管理：列表 + 发布/编辑/删除。create 为契约幂等端点（client 层自动携带 Idempotency-Key）。
	// R13：补「级别」筛选（TabBar，与其他模块的筛选态语言一致）、页面级动作右置、数字列右对齐；
	// 级别徽章保留语义色，危险动作保持 soft danger。

	const canRead = $derived(hasAdminPermission('admin.announcements.read'));
	const canWrite = $derived(hasAdminPermission('admin.announcements.write'));

	const SEVERITIES: { value: AnnounceSeverity; label: string }[] = [
		{ value: 'info', label: '信息' },
		{ value: 'warning', label: '警告' },
		{ value: 'error', label: '严重' }
	];
	const SEVERITY_TABS = [{ id: '', label: '全部' }, ...SEVERITIES.map((s) => ({ id: s.value, label: s.label }))];
	let severityFilter = $state<'' | AnnounceSeverity>('');

	const list = createAdminList<AdminAnnouncementItem>(
		(cursor, limit) =>
			adminListAnnouncements({
				severity: (severityFilter || null) as AnnounceSeverity | null,
				pagination: { limit, cursor }
			}),
		() => severityFilter
	);

	const severityBadge: Record<AnnounceSeverity, string> = {
		info: 'bg-surface-subtle text-text-secondary',
		// 警告归位 warning 族而非 accent（r57：语义色各归其位,亮 accent 收给主操作）
		warning: 'bg-status-warning-bg text-text-status-warning-on-soft',
		error: 'bg-status-danger text-text-on-accent'
	};

	// 编辑弹层（create 与 update 共用；editTarget=null 即发布模式）
	let editOpen = $state(false);
	let editTarget = $state<AdminAnnouncementItem | null>(null);
	let title = $state('');
	let content = $state('');
	let severity = $state<AnnounceSeverity>('info');
	let actionUrl = $state('');
	let sendPush = $state(false);
	let pending = $state(false);

	const openCreate = () => {
		editTarget = null;
		title = '';
		content = '';
		severity = 'info';
		actionUrl = '';
		sendPush = false;
		editOpen = true;
	};

	const openEdit = (a: AdminAnnouncementItem) => {
		editTarget = a;
		title = a.title;
		content = a.content;
		severity = a.severity;
		actionUrl = a.action_url ?? '';
		sendPush = false;
		editOpen = true;
	};

	const submitEdit = async () => {
		if (pending) return;
		if (!title.trim() || !content.trim()) {
			showToast('标题与内容不能为空', 'danger');
			return;
		}
		pending = true;
		try {
			const res = editTarget
				? await adminUpdateAnnouncement({
						announcement_id: editTarget.announcement_id,
						title: title.trim(),
						content: content.trim(),
						severity,
						action_url: actionUrl.trim() // 空串 = 清除链接（spec Patch 约定）
					})
				: await adminCreateAnnouncement({ title: title.trim(), content: content.trim(), severity, action_url: actionUrl.trim() || null, send_push: sendPush });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast(editTarget ? '公告已更新' : '公告已发布', 'success');
			editOpen = false;
			list.reload();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			pending = false;
		}
	};

	let deleteTarget = $state<AdminAnnouncementItem | null>(null);
	const submitDelete = async () => {
		if (!deleteTarget || pending) return;
		pending = true;
		try {
			const res = await adminDeleteAnnouncement({ announcement_id: deleteTarget.announcement_id });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast('公告已删除', 'success');
			deleteTarget = null;
			list.reload();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			pending = false;
		}
	};
</script>

<svelte:head>
	<title>系统公告 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.announcements.read 权限" iconTone="accent">
		{#snippet icon()}<Megaphone class="size-7" strokeWidth={1.5} />{/snippet}
	</EmptyState>
{:else}
	<div class="flex flex-col gap-4">
		<!-- 工具栏：级别筛选（左）+ 页面级动作（右）；移动端上下堆叠 -->
		<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<TabBar items={SEVERITY_TABS} value={severityFilter} onChange={(id) => (severityFilter = id as '' | AnnounceSeverity)} />
			{#if canWrite}
				<Button type="button" variant="primary" size="sm" onclick={openCreate} class="hit-slop-44 self-end sm:self-auto">
					<Plus class="size-3.5" strokeWidth={2} aria-hidden="true" />
					发布公告
				</Button>
			{/if}
		</div>

		{#if list.loading}
			<InlineLoading label="加载公告列表…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="还没有系统公告" description="发布一条公告，让全站用户看到平台最新安排。" iconTone="neutral">
				{#snippet icon()}<Megaphone class="size-7" strokeWidth={1.5} />{/snippet}
				{#snippet action()}
					{#if canWrite}
						<Button type="button" variant="primary" size="sm" class="hit-slop-44" onclick={openCreate}>发布公告</Button>
					{/if}
				{/snippet}
			</EmptyState>
		{:else}
			<div class={TABLE_WRAP} use:scrollFade>
				<table class={TABLE}>
					<thead>
						<tr>
							<th scope="col" class={TH}>标题</th>
							<th scope="col" class={TH}>级别</th>
							<th scope="col" class={TH_SECONDARY}>发布人</th>
							<th scope="col" class={TH_SECONDARY_NUM}>浏览</th>
							<th scope="col" class={TH_SECONDARY}>推送</th>
							<th scope="col" class={TH_SECONDARY}>发布时间</th>
							<th scope="col" class={TH}><span class="sr-only">操作</span></th>
						</tr>
					</thead>
					<tbody>
						{#each list.items as a (a.announcement_id)}
							<tr class={TR}>
								<td class={`${TD} max-w-64 truncate`} title={a.content}>{a.title}</td>
								<td class={TD}><span class="badge {severityBadge[a.severity]}">{SEVERITIES.find((s) => s.value === a.severity)?.label}</span></td>
								<td class={TD_SECONDARY}>{a.created_by?.nickname ?? '—'}</td>
								<td class={TD_SECONDARY_NUM}>{a.view_count}</td>
								<td class={TD_SECONDARY}>{a.send_push ? (a.push_sent_at ? '已推送' : '待推送') : '—'}</td>
								<td class={TD_SECONDARY}>{formatDateTime(a.created_at)}</td>
								<td class={TD_ACTIONS}>
									{#if canWrite}
										<span class="flex justify-end gap-1.5">
											<button type="button" class={ROW_BTN} onclick={() => openEdit(a)}>编辑</button>
											<button type="button" class={ROW_BTN_DANGER} onclick={() => (deleteTarget = a)}>删除</button>
										</span>
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

	<Dialog.Root bind:open={editOpen} onOpenChange={(next) => (next ? undefined : (editOpen = false))}>
		<Dialog.Content class="max-w-100 p-0">
			<div class="px-5 pt-6 pb-4">
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">{editTarget ? '编辑公告' : '发布公告'}</Dialog.Title>
				<div class="mt-4 flex flex-col gap-3">
					<label class="flex flex-col gap-1.5">
						<span class="text-xs text-text-secondary">标题</span>
						<input bind:value={title} maxlength={100} class="input-surface rounded-md px-3 py-2 text-sm text-text-primary" />
					</label>
					<label class="flex flex-col gap-1.5">
						<span class="text-xs text-text-secondary">内容</span>
						<textarea bind:value={content} rows={4} maxlength={2000} class="input-surface resize-none rounded-md px-3 py-2 text-sm leading-copy text-text-primary"></textarea>
					</label>
					<div class="grid grid-cols-2 gap-3">
						<label class="flex flex-col gap-1.5">
							<span class="text-xs text-text-secondary">级别</span>
							<select bind:value={severity} class="input-surface rounded-md px-3 py-2 text-sm text-text-primary">
								{#each SEVERITIES as s (s.value)}
									<option value={s.value}>{s.label}</option>
								{/each}
							</select>
						</label>
						<label class="flex flex-col gap-1.5">
							<span class="text-xs text-text-secondary">跳转链接（可留空）</span>
							<input bind:value={actionUrl} placeholder="https://…" class="input-surface rounded-md px-3 py-2 text-sm text-text-primary" />
						</label>
					</div>
					{#if !editTarget}
						<!-- 布尔开关与系统配置/设置页统一用 Toggle -->
						<div class="flex items-center justify-between gap-3 text-sm text-text-primary">
							<span>同时推送通知</span>
							<Toggle on={sendPush} label="同时推送通知" onToggle={() => (sendPush = !sendPush)} />
						</div>
					{/if}
				</div>
			</div>
			<div class="flex justify-end gap-2 px-5 pb-5">
				<Button type="button" variant="secondary" size="md" onclick={() => (editOpen = false)}>取消</Button>
				<Button type="button" variant="primary" size="md" loading={pending} onclick={() => void submitEdit()}>{editTarget ? '保存' : '发布'}</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>

	<ConfirmDialog
		open={deleteTarget !== null}
		title="删除公告「{deleteTarget?.title}」？"
		description="删除后用户端不再展示，该操作记入审计日志。"
		confirmLabel="确认删除"
		loading={pending}
		onCancel={() => (deleteTarget = null)}
		onConfirm={() => void submitDelete()}
	/>
{/if}
