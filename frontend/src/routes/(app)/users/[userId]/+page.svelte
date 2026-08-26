<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, MessageCircle, UserPlus, UserCheck, Share2, MoreHorizontal, Flag, PencilLine, UserX, X, Check, ArrowLeft } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import RailLocationMini from '@/components/rail/RailLocationMini.svelte';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import { openTheDoor, follow, block, createReport, setRemark } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import * as Dialog from '@/components/ui/dialog';
	import { showToast } from '@/stores/toast.svelte';
	import { requireAuth } from '@/stores/auth.svelte';

	// 用户详情：关注/私信/屏蔽/举报/备注 + 瞬间画廊 + 桌面 Rail。

	let { params }: { params: { userId: string } } = $props();

	interface UserData {
		name: string;
		handle: string;
		bio: string;
		location: string;
		avatar: string;
		followers: number;
		following: number;
		moments: number;
		galleryMoments: Array<{ id: string; url: string }>;
		tags: string[];
	}

	let user = $state<UserData>({
		name: '',
		handle: '',
		bio: '',
		location: '',
		avatar: '',
		followers: 0,
		following: 0,
		moments: 0,
		galleryMoments: [],
		tags: []
	});
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);
	let mutualFriends = $state<{ id: string; name: string; sub: string; avatar: string }[]>([]);
	let followed = $state(false);
	let moreOpen = $state(false);
	let blocked = $state(false);
	let remark = $state('');
	let remarkDraft = $state('');
	let savingRemark = $state(false);
	let reportSent = $state(false);
	let shareCopied = $state(false);

	const uid = $derived(params.userId ?? '');

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchProfile() {
			try {
				loading = true;
				loadError = null;
				const res = await openTheDoor(uid, true).catch(() => ({ code: -1, message: '网络异常', data: null }));
				if (cancelled) return;
				const data = res.code === 0 ? res.data : null;
				if (res.code !== 0) {
					if (!cancelled) loadError = res.message ?? '加载失败';
					return;
				}
				if (data) {
					const u = data.user;
					const s = u.stats;
					const r = u.relation;
					user = {
						name: u.user.nickname,
						handle: u.user.user_id ? `@${u.user.nickname.replace(/\s/g, '_')}` : '',
						bio: u.bio ?? '',
						location: u.location?.location_name ?? '',
						avatar: u.user.avatar_url ?? '',
						followers: s?.followers_count ?? 0,
						following: s?.following_count ?? 0,
						moments: s?.moments_count ?? 0,
						galleryMoments:
	// 只展示带图瞬间（无图动态渲染 <img src=""> 会破图）；保留 moment_id 供跳转详情
	data.recent_moments?.items
		?.filter((m) => m.images?.[0]?.url)
		.map((m) => ({ id: m.moment_id, url: m.images![0]!.url })) ?? [],
						tags: u.tags ?? []
					};
					if (r?.is_following !== undefined) followed = r.is_following;
					if (r?.is_blocked !== undefined) blocked = r.is_blocked;

					const mutual = u.social_context?.mutual_friends_preview;
					if (mutual) {
						mutualFriends = mutual.map((f) => ({
							id: f.user_id,
							name: f.nickname,
							sub: '',
							avatar: f.avatar_url ?? ''
						}));
					}
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchProfile();
		return () => {
			cancelled = true;
		};
	});

	const fmtNum = (n: number) => (n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n));

	// in-flight 去重（防连点双翻转）；业务失败（code≠0）显式判 code 回滚
	let followPending = false;
	let blockPending = false;

	const toggleFollow = async () => {
		if (blocked || followPending) return;
		if (!requireAuth({ action: '关注', targetId: uid, run: () => void toggleFollow() })) return;
		const next = !followed;
		const prevFollowed = followed;
		followed = next;
		followPending = true;
		try {
			// spec: follow/unfollow uses user_id + active boolean.
			const res = await follow({ user_id: uid, active: next });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			followed = prevFollowed;
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			followPending = false;
		}
	};

	const toggleBlock = async () => {
		if (blockPending) return;
		if (!requireAuth({ action: '屏蔽', targetId: uid, run: () => void toggleBlock() })) return;
		const next = !blocked;
		const prevBlocked = blocked;
		const prevFollowed = followed;
		blocked = next;
		followed = false;
		moreOpen = false;
		blockPending = true;
		try {
			// spec: block/unblock uses user_id + active boolean.
			const res = await block({ user_id: uid, active: next });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			blocked = prevBlocked;
			followed = prevFollowed;
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			blockPending = false;
		}
	};

	const submitReport = () => {
		if (!requireAuth({ action: '举报', targetId: uid, run: submitReport })) return;
		createReport({ target_type: 'user', target_id: uid, reason: 'other' })
			.then((res) => {
				if (res.code === 0) reportSent = true;
				else showToast(res.message || '举报失败', 'danger');
			})
			.catch(() => showToast('举报失败，请重试', 'danger'));
		moreOpen = false;
	};

	const copyProfileLink = () => {
		navigator.clipboard?.writeText(`${window.location.origin}${ROUTES.user(uid)}`);
		shareCopied = true;
		window.setTimeout(() => (shareCopied = false), 1600);
	};
</script>
<svelte:head>
	<title>用户主页 · HotSpot</title>
</svelte:head>


<Dialog.Root open={moreOpen} onOpenChange={(next) => (next ? undefined : (moreOpen = false))}>
		<Dialog.Content
			side="bottom"
			class="inset-x-4 bottom-4 mx-auto max-w-md rounded-xl p-5"
			aria-label={`${user.name} 的更多操作`}
			style="box-shadow:var(--cst-elevation-modal);"
		>
			<div class="flex items-center justify-between">
				<div>
					<p class="eyebrow">关系与安全</p>
					<p class="mt-1 text-md text-text-primary">{user.name}</p>
				</div>
				<ActionIconButton icon={X} label="关闭" onClick={() => (moreOpen = false)} />
			</div>
				<label class="mt-5 block text-sm text-text-secondary">
					备注名称
					<input
						bind:value={remarkDraft}
						oninput={(e) => (remarkDraft = e.currentTarget.value.slice(0, 24))}
						placeholder="只对你可见"
						class="mt-2 w-full input-surface rounded-md px-3 py-2.5 text-sm text-text-primary"
					/>
				</label>
			<button
				type="button"
				disabled={!followed || savingRemark}
				class="hover-btn mt-2 flex w-full items-center justify-center gap-2 rounded-sm border border-border-default py-2.5 text-sm text-text-primary focus-ring disabled:opacity-muted"
				onclick={async () => {
					if (!followed) return;
					savingRemark = true;
					try {
						const res = await setRemark({ user_id: uid, remark: remarkDraft.trim() });
						if (res.code === 0) {
							remark = remarkDraft.trim();
							moreOpen = false;
							showToast('备注已保存', 'success');
						} else {
							showToast(res.message || '保存失败', 'danger');
						}
					} catch {
						showToast('保存失败，请重试', 'danger');
					} finally {
						savingRemark = false;
					}
				}}
			>
				<PencilLine size={14} />
				{savingRemark ? '保存中…' : '保存备注'}
			</button>
			{#if !followed}
				<p class="mt-1 text-xs text-text-secondary">关注对方后可设备注</p>
			{/if}
				<button
					type="button"
					class="hover-btn mt-2 flex w-full items-center justify-center gap-2 rounded-sm py-2.5 text-sm text-status-danger outline-none focus-visible:ring-2 focus-visible:ring-status-danger"
					onclick={toggleBlock}
				>
					<UserX size={14} />
					{blocked ? '解除屏蔽' : '屏蔽用户'}
				</button>
				<button
					type="button"
					class="hover-btn mt-1 flex w-full items-center justify-center gap-2 rounded-sm py-2.5 text-sm text-text-secondary focus-ring"
					onclick={submitReport}
				>
					{#if reportSent}
						<Check size={14} />
						举报已记录
					{:else}
						<Flag size={14} />
						举报用户
					{/if}
				</button>
		</Dialog.Content>
	</Dialog.Root>

{#if loadError}
	<LoadError message={loadError} onRetry={() => reloadKey++} />
{:else}
<SplitPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	<!-- 资料头（无封面数据：紧凑头部，返回/分享/更多 浮在首行，对齐动态详情作者行层级） -->
	<div class="flex items-center justify-between">
		<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} />
		<div class="flex gap-2">
			<ActionIconButton icon={Share2} label={shareCopied ? '链接已复制' : '分享资料'} onClick={copyProfileLink} />
			<ActionIconButton
				icon={MoreHorizontal}
				label="更多操作"
				onClick={() => {
					remarkDraft = remark;
					moreOpen = true;
				}}
			/>
		</div>
	</div>

	<!-- Avatar + 身份（gutter 由 SplitPageLayout 提供） -->
	<div class="mt-5 flex flex-col pb-5">
				<div class="flex items-end justify-between gap-4">
					<div class="flex min-w-0 items-center gap-4">
						<div class="relative shrink-0">
							<Avatar src={user.avatar} name={user.name} size={80} class="shadow-raised-lg" style="border:var(--cst-border-width-heavy) solid var(--cst-surface-primary);" />
							<span class="absolute bottom-1 right-1 rounded-full bg-status-online size-3.5 border-2 border-surface-primary"  ></span>
						</div>
						<div class="min-w-0 pt-1">
							<h1 class="truncate text-2xl font-medium tracking-title text-text-primary">{user.name}</h1>
							<p class="mt-1 truncate text-text-secondary">{user.handle}{remark ? ` · ${remark}` : ''}</p>
						</div>
					</div>

					<!-- Action buttons -->
					<div class="flex shrink-0 gap-2 pb-1">
						<button
							type="button"
							class="hover-btn tap-92 flex items-center gap-1.5 rounded-full border border-border-default bg-surface-primary px-3.5 py-2 text-xs text-text-secondary shadow-raised-sm focus-ring disabled:cursor-not-allowed disabled:opacity-muted"
							disabled={blocked}
							onclick={() => !blocked && void goto(ROUTES.message(uid))}
						>
							<MessageCircle class="size-3.5"  strokeWidth={1.5} />
							私信
						</button>
						<button
							type="button"
							class="hover-btn tap-92 flex items-center gap-1.5 px-4 py-2 rounded-full focus-ring transition-all disabled:cursor-not-allowed disabled:opacity-muted text-xs {followed ? 'bg-surface-primary text-accent-default' : 'bg-accent-default text-text-on-accent'}"
							style={followed
								? 'border:var(--cst-border-width-regular) solid var(--cst-accent-default);box-shadow:var(--shadow-raised-sm);'
								: 'box-shadow:var(--shadow-glow-md);border:var(--cst-border-width-hairline) solid transparent;'}
							disabled={blocked}
							onclick={toggleFollow}
						>
							{#if followed}
								<UserCheck class="size-3.5"  strokeWidth={1.5} />
								已关注
							{:else}
								<UserPlus class="size-3.5"  strokeWidth={1.5} />
								关注
							{/if}
						</button>
					</div>
				</div>

			{#if blocked}
				<div class="mt-5 flex items-center gap-2 rounded-sm ring-1 ring-status-danger/30 bg-danger-bg px-3 py-2 text-sm text-status-danger">
					<UserX size={14} />
					你已屏蔽此用户；关注和私信已停用。
				</div>
			{/if}
			<p class="mt-5 max-w-115 text-sm leading-6 text-text-secondary">{user.bio}</p>

			<!-- 位置 + 标签 -->
			<div class="mt-5 flex flex-wrap items-center gap-x-4 gap-y-2">
				{#if user.location}
					<span class="flex items-center gap-1.5 text-text-secondary text-sm">
						<MapPin class="size-3.5 text-accent-default"   strokeWidth={1.75} />
						{user.location}
					</span>
				{/if}
				{#each user.tags as t (t)}
					<span class="rounded-full bg-accent-soft px-2.5 py-0.5 text-xs text-accent-default border border-accent-border" >
						#{t}
					</span>
				{/each}
			</div>

			<!-- Stats -->
			<div class="mt-6">
				<MetricRow
					items={[
						{ label: '瞬间', value: fmtNum(user.moments) },
						{ label: '粉丝', value: fmtNum(user.followers) },
						{ label: '关注', value: fmtNum(user.following) }
					]}
				/>
			</div>

	<!-- Gallery（spec UserProfileBundle 仅含 recent_moments，无喜欢/收藏流）。
	     整格跳转动态详情（大图查看在详情页内）。 -->
	<div class="mt-5 grid grid-cols-3 gap-1.5">
		{#each user.galleryMoments as g, i (g.id)}
			<button
				type="button"
				class="image-hover-frame tap-97 relative cursor-pointer rounded-sm overflow-hidden focus-ring"
				style="aspect-ratio:1;"
				aria-label={`查看瞬间 ${i + 1}`}
				onclick={() => void goto(ROUTES.moment(g.id))}
			>
				<img src={g.url} alt={`photo ${i}`} loading="lazy" class="image-hover-frame__image w-full h-full object-cover" />
			</button>
		{/each}
	</div>
</div>

	{#snippet rail()}
	<RailSection title="所在位置">
		<RailLocationMini name={user.location} />
	</RailSection>

	<RailSection title="兴趣标签">
		<div class="flex gap-2 flex-wrap">
			{#each user.tags as t (t)}
				<span class="px-2.5 py-0.5 rounded-full bg-accent-soft text-accent-default text-xs border border-accent-border" >
					#{t}
				</span>
			{/each}
		</div>
	</RailSection>

	{#if mutualFriends.length > 0}
		<RailSection title="共同好友">
			<div class="flex flex-col gap-3">
				{#each mutualFriends as f (f.id)}
					<RailUserRow avatar={f.avatar} name={f.name} sub={f.sub} onClick={() => void goto(ROUTES.user(f.id))} />
				{/each}
			</div>
		</RailSection>
	{/if}
	{/snippet}
</SplitPageLayout>
{/if}

