<script lang="ts">
  import { onMount } from 'svelte';
  import { adminAnnouncementsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import { usePagination } from '$lib/hooks/use-pagination.svelte';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import AdminPageHeader from '$lib/components/common/admin-page-header.svelte';
  import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
  import * as Dialog from '$lib/components/ui/dialog';
  import ConfirmDialog from '$lib/components/common/confirm-dialog.svelte';
  import type { SystemAnnouncementItem, AnnouncementSeverity } from '$lib/types/api';
  import { formatDateTime } from '$lib/utils/format';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import Plus from 'lucide-svelte/icons/plus';
  import Megaphone from 'lucide-svelte/icons/megaphone';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import Spinner from '$lib/components/common/spinner.svelte';

  let deleteConfirmOpen = $state(false);
  let pendingDeleteId = $state<string | null>(null);

  // ============ 分页 ============

  const pagination = usePagination<SystemAnnouncementItem>({
    uniqueKey: 'announcement_id',
    onError: (msg) => toast.error(msg)
  });

  function loadAnnouncements(refresh = true) {
    pagination.load(
      (cursor) => adminAnnouncementsService.search({ pagination: { limit: 20, cursor } }),
      refresh
    );
  }

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

  const SEVERITY_BADGE: Record<
    AnnouncementSeverity,
    { variant: BadgeVariant; text: string; color: string }
  > = {
    error: { variant: 'destructive', text: '紧急', color: 'var(--theme-error)' },
    warning: { variant: 'default', text: '警告', color: 'var(--theme-warning)' },
    info: { variant: 'secondary', text: '通知', color: 'var(--brand)' }
  };

  onMount(() => {
    loadAnnouncements();
  });
</script>

<svelte:head>
  <title>公告管理 - HotSpot 管理后台</title>
</svelte:head>

<AdminPageHeader title="公告管理" description="发布与管理公告">
  {#snippet actions()}
    <Button
      variant="flat"
      size="sm"
      onclick={() => loadAnnouncements()}
      disabled={pagination.loading}
    >
      <RefreshCw
        size={14}
        strokeWidth={1.75}
        class={pagination.loading ? 'animate-spin mr-2' : 'mr-2'}
      />
      刷新
    </Button>
    <Button variant="flat" size="sm" onclick={openCreateDialog}>
      <Plus size={14} strokeWidth={1.75} class="mr-2" />
      发布公告
    </Button>
  {/snippet}
</AdminPageHeader>

<div class="page-pad">
  <div class="page-shell page-shell--wide space-y-6">
    <!-- Announcements list -->
    <div
      class="rounded-[var(--radius-lg)] p-5"
      style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
    >
      <!-- Section header -->
      <div class="mb-4 flex items-center gap-2">
        <Megaphone size={18} strokeWidth={1.75} style="color: var(--brand);" />
        <span class="text-title" style="font-weight: 600; color: var(--text-primary)">公告列表</span
        >
      </div>

      {#if pagination.loading && pagination.items.length === 0}
        <div class="space-y-4">
          {#each Array(3) as _, i (i)}
            <div
              class="rounded-[var(--radius-md)] p-4"
              style="background: var(--surface-bg); border: 1px solid var(--border-active);"
            >
              <div
                style="width: 48px; height: 16px; background: var(--border-theme); border-radius: var(--radius-xs); margin-bottom: 8px;"
              ></div>
              <div
                style="width: 100%; height: 12px; background: var(--border-theme); border-radius: var(--radius-xs); margin-bottom: 6px;"
              ></div>
              <div
                style="width: 75%; height: 12px; background: var(--border-theme); border-radius: var(--radius-xs);"
              ></div>
            </div>
          {/each}
        </div>
      {:else if pagination.items.length === 0}
        <div class="text-body py-8 text-center" style="color: var(--text-secondary)">暂无公告</div>
      {:else}
        <div class="space-y-4">
          {#each pagination.items as announcement (announcement.announcement_id)}
            {@const badge = SEVERITY_BADGE[announcement.severity]}
            <div
              class="rounded-[var(--radius-md)] p-4"
              style="background: var(--surface-bg); border: 1px solid var(--border-active);"
            >
              <div class="flex items-start justify-between gap-4">
                <div class="flex-1">
                  <div class="flex items-center gap-2">
                    <span class="text-body-lg" style="font-weight: 600; color: var(--text-primary)"
                      >{announcement.title}</span
                    >
                    <span
                      class="text-caption rounded-full px-2.5 py-0.5"
                      style="font-weight: 500; color: {badge.color}; background: {badge.color}15; border: 1px solid {badge.color}30"
                    >
                      {badge.text}
                    </span>
                  </div>
                  <p
                    class="text-body mt-2 whitespace-pre-wrap"
                    style="color: var(--text-secondary)"
                  >
                    {announcement.content}
                  </p>
                  <p class="text-caption mt-2" style="color: var(--text-secondary)">
                    发布于 {formatDateTime(announcement.created_at)}
                  </p>
                </div>
                <button
                  type="button"
                  class="flex items-center justify-center rounded-full outline-none transition-opacity hover:opacity-70"
                  style="width: 36px; height: 36px; box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
                  onclick={() => requestDelete(announcement.announcement_id)}
                  aria-label="删除公告"
                >
                  <Trash2 size={14} strokeWidth={1.75} style="color: var(--theme-error);" />
                </button>
              </div>
            </div>
          {/each}
        </div>

        <LoadMoreButton
          loading={pagination.loading}
          hasMore={pagination.hasMore}
          onLoadMore={() => loadAnnouncements(false)}
        />
      {/if}
    </div>
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
        <label for="title" class="text-meta" style="color: var(--text-primary)">标题</label>
        <Input id="title" placeholder="公告标题" bind:value={title} autocomplete="off" />
      </div>
      <div class="space-y-2">
        <label for="content" class="text-meta" style="color: var(--text-primary)">内容</label>
        <textarea
          id="content"
          placeholder="公告内容"
          rows="5"
          bind:value={content}
          class="text-body field-inset w-full rounded-[var(--radius-md)] px-4 py-2.5 outline-none resize-none"
          style="background: var(--surface-bg); color: var(--text-primary)"
        ></textarea>
      </div>
      <div class="space-y-2">
        <span class="text-meta" style="color: var(--text-primary)">级别</span>
        <div class="flex gap-2">
          {#each [{ key: 'info', label: '通知', color: 'var(--brand)' }, { key: 'warning', label: '警告', color: 'var(--theme-warning)' }, { key: 'error', label: '紧急', color: 'var(--theme-error)' }] as opt}
            <Button
              type="button"
              variant={severity === opt.key ? 'pressed' : 'flat'}
              size="sm"
              onclick={() => (severity = opt.key as AnnouncementSeverity)}
            >
              {opt.label}
            </Button>
          {/each}
        </div>
      </div>
    </div>
    <Dialog.Footer>
      <Button variant="flat" onclick={closeOverlayDialogs}>取消</Button>
      <Button variant="flat" onclick={handleSubmit} disabled={isSubmitting}>
        {#if isSubmitting}
          <Spinner size="sm" strokeWidth={1.75} class="mr-2" />
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
