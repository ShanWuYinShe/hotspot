<script lang="ts">
	import { CalendarDays, Sparkles, Users } from '@lucide/svelte';
	import JoinButton from './JoinButton.svelte';
	import type { Group } from './group-data';

	let { group, onToggle, onOpen }: { group: Group; onToggle: () => void; onOpen: () => void } = $props();
</script>

<div
	class="neu-hover-card neu-surface-hero neu-card-hover rounded-[24px] overflow-hidden cursor-pointer h-full neu-focus-ring"
	role="button"
	tabindex="0"
	onclick={onOpen}
	onkeydown={(e) => {
		if (e.target !== e.currentTarget) return;
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onOpen();
		}
	}}
>
	<div class="relative h-48">
		<img src={group.banner} alt={group.name} class="w-full h-full object-cover" />
		<div class="absolute inset-0" style="background:linear-gradient(to top, rgba(0,0,0,.72), transparent 70%);"></div>
		<div class="absolute top-3 left-3 flex items-center gap-1.5 px-2 py-1 rounded-full" style="background:var(--overlay-strong);backdrop-filter:blur(8px);">
			<Sparkles size={11} color="white" />
			<span class="text-white text-[10px]">本周精选</span>
		</div>
		<div class="absolute bottom-3 left-4 right-4">
			<p class="text-white text-lg font-semibold">{group.name}</p>
			<p class="truncate text-white/80 text-[11px] mt-[3px]">{group.desc}</p>
		</div>
	</div>
	<div class="p-4 flex items-center gap-3">
		<div class="flex-1 min-w-0">
			<div class="flex items-center gap-3 text-[var(--neu-text-muted)] text-[11px]">
				<span class="flex items-center gap-1">
					<Users size={12} />
					{group.members}
				</span>
				<span class="flex items-center gap-1 text-[var(--semantic-online)]">
					<span class="rounded-full" style="width:6px;height:6px;background:var(--semantic-online);"></span>
					{group.online} 在线
				</span>
				<span>{group.distance}</span>
			</div>
			<p class="truncate flex items-center gap-1 mt-2 text-[var(--neu-accent-primary)] text-[11px]">
				<CalendarDays size={12} />
				{group.activity}
			</p>
		</div>
		<JoinButton {group} {onToggle} />
	</div>
</div>
