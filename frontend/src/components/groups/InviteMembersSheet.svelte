<script lang="ts">
	// 邀请成员入群：搜索用户（users/search）+ 多选 + 批量邀请（members/invite）。
	import * as Dialog from '@/components/ui/dialog';
	import { X, Search, UserPlus, Check } from '@lucide/svelte';
	import { searchUsers, inviteGroupMember } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import type { UserItem } from '@/api/types';

	let {
		open,
		groupId,
		onClose,
		onInvited
	}: {
		open: boolean;
		groupId: string;
		onClose: () => void;
		onInvited?: () => void;
	} = $props();

	let keyword = $state('');
	let results = $state<UserItem[]>([]);
	let loading = $state(false);
	let searchError = $state<string | null>(null);
	let selected = $state<Set<string>>(new Set());
	let inviting = $state(false);
	let searchTimer: ReturnType<typeof setTimeout> | null = null;
	// 竞态序号：防抖挡不住「慢的先发、快的后发」的乱序响应，
	// 只有最新一次搜索的结果允许落地，否则会覆盖成旧关键词的结果。
	let searchSeq = 0;

	const doSearch = async () => {
		const q = keyword.trim();
		if (!q) { results = []; searchError = null; return; }
		const seq = ++searchSeq;
		loading = true;
		searchError = null;
		try {
			const res = await searchUsers({ keyword: q, pagination: { limit: 20 } });
			if (seq !== searchSeq) return;
			if (res.code === 0) results = res.data?.items ?? [];
			else {
				results = [];
				searchError = res.message ?? '搜索失败';
			}
		} catch {
			if (seq !== searchSeq) return;
			// 失败必须清空结果：否则列表留着上一次关键词的命中项，
			// 用户会以为这是当前搜索的结果并误邀请
			results = [];
			searchError = '网络异常，请稍后重试';
		} finally {
			if (seq === searchSeq) loading = false;
		}
	};

	// 防抖输入
	const onInput = () => {
		if (searchTimer) clearTimeout(searchTimer);
		searchTimer = setTimeout(doSearch, 350);
	};

	// 打开时重置
	$effect(() => {
		// searchSeq++ 让仍在飞行的上一次搜索作废，避免关闭后回填结果
		if (open) { keyword = ''; results = []; searchError = null; searchSeq++; selected = new Set(); inviting = false; }
	});

	const toggle = (uid: string) => {
		const next = new Set(selected);
		next.has(uid) ? next.delete(uid) : next.add(uid);
		selected = next;
	};

	const handleInvite = async () => {
		if (selected.size === 0) return;
		inviting = true;
		const ids = Array.from(selected);
		try {
			const settled = await Promise.all(ids.map((uid) => inviteGroupMember({ group_id: groupId, user_id: uid }).catch(() => ({ code: -1 }))));
			const okCount = settled.filter((r) => r.code === 0).length;
			if (okCount === ids.length) showToast(`已邀请 ${okCount} 人`, 'success');
			else if (okCount > 0) showToast(`已邀请 ${okCount}/${ids.length} 人`, 'success');
			else showToast('邀请失败', 'danger');
			onInvited?.();
			onClose();
		} catch {
			showToast('邀请失败，请重试', 'danger');
		}
		inviting = false;
	};
</script>

<Dialog.Root open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content
		side="bottom"
		class="inset-x-4 bottom-4 mx-auto max-w-md rounded-xl bg-surface-primary p-4"
		aria-label="邀请成员"
		style="box-shadow:var(--cst-elevation-modal);"
	>
		<div class="flex items-center justify-between px-1 pb-3">
			<p class="text-md font-medium text-text-primary">邀请成员入群</p>
			<button type="button" onclick={onClose} class="hit-slop flex h-7 w-7 items-center justify-center rounded-full text-text-secondary outline-none hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default" aria-label="关闭"><X size={15} /></button>
		</div>

		<div class="flex items-center gap-2 input-surface rounded-md px-3 py-2.5">
			<Search size={14} class="text-text-secondary" strokeWidth={1.75} />
			<input bind:value={keyword} oninput={onInput} placeholder="搜索用户昵称…" class="flex-1 bg-transparent text-sm text-text-primary" />
		</div>

		<div class="mt-3 max-h-72 overflow-y-auto no-scrollbar pb-3">
			{#if loading}
				<p class="py-6 text-center text-sm text-text-secondary">搜索中…</p>
			{:else if searchError}
				<!-- 搜索失败必须与「无匹配用户」区分开：否则用户以为真的没人而放弃邀请。
				     py-8 收紧（LoadError 默认 py-20，会撑爆 max-h-72 的 sheet 列表） -->
				<LoadError message={searchError} onRetry={() => void doSearch()} class="py-8" />
			{:else if results.length === 0}
				<p class="py-6 text-center text-sm text-text-secondary">{keyword.trim() ? '无匹配用户' : '输入昵称搜索用户'}</p>
			{:else}
				{#each results as u (u.user.user_id)}
					<button type="button" onclick={() => toggle(u.user.user_id)} class="hover-btn flex w-full items-center gap-2.5 rounded-md px-2 py-2 text-left outline-none focus-visible:ring-2 focus-visible:ring-accent-default">
						<div class="relative">
							<Avatar src={u.user.avatar_url ?? undefined} name={u.user.nickname} size={36} />
							{#if selected.has(u.user.user_id)}
								<span class="absolute -bottom-0.5 -right-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-accent-default text-text-on-accent"><Check size={10} strokeWidth={3} /></span>
							{/if}
						</div>
						<span class="min-w-0 flex-1">
							<span class="block truncate text-sm text-text-primary">{u.user.nickname}</span>
						</span>
					</button>
				{/each}
			{/if}
		</div>

		<button type="button" disabled={selected.size === 0 || inviting} onclick={handleInvite} class="hover-btn mt-3 flex w-full items-center justify-center gap-2 rounded-full px-4 py-3 text-sm font-medium outline-none focus-visible:ring-2 focus-visible:ring-accent-default disabled:opacity-muted enabled:bg-accent-default enabled:text-text-on-accent">
			<UserPlus size={15} strokeWidth={1.75} />
			{inviting ? '邀请中…' : `邀请${selected.size > 0 ? ` ${selected.size} 人` : ''}`}
		</button>
	</Dialog.Content>
</Dialog.Root>
