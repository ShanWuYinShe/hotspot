<script lang="ts">
	import { adminSearchUsers, adminSetUserBan } from '@/api';
	import { contentFade } from '@/lib/motion';
	import type { AdminUserItem, UserAccountStatus, UserType } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { Users } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import Button from '@/components/ui/button.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import {
		TABLE_WRAP, TABLE, TH, TH_SECONDARY, TR, TD, TD_SECONDARY, TD_ACTIONS,
		ROW_BTN, ROW_BTN_DANGER, TOOLBAR, SEARCH_WRAP, SELECT, META, META_MONO
	} from '../shared/table';
	import { scrollFade } from '@/lib/scroll-fade';
	import { formatDateTime } from '@/lib/utils';

	// 用户管理：筛选（状态 / 注册类型 / 按昵称或 ID 搜索）+ 封禁、解封（两态统一 set_ban）。
	// R13：工具栏/筛选/表格语言与其他 8 个模块对齐；封禁态改软底 danger 徽章 + 时间次级说明，
	// 避免一排实底红胶囊压过页面主信息。

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

	const hasFilter = $derived(Boolean(query || status || userType));
	const clearFilters = () => {
		query = '';
		status = '';
		userType = '';
	};

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
		// 天数填了就必须是正整数：负数/0 静默转永久封禁会放大操作后果（后端对负数本就拒绝）。
		if (active && banDays !== null && (!Number.isFinite(banDays) || banDays < 1)) {
			showToast('封禁天数需为正整数，留空表示永久封禁', 'danger');
			return;
		}
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
</script>

<svelte:head>
	<title>用户管理 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.users.read 权限" iconTone="accent">
		{#snippet icon()}<Users class="size-7" strokeWidth={1.5} />{/snippet}
	</EmptyState>
{:else}
	<div class="flex flex-col gap-4">
		<div class={TOOLBAR}>
			<div class={SEARCH_WRAP}>
				<SearchInput value={query} onChange={(v) => (query = v)} placeholder="搜索用户…" class="px-4 py-2.5 rounded-md" />
			</div>
			<select bind:value={searchField} class={SELECT} aria-label="搜索字段">
				<option value="nickname">按昵称</option>
				<option value="user_id">按用户 ID</option>
				<option value="login_id">按登录 ID</option>
			</select>
			<select bind:value={userType} class={SELECT} aria-label="注册类型">
				<option value="">全部类型</option>
				<option value="normal">正式</option>
				<option value="temp">临时</option>
			</select>
		</div>

		<TabBar items={STATUS_TABS} value={status} onChange={(id) => (status = id)} />

		<div use:contentFade={list.loading}>
			{#if list.loading}
				<InlineLoading label="加载用户列表…" />
			{:else if list.error}
				<LoadError message={list.error} onRetry={list.reload} />
			{:else if list.count === 0}
				<EmptyState title="没有匹配的用户" description="换个搜索词，或清除筛选条件后重试。" iconTone="neutral">
					{#snippet icon()}<Users class="size-7" strokeWidth={1.5} />{/snippet}
					{#snippet action()}
						{#if hasFilter}
							<Button type="button" variant="secondary" size="sm" onclick={clearFilters}>清除筛选</Button>
						{/if}
					{/snippet}
				</EmptyState>
			{:else}
				<div class={TABLE_WRAP} use:scrollFade>
					<table class={TABLE}>
						<thead>
							<tr>
								<th scope="col" class={TH}>用户</th>
								<th scope="col" class={TH_SECONDARY}>登录 ID</th>
								<th scope="col" class={TH}>状态</th>
								<th scope="col" class={TH_SECONDARY}>邮箱</th>
								<th scope="col" class={TH_SECONDARY}>注册时间</th>
								<th scope="col" class={TH_SECONDARY}>最后活跃</th>
								<th scope="col" class={TH}><span class="sr-only">操作</span></th>
							</tr>
						</thead>
						<tbody>
							{#each list.items as u (u.user.user_id)}
								<tr class={TR}>
									<td class={TD}>
										<span class="flex items-center gap-2">
											<span class="text-text-primary">{u.user.nickname}</span>
											<span class={`${META_MONO} max-sm:hidden`}>{u.user.user_id}</span>
										</span>
									</td>
									<td class={TD_SECONDARY}>{u.login_id}</td>
									<td class={TD}>
										{#if u.status === 'banned'}
											<span class="badge bg-danger-bg/50 text-text-status-danger-on-soft">已封禁</span>
											<span class={`ml-1.5 ${META}`}>
												{u.banned_until ? `至 ${formatDateTime(u.banned_until)}` : '永久'}
											</span>
										{:else}
											<span class="badge bg-surface-subtle text-text-secondary">正常</span>
										{/if}
									</td>
									<td class={TD_SECONDARY}>{u.email ?? '—'}</td>
									<td class={TD_SECONDARY}>{formatDateTime(u.created_at)}</td>
									<td class={TD_SECONDARY}>{u.last_active_at ? formatDateTime(u.last_active_at) : '—'}</td>
									<td class={TD_ACTIONS}>
										{#if canWrite}
											<button type="button" class={u.status === 'banned' ? ROW_BTN : ROW_BTN_DANGER} onclick={() => openBan(u)}>
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
						<span class="text-xs text-text-secondary">封禁天数（留空 = 永久封禁）</span>
						<input type="number" min={1} bind:value={banDays} class="input-surface rounded-md px-3 py-2 text-sm text-text-primary" />
					</label>
				{/if}
			</div>
		{/snippet}
	</ConfirmDialog>
{/if}
