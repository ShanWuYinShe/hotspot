<script lang="ts">
	import { MapPin, UserCheck, UserPlus } from '@lucide/svelte';
	import TagPill from '@/components/shared/TagPill.svelte';
	import OnlineDot from '@/components/shared/OnlineDot.svelte';

	// 附近用户卡片（迁移自 React NearbyPeoplePage.tsx 内 UserCard）。

	export interface NearbyUser {
		id: number;
		name: string;
		distance: string;
		avatar: string;
		status: string;
		tag: string;
		mutual: number;
		note: string;
	}

	let {
		user,
		followed,
		onFollow,
		onNavigate
	}: {
		user: NearbyUser;
		followed: boolean;
		onFollow: () => void;
		onNavigate: () => void;
	} = $props();
</script>

<article
	class="neu-hover-card flex h-full flex-col rounded-[20px] overflow-hidden cursor-pointer neu-surface-flat neu-card-hover neu-focus-ring"
	style="background:var(--neu-bg);"
	role="button"
	tabindex="0"
	onclick={onNavigate}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onNavigate();
		}
	}}
>
	<div class="relative min-h-[160px] flex-1 overflow-hidden">
		<img src={user.avatar} alt={user.name} class="w-full h-full object-cover" />
		<div class="absolute inset-x-0 bottom-0 h-3/5 bg-gradient-to-t from-black/70 to-transparent"></div>
		<div class="absolute top-2.5 right-2.5 flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-[var(--overlay-strong)]" style="backdrop-filter:blur(8px);">
			<MapPin class="w-[9px] h-[9px] text-white" strokeWidth={1.5} />
			<span class="text-white text-[10px]">{user.distance}</span>
		</div>
		<div class="absolute bottom-0 inset-x-0 flex items-end justify-between gap-2 px-3 pb-2.5">
			<div class="min-w-0">
				<p class="truncate text-white text-sm font-semibold">{user.name}</p>
				<p class="truncate text-white/78 text-[10px] mt-0.5">{user.note}</p>
			</div>
			{#if user.status === '在线'}
				<OnlineDot size="sm" className="relative static" />
			{/if}
		</div>
	</div>
	<div class="px-3 pt-2.5 pb-3">
		<div class="flex items-center justify-between gap-2 mb-2.5">
			<TagPill size="sm" active>#{user.tag}</TagPill>
			<span class="truncate text-[var(--neu-text-muted)] text-[10px]">{user.mutual ? `${user.mutual} 位共同好友` : '等待认识'}</span>
		</div>
		<div class="flex items-center justify-between gap-2">
			<span class="text-[var(--neu-text-muted)] text-[10px]">{followed ? '已关注' : '感兴趣就打个招呼'}</span>
			<button
				type="button"
				class="neu-hover-btn flex items-center justify-center rounded-full outline-none flex-shrink-0 w-[30px] h-[30px]"
				aria-label={followed ? `取消关注 ${user.name}` : `关注 ${user.name}`}
				style={followed
					? 'background:var(--neu-accent-soft);border:1px solid var(--neu-accent-soft-border);color:var(--neu-accent-primary);'
					: 'background:var(--neu-accent-primary);color:white;box-shadow:0 2px 8px var(--neu-glow);'}
				onclick={(e) => {
					e.stopPropagation();
					onFollow();
				}}
			>
				{#if followed}
					<UserCheck size={15} />
				{:else}
					<UserPlus size={15} />
				{/if}
			</button>
		</div>
	</div>
</article>
