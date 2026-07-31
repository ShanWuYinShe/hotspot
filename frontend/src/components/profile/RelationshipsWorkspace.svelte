<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { MessageCircle, Heart, UserPlus } from '@lucide/svelte';
	import SectionFriends from '@/components/profile/SectionFriends.svelte';
	import SectionFollowing from '@/components/profile/SectionFollowing.svelte';
	import SectionFans from '@/components/profile/SectionFans.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 关系网络工作区（好友/关注/粉丝三 Tab，?tab= 参数驱动）。
	// 迁移自 React ProfileFeature.tsx 内 RelationshipsWorkspace。

	type RelationshipTab = 'friends' | 'following' | 'followers';

	const { friends, following, followers } = useRelationships();

	const tabParam = $derived(page.url.searchParams.get('tab'));
	const activeTab = $derived<RelationshipTab>(tabParam === 'following' || tabParam === 'followers' ? tabParam : 'friends');

	const RELATIONSHIP_TABS = [
		{ id: 'friends', label: '好友', detail: '可以直接对话的双向关系', count: friends.length, icon: MessageCircle },
		{ id: 'following', label: '关注', detail: '你主动订阅的动态与创作者', count: following.length, icon: Heart },
		{ id: 'followers', label: '粉丝', detail: '关注你的人与待回关关系', count: followers.length, icon: UserPlus }
	] as const;
	const activeMeta = $derived(RELATIONSHIP_TABS.find((tab) => tab.id === activeTab)!);

	const setTab = (tab: RelationshipTab) => {
		void goto(`${ROUTES.meSection('relationships')}?tab=${tab}`);
	};
</script>

<div class="flex h-full min-h-0 w-full flex-1 flex-col">
	<div class="flex-shrink-0 border-b border-[var(--neu-border)] bg-[var(--neu-bg)] px-6 pt-5 lg:px-8">
		<div class="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-6">
			<div>
				<div class="flex items-center gap-2">
					<p class="text-[var(--neu-text)] text-base">关系网络</p>
					<span class="rounded-full px-2 py-0.5 bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] text-[10px]" style="border:1px solid var(--neu-accent-soft-border);">
						{friends.length + following.length + followers.length} 条关系
					</span>
				</div>
				<p class="mt-1 text-[var(--neu-text-muted)] text-xs">{activeMeta.detail}</p>
			</div>
			<span class="mt-0.5 whitespace-nowrap text-[var(--neu-text-muted)] text-xs">{activeMeta.count} 人</span>
		</div>

		<div class="mt-5 flex gap-1" role="tablist" aria-label="关系类型">
			{#each RELATIONSHIP_TABS as { id, label, count, icon: Icon }}
				{@const selected = activeTab === id}
				<button
					type="button"
					role="tab"
					aria-selected={selected}
					class="neu-hover-btn relative flex items-center gap-1.5 px-3 pb-3 pt-1 outline-none text-[13px] {selected ? 'text-[var(--neu-accent-primary)] font-semibold' : 'text-[var(--neu-text-muted)] font-normal'}"
					onclick={() => setTab(id)}
				>
					<Icon style="width:13px;height:13px;" strokeWidth={selected ? 2 : 1.5} />
					{label}
					<span class="text-[10px]" style="opacity:0.75;">{count}</span>
					{#if selected}
						<span class="absolute bottom-0 left-2 right-2 h-0.5 rounded-full bg-[var(--neu-accent-primary)]"></span>
					{/if}
				</button>
			{/each}
		</div>
	</div>

	<div class="min-h-0 w-full flex-1 overflow-hidden">
		{#if activeTab === 'friends'}
			<SectionFriends embedded />
		{:else if activeTab === 'following'}
			<SectionFollowing embedded />
		{:else}
			<SectionFans embedded />
		{/if}
	</div>
</div>
