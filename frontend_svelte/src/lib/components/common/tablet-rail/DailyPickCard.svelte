<script lang="ts">
  import { Sparkles, Heart } from 'lucide-svelte';
</script>

<button class="neu-surface-flat rounded-[18px] overflow-hidden text-left w-full">
  <div
    class="relative"
    style="height: 110px; background: linear-gradient(135deg, var(--neu-accent-soft) 0%, var(--neu-border) 100%);"
  >
    <!-- Placeholder gradient as hero image -->
    <div
      class="w-full h-full"
      style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); opacity: 0.85;"
    ></div>
    <div
      class="absolute top-2 left-2 px-2 py-0.5 rounded-full flex items-center gap-1"
      style="background: rgba(0,0,0,0.32); backdrop-filter: blur(4px);"
    >
      <Sparkles size={9} color="white" strokeWidth={2} />
      <span style="font-size: 9px; color: white; letter-spacing: 0.05em;">今日精选</span>
    </div>
  </div>

  <div class="p-3">
    <p style="font-size: 12px; color: var(--neu-text); line-height: 1.4;">
      雨后的旧书店，光线像被泡软了
    </p>
    <div class="flex items-center gap-2 mt-1.5">
      <Heart size={11} color="var(--neu-text-muted)" strokeWidth={1.5} />
      <span style="font-size: 10px; color: var(--neu-text-muted);">1.2k</span>
      <span style="font-size: 10px; color: var(--neu-text-muted);">·</span>
      <span style="font-size: 10px; color: var(--neu-text-muted);">来自 @lumi</span>
    </div>
  </div>
</button>
