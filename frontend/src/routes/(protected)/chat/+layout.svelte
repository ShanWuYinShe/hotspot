<script lang="ts">
  import { platformStore } from '$lib/stores/platform.svelte';
  import ChatListPanel from '$lib/components/chat/chat-list-panel.svelte';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import { page } from '$app/stores';
  import type { Snippet } from 'svelte';

  let { children } = $props<{ children: Snippet }>();

  const isOnChatRoot = $derived($page.url.pathname === '/chat');
</script>

{#if platformStore.isWide}
  <!-- Wide (desktop + 横屏/大尺寸平板): bounded split-pane master-detail workspace -->
  <div class="page-pad box-border h-full min-h-0">
    <div
      class="page-shell page-shell--wide flex h-full min-h-0 overflow-hidden rounded-[var(--radius-button)]"
      style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
    >
      <!-- Left pane: chat list -->
      <div
        class="flex flex-shrink-0 flex-col"
        style="width: 300px; border-right: 1px solid var(--border-theme);"
      >
        <ChatListPanel />
      </div>

      <!-- Right pane: chat content -->
      <div class="flex flex-1 overflow-hidden">
        {#if isOnChatRoot}
          <!-- Empty state when no conversation selected -->
          <div class="flex flex-1 items-center justify-center overflow-hidden">
            <div class="flex flex-col items-center gap-3">
              <div
                class="surface-flat flex items-center justify-center rounded-full"
                style="width: 72px; height: 72px;"
              >
                <MessageCircle
                  size={28}
                  strokeWidth={1.25}
                  color="var(--text-secondary)"
                  style="opacity: 0.5;"
                />
              </div>
              <p class="text-title" style="color: var(--text-primary)">选择一个会话</p>
              <p class="text-body" style="color: var(--text-secondary)">
                从左侧列表中选择或开始新对话
              </p>
            </div>
          </div>
        {:else}
          {@render children()}
        {/if}
      </div>
    </div>
  </div>
{:else}
  {@render children()}
{/if}
