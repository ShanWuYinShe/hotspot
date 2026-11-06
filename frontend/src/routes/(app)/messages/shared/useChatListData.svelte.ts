import { untrack } from 'svelte';
import { page } from '$app/state';
import { goto } from '$app/navigation';
import { getPrivateChats, deletePrivateChat, listMyGroups, markPrivateRead, markGroupRead, updatePrivateChatPreferences, updateGroupPreferences, netError } from '@/api';
import { showToast } from '@/stores/toast.svelte';
import { formatChatListTime } from '@/lib/chat.svelte';
import { compareChatItems, type ChatItem } from '@/lib/chat-list';
import { presence, watchPresence, seedOnline } from '@/stores/presence.svelte';
import { setChatUnread } from '@/stores/unread.svelte';
import { WS_PRIVATE_MESSAGE_EVENT, WS_GROUP_MESSAGE_EVENT } from '@/lib/ws/client';
import { canUseSplitPane, resolvePushChatHref } from './deep-link';
import { platform } from '@/stores/platform.svelte';

export function useChatListData() {
	let chats = $state<ChatItem[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let needsAuth = $state(false);
	let reloadKey = $state(0);
	let unwatchPresence: (() => void) | null = null;

	let searchQuery = $state('');
	let showNewChat = $state(false);
	let newChatTab = $state<'add' | 'chat'>('chat');
	let showCreateGroup = $state(false);
	let selectedId = $state<number | string | null>(null);
	let deleteTarget = $state<ChatItem | null>(null);
	let deletingChat = $state(false);
	let moreTarget = $state<ChatItem | null>(null);

	const queryPeer = $derived(page.url.searchParams.get('peer'));
	const queryGroup = $derived(page.url.searchParams.get('group'));
	const useSplitPane = $derived(canUseSplitPane(platform));

	// WS 新消息事件监听（防抖 250ms，避免群聊高频连发时引发瞬时并发 HTTP 请求风暴）
	$effect(() => {
		let timer: ReturnType<typeof setTimeout> | null = null;
		const onWsMessage = () => {
			if (timer) clearTimeout(timer);
			timer = setTimeout(() => {
				reloadKey++;
			}, 250);
		};
		window.addEventListener(WS_PRIVATE_MESSAGE_EVENT, onWsMessage);
		window.addEventListener(WS_GROUP_MESSAGE_EVENT, onWsMessage);
		return () => {
			if (timer) clearTimeout(timer);
			window.removeEventListener(WS_PRIVATE_MESSAGE_EVENT, onWsMessage);
			window.removeEventListener(WS_GROUP_MESSAGE_EVENT, onWsMessage);
		};
	});

	// 拉取会话列表
	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchChats() {
			try {
				if (untrack(() => !chats.length)) loading = true;
				loadError = null;
				needsAuth = false;

				const [privRes, groupRes] = await Promise.all([
					getPrivateChats().catch(() => netError({ items: [] })),
					listMyGroups().catch(() => netError({ items: [] }))
				]);
				if (cancelled) return;

				const privData = privRes.code === 0 && privRes.data ? privRes.data.items : [];
				const groupData = groupRes.code === 0 && groupRes.data ? groupRes.data.items : [];

				if (privRes.code === 10100 || groupRes.code === 10100) {
					if (!cancelled) needsAuth = true;
					return;
				}
				if (privRes.code !== 0 || groupRes.code !== 0) {
					if (!cancelled) loadError = privRes.message ?? groupRes.message ?? '加载失败';
					return;
				}

				seedOnline(privData.filter((c) => c.presence_status === 'online').map((c) => c.other_user?.user_id ?? ''));

				const privateChats: ChatItem[] = privData.map((c, i) => ({
					id: `chat-${i}`,
					peerUserId: c.other_user?.user_id ?? '',
					name: c.other_user?.nickname ?? '',
					avatar: c.other_user?.avatar_url ?? '',
					lastMessage: c.last_message?.content ?? '',
					time: formatChatListTime(c.last_message?.sent_at ?? c.last_activity_at),
					timestamp: c.last_message?.sent_at ?? c.last_activity_at ?? '',
					unread: c.unread_count ?? 0,
					isPinned: c.is_pinned ?? false,
					isMuted: c.is_muted ?? false,
					online: c.presence_status === 'online',
					kind: 'private'
				}));

				const groupChats: ChatItem[] = groupData.map((c, i) => {
					const g = c.group;
					return {
						id: g.group_id ?? `group-${i}`,
						peerUserId: '',
						name: g.name,
						avatar: g.avatar_url ?? '',
						lastMessage: c.last_message?.content ?? '',
						time: formatChatListTime(c.last_message?.sent_at ?? c.last_activity_at ?? ''),
						timestamp: c.last_message?.sent_at ?? c.last_activity_at ?? '',
						unread: c.viewer_state?.unread_count ?? 0,
						isPinned: c.viewer_state?.is_pinned ?? false,
						isMuted: c.viewer_state?.is_muted ?? false,
						online: false,
						group: true,
						kind: 'group'
					};
				});

				const items = [...privateChats, ...groupChats].sort(compareChatItems);
				chats = items;

				unwatchPresence?.();
				unwatchPresence = watchPresence(
					items.filter((c) => c.kind !== 'group').map((c) => c.peerUserId).filter(Boolean)
				);
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}

		void fetchChats();
		return () => {
			cancelled = true;
			unwatchPresence?.();
			unwatchPresence = null;
		};
	});

	// 实时在线判定
	const liveChats = $derived(
		chats.map((c) => ({ ...c, online: presence.isOnline(c.peerUserId) }))
	);

	const filtered = $derived(
		liveChats.filter((c) => c.name.toLowerCase().includes(searchQuery.toLowerCase()))
	);
	const onlineChats = $derived(liveChats.filter((c) => c.online));
	const totalUnread = $derived(chats.reduce((s, c) => s + c.unread, 0));

	$effect(() => {
		setChatUnread(totalUnread);
	});

	// 深链落单栏推入页
	$effect(() => {
		if (useSplitPane) return;
		const href = resolvePushChatHref(queryPeer, queryGroup);
		if (href) void goto(href, { replaceState: true });
	});

	const selectedChat = $derived(
		liveChats.find((c) => {
			if (selectedId !== null && c.id === selectedId) return true;
			if (queryGroup && c.kind === 'group' && String(c.id) === queryGroup) return true;
			if (queryPeer && c.kind !== 'group' && (c.peerUserId === queryPeer || String(c.id) === queryPeer)) return true;
			return false;
		})
	);

	// 桌面端深链参数自动对齐选中行
	$effect(() => {
		if (!useSplitPane) return;
		if (queryGroup) {
			const found = liveChats.find((c) => c.kind === 'group' && String(c.id) === queryGroup);
			if (found && selectedId !== found.id) {
				selectedId = found.id;
			}
		} else if (queryPeer) {
			const found = liveChats.find((c) => c.kind !== 'group' && (c.peerUserId === queryPeer || String(c.id) === queryPeer));
			if (found && selectedId !== found.id) {
				selectedId = found.id;
			}
		}
	});

	const selectChat = (chat: ChatItem) => {
		if (selectedId !== chat.id) selectedId = chat.id;
		if (useSplitPane) {
			const targetParam = chat.kind === 'group' ? `group=${encodeURIComponent(String(chat.id))}` : `peer=${encodeURIComponent(chat.peerUserId || String(chat.id))}`;
			const currentParam = chat.kind === 'group' ? (queryGroup ? `group=${queryGroup}` : '') : (queryPeer ? `peer=${queryPeer}` : '');
			if (targetParam !== currentParam) {
				void goto(`?${targetParam}`, { replaceState: true, noScroll: true, keepFocus: true });
			}
		}
		if (chat.unread > 0) {
			chats = chats.map((c) => (c.id === chat.id ? { ...c, unread: 0 } : c)).sort(compareChatItems);
			if (chat.kind === 'group') {
				void markGroupRead({ scope: 'group', group_id: String(chat.id) }).catch(() => {});
			} else {
				void markPrivateRead({ scope: 'session', target_user_id: chat.peerUserId || String(chat.id) }).catch(() => {});
			}
		}
	};

	const togglePin = (chat: ChatItem) => {
		const next = !chat.isPinned;
		chats = chats.map((c) => (c.id === chat.id ? { ...c, isPinned: next } : c)).sort(compareChatItems);
		const req =
			chat.kind === 'group'
				? updateGroupPreferences({ group_id: String(chat.id), is_pinned: next })
				: updatePrivateChatPreferences({ target_user_id: chat.peerUserId || String(chat.id), is_pinned: next });
		void req
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message || '设置失败');
			})
			.catch(() => {
				chats = chats.map((c) => (c.id === chat.id ? { ...c, isPinned: !next } : c)).sort(compareChatItems);
				showToast('置顶设置失败，请重试', 'danger');
			});
	};

	const toggleMute = (chat: ChatItem) => {
		const next = !chat.isMuted;
		chats = chats.map((c) => (c.id === chat.id ? { ...c, isMuted: next } : c));
		const req =
			chat.kind === 'group'
				? updateGroupPreferences({ group_id: String(chat.id), is_muted: next })
				: updatePrivateChatPreferences({ target_user_id: chat.peerUserId || String(chat.id), is_muted: next });
		void req
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message || '设置失败');
			})
			.catch(() => {
				chats = chats.map((c) => (c.id === chat.id ? { ...c, isMuted: !next } : c));
				showToast('免打扰设置失败，请重试', 'danger');
			});
	};

	const handleDeleteChat = async () => {
		const chat = deleteTarget;
		if (!chat || deletingChat) return;
		deletingChat = true;
		try {
			const res = await deletePrivateChat({ target_user_id: chat.peerUserId || String(chat.id) });
			if (res.code === 0) {
				chats = chats.filter((c) => c.id !== chat.id);
				if (selectedId === chat.id) selectedId = null;
				showToast('会话已删除', 'success');
			} else {
				showToast(res.message || '删除失败，请重试', 'danger');
			}
		} catch {
			showToast('删除失败，请重试', 'danger');
		} finally {
			deletingChat = false;
			deleteTarget = null;
		}
	};

	return {
		get chats() { return chats; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get needsAuth() { return needsAuth; },
		get filtered() { return filtered; },
		get onlineChats() { return onlineChats; },
		get totalUnread() { return totalUnread; },
		get queryPeer() { return queryPeer; },
		get queryGroup() { return queryGroup; },
		get selectedId() { return selectedId; },
		set selectedId(val: number | string | null) { selectedId = val; },
		get selectedChat() { return selectedChat; },
		get searchQuery() { return searchQuery; },
		set searchQuery(val: string) { searchQuery = val; },
		get showNewChat() { return showNewChat; },
		set showNewChat(val: boolean) { showNewChat = val; },
		get newChatTab() { return newChatTab; },
		set newChatTab(val: 'add' | 'chat') { newChatTab = val; },
		get showCreateGroup() { return showCreateGroup; },
		set showCreateGroup(val: boolean) { showCreateGroup = val; },
		get deleteTarget() { return deleteTarget; },
		set deleteTarget(val: ChatItem | null) { deleteTarget = val; },
		get deletingChat() { return deletingChat; },
		get moreTarget() { return moreTarget; },
		set moreTarget(val: ChatItem | null) { moreTarget = val; },
		reload() { reloadKey++; },
		selectChat,
		togglePin,
		toggleMute,
		handleDeleteChat
	};
}

export type ChatListData = ReturnType<typeof useChatListData>;
