<script lang="ts">
	import { adminSearchTags, adminSetTagBan } from '@/api';
	import type { AdminTagItem } from '@/api/types';
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
	import { TABLE_WRAP, TABLE, TH, TR, TD, TD_ACTIONS, ROW_BTN, ROW_BTN_DANGER } from '../shared/table';
	import { formatDateTime } from '@/lib/utils';

	// 标签管理：模糊搜索 + 软屏蔽 / 解屏蔽；物理删除（hard_delete）为高危不可逆，需显式勾选。
	// spec：标签业务唯一键为 name；屏蔽后新内容引用该标签将失败，历史数据保留。

	const canRead = $derived(hasAdminPermission('admin.tags.read'));
	const canWrite = $derived(hasAdminPermission('admin.tags.write'));
	let query = $state('');

	const list = createAdminList<AdminTagItem>(
		(cursor, limit) => adminSearchTags({ keyword: query || null, pagination: { limit, cursor } }),
		() => query
	);

	let banTarget = $state<AdminTagItem | null>(null);
	let hardDelete = $state(false);
	let pending = $state(false);

	const openBan = (t: AdminTagItem) => {
		banTarget = t;
		hardDelete = false;
	};
	const unbanCurrent = async () => {
		if (!banTarget) return;
		await unban(banTarget.name);
		banTarget = null;
	};

	const submitBan = async (active: boolean) => {
		if (!banTarget && !active) return; // 解屏蔽无需目标（列表行内直接传 name）
		pending = true;
		try {
			const res = await adminSetTagBan({ name: banTarget?.name ?? '', active, hard_delete: active && hardDelete ? true : null });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast(active ? (hardDelete ? '标签已物理删除' : '标签已屏蔽') : '标签已解除屏蔽', 'success');
			banTarget = null;
			list.reload();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			pending = false;
		}
	};

	const unban = async (name: string) => {
		if (pending) return;
		pending = true;
		try {
			const res = await adminSetTagBan({ name, active: false, hard_delete: null });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast('标签已解除屏蔽', 'success');
			list.reload();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			pending = false;
		}
	};
</script>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.tags.read 权限" />
{:else}
	<div class="flex flex-col gap-4">
		<div class="w-full max-w-80">
			<SearchInput value={query} onChange={(v) => (query = v)} placeholder="搜索标签…" class="px-4 py-2.5 rounded-md" />
		</div>

		{#if list.loading}
			<InlineLoading label="加载标签列表…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的标签" />
		{:else}
			<div class={TABLE_WRAP}>
				<table class={TABLE}>
					<thead>
						<tr>
							<th class={TH}>标签</th>
							<th class={TH}>使用次数</th>
							<th class={TH}>创建时间</th>
							<th class={TH}></th>
						</tr>
					</thead>
					<tbody>
						{#each list.items as t (t.tag_id)}
							<tr class={TR}>
								<td class={TD}>
									<span class="badge bg-accent-soft text-accent-default">#{t.name}</span>
								</td>
								<td class={TD}>{t.usage_count}</td>
								<td class={TD}>{formatDateTime(t.created_at)}</td>
								<td class={TD_ACTIONS}>
									{#if canWrite}
										<button type="button" class={ROW_BTN} onclick={() => openBan(t)}>状态管理</button>
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

	<Dialog.Root open={banTarget !== null} onOpenChange={(next) => (next ? undefined : (banTarget = null))}>
		<Dialog.Content class="max-w-90 p-0">
			<div class="px-5 pt-6 pb-4">
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">屏蔽标签 #{banTarget?.name}？</Dialog.Title>
				<Dialog.Description class="mt-1 text-sm text-text-secondary leading-relaxed">
					软屏蔽：该标签不再可被新建/搜索命中，历史数据保留不变。
				</Dialog.Description>
				<label class="mt-4 flex items-start gap-2.5 rounded-md border border-status-danger/30 bg-danger-bg/50 p-3">
					<input type="checkbox" bind:checked={hardDelete} class="mt-0.5" />
					<span class="text-xs leading-relaxed text-text-primary">
						物理删除（不可逆）：从所有历史引用中移除该标签，删除后同名标签视为新标签可重新创建。
					</span>
				</label>
			</div>
			<div class="flex justify-end gap-2 px-5 pb-5">
				<Button type="button" variant="secondary" size="md" onclick={() => (banTarget = null)}>取消</Button>
				<Button type="button" variant="secondary" size="md" disabled={pending} onclick={() => void unbanCurrent()}>解除屏蔽</Button>
				<Button type="button" variant={hardDelete ? 'danger' : 'primary'} size="md" loading={pending} onclick={() => void submitBan(true)}>
					{hardDelete ? '物理删除' : '确认屏蔽'}
				</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>
{/if}
