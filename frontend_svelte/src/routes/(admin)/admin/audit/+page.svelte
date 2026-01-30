<script lang="ts">
	import { onMount } from 'svelte';
	import { adminAuditService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { AuditLogItem } from '$lib/types/api';
	import Search from 'lucide-svelte/icons/search';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import FileText from 'lucide-svelte/icons/file-text';
	import User from 'lucide-svelte/icons/user';
	import Clock from 'lucide-svelte/icons/clock';

	let logs = $state<AuditLogItem[]>([]);
	let isLoading = $state(true);
	let cursor = $state<string | undefined>(undefined);
	let hasMore = $state(false);
	let actionFilter = $state('');

	async function loadLogs(append = false) {
		if (!append) isLoading = true;
		try {
			const response = await adminAuditService.getList({
				action: actionFilter || undefined,
				limit: 30,
				cursor: append ? cursor : undefined
			});

			if (isSuccess(response) && response.data) {
				if (append) {
					logs = [...logs, ...response.data.items];
				} else {
					logs = response.data.items;
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

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleString('zh-CN');
	}

	function getActionBadge(action: string) {
		if (action.includes('create') || action.includes('add')) {
			return { variant: 'default' as const, color: 'text-green-500' };
		}
		if (action.includes('delete') || action.includes('remove') || action.includes('ban')) {
			return { variant: 'destructive' as const, color: 'text-red-500' };
		}
		if (action.includes('update') || action.includes('edit')) {
			return { variant: 'secondary' as const, color: 'text-blue-500' };
		}
		return { variant: 'outline' as const, color: '' };
	}

	onMount(() => {
		loadLogs();
	});
</script>

<svelte:head>
	<title>审计日志 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">审计日志</h1>
		<Button variant="outline" size="sm" onclick={() => loadLogs()} disabled={isLoading}>
			<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
			刷新
		</Button>
	</div>

	<!-- Search -->
	<div class="flex gap-2">
		<div class="relative flex-1">
			<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
			<Input
				type="search"
				placeholder="按操作类型筛选..."
				class="pl-9"
				autocomplete="off"
				bind:value={actionFilter}
				onkeydown={(e) => e.key === 'Enter' && loadLogs()}
			/>
		</div>
		<Button onclick={() => loadLogs()}>筛选</Button>
	</div>

	<!-- Audit logs -->
	<Card>
		<CardHeader>
			<CardTitle class="flex items-center gap-2">
				<FileText class="h-5 w-5" />
				操作日志
			</CardTitle>
		</CardHeader>
		<CardContent>
			{#if isLoading && logs.length === 0}
				<div class="space-y-4">
					{#each Array(10) as _}
						<div class="flex items-center gap-4">
							<Skeleton class="h-10 w-10 rounded-full" />
							<div class="flex-1 space-y-2">
								<Skeleton class="h-4 w-48" />
								<Skeleton class="h-3 w-32" />
							</div>
							<Skeleton class="h-4 w-24" />
						</div>
					{/each}
				</div>
			{:else if logs.length === 0}
				<div class="py-8 text-center text-muted-foreground">暂无审计日志</div>
			{:else}
				<div class="space-y-3">
					{#each logs as log (log.audit_id)}
						<div class="flex items-start gap-4 rounded-lg border p-3">
							<div class="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
								<User class="h-5 w-5 text-muted-foreground" />
							</div>
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<span class="font-medium">{log.operator?.nickname || '系统'}</span>
									<Badge variant={getActionBadge(log.action).variant}>
										{log.action}
									</Badge>
								</div>
								<p class="mt-1 text-sm text-muted-foreground">
									{log.action}
								</p>
								{#if log.target_type && log.target_id}
									<p class="text-xs text-muted-foreground">
										对象: {log.target_type} ({log.target_id})
									</p>
								{/if}
								{#if log.ip_address}
									<p class="text-xs text-muted-foreground">IP: {log.ip_address}</p>
								{/if}
							</div>
							<div class="flex shrink-0 items-center gap-1 text-xs text-muted-foreground">
								<Clock class="h-3 w-3" />
								{formatDate(log.created_at)}
							</div>
						</div>
					{/each}
				</div>

				{#if hasMore}
					<Button variant="outline" class="mt-4 w-full" onclick={() => loadLogs(true)}>
						加载更多
					</Button>
				{/if}
			{/if}
		</CardContent>
	</Card>
</div>
