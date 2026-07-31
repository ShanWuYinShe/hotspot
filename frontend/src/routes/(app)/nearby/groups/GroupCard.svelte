<script lang="ts">
	import { CalendarDays, Globe, Lock, MapPin, Users } from '@lucide/svelte';
	import JoinButton from './JoinButton.svelte';
	import type { Group } from './group-data';

	let { group, onToggle, onOpen }: { group: Group; onToggle: () => void; onOpen: () => void } = $props();
</script>

<article
	class="neu-hover-card flex h-full flex-col rounded-[18px] overflow-hidden cursor-pointer neu-surface-flat neu-card-hover neu-focus-ring"
	style="background:var(--neu-bg);"
	role="button"
	tabindex="0"
	onclick={onOpen}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onOpen();
		}
	}}
>
	<div class="relative min-h-[96px] flex-1 overflow-hidden">
		<img src={group.banner} alt={group.name} class="w-full h-full object-cover" />
		<div class="absolute inset-0" style="background:linear-gradient(to top, rgba(0,0,0,.68), transparent 60%);"></div>
		<span class="absolute top-2 left-2 px-1.5 py-0.5 rounded-full text-white bg-[var(--neu-accent-primary)] text-[9px]">{group.category}</span>
		<span class="absolute top-2 right-2 flex items-center justify-center rounded-full" style="width:20px;height:20px;background:var(--overlay-strong);">
			{#if group.privacy === 'private'}
				<Lock size={10} color="white" />
			{:else}
				<Globe size={10} color="white" />
			{/if}
		</span>
		<p class="absolute bottom-2.5 left-3 right-3 line-clamp-2 text-white text-[13px] font-semibold">{group.name}</p>
	</div>
	<div class="p-3">
		<div class="flex items-center justify-between gap-2 mb-2">
			<span class="flex items-center gap-1 truncate text-[var(--neu-text-muted)] text-[10px]">
				<Users size={10} />
				{group.members} 人
			</span>
			<span class="flex items-center gap-1 flex-shrink-0 text-[var(--neu-accent-primary)] text-[10px]">
				<MapPin size={10} />
				{group.distance}
			</span>
		</div>
		<p class="truncate flex items-center gap-1 mb-3 text-[var(--semantic-online)] text-[10px]">
			<CalendarDays size={10} />
			{group.activity}
		</p>
		<div class="flex items-center justify-between gap-2">
			<div class="flex -space-x-2">
				{#each group.avatars as avatar, index}
					<img src={avatar} alt="群成员" class="w-5 h-5 rounded-full object-cover" style="border:1px solid var(--neu-bg);z-index:{3 - index};" />
				{/each}
			</div>
			<JoinButton {group} {onToggle} />
		</div>
	</div>
</article>
