<script lang="ts">
  import { MapPin } from 'lucide-svelte';
</script>

<button class="neu-surface-flat rounded-[18px] p-4 text-left w-full overflow-hidden relative">
  <div class="flex items-center justify-between mb-2.5">
    <span
      style="font-size: 10px; color: var(--neu-text-muted); text-transform: uppercase; letter-spacing: 0.08em;"
    >
      此刻附近
    </span>
    <div class="flex items-center gap-1.5">
      <span class="pulse-dot"></span>
      <span style="font-size: 10px; color: var(--neu-text-muted);">实时</span>
    </div>
  </div>

  <div class="flex items-center gap-2 mb-2">
    <MapPin size={13} color="var(--neu-accent-primary)" strokeWidth={1.75} />
    <span style="font-size: 13px; color: var(--neu-text);">云龙湖周边</span>
  </div>

  <div class="flex items-baseline gap-1.5">
    <span style="font-size: 26px; color: var(--neu-text); font-weight: 600; line-height: 1;"
      >142</span
    >
    <span style="font-size: 11px; color: var(--neu-text-muted);">人在线 · 比昨天 +18</span>
  </div>

  <!-- sparkline -->
  <svg
    width="100%"
    height="28"
    viewBox="0 0 200 28"
    preserveAspectRatio="none"
    style="margin-top: 8px; opacity: 0.55;"
  >
    <path
      d="M0,20 L20,16 L40,18 L60,12 L80,14 L100,8 L120,11 L140,6 L160,9 L180,4 L200,7"
      fill="none"
      stroke="var(--neu-accent-primary)"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</button>

<style>
  .pulse-dot {
    display: inline-block;
    width: 6px;
    height: 6px;
    border-radius: 999px;
    background: #22c55e;
    box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.18);
    animation: pulse-glow 2s ease-in-out infinite;
  }

  @keyframes pulse-glow {
    0%,
    100% {
      box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.18);
    }
    50% {
      box-shadow: 0 0 0 6px rgba(34, 197, 94, 0.08);
    }
  }
</style>
