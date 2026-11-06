<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, MapPin, UserPlus } from '@lucide/svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import FollowButton from '@/components/ui/FollowButton.svelte';
	import { relationshipsHref } from '@/lib/relationships';
	import { ROUTES } from '@/lib/routes';
	import { TYPE_CONFIG, type NotifyViewItem } from './notify-data';

	let {
		notifies,
		unread,
		meId,
		followedBack,
		onFollowBack
	}: {
		notifies: NotifyViewItem[];
		unread: number;
		meId: string;
		followedBack: Set<string>;
		onFollowBack: (n: NotifyViewItem) => void;
	} = $props();

	const typeStats = $derived([
		{ type: 'reaction', count: notifies.filter((n) => n.type === 'reaction').length },
		{ type: 'follow', count: notifies.filter((n) => n.type === 'follow').length },
		{ type: 'comment', count: notifies.filter((n) => n.type === 'comment').length },
		{ type: 'nearby', count: notifies.filter((n) => n.type === 'nearby').length }
	]);
</script>

<div class="flex flex-col gap-4 py-2">
	<RailSection title="通知概览" variant="island" icon={Bell}>
		<MetricRow
			variant="strip"
			size="sm"
			items={[
				{ value: unread, label: '未读' },
				{ value: notifies.length - unread, label: '已读' },
				{ value: notifies.length, label: '全部' }
			]}
		/>
		<div class="mt-3 flex flex-col">
			{#each typeStats as { type, count }}
				{@const cfg = TYPE_CONFIG[type] ?? TYPE_CONFIG.reaction}
				<div class="flex items-center gap-2 py-1.5">
					<span class="flex size-5 shrink-0 items-center justify-center rounded-full" style="background:{cfg.bg};">
						<cfg.icon class="size-3" style="color:{cfg.color};" strokeWidth={1.75} aria-hidden="true" />
					</span>
					<span class="flex-1 text-xs text-text-secondary">{cfg.label}</span>
					<span class="text-xs font-medium tabular-nums text-text-primary">{count}</span>
				</div>
			{/each}
		</div>
	</RailSection>

	{#if notifies.filter((n) => n.type === 'follow').length > 0}
		<RailSection title="新关注你的人" variant="island" icon={UserPlus} action="全部" onAction={() => void goto(relationshipsHref('fans'))}>
			<div class="flex flex-col gap-3">
				{#each notifies.filter((n) => n.type === 'follow' && n.user.id !== meId).slice(0, 3) as n (n.id)}
					<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.time}>
						{#snippet trailing()}
							<FollowButton
								active={followedBack.has(n.user.id)}
								disabled={followedBack.has(n.user.id)}
								labelActive="已关注"
								labelInactive="回关"
								onToggle={() => onFollowBack(n)}
							/>
						{/snippet}
					</RailUserRow>
				{/each}
			</div>
		</RailSection>
	{/if}

	{#if notifies.filter((n) => n.type === 'nearby').length > 0}
		<RailSection title="此刻附近" variant="island" icon={MapPin} action="查看" onAction={() => void goto(ROUTES.nearbyPeople)}>
			<div class="flex flex-col gap-3">
				{#each notifies.filter((n) => n.type === 'nearby').slice(0, 3) as n (n.id)}
					<RailUserRow avatar={n.user.avatar} name={n.user.name} sub={n.text} />
				{/each}
			</div>
		</RailSection>
	{/if}
</div>
