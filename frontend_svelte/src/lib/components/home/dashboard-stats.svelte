<script lang="ts">
  import { dev } from '$app/environment';
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import type { MyProfile } from '$lib/types/api';
  import Heart from 'lucide-svelte/icons/heart';
  import Users from 'lucide-svelte/icons/users';
  import TrendingUp from 'lucide-svelte/icons/trending-up';

  interface Props {
    refreshTrigger?: number;
  }

  let { refreshTrigger = 0 }: Props = $props();

  let stats = $state<MyProfile['stats'] | null>(null);
  let loading = $state(false);
  let loadVersion = 0;

  async function loadStats() {
    const version = ++loadVersion;
    loading = true;
    try {
      const response = await usersService.getMyProfile();
      if (version !== loadVersion) return;
      if (isSuccess(response) && response.data) {
        stats = response.data.stats;
      }
    } catch (error) {
      if (version !== loadVersion) return;
      if (dev) console.error('Failed to load user stats:', error);
    } finally {
      if (version === loadVersion) loading = false;
    }
  }

  $effect(() => {
    refreshTrigger;
    loadStats();
  });
</script>

<div class="grid grid-cols-3 gap-3">
  {#each [{ icon: Heart, label: '动态', value: stats?.moments_count ?? 0, color: 'var(--neu-accent-primary)' }, { icon: Users, label: '关注者', value: stats?.followers_count ?? 0, color: 'var(--neu-warning)' }, { icon: TrendingUp, label: '好友', value: stats?.friends_count ?? 0, color: 'var(--neu-success)' }] as stat}
    <div
      class="flex flex-col items-center justify-center rounded-[18px] py-3.5"
      style="background: var(--neu-bg); box-shadow: 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
    >
      <stat.icon size={16} strokeWidth={1.75} color={stat.color} />
      {#if loading}
        <div
          class="mt-1 rounded"
          style="width: 20px; height: 16px; background: var(--neu-border);"
        ></div>
      {:else}
        <span style="font-size: 20px; color: var(--neu-text); font-weight: 700;">{stat.value}</span>
      {/if}
      <span style="font-size: 10px; color: var(--neu-text-muted); margin-top: 1px;"
        >{stat.label}</span
      >
    </div>
  {/each}
</div>
