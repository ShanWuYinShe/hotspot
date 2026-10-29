<script lang="ts">
	// 邀请成员入群：搜索用户（users/search）+ 多选 + 批量邀请（members/invite）。
	import * as Dialog from '@/components/ui/dialog';
	import { X, Search, UserPlus, Check } from '@lucide/svelte';
	import { searchUsers, inviteGroupMember, netError } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import Button from '@/components/ui/button.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
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
			const settled = await Promise.all(ids.map((uid) => inviteGroupMember({ group_id: groupId, user_id: uid }).catch(() => netError(undefined))));
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
	<Dialog.Content side="bottom" floating class="p-4" aria-label="邀请成员">
		<div class="flex items-center justify-between px-1 pb-3">
			<div class="min-w-0">
				<p class="text-md font-semibold text-text-primary">邀请成员入群</p>
				<p class="mt-0.5 text-xs text-text-secondary">搜索昵称后可多选邀请</p>
			</div>
			<button type="button" onclick={onClose} class="hit-slop flex size-8 shrink-0 items-center justify-center rounded-full text-text-secondary focus-ring hover:bg-interactive-hover" aria-label="关闭"><X size={15} strokeWidth={1.75} /></button>
		</div>

		<SearchInput
			value={keyword}
			onChange={(v) => {
				keyword = v;
				onInput();
			}}
			placeholder="搜索用户昵称…"
			ariaLabel="搜索用户昵称"
		/>

		<div class="mt-3 max-h-72 overflow-y-auto no-scrollbar pb-3">
			{#if loading}
				<InlineLoading label="搜索中…" class="py-6" />
			{:else if searchError}
				<!-- 搜索失败必须与「无匹配用户」区分开：否则用户以为真的没人而放弃邀请。
				     py-8 收紧（LoadError 默认 py-20，会撑爆 max-h-72 的 sheet 列表） -->
				<LoadError message={searchError} onRetry={() => void doSearch()} class="py-8" />
			{:else if results.length === 0}
				<EmptyState
					bare
					title={keyword.trim() ? '没有找到匹配的用户，换个昵称试试' : '输入昵称搜索用户'}
					class="py-6"
				>
					{#snippet icon()}
						<Search size={22} class="text-text-secondary" strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			{:else}
				{#each results as u (u.user.user_id)}
					{@const isSelected = selected.has(u.user.user_id)}
					<!-- M3 checkbox 列表范式：选中行底色变化 + 尾部 checkbox 双态；
					     hover 差异化——未选中 hover 显示「将选中」淡底预览，选中 hover 只加深一档 -->
					<button
						type="button"
						onclick={() => toggle(u.user.user_id)}
						aria-pressed={isSelected}
						class="hover-btn flex w-full items-center gap-2.5 rounded-md px-2 py-2 text-left transition-colors duration-rapid ease-standard outline-none focus-visible:ring-2 focus-visible:ring-accent-default
							{isSelected ? 'bg-accent-soft hover:bg-interactive-active' : 'hover:bg-interactive-hover'}"
					>
						<Avatar src={u.user.avatar_url ?? undefined} name={u.user.nickname} size="md" />
						<span class="min-w-0 flex-1">
							<span class="block truncate text-md font-medium text-text-primary">{u.user.nickname}</span>
						</span>
						<span
							class="flex size-4.5 shrink-0 items-center justify-center rounded-full transition-colors duration-rapid ease-standard
								{isSelected ? 'bg-accent-default text-text-on-accent' : 'border border-border-strong text-transparent'}"
						>
							<Check size={11} strokeWidth={3} />
						</span>
					</button>
				{/each}
			{/if}
		</div>

		<Button
			type="button"
			variant="primary"
			size="lg"
			class="mt-3 w-full"
			disabled={selected.size === 0}
			loading={inviting}
			onclick={() => void handleInvite()}
		>
			<UserPlus size={15} strokeWidth={1.75} />
			{selected.size > 0 ? `邀请 ${selected.size} 人` : '邀请'}
		</Button>
	</Dialog.Content>
</Dialog.Root>
