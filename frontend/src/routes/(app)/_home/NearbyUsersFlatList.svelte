<script lang="ts">
	import BorderedList from '@/components/data-display/BorderedList.svelte';
	import InteractiveListRow from '@/components/navigation/InteractiveListRow.svelte';
	import type { NearbyUser } from '@/routes/(app)/_home/nearby-users';

	// 紧凑附近用户行列表（对齐 Figma Make HomePage NearbyUsersFlatList）。
	// BorderedList 容器 + InteractiveListRow 行（focus ring + 键盘 + aria + tap opacity）。

	let {
		users,
		followingIds,
		onToggle
	}: {
		users: NearbyUser[];
		followingIds: Set<string>;
		onToggle: (id: string) => void;
	} = $props();
</script>

{#if users.length > 0}
	<BorderedList>
		{#each users as u (u.id)}
			{@const followed = followingIds.has(u.id)}
				<InteractiveListRow class="flex items-center gap-3 px-4 py-3 cursor-pointer">
					<div class="relative flex-shrink-0">
						<div class="rounded-full overflow-hidden" style="width:38px;height:38px;">
							<img src={u.avatar} alt={u.name} class="w-full h-full object-cover" />
						</div>
					{#if u.status === '在线'}
						<span
							class="absolute bottom-0 right-0 rounded-full"
							style="width:9px;height:9px;background:var(--semantic-online);border:2px solid var(--neu-bg);"
						></span>
					{/if}
				</div>
				<div class="flex-1 min-w-0">
					<p style="font-size:13px;color:var(--neu-text);">{u.name}</p>
					<div class="flex items-center gap-1.5 mt-0.5">
						<span style="font-size:10px;color:var(--neu-accent-primary);">#{u.tag}</span>
						<span style="font-size:10px;color:var(--neu-text-muted);opacity:0.5;">·</span>
						<span style="font-size:10px;color:var(--neu-text-muted);">{u.distance}</span>
					</div>
				</div>
				<button
					type="button"
					class="neu-hover-btn neu-tap-93 px-3 py-1 rounded-full outline-none transition-all"
					style={followed
						? 'background:transparent;color:var(--neu-text-muted);border:1px solid var(--neu-border-pressed);font-size:11px;'
						: 'background:var(--neu-accent-primary);color:white;border:1px solid transparent;font-size:11px;'}
					onclick={(e) => {
						e.stopPropagation();
						onToggle(u.id);
					}}
				>
					{followed ? '已关注' : '关注'}
				</button>
			</InteractiveListRow>
		{/each}
	</BorderedList>
{/if}
