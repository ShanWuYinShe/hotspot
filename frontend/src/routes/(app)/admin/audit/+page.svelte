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
	import * as Dialog from '@/components/ui/dialog';
	import { TABLE_WRAP, TABLE, TH, TH_SECONDARY, TR, TD, TD_SECONDARY } from '../shared/table';
	import { scrollFade } from '@/lib/scroll-fade';
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
	// 操作者/时间范围筛选（r59：时效追溯与安全排查高频;spec FetchAuditLogsRequest 全量支持）
	let operatorId = $state('');
	// 详情全文弹层（r59：详情列截断生硬→点击展开完整内容）
	let detailTarget = $state<AuditLogItem | null>(null);
	let startDate = $state('');
	let endDate = $state('');

	const toIso = (d: string, endOfDay: boolean): string | null => {
		if (!d) return null;
		const dt = new Date(endOfDay ? `${d}T23:59:59` : `${d}T00:00:00`);
		return isNaN(dt.getTime()) ? null : dt.toISOString();
	};

	const list = createAdminList<AuditLogItem>(
		(cursor, limit) =>
			adminListAuditLogs({
				action: (action || null) as AuditAction | null,
				target_type: (targetType || null) as AuditTargetType | null,
				operator_id: operatorId.trim() || null,
				start_time: toIso(startDate, false),
				end_time: toIso(endDate, true),
				pagination: { limit, cursor }
			}),
		() => `${action}|${targetType}|${operatorId}|${startDate}|${endDate}`
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
	<EmptyState title="无访问权限" description="需要 admin.audit.read 权限" bare />
{:else}
	<div class="flex flex-col gap-4">
		<!-- 筛选区：移动端两列网格（原为 5 个控件竖向堆叠，实测 241px，把表格推到 y=389，
		     首屏只剩半张表；全站复审 S4-09）。网格化后同屏可见的控件数不变，仅省竖向空间。
		     日期两件套用 sm:contents 在桌面端还原为父级 flex 的普通子项。 -->
		<div class="grid grid-cols-2 gap-2.5 sm:flex sm:flex-row sm:flex-wrap sm:items-center sm:gap-3">
			<select bind:value={action} class="input-surface w-full rounded-md px-3 py-2 text-xs text-text-primary sm:w-auto" aria-label="动作">
				{#each ACTIONS as a (a.value)}
					<option value={a.value}>{a.label}</option>
				{/each}
			</select>
			<select bind:value={targetType} class="input-surface w-full rounded-md px-3 py-2 text-xs text-text-primary sm:w-auto" aria-label="目标类型">
				{#each TARGET_TYPES as t (t.value)}
					<option value={t.value}>{t.label}</option>
				{/each}
			</select>
			<input
				bind:value={operatorId}
				class="input-surface col-span-2 w-full rounded-md px-3 py-2 text-xs text-text-primary sm:col-span-1 sm:w-56"
				placeholder="操作者 user_id…"
				aria-label="操作者 ID"
			/>
			<div class="col-span-2 flex items-center gap-2 sm:contents">
				<input bind:value={startDate} type="date" class="input-surface min-w-0 flex-1 rounded-md px-3 py-2 text-xs text-text-primary sm:flex-none" aria-label="开始日期" />
				<span class="text-xxs text-text-secondary shrink-0 sm:self-center">至</span>
				<input bind:value={endDate} type="date" class="input-surface min-w-0 flex-1 rounded-md px-3 py-2 text-xs text-text-primary sm:flex-none" aria-label="结束日期" />
			</div>
			{#if !canReadIp}
				<span class="col-span-2 text-xxs text-text-secondary sm:self-center">IP 字段需 admin.audit.read_ip 权限</span>
			{/if}
		</div>

		{#if list.loading}
			<InlineLoading label="加载审计日志…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的日志" bare />
		{:else}
			<div class={TABLE_WRAP} use:scrollFade>
				<table class={TABLE}>
					<thead>
						<tr>
							<th scope="col" class={TH}>时间</th>
							<th scope="col" class={TH}>操作者</th>
							<th scope="col" class={TH}>动作</th>
							<th scope="col" class={TH}>目标</th>
							<th scope="col" class={TH_SECONDARY}>详情</th>
							<th scope="col" class={TH_SECONDARY}>IP 位置</th>
							{#if canReadIp}<th scope="col" class={TH_SECONDARY}>IP 地址</th>{/if}
							<th scope="col" class={TH_SECONDARY}>设备</th>
						</tr>
					</thead>
					<tbody>
						{#each list.items as log (log.log_id)}
							<tr class={TR}>
								<td class={TD}>{formatDateTime(log.created_at)}</td>
								<td class={TD}>
									{#if log.operator}
										<!-- 点击操作者=按其 user_id 筛选（r59 关联追溯） -->
										<button
											type="button"
											class="link-btn hit-slop-44 focus-ring text-xs"
											title="筛选该操作者的日志"
											onclick={() => (operatorId = log.operator.user_id)}
										>
											{log.operator.nickname}
										</button>
									{:else}
										—
									{/if}
								</td>
								<td class={TD}><span class="font-mono text-xs">{log.action}</span></td>
								<td class={TD}>
									<span class="text-xs text-text-secondary">{log.target_type}</span>
									{#if log.target_name}<span class="ml-1">{log.target_name}</span>{:else}<span class="ml-1 font-mono text-xs text-text-secondary">{log.target_id}</span>{/if}
								</td>
								<td class={TD_SECONDARY}>
									<!-- 截断 + 点击弹窗看全文（r59：生硬截断→可展开） -->
									<button
										type="button"
										class="link-btn hit-slop-44 focus-ring max-w-72 truncate text-left"
										title={detailsText(log.details)}
										onclick={() => (detailTarget = log)}
									>
										{detailsText(log.details)}
									</button>
								</td>
								<td class={TD_SECONDARY}>{log.ip_location ?? '—'}</td>
								{#if canReadIp}<td class={TD_SECONDARY}>{log.ip_address ?? '—'}</td>{/if}
								<td class={TD_SECONDARY}>{log.device_summary ?? '—'}</td>
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

<!-- 详情全文弹层（r59：详情列截断→点击展开完整内容） -->
<Dialog.Root
	open={detailTarget !== null}
	onOpenChange={(next) => (next ? undefined : (detailTarget = null))}
>
	<Dialog.Content class="max-w-120 p-0">
		<div class="px-5 pt-6 pb-5">
			<Dialog.Title class="text-base font-semibold tracking-title text-text-primary">
				{detailTarget ? `${detailTarget.action} · ${formatDateTime(detailTarget.created_at)}` : ''}
			</Dialog.Title>
			{#if detailTarget?.operator}
				<p class="mt-1 text-xs text-text-secondary">操作者：{detailTarget.operator.nickname}（{detailTarget.operator.user_id}）</p>
			{/if}
			<pre class="mt-4 max-h-96 overflow-auto rounded-md bg-surface-subtle p-3 font-mono text-xxs leading-relaxed whitespace-pre-wrap break-all text-text-secondary">{detailTarget ? JSON.stringify(detailTarget.details, null, 2) : ''}</pre>
		</div>
	</Dialog.Content>
</Dialog.Root>
