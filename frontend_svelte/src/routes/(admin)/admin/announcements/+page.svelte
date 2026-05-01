<script lang="ts">
	import { onMount } from 'svelte';
	import { adminAnnouncementsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { NeuButton } from '$lib/components/ui/neu-button';
	import { NeuInput } from '$lib/components/ui/neu-input';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
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

	const SEVERITY_BADGE: Record<AnnouncementSeverity, { variant: BadgeVariant; text: string; color: string }> = {
		error: { variant: 'destructive', text: '紧急', color: 'var(--neu-error)' },
		warning: { variant: 'default', text: '警告', color: 'var(--neu-warning)' },
		info: { variant: 'secondary', text: '通知', color: 'var(--neu-accent-primary)' }
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
				<NeuButton
					variant="flat"
					size="sm"
					onclick={() => loadAnnouncements()}
					disabled={pagination.loading}
				>
					<RefreshCw size={14} strokeWidth={1.75} class={pagination.loading ? 'animate-spin mr-2' : 'mr-2'} />
					刷新
				</NeuButton>
				<NeuButton
					size="sm"
					onclick={openCreateDialog}
				>
					<Plus size={14} strokeWidth={1.75} class="mr-2" />
					发布公告
				</NeuButton>
			</div>
		{/snippet}
	</PageHeader>

	<!-- Announcements list -->
	<div
		class="rounded-[22px] p-5"
		style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
	>
		<!-- Section header -->
		<div class="mb-4 flex items-center gap-2">
			<Megaphone size={18} strokeWidth={1.75} style="color: var(--neu-accent-primary);" />
			<span style="font-size: 16px; font-weight: 600; color: var(--neu-text);">公告列表</span>
		</div>

		{#if pagination.loading && pagination.items.length === 0}
			<div class="space-y-4">
				{#each Array(3) as _, i (i)}
					<div
						class="rounded-[18px] p-4"
						style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border-pressed);"
					>
						<div style="width: 48px; height: 16px; background: var(--neu-border); border-radius: 4px; margin-bottom: 8px;"></div>
						<div style="width: 100%; height: 12px; background: var(--neu-border); border-radius: 4px; margin-bottom: 6px;"></div>
						<div style="width: 75%; height: 12px; background: var(--neu-border); border-radius: 4px;"></div>
					</div>
				{/each}
			</div>
		{:else if pagination.items.length === 0}
			<div class="py-8 text-center" style="font-size: 13px; color: var(--neu-text-muted);">暂无公告</div>
		{:else}
			<div class="space-y-4">
				{#each pagination.items as announcement (announcement.announcement_id)}
					{@const badge = SEVERITY_BADGE[announcement.severity]}
					<div
						class="rounded-[18px] p-4"
						style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border-pressed);"
					>
						<div class="flex items-start justify-between gap-4">
							<div class="flex-1">
								<div class="flex items-center gap-2">
									<span style="font-size: 15px; font-weight: 600; color: var(--neu-text);">{announcement.title}</span>
									<span
										class="rounded-full px-2.5 py-0.5"
										style="font-size: 11px; font-weight: 500; color: {badge.color}; background: {badge.color}15; border: 1px solid {badge.color}30;"
									>
										{badge.text}
									</span>
								</div>
								<p class="mt-2 whitespace-pre-wrap" style="font-size: 13px; color: var(--neu-text-muted);">
									{announcement.content}
								</p>
								<p class="mt-2" style="font-size: 11px; color: var(--neu-text-muted);">
									发布于 {formatDateTime(announcement.created_at)}
								</p>
							</div>
							<button
								type="button"
								class="flex items-center justify-center rounded-full outline-none transition-opacity hover:opacity-70"
								style="width: 36px; height: 36px; box-shadow: 4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
								onclick={() => requestDelete(announcement.announcement_id)}
								aria-label="删除公告"
							>
								<Trash2 size={14} strokeWidth={1.75} style="color: var(--neu-error);" />
							</button>
						</div>
					</div>
				{/each}
			</div>

			<LoadMoreButton loading={pagination.loading} hasMore={pagination.hasMore} onLoadMore={() => loadAnnouncements(false)} />
		{/if}
	</div>
</div>

<!-- Create dialog -->
<Dialog.Root bind:open={showDialog}>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>发布公告</Dialog.Title>
		</Dialog.Header>
		<div class="space-y-4 py-4">
			<div class="space-y-2">
				<label for="title" style="font-size: 12px; color: var(--neu-text);">标题</label>
				<NeuInput id="title" placeholder="公告标题" bind:value={title} autocomplete="off" />
			</div>
			<div class="space-y-2">
				<label for="content" style="font-size: 12px; color: var(--neu-text);">内容</label>
				<textarea id="content" placeholder="公告内容" rows="5" bind:value={content} class="w-full rounded-[14px] px-4 py-2.5 outline-none resize-none" style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); font-size: 13px;"></textarea>
			</div>
			<div class="space-y-2">
				<label style="font-size: 12px; color: var(--neu-text);">级别</label>
				<div class="flex gap-2">
					{#each [
						{ key: 'info', label: '通知', color: 'var(--neu-accent-primary)' },
						{ key: 'warning', label: '警告', color: 'var(--neu-warning)' },
						{ key: 'error', label: '紧急', color: 'var(--neu-error)' }
					] as opt}
						<NeuButton
							type="button"
							variant={severity === opt.key ? 'pressed' : 'flat'}
							size="sm"
							onclick={() => (severity = opt.key as AnnouncementSeverity)}
						>
							{opt.label}
						</NeuButton>
					{/each}
				</div>
			</div>
		</div>
		<Dialog.Footer>
			<NeuButton variant="flat" onclick={closeOverlayDialogs}>取消</NeuButton>
			<NeuButton onclick={handleSubmit} disabled={isSubmitting}>
				{#if isSubmitting}
					<Loader2 size={14} strokeWidth={1.75} class="mr-2 animate-spin" />
				{/if}
				发布
			</NeuButton>
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
