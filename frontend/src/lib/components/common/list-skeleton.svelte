<script lang="ts">
  interface Props {
    type: 'user' | 'group' | 'moment' | 'chat' | 'notification';
    count?: number;
    class?: string;
    layout?: 'grid' | 'list';
  }

  let { type, count = 6, class: className = '', layout = 'grid' }: Props = $props();

  const containerClass = $derived(
    layout === 'grid' ? 'grid gap-4 sm:grid-cols-2 lg:grid-cols-3' : 'space-y-3'
  );
</script>

<div class="{containerClass} {className}">
  {#each Array(count) as _, i (i)}
    <div
      class="rounded-[var(--radius-lg)] p-5"
      style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
    >
      {#if type === 'user'}
        <div class="flex items-center gap-3.5">
          <div
            class="rounded-full"
            style="width: 44px; height: 44px; background: var(--border-theme);"
          ></div>
          <div class="flex-1 space-y-2">
            <div
              class="rounded"
              style="height: 12px; width: 80px; background: var(--border-theme);"
            ></div>
            <div
              class="rounded"
              style="height: 10px; width: 60px; background: var(--border-theme);"
            ></div>
          </div>
        </div>
        <div
          class="mt-3 rounded"
          style="height: 10px; width: 100%; background: var(--border-theme);"
        ></div>
        <div
          class="mt-1.5 rounded"
          style="height: 10px; width: 75%; background: var(--border-theme);"
        ></div>
      {:else if type === 'group'}
        <div class="flex items-center gap-3">
          <div
            class="rounded-[var(--radius-sm)]"
            style="width: 44px; height: 44px; background: var(--border-theme);"
          ></div>
          <div class="flex-1 space-y-2">
            <div
              class="rounded"
              style="height: 12px; width: 80px; background: var(--border-theme);"
            ></div>
            <div
              class="rounded"
              style="height: 10px; width: 100%; background: var(--border-theme);"
            ></div>
          </div>
        </div>
        <div
          class="mt-3 rounded"
          style="height: 10px; width: 100%; background: var(--border-theme);"
        ></div>
        <div
          class="mt-1.5 rounded"
          style="height: 10px; width: 65%; background: var(--border-theme);"
        ></div>
      {:else if type === 'moment'}
        <div class="space-y-3">
          <div class="flex items-center gap-3">
            <div
              class="rounded-full"
              style="width: 36px; height: 36px; background: var(--border-theme);"
            ></div>
            <div class="space-y-1.5">
              <div
                class="rounded"
                style="height: 12px; width: 80px; background: var(--border-theme);"
              ></div>
              <div
                class="rounded"
                style="height: 10px; width: 60px; background: var(--border-theme);"
              ></div>
            </div>
          </div>
          <div
            class="rounded-[var(--radius-md)]"
            style="height: 120px; width: 100%; background: var(--border-theme);"
          ></div>
          <div
            class="rounded"
            style="height: 10px; width: 100%; background: var(--border-theme);"
          ></div>
          <div
            class="rounded"
            style="height: 10px; width: 65%; background: var(--border-theme);"
          ></div>
        </div>
      {:else if type === 'chat'}
        <div class="flex items-center gap-3">
          <div
            class="rounded-full"
            style="width: 44px; height: 44px; background: var(--border-theme);"
          ></div>
          <div class="flex-1 space-y-2">
            <div class="flex items-center justify-between">
              <div
                class="rounded"
                style="height: 12px; width: 80px; background: var(--border-theme);"
              ></div>
              <div
                class="rounded"
                style="height: 10px; width: 40px; background: var(--border-theme);"
              ></div>
            </div>
            <div
              class="rounded"
              style="height: 10px; width: 100%; background: var(--border-theme);"
            ></div>
          </div>
        </div>
      {:else if type === 'notification'}
        <div class="flex gap-3">
          <div
            class="rounded-[var(--radius-sm)]"
            style="width: 36px; height: 36px; background: var(--border-theme);"
          ></div>
          <div class="flex-1 space-y-2">
            <div
              class="rounded"
              style="height: 12px; width: 100px; background: var(--border-theme);"
            ></div>
            <div
              class="rounded"
              style="height: 10px; width: 100%; background: var(--border-theme);"
            ></div>
            <div
              class="rounded"
              style="height: 10px; width: 60px; background: var(--border-theme);"
            ></div>
          </div>
        </div>
      {/if}
    </div>
  {/each}
</div>
