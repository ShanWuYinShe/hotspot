<script lang="ts">
	import { ChevronRight } from '@lucide/svelte';
	import SectionHead from '@/routes/(app)/_home/SectionHead.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';

	// 附近的人雷达条——扁平头像横排（迁移自旧版 NearbyFeedPage）。

	export interface NearbyUser {
		id: string;
		name: string;
		distance: string;
		avatar: string;
		status: string;
		tag: string;
	}

	let {
		users,
		onSeeAll,
		onSelectUser
	}: {
		users: NearbyUser[];
		onSeeAll: () => void;
		onSelectUser: (userId: string) => void;
	} = $props();
</script>

<div>
	<SectionHead title="附近的人 · 实时" action="查看全部" onAction={onSeeAll} />
	<div class="flex gap-4 overflow-x-auto no-scrollbar pb-1">
		{#each users as u (u.id)}
			<button
				type="button"
				class="neu-hover-btn flex flex-col items-center gap-2 flex-shrink-0 rounded-[14px] neu-focus-ring"
				aria-label={`查看${u.name}的个人主页`}
				onclick={() => onSelectUser(u.id)}
			>
				<div class="relative">
					<div class="rounded-full overflow-hidden" style="width:56px;height:56px;border:1.5px solid var(--neu-border);">
						<img src={u.avatar} alt="" class="w-full h-full object-cover" />
					</div>
					{#if u.status === '在线'}
						<OnlineDot size="lg" />
					{/if}
				</div>
				<span class="text-[11px]" style="color:var(--neu-text);">{u.name}</span>
				<span class="text-[10px]" style="color:var(--neu-accent-primary);margin-top:-4px;">{u.distance}</span>
			</button>
		{/each}
		<button
			type="button"
			class="neu-hover-btn flex flex-col items-center gap-2 flex-shrink-0 rounded-[14px] neu-focus-ring"
			onclick={onSeeAll}
		>
			<div class="flex items-center justify-center rounded-full" style="width:56px;height:56px;border:1.5px dashed var(--neu-border-pressed);">
				<ChevronRight style="width:16px;height:16px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</div>
			<span class="text-[11px]" style="color:var(--neu-text-muted);">更多</span>
		</button>
	</div>
</div>
