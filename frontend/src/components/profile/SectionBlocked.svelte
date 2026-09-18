<script lang="ts">
	import { ShieldBan } from '@lucide/svelte';
	import ProfileUserSection from '@/components/profile/ProfileUserSection.svelte';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { fetchBlocked, unblock, netError } from '@/api';
	import { formatDateTime } from '@/lib/utils';

	// 黑名单分区（/me/relationships 第四 tab）：users/fetch_blocked 拉取 +
	// 单条解除拉黑（users/set_block active=false，两态统一端点的 false 侧）。
	// 骨架与行样式对齐好友/关注/粉丝三个 Section（ProfileUserSection + UserCardRow）。

	// 行模型：ProfileUserSection 泛型要求 items 携带 id（spec user_id）
	interface BlockedRow {
		id: string;
		name: string;
		avatar: string;
		blockedAt: string;
	}

	let { embedded = false }: { embedded?: boolean } = $props();

	let rows = $state<BlockedRow[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let needsAuth = $state(false);
	let reloadKey = $state(0);

	let searchQ = $state('');
	const filtered = $derived(
		rows.filter((r) => searchQ === '' || r.name.toLowerCase().includes(searchQ.toLowerCase()))
	);

	// 解除拉黑确认 + in-flight 防连点
	let unblockTarget = $state<BlockedRow | null>(null);
	let unblockPending = $state(false);

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function load() {
			loading = true;
			error = null;
			needsAuth = false;
			try {
				const res = await fetchBlocked({ pagination: null }).catch(() => netError(null));
				if (cancelled) return;
				if (res.code === 10100) {
					needsAuth = true;
					return;
				}
				if (res.code === 0 && res.data) {
					rows = res.data.items.map((b) => ({
						id: b.user.user_id,
						name: b.user.nickname,
						avatar: b.user.avatar_url ?? '',
						blockedAt: b.blocked_at
					}));
				} else {
					error = res.message || '加载失败，请稍后重试';
				}
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void load();
		return () => {
			cancelled = true;
		};
	});

	const confirmUnblock = async () => {
		if (!unblockTarget || unblockPending) return;
		unblockPending = true;
		try {
			// spec: users/set_block（两态统一）；解除拉黑不自动恢复关注关系
			const res = await unblock({ user_id: unblockTarget.id });
			if (res.code === 0) {
				rows = rows.filter((r) => r.id !== unblockTarget?.id);
				showToast('已解除拉黑', 'success');
			} else {
				showToast(res.message || '操作失败，请重试', 'danger');
			}
		} catch {
			showToast('操作失败，请重试', 'danger');
		} finally {
			unblockPending = false;
			unblockTarget = null;
		}
	};
</script>

<ProfileUserSection
	title="黑名单"
	subtitle="被拉黑的用户无法关注你或向你发送消息"
	count={rows.length}
	countSuffix=" 人"
	{embedded}
	search={{ value: searchQ, onChange: (v) => (searchQ = v), placeholder: '搜索黑名单…' }}
	emptyTitle="黑名单为空"
	emptyDescription="在用户主页「更多操作」中拉黑的用户会出现在这里。"
	emptyIcon={ShieldBan}
	loading={loading}
	{error}
	onRetry={() => reloadKey++}
	{needsAuth}
	items={filtered}
>
	{#snippet children(b)}
		<UserCardRow
			name={b.name}
			avatar={b.avatar}
			tone={embedded ? 'card' : 'row'}
			ariaLabel={`查看 ${b.name} 的主页`}
		>
			{#snippet subtitle()}
				<p class="mt-0.5 text-text-secondary text-xs">拉黑于 {formatDateTime(b.blockedAt)}</p>
			{/snippet}
			{#snippet actions()}
				<button
					type="button"
					class="hover-btn tap-92 hit-slop flex shrink-0 items-center justify-center rounded-full border border-border-default bg-surface-primary px-2.5 py-1.5 text-sm text-text-secondary focus-ring hover:bg-accent-soft"
					onclick={(e) => {
						e.stopPropagation();
						unblockTarget = b;
					}}
				>
					解除拉黑
				</button>
			{/snippet}
		</UserCardRow>
	{/snippet}
</ProfileUserSection>

<ConfirmDialog
	open={unblockTarget !== null}
	title={`解除拉黑「${unblockTarget?.name ?? ''}」？`}
	description="解除后对方可以重新关注你、给你发送消息。"
	icon={ShieldBan}
	confirmVariant="primary"
	confirmLabel="解除拉黑"
	loading={unblockPending}
	onCancel={() => (unblockTarget = null)}
	onConfirm={() => void confirmUnblock()}
/>
