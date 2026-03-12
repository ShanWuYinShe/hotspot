<script lang="ts">
	import { onMount } from 'svelte';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { authService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { SessionItem } from '$lib/types/api';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Shield from 'lucide-svelte/icons/shield';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	let sessions = $state.raw<SessionItem[]>([]);
	let isLoading = $state(true);
	let isRevoking = $state<string | null>(null);

	function formatTime(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleString('zh-CN', { hour12: false });
	}

	async function loadSessions() {
		isLoading = true;
		try {
			const response = await withMinimumLoadingTime(() => authService.getSessions());
			if (isSuccess(response) && response.data) {
				sessions = response.data.items;
			}
		} catch {
			toast.error('加载会话失败');
		} finally {
			isLoading = false;
		}
	}

	async function revoke(sessionId: string) {
		isRevoking = sessionId;
		try {
			const response = await authService.revokeSession({ session_id: sessionId });
			if (isSuccess(response)) {
				sessions = sessions.filter((item) => item.session_id !== sessionId);
				toast.success('已撤销会话');
			} else {
				toast.error(response.message || '撤销失败');
			}
		} catch {
			toast.error('撤销失败');
		} finally {
			isRevoking = null;
		}
	}

	onMount(() => {
		loadSessions();
	});
</script>

<svelte:head>
	<title>会话管理 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-3xl space-y-5">
	<PageHeader title="会话管理" description="查看并管理已登录设备">
		{#snippet actions()}
			<Button
				variant="outline"
				size="sm"
				class="rounded-2xl border-border/60 bg-background/70"
				onclick={loadSessions}
				disabled={isLoading}
			>
				<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
				刷新
			</Button>
		{/snippet}
	</PageHeader>

	{#if isLoading}
		<div class="space-y-3">
			{#each Array(4) as _, i (i)}
				<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
					<CardContent class="flex items-center gap-4 p-4">
						<Skeleton class="h-12 w-12 rounded-xl" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-48" />
							<Skeleton class="h-3 w-64" />
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>
	{:else if sessions.length === 0}
		<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
			<CardContent class="flex flex-col items-center justify-center py-12">
				<Shield class="mb-4 h-10 w-10 text-muted-foreground" />
				<p class="text-muted-foreground">暂无会话记录</p>
			</CardContent>
		</Card>
	{:else}
		<div class="space-y-3">
			{#each sessions as session (session.session_id)}
				<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
					<CardContent class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
						<div class="min-w-0">
							<div class="flex items-center gap-2">
								<p class="font-medium">{session.device_name || session.device_type || '未知设备'}</p>
								{#if session.is_current}
									<Badge variant="secondary">当前设备</Badge>
								{/if}
							</div>
							<div class="mt-1 text-xs text-muted-foreground">
								{session.ip_location || '未知位置'} · 最后活跃 {formatTime(session.last_active_at)}
							</div>
						</div>
						<div>
							<Button
								variant="outline"
								size="sm"
								class="rounded-2xl border-border/60 bg-background/70"
								disabled={session.is_current || isRevoking === session.session_id}
								onclick={() => revoke(session.session_id)}
							>
								{#if isRevoking === session.session_id}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								撤销
							</Button>
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>
	{/if}
</div>
