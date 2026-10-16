<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { MoreHorizontal, Link2, Send, PencilLine, Trash2, Heart, MessageCircle, Bookmark, Share2, Check, Flag } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import { platform } from '@/stores/platform.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import RailLocationMini from '@/components/rail/RailLocationMini.svelte';
	import RailMomentThumb from '@/components/rail/RailMomentThumb.svelte';
	import MomentPostBody from './MomentPostBody.svelte';
	import type { CommentItem } from '@/api/types';

	/** CommentItem → MomentComment（含 top_replies 一级回复映射；列表映射共用） */
	function toMomentComment(c: CommentItem): MomentComment {
		return {
			id: c.comment_id,
			userId: c.user?.user_id ?? '',
			user: c.user?.nickname ?? '',
			avatar: c.user?.avatar_url ?? '',
			text: c.content ?? '',
			time: formatMomentTime(c.created_at),
			likes: (c.reactions || []).reduce((sum, rx) => sum + (rx.count ?? 0), 0),
			liked: (c.reactions || []).some((rx) => rx.reacted_by_me),
			isDeleted: c.is_deleted,
			replies: (c.top_replies ?? []).map(toMomentComment),
			repliesCount: c.replies_count ?? 0
		};
	}
	import MomentDetailRail from './shared/MomentDetailRail.svelte';
	import DeleteMomentDialog from './shared/DeleteMomentDialog.svelte';
	import CommentInputBar from './shared/CommentInputBar.svelte';
	import { toNearbyRailRows } from './nearby-rail';
	import MomentEditDialog from '@/components/overlays/MomentEditDialog.svelte';
	import ReportDialog from '@/components/overlays/ReportDialog.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { getMomentDetail, listComments, listCommentReplies, setReaction, createComment, deleteComment, favoriteMoment, fetchNearbyMoments, searchMoments, deleteMoment, listMyMoments , netError } from '@/api';
	import { formatMomentTime } from '@/lib/utils';
	import { smartBack } from '@/lib/smart-back';
	import { copyShareLink } from '@/lib/share';
	import { clearMomentsProgress } from '@/lib/moments-feed-progress.svelte';

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
			fetchNearbyMoments({ pagination: { limit: 3 } }).catch(() => netError(null)),
			searchMoments({ sort_by: 'popular', pagination: { limit: 4 } }).catch(() => netError(null))
		]).then(([nearbyRes, relatedRes]) => {
			if (cancelled) return;
			const nearbyItems = nearbyRes.code === 0 ? nearbyRes.data?.items ?? [] : [];
			const relatedItems = relatedRes.code === 0 ? relatedRes.data?.items ?? [] : [];
			nearbyNow = toNearbyRailRows(nearbyItems);
			// 「更多动态」剔除当前帖( Gemini 审查 r19:重复推荐正在看的帖子 );
			// 多拉一条补偿过滤损耗,展示仍截 3 条
			relatedMoments = relatedItems
				.filter((m) => m.moment_id !== params.momentId)
				.slice(0, 3)
				.map((m) => ({
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
	// 评论 cursor 分页（契约 listComments 支持 pagination）：null = 无更多页。
	// 此前只拉第一页且无加载更多入口，超出首页的评论永远不可见。
	const COMMENT_PAGE_SIZE = 20;
	let commentsNextCursor = $state<string | null>(null);
	let commentsLoadingMore = $state(false);
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
	// 编辑所需原始字段（扁平 moment 未保留 description/tags/privacy_level）。
	// privacy_level 三态：null = 拥有者可见性未探测完成，编辑弹窗据此禁用可见性修改。
	let momentRaw = $state<{ description: string; tags: string[]; privacy_level: 'public' | 'private' | null }>({ description: '', tags: [], privacy_level: null });
	let deleteOpen = $state(false);
	let editOpen = $state(false);
	let deleting = $state(false);
	// 举报目标：动态本体或某条评论（spec api_spec_report.md target_type='moment'|'comment'）
	let reportTarget = $state<{ target_type: 'moment' | 'comment'; target_id: string } | null>(null);

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
					getMomentDetail(momentId).catch(() => netError(null)),
					listComments(momentId, { limit: COMMENT_PAGE_SIZE }).catch(() => netError({ items: [], next_cursor: null }))
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
						time: formatMomentTime(detailData.created_at),
						likes: (detailData.reactions ?? []).reduce((sum, r) => sum + r.count, 0),
						tag: detailData.tags?.[0] ?? '',
						comments: []
					};
					// 评论计数基数 = 服务端权威值；本页内新增走 commentDelta
					commentCountBase = detailData.comments_count ?? 0;
					commentDelta = 0;
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
						privacy_level: null
					};
					// privacy_level 属拥有者数据（OwnerMomentItem），不在公开详情类型 MomentItem 中：
					// 拥有者编辑前经 moments/fetch_created 取回实际可见性（视角分离）。
					// 探测完成前保持 null，编辑弹窗会禁用可见性修改并按「保持原值」提交。
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
					const items: MomentComment[] = (commentsData.items || []).map(toMomentComment);
					apiComments = items;
					commentsNextCursor = commentsData.next_cursor ?? null;
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

	// 评论计数基数取服务端权威 comments_count（评论列表只取第一页，
	// 本地列表长度会少算第 2 页起的评论）；commentDelta 承载本页内的增减。
	let commentCountBase = $state(0);
	let commentDelta = $state(0);
	const commentCount = $derived(commentCountBase + commentDelta);
	// 点赞计数：moment.likes 由本页维护（服务端快照 + toggleLike 乐观增量，已含本人状态），
	// 与 MomentPostBody 内 likeCount 同口径，避免按 liked 叠加导致刷新后计数双倍。
	const likeCount = $derived(moment.likes);
	const isAuthor = $derived(Boolean(auth.user?.user_id) && auth.user?.user_id === moment.authorId);

	// in-flight 去重：写请求未完成时忽略重复触发，避免连点双翻转 + 并发乱序
	let likePending = $state(false);
	let savePending = $state(false);
	let commentPending = $state(false);
	const commentLikePending = new Set<string>();

	// 回复目标（顶层评论）：非空时提交携带 parent_id，输入区上方显示取消条。
	let replyTarget = $state<{ commentId: string; userId: string; user: string } | null>(null);
	const startReply = (user: string, commentId: string) => {
		replyTarget = { commentId, userId: '', user };
		input = '@' + user + ' ';
		commentInputEl?.focus();
	};
	const cancelReply = () => {
		replyTarget = null;
		input = input.replace(/^@\S+\s*/, '');
	};
	// 「查看全部回复」：fetch_replies 拉全量（时间正序）替换预览。
	const expandReplies = async (commentId: string) => {
		const res = await listCommentReplies(commentId, { limit: 100 }).catch(() => null);
		if (!res || res.code !== 0 || !res.data) {
			showToast('加载回复失败，请重试', 'danger');
			return;
		}
		const full = res.data.items.map(toMomentComment);
		apiComments = apiComments.map((c) => (c.id === commentId ? { ...c, replies: full } : c));
	};

	const handleComment = async () => {
		const text = input.trim();
		if (!text || commentPending) return;
		// 写操作先过登录闸：匿名提交只会失败回滚，用户侧无任何反馈
		if (!requireAuth({ action: '评论', targetId: momentId, run: () => void handleComment() })) return;
		const reply = replyTarget;
		input = '';
		if (reply) {
			// 回复：携带 parent_id（spec 两级评论模型）+ mentions 通知被回复人；
			// 成功后插入父级 replies（父级不在已加载集合时重拉第一页）。
			commentPending = true;
			try {
				const res = await createComment({
					moment_id: momentId,
					content: text,
					parent_id: reply.commentId,
					mentions: reply.userId ? [reply.userId] : []
				});
				if (res.code !== 0 || !res.data) throw new Error(res.message || '回复发布失败');
				showToast('回复已发布', 'success');
				commentDelta += 1;
				const parent = apiComments.find((c) => c.id === reply.commentId);
				if (parent) {
					apiComments = apiComments.map((c) =>
						c.id === reply.commentId ? { ...c, replies: [...c.replies, toMomentComment(res.data)] } : c
					);
				} else {
					const refreshed = await listComments(momentId, { limit: COMMENT_PAGE_SIZE }).catch(() => netError({ items: [], next_cursor: null }));
					if (refreshed.code === 0 && refreshed.data) {
						apiComments = refreshed.data.items.map(toMomentComment);
						commentsNextCursor = refreshed.data.next_cursor ?? null;
					}
				}
				replyTarget = null;
			} catch (err) {
				showToast(err instanceof Error && err.message ? err.message : '回复发布失败，请重试', 'danger');
			} finally {
				commentPending = false;
			}
			return;
		}
		// Optimistic append; createComment confirms server-side.
		const optimistic: MomentComment = {
			id: `optimistic-${Date.now()}`,
			userId: auth.user?.user_id ?? '',
			user: auth.user?.nickname ?? '',
			avatar: auth.user?.avatar_url ?? '',
			text,
			time: '刚刚',
			likes: 0,
			liked: false,
			isDeleted: false,
			replies: [],
			repliesCount: 0
		};
		apiComments = [...apiComments, optimistic];
		commentDelta += 1;
		commentPending = true;
		try {
			// 业务失败（HTTP 200 + code≠0）api client 不抛错，必须显式判 code
			const res = await createComment({ moment_id: momentId, content: text });
			if (res.code !== 0 || !res.data) throw new Error(res.message || '评论发布失败');
			// 用服务端真实 CommentItem 回填乐观条目：后续点赞走
			// reactions/set(target_id=comment_id)，乐观占位 id 必然失败。
			const confirmed = toMomentComment(res.data);
			apiComments = apiComments.map((item) => (item.id === optimistic.id ? confirmed : item));
		} catch (err) {
			apiComments = apiComments.filter((c) => c.id !== optimistic.id);
			commentDelta -= 1;
			showToast(err instanceof Error && err.message ? err.message : '评论发布失败，请重试', 'danger');
		} finally {
			commentPending = false;
		}
	};

	// 加载更多评论（cursor 分页）：与 notify 页同款 LoadMoreButton 交互
	const loadMoreComments = async () => {
		if (commentsNextCursor === null || commentsLoadingMore) return;
		commentsLoadingMore = true;
		try {
			const res = await listComments(momentId, { limit: COMMENT_PAGE_SIZE, cursor: commentsNextCursor }).catch(() => netError({ items: [], next_cursor: null }));
			if (res.code === 0 && res.data) {
				const items: MomentComment[] = (res.data.items || []).map(toMomentComment);
				const known = new Set(apiComments.map((c) => c.id));
				apiComments = [...apiComments, ...items.filter((c) => !known.has(c.id))];
				commentsNextCursor = res.data.next_cursor ?? null;
			}
		} finally {
			commentsLoadingMore = false;
		}
	};

	// 评论点赞：登录闸 + 乐观翻转（likes 同步 ±1，服务端计数已含本人，刷新后口径一致）
	// + reactions/set(target_type='comment')，失败回滚
	const toggleCommentLike = async (commentId: string) => {		const target = apiComments.find((c) => c.id === commentId);
		if (!target || commentLikePending.has(commentId)) return;
		if (!requireAuth({ action: '点赞', targetId: commentId, run: () => void toggleCommentLike(commentId) })) return;
		const prev = target.liked;
		const prevLikes = target.likes;
		apiComments = apiComments.map((c) => (c.id === commentId ? { ...c, liked: !prev, likes: prevLikes + (prev ? -1 : 1) } : c));
		commentLikePending.add(commentId);
		try {
			const res = await setReaction({ target_type: 'comment', target_id: commentId, emoji: '👍', active: !prev });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			apiComments = apiComments.map((c) => (c.id === commentId ? { ...c, liked: prev, likes: prevLikes } : c));
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			commentLikePending.delete(commentId);
		}
	};

	// 删除自己的评论（spec: comments/delete，仅作者；软删除——顶层删后子评论保留，
	// 列表展示「该评论已删除」占位）。计数口径与 handleComment 的 +1 对称取 -1。
	let commentDeleteTarget = $state<string | null>(null);
	let commentDeletePending = $state(false);
	const handleCommentDelete = async () => {
		const target = commentDeleteTarget;
		if (!target || commentDeletePending) return;
		commentDeletePending = true;
		try {
			const res = await deleteComment({ comment_id: target });
			if (res.code !== 0) throw new Error(res.message || '删除失败');
			showToast('评论已删除', 'success');
			commentDelta -= 1;
			apiComments = apiComments.map((c) => (c.id === target ? { ...c, isDeleted: true, text: '', liked: false, likes: 0 } : c));
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '删除失败，请重试', 'danger');
		} finally {
			commentDeletePending = false;
			commentDeleteTarget = null;
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
		if (likePending) return;
		if (!requireAuth({ action: '点赞', targetId: momentId, run: () => void toggleLike() })) return;
		const next = !liked;
		const prevLikes = moment.likes;
		liked = next;
		// 点赞数乐观 ±1：moment.likes 是服务端快照（reactions[].count 已含本人），
		// 本页只承载快照后的增量；刷新重拉后口径自然对齐。
		moment = { ...moment, likes: prevLikes + (next ? 1 : -1) };
		likePending = true;
		try {
			const res = await setReaction({ target_type: 'moment', target_id: momentId, emoji: likeEmoji, active: next });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			liked = !next;
			moment = { ...moment, likes: prevLikes };
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			likePending = false;
		}
	};

	const toggleSave = async () => {
		if (savePending) return;
		if (!requireAuth({ action: '收藏', targetId: momentId, run: () => void toggleSave() })) return;
		const next = !saved;
		saved = next;
		savePending = true;
		try {
			const res = await favoriteMoment({ moment_id: momentId, active: next });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			saved = !next;
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			savePending = false;
		}
	};

	const handleDelete = async () => {
		deleting = true;
		try {
			const res = await deleteMoment({ moment_id: momentId });
			if (res.code === 0) {
				showToast('动态已删除', 'success');
				// 已删瞬间不再出现在附近 feed 的会话缓存里（返回列表即重拉）。
				clearMomentsProgress();
				smartBack();
			} else {
				showToast(res.message || '删除失败', 'danger');
			}
		} catch {
			showToast('删除失败，请重试', 'danger');
		} finally {
			deleting = false;
		}
	};

	const handleUpdated = (updated: { description: string; tags: string[]; privacy_level: 'public' | 'private' | null }) => {
		moment = { ...moment, content: updated.description, tag: updated.tags[0] ?? '' };
		momentRaw = updated;
	};

	// 分享：share/create 生成分享码并复制 /share/[code] 落地链接；
	// 创建失败（匿名/网络异常）降级为复制页面原始链接（lib/share 统一降级链）。
	const copyMomentLink = async () => {
		const result = await copyShareLink({ type: 'moment', id: momentId });
		shareStatus = result === 'failed' ? 'failed' : 'copied';
		moreOpen = false;
		window.setTimeout(() => (shareStatus = 'idle'), 1800);
	};

	let moreMenuEl = $state<HTMLElement | null>(null);
	// 评论输入框元素（focusComments/replyTo 聚焦用）
	let commentInputEl = $state<HTMLInputElement | null>(null);
</script>
<svelte:head>
	<title>瞬间详情 · HotSpot</title>
</svelte:head>


<SplitPageLayout contentClassName="animate-fade-in pt-4 pb-28">
	{#snippet header()}
		<!-- 命名对齐全站术语「瞬间」(左栏/搜索/设置均用瞬间;Gemini 审查 r19) -->
		<PageHeader title="瞬间详情" titleSize={platform.platform === 'desktop' ? 'xl' : '2xl'} onBack={() => smartBack()}>
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
								<DropdownMenu.Item class="text-status-danger hover:bg-danger-bg data-[highlighted]:bg-danger-bg" onSelect={() => { deleteOpen = true; moreOpen = false; }}>
									<Trash2 class="size-3.5" strokeWidth={1.75} />
									删除动态
								</DropdownMenu.Item>
							{:else if moment.id}
								<!-- 非作者可举报动态（spec: api_spec_report.md，目标属于当前用户时服务端 10400） -->
								<DropdownMenu.Item class="text-status-danger hover:bg-danger-bg data-[highlighted]:bg-danger-bg" onSelect={() => { reportTarget = { target_type: 'moment', target_id: momentId }; moreOpen = false; }}>
									<Flag class="size-3.5" strokeWidth={1.75} />
									举报动态
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
		<InlineLoading label="加载动态…" />
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
			hasMoreComments={commentsNextCursor !== null}
			loadingMoreComments={commentsLoadingMore}
			{liked}
			{saved}
			{shareStatus}
			onToggleLike={() => void toggleLike()}
			onToggleSave={() => void toggleSave()}
			onShare={() => void copyMomentLink()}
			onCommentClick={focusComments}
			onReply={(user, commentId) => startReply(user, commentId)}
			onExpandReplies={(commentId) => void expandReplies(commentId)}
			onToggleCommentLike={(id) => void toggleCommentLike(id)}
			onReportComment={(id) => (reportTarget = { target_type: 'comment', target_id: id })}
			onDeleteComment={(id) => (commentDeleteTarget = id)}
			onLoadMoreComments={() => void loadMoreComments()}
		/>
	{/if}

	{#snippet footer()}
		<!-- 移动端互动操作条（拇指热区）：点赞/评论/收藏/分享 44pt，内容区只读 -->
		{#if platform.platform === 'mobile'}
			<div class="flex shrink-0 items-stretch border-t border-border-default bg-surface-primary">
				<button
					type="button"
					class="hover-btn tap-92 flex h-12 flex-1 items-center justify-center gap-1.5 focus-ring {liked
						? 'text-accent-default'
						: 'text-text-secondary active:text-accent-default'}"
					aria-label={liked ? '取消点赞' : '点赞'}
					aria-pressed={liked}
					onclick={() => void toggleLike()}
				>
					<Heart class="size-5"  strokeWidth={1.5} fill={liked ? 'currentColor' : 'none'} />
					<span class="text-sm">{likeCount}</span>
				</button>
				<button
					type="button"
					class="hover-btn tap-92 flex h-12 flex-1 items-center justify-center gap-1.5 text-text-secondary focus-ring"
					aria-label="查看评论"
					onclick={focusComments}
				>
					<MessageCircle class="size-5"  strokeWidth={1.5} />
					<span class="text-sm">{commentCount}</span>
				</button>
				<button
					type="button"
					class="hover-btn tap-92 flex h-12 flex-1 items-center justify-center gap-1.5 focus-ring {saved
						? 'text-accent-default'
						: 'text-text-secondary active:text-accent-default'}"
					aria-label={saved ? '取消收藏' : '收藏'}
					aria-pressed={saved}
					onclick={() => void toggleSave()}
				>
					<Bookmark class="size-5"  strokeWidth={1.5} fill={saved ? 'currentColor' : 'none'} />
				</button>
				<button
					type="button"
					class="hover-btn tap-92 flex h-12 flex-1 items-center justify-center gap-1.5 focus-ring {shareStatus === 'copied' ? 'text-accent-default' : 'text-text-secondary'}"
					aria-label={shareStatus === 'failed' ? '复制动态链接失败' : '复制动态链接'}
					onclick={() => void copyMomentLink()}
				>
					{#if shareStatus === 'copied'}
						<Check class="size-5"  strokeWidth={1.75} />
					{:else}
						<Share2 class="size-5"  strokeWidth={1.5} />
					{/if}
				</button>
			</div>
		{/if}
		<!-- 评论输入栏固定在主区底部，不随正文滚动 -->
		{#if replyTarget}
			<div class="mx-6 mb-1 flex items-center justify-between rounded-md border border-border-default bg-surface-subtle px-3 py-1.5">
				<p class="truncate text-xs text-text-secondary">回复 @{replyTarget.user}</p>
				<button
					type="button"
					class="link-btn shrink-0 text-xs text-text-secondary"
					onclick={cancelReply}
				>取消回复</button>
			</div>
		{/if}
		<CommentInputBar bind:input bind:inputEl={commentInputEl} onSend={() => void handleComment()} />
	{/snippet}

	{#snippet rail()}
		<MomentDetailRail {moment} {nearbyNow} {relatedMoments} />
	{/snippet}
</SplitPageLayout>

<DeleteMomentDialog open={deleteOpen} {deleting} onCancel={() => (deleteOpen = false)} onConfirm={handleDelete} />

<ConfirmDialog
	open={commentDeleteTarget !== null}
	title="删除这条评论？"
	description="删除后将显示「该评论已删除」，其下的回复保留。"
	confirmLabel={commentDeletePending ? '删除中…' : '确认删除'}
	loading={commentDeletePending}
	onCancel={() => (commentDeleteTarget = null)}
	onConfirm={() => void handleCommentDelete()}
/>

<ReportDialog
	open={reportTarget !== null}
	targetType={reportTarget?.target_type ?? 'moment'}
	targetId={reportTarget?.target_id ?? ''}
	targetLabel={reportTarget?.target_type === 'comment' ? '这条评论' : '这条动态'}
	onClose={() => (reportTarget = null)}
/>

<MomentEditDialog
	open={editOpen}
	{momentId}
	initial={momentRaw}
	onClose={() => (editOpen = false)}
	onUpdated={handleUpdated}
/>
