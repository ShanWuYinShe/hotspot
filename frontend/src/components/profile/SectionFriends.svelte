<script lang="ts">
	import { goto } from '$app/navigation';
	import { MessageCircle, Users } from '@lucide/svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import ProfileUserSection from '@/components/profile/ProfileUserSection.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import { useRelationships, type MappedFriend } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 好友分区：ProfileUserSection 骨架 + UserCardRow 行。

	let { embedded = false, searchQ = '' }: { embedded?: boolean; searchQ?: string } = $props();
	let activeFilter = $state<'all' | 'online' | 'offline'>('all');

	// getter 对象解构会拿到非响应式快照，$derived 桥接
	const rel = useRelationships();
	const friends = $derived(rel.friends);
	const loading = $derived(rel.loading);

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
	filters={FILTERS}
	filterValue={activeFilter}
	onFilter={(id) => (activeFilter = id as 'all' | 'online' | 'offline')}
	emptyTitle="还没有好友"
	emptyDescription="去附近的人里认识新朋友。"
	emptyIcon={Users}
	loading={loading}
	error={rel.error}
	onRetry={rel.retry}
	needsAuth={rel.needsAuth}
	items={filtered}
>
	{#snippet children(f: MappedFriend)}
		<UserCardRow
			name={f.name}
			avatar={f.avatar}
			tone="row"
			onClick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
			ariaLabel={`查看 ${f.name} 的主页`}
		>
			{#snippet avatarBadge()}
				<OnlineDot size="md" color={f.online ? undefined : 'var(--cst-text-secondary)'} class={f.online ? '' : 'opacity-muted'} />
			{/snippet}
			{#snippet subtitle()}
				<p class="mt-0.5 text-text-secondary text-xs">{f.handle}</p>
			{/snippet}
			{#snippet actions()}
				<!-- 只留「发消息」：整行点击已跳该用户主页（UserCardRow onClick），
				     原先并存的「查看主页」按钮是同一行内跳同一目标的重复入口；
				     关注/粉丝两个分区同样只保留一个行内动作，三 tab 模式一致。 -->
				<button
					type="button"
					class="hover-btn tap-92 hit-slop flex shrink-0 items-center justify-center gap-1 rounded-full bg-surface-subtle/60 px-2.5 py-1.5 text-xs text-text-primary focus-ring hover:bg-surface-subtle"
					onclick={(e) => {
						e.stopPropagation();
						void goto(ROUTES.message(f.id));
					}}
				>
					<MessageCircle class="size-3"  strokeWidth={2} />
					发消息
				</button>
			{/snippet}
		</UserCardRow>
	{/snippet}
</ProfileUserSection>
