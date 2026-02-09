<script lang="ts">
	import { onMount } from 'svelte';
	import { adminAnnouncementsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
import type { SystemAnnouncementItem, AnnouncementSeverity } from '$lib/types/api';
	import { formatDateTime } from '$lib/utils/format';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Plus from 'lucide-svelte/icons/plus';
	import Megaphone from 'lucide-svelte/icons/megaphone';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import Loader2 from 'lucide-svelte/icons/loader-2';

let announcements = $state.raw<SystemAnnouncementItem[]>([]);
	let isLoading = $state(true);
	let cursor = $state<string | undefined>(undefined);
	let hasMore = $state(false);

	// Create dialog
	let showDialog = $state(false);
	let isSubmitting = $state(false);
	let title = $state('');
	let content = $state('');
	let severity = $state<AnnouncementSeverity>('info');

	async function loadAnnouncements(append = false) {
		if (!append) isLoading = true;
		try {
			const response = await adminAnnouncementsService.search({
				pagination: {
					limit: 20,
					cursor: append ? cursor : undefined
				}
			});

			if (isSuccess(response) && response.data) {
				if (append) {
					announcements = [...announcements, ...response.data.items];
				} else {
					announcements = response.data.items;
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

	function openCreateDialog() {
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

	async function handleDelete(announcementId: string) {
		if (!confirm('确定要删除该公告吗？')) return;

		try {
			const response = await adminAnnouncementsService.delete({ announcement_id: announcementId });
			if (isSuccess(response)) {
				announcements = announcements.filter(a => a.announcement_id !== announcementId);
				toast.success('已删除');
			} else {
				toast.error(response.message || '删除失败');
			}
		} catch {
			toast.error('删除失败');
		}
	}

	function getSeverityBadge(s: AnnouncementSeverity) {
		switch (s) {
			case 'error':
				return { variant: 'destructive' as const, text: '紧急' };
			case 'warning':
				return { variant: 'default' as const, text: '警告' };
			case 'info':
			default:
				return { variant: 'secondary' as const, text: '通知' };
		}
	}

	onMount(() => {
		loadAnnouncements();
	});
</script>

<svelte:head>
	<title>公告管理 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">公告管理</h1>
		<div class="flex gap-2">
			<Button variant="outline" size="sm" onclick={() => loadAnnouncements()} disabled={isLoading}>
				<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
				刷新
			</Button>
			<Button size="sm" onclick={openCreateDialog}>
				<Plus class="mr-2 h-4 w-4" />
				发布公告
			</Button>
		</div>
	</div>

	<!-- Announcements list -->
	<Card>
		<CardHeader>
			<CardTitle class="flex items-center gap-2">
				<Megaphone class="h-5 w-5" />
				公告列表
			</CardTitle>
		</CardHeader>
		<CardContent>
			{#if isLoading && announcements.length === 0}
				<div class="space-y-4">
					{#each Array(3) as _}
						<div class="space-y-2 rounded-lg border p-4">
							<Skeleton class="h-5 w-48" />
							<Skeleton class="h-4 w-full" />
							<Skeleton class="h-4 w-3/4" />
						</div>
					{/each}
				</div>
			{:else if announcements.length === 0}
				<div class="py-8 text-center text-muted-foreground">暂无公告</div>
			{:else}
				<div class="space-y-4">
					{#each announcements as announcement (announcement.announcement_id)}
						<div class="rounded-lg border p-4">
							<div class="flex items-start justify-between gap-4">
								<div class="flex-1">
									<div class="flex items-center gap-2">
										<h3 class="font-semibold">{announcement.title}</h3>
										<Badge variant={getSeverityBadge(announcement.severity).variant}>
											{getSeverityBadge(announcement.severity).text}
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
									onclick={() => handleDelete(announcement.announcement_id)}
									aria-label="删除公告"
								>
									<Trash2 class="h-4 w-4" />
								</Button>
							</div>
						</div>
					{/each}
				</div>

				{#if hasMore}
					<Button variant="outline" class="mt-4 w-full" onclick={() => loadAnnouncements(true)}>
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
			<Button variant="outline" onclick={() => (showDialog = false)}>取消</Button>
			<Button onclick={handleSubmit} disabled={isSubmitting}>
				{#if isSubmitting}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				发布
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
