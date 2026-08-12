<script lang="ts">
	import { goto } from '$app/navigation';
	import { ArrowLeft, Bell, Check, MapPin, MessageCircle, Settings, Users } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { ROUTES } from '@/lib/routes';
	import { useGroupDetail } from '@/lib/group-data.svelte';
	import { presence, watchPresence } from '@/stores/presence.svelte';

	// 群组详情：封面 + 简介/成员/公告三 Tab + 加入按钮 + 桌面侧栏。
	// 状态/加载/WS/操作收敛到 useGroupDetail 工厂；本页只保留 UI + 角色门控。
	// WS 实时：工厂订阅 group 频道，成员/详情/公告变更触发对应刷新；解散由 onDismissed 处理。

	let { params }: { params: { groupId: string } } = $props();

	type GroupTab = 'overview' | 'members' | 'announcements';
	let tab = $state<GroupTab>('overview');

	const gd = useGroupDetail({
		groupId: () => params.groupId,
		onDismissed: () => {
			window.alert('该群组已被解散');
			void goto(ROUTES.nearbyGroups);
		},
		watchPresence,
		isOnline: (id) => presence.isOnline(id)
	});
</script>
<svelte:head>
	<title>群组主页 · HotSpot</title>
</svelte:head>


{#if gd.loadError}
	<LoadError message={gd.loadError} onRetry={() => void gd.loadAll(params.groupId ?? '')} />
{:else}
<StandardPageLayout contentClassName="animate-fade-in pb-6">
	<!-- 全幅封面（负 margin 抵消 ContentColumn gutter） -->
	<div class="relative h-[var(--cst-cover-height)] overflow-hidden -mx-[var(--cst-page-gutter)]">
		<img src={gd.detail.banner} alt={`${gd.detail.name} 的群组封面`} class="h-full w-full object-cover" />
		<!-- gradient scrim 走 inline background（gradient 不能作 background-color 值，
		     与 NearbyImageCard 的 --cst-surface-media-scrim-card 用法一致） -->
		<div class="absolute inset-0" style="background:var(--cst-surface-cover-scrim);"></div>
		<div class="absolute left-5 top-4">
			<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} variant="overlay" tone="inverse" />
		</div>
		<div class="absolute bottom-5 left-6 right-6">
			<span class="rounded-full bg-surface-glass-badge px-2.5 py-1 text-xs text-content-inverse backdrop-blur-sm">{gd.detail.category}</span>
			<h1 class="mt-3 text-2xl md:text-3xl font-medium tracking-title text-content-inverse">{gd.detail.name}</h1>
		</div>
	</div>

	<div class="grid gap-8 pt-6 lg:grid-cols-[minmax(0,1fr)_80]">
		<main class="min-w-0">
			<div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-text-secondary">
				<span class="flex items-center gap-1.5">
					<Users size={14} />
					{gd.detail.members} 位成员
				</span>
				<span class="flex items-center gap-1.5 text-status-online">
					<span class="h-2 w-2 rounded-full bg-status-online"></span>
					{gd.onlineMemberCount} 人在线
				</span>
				<span class="flex items-center gap-1.5">
					<MapPin size={14} />
					距你 {gd.detail.distance}
				</span>
			</div>

			<!-- 分类 Tab（统一 TabBar 下划线样式） -->
			<div class="mt-4 border-b border-border-default">
				<TabBar
					items={[
						{ id: 'overview', label: '简介' },
						{ id: 'members', label: '成员' },
						{ id: 'announcements', label: '公告' }
					]}
					value={tab}
					onChange={(id) => (tab = id as GroupTab)}
				/>
			</div>

			{#if tab === 'overview'}
				<div class="py-6">
					<p class="max-w-3xl text-sm leading-7 text-text-secondary">{gd.detail.description}</p>
				</div>
			{:else if tab === 'members'}
				<!-- 空旷内容区 → List bordered（有边框列表契约）；列表行不设独立边框 -->
				<ListContainer bordered class="mt-6">
					{#each gd.members as member (member.id)}
						<ListItem
							onClick={() => {
								if (member.userId) void goto(ROUTES.user(member.userId));
							}}
							ariaLabel={`查看 ${member.name} 的资料`}
							class="app-list-row app-list-row--standard"
						>
							<div class="relative">
								<img src={member.avatar} alt="" loading="lazy" class="h-10 w-10 rounded-full object-cover" />
								{#if gd.onlineMemberIds.has(member.userId)}
									<span class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-surface-primary bg-status-online"></span>
								{/if}
							</div>
							<span class="min-w-0 flex-1">
								<span class="block text-sm text-text-primary">{member.name}</span>
								<span class="text-xs text-text-secondary">{member.role}</span>
							</span>
							<span class="text-xs text-text-secondary">查看资料</span>
						</ListItem>
					{/each}
				</ListContainer>
			{:else}
				<!-- 空旷内容区 → List bordered（有边框列表契约） -->
				<ListContainer bordered class="mt-6">
					{#each gd.announcements as item (item.title)}
						<article class="px-4 py-4">
							<p class="text-sm font-medium text-text-primary">{item.title}</p>
							<p class="mt-1 text-sm leading-5 text-text-secondary">{item.body}</p>
							<p class="mt-2 text-xs text-text-secondary">{item.time}</p>
						</article>
					{/each}
				</ListContainer>
			{/if}
		</main>

		<aside class="flex h-fit flex-col gap-3 rounded-lg ring-1 ring-border-default bg-surface-primary p-4">
			<button
				type="button"
				class="hover-btn w-full rounded-full px-4 py-3 text-sm font-medium focus-ring {gd.joined ? 'border border-accent-default text-accent-default' : 'bg-accent-default text-text-on-accent'}"
				onclick={gd.toggleJoin}
			>
				{#if gd.joined}
					<span class="flex items-center justify-center gap-1.5">
						<Check size={15} />
						已加入 · 点击退出
					</span>
				{:else}
					加入群组
				{/if}
			</button>
			{#if gd.joined && (gd.myRole === 'owner' || gd.myRole === 'admin')}
				<!-- 角色门控：仅群主/管理员可见的管理入口（占位，实际管理 UI 由后续批次填充） -->
				<button
					type="button"
					onclick={() => { /* TODO 群管理批次 N：打开群管理抽屉 */ }}
					class="hover-btn flex w-full items-center justify-center gap-2 rounded-full border border-border-default px-4 py-3 text-sm text-text-primary focus-ring"
				>
					<Settings size={15} />
					群管理
				</button>
			{/if}
			{#if gd.joined}
				<button type="button" onclick={() => void goto(ROUTES.groupMessage(params.groupId))} class="hover-btn flex w-full items-center justify-center gap-2 rounded-full border border-border-default px-4 py-3 text-sm text-text-primary focus-ring">
					<MessageCircle size={15} />
					打开群聊
				</button>
			{:else}
				<p class="px-2 text-center text-xs leading-5 text-text-secondary">
					<Bell size={13} class="mr-1 inline" />
					加入后可管理群组通知并进入群聊
				</p>
			{/if}
		</aside>
	</div>
</StandardPageLayout>
{/if}
