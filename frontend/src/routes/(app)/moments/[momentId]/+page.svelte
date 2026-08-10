<script lang="ts">
	import { goto } from '$app/navigation';
	import { MoreHorizontal, Link2, Send } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { auth } from '@/stores/auth.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import RailLocationMini from '@/components/rail/RailLocationMini.svelte';
	import RailMomentThumb from '@/components/rail/RailMomentThumb.svelte';
	import MomentPostBody from './MomentPostBody.svelte';
	import { getMomentDetail, listComments, setReaction, createComment, favoriteMoment, getFeedNearby, searchMoments } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import { formatDistance, formatDateTime } from '@/lib/utils';

	// 动态详情：点赞/收藏/评论/复制链接/更多菜单 + 桌面右 Rail。

	let { params }: { params: { momentId: string } } = $props();

	// 页面私有类型：内容体由 MomentPostBody 渲染（类型与其导出保持一致）。
	type MomentComment = import('./MomentPostBody.svelte').MomentComment;
	type MomentData = import('./MomentPostBody.svelte').MomentData;

	// 桌面 rail 数据：此刻附近来自 feed/nearby，更多动态来自 moments/search（均走 MSW mock）。
	let nearbyNow = $state<{ id: string; name: string; sub: string; avatar: string }[]>([]);
	let relatedMoments = $state<{ id: string; title: string; meta: string; image: string }[]>([]);

	$effect(() => {
		let cancelled = false;
		void Promise.all([
			getFeedNearby({ pagination: { limit: 3 } }).catch(() => ({ code: -1, message: '网络异常', data: null })),
			searchMoments({ keyword: '', pagination: { limit: 3 } }).catch(() => ({ code: -1, message: '网络异常', data: null }))
		]).then(([nearbyRes, relatedRes]) => {
			if (cancelled) return;
			const nearbyItems = nearbyRes.code === 0 ? nearbyRes.data?.items ?? [] : [];
			const relatedItems = relatedRes.code === 0 ? relatedRes.data?.items ?? [] : [];
			nearbyNow = nearbyItems.map((m) => ({
				id: m.user?.user_id ?? m.moment_id,
				name: m.user?.nickname ?? '',
				sub: [formatDistance(m.location?.distance), m.tags?.[0]].filter(Boolean).join(' · '),
				avatar: m.user?.avatar_url ?? ''
			}));
			relatedMoments = relatedItems.map((m) => ({
				id: m.moment_id,
				title: m.description ?? '',
				meta: `${m.user?.nickname ?? ''} · ${m.likes_count ?? 0} 赞`,
				image: m.images?.[0]?.url ?? ''
			}));
		});
		return () => {
			cancelled = true;
		};
	});

	// params 为路由参数（SPA 内可能变化），此处用 $derived 保持响应。
	// spec: moment_id 为 string，客户端不解析 ID 内容，直接透传使用。
	const momentId = $derived(params.momentId);

	function emptyMoment(): MomentData {
		return {
			id: '',
			authorId: '',
			user: '',
			avatar: '',
			location: '',
			image: '',
			content: '',
			time: '',
			likes: 0,
			tag: '',
			comments: []
		};
	}
	let moment = $state<MomentData>(emptyMoment());
	let apiComments = $state<MomentComment[]>([]);
	let loading = $state(true);
	let input = $state('');
	let moreOpen = $state(false);
	let shareStatus = $state<'idle' | 'copied' | 'failed'>('idle');
	// Reaction/favorite state derived from the moment detail (spec: liked state
	// comes from reactions[].reacted_by_me; saved from my_state.is_favorited).
	let liked = $state(false);
	let saved = $state(false);
	let likeEmoji = $state('👍');

	$effect(() => {
		let cancelled = false;
		async function fetchMoment() {
			try {
				loading = true;
				// 路由参数切换（SPA 内 momentId 变化）时清空旧数据，避免新数据到达前残留展示
				moment = emptyMoment();
				apiComments = [];
				const [detailRes, commentsRes] = await Promise.all([
					getMomentDetail(momentId).catch(() => ({ code: -1, message: '网络异常', data: null })),
					listComments(momentId).catch(() => ({ code: -1, message: '网络异常', data: { items: [] } }))
				]);
				if (cancelled) return;

				const detailData = detailRes.code === 0 ? detailRes.data : null;
				const commentsData = commentsRes.code === 0 ? commentsRes.data : null;

				if (detailData) {
					moment = {
						id: momentId,
						authorId: detailData.user?.user_id ?? '',
						user: detailData.user?.nickname ?? '',
							avatar: detailData.user?.avatar_url ?? '',
						location: detailData.location?.location_name ?? '',
						image: detailData.images?.[0]?.url ?? '',
						content: detailData.description ?? '',
						time: formatDateTime(detailData.created_at),
						likes: detailData.likes_count ?? 0,
						tag: detailData.tags?.[0] ?? '',
						comments: []
					};
					// Derive liked from reactions[].reacted_by_me (spec); saved from my_state.
					const mine = (detailData.reactions || []).find((r) => r.reacted_by_me);
					if (mine) {
						liked = true;
						likeEmoji = mine.emoji ?? '👍';
					} else {
						liked = false;
					}
					saved = Boolean(detailData.my_state?.is_favorited);
				}

				if (commentsData) {
					const items: MomentComment[] = (commentsData.items || []).map((c, i) => ({
						id: i + 1,
						user: c.user?.nickname ?? '',
						avatar: c.user?.avatar_url ?? '',
						text: c.content ?? '',
						time: formatDateTime(c.created_at),
						likes: (c.reactions || []).reduce((s, r) => s + (r.count ?? 0), 0)
					}));
					apiComments = items;
				}
			} catch {
				// empty data is normal
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchMoment();
		return () => {
			cancelled = true;
		};
	});

	const commentCount = $derived(apiComments.length);

	const handleComment = async () => {
		const text = input.trim();
		if (!text) return;
		input = '';
		// Optimistic append; createComment confirms server-side.
		const optimistic: MomentComment = {
			id: Date.now(),
			user: auth.user?.nickname ?? '',
			avatar: auth.user?.avatar_url ?? '',
			text,
			time: '刚刚',
			likes: 0
		};
		apiComments = [...apiComments, optimistic];
		try {
			await createComment({ moment_id: momentId, content: text });
		} catch {
			apiComments = apiComments.filter((c) => c.id !== optimistic.id);
		}
	};

	const toggleLike = async () => {
		const next = !liked;
		liked = next;
		try {
			await setReaction({ target_type: 'moment', target_id: momentId, emoji: likeEmoji, active: next });
		} catch {
			liked = !next;
		}
	};

	const toggleSave = async () => {
		const next = !saved;
		saved = next;
		try {
			await favoriteMoment({ moment_id: momentId, active: next });
		} catch {
			saved = !next;
		}
	};

	const copyMomentLink = async () => {
		const momentUrl = `${window.location.origin}${ROUTES.moment(momentId)}`;
		try {
			if (!navigator.clipboard?.writeText) throw new Error('Clipboard API unavailable');
			await navigator.clipboard.writeText(momentUrl);
			shareStatus = 'copied';
		} catch {
			shareStatus = 'failed';
		}
		moreOpen = false;
		window.setTimeout(() => (shareStatus = 'idle'), 1800);
	};

	let moreMenuEl = $state<HTMLElement | null>(null);
</script>
<svelte:head>
	<title>动态详情 · HotSpot</title>
</svelte:head>


<SplitPageLayout contentClassName="animate-fade-in pb-6">
	{#snippet header()}
		<PageHeader title="动态详情" titleSize={16} onBack={() => history.back()}>
			{#snippet trailing()}
				<DropdownMenu.Root bind:open={moreOpen}>
					<div bind:this={moreMenuEl} class="relative">
						<ActionIconButton icon={MoreHorizontal} label="更多操作" onClick={() => (moreOpen = !moreOpen)} />
					</div>
					<DropdownMenu.Content customAnchor={moreMenuEl}>
						<DropdownMenu.Item onSelect={() => void copyMomentLink()}>
							<Link2 class="size-3.5 text-accent-default"   strokeWidth={1.75} />
							复制链接
						</DropdownMenu.Item>
						<DropdownMenu.Item class="text-text-secondary">
							取消
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			{/snippet}
		</PageHeader>

		<p aria-live="polite" class="sr-only">
			{shareStatus === 'copied' ? '动态链接已复制' : shareStatus === 'failed' ? '无法复制动态链接' : ''}
		</p>
	{/snippet}

	<!-- 内容（gutter 由 SplitPageLayout 提供；全幅封面在 MomentPostBody 内负 margin 抵消） -->
	<MomentPostBody
		{moment}
		comments={apiComments}
		{commentCount}
		{liked}
		{saved}
		{shareStatus}
		onToggleLike={() => void toggleLike()}
		onToggleSave={() => void toggleSave()}
		onShare={() => void copyMomentLink()}
	/>

	{#snippet footer()}
		<!-- 评论输入栏固定在主区底部，不随正文滚动 -->
		<div class="shrink-0 flex items-center gap-2.5 px-4 py-3 bg-surface-primary border-t border-border-default" >
			<Avatar class="border border-border-default" src={auth.user?.avatar_url} name={auth.user?.nickname} size={30}  />
			<div class="flex-1 flex items-center px-4 rounded-md input-surface h-9.5" >
				<input
					bind:value={input}
					onkeydown={(e) => {
						if (e.key === 'Enter') void handleComment();
					}}
					placeholder="写下你的评论…"
					class="flex-1 bg-transparent text-text-primary text-sm"
				/>
			</div>
			<button
				type="button"
				class="hover-btn tap-88 flex items-center justify-center rounded-full focus-ring shrink-0 size-9" style="background:{input.trim() ? 'var(--cst-accent-default)' : 'var(--cst-surface-primary)'};box-shadow:{input.trim() ? 'var(--shadow-glow-md)' : 'var(--shadow-raised-sm)'};border:1px solid var(--cst-border-default);transition:all 0.2s var(--cst-ease-standard);"
				onclick={() => void handleComment()}
			>
				<Send class="size-3.5" style="color:{input.trim() ? 'var(--cst-text-on-accent)' : 'var(--cst-text-secondary)'};transform:translateX(1px);" strokeWidth={2} />
			</button>
		</div>
	{/snippet}

	{#snippet rail()}
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

	<!-- 此刻附近（feed/nearby） -->
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
	{/snippet}
</SplitPageLayout>
