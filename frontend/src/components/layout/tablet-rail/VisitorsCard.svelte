<script lang="ts">
	import { Eye, ChevronRight } from '@lucide/svelte';

	// 最近访客卡（迁移自 React TabletRail.tsx，静态 mock 内容）。

	const VISITORS = [
		{ name: '鹿鸣', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80', min: 4 },
		{ name: 'Mika', avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80', min: 12 },
		{ name: '南枝', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80', min: 23 },
		{ name: '野原', avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80', min: 41 },
		{ name: '阿川', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80', min: 68 }
	];
</script>

<button type="button" class="neu-hover-card neu-surface-flat rounded-[18px] p-4 text-left outline-none w-full">
	<div class="flex items-center justify-between mb-3">
		<span class="neu-eyebrow">今天看过你</span>
		<Eye style="width:14px;height:14px;color:var(--neu-text-muted);" strokeWidth={1.75} />
	</div>
	<div class="flex items-center mb-2.5">
		<div class="flex">
			{#each VISITORS.slice(0, 4) as v, i}
				<div
					class="rounded-full overflow-hidden"
					style="width:30px;height:30px;margin-left:{i === 0 ? 0 : -8}px;border:2px solid var(--neu-bg);z-index:{10 - i};"
				>
					<img src={v.avatar} alt={v.name} class="w-full h-full object-cover" />
				</div>
			{/each}
		</div>
		<div class="ml-auto flex items-baseline gap-1">
			<span style="font-size:22px;color:var(--neu-text);font-weight:600;line-height:1;">18</span>
			<span style="font-size:11px;color:var(--neu-text-muted);">位</span>
		</div>
	</div>
	<div class="flex items-center justify-between">
		<p style="font-size:11px;color:var(--neu-text-muted);">
			{VISITORS[0].name} 刚刚 · 还有 {VISITORS.length - 1} 位
		</p>
		<ChevronRight style="width:12px;height:12px;color:var(--neu-text-muted);" strokeWidth={1.5} />
	</div>
</button>
