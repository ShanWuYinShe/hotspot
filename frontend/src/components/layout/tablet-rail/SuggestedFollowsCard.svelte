<script lang="ts">
	import { UserPlus } from '@lucide/svelte';

	// 你可能认识卡（迁移自旧版 TabletRail，静态 mock 内容）。

	const SUGGESTED = [
		{ name: '林一', handle: '@linyi.photos', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80', mutual: 8 },
		{ name: '小满', handle: '@xiaoman.life', avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80', mutual: 5 },
		{ name: '阿川', handle: '@achuan.dev', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80', mutual: 3 }
	];
</script>

<div class="neu-surface-flat rounded-[18px] p-4">
	<div class="flex items-center justify-between mb-3">
		<span class="neu-eyebrow">你可能认识</span>
		<UserPlus style="width:14px;height:14px;color:var(--neu-text-muted);" strokeWidth={1.75} />
	</div>
	<div class="flex flex-col gap-2.5">
		{#each SUGGESTED as s}
			<div class="flex items-center gap-2.5">
				<div class="rounded-full overflow-hidden flex-shrink-0" style="width:32px;height:32px;">
					<img src={s.avatar} alt={s.name} class="w-full h-full object-cover" />
				</div>
				<div class="flex-1 min-w-0">
					<p style="font-size:12px;color:var(--neu-text);line-height:1.2;">{s.name}</p>
					<p style="font-size:10px;color:var(--neu-text-muted);margin-top:1px;">
						{s.mutual} 位共同好友
					</p>
				</div>
				<button
					type="button"
					class="neu-hover-btn px-2.5 py-1 rounded-full outline-none"
					style="background:var(--neu-accent-soft);border:1px solid var(--neu-accent-soft-border);color:var(--neu-accent-primary);font-size:10px;font-weight:500;"
				>
					关注
				</button>
			</div>
		{/each}
	</div>
</div>
