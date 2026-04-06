<script lang="ts">
	import { onMount } from 'svelte';
	import { reportsService } from '$lib/api/services';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { formatDateTime, formatRelativeTime } from '$lib/utils/format';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
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
			<Button
				variant="outline"
				size="sm"
				class="rounded-2xl border-border/60 bg-background/70"
				onclick={() => loadReports(true)}
				disabled={pagination.loading}
			>
				<RefreshCw class="mr-2 h-4 w-4 {pagination.loading ? 'animate-spin' : ''}" />
				刷新
			</Button>
		{/snippet}
	</PageHeader>

	<div class="space-y-3 rounded-3xl border border-border/60 bg-card/75 p-4 shadow-sm backdrop-blur-md">
		<div>
			<p class="text-sm font-medium text-foreground">筛选条件</p>
			<p class="mt-1 text-xs text-muted-foreground">按状态和目标类型快速定位处理中的举报。</p>
		</div>
		<div class="space-y-3">
			<div class="flex flex-wrap gap-2">
				{#each statusOptions as option (option.value)}
					<Button
						variant={activeStatus === option.value ? 'default' : 'outline'}
						size="sm"
						class="rounded-full"
						disabled={pagination.loading}
						onclick={async () => {
							if (pagination.loading || activeStatus === option.value) return;
							activeStatus = option.value;
							await loadReports(true);
						}}
					>
						{option.label}
					</Button>
				{/each}
			</div>
			<div class="flex flex-wrap gap-2">
				{#each targetTypeOptions as option (option.value)}
					<Button
						variant={activeTargetType === option.value ? 'default' : 'outline'}
						size="sm"
						class="rounded-full"
						disabled={pagination.loading}
						onclick={async () => {
							if (pagination.loading || activeTargetType === option.value) return;
							activeTargetType = option.value;
							await loadReports(true);
						}}
					>
						{option.label}
					</Button>
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
				<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
					<CardContent class="space-y-4 p-4 sm:p-5">
						<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
							<div class="min-w-0 flex-1 space-y-2">
								<div class="flex flex-wrap items-center gap-2">
									<Badge variant="outline" class="rounded-full">{REPORT_TARGET_TYPE_LABEL[item.target_type]}</Badge>
									<Badge class={`rounded-full border ${REPORT_STATUS_BADGE_CLASS[item.status]}`}>
										{REPORT_STATUS_LABEL[item.status]}
									</Badge>
								</div>
								<div>
									<h2 class="text-base font-semibold text-foreground">{REPORT_REASON_LABEL[item.reason]}</h2>
									<p class="mt-1 text-sm text-muted-foreground">目标 ID：{item.target_id}</p>
								</div>
							</div>
							<div class="text-sm text-muted-foreground">
								<p>{formatRelativeTime(item.created_at)}</p>
							</div>
						</div>

						<div class="grid gap-3 rounded-2xl border border-border/50 bg-background/60 px-4 py-3 text-sm sm:grid-cols-2 lg:grid-cols-3">
							<div>
								<p class="text-xs text-muted-foreground">提交时间</p>
								<p class="mt-1 font-medium">{formatDateTime(item.created_at)}</p>
							</div>
							<div>
								<p class="text-xs text-muted-foreground">处理时间</p>
								<p class="mt-1 font-medium">{item.handled_at ? formatDateTime(item.handled_at) : '尚未处理'}</p>
							</div>
							<div>
								<p class="text-xs text-muted-foreground">举报编号</p>
								<p class="mt-1 font-mono text-xs font-medium">{item.report_id}</p>
							</div>
						</div>

						{#if item.description}
							<div class="rounded-2xl border border-border/50 bg-background/60 px-4 py-3">
								<p class="text-xs text-muted-foreground">补充说明</p>
								<p class="mt-1 text-sm leading-6 text-foreground">{item.description}</p>
							</div>
						{/if}
					</CardContent>
				</Card>
			{/each}
		</div>

		<LoadMoreButton
			loading={pagination.loadingMore}
			hasMore={pagination.hasMore}
			onLoadMore={() => loadReports(false)}
		/>
	{/if}
</div>
