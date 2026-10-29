<script lang="ts">
	import { adminSearchTags, adminSetTagBan } from '@/api';
	import type { AdminTagItem } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { Tags } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import {
		TABLE_WRAP, TABLE, TH, TH_NUM, TH_SECONDARY, TR, TD, TD_NUM, TD_SECONDARY, TD_ACTIONS,
		ROW_BTN, TOOLBAR, SEARCH_WRAP
	} from '../shared/table';
	import { scrollFade } from '@/lib/scroll-fade';
	import { formatDateTime } from '@/lib/utils';

	// 标签管理：模糊搜索 + 软屏蔽 / 解屏蔽；物理删除（hard_delete）为高危不可逆，需显式勾选。
	// spec：标签业务唯一键为 name；屏蔽后新内容引用该标签将失败，历史数据保留。
	// R13：工具栏/数字列/徽章语言与其他模块统一（已屏蔽改软底 danger 徽章，不再抢视觉）。

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

<svelte:head>
	<title>标签管理 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.tags.read 权限" iconTone="accent">
		{#snippet icon()}<Tags class="size-7" strokeWidth={1.5} />{/snippet}
	</EmptyState>
{:else}
	<div class="flex flex-col gap-4">
		<div class={TOOLBAR}>
			<div class={SEARCH_WRAP}>
				<SearchInput value={query} onChange={(v) => (query = v)} placeholder="搜索标签…" ariaLabel="搜索标签" />
			</div>
		</div>

		{#if list.loading}
			<InlineLoading label="加载标签列表…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的标签" description="换个关键词试试，或清空搜索词后重试。" iconTone="neutral">
				{#snippet icon()}<Tags class="size-7" strokeWidth={1.5} />{/snippet}
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
							<th scope="col" class={TH}>标签</th>
							<th scope="col" class={TH_NUM}>使用次数</th>
							<th scope="col" class={TH_SECONDARY}>创建时间</th>
							<th scope="col" class={TH}><span class="sr-only">操作</span></th>
						</tr>
					</thead>
					<tbody>
						{#each list.items as t (t.tag_id)}
							<tr class={TR}>
								<td class={TD}>
									<!-- 限宽截断：移动端该表实测仅剩 1px 余量（tableW 356 / wrap 358，全站复审
									     S4-24），标签名变长即溢出；截断后表格宽度不再随标签名无限增长 -->
									<span class="badge max-w-32 truncate bg-accent-soft text-text-accent-on-soft">#{t.name}</span>
									{#if t.is_blocked}
										<!-- 已屏蔽态必须可见：否则与正常标签无从区分（全站复审 s4-22 的根因） -->
										<span class="badge bg-danger-bg/50 text-text-status-danger-on-soft">已屏蔽</span>
									{/if}
								</td>
								<td class={TD_NUM}>{t.usage_count}</td>
								<td class={TD_SECONDARY}>{formatDateTime(t.created_at)}</td>
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
				<LoadMoreButton loading={list.loadingMore} onLoad={list.loadMore} />
			{:else}
				<ListEnd />
			{/if}
		{/if}
	</div>

	<Dialog.Root open={banTarget !== null} onOpenChange={(next) => (next ? undefined : (banTarget = null))}>
		<Dialog.Content class="max-w-90 p-0">
			<div class="px-5 pt-6 pb-4">
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">
					{banTarget?.is_blocked ? `解除屏蔽 #${banTarget?.name}？` : `屏蔽标签 #${banTarget?.name}？`}
				</Dialog.Title>
				<Dialog.Description class="mt-1 text-sm text-text-secondary leading-relaxed">
					{banTarget?.is_blocked
						? '解除后该标签恢复可被新建/搜索命中；历史引用不受影响。'
						: '软屏蔽：该标签不再可被新建/搜索命中，历史数据保留不变。'}
				</Dialog.Description>
				{#if !banTarget?.is_blocked}
				<label class="mt-4 flex items-start gap-2.5 rounded-lg bg-danger-bg/60 p-3">
					<input type="checkbox" bind:checked={hardDelete} class="mt-0.5" />
					<span class="text-xs leading-relaxed text-text-primary">
						物理删除（不可逆）：从所有历史引用中移除该标签，删除后同名标签视为新标签可重新创建。
					</span>
				</label>
				{/if}
			</div>
			<!-- 动作按当前态渲染（全站复审 s4-22）：此前「解除屏蔽」无条件出现，在正常标签上
			     点它会发 no-op 请求并提示成功（spec tags/set_ban 对未屏蔽标签解屏蔽返回成功），
			     误导操作者；现按 is_blocked 只给适用动作。 -->
			<div class="flex justify-end gap-2 px-5 pb-5">
				<Button type="button" variant="secondary" size="md" onclick={() => (banTarget = null)}>取消</Button>
				{#if banTarget?.is_blocked}
					<Button type="button" variant="primary" size="md" loading={pending} onclick={() => void unbanCurrent()}>确认解除屏蔽</Button>
				{:else}
					<Button type="button" variant={hardDelete ? 'danger' : 'primary'} size="md" loading={pending} onclick={() => void submitBan(true)}>
						{hardDelete ? '物理删除' : '确认屏蔽'}
					</Button>
				{/if}
			</div>
		</Dialog.Content>
	</Dialog.Root>
{/if}
