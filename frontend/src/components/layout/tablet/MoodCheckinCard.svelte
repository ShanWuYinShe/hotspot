<script lang="ts">
	import { Sparkles } from '@lucide/svelte';

	// 今日心情打卡卡（迁移自旧版 TabletRail，静态 mock 内容）。

	const MOODS = [
		{ emoji: '🌤', label: '舒服' },
		{ emoji: '🌊', label: '平静' },
		{ emoji: '🔥', label: '热血' },
		{ emoji: '🌙', label: '想独处' },
		{ emoji: '🍵', label: '慢一点' }
	];
</script>

<div class="neu-tap-94 neu-surface-flat rounded-[18px] p-4">
	<div class="flex items-center justify-between mb-3">
		<span class="neu-eyebrow">今日心情</span>
		<Sparkles style="width:14px;height:14px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
	</div>
	<p style="font-size:13px;color:var(--neu-text);margin-bottom:12px;line-height:1.5;">
		今天的你是怎样的？
	</p>
	<div class="flex gap-1.5 flex-wrap">
		{#each MOODS as m}
			<button
				type="button"
				class="neu-hover-btn flex items-center gap-1 px-2.5 py-1.5 rounded-full outline-none neu-surface-ghost"
				style="border-color:var(--neu-border);"
			>
				<span style="font-size:14px;">{m.emoji}</span>
				<span style="font-size:11px;color:var(--neu-text-muted);">{m.label}</span>
			</button>
		{/each}
	</div>
</div>
