<script lang="ts">
	import { adminReportsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import type { ReportItem, ReportStatus } from '$lib/types/api';
	import {
		REPORT_REASON_LABEL,
		REPORT_STATUS_LABEL,
		REPORT_TARGET_TYPE_LABEL,
	} from '$lib/utils/content-labels';
	import { formatDateTime } from '$lib/utils/format';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Flag from 'lucide-svelte/icons/flag';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';
	import Check from 'lucide-svelte/icons/check';
	import X from 'lucide-svelte/icons/x';

	const pagination = usePagination<ReportItem>({ uniqueKey: 'report_id' });
	let statusFilter = $state<ReportStatus | 'all'>('pending');

	function loadReports(refresh = true) {
		pagination.load(
			(cursor) =>
				adminReportsService.search({
					status: statusFilter === 'all' ? undefined : statusFilter,
					pagination: { limit: 20, cursor }
				}),
			refresh
		);
	}

	$effect(() => { if (pagination.error) toast.error(pagination.error); });
	$effect(() => { statusFilter; loadReports(); });

	async function handleProcess(reportId: string, action: 'resolve' | 'dismiss') {
		try {
			const response = await adminReportsService.handle({
				report_id: reportId,
				action,
				note: action === 'resolve' ? '举报属实，已处理' : '举报不属实'
			});
			if (isSuccess(response)) {
				const newStatus: ReportStatus = action === 'resolve' ? 'resolved' : 'dismissed';
				pagination.updateItem((r) => r.report_id === reportId ? { ...r, status: newStatus } : null);
				toast.success(action === 'resolve' ? '已处理' : '已驳回');
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch { toast.error('操作失败'); }
	}

	const STATUS_TEXT: Record<string, string> = {
		pending: REPORT_STATUS_LABEL.pending,
		reviewing: REPORT_STATUS_LABEL.reviewing,
		resolved: REPORT_STATUS_LABEL.resolved,
		dismissed: REPORT_STATUS_LABEL.dismissed
	};
</script>

<svelte:head>
	<title>举报审核 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 style="font-size: 22px; font-weight: 600; color: var(--neu-text);">举报审核</h1>
		<button class="flex items-center gap-1.5 rounded-full px-4 py-2 outline-none" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); font-size: 13px; color: var(--neu-text);" onclick={() => loadReports()} disabled={pagination.loading}>
			<RefreshCw size={14} strokeWidth={1.5} class={pagination.loading ? 'animate-spin' : ''} />
			刷新
		</button>
	</div>

	<!-- Filter -->
	<div class="flex gap-2">
		{#each [{ value: 'pending', label: '待处理' }, { value: 'resolved', label: '已处理' }, { value: 'dismissed', label: '已驳回' }, { value: 'all', label: '全部' }] as f (f.value)}
			<button
				class="rounded-full px-4 py-1.5 outline-none"
				style="font-size: 12px; font-weight: 500; {statusFilter === f.value ? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 3px 10px var(--neu-glow);' : 'background: var(--neu-bg); color: var(--neu-text); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);'}"
				onclick={() => (statusFilter = f.value as ReportStatus | 'all')}
			>{f.label}</button>
		{/each}
	</div>

	<!-- Reports list -->
	<div class="rounded-[22px] overflow-hidden" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
		<div class="flex items-center gap-2 px-4 py-3" style="border-bottom: 1px solid var(--neu-border);">
			<Flag size={16} strokeWidth={1.5} color="var(--neu-accent-primary)" />
			<span style="font-size: 14px; font-weight: 500; color: var(--neu-text);">举报列表</span>
		</div>
		<div class="p-4">
			{#if pagination.loading && pagination.items.length === 0}
				<div class="space-y-3">
					{#each Array(5) as _, i (i)}
						<div class="rounded-[18px] p-3" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<div class="flex items-center gap-3">
								<div class="rounded-full" style="width: 36px; height: 36px; background: var(--neu-border);"></div>
								<div class="flex-1 space-y-2">
									<div class="rounded" style="height: 12px; width: 120px; background: var(--neu-border);"></div>
									<div class="rounded" style="height: 10px; width: 160px; background: var(--neu-border);"></div>
								</div>
							</div>
						</div>
					{/each}
				</div>
			{:else if pagination.items.length === 0}
				<p class="py-8 text-center" style="font-size: 13px; color: var(--neu-text-muted);">{statusFilter === 'pending' ? '暂无待处理举报' : '暂无举报记录'}</p>
			{:else}
				<div class="space-y-3">
					{#each pagination.items as report (report.report_id)}
						<div class="rounded-[18px] p-3" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<div class="flex items-start justify-between gap-3">
								<div class="flex items-start gap-3">
									<div class="flex flex-shrink-0 items-center justify-center rounded-full" style="width: 36px; height: 36px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
										<span style="font-size: 12px; color: var(--neu-text-muted);">?</span>
									</div>
									<div>
										<div class="flex items-center gap-1.5">
											<span style="font-size: 13px; font-weight: 500; color: var(--neu-text);">用户举报</span>
											<span style="font-size: 12px; color: var(--neu-text-muted);">举报了</span>
											<span class="rounded-full px-1.5 py-0.5" style="font-size: 10px; background: var(--neu-bg); color: var(--neu-text); border: 1px solid var(--neu-border);">{REPORT_TARGET_TYPE_LABEL[report.target_type] ?? report.target_type}</span>
										</div>
										<div class="flex items-center gap-1.5 mt-1">
											<span class="rounded-full px-1.5 py-0.5" style="font-size: 10px; background: var(--neu-bg); color: var(--neu-text-muted); border: 1px solid var(--neu-border);">{REPORT_REASON_LABEL[report.reason] ?? report.reason}</span>
											{#if report.description}
												<span style="font-size: 12px; color: var(--neu-text-muted);">{report.description}</span>
											{/if}
										</div>
										<p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 4px;">{formatDateTime(report.created_at)}</p>
									</div>
								</div>
								<div class="flex items-center gap-2">
									<span class="rounded-full px-2 py-0.5" style="font-size: 10px; font-weight: 500; {report.status === 'pending' ? 'background: var(--neu-warning); color: var(--neu-active-text);' : report.status === 'resolved' ? 'background: var(--neu-success); color: var(--neu-active-text);' : 'background: var(--neu-border); color: var(--neu-text-muted);'}">{STATUS_TEXT[report.status] ?? report.status}</span>
									{#if report.status === 'pending'}
										<DropdownMenu.Root>
											<DropdownMenu.Trigger>
												{#snippet child({ props })}
													<button class="flex items-center justify-center rounded-full outline-none" style="width: 30px; height: 30px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);" aria-label="更多操作" {...props}>
														<MoreVertical size={14} strokeWidth={1.5} color="var(--neu-text-muted)" />
													</button>
												{/snippet}
											</DropdownMenu.Trigger>
											<DropdownMenu.Content align="end">
												<DropdownMenu.Item>
													{#snippet child({ props })}
														<button style="display: flex; align-items: center; gap: 6px; font-size: 13px;" {...props} onclick={() => handleProcess(report.report_id, 'resolve')}>
															<Check size={14} strokeWidth={1.5} color="var(--neu-success)" />
															确认并处理
														</button>
													{/snippet}
												</DropdownMenu.Item>
												<DropdownMenu.Item>
													{#snippet child({ props })}
														<button style="display: flex; align-items: center; gap: 6px; font-size: 13px;" {...props} onclick={() => handleProcess(report.report_id, 'dismiss')}>
															<X size={14} strokeWidth={1.5} color="var(--neu-error)" />
															驳回举报
														</button>
													{/snippet}
												</DropdownMenu.Item>
											</DropdownMenu.Content>
										</DropdownMenu.Root>
									{/if}
								</div>
							</div>
						</div>
					{/each}
				</div>
				{#if pagination.hasMore}
					<button class="mt-4 w-full rounded-full py-2.5 outline-none" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); font-size: 13px; color: var(--neu-text);" onclick={() => loadReports(false)}>加载更多</button>
				{/if}
			{/if}
		</div>
	</div>
</div>
