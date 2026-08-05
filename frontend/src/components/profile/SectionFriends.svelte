<script lang="ts">
	import { goto } from '$app/navigation';
	import { MessageCircle, Users } from '@lucide/svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import ProfileUserSection from '@/components/profile/ProfileUserSection.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import { useRelationships, type MappedFriend } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 好友分区：ProfileUserSection 骨架 + UserCardRow 行。

	let { embedded = false }: { embedded?: boolean } = $props();

	let searchQ = $state('');
	let activeFilter = $state<'all' | 'online' | 'offline'>('all');

	const { friends } = useRelationships();

	const filtered = $derived(
		friends
			.filter((f) => activeFilter === 'all' || (activeFilter === 'online' ? f.online : !f.online))
			.filter((f) => searchQ === '' || f.name.toLowerCase().includes(searchQ.toLowerCase()) || f.handle.includes(searchQ))
	);
	const onlineCount = $derived(friends.filter((f) => f.online).length);

	const FILTERS: SectionFilter[] = [
		{ id: 'all', label: '全部' },
		{ id: 'online', label: '在线' },
		{ id: 'offline', label: '离线' }
	];
</script>

<ProfileUserSection
	title="好友"
	subtitle="双向关系，可直接开启对话"
	count={friends.length}
	countSuffix=" 位好友"
	{embedded}
	search={{ value: searchQ, onChange: (v) => (searchQ = v), placeholder: '搜索好友…' }}
	filters={FILTERS}
	filterValue={activeFilter}
	onFilter={(id) => (activeFilter = id as 'all' | 'online' | 'offline')}
	emptyTitle="还没有好友"
	emptyDescription="去附近的人里认识新朋友。"
	emptyIcon={Users}
	items={filtered}
>
	{#snippet children(f: MappedFriend)}
		<UserCardRow
			name={f.name}
			avatar={f.avatar}
			tone={embedded ? 'card' : 'row'}
			onClick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
			ariaLabel={`查看 ${f.name} 的主页`}
		>
			{#snippet avatarBadge()}
				<span class="absolute bottom-0 right-0 rounded-full {f.online ? 'bg-status-online' : 'bg-text-secondary'}" style="width:10px;height:10px;border:2px solid var(--surface-primary);opacity:{f.online ? 1 : 0.5};"></span>
			{/snippet}
			{#snippet nameBadge()}
				<TagPill size="sm">{f.tag}</TagPill>
			{/snippet}
			{#snippet subtitle()}
				<p class="mt-0.5 text-text-secondary text-xs">
					{f.handle} · {f.mutual} 共同好友
				</p>
			{/snippet}
			{#snippet actions()}
				<div class="flex flex-col gap-1.5 shrink-0">
					<button
						type="button"
						class="hover-btn tap-93 flex items-center justify-center gap-1 rounded-full bg-accent-default px-2.5 py-1.5 text-xs text-white outline-none"
						style="box-shadow:0 3px 8px var(--glow-accent);"
						onclick={(e) => {
							e.stopPropagation();
							void goto(ROUTES.message(f.id));
						}}
					>
						<MessageCircle style="width:11px;height:11px;" strokeWidth={2} />
						发消息
					</button>
					<button
						type="button"
						class="hover-btn tap-93 flex items-center justify-center gap-1 rounded-full bg-surface-primary px-2.5 py-1.5 text-xs text-text-secondary outline-none"
						style="border:1px solid var(--border-default);box-shadow:var(--shadow-neu-xs);"
						onclick={(e) => {
							e.stopPropagation();
							void goto(`${ROUTES.user(f.id)}?from=relationships`);
						}}
					>
						查看主页
					</button>
				</div>
			{/snippet}
		</UserCardRow>
	{/snippet}
</ProfileUserSection>
