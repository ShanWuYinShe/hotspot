<script lang="ts">
	import { adminListAuditLogs } from '@/api';
	import type { AuditAction, AuditLogItem, AuditTargetType } from '@/api/types';
	import { hasAdminPermission } from '@/stores/admin.svelte';
	import { ScrollText } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import LoadMoreButton from '@/components/list/LoadMoreButton.svelte';
	import Button from '@/components/ui/button.svelte';
	import { createAdminList } from '../shared/admin-list.svelte';
	import {
		TABLE_WRAP, TABLE, TH, TH_SECONDARY, TR, TD, TD_SECONDARY
	} from '../shared/table';
	import { AUDIT_ACTION_LABELS, AUDIT_TARGET_LABELS } from '../shared/labels';
	import * as Dialog from '@/components/ui/dialog';
	import { scrollFade } from '@/lib/scroll-fade';
	import { formatDateTime } from '@/lib/utils';

	// 审计日志：动作/目标类型过滤；ip_address 仅 admin.audit.read_ip 权限可见（不足为 null）。
	// R13：动作/目标列改中文标签（英文枚举键保留在 title 与详情弹层），筛选控件与其余模块统一；
	// 筛选条件按「动作+目标 / 操作者 / 时间范围」三组收拢，移动端仍为两列网格。

	const canRead = $derived(hasAdminPermission('admin.audit.read'));
	const canReadIp = $derived(hasAdminPermission('admin.audit.read_ip'));

	// 动作筛选下拉：全部 + 按 labels 的顺序（与审计日志动作全集一致）
	const ACTIONS: { value: '' | AuditAction; label: string }[] = [
		{ value: '', label: '全部动作' },
		...Object.entries(AUDIT_ACTION_LABELS).map(([value, label]) => ({ value: value as AuditAction, label }))
	];
	const TARGET_TYPES = [
		{ value: '', label: '全部目标' },
		...Object.entries(AUDIT_TARGET_LABELS).map(([value, label]) => ({ value, label }))
	];
	let action = $state('');
	let targetType = $state('');
	// 操作者/时间范围筛选（r59：时效追溯与安全排查高频;spec FetchAuditLogsRequest 全量支持）
	let operatorId = $state('');
	// 详情全文弹层（r59：详情列截断生硬→点击展开完整内容）
	let detailTarget = $state<AuditLogItem | null>(null);
	let startDate = $state('');
	let endDate = $state('');

	const hasFilter = $derived(Boolean(action || targetType || operatorId || startDate || endDate));
	const clearFilters = () => {
		action = '';
		targetType = '';
		operatorId = '';
		startDate = '';
		endDate = '';
	};

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
	<EmptyState title="无访问权限" description="需要 admin.audit.read 权限" iconTone="accent">
		{#snippet icon()}<ScrollText class="size-7" strokeWidth={1.5} />{/snippet}
	</EmptyState>
{:else}
	<div class="flex flex-col gap-4">
		<!-- 筛选区：移动端两列网格（原为 5 个控件竖向堆叠，实测 241px，把表格推到 y=389，
		     2026-10-29 输入统一 text-md 后控件高度 +3~5px，仍保留网格以免首屏被控件占满）；
		     首屏只剩半张表；全站复审 S4-09）。网格化后同屏可见的控件数不变，仅省竖向空间。
		     日期两件套用 sm:contents 在桌面端还原为父级 flex 的普通子项。 -->
		<div class="grid grid-cols-2 gap-2.5 sm:flex sm:flex-row sm:flex-wrap sm:items-center">
			<select bind:value={action} class="input-surface w-full rounded-md px-3.5 py-2 text-md text-text-primary sm:w-auto" aria-label="动作">
				{#each ACTIONS as a (a.value)}
					<option value={a.value}>{a.label}</option>
				{/each}
			</select>
			<select bind:value={targetType} class="input-surface w-full rounded-md px-3.5 py-2 text-md text-text-primary sm:w-auto" aria-label="目标类型">
				{#each TARGET_TYPES as t (t.value)}
					<option value={t.value}>{t.label}</option>
				{/each}
			</select>
			<input
				bind:value={operatorId}
				class="input-surface col-span-2 w-full rounded-md px-3.5 py-2 text-md text-text-primary sm:col-span-1 sm:w-56"
				placeholder="操作者 user_id…"
				aria-label="操作者 ID"
			/>
			<div class="col-span-2 flex items-center gap-2 sm:contents">
				<input bind:value={startDate} type="date" class="input-surface min-w-0 flex-1 rounded-md px-3.5 py-2 text-md text-text-primary sm:flex-none" aria-label="开始日期" />
				<span class="shrink-0 text-xxs text-text-secondary sm:self-center">至</span>
				<input bind:value={endDate} type="date" class="input-surface min-w-0 flex-1 rounded-md px-3.5 py-2 text-md text-text-primary sm:flex-none" aria-label="结束日期" />
			</div>
			{#if !canReadIp}
				<span class="col-span-2 text-xxs text-text-secondary sm:self-center">IP 字段需 admin.audit.read_ip 权限</span>
			{/if}
			{#if hasFilter}
				<!-- 文字链（内联链接属登记触控例外）；避免伪元素热区覆盖相邻日期输入 -->
				<button type="button" class="link-btn col-span-2 self-center text-xs focus-ring sm:col-span-1" onclick={clearFilters}>清除筛选</button>
			{/if}
		</div>

		{#if list.loading}
			<InlineLoading label="加载审计日志…" />
		{:else if list.error}
			<LoadError message={list.error} onRetry={list.reload} />
		{:else if list.count === 0}
			<EmptyState title="没有匹配的日志" description="调整动作、操作者或时间范围后再试。" iconTone="neutral">
				{#snippet icon()}<ScrollText class="size-7" strokeWidth={1.5} />{/snippet}
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
							<th scope="col" class={TH}>时间</th>
							<th scope="col" class={TH}>操作者</th>
							<th scope="col" class={TH}>动作</th>
							<th scope="col" class={TH}>目标</th>
							<th scope="col" class={TH_SECONDARY}>详情</th>
							<!-- IP / 设备属排障辅助列：1440 及以下折叠进「详情」弹层（避免 8 列挤压后最右列被裁），
							     1536（项目 xl 断点）以上恢复常驻列 -->
							<th scope="col" class={`${TH_SECONDARY} max-xl:hidden`}>IP 位置</th>
							{#if canReadIp}<th scope="col" class={`${TH_SECONDARY} max-xl:hidden`}>IP 地址</th>{/if}
							<th scope="col" class={`${TH_SECONDARY} max-xl:hidden`}>设备</th>
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
											class="link-btn hit-slop-44 py-0.5 focus-ring text-xs"
											title="筛选该操作者的日志"
											onclick={() => (operatorId = log.operator.user_id)}
										>
											{log.operator.nickname}
										</button>
									{:else}
										—
									{/if}
								</td>
								<td class={TD}>
									<!-- 动作列中文优先（英文枚举键保留在 title 供技术核对） -->
									<span class="text-sm text-text-primary" title={log.action}>
										{AUDIT_ACTION_LABELS[log.action] ?? log.action}
									</span>
								</td>
								<td class={TD}>
									<span class="text-xs text-text-secondary">{AUDIT_TARGET_LABELS[log.target_type] ?? log.target_type}</span>
									{#if log.target_name}<span class="ml-1">{log.target_name}</span>{:else}<span class="ml-1 font-mono text-xs text-text-secondary">{log.target_id}</span>{/if}
								</td>
								<td class={TD_SECONDARY}>
									<!-- 截断 + 点击弹窗看全文（r59：生硬截断→可展开） -->
									<button
										type="button"
										class="link-btn hit-slop-44 max-w-72 truncate py-0.5 text-left focus-ring"
										title={detailsText(log.details)}
										onclick={() => (detailTarget = log)}
									>
										{detailsText(log.details)}
									</button>
								</td>
								<td class={`${TD_SECONDARY} max-xl:hidden`}>{log.ip_location ?? '—'}</td>
								{#if canReadIp}<td class={`${TD_SECONDARY} max-xl:hidden`}>{log.ip_address ?? '—'}</td>{/if}
								<td class={`${TD_SECONDARY} max-xl:hidden`}>{log.device_summary ?? '—'}</td>
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
				{detailTarget ? `${AUDIT_ACTION_LABELS[detailTarget.action] ?? detailTarget.action} · ${formatDateTime(detailTarget.created_at)}` : ''}
			</Dialog.Title>
			{#if detailTarget?.operator}
				<p class="mt-1 text-xs text-text-secondary">操作者：{detailTarget.operator.nickname}（{detailTarget.operator.user_id}）</p>
			{/if}
			<!-- 窄桌面（≤1440）折叠进本弹层的排障字段 -->
			<div class="mt-3 grid grid-cols-1 gap-x-4 gap-y-2 text-xs sm:grid-cols-2">
				<p class="flex gap-2">
					<span class="shrink-0 text-text-secondary">IP 位置</span>
					<span class="min-w-0 text-text-primary">{detailTarget?.ip_location ?? '—'}</span>
				</p>
				<p class="flex gap-2">
					<span class="shrink-0 text-text-secondary">设备</span>
					<span class="min-w-0 text-text-primary">{detailTarget?.device_summary ?? '—'}</span>
				</p>
				{#if canReadIp}
					<p class="flex gap-2">
						<span class="shrink-0 text-text-secondary">IP 地址</span>
						<span class="min-w-0 font-mono text-text-primary">{detailTarget?.ip_address ?? '—'}</span>
					</p>
				{/if}
			</div>
			<pre class="mt-4 max-h-96 overflow-auto rounded-md bg-surface-subtle p-3 font-mono text-xxs leading-relaxed whitespace-pre-wrap break-all text-text-secondary">{detailTarget ? JSON.stringify(detailTarget.details, null, 2) : ''}</pre>
		</div>
	</Dialog.Content>
</Dialog.Root>
