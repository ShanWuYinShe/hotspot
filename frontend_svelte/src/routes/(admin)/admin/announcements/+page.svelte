<script lang="ts">
	import { onMount } from 'svelte';
	import { adminAnnouncementsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
	import ConfirmDialog from '$lib/components/common/confirm-dialog.svelte';
	import type { SystemAnnouncementItem, AnnouncementSeverity } from '$lib/types/api';
	import { formatDateTime } from '$lib/utils/format';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Plus from 'lucide-svelte/icons/plus';
	import Megaphone from 'lucide-svelte/icons/megaphone';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	let deleteConfirmOpen = $state(false);
	let pendingDeleteId = $state<string | null>(null);

	// ============ 分页 ============

	const pagination = usePagination<SystemAnnouncementItem>({
		uniqueKey: 'announcement_id'
	});

	function loadAnnouncements(refresh = true) {
		pagination.load(
			(cursor) =>
				adminAnnouncementsService.search({ pagination: { limit: 20, cursor } }),
			refresh
		);
	}

	// 错误提示
	$effect(() => {
		if (pagination.error) toast.error(pagination.error);
	});

	// ============ 创建公告 ============

	let showDialog = $state(false);
	let isSubmitting = $state(false);
	let title = $state('');
	let content = $state('');
	let severity = $state<AnnouncementSeverity>('info');

	function closeOverlayDialogs() {
		showDialog = false;
		deleteConfirmOpen = false;
		pendingDeleteId = null;
	}

	function openCreateDialog() {
		closeOverlayDialogs();
		title = '';
		content = '';
		severity = 'info';
		showDialog = true;
	}

	async function handleSubmit() {
		if (!title.trim() || !content.trim()) {
			toast.error('请填写完整信息');
			return;
		}

		isSubmitting = true;
		try {
			const response = await adminAnnouncementsService.create({
				title: title.trim(),
				content: content.trim(),
				severity
			});

			if (isSuccess(response)) {
				toast.success('创建成功');
				showDialog = false;
				loadAnnouncements();
			} else {
				toast.error(response.message || '创建失败');
			}
		} catch {
			toast.error('操作失败');
		} finally {
			isSubmitting = false;
		}
	}

	function requestDelete(announcementId: string) {
		closeOverlayDialogs();
		pendingDeleteId = announcementId;
		deleteConfirmOpen = true;
	}

	async function handleDelete() {
		if (!pendingDeleteId) return;
		try {
			const response = await adminAnnouncementsService.delete({
				announcement_id: pendingDeleteId
			});
			if (isSuccess(response)) {
				const deletedId = pendingDeleteId;
				pagination.removeItem((a) => a.announcement_id === deletedId);
				toast.success('已删除');
			} else {
				toast.error(response.message || '删除失败');
			}
		} catch {
			toast.error('删除失败');
		} finally {
			pendingDeleteId = null;
		}
	}

	// ============ 常量 ============

	type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

	const SEVERITY_BADGE: Record<AnnouncementSeverity, { variant: BadgeVariant; text: string }> = {
		error: { variant: 'destructive', text: '紧急' },
		warning: { variant: 'default', text: '警告' },
		info: { variant: 'secondary', text: '通知' }
	};

	onMount(() => {
		loadAnnouncements();
	});
</script>

<svelte:head>
	<title>公告管理 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<PageHeader title="公告管理">
		{#snippet actions()}
			<div class="flex gap-2">
				<Button
					variant="outline"
					size="sm"
					onclick={() => loadAnnouncements()}
					disabled={pagination.loading}
				>
					<RefreshCw class="mr-2 h-4 w-4 {pagination.loading ? 'animate-spin' : ''}" />
					刷新
				</Button>
				<Button size="sm" onclick={openCreateDialog}>
					<Plus class="mr-2 h-4 w-4" />
					发布公告
				</Button>
			</div>
		{/snippet}
	</PageHeader>

	<!-- Announcements list -->
	<Card>
		<CardHeader>
			<CardTitle class="flex items-center gap-2">
				<Megaphone class="h-5 w-5" />
				公告列表
			</CardTitle>
		</CardHeader>
		<CardContent>
			{#if pagination.loading && pagination.items.length === 0}
				<div class="space-y-4">
					{#each Array(3) as _, i (i)}
						<div class="space-y-2 rounded-lg border p-4">
							<Skeleton class="h-5 w-48" />
							<Skeleton class="h-4 w-full" />
							<Skeleton class="h-4 w-3/4" />
						</div>
					{/each}
				</div>
			{:else if pagination.items.length === 0}
				<div class="py-8 text-center text-muted-foreground">暂无公告</div>
			{:else}
				<div class="space-y-4">
					{#each pagination.items as announcement (announcement.announcement_id)}
						{@const badge = SEVERITY_BADGE[announcement.severity]}
						<div class="rounded-lg border p-4">
							<div class="flex items-start justify-between gap-4">
								<div class="flex-1">
									<div class="flex items-center gap-2">
										<h3 class="font-semibold">{announcement.title}</h3>
										<Badge variant={badge.variant}>
											{badge.text}
										</Badge>
									</div>
									<p class="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">
										{announcement.content}
									</p>
									<p class="mt-2 text-xs text-muted-foreground">
										发布于 {formatDateTime(announcement.created_at)}
									</p>
								</div>
								<Button
									variant="ghost"
									size="icon"
									class="text-destructive"
									onclick={() => requestDelete(announcement.announcement_id)}
									aria-label="删除公告"
								>
									<Trash2 class="h-4 w-4" />
								</Button>
							</div>
						</div>
					{/each}
				</div>

			{#if pagination.hasMore}
				<Button variant="outline" class="mt-4 w-full" onclick={() => loadAnnouncements(false)}>
					加载更多
				</Button>
			{/if}
			{/if}
		</CardContent>
	</Card>
</div>

<!-- Create dialog -->
<Dialog.Root bind:open={showDialog}>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>发布公告</Dialog.Title>
		</Dialog.Header>
		<div class="space-y-4 py-4">
			<div class="space-y-2">
				<Label for="title">标题</Label>
				<Input id="title" placeholder="公告标题" bind:value={title} autocomplete="off" />
			</div>
			<div class="space-y-2">
				<Label for="content">内容</Label>
				<Textarea id="content" placeholder="公告内容" rows={5} bind:value={content} />
			</div>
			<div class="space-y-2">
				<Label>级别</Label>
				<div class="flex gap-2">
					<Button
						type="button"
						variant={severity === 'info' ? 'default' : 'outline'}
						size="sm"
						onclick={() => (severity = 'info')}
					>
						通知
					</Button>
					<Button
						type="button"
						variant={severity === 'warning' ? 'default' : 'outline'}
						size="sm"
						onclick={() => (severity = 'warning')}
					>
						警告
					</Button>
					<Button
						type="button"
						variant={severity === 'error' ? 'default' : 'outline'}
						size="sm"
						onclick={() => (severity = 'error')}
					>
						紧急
					</Button>
				</div>
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={closeOverlayDialogs}>取消</Button>
			<Button onclick={handleSubmit} disabled={isSubmitting}>
				{#if isSubmitting}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				发布
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<ConfirmDialog
	bind:open={deleteConfirmOpen}
	title="删除公告"
	description="确定要删除该公告吗？此操作不可撤销。"
	confirmText="删除"
	variant="destructive"
	onconfirm={handleDelete}
/>
