<script lang="ts">
	import { Sparkles, Heart } from '@lucide/svelte';

	// 今日精选卡（对齐 Figma Make DailyPickCard）：全图叠加式——
	// aspect-square 全图 + media-scrim-rail 遮罩 + card-menu 混色底条 + image-hover-frame。
</script>

<button
	type="button"
	class="image-hover-frame neu-hover-card neu-card-hover relative aspect-square w-full overflow-hidden rounded-relaxed text-left outline-none"
>
	<img
		src="https://images.unsplash.com/photo-1502082553048-f009c37129b9?w=600&q=80"
		alt="雨后的旧书店"
		class="image-hover-frame__image absolute inset-0 h-full w-full object-cover"
	/>
	<!-- 图片渐变：与底条自然过渡 -->
	<div class="pointer-events-none absolute inset-0" style="background:var(--surface-media-scrim-rail);"></div>

	<div class="absolute left-2 top-2 flex items-center gap-1 rounded-full px-2 py-0.5 backdrop-blur-sm" style="background:var(--overlay-strong);">
		<Sparkles style="width:9px;height:9px;color:white;" strokeWidth={2} />
		<span style="font-size:9px;color:white;letter-spacing:0.05em;">今日精选</span>
	</div>

	<div class="absolute inset-x-0 bottom-0 border-t p-3" style="background:var(--surface-card-menu);border-color:var(--border-card-menu);">
		<p class="line-clamp-2" style="font-size:12px;color:var(--neu-text);line-height:1.4;">
			雨后的旧书店，光线像被泡软了
		</p>
		<div class="flex items-center gap-2 mt-1.5" style="font-size:10px;color:var(--neu-text-muted);">
			<Heart style="width:11px;height:11px;" strokeWidth={1.5} />
			1.2k
			<span>·</span>
			<span>来自 @鹿屿</span>
		</div>
	</div>
</button>
