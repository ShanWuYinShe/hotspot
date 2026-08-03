<script lang="ts">
	import { goto } from '$app/navigation';
	import { MoreHorizontal, Link2 } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { auth } from '@/stores/auth.svelte';
	import DesktopDetailSplit from '@/components/layout/desktop/DesktopDetailSplit.svelte';
	import RailSection from '@/components/layout/desktop/RailSection.svelte';
	import RailUserRow from '@/components/layout/desktop/RailUserRow.svelte';
	import RailLocationMini from '@/components/layout/desktop/RailLocationMini.svelte';
	import RailMomentThumb from '@/components/layout/desktop/RailMomentThumb.svelte';
	import MomentPostBody from './MomentPostBody.svelte';
	import { getMomentDetail, listComments, setReaction, createComment, favoriteMoment } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 动态详情：点赞/收藏/评论/复制链接/更多菜单 + 桌面右 Rail。
	// 迁移自旧版 MomentDetailPage。

	let { params }: { params: { momentId: string } } = $props();

	// 页面私有类型：内容体由 MomentPostBody 渲染（类型与其导出保持一致）。
	type MomentComment = import('./MomentPostBody.svelte').MomentComment;
	type MomentData = import('./MomentPostBody.svelte').MomentData;

	// 桌面 rail 占位数据（对齐 Figma PostDetailPage NEARBY_NOW/RELATED_MOMENTS；
	// 后端无专用端点，先用占位引导，后续接入 feed/nearby 与相关推荐）。
	const NEARBY_NOW = [
		{ id: 1, name: '鹿鸣', sub: '246m · 摄影', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80' },
		{ id: 2, name: '阿禾', sub: '388m · 旅行', avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80' },
		{ id: 3, name: '野原', sub: '520m · 音乐', avatar: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80' }
	];
	const RELATED_MOMENTS = [
		{ id: 40, title: '植物园晨光', meta: '南枝 · 124 赞', image: 'https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=120' },
		{ id: 41, title: '城市夜光', meta: '本地向导 · 89 赞', image: 'https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=120' },
		{ id: 42, title: '老街手冲馆', meta: '野原 · 56 赞', image: 'https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=120' }
	];

	// params 为路由参数（SPA 内可能变化），此处用 $derived 保持响应。
	const numericMomentId = $derived(Number(params.momentId));
	const momentId = $derived(String(numericMomentId));

	function emptyMoment(): MomentData {
		return {
			id: 0,
			user: '',
			handle: '',
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
					getMomentDetail(momentId).catch(() => ({ code: -1, data: null })),
					listComments(momentId).catch(() => ({ code: -1, data: { items: [] } }))
				]);
				if (cancelled) return;

				const detailData = detailRes.code === 0 ? detailRes.data : null;
				const commentsData = commentsRes.code === 0 ? commentsRes.data : null;

				if (detailData) {
					moment = {
						id: numericMomentId,
						user: detailData.user?.nickname ?? '',
						handle: '',
						avatar: detailData.user?.avatar_url ?? '',
						location: detailData.location?.location_name ?? '',
						image: detailData.images?.[0]?.url ?? '',
						content: detailData.description ?? '',
						time: detailData.created_at ?? '',
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
						time: c.created_at ?? '',
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
		const momentUrl = `${window.location.origin}${ROUTES.moment(numericMomentId)}`;
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

{#snippet main()}
	<div class="flex flex-col bg-[var(--neu-bg)]" style="height:100%;">
		<!-- Header -->
		<PageHeader title="动态详情" titleSize={16} onBack={() => history.back()} class="flex-shrink-0 border-b border-[var(--neu-border)]">
			{#snippet trailing()}
				<DropdownMenu.Root bind:open={moreOpen}>
					<div bind:this={moreMenuEl} class="relative">
						<ActionIconButton icon={MoreHorizontal} label="更多操作" onClick={() => (moreOpen = !moreOpen)} />
					</div>
					<DropdownMenu.Content customAnchor={moreMenuEl}>
						<DropdownMenu.Item onSelect={() => void copyMomentLink()}>
							<Link2 style="width:14px;height:14px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
							复制链接
						</DropdownMenu.Item>
						<DropdownMenu.Item class="text-[var(--neu-text-muted)]">
							取消
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			{/snippet}
		</PageHeader>

		<p aria-live="polite" class="sr-only">
			{shareStatus === 'copied' ? '动态链接已复制' : shareStatus === 'failed' ? '无法复制动态链接' : ''}
		</p>

		<!-- Scrollable content -->
		<div class="flex-1 overflow-y-auto no-scrollbar">
			<MomentPostBody
				{moment}
				comments={apiComments}
				{commentCount}
				{liked}
				{saved}
				{shareStatus}
				bind:input
				onToggleLike={() => void toggleLike()}
				onToggleSave={() => void toggleSave()}
				onShare={() => void copyMomentLink()}
				onComment={() => void handleComment()}
			/>
		</div>
	</div>
{/snippet}

{#snippet rail()}
	<RailSection title="作者">
		<RailUserRow avatar={moment.avatar} name={moment.user} sub={`${moment.handle} · ${moment.location}`} onClick={() => void goto(ROUTES.user(moment.id))}>
			{#snippet trailing()}
				<button
					type="button"
					class="neu-hover-btn px-2.5 py-1 rounded-full outline-none flex-shrink-0 bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)]"
					style="font-size:11px;border:1px solid var(--neu-accent-soft-border);"
					onclick={(e) => {
						e.stopPropagation();
						void goto(ROUTES.message(moment.id));
					}}
				>
					私信
				</button>
			{/snippet}
		</RailUserRow>
	</RailSection>

	<RailSection title="相关位置">
		<RailLocationMini name={moment.location} desc="共 12 条动态 · 6 人在此" />
	</RailSection>

	<!-- 此刻附近（对齐 Figma PostDetailPage rail；后端无专用端点，先用占位引导） -->
	<RailSection title="此刻附近" action="查看全部" onAction={() => void goto(ROUTES.nearbyPeople)}>
		<div class="flex flex-col gap-3">
			{#each NEARBY_NOW as u (u.id)}
				<RailUserRow avatar={u.avatar} name={u.name} sub={u.sub} onClick={() => void goto(ROUTES.user(String(u.id)))} />
			{/each}
		</div>
	</RailSection>

	<!-- 更多动态（对齐 Figma PostDetailPage rail） -->
	<RailSection title="更多动态" action="发现" onAction={() => void goto(ROUTES.nearbyMoments)}>
		<div class="flex flex-col gap-3">
			{#each RELATED_MOMENTS as p (p.id)}
				<RailMomentThumb image={p.image} title={p.title} meta={p.meta} onClick={() => void goto(ROUTES.moment(String(p.id)))} />
			{/each}
		</div>
	</RailSection>
{/snippet}

<DesktopDetailSplit {main} {rail} />
