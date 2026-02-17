<script lang="ts">
	import { adminReportsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Avatar, AvatarFallback } from '$lib/components/ui/avatar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import type { ReportItem, ReportStatus } from '$lib/types/api';
	import { formatDateTime } from '$lib/utils/format';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Flag from 'lucide-svelte/icons/flag';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';
	import Check from 'lucide-svelte/icons/check';
	import X from 'lucide-svelte/icons/x';

	// ============ 分页 ============

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

	// 错误提示
	$effect(() => {
		if (pagination.error) toast.error(pagination.error);
	});

	// 筛选条件变化 → 自动刷新
	$effect(() => {
		statusFilter; // 依赖声明
		loadReports();
	});

	// ============ 操作 ============

	async function handleProcess(reportId: string, action: 'resolve' | 'dismiss') {
		try {
			const response = await adminReportsService.handle({
				report_id: reportId,
				action,
				note: action === 'resolve' ? '举报属实，已处理' : '举报不属实'
			});

			if (isSuccess(response)) {
				const newStatus: ReportStatus = action === 'resolve' ? 'resolved' : 'dismissed';
				pagination.updateItem((r) =>
					r.report_id === reportId ? { ...r, status: newStatus } : null
				);
				toast.success(action === 'resolve' ? '已处理' : '已驳回');
			} else {
				toast.error(response.message || '操作失败');
			}
		} catch {
			toast.error('操作失败');
		}
	}

	// ============ 常量 ============

	type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

	const REASON_LABEL: Record<string, string> = {
		spam: '垃圾内容',
		harassment: '骚扰',
		violence: '暴力',
		hate_speech: '仇恨言论',
		sexual_content: '色情内容',
		misinformation: '虚假信息',
		impersonation: '冒充',
		self_harm: '自残',
		other: '其他'
	};

	const TARGET_TYPE_LABEL: Record<string, string> = {
		user: '用户',
		moment: '瞬间',
		comment: '评论',
		group: '群组',
		message: '消息'
	};

	const STATUS_BADGE: Record<ReportStatus, { variant: BadgeVariant; text: string }> = {
		pending: { variant: 'secondary', text: '待处理' },
		reviewing: { variant: 'default', text: '审核中' },
		resolved: { variant: 'default', text: '已处理' },
		dismissed: { variant: 'outline', text: '已驳回' }
	};
</script>

<svelte:head>
	<title>举报审核 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">举报审核</h1>
		<Button variant="outline" size="sm" onclick={() => loadReports()} disabled={pagination.loading}>
			<RefreshCw class="mr-2 h-4 w-4 {pagination.loading ? 'animate-spin' : ''}" />
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
			{#if pagination.loading && pagination.items.length === 0}
				<div class="space-y-4">
					{#each Array(5) as _, i (i)}
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
			{:else if pagination.items.length === 0}
				<div class="py-8 text-center text-muted-foreground">
					{statusFilter === 'pending' ? '暂无待处理举报' : '暂无举报记录'}
				</div>
			{:else}
				<div class="space-y-4">
					{#each pagination.items as report (report.report_id)}
						{@const badge = STATUS_BADGE[report.status]}
						<div class="rounded-lg border p-4">
							<div class="flex items-start justify-between gap-4">
								<div class="flex items-start gap-3">
									<Avatar class="h-10 w-10">
										<AvatarFallback>?</AvatarFallback>
									</Avatar>
									<div>
										<div class="flex items-center gap-2">
											<span class="font-medium">用户举报</span>
											<span class="text-muted-foreground">举报了</span>
											<Badge variant="outline">{TARGET_TYPE_LABEL[report.target_type] ?? report.target_type}</Badge>
										</div>
										<div class="mt-1 flex items-center gap-2">
											<Badge variant="secondary">{REASON_LABEL[report.reason] ?? report.reason}</Badge>
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
									<Badge variant={badge.variant}>
										{badge.text}
									</Badge>

									{#if report.status === 'pending'}
										<DropdownMenu.Root>
											<DropdownMenu.Trigger>
												{#snippet child({ props })}
													<Button variant="ghost" size="icon" aria-label="更多操作" {...props}>
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

			{#if pagination.hasMore}
				<Button variant="outline" class="mt-4 w-full" onclick={() => loadReports(false)}>
					加载更多
				</Button>
			{/if}
			{/if}
		</CardContent>
	</Card>
</div>
