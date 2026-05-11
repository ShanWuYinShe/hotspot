<script lang="ts">
  import { platformStore } from '$lib/stores/platform.svelte';
  import ChatListPanel from '$lib/components/chat/chat-list-panel.svelte';
  import MessageSquareDashed from 'lucide-svelte/icons/message-square-dashed';
  import { page } from '$app/stores';
  import type { Snippet } from 'svelte';

  let { children } = $props<{ children: Snippet }>();

  const isOnChatRoot = $derived($page.url.pathname === '/chat');
</script>

{#if platformStore.isDesktop}
  <!-- Desktop: split-pane (300px left + right content) -->
  <div class="flex h-full">
    <!-- Left pane: chat list -->
    <div
      class="flex-shrink-0 flex flex-col"
      style="width: 300px; border-right: 1px solid var(--neu-border);"
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
              class="flex items-center justify-center rounded-full"
              style="width: 72px; height: 72px; box-shadow: 8px 8px 18px var(--neu-shadow-dark), -8px -8px 18px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
            >
              <MessageSquareDashed
                size={28}
                strokeWidth={1.25}
                color="var(--neu-text-muted)"
                style="opacity: 0.5;"
              />
            </div>
            <p style="font-size: 16px; color: var(--neu-text);">选择一个会话</p>
            <p style="font-size: 13px; color: var(--neu-text-muted);">
              从左侧列表中选择或开始新对话
            </p>
          </div>
        </div>
      {:else}
        {@render children()}
      {/if}
    </div>
  </div>
{:else}
  {@render children()}
{/if}
