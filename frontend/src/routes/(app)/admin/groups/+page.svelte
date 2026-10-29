<script lang="ts">
	import { adminSearchGroups, adminDismissGroup } from '@/api';
	import type { AdminGroupItem } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { UsersRound } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import Button from '@/components/ui/button.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import {
		TABLE_WRAP, TABLE, TH, TH_NUM, TH_SECONDARY, TR, TD, TD_NUM, TD_SECONDARY, TD_ACTIONS,
		ROW_BTN_DANGER, TOOLBAR, SEARCH_WRAP, META_MONO
	} from '../shared/table';
	import { scrollFade } from '@/lib/scroll-fade';
	import { formatDateTime } from '@/lib/utils';

	// 群组管理：按名称搜索 + 解散（reason 仅入审计日志，不推送给成员）。
	// R13：工具栏/表头/数字列（成员数右对齐等宽数字）/空态与其他模块统一。

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

<svelte:head>
	<title>群组管理 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.groups.read 权限" iconTone="accent">
		{#snippet icon()}<UsersRound class="size-7" strokeWidth={1.5} />{/snippet}
	</EmptyState>
{:else}
	<div class="flex flex-col gap-4">
		<div class={TOOLBAR}>
			<div class={SEARCH_WRAP}>
				<SearchInput value={query} onChange={(v) => (query = v)} placeholder="搜索群组名称…" ariaLabel="搜索群组名称" />
			</div>
		</div>

		{#if list.loading}
			<InlineLoading label="加载群组列表…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的群组" description="换个群组名称试试，或清空搜索词后重试。" iconTone="neutral">
				{#snippet icon()}<UsersRound class="size-7" strokeWidth={1.5} />{/snippet}
				{#snippet action()}
					{#if query}
						<Button type="button" variant="secondary" size="sm" onclick={() => (query = '')}>清空搜索</Button>
					{/if}
				{/snippet}
			</EmptyState>
		{:else}
			<div class={TABLE_WRAP} use:scrollFade>
				<table class={TABLE}>
					<thead>
						<tr>
							<th scope="col" class={TH}>群组</th>
							<th scope="col" class={TH}>群主</th>
							<th scope="col" class={TH_NUM}>成员数</th>
							<th scope="col" class={TH_SECONDARY}>创建时间</th>
							<th scope="col" class={TH}><span class="sr-only">操作</span></th>
						</tr>
					</thead>
					<tbody>
						{#each list.items as g (g.group.group_id)}
							<tr class={TR}>
								<td class={TD}>
									<span class="flex items-center gap-2">
										<span class="text-text-primary">{g.group.name}</span>
										<span class={`${META_MONO} max-sm:hidden`}>{g.group.group_id}</span>
									</span>
								</td>
								<td class={TD}>{g.owner?.nickname ?? '—'}</td>
								<td class={TD_NUM}>{g.group.members_count}</td>
								<td class={TD_SECONDARY}>{formatDateTime(g.created_at)}</td>
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
				<LoadMoreButton loading={list.loadingMore} onLoad={list.loadMore} />
			{:else}
				<ListEnd />
			{/if}
		{/if}
	</div>

	<ConfirmDialog
		open={dismissTarget !== null}
		title="解散群组「{dismissTarget?.group.name}」？"
		description="解散后所有成员将收到系统解散通知（不暴露平台管理员身份），待处理的申请与邀请自动作废。原因仅记入审计日志。"
		confirmLabel="确认解散"
		loading={pending}
		onCancel={() => (dismissTarget = null)}
		onConfirm={() => void submitDismiss()}
	>
		{#snippet extra()}
			<label class="flex flex-col gap-1.5">
				<span class="text-xs text-text-secondary">解散原因（可留空）</span>
				<textarea bind:value={dismissReason} rows={2} maxlength={500} class="input-surface resize-none rounded-md px-3.5 py-2 text-md text-text-primary"></textarea>
			</label>
		{/snippet}
	</ConfirmDialog>
{/if}
