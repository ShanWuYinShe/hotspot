<script lang="ts">
	import { onMount } from 'svelte';
	import { adminReportsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import type { ReportItem, ReportStatus } from '$lib/types/api';
	import { formatDateTime } from '$lib/utils/format';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Flag from 'lucide-svelte/icons/flag';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';
	import Check from 'lucide-svelte/icons/check';
	import X from 'lucide-svelte/icons/x';

	let reports = $state<ReportItem[]>([]);
	let isLoading = $state(true);
	let cursor = $state<string | undefined>(undefined);
	let hasMore = $state(false);
	let statusFilter = $state<ReportStatus | 'all'>('pending');

	async function loadReports(append = false) {
		if (!append) isLoading = true;
		try {
			const response = await adminReportsService.getList({
				status: statusFilter === 'all' ? undefined : statusFilter,
				limit: 20,
				cursor: append ? cursor : undefined
			});

			if (isSuccess(response) && response.data) {
				if (append) {
					reports = [...reports, ...response.data.items];
				} else {
					reports = response.data.items;
				}
				cursor = response.data.pagination.next_cursor;
				hasMore = response.data.pagination.has_more;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	async function handleProcess(reportId: string, action: 'resolve' | 'dismiss') {
		try {
			const response = await adminReportsService.handle({
				report_id: reportId,
				action,
				resolution: action === 'resolve' ? '举报属实，已处理' : '举报不属实'
			});

			if (isSuccess(response)) {
				reports = reports.map(r =>
					r.report_id === reportId ? { ...r, status: action === 'resolve' ? 'resolved' : 'dismissed' } : r
				);
				toast.success(action === 'resolve' ? '已处理' : '已驳回');
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		}
	}

	function getReasonText(reason: string): string {
		const reasonMap: Record<string, string> = {
			spam: '垃圾内容',
			harassment: '骚扰',
			inappropriate: '不当内容',
			violence: '暴力',
			illegal: '违法内容',
			other: '其他'
		};
		return reasonMap[reason] || reason;
	}

	function getTargetTypeText(type: string): string {
		const typeMap: Record<string, string> = {
			user: '用户',
			moment: '瞬间',
			comment: '评论',
			group: '群组',
			message: '消息'
		};
		return typeMap[type] || type;
	}

	function getStatusBadge(status: ReportStatus) {
		switch (status) {
			case 'pending':
				return { variant: 'secondary' as const, text: '待处理' };
			case 'reviewing':
				return { variant: 'default' as const, text: '审核中' };
			case 'resolved':
				return { variant: 'default' as const, text: '已处理' };
			case 'dismissed':
				return { variant: 'outline' as const, text: '已驳回' };
			default:
				return { variant: 'outline' as const, text: status };
		}
	}

	// Watch for filter changes
	$effect(() => {
		loadReports();
	});

	onMount(() => {
		// Initial load handled by $effect
	});
</script>

<svelte:head>
	<title>举报审核 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">举报审核</h1>
		<Button variant="outline" size="sm" onclick={() => loadReports()} disabled={isLoading}>
			<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
			刷新
		</Button>
	</div>

	<!-- Filter -->
	<div class="flex gap-2">
		<Button
			variant={statusFilter === 'pending' ? 'default' : 'outline'}
			size="sm"
			onclick={() => (statusFilter = 'pending')}
		>
			待处理
		</Button>
		<Button
			variant={statusFilter === 'resolved' ? 'default' : 'outline'}
			size="sm"
			onclick={() => (statusFilter = 'resolved')}
		>
			已处理
		</Button>
		<Button
			variant={statusFilter === 'dismissed' ? 'default' : 'outline'}
			size="sm"
			onclick={() => (statusFilter = 'dismissed')}
		>
			已驳回
		</Button>
		<Button
			variant={statusFilter === 'all' ? 'default' : 'outline'}
			size="sm"
			onclick={() => (statusFilter = 'all')}
		>
			全部
		</Button>
	</div>

	<!-- Reports list -->
	<Card>
		<CardHeader>
			<CardTitle class="flex items-center gap-2">
				<Flag class="h-5 w-5" />
				举报列表
			</CardTitle>
		</CardHeader>
		<CardContent>
			{#if isLoading && reports.length === 0}
				<div class="space-y-4">
					{#each Array(5) as _}
						<div class="rounded-lg border p-4">
							<div class="flex items-center gap-4">
								<Skeleton class="h-10 w-10 rounded-full" />
								<div class="flex-1 space-y-2">
									<Skeleton class="h-4 w-32" />
									<Skeleton class="h-3 w-48" />
								</div>
								<Skeleton class="h-6 w-16" />
							</div>
						</div>
					{/each}
				</div>
			{:else if reports.length === 0}
				<div class="py-8 text-center text-muted-foreground">
					{statusFilter === 'pending' ? '暂无待处理举报' : '暂无举报记录'}
				</div>
			{:else}
				<div class="space-y-4">
					{#each reports as report (report.report_id)}
						<div class="rounded-lg border p-4">
							<div class="flex items-start justify-between gap-4">
								<div class="flex items-start gap-3">
									<Avatar class="h-10 w-10">
										<AvatarImage src={report.reporter?.avatar_url} alt={report.reporter?.nickname} />
										<AvatarFallback>{report.reporter?.nickname?.slice(0, 2) || '?'}</AvatarFallback>
									</Avatar>
									<div>
										<div class="flex items-center gap-2">
											<span class="font-medium">{report.reporter?.nickname || '未知用户'}</span>
											<span class="text-muted-foreground">举报了</span>
											<Badge variant="outline">{getTargetTypeText(report.target_type)}</Badge>
										</div>
										<div class="mt-1 flex items-center gap-2">
											<Badge variant="secondary">{getReasonText(report.reason)}</Badge>
											{#if report.description}
												<span class="text-sm text-muted-foreground">{report.description}</span>
											{/if}
										</div>
										<p class="mt-2 text-xs text-muted-foreground">
											{formatDateTime(report.created_at)}
										</p>
									</div>
								</div>
								<div class="flex items-center gap-2">
									<Badge variant={getStatusBadge(report.status).variant}>
										{getStatusBadge(report.status).text}
									</Badge>

									{#if report.status === 'pending'}
										<DropdownMenu.Root>
											<DropdownMenu.Trigger>
												{#snippet child({ props })}
													<Button variant="ghost" size="icon" {...props}>
														<MoreVertical class="h-4 w-4" />
													</Button>
												{/snippet}
											</DropdownMenu.Trigger>
											<DropdownMenu.Content align="end">
												<DropdownMenu.Item>
													{#snippet child({ props })}
														<button {...props} onclick={() => handleProcess(report.report_id, 'resolve')}>
															<Check class="mr-2 h-4 w-4 text-green-500" />
															确认并处理
														</button>
													{/snippet}
												</DropdownMenu.Item>
												<DropdownMenu.Item>
													{#snippet child({ props })}
														<button {...props} onclick={() => handleProcess(report.report_id, 'dismiss')}>
															<X class="mr-2 h-4 w-4 text-red-500" />
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

				{#if hasMore}
					<Button variant="outline" class="mt-4 w-full" onclick={() => loadReports(true)}>
						加载更多
					</Button>
				{/if}
			{/if}
		</CardContent>
	</Card>
</div>
