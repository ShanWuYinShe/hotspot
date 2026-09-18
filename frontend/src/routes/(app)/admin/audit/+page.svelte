<script lang="ts">
	import { adminListAuditLogs } from '@/api';
	import type { AuditAction, AuditLogItem, AuditTargetType } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import { TABLE_WRAP, TABLE, TH, TR, TD } from '../shared/table';
	import { formatDateTime } from '@/lib/utils';

	// 审计日志：动作/目标类型过滤；ip_address 仅 admin.audit.read_ip 权限可见（不足为 null）。

	const canRead = $derived(hasAdminPermission('admin.audit.read'));
	const canReadIp = $derived(hasAdminPermission('admin.audit.read_ip'));

	const ACTIONS: { value: '' | AuditAction; label: string }[] = [
		{ value: '', label: '全部动作' },
		{ value: 'user.ban', label: '封禁用户' },
		{ value: 'user.unban', label: '解封用户' },
		{ value: 'group.dismiss', label: '解散群组' },
		{ value: 'announcement.create', label: '发布公告' },
		{ value: 'announcement.update', label: '更新公告' },
		{ value: 'announcement.delete', label: '删除公告' },
		{ value: 'role.create', label: '创建角色' },
		{ value: 'role.update', label: '更新角色' },
		{ value: 'role.delete', label: '删除角色' },
		{ value: 'role.set_permissions', label: '设置角色权限' },
		{ value: 'admin.award', label: '授予管理员' },
		{ value: 'admin.revoke', label: '撤销管理员' },
		{ value: 'report.judge', label: '裁决举报' },
		{ value: 'tag.set_ban', label: '标签屏蔽' },
		{ value: 'config.update', label: '更新配置' }
	];
	const TARGET_TYPES = [
		{ value: '', label: '全部目标' },
		{ value: 'user', label: '用户' },
		{ value: 'group', label: '群组' },
		{ value: 'announcement', label: '公告' },
		{ value: 'role', label: '角色' },
		{ value: 'admin', label: '管理员' },
		{ value: 'report', label: '举报' },
		{ value: 'tag', label: '标签' },
		{ value: 'config', label: '配置' }
	];
	let action = $state('');
	let targetType = $state('');

	const list = createAdminList<AuditLogItem>(
		(cursor, limit) =>
			adminListAuditLogs({
				action: (action || null) as AuditAction | null,
				target_type: (targetType || null) as AuditTargetType | null,
				pagination: { limit, cursor }
			}),
		() => `${action}|${targetType}`
	);

	const detailsText = (d: unknown): string => {
		if (!d || typeof d !== 'object') return '—';
		const { type, ...rest } = d as Record<string, unknown> & { type: string };
		const parts = Object.entries(rest)
			.filter(([, v]) => v !== null && v !== undefined)
			.map(([k, v]) => `${k}: ${typeof v === 'object' ? JSON.stringify(v) : String(v)}`);
		return `${type}${parts.length ? ' · ' + parts.join(' · ') : ''}`;
	};
</script>

<svelte:head>
	<title>审计日志 · HotSpot</title>
</svelte:head>

{#if !canRead}
	<EmptyState title="无访问权限" description="需要 admin.audit.read 权限" />
{:else}
	<div class="flex flex-col gap-4">
		<div class="flex flex-col gap-3 sm:flex-row">
			<select bind:value={action} class="input-surface rounded-md px-3 py-2 text-xs text-text-primary" aria-label="动作">
				{#each ACTIONS as a (a.value)}
					<option value={a.value}>{a.label}</option>
				{/each}
			</select>
			<select bind:value={targetType} class="input-surface rounded-md px-3 py-2 text-xs text-text-primary" aria-label="目标类型">
				{#each TARGET_TYPES as t (t.value)}
					<option value={t.value}>{t.label}</option>
				{/each}
			</select>
			{#if !canReadIp}
				<span class="text-xxs self-center text-text-secondary">IP 字段需 admin.audit.read_ip 权限</span>
			{/if}
		</div>

		{#if list.loading}
			<InlineLoading label="加载审计日志…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的日志" />
		{:else}
			<div class={TABLE_WRAP}>
				<table class={TABLE}>
					<thead>
						<tr>
							<th scope="col" class={TH}>时间</th>
							<th scope="col" class={TH}>操作者</th>
							<th scope="col" class={TH}>动作</th>
							<th scope="col" class={TH}>目标</th>
							<th scope="col" class={TH}>详情</th>
							<th scope="col" class={TH}>IP 位置</th>
							{#if canReadIp}<th scope="col" class={TH}>IP 地址</th>{/if}
							<th scope="col" class={TH}>设备</th>
						</tr>
					</thead>
					<tbody>
						{#each list.items as log (log.log_id)}
							<tr class={TR}>
								<td class={TD}>{formatDateTime(log.created_at)}</td>
								<td class={TD}>{log.operator?.nickname ?? '—'}</td>
								<td class={TD}><span class="font-mono text-xs">{log.action}</span></td>
								<td class={TD}>
									<span class="text-xs text-text-secondary">{log.target_type}</span>
									{#if log.target_name}<span class="ml-1">{log.target_name}</span>{:else}<span class="ml-1 font-mono text-xs text-text-secondary">{log.target_id}</span>{/if}
								</td>
								<td class={`${TD} max-w-72 truncate`} title={detailsText(log.details)}>{detailsText(log.details)}</td>
								<td class={TD}>{log.ip_location ?? '—'}</td>
								{#if canReadIp}<td class={TD}>{log.ip_address ?? '—'}</td>{/if}
								<td class={TD}>{log.device_summary ?? '—'}</td>
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
{/if}
