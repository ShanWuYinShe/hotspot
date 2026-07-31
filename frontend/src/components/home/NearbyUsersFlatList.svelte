<script lang="ts">
	import type { Snippet } from 'svelte';
	import OnlineDot from '@/components/shared/OnlineDot.svelte';
	import type { NearbyUser } from '@/components/home/PeopleRadarStrip.svelte';

	// 紧凑附近用户行列表（迁移自 React NearbyFeedPage.tsx 内 NearbyUsersFlatList/NearbyUserRow）。

	let {
		users,
		followingIds,
		onToggle
	}: {
		users: NearbyUser[];
		followingIds: Set<number>;
		onToggle: (id: number) => void;
	} = $props();
</script>

{#if users.length > 0}
	<div class="flex flex-col bg-[var(--neu-border)]" style="gap:1px;border-radius:16px;box-shadow:0 0 0 1px var(--neu-border);overflow:hidden;">
		<div style="overflow:hidden;border-radius:16px;">
			{#each users as u (u.id)}
				{@const followed = followingIds.has(u.id)}
				<div class="neu-hover-bg flex items-center gap-3 px-4 py-3 cursor-pointer" style="background:var(--neu-bg);">
					<div class="relative flex-shrink-0">
						<div class="rounded-full overflow-hidden" style="width:38px;height:38px;">
							<img src={u.avatar} alt={u.name} class="w-full h-full object-cover" />
						</div>
						{#if u.status === '在线'}
							<OnlineDot size="md" />
						{/if}
					</div>
					<div class="flex-1 min-w-0">
						<p class="text-[13px]" style="color:var(--neu-text);">{u.name}</p>
						<div class="flex items-center gap-1.5 mt-0.5">
							<span class="text-[10px]" style="color:var(--neu-accent-primary);">#{u.tag}</span>
							<span class="text-[10px]" style="color:var(--neu-text-muted);opacity:0.5;">·</span>
							<span class="text-[10px]" style="color:var(--neu-text-muted);">{u.distance}</span>
						</div>
					</div>
					<button
						type="button"
						class="neu-hover-btn px-3 py-1 rounded-full outline-none transition-all text-[11px]"
						style={followed
							? 'background:transparent;color:var(--neu-text-muted);border:1px solid var(--neu-border-pressed);'
							: 'background:var(--neu-accent-primary);color:white;border:1px solid transparent;'}
						onclick={(e) => {
							e.stopPropagation();
							onToggle(u.id);
						}}
					>
						{followed ? '已关注' : '关注'}
					</button>
				</div>
			{/each}
		</div>
	</div>
{/if}
