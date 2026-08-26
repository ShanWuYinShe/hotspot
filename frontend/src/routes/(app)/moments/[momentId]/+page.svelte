<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { MoreHorizontal, Link2, Send, PencilLine, Trash2 } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import RailLocationMini from '@/components/rail/RailLocationMini.svelte';
	import RailMomentThumb from '@/components/rail/RailMomentThumb.svelte';
	import MomentPostBody from './MomentPostBody.svelte';
	import { toNearbyRailRows } from './nearby-rail';
	import MomentEditDialog from '@/components/overlays/MomentEditDialog.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { showToast } from '@/stores/toast.svelte';
	import { getMomentDetail, listComments, setReaction, createComment, favoriteMoment, fetchNearbyMoments, searchMoments, deleteMoment, listMyMoments } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import { formatDateTime } from '@/lib/utils';

	// 动态详情：点赞/收藏/评论/复制链接/更多菜单 + 桌面右 Rail。

	let { params }: { params: { momentId: string } } = $props();

	// 页面私有类型：内容体由 MomentPostBody 渲染（类型与其导出保持一致）。
	type MomentComment = import('./MomentPostBody.svelte').MomentComment;
	type MomentData = import('./MomentPostBody.svelte').MomentData;

	// 桌面 rail 数据：此刻附近来自 moments/fetch_nearby，更多动态来自 moments/search 热门模式（均走 MSW mock）。
	let nearbyNow = $state<{ id: string; name: string; sub: string; avatar: string }[]>([]);
	let relatedMoments = $state<{ id: string; title: string; meta: string; image: string }[]>([]);

	$effect(() => {
		let cancelled = false;
		void Promise.all([
			fetchNearbyMoments({ pagination: { limit: 3 } }).catch(() => ({ code: -1, message: '网络异常', data: null })),
			searchMoments({ sort_by: 'popular', pagination: { limit: 3 } }).catch(() => ({ code: -1, message: '网络异常', data: null }))
		]).then(([nearbyRes, relatedRes]) => {
			if (cancelled) return;
			const nearbyItems = nearbyRes.code === 0 ? nearbyRes.data?.items ?? [] : [];
			const relatedItems = relatedRes.code === 0 ? relatedRes.data?.items ?? [] : [];
			nearbyNow = toNearbyRailRows(nearbyItems);
			relatedMoments = relatedItems.map((m) => ({
				id: m.moment_id,
				title: m.description ?? '',
				meta: `${m.user?.nickname ?? ''} · ${(m.reactions ?? []).reduce((sum, r) => sum + r.count, 0)} 赞`,
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
			images: [],
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
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);
	let input = $state('');
	let moreOpen = $state(false);
	let shareStatus = $state<'idle' | 'copied' | 'failed'>('idle');
	// Reaction/favorite state derived from the moment detail (spec: liked state
	// comes from reactions[].reacted_by_me; saved from my_state.is_favorited).
	let liked = $state(false);
	let saved = $state(false);
	let likeEmoji = $state('👍');
	// 编辑所需原始字段（扁平 moment 未保留 description/tags/privacy_level）
	let momentRaw = $state<{ description: string; tags: string[]; privacy_level: 'public' | 'private' }>({ description: '', tags: [], privacy_level: 'public' });
	let deleteOpen = $state(false);
	let editOpen = $state(false);
	let deleting = $state(false);

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchMoment() {
			try {
				loading = true;
				loadError = null;
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

				// 动态不存在或加载失败：与 notify/users 分栏页一致，展示 LoadError + 重试
				if (!detailData) {
					loadError = detailRes.code === 0 ? '动态不存在或已被删除' : (detailRes.message || '加载失败');
				} else {
					loadError = null;
				}

				if (detailData) {
					moment = {
						id: momentId,
						authorId: detailData.user?.user_id ?? '',
						user: detailData.user?.nickname ?? '',
							avatar: detailData.user?.avatar_url ?? '',
						location: detailData.location?.location_name ?? '',
						images: (detailData.images ?? []).map((im) => im.url).filter(Boolean),
						content: detailData.description ?? '',
						time: formatDateTime(detailData.created_at),
						likes: (detailData.reactions ?? []).reduce((sum, r) => sum + r.count, 0),
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
					saved = Boolean(detailData.is_favorited);
					momentRaw = {
						description: detailData.description ?? '',
						tags: detailData.tags ?? [],
						privacy_level: 'public'
					};
					// privacy_level 属拥有者数据（OwnerMomentItem），不在公开详情类型 MomentItem 中：
					// 拥有者编辑前经 moments/fetch_created 取回实际可见性（视角分离）。
					if (detailData.user?.user_id === auth.user?.user_id) {
						void (async () => {
							let cursor: string | null = null;
							for (let i = 0; i < 10; i++) {
								const ownRes: Awaited<ReturnType<typeof listMyMoments>> | null = await listMyMoments({ pagination: { limit: 100, cursor: cursor ?? undefined } }).catch(() => null);
								if (!ownRes || ownRes.code !== 0 || !ownRes.data) break;
								const found = ownRes.data.items.find((m) => m.moment_id === momentId);
								if (found) {
									momentRaw = { ...momentRaw, privacy_level: found.privacy_level };
									break;
								}
								cursor = ownRes.data.next_cursor;
								if (!cursor) break;
							}
						})();
					}
				}

				if (commentsData) {
					const items: MomentComment[] = (commentsData.items || []).map((c) => ({
						// 真实 comment_id（评论点赞 reactions/set target_id 用）
						id: c.comment_id,
						userId: c.user?.user_id ?? '',
						user: c.user?.nickname ?? '',
						avatar: c.user?.avatar_url ?? '',
						text: c.content ?? '',
						time: formatDateTime(c.created_at),
						likes: (c.reactions || []).reduce((s, r) => s + (r.count ?? 0), 0),
						liked: (c.reactions || []).some((r) => r.reacted_by_me)
					}));
					apiComments = items;
				}
			} catch {
				// empty data is normal
			} finally {
				if (!cancelled) {
					loading = false;
					// 卡片评论按钮深链（?focus=comments）：直达评论区并聚焦输入栏。
					// rAF 等 loading=false 的 DOM 渲染完成后再定位。
					if (page.url.searchParams.get('focus') === 'comments') {
						requestAnimationFrame(() => focusComments());
					}
				}
			}
		}
		void fetchMoment();
		return () => {
			cancelled = true;
		};
	});

	const commentCount = $derived(apiComments.length);
	const isAuthor = $derived(Boolean(auth.user?.user_id) && auth.user?.user_id === moment.authorId);

	const handleComment = async () => {
		const text = input.trim();
		if (!text) return;
		// 写操作先过登录闸：匿名提交只会失败回滚，用户侧无任何反馈
		if (!requireAuth({ action: '评论', targetId: momentId, run: () => void handleComment() })) return;
		input = '';
		// Optimistic append; createComment confirms server-side.
		const optimistic: MomentComment = {
			id: `optimistic-${Date.now()}`,
			userId: auth.user?.user_id ?? '',
			user: auth.user?.nickname ?? '',
			avatar: auth.user?.avatar_url ?? '',
			text,
			time: '刚刚',
			likes: 0,
			liked: false
		};
		apiComments = [...apiComments, optimistic];
		try {
			await createComment({ moment_id: momentId, content: text });
		} catch {
			apiComments = apiComments.filter((c) => c.id !== optimistic.id);
		}
	};

	// 评论点赞：登录闸 + 乐观翻转 + reactions/set(target_type='comment')，失败回滚
	const toggleCommentLike = async (commentId: string) => {
		const target = apiComments.find((c) => c.id === commentId);
		if (!target) return;
		if (!requireAuth({ action: '点赞', targetId: commentId, run: () => void toggleCommentLike(commentId) })) return;
		const prev = target.liked;
		apiComments = apiComments.map((c) => (c.id === commentId ? { ...c, liked: !prev } : c));
		try {
			await setReaction({ target_type: 'comment', target_id: commentId, emoji: '👍', active: !prev });
		} catch {
			apiComments = apiComments.map((c) => (c.id === commentId ? { ...c, liked: prev } : c));
		}
	};

	// 操作条评论计数点击：滚动到评论区并聚焦输入栏
	const focusComments = () => {
		document.getElementById('moment-comments')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
		commentInputEl?.focus();
	};

	// 评论区"回复"：聚焦输入栏并 @ 对方
	const replyTo = (user: string) => {
		input = `@${user} `;
		commentInputEl?.focus();
	};

	const toggleLike = async () => {
		if (!requireAuth({ action: '点赞', targetId: momentId, run: () => void toggleLike() })) return;
		const next = !liked;
		liked = next;
		try {
			await setReaction({ target_type: 'moment', target_id: momentId, emoji: likeEmoji, active: next });
		} catch {
			liked = !next;
		}
	};

	const toggleSave = async () => {
		if (!requireAuth({ action: '收藏', targetId: momentId, run: () => void toggleSave() })) return;
		const next = !saved;
		saved = next;
		try {
			await favoriteMoment({ moment_id: momentId, active: next });
		} catch {
			saved = !next;
		}
	};

	const handleDelete = async () => {
		deleting = true;
		try {
			const res = await deleteMoment({ moment_id: momentId });
			if (res.code === 0) {
				showToast('动态已删除', 'success');
				history.back();
			} else {
				showToast(res.message || '删除失败', 'danger');
			}
		} catch {
			showToast('删除失败，请重试', 'danger');
		} finally {
			deleting = false;
		}
	};

	const handleUpdated = (updated: { description: string; tags: string[]; privacy_level: 'public' | 'private' }) => {
		moment = { ...moment, content: updated.description, tag: updated.tags[0] ?? '' };
		momentRaw = updated;
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
	// 评论输入框元素（focusComments/replyTo 聚焦用）
	let commentInputEl = $state<HTMLInputElement | null>(null);
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
						{#if isAuthor}
							<DropdownMenu.Item onSelect={() => { editOpen = true; moreOpen = false; }}>
								<PencilLine class="size-3.5 text-accent-default" strokeWidth={1.75} />
								编辑动态
							</DropdownMenu.Item>
							<DropdownMenu.Item class="text-status-danger" onSelect={() => { deleteOpen = true; moreOpen = false; }}>
								<Trash2 class="size-3.5" strokeWidth={1.75} />
								删除动态
							</DropdownMenu.Item>
						{/if}
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			{/snippet}
		</PageHeader>

		<p aria-live="polite" class="sr-only">
			{shareStatus === 'copied' ? '动态链接已复制' : shareStatus === 'failed' ? '无法复制动态链接' : ''}
		</p>
	{/snippet}

	<!-- 内容（gutter 由 SplitPageLayout 提供；全幅封面在 MomentPostBody 内负 margin 抵消） -->
	{#if loading}
		<div class="flex flex-col items-center gap-3 py-20">
			<div class="spinner size-7"></div>
			<p class="text-text-secondary text-sm">加载动态…</p>
		</div>
	{:else if loadError}
		<LoadError message={loadError} onRetry={() => reloadKey++} />
	{:else if !moment.id}
		<EmptyState title="动态不存在" description="这条动态可能已被删除或没有权限查看。">
			{#snippet icon()}
				<MoreHorizontal class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
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
			onCommentClick={focusComments}
			onReply={replyTo}
			onToggleCommentLike={(id) => void toggleCommentLike(id)}
		/>
	{/if}

	{#snippet footer()}
		<!-- 评论输入栏固定在主区底部，不随正文滚动 -->
		<div class="shrink-0 flex items-center gap-2.5 px-4 py-3 bg-surface-primary border-t border-border-default" >
			<Avatar class="border border-border-default" src={auth.user?.avatar_url} name={auth.user?.nickname} size={30}  />
			<div class="flex-1 flex items-center px-4 rounded-md input-surface h-9.5" >
				<input
					bind:this={commentInputEl}
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
				class="hover-btn scale-btn tap-88 flex items-center justify-center rounded-full focus-ring shrink-0 size-9" style="background:{input.trim() ? 'var(--cst-accent-default)' : 'var(--cst-surface-primary)'};box-shadow:{input.trim() ? 'var(--shadow-glow-md)' : 'var(--shadow-raised-sm)'};border:var(--cst-border-width-hairline) solid var(--cst-border-default);transition:all var(--cst-motion-standard) var(--cst-ease-standard);"
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
	{/snippet}
</SplitPageLayout>

<Dialog.Root open={deleteOpen} onOpenChange={(next) => (next ? undefined : (deleteOpen = false))}>
	<Dialog.Content
		side="center"
		class="max-w-sm p-5"
		aria-label="确认删除动态"
	>
		<p class="text-md text-text-primary">删除这条动态？</p>
		<p class="mt-1 text-sm text-text-secondary">删除后无法恢复，相关评论也将一并移除。</p>
		<div class="mt-4 flex items-center justify-end gap-3">
			<button
				type="button"
				onclick={() => (deleteOpen = false)}
				class="hover-btn rounded-full px-4 py-2 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
			>
				取消
			</button>
			<button
				type="button"
				disabled={deleting}
				onclick={handleDelete}
				class="hover-btn rounded-full px-5 py-2 text-sm font-medium text-text-on-accent outline-none focus-visible:ring-2 focus-visible:ring-status-danger enabled:bg-status-danger disabled:opacity-muted"
			>
				{deleting ? '删除中…' : '确认删除'}
			</button>
		</div>
	</Dialog.Content>
</Dialog.Root>

<MomentEditDialog
	open={editOpen}
	{momentId}
	initial={momentRaw}
	onClose={() => (editOpen = false)}
	onUpdated={handleUpdated}
/>
