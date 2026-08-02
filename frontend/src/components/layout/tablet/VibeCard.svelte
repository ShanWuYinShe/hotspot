<script lang="ts">
	import { Sun, Cloud, Music } from '@lucide/svelte';

	// 今日氛围卡（天气 + 音乐，迁移自旧版 TabletRail，静态 mock 内容）。
</script>

<div class="neu-tap-94 neu-surface-flat rounded-[18px] p-4">
	<div class="flex items-center justify-between mb-3">
		<span class="neu-eyebrow">今日氛围</span>
		<Sun style="width:14px;height:14px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
	</div>
	<div class="flex items-center gap-3 mb-3">
		<div class="flex items-center gap-1.5">
			<Cloud style="width:16px;height:16px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			<span style="font-size:13px;color:var(--neu-text);">多云 · 21°</span>
		</div>
		<span style="font-size:10px;color:var(--neu-text-muted);">适合散步</span>
	</div>
	<div class="flex items-center gap-2.5 p-2 rounded-[12px]" style="background:var(--neu-border);">
		<div class="flex items-center justify-center rounded-[8px] flex-shrink-0" style="width:32px;height:32px;background:var(--neu-accent-primary);opacity:0.9;">
			<Music style="width:14px;height:14px;color:white;" strokeWidth={2} />
		</div>
		<div class="flex-1 min-w-0">
			<p style="font-size:11px;color:var(--neu-text);line-height:1.2;" class="truncate">
				漫长的告白
			</p>
			<p style="font-size:9px;color:var(--neu-text-muted);margin-top:1px;">陈鸿宇 · 单曲循环</p>
		</div>
	</div>
</div>
