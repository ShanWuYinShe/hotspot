<script lang="ts">
	import { MapPin } from '@lucide/svelte';

	// 此刻附近卡（迁移自旧版 TabletRail，静态 mock 内容）。

	const SPARKLINE = 'M0,20 L20,16 L40,18 L60,12 L80,14 L100,8 L120,11 L140,6 L160,9 L180,4 L200,7';
</script>

<button
	type="button"
	class="neu-hover-card neu-surface-flat rounded-[18px] p-4 text-left outline-none w-full overflow-hidden relative"
>
	<div class="flex items-center justify-between mb-2.5">
		<span class="neu-eyebrow">此刻附近</span>
		<div class="flex items-center gap-1.5">
			<span
				style="width:6px;height:6px;border-radius:999px;background:var(--semantic-online);box-shadow:0 0 0 3px color-mix(in srgb, var(--semantic-online) 18%, transparent);"
			></span>
			<span style="font-size:10px;color:var(--neu-text-muted);">实时</span>
		</div>
	</div>
	<div class="flex items-center gap-2 mb-2">
		<MapPin style="width:13px;height:13px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
		<span style="font-size:13px;color:var(--neu-text);">云龙湖周边</span>
	</div>
	<div class="flex items-baseline gap-1.5">
		<span style="font-size:26px;color:var(--neu-text);font-weight:600;line-height:1;">142</span>
		<span style="font-size:11px;color:var(--neu-text-muted);">人在线 · 比昨天 +18</span>
	</div>
	<!-- sparkline 装饰 -->
	<svg
		width="100%"
		height="28"
		viewBox="0 0 200 28"
		preserveAspectRatio="none"
		style="margin-top:8px;opacity:0.55;"
	>
		<path
			d={SPARKLINE}
			fill="none"
			stroke="var(--neu-accent-primary)"
			stroke-width="1.5"
			stroke-linecap="round"
			stroke-linejoin="round"
		></path>
	</svg>
</button>
