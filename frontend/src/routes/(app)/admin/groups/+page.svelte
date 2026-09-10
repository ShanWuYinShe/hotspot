<script lang="ts">
	import { adminSearchGroups, adminDismissGroup } from '@/api';
	import type { AdminGroupItem } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import { TABLE_WRAP, TABLE, TH, TR, TD, TD_ACTIONS, ROW_BTN_DANGER } from '../shared/table';
	import { formatDateTime } from '@/lib/utils';

	// 群组管理：按名称搜索 + 解散（reason 仅入审计日志，不推送给成员）。

	const canRead = $derived(hasAdminPermission('admin.groups.read'));
	const canWrite = $derived(hasAdminPermission('admin.groups.write'));
	let query = $state('');

	const list = createAdminList<AdminGroupItem>(
		(cursor, limit) => adminSearchGroups({ name: query || null, pagination: { limit, cursor } }),
		() => query
	);

	let dismissTarget = $state<AdminGroupItem | null>(null);
	let dismissReason = $state('');
	let pending = $state(false);

	const submitDismiss = async () => {
		if (!dismissTarget || pending) return;
		pending = true;
		try {
			const res = await adminDismissGroup({ group_id: dismissTarget.group.group_id, reason: dismissReason.trim() || null });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast('群组已解散', 'success');
			dismissTarget = null;
			list.reload();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			pending = false;
		}
	};
</script>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.groups.read 权限" />
{:else}
	<div class="flex flex-col gap-4">
		<div class="w-full max-w-80">
			<SearchInput value={query} onChange={(v) => (query = v)} placeholder="搜索群组名称…" class="px-4 py-2.5 rounded-md" />
		</div>

		{#if list.loading}
			<InlineLoading label="加载群组列表…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的群组" />
		{:else}
			<div class={TABLE_WRAP}>
				<table class={TABLE}>
					<thead>
						<tr>
							<th class={TH}>群组</th>
							<th class={TH}>群主</th>
							<th class={TH}>成员数</th>
							<th class={TH}>创建时间</th>
							<th class={TH}></th>
						</tr>
					</thead>
					<tbody>
						{#each list.items as g (g.group.group_id)}
							<tr class={TR}>
								<td class={TD}>
									<span class="flex items-center gap-2">
										<span class="text-text-primary">{g.group.name}</span>
										<span class="text-xs text-text-secondary font-mono">{g.group.group_id}</span>
									</span>
								</td>
								<td class={TD}>{g.owner?.nickname ?? '—'}</td>
								<td class={TD}>{g.group.members_count}</td>
								<td class={TD}>{formatDateTime(g.created_at)}</td>
								<td class={TD_ACTIONS}>
									{#if canWrite}
										<button
											type="button"
											class={ROW_BTN_DANGER}
											onclick={() => { dismissTarget = g; dismissReason = ''; }}
										>解散</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			{#if list.hasMore}
				<div class="flex justify-center">
					<button type="button" class="hover-btn rounded-full border border-border-default bg-surface-primary px-5 py-2 text-sm text-text-secondary focus-ring transition-colors hover:text-text-primary disabled:opacity-faint" disabled={list.loadingMore} onclick={list.loadMore}>
						{list.loadingMore ? '加载中…' : '加载更多'}
					</button>
				</div>
			{:else}
				<ListEnd />
			{/if}
		{/if}
	</div>

	<Dialog.Root open={dismissTarget !== null} onOpenChange={(next) => (next ? undefined : (dismissTarget = null))}>
		<Dialog.Content class="max-w-90 p-0">
			<div class="px-5 pt-6 pb-4">
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">解散群组「{dismissTarget?.group.name}」？</Dialog.Title>
				<Dialog.Description class="mt-1 text-sm text-text-secondary leading-relaxed">
					解散后所有成员将收到系统解散通知（不暴露平台管理员身份），待处理的申请与邀请自动作废。原因仅记入审计日志。
				</Dialog.Description>
				<label class="mt-4 flex flex-col gap-1.5">
					<span class="text-xs text-text-secondary">解散原因（可留空）</span>
					<textarea bind:value={dismissReason} rows={2} maxlength={500} class="input-surface resize-none rounded-md px-3 py-2 text-sm text-text-primary"></textarea>
				</label>
			</div>
			<div class="flex justify-end gap-2 px-5 pb-5">
				<Button type="button" variant="secondary" size="md" onclick={() => (dismissTarget = null)}>取消</Button>
				<Button type="button" variant="danger" size="md" loading={pending} onclick={() => void submitDismiss()}>确认解散</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>
{/if}
