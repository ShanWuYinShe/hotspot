<script lang="ts">
	import { onMount } from 'svelte';
	import { reportsService } from '$lib/api/services';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { formatDateTime, formatRelativeTime } from '$lib/utils/format';
	import {
		REPORT_REASON_LABEL,
		REPORT_STATUS_BADGE_CLASS,
		REPORT_STATUS_LABEL,
		REPORT_TARGET_TYPE_LABEL
	} from '$lib/utils/content-labels';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { ReportItem, ReportStatus, ReportTargetType } from '$lib/types/api';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
	import ShieldAlert from 'lucide-svelte/icons/shield-alert';

	const statusOptions: Array<{ value: ReportStatus | 'all'; label: string }> = [
		{ value: 'all', label: '全部状态' },
		{ value: 'pending', label: '待处理' },
		{ value: 'reviewing', label: '审核中' },
		{ value: 'resolved', label: '已处理' },
		{ value: 'dismissed', label: '已驳回' }
	];

	const targetTypeOptions: Array<{ value: ReportTargetType | 'all'; label: string }> = [
		{ value: 'all', label: '全部类型' },
		{ value: 'user', label: REPORT_TARGET_TYPE_LABEL.user },
		{ value: 'moment', label: REPORT_TARGET_TYPE_LABEL.moment },
		{ value: 'comment', label: REPORT_TARGET_TYPE_LABEL.comment },
		{ value: 'message', label: REPORT_TARGET_TYPE_LABEL.message },
		{ value: 'group', label: REPORT_TARGET_TYPE_LABEL.group }
	];

	const pagination = usePagination<ReportItem>({
		uniqueKey: (item) => item.report_id
	});

	let activeStatus = $state<ReportStatus | 'all'>('all');
	let activeTargetType = $state<ReportTargetType | 'all'>('all');

	async function loadReports(refresh = false) {
		await pagination.load(
			(cursor) =>
				reportsService.getMyReports({
					status: activeStatus === 'all' ? undefined : activeStatus,
					target_type: activeTargetType === 'all' ? undefined : activeTargetType,
					pagination: { limit: 20, cursor }
				}),
			refresh
		);
	}

	onMount(() => {
		loadReports(true);
	});
</script>

<svelte:head>
	<title>我的举报记录 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-4xl space-y-5">
	<PageHeader title="我的举报记录" description="查看你提交过的举报，以及当前处理进度" backHref="/settings" backLabel="返回设置">
		{#snippet actions()}
			<button
				class="flex items-center rounded-full px-4 py-2 text-sm transition-opacity hover:opacity-80 disabled:opacity-50"
				style="background: var(--neu-bg); box-shadow: 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text);"
				onclick={() => loadReports(true)}
				disabled={pagination.loading}
			>
				<RefreshCw size={14} class="mr-2 {pagination.loading ? 'animate-spin' : ''}" />
				刷新
			</button>
		{/snippet}
	</PageHeader>

	<!-- Filter Card -->
	<div class="rounded-[24px] p-4" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
		<div class="mb-3">
			<p style="font-size: 13px; font-weight: 500; color: var(--neu-text);">筛选条件</p>
			<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">按状态和目标类型快速定位处理中的举报。</p>
		</div>
		<div class="space-y-3">
			<div class="flex flex-wrap gap-2">
				{#each statusOptions as option (option.value)}
					<button
						class="rounded-full px-4 py-1.5 text-sm transition-all disabled:opacity-50"
						style={activeStatus === option.value
							? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 8px var(--neu-glow);'
							: 'background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text);'}
						disabled={pagination.loading}
						onclick={async () => {
							if (pagination.loading || activeStatus === option.value) return;
							activeStatus = option.value;
							await loadReports(true);
						}}
					>
						{option.label}
					</button>
				{/each}
			</div>
			<div class="flex flex-wrap gap-2">
				{#each targetTypeOptions as option (option.value)}
					<button
						class="rounded-full px-4 py-1.5 text-sm transition-all disabled:opacity-50"
						style={activeTargetType === option.value
							? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 8px var(--neu-glow);'
							: 'background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text);'}
						disabled={pagination.loading}
						onclick={async () => {
							if (pagination.loading || activeTargetType === option.value) return;
							activeTargetType = option.value;
							await loadReports(true);
						}}
					>
						{option.label}
					</button>
				{/each}
			</div>
		</div>
	</div>

	{#if pagination.loading}
		<ListSkeleton type="notification" count={4} layout="list" />
	{:else if pagination.error}
		<EmptyState
			icon={AlertTriangle}
			title="加载举报记录失败"
			description={pagination.error}
			actionText="重试"
			onAction={() => loadReports(true)}
		/>
	{:else if pagination.items.length === 0}
		<EmptyState
			icon={ShieldAlert}
			title="还没有举报记录"
			description="你提交过的举报会显示在这里，方便随时查看处理进度。"
		/>
	{:else}
		<div class="space-y-3">
			{#each pagination.items as item (item.report_id)}
				<div class="rounded-[24px] p-4 sm:p-5" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
					<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
						<div class="min-w-0 flex-1 space-y-2">
							<div class="flex flex-wrap items-center gap-2">
								<span class="inline-flex items-center rounded-full px-3 py-0.5 text-xs font-medium" style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text-muted);">
									{REPORT_TARGET_TYPE_LABEL[item.target_type]}
								</span>
								<span class={`inline-flex items-center rounded-full border px-3 py-0.5 text-xs font-medium ${REPORT_STATUS_BADGE_CLASS[item.status]}`}>
									{REPORT_STATUS_LABEL[item.status]}
								</span>
							</div>
							<div>
								<h2 style="font-size: 15px; font-weight: 600; color: var(--neu-text);">{REPORT_REASON_LABEL[item.reason]}</h2>
								<p style="font-size: 13px; color: var(--neu-text-muted); margin-top: 4px;">目标 ID：{item.target_id}</p>
							</div>
						</div>
						<div style="font-size: 13px; color: var(--neu-text-muted);">
							<p>{formatRelativeTime(item.created_at)}</p>
						</div>
					</div>

					<div class="mt-4 grid gap-3 rounded-[18px] px-4 py-3 text-sm sm:grid-cols-2 lg:grid-cols-3" style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<div>
							<p style="font-size: 11px; color: var(--neu-text-muted);">提交时间</p>
							<p style="font-size: 13px; font-weight: 500; color: var(--neu-text); margin-top: 4px;">{formatDateTime(item.created_at)}</p>
						</div>
						<div>
							<p style="font-size: 11px; color: var(--neu-text-muted);">处理时间</p>
							<p style="font-size: 13px; font-weight: 500; color: var(--neu-text); margin-top: 4px;">{item.handled_at ? formatDateTime(item.handled_at) : '尚未处理'}</p>
						</div>
						<div>
							<p style="font-size: 11px; color: var(--neu-text-muted);">举报编号</p>
							<p style="font-size: 12px; font-weight: 500; font-family: monospace; color: var(--neu-text); margin-top: 4px;">{item.report_id}</p>
						</div>
					</div>

					{#if item.description}
						<div class="mt-3 rounded-[18px] px-4 py-3" style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<p style="font-size: 11px; color: var(--neu-text-muted);">补充说明</p>
							<p style="font-size: 13px; line-height: 1.6; color: var(--neu-text); margin-top: 4px;">{item.description}</p>
						</div>
					{/if}
				</div>
			{/each}
		</div>

		<LoadMoreButton
			loading={pagination.loadingMore}
			hasMore={pagination.hasMore}
			onLoadMore={() => loadReports(false)}
		/>
	{/if}
</div>
