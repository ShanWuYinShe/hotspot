<script lang="ts">
	import { Trophy } from '@lucide/svelte';

	// 成就进度卡（迁移自旧版 TabletRail，静态 mock 内容）。

	const progress = 0.72;
</script>

<div class="neu-surface-flat rounded-[18px] p-4">
	<div class="flex items-center justify-between mb-2.5">
		<span class="neu-eyebrow">下一枚徽章</span>
		<Trophy style="width:14px;height:14px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
	</div>
	<div class="flex items-center gap-2.5 mb-3">
		<div
			class="flex items-center justify-center rounded-[12px] flex-shrink-0"
			style="width:38px;height:38px;background:var(--neu-accent-soft);border:1px solid var(--neu-accent-soft-border);font-size:20px;"
		>
			🌟
		</div>
		<div class="flex-1 min-w-0">
			<p style="font-size:13px;color:var(--neu-text);line-height:1.3;">城市漫游者</p>
			<p style="font-size:11px;color:var(--neu-text-muted);margin-top:1px;">已探索 27 / 38 个地点</p>
		</div>
	</div>
	<div style="height:5px;border-radius:999px;background:var(--neu-border);overflow:hidden;">
		<div style="height:100%;width:{progress * 100}%;background:var(--neu-accent-primary);border-radius:999px;"></div>
	</div>
	<p style="font-size:10px;color:var(--neu-text-muted);margin-top:6px;">
		再去 11 个新地方就解锁
	</p>
</div>
