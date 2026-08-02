<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, Heart, MessageCircle, Share2, Bookmark, MoreHorizontal, Send, Link2, Check } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { auth } from '@/stores/auth.svelte';
	import DesktopDetailSplit from '@/components/layout/desktop/DesktopDetailSplit.svelte';
	import RailSection from '@/components/layout/desktop/RailSection.svelte';
	import RailUserRow from '@/components/layout/desktop/RailUserRow.svelte';
	import RailLocationMini from '@/components/layout/desktop/RailLocationMini.svelte';
	import RailPostThumb from '@/components/layout/desktop/RailPostThumb.svelte';
	import { getMomentDetail, listComments, setReaction, createComment, favoriteMoment } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 动态详情：点赞/收藏/评论/复制链接/更多菜单 + 桌面右 Rail。
	// 迁移自旧版 MomentDetailPage。

	let { params }: { params: { postId: string } } = $props();

	interface PostComment {
		id: number;
		user: string;
		avatar: string;
		text: string;
		time: string;
		likes: number;
	}
	interface PostData {
		id: number;
		user: string;
		handle: string;
		avatar: string;
		location: string;
		image: string;
		content: string;
		time: string;
		likes: number;
		tag: string;
		comments: PostComment[];
	}

	// 桌面 rail 占位数据（对齐 Figma PostDetailPage NEARBY_NOW/RELATED_POSTS；
	// 后端无专用端点，先用占位引导，后续接入 feed/nearby 与相关推荐）。
	const NEARBY_NOW = [
		{ id: 1, name: '鹿鸣', sub: '246m · 摄影', avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80' },
		{ id: 2, name: '阿禾', sub: '388m · 旅行', avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80' },
		{ id: 3, name: '野原', sub: '520m · 音乐', avatar: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80' }
	];
	const RELATED_POSTS = [
		{ id: 40, title: '植物园晨光', meta: '南枝 · 124 赞', image: 'https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=120' },
		{ id: 41, title: '城市夜光', meta: '本地向导 · 89 赞', image: 'https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=120' },
		{ id: 42, title: '老街手冲馆', meta: '野原 · 56 赞', image: 'https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=120' }
	];

	// params 为路由参数（SPA 内可能变化），此处用 $derived 保持响应。
	const numericPostId = $derived(Number(params.postId));
	const momentId = $derived(String(numericPostId));

	function emptyPost(): PostData {
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
	let post = $state<PostData>(emptyPost());
	let apiComments = $state<PostComment[]>([]);
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
		async function fetchPost() {
			try {
				loading = true;
				// 路由参数切换（SPA 内 postId 变化）时清空旧数据，避免新数据到达前残留展示
				post = emptyPost();
				apiComments = [];
				const [detailRes, commentsRes] = await Promise.all([
					getMomentDetail(momentId).catch(() => ({ code: -1, data: null })),
					listComments(momentId).catch(() => ({ code: -1, data: { items: [] } }))
				]);
				if (cancelled) return;

				const detailData = (detailRes.code === 0 ? detailRes.data : null) as any;
				const commentsData = (commentsRes.code === 0 ? commentsRes.data : null) as any;

				if (detailData) {
					post = {
						id: detailData.id ?? detailData.moment_id ?? numericPostId,
						user: detailData.user?.nickname ?? detailData.author_name ?? detailData.nickname ?? '',
						handle: detailData.handle ?? detailData.username ?? '',
						avatar: detailData.avatar ?? detailData.user?.avatar_url ?? detailData.author_avatar ?? '',
						location: detailData.location?.location_name ?? detailData.place ?? '',
						image: detailData.image ?? detailData.images?.[0]?.url ?? detailData.images?.[0] ?? '',
						content: detailData.content ?? detailData.text ?? detailData.body ?? '',
						time: detailData.time ?? detailData.created_at ?? '',
						likes: detailData.likes_count ?? 0,
						tag: detailData.tag ?? detailData.tags?.[0] ?? '',
						comments: []
					};
					// Derive liked from reactions[].reacted_by_me (spec); saved from my_state.
					const mine = (detailData.reactions || []).find((r: any) => r.reacted_by_me);
					if (mine) {
						liked = true;
						likeEmoji = mine.emoji ?? '👍';
					} else {
						liked = false;
					}
					saved = Boolean(detailData.my_state?.is_favorited);
				}

				if (commentsData) {
					const items: PostComment[] = (commentsData.items || []).map((c: any, i: number) => ({
						id: c.id ?? i + 1,
						user: c.user?.nickname ?? c.author_name ?? c.nickname ?? '',
						avatar: c.avatar ?? c.user?.avatar_url ?? c.author_avatar ?? '',
						text: c.text ?? c.content ?? c.body ?? '',
						time: c.time ?? c.created_at ?? '',
						likes: (c.reactions || []).reduce((s: number, r: any) => s + (r.count ?? 0), 0)
					}));
					apiComments = items;
				}
			} catch {
				// empty data is normal
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchPost();
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
		const optimistic: PostComment = {
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

	const copyPostLink = async () => {
		const postUrl = `${window.location.origin}${ROUTES.moment(numericPostId)}`;
		try {
			if (!navigator.clipboard?.writeText) throw new Error('Clipboard API unavailable');
			await navigator.clipboard.writeText(postUrl);
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
		<PageHeader title={post.user || '详情'} onBack={() => history.back()} className="flex-shrink-0 border-b border-[var(--neu-border)]">
			{#snippet trailing()}
				<DropdownMenu.Root bind:open={moreOpen}>
					<div bind:this={moreMenuEl} class="relative">
						<ActionIconButton icon={MoreHorizontal} label="更多操作" onClick={() => (moreOpen = !moreOpen)} />
					</div>
					<DropdownMenu.Content customAnchor={moreMenuEl}>
						<DropdownMenu.Item onSelect={() => void copyPostLink()}>
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
			<!-- Post image -->
			<div class="relative overflow-hidden" style="height:clamp(220px, 38vw, 380px);">
				{#if post.image}
					<img src={post.image} alt="post" class="w-full h-full object-cover" />
				{:else}
					<div class="flex h-full w-full flex-col items-center justify-center gap-2 bg-[color-mix(in_srgb,var(--neu-accent-primary)_12%,var(--neu-bg))]">
						<MapPin size={28} class="text-[var(--neu-accent-primary)]" strokeWidth={1.5} />
						<span class="text-[var(--neu-text-muted)] text-xs">来自 {post.location || '附近'} 的动态</span>
					</div>
				{/if}
				<div class="absolute top-3 left-3 px-2.5 py-0.5 rounded-full bg-[var(--neu-accent-primary)]" style="box-shadow:0 2px 8px var(--neu-glow);">
					<span class="text-white text-[10px]">#{post.tag}</span>
				</div>
			</div>

			<ContentColumn maxWidth="var(--content-reading-max)" class="pt-4 pb-3">
				<!-- Author -->
				<div class="flex items-center gap-3 mb-3">
					<button
						type="button"
						class="neu-hover-btn rounded-full overflow-hidden flex-shrink-0 cursor-pointer neu-focus-ring focus-visible:ring-offset-[var(--neu-bg)]"
						style="width:44px;height:44px;padding:0;background:transparent;border:2px solid var(--neu-accent-primary);"
						aria-label={`查看${post.user}的个人主页`}
						onclick={() => void goto(ROUTES.user(post.id))}
					>
						<Avatar src={post.avatar} name={post.user} size={44} style="border:none;" />
					</button>
					<div class="flex-1">
						<p class="text-[var(--neu-text)] text-[15px]">{post.user}</p>
						<div class="flex items-center gap-1.5">
							<span class="text-[var(--neu-text-muted)] text-xs">{post.handle}</span>
							<span class="text-[var(--neu-text-muted)] text-[10px]" style="opacity:0.5;">·</span>
							<MapPin style="width:10px;height:10px;color:var(--neu-accent-primary);" strokeWidth={1.5} />
							<span class="text-[var(--neu-text-muted)] text-[11px]">{post.location}</span>
						</div>
					</div>
					<span class="text-[var(--neu-text-muted)] text-[11px]">{post.time}</span>
				</div>

				<!-- Content -->
				<p class="text-[var(--neu-text)] text-sm" style="line-height:1.75;">{post.content}</p>

				<!-- Actions -->
				<div class="flex items-center justify-between mt-4 py-3" style="border-top:1px solid var(--neu-border);border-bottom:1px solid var(--neu-border);">
					<div class="flex gap-4">
						<button
							type="button"
							class="neu-hover-action flex items-center gap-1.5 outline-none {liked ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}"
							onclick={() => void toggleLike()}
						>
							<Heart style="width:19px;height:19px;" strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
							<span class="text-[13px]">{post.likes + (liked ? 1 : 0)}</span>
						</button>
						<div class="flex items-center gap-1.5 text-[var(--neu-text-muted)]">
							<MessageCircle style="width:19px;height:19px;" strokeWidth={1.5} />
							<span class="text-[13px]">{commentCount}</span>
						</div>
					</div>
					<div class="flex gap-3">
						<button
							type="button"
							class="neu-hover-action outline-none {saved ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}"
							onclick={() => void toggleSave()}
						>
							<Bookmark style="width:19px;height:19px;" strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
						</button>
						<button
							type="button"
							onclick={() => void copyPostLink()}
							aria-label={shareStatus === 'failed' ? '复制动态链接失败' : '复制动态链接'}
							class="neu-hover-action rounded-full neu-focus-ring {shareStatus === 'copied' ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}"
						>
							{#if shareStatus === 'copied'}
								<Check style="width:19px;height:19px;" strokeWidth={1.8} />
							{:else}
								<Share2 style="width:19px;height:19px;" strokeWidth={1.5} />
							{/if}
						</button>
					</div>
				</div>

				<!-- Comments -->
				<p class="uppercase mt-4 mb-3 text-[var(--neu-text-muted)] text-[10px]" style="letter-spacing:0.14em;">
					评论 · {commentCount}
				</p>
				<div class="flex flex-col gap-4">
					{#each apiComments as c (c.id)}
						<div class="flex gap-3">
							<Avatar src={c.avatar} name={c.user} size={34} />
							<div class="flex-1">
								<div class="flex items-baseline gap-2">
									<span class="text-[var(--neu-text)] text-[13px]">{c.user}</span>
									<span class="text-[var(--neu-text-muted)] text-[11px]">{c.time}</span>
								</div>
								<p class="text-[var(--neu-text-muted)] text-[13px]" style="line-height:1.6;margin-top:2px;">{c.text}</p>
								<div class="flex items-center gap-1 mt-1.5">
									<button type="button" class="neu-hover-action flex items-center gap-1 outline-none text-[var(--neu-text-muted)]">
										<Heart style="width:12px;height:12px;" strokeWidth={1.5} />
										<span class="text-[11px]">{c.likes}</span>
									</button>
									<span class="text-[var(--neu-text-muted)] text-[11px]" style="margin-left:8px;cursor:pointer;">回复</span>
								</div>
							</div>
						</div>
					{/each}
				</div>
				<div style="height:80px;"></div>
			</ContentColumn>
		</div>

		<!-- Input bar -->
		<div class="flex-shrink-0 flex items-center gap-2.5 px-4 py-3 bg-[var(--neu-bg)]" style="border-top:1px solid var(--neu-border);">
			<Avatar src={auth.user?.avatar_url} name={auth.user?.nickname} size={30} style="border:1px solid var(--neu-border);" />
			<div class="flex-1 flex items-center px-4 rounded-[18px] neu-input-surface" style="height:38px;">
				<input
					bind:value={input}
					onkeydown={(e) => {
						if (e.key === 'Enter') void handleComment();
					}}
					placeholder="写下你的评论…"
					class="flex-1 bg-transparent outline-none text-[var(--neu-text)] text-[13px]"
				/>
			</div>
			<button
				type="button"
				class="neu-hover-btn flex items-center justify-center rounded-full outline-none flex-shrink-0"
				style="width:36px;height:36px;background:{input.trim() ? 'var(--neu-accent-primary)' : 'var(--neu-bg)'};box-shadow:{input.trim() ? '0 4px 14px var(--neu-glow)' : '4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)'};border:1px solid var(--neu-border);transition:all 0.2s;"
				onclick={() => void handleComment()}
			>
				<Send style="width:14px;height:14px;color:{input.trim() ? 'white' : 'var(--neu-text-muted)'};transform:translateX(1px);" strokeWidth={2} />
			</button>
		</div>
	</div>
{/snippet}

{#snippet rail()}
	<RailSection title="作者">
		<RailUserRow avatar={post.avatar} name={post.user} sub={`${post.handle} · ${post.location}`} onClick={() => void goto(ROUTES.user(post.id))}>
			{#snippet trailing()}
				<button
					type="button"
					class="neu-hover-btn px-2.5 py-1 rounded-full outline-none flex-shrink-0 bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)]"
					style="font-size:11px;border:1px solid var(--neu-accent-soft-border);"
					onclick={(e) => {
						e.stopPropagation();
						void goto(ROUTES.message(post.id));
					}}
				>
					私信
				</button>
			{/snippet}
		</RailUserRow>
	</RailSection>

	<RailSection title="相关位置">
		<RailLocationMini name={post.location} desc="共 12 条动态 · 6 人在此" />
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
			{#each RELATED_POSTS as p (p.id)}
				<RailPostThumb image={p.image} title={p.title} meta={p.meta} onClick={() => void goto(ROUTES.moment(String(p.id)))} />
			{/each}
		</div>
	</RailSection>
{/snippet}

<DesktopDetailSplit {main} {rail} />
