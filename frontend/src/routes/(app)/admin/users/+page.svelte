<script lang="ts">
	import { adminSearchUsers, adminSetUserBan } from '@/api';
	import type { AdminUserItem, UserAccountStatus, UserType } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import { TABLE_WRAP, TABLE, TH, TR, TD, TD_ACTIONS, ROW_BTN_DANGER } from '../shared/table';
	import { formatDateTime } from '@/lib/utils';

	// 用户管理：筛选（状态 / 注册类型 / 按昵称或 ID 搜索）+ 封禁、解封（两态统一 set_ban）。

	const canWrite = $derived(hasAdminPermission('admin.users.write'));
	const canRead = $derived(hasAdminPermission('admin.users.read'));

	const STATUS_TABS = [
		{ id: '', label: '全部' },
		{ id: 'normal', label: '正常' },
		{ id: 'banned', label: '已封禁' }
	];
	let status = $state('');
	let userType = $state<'' | UserType>('');
	let searchField = $state<'nickname' | 'user_id' | 'login_id'>('nickname');
	let query = $state('');

	const list = createAdminList<AdminUserItem>(
		(cursor, limit) =>
			adminSearchUsers({
				nickname: searchField === 'nickname' && query ? query : null,
				user_id: searchField === 'user_id' && query ? query : null,
				login_id: searchField === 'login_id' && query ? query : null,
				status: (status || null) as UserAccountStatus | null,
				user_type: (userType || null) as UserType | null,
				pagination: { limit, cursor }
			}),
		() => `${query}|${searchField}|${status}|${userType}`
	);

	// 封禁/解封弹层（两态同一 Dialog，按 target 当前状态切换文案）
	let banTarget = $state<AdminUserItem | null>(null);
	let banReason = $state('');
	let banDays = $state<number | null>(null);
	let banPending = $state(false);

	const openBan = (u: AdminUserItem) => {
		banTarget = u;
		banReason = '';
		banDays = null;
	};

	const submitBan = async (active: boolean) => {
		if (!banTarget || banPending) return;
		banPending = true;
		try {
			const res = await adminSetUserBan({
				user_id: banTarget.user.user_id,
				active,
				reason: banReason.trim() || null,
				duration_days: active ? (banDays && banDays > 0 ? Math.floor(banDays) : null) : null
			});
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast(active ? '已封禁' : '已解封', 'success');
			banTarget = null;
			list.reload();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			banPending = false;
		}
	};

	const statusText = (u: AdminUserItem) => (u.status === 'banned' ? (u.banned_until ? `封禁至 ${formatDateTime(u.banned_until)}` : '永久封禁') : '正常');
</script>

<svelte:head>
	<title>用户管理 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.users.read 权限" />
{:else}
	<div class="flex flex-col gap-4">
		<div class="flex flex-col gap-3 sm:flex-row sm:items-center">
			<div class="w-full max-w-80">
				<SearchInput value={query} onChange={(v) => (query = v)} placeholder="搜索用户…" class="px-4 py-2.5 rounded-md" />
			</div>
			<select bind:value={searchField} class="input-surface rounded-md px-3 py-2 text-xs text-text-primary" aria-label="搜索字段">
				<option value="nickname">按昵称</option>
				<option value="user_id">按用户 ID</option>
				<option value="login_id">按登录 ID</option>
			</select>
			<select bind:value={userType} class="input-surface rounded-md px-3 py-2 text-xs text-text-primary" aria-label="注册类型">
				<option value="">全部类型</option>
				<option value="normal">正式</option>
				<option value="temp">临时</option>
			</select>
		</div>

		<TabBar items={STATUS_TABS} value={status} onChange={(id) => (status = id)} />

		{#if list.loading}
			<InlineLoading label="加载用户列表…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的用户" />
		{:else}
			<div class={TABLE_WRAP}>
				<table class={TABLE}>
					<thead>
						<tr>
							<th class={TH}>用户</th>
							<th class={TH}>登录 ID</th>
							<th class={TH}>状态</th>
							<th class={TH}>邮箱</th>
							<th class={TH}>注册时间</th>
							<th class={TH}>最后活跃</th>
							<th class={TH}></th>
						</tr>
					</thead>
					<tbody>
						{#each list.items as u (u.user.user_id)}
							<tr class={TR}>
								<td class={TD}>
									<span class="flex items-center gap-2">
										<span class="text-text-primary">{u.user.nickname}</span>
										<span class="text-xs text-text-secondary font-mono">{u.user.user_id}</span>
									</span>
								</td>
								<td class={TD}>{u.login_id}</td>
								<td class={TD}>
									<span class="badge {u.status === 'banned' ? 'bg-danger-bg text-status-danger' : 'bg-surface-subtle text-text-secondary'}">{statusText(u)}</span>
								</td>
								<td class={TD}>{u.email ?? '—'}</td>
								<td class={TD}>{formatDateTime(u.created_at)}</td>
								<td class={TD}>{u.last_active_at ? formatDateTime(u.last_active_at) : '—'}</td>
								<td class={TD_ACTIONS}>
									{#if canWrite}
										<button type="button" class={u.status === 'banned' ? ROW_BTN_DANGER : ROW_BTN_DANGER} onclick={() => openBan(u)}>
											{u.status === 'banned' ? '解封' : '封禁'}
										</button>
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
		open={banTarget !== null}
		title={banTarget?.status === 'banned' ? '解封用户' : '封禁用户'}
		description={banTarget?.status === 'banned'
			? `目标：${banTarget?.user.nickname}（${banTarget?.user.user_id}）`
			: `目标：${banTarget?.user.nickname}（${banTarget?.user.user_id}）。封禁将断开其在线连接并阻止登录，操作记入审计日志。`}
		confirmLabel={banTarget?.status === 'banned' ? '确认解封' : '确认封禁'}
		confirmVariant={banTarget?.status === 'banned' ? 'primary' : 'danger'}
		loading={banPending}
		onCancel={() => (banTarget = null)}
		onConfirm={() => void submitBan(banTarget?.status !== 'banned')}
	>
		{#snippet extra()}
			<div class="flex flex-col gap-3">
				<label class="flex flex-col gap-1.5">
					<span class="text-xs text-text-secondary">原因（记入审计日志，可留空）</span>
					<textarea bind:value={banReason} rows={2} maxlength={500} class="input-surface resize-none rounded-md px-3 py-2 text-sm text-text-primary"></textarea>
				</label>
				{#if banTarget?.status !== 'banned'}
					<label class="flex flex-col gap-1.5">
						<span class="text-xs text-text-secondary">封禁天数（留空或 0 = 永久）</span>
						<input type="number" min={0} bind:value={banDays} class="input-surface rounded-md px-3 py-2 text-sm text-text-primary" />
					</label>
				{/if}
			</div>
		{/snippet}
	</ConfirmDialog>
{/if}
