<script lang="ts">
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import RailLocationMini from '@/components/rail/RailLocationMini.svelte';
	import RailMomentThumb from '@/components/rail/RailMomentThumb.svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import type { MomentDetail, RailPerson, RailMoment } from './detail-types';

	// MomentDetailRail —— 动态详情页右侧栏（作者/相关位置/此刻附近/更多动态）。
	let {
		moment,
		nearbyNow,
		relatedMoments
	}: {
		moment: MomentDetail;
		nearbyNow: RailPerson[];
		relatedMoments: RailMoment[];
	} = $props();
</script>

<RailSection title="作者">
	<RailUserRow avatar={moment.avatar} name={moment.user} sub={moment.location} onClick={() => { if (moment.authorId) void goto(ROUTES.user(moment.authorId)); }}>
		{#snippet trailing()}
			<button
				type="button"
				class="hover-btn shrink-0 rounded-full border border-accent-border bg-accent-soft px-2.5 py-1 text-xs text-accent-default focus-ring"
				onclick={(e) => {
					e.stopPropagation();
					if (moment.authorId) void goto(ROUTES.message(moment.authorId));
				}}
			>
				私信
			</button>
		{/snippet}
	</RailUserRow>
</RailSection>

<RailSection title="相关位置">
	<RailLocationMini name={moment.location} />
</RailSection>

<!-- 此刻附近（moments/fetch_nearby） -->
<RailSection title="此刻附近" action="查看全部" onAction={() => void goto(ROUTES.nearbyPeople)}>
	<div class="flex flex-col gap-3">
		{#each nearbyNow as u (u.id)}
			<RailUserRow avatar={u.avatar} name={u.name} sub={u.sub} onClick={() => { if (u.id) void goto(ROUTES.user(u.id)); }} />
		{/each}
	</div>
</RailSection>

<!-- 更多动态（moments/search） -->
<RailSection title="更多动态" action="发现" onAction={() => void goto(ROUTES.nearbyMoments)}>
	<div class="flex flex-col gap-3">
		{#each relatedMoments as p (p.id)}
			<RailMomentThumb image={p.image} title={p.title} meta={p.meta} onClick={() => void goto(ROUTES.moment(p.id))} />
		{/each}
	</div>
</RailSection>
