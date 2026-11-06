import { page } from '$app/state';
import { auth, requireAuth } from '@/stores/auth.svelte';
import { showToast } from '@/stores/toast.svelte';
import { smartBack } from '@/lib/smart-back';
import { copyShareLink } from '@/lib/share';
import { clearMomentsProgress } from '@/lib/moments-feed-progress.svelte';
import { formatMomentTime } from '@/lib/utils';
import { toNearbyRailRows } from '../nearby-rail';
import {
	getMomentDetail,
	listComments,
	listCommentReplies,
	setReaction,
	createComment,
	deleteComment,
	favoriteMoment,
	fetchNearbyMoments,
	searchMoments,
	deleteMoment,
	listMyMoments,
	netError
} from '@/api';
import type { CommentItem } from '@/api/types';
import type { MomentComment, MomentData } from '../MomentPostBody.svelte';

export const COMMENT_PAGE_SIZE = 20;

export function toMomentComment(c: CommentItem): MomentComment {
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

export function emptyMoment(): MomentData {
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

export function useMomentDetailData(getMomentId: () => string) {
	const momentId = $derived(getMomentId());

	let moment = $state<MomentData>(emptyMoment());
	let apiComments = $state<MomentComment[]>([]);
	let commentsNextCursor = $state<string | null>(null);
	let commentsLoadingMore = $state(false);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	let input = $state('');
	let moreOpen = $state(false);
	let shareStatus = $state<'idle' | 'copied' | 'failed'>('idle');
	let liked = $state(false);
	let saved = $state(false);
	let likeEmoji = $state('👍');
	let momentRaw = $state<{ description: string; tags: string[]; privacy_level: 'public' | 'private' | null }>({
		description: '',
		tags: [],
		privacy_level: null
	});

	let deleteOpen = $state(false);
	let editOpen = $state(false);
	let deleting = $state(false);
	let reportTarget = $state<{ target_type: 'moment' | 'comment'; target_id: string } | null>(null);
	let commentDeleteTarget = $state<string | null>(null);
	let commentDeletePending = $state(false);

	let commentCountBase = $state(0);
	let commentDelta = $state(0);
	let newCommentId = $state<string | null>(null);
	let newReplyId = $state<string | null>(null);

	let likePending = $state(false);
	let savePending = $state(false);
	let commentPending = $state(false);
	const commentLikePending = new Set<string>();

	let replyTarget = $state<{ commentId: string; userId: string; user: string } | null>(null);

	// Rail 数据
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
			relatedMoments = relatedItems
				.filter((m) => m.moment_id !== momentId)
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

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchMoment() {
			try {
				loading = true;
				loadError = null;
				moment = emptyMoment();
				apiComments = [];
				const [detailRes, commentsRes] = await Promise.all([
					getMomentDetail(momentId).catch(() => netError(null)),
					listComments(momentId, { limit: COMMENT_PAGE_SIZE }).catch(() => netError({ items: [], next_cursor: null }))
				]);
				if (cancelled) return;

				const detailData = detailRes.code === 0 ? detailRes.data : null;
				const commentsData = commentsRes.code === 0 ? commentsRes.data : null;

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
						imageSizes: (detailData.images ?? []).map((im) => ({
							width: im.width ?? null,
							height: im.height ?? null
						})),
						content: detailData.description ?? '',
						time: formatMomentTime(detailData.created_at),
						likes: (detailData.reactions ?? []).reduce((sum, r) => sum + r.count, 0),
						tag: detailData.tags?.[0] ?? '',
						comments: []
					};
					commentCountBase = detailData.comments_count ?? 0;
					commentDelta = 0;
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
				}

				if (commentsData) {
					apiComments = (commentsData.items || []).map(toMomentComment);
					commentsNextCursor = commentsData.next_cursor ?? null;
				}
			} finally {
				if (!cancelled) {
					loading = false;
					if (page.url.searchParams.get('focus') === 'comments') {
						requestAnimationFrame(() => {
							document.getElementById('moment-comments')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
						});
					}
				}
			}
		}
		void fetchMoment();
		return () => {
			cancelled = true;
		};
	});

	function markNewComment(id: string) {
		newCommentId = id;
		setTimeout(() => { if (newCommentId === id) newCommentId = null; }, 600);
	}
	function markNewReply(id: string) {
		newReplyId = id;
		setTimeout(() => { if (newReplyId === id) newReplyId = null; }, 600);
	}

	const commentCount = $derived(commentCountBase + commentDelta);
	const isAuthor = $derived(Boolean(auth.user?.user_id) && auth.user?.user_id === moment.authorId);

	const startReply = (user: string, commentId: string) => {
		replyTarget = { commentId, userId: '', user };
		input = '@' + user + ' ';
	};

	const cancelReply = () => {
		replyTarget = null;
		input = input.replace(/^@\S+\s*/, '');
	};

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
		if (!requireAuth({ action: '评论', targetId: momentId, run: () => void handleComment() })) return;
		const reply = replyTarget;
		input = '';

		if (reply) {
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
				const newReply = toMomentComment(res.data);
				const parent = apiComments.find((c) => c.id === reply.commentId);
				if (parent) {
					apiComments = apiComments.map((c) =>
						c.id === reply.commentId ? { ...c, replies: [...c.replies, newReply] } : c
					);
					markNewReply(newReply.id);
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
		markNewComment(optimistic.id);
		commentDelta += 1;
		commentPending = true;
		try {
			const res = await createComment({ moment_id: momentId, content: text });
			if (res.code !== 0 || !res.data) throw new Error(res.message || '评论发布失败');
			const confirmed = toMomentComment(res.data);
			apiComments = apiComments.map((item) => (item.id === optimistic.id ? confirmed : item));
			markNewComment(confirmed.id);
		} catch (err) {
			apiComments = apiComments.filter((c) => c.id !== optimistic.id);
			commentDelta -= 1;
			showToast(err instanceof Error && err.message ? err.message : '评论发布失败，请重试', 'danger');
		} finally {
			commentPending = false;
		}
	};

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

	const toggleCommentLike = async (commentId: string) => {
		const target = apiComments.find((c) => c.id === commentId);
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

	const toggleLike = async () => {
		if (likePending) return;
		if (!requireAuth({ action: '点赞', targetId: momentId, run: () => void toggleLike() })) return;
		const next = !liked;
		const prevLikes = moment.likes;
		liked = next;
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

	const openEditDialog = async () => {
		editOpen = true;
		moreOpen = false;
		if (isAuthor && momentRaw.privacy_level === null) {
			try {
				let cursor: string | null = null;
				for (let i = 0; i < 2; i++) {
					const ownRes: Awaited<ReturnType<typeof listMyMoments>> | null = await listMyMoments({ pagination: { limit: 20, cursor: cursor ?? undefined } }).catch(() => null);
					if (!ownRes || ownRes.code !== 0 || !ownRes.data) break;
					const found = ownRes.data.items.find((m) => m.moment_id === momentId);
					if (found) {
						momentRaw = { ...momentRaw, privacy_level: found.privacy_level };
						break;
					}
					cursor = ownRes.data.next_cursor;
					if (!cursor) break;
				}
			} catch {}
		}
	};

	const copyMomentLink = async () => {
		const result = await copyShareLink({ type: 'moment', id: momentId });
		shareStatus = result === 'failed' ? 'failed' : 'copied';
		moreOpen = false;
		window.setTimeout(() => (shareStatus = 'idle'), 1800);
	};

	return {
		get moment() { return moment; },
		get apiComments() { return apiComments; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get input() { return input; },
		set input(val: string) { input = val; },
		get moreOpen() { return moreOpen; },
		set moreOpen(val: boolean) { moreOpen = val; },
		get shareStatus() { return shareStatus; },
		get liked() { return liked; },
		get saved() { return saved; },
		get momentRaw() { return momentRaw; },
		get deleteOpen() { return deleteOpen; },
		set deleteOpen(val: boolean) { deleteOpen = val; },
		get editOpen() { return editOpen; },
		set editOpen(val: boolean) { editOpen = val; },
		get deleting() { return deleting; },
		get reportTarget() { return reportTarget; },
		set reportTarget(val: { target_type: 'moment' | 'comment'; target_id: string } | null) { reportTarget = val; },
		get commentDeleteTarget() { return commentDeleteTarget; },
		set commentDeleteTarget(val: string | null) { commentDeleteTarget = val; },
		get commentDeletePending() { return commentDeletePending; },
		get commentCount() { return commentCount; },
		get isAuthor() { return isAuthor; },
		get replyTarget() { return replyTarget; },
		get commentsNextCursor() { return commentsNextCursor; },
		get commentsLoadingMore() { return commentsLoadingMore; },
		get newCommentId() { return newCommentId; },
		get newReplyId() { return newReplyId; },
		get nearbyNow() { return nearbyNow; },
		get relatedMoments() { return relatedMoments; },
		reload() { reloadKey++; },
		startReply,
		cancelReply,
		expandReplies,
		handleComment,
		loadMoreComments,
		toggleCommentLike,
		handleCommentDelete,
		toggleLike,
		toggleSave,
		handleDelete,
		handleUpdated,
		openEditDialog,
		copyMomentLink
	};
}

export type MomentDetailData = ReturnType<typeof useMomentDetailData>;
