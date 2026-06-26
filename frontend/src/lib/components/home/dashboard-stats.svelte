<script lang="ts">
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { useWidget } from '$lib/hooks/use-widget.svelte';
  import type { MyProfile } from '$lib/types/api';
  import Heart from 'lucide-svelte/icons/heart';
  import Users from 'lucide-svelte/icons/users';
  import TrendingUp from 'lucide-svelte/icons/trending-up';

  interface Props {
    refreshTrigger?: number;
  }

  let { refreshTrigger = 0 }: Props = $props();

  const {
    data: stats,
    loading,
    reload
  } = useWidget<MyProfile['stats']>(async () => {
    const response = await usersService.getMyProfile();
    if (!isSuccess(response) || !response.data) throw new Error('Failed to load stats');
    return response.data.stats;
  });

  $effect(() => {
    refreshTrigger;
    reload();
  });
</script>

<div class="grid grid-cols-3 gap-3">
  {#each [{ icon: Heart, label: '动态', value: stats?.moments_count ?? 0, color: 'var(--brand)' }, { icon: Users, label: '关注者', value: stats?.followers_count ?? 0, color: 'var(--warning)' }, { icon: TrendingUp, label: '好友', value: stats?.friends_count ?? 0, color: 'var(--success)' }] as stat}
    <div
      class="flex flex-col items-center justify-center rounded-[var(--radius-md)] py-3.5"
      style="background: var(--surface-bg); box-shadow: 5px 5px 12px var(--shadow-dark), -5px -5px 12px var(--shadow-light); border: 1px solid var(--border-theme);"
    >
      <stat.icon size={16} strokeWidth={1.75} color={stat.color} />
      {#if loading}
        <div
          class="mt-1 rounded"
          style="width: 20px; height: 16px; background: var(--border-theme);"
        ></div>
      {:else}
        <span class="text-h1" style="color: var(--text-primary); font-weight: 700"
          >{stat.value}</span
        >
      {/if}
      <span class="text-xs" style="color: var(--text-secondary); margin-top: 1px">{stat.label}</span
      >
    </div>
  {/each}
</div>
