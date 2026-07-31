<script lang="ts">
	import { Sparkles, Heart } from '@lucide/svelte';

	// 今日精选卡（迁移自 React TabletRail.tsx，静态 mock 内容）。
</script>

<button
	type="button"
	class="neu-hover-card neu-surface-flat rounded-[18px] overflow-hidden text-left outline-none w-full"
>
	<div class="relative" style="height:110px;background:linear-gradient(135deg, var(--neu-accent-soft) 0%, var(--neu-border) 100%);">
		<img
			src="https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=400&q=80"
			alt="pick"
			class="w-full h-full object-cover"
			style="opacity:0.92;"
		/>
		<div class="absolute top-2 left-2 px-2 py-0.5 rounded-full backdrop-blur-sm flex items-center gap-1" style="background:var(--overlay-strong);">
			<Sparkles style="width:9px;height:9px;color:white;" strokeWidth={2} />
			<span style="font-size:9px;color:white;letter-spacing:0.05em;">今日精选</span>
		</div>
	</div>
	<div class="p-3">
		<p style="font-size:12px;color:var(--neu-text);line-height:1.4;">
			雨后的旧书店，光线像被泡软了
		</p>
		<div class="flex items-center gap-2 mt-1.5">
			<Heart style="width:11px;height:11px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			<span style="font-size:10px;color:var(--neu-text-muted);">1.2k</span>
			<span style="font-size:10px;color:var(--neu-text-muted);">·</span>
			<span style="font-size:10px;color:var(--neu-text-muted);">来自 @鹿屿</span>
		</div>
	</div>
</button>
