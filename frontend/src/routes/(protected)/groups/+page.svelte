<script lang="ts">
  import { onMount } from 'svelte';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { groupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import { formatDateTime } from '$lib/utils/format';
  import { t } from '$lib/i18n';
  import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
  import EmptyState from '$lib/components/common/empty-state.svelte';
  import CreateGroupModal from '$lib/components/common/create-group-modal.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import type { GroupApplicationItem, GroupInviteItem, GroupItem } from '$lib/types/api';
  import Plus from 'lucide-svelte/icons/plus';
  import Users from 'lucide-svelte/icons/users';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import Inbox from 'lucide-svelte/icons/inbox';
  import CheckCircle2 from 'lucide-svelte/icons/check-circle-2';

  let groups = $state.raw<GroupItem[]>([]);
  let invites = $state.raw<GroupInviteItem[]>([]);
  let applications = $state.raw<GroupApplicationItem[]>([]);
  let isLoading = $state(true);
  let invitesLoading = $state(false);
  let applicationsLoading = $state(false);
  let showCreateGroup = $state(false);
  let processingInviteIds = $state<Set<string>>(new Set());
  let processingApplicationIds = $state<Set<string>>(new Set());

  let anyLoading = $derived(isLoading || invitesLoading || applicationsLoading);

  async function loadGroups() {
    isLoading = true;
    try {
      const response = await withMinimumLoadingTime(() =>
        groupsService.getMyGroups({ pagination: { limit: 50 } })
      );
      if (isSuccess(response) && response.data) {
        groups = response.data.items;
      }
    } catch {
      toast.error(t('groupsMine.loadFailed'));
    } finally {
      isLoading = false;
    }
  }

  async function loadInvites() {
    invitesLoading = true;
    try {
      const response = await groupsService.getInvites({
        status: 'pending',
        pagination: { limit: 50 }
      });
      if (isSuccess(response) && response.data) {
        invites = response.data.items;
      }
    } catch {
      toast.error(t('groupsMine.invitesLoadFailed'));
    } finally {
      invitesLoading = false;
    }
  }

  async function loadApplications() {
    applicationsLoading = true;
    try {
      const response = await groupsService.getMyApplications({
        status: 'pending',
        pagination: { limit: 50 }
      });
      if (isSuccess(response) && response.data) {
        applications = response.data.items;
      }
    } catch {
      toast.error(t('groupsMine.applicationsLoadFailed'));
    } finally {
      applicationsLoading = false;
    }
  }

  // 页面级单一刷新入口：一并刷新「已加入群组 / 收到的邀请 / 我的申请」，
  // 取代此前三个分散的刷新按钮（头部 + 邀请卡 + 申请卡）。
  async function refreshAll() {
    await Promise.all([loadGroups(), loadInvites(), loadApplications()]);
  }

  function setInviteProcessing(inviteId: string, processing: boolean) {
    const next = new Set(processingInviteIds);
    if (processing) next.add(inviteId);
    else next.delete(inviteId);
    processingInviteIds = next;
  }

  function setApplicationProcessing(applicationId: string, processing: boolean) {
    const next = new Set(processingApplicationIds);
    if (processing) next.add(applicationId);
    else next.delete(applicationId);
    processingApplicationIds = next;
  }

  function isInviteProcessing(inviteId: string) {
    return processingInviteIds.has(inviteId);
  }

  function isApplicationProcessing(applicationId: string) {
    return processingApplicationIds.has(applicationId);
  }

  async function handleInvite(invite: GroupInviteItem, action: 'accept' | 'reject') {
    if (isInviteProcessing(invite.invite_id)) return;
    setInviteProcessing(invite.invite_id, true);
    try {
      const response =
        action === 'accept'
          ? await groupsService.acceptInvite({ invite_id: invite.invite_id })
          : await groupsService.rejectInvite({ invite_id: invite.invite_id });
      if (isSuccess(response)) {
        invites = invites.filter((item) => item.invite_id !== invite.invite_id);
        toast.success(action === 'accept' ? t('groupsMine.inviteAccepted') : t('groupsMine.inviteRejected'));
        if (action === 'accept') {
          await loadGroups();
        }
      } else {
        toast.error(response.message || t('groupsMine.operationFailed'));
      }
    } catch {
      toast.error(t('groupsMine.operationFailed'));
    } finally {
      setInviteProcessing(invite.invite_id, false);
    }
  }

  async function handleCancelApplication(application: GroupApplicationItem) {
    if (isApplicationProcessing(application.application_id)) return;
    setApplicationProcessing(application.application_id, true);
    try {
      const response = await groupsService.cancelApplication({
        application_id: application.application_id
      });
      if (isSuccess(response)) {
        applications = applications.filter(
          (item) => item.application_id !== application.application_id
        );
        toast.success(t('groupsMine.applicationCanceled'));
      } else {
        toast.error(response.message || t('groupsMine.cancelFailed'));
      }
    } catch {
      toast.error(t('groupsMine.cancelFailed'));
    } finally {
      setApplicationProcessing(application.application_id, false);
    }
  }

  onMount(() => {
    loadGroups();
    loadInvites();
    loadApplications();
  });
</script>

<svelte:head>
  <title>{t('groupsMine.title')} - HotSpot</title>
</svelte:head>

<div class="flex flex-col">
  <!-- Sticky header -->
  <div
    class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
    style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
  >
    <div class="page-shell">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="page-title" style="color: var(--text-primary);">{t('groupsMine.title')}</h1>
          <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
            {t('groupsMine.subtitle')}
          </p>
        </div>
        <div class="flex gap-2">
          <button
            class="text-meta flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary)"
            onclick={refreshAll}
            disabled={anyLoading}
          >
            <RefreshCw size={13} class={anyLoading ? 'animate-spin' : ''} />
            {t('groupsMine.refresh')}
          </button>
          <button
            class="text-meta flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
            style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
            onclick={() => (showCreateGroup = true)}
          >
            <Plus size={13} />
            {t('groupsMine.createGroup')}
          </button>
        </div>
      </div>
    </div>
  </div>

  <div class="flex flex-col page-pad">
    <div class="page-shell flex flex-col gap-5">
      {#if isLoading}
        <ListSkeleton type="group" count={6} layout="grid" />
      {:else if groups.length === 0}
        <EmptyState icon={Users} title={t('groupsMine.emptyTitle')} description={t('groupsMine.emptyDesc')}>
          <div class="mt-4 flex gap-3">
            <a
              href="/groups/nearby"
              class="rounded-full px-4 py-2 text-sm"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary); text-decoration: none;"
              >{t('groupsMine.discoverNearby')}</a
            >
            <button
              class="rounded-full px-4 py-2 text-sm"
              style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
              onclick={() => (showCreateGroup = true)}>{t('groupsMine.createGroup')}</button
            >
          </div>
        </EmptyState>
      {:else}
        <div class="density-grid">
          {#each groups as group (group.group.group_id)}
            <a href="/groups/{group.group.group_id}" style="text-decoration: none;">
              <div
                class="flex items-center gap-4 rounded-[var(--radius-md)] p-4"
                style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
              >
                <Avatar
                  src={group.group.avatar_url}
                  name={group.group.name}
                  alt={group.group.name}
                  size={48}
                  type="group"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex items-center justify-between">
                    <p
                      class="text-body-lg truncate"
                      style="font-weight: 500; color: var(--text-primary)"
                    >
                      {group.group.name}
                    </p>
                    {#if (group.my_state?.unread_count ?? 0) > 0}
                      <div
                        class="text-caption flex items-center justify-center rounded-full"
                        style="width: 20px; height: 20px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; box-shadow: var(--shadow-brand-btn)"
                      >
                        {(group.my_state?.unread_count ?? 0) > 99
                          ? t('groupsMine.unreadMax')
                          : (group.my_state?.unread_count ?? 0)}
                      </div>
                    {/if}
                  </div>
                  <p class="text-body" style="color: var(--text-secondary)">
                    {t('groupsMine.membersCount', { count: group.group.members_count })}
                  </p>
                </div>
              </div>
            </a>
          {/each}
        </div>
      {/if}

      <!-- Invites & Applications -->
      <div class="grid gap-4 lg:grid-cols-2">
        <!-- Invites -->
        <div
          class="rounded-[var(--radius-lg)] p-4"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
        >
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <div
                class="flex items-center justify-center rounded-full"
                style="width: 28px; height: 28px; border: 1px solid var(--border-theme);"
              >
                <Inbox size={12} color="var(--text-secondary)" strokeWidth={1.5} />
              </div>
              <span class="text-body-lg" style="font-weight: 600; color: var(--text-primary)"
                >{t('groupsMine.invitesTitle')}</span
              >
            </div>
          </div>
          {#if invitesLoading && invites.length === 0}
            <p class="text-body" style="color: var(--text-secondary)">{t('groupsMine.loading')}</p>
          {:else if invites.length === 0}
            <p class="text-body" style="color: var(--text-secondary)">{t('groupsMine.noInvites')}</p>
          {:else}
            <div class="space-y-2">
              {#each invites as invite (invite.invite_id)}
                <div
                  class="rounded-[var(--radius-md)] p-3"
                  style="border: 1px solid var(--border-theme);"
                >
                  <p class="text-body" style="font-weight: 500; color: var(--text-primary)">
                    {invite.group.name}
                  </p>
                  <p class="text-meta" style="color: var(--text-secondary); margin-top: 4px">
                    {t('groupsMine.inviter', { nickname: invite.inviter.nickname })}
                  </p>
                  {#if invite.message}
                    <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
                      {invite.message}
                    </p>
                  {/if}
                  <div class="mt-2 flex gap-2">
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none"
                      style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
                      disabled={isInviteProcessing(invite.invite_id)}
                      onclick={() => handleInvite(invite, 'accept')}
                    >
                      {t('groupsMine.accept')}
                    </button>
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none"
                      style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
                      disabled={isInviteProcessing(invite.invite_id)}
                      onclick={() => handleInvite(invite, 'reject')}
                    >
                      {t('groupsMine.reject')}
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Applications -->
        <div
          class="rounded-[var(--radius-lg)] p-4"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
        >
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <div
                class="flex items-center justify-center rounded-full"
                style="width: 28px; height: 28px; border: 1px solid var(--border-theme);"
              >
                <CheckCircle2 size={12} color="var(--text-secondary)" strokeWidth={1.5} />
              </div>
              <span class="text-body-lg" style="font-weight: 600; color: var(--text-primary)"
                >{t('groupsMine.applicationsTitle')}</span
              >
            </div>
          </div>
          {#if applicationsLoading && applications.length === 0}
            <p class="text-body" style="color: var(--text-secondary)">{t('groupsMine.loading')}</p>
          {:else if applications.length === 0}
            <p class="text-body" style="color: var(--text-secondary)">{t('groupsMine.noApplications')}</p>
          {:else}
            <div class="space-y-2">
              {#each applications as application (application.application_id)}
                <div
                  class="rounded-[var(--radius-md)] p-3"
                  style="border: 1px solid var(--border-theme);"
                >
                  <p class="text-body" style="font-weight: 500; color: var(--text-primary)">
                    {t('groupsMine.applicationItemTitle')}
                  </p>
                  {#if application.message}
                    <p class="text-meta" style="color: var(--text-secondary); margin-top: 4px">
                      {application.message}
                    </p>
                  {/if}
                  <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
                    {t('groupsMine.applicationTime', { time: formatDateTime(application.created_at) })}
                  </p>
                  <div class="mt-2 flex gap-2">
                    <button
                      class="rounded-full px-3 py-1 text-xs outline-none"
                      style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
                      disabled={isApplicationProcessing(application.application_id)}
                      onclick={() => handleCancelApplication(application)}
                    >
                      {t('groupsMine.cancelApplication')}
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>

<CreateGroupModal bind:open={showCreateGroup} onCreated={loadGroups} />
