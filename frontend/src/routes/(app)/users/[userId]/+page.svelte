<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, MessageCircle, UserPlus, UserCheck, Share2, Heart, Grid3X3, Bookmark, MoreHorizontal, Flag, PencilLine, UserX, X, Check, ArrowLeft } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import RailLocationMini from '@/components/rail/RailLocationMini.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import { getProfile, follow, block, createReport } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import * as Dialog from '@/components/ui/dialog';

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
		gallery: { moments: string[]; liked: string[]; saved: string[] };
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
		gallery: { moments: [], liked: [], saved: [] },
		tags: []
	});
	let loading = $state(true);
	let mutualFriends = $state<{ id: string; name: string; sub: string; avatar: string }[]>([]);
	let followed = $state(false);
	let activeTab = $state<'moments' | 'liked' | 'saved'>('moments');
	const galleryItems = $derived(user.gallery[activeTab]);
	let moreOpen = $state(false);
	let blocked = $state(false);
	let remark = $state('');
	let remarkDraft = $state('');
	let reportSent = $state(false);
	let shareCopied = $state(false);

	const uid = $derived(params.userId ?? '');

	$effect(() => {
		let cancelled = false;
		async function fetchProfile() {
			try {
				loading = true;
				const res = await getProfile(uid).catch(() => ({ code: -1, data: null }));
				if (cancelled) return;
				const data = res.code === 0 ? res.data : null;
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
						gallery: {
		moments: data.recent_moments?.items?.map((m) => m.images?.[0]?.url ?? '') ?? [],
		liked: data.liked_moments?.items?.map((m) => m.images?.[0]?.url ?? '') ?? [],
		saved: data.favorited_moments?.items?.map((m) => m.images?.[0]?.url ?? '') ?? []
	},
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
				// empty data is normal
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

	const toggleFollow = () => {
		if (blocked) return;
		const next = !followed;
		followed = next;
		// spec: follow/unfollow uses user_id + active boolean.
		follow({ user_id: uid, active: next }).catch(() => (followed = !next));
	};

	const toggleBlock = () => {
		const next = !blocked;
		blocked = next;
		followed = false;
		moreOpen = false;
		// spec: block/unblock uses user_id + active boolean.
		block({ user_id: uid, active: next }).catch(() => {
			blocked = !next;
			followed = next;
		});
	};

	const submitReport = () => {
		createReport({ target_type: 'user', target_id: uid, reason: 'other' })
			.then(() => (reportSent = true))
			.catch(() => {});
		moreOpen = false;
	};

	const copyProfileLink = () => {
		navigator.clipboard?.writeText(`${window.location.origin}${ROUTES.user(uid)}`);
		shareCopied = true;
		window.setTimeout(() => (shareCopied = false), 1600);
	};
</script>

<Dialog.Root open={moreOpen} onOpenChange={(next) => (next ? undefined : (moreOpen = false))}>
		<Dialog.Content
			side="bottom"
			class="inset-x-4 bottom-4 mx-auto max-w-md rounded-feature p-5"
			aria-label={`${user.name} 的更多操作`}
			style="box-shadow:0 24px 80px rgba(0,0,0,.28);"
		>
			<div class="flex items-center justify-between">
				<div>
					<p class="eyebrow">关系与安全</p>
					<p class="mt-1 text-[16px] text-text-primary">{user.name}</p>
				</div>
				<ActionIconButton icon={X} label="关闭" onClick={() => (moreOpen = false)} size={32} />
			</div>
				<label class="mt-5 block text-sm text-text-secondary">
					备注名称
					<input
						bind:value={remarkDraft}
						oninput={(e) => (remarkDraft = e.currentTarget.value.slice(0, 24))}
						placeholder="只对你可见"
						class="mt-2 w-full rounded-xl ring-1 ring-border-default bg-transparent px-3 py-2.5 text-base text-text-primary focus-ring"
					/>
				</label>
				<button
					type="button"
					class="hover-btn mt-2 flex w-full items-center justify-center gap-2 rounded-xl ring-1 ring-border-default py-2.5 text-base text-text-primary focus-ring"
					onclick={() => {
						remark = remarkDraft.trim();
						moreOpen = false;
					}}
				>
					<PencilLine size={14} />
					保存备注
				</button>
				<button
					type="button"
					class="hover-btn mt-2 flex w-full items-center justify-center gap-2 rounded-xl py-2.5 text-base text-status-danger outline-none focus-visible:ring-2 focus-visible:ring-status-danger"
					onclick={toggleBlock}
				>
					<UserX size={14} />
					{blocked ? '解除屏蔽' : '屏蔽用户'}
				</button>
				<button
					type="button"
					class="hover-btn mt-1 flex w-full items-center justify-center gap-2 rounded-xl py-2.5 text-base text-text-secondary focus-ring"
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

<SplitPageLayout contentClassName="pt-5 pb-6">
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
							<div class="overflow-hidden rounded-full size-20 shadow-neu-xl"   style="border:3px solid var(--surface-primary)">
								<img src={user.avatar} alt={user.name} class="w-full h-full object-cover" />
							</div>
							<span class="absolute bottom-1 right-1 rounded-full bg-status-online size-3.5 border-2 border-surface-primary"  ></span>
						</div>
						<div class="min-w-0 pt-1">
							<h1 class="truncate text-[26px] font-medium tracking-[-0.01em] text-text-primary">{user.name}</h1>
							<p class="mt-1 truncate text-text-secondary">{user.handle}{remark ? ` · ${remark}` : ''}</p>
						</div>
					</div>

					<!-- Action buttons -->
					<div class="flex shrink-0 gap-2 pb-1">
						<button
							type="button"
							class="hover-btn tap-93 flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none disabled:cursor-not-allowed disabled:opacity-45 bg-surface-primary text-text-secondary text-xs"
							style="border:1px solid var(--border-default);box-shadow:var(--shadow-neu-md);"
							disabled={blocked}
							onclick={() => !blocked && void goto(ROUTES.message(uid))}
						>
							<MessageCircle class="size-3.5"  strokeWidth={1.5} />
							私信
						</button>
						<button
							type="button"
							class="hover-btn tap-93 flex items-center gap-1.5 px-4 py-2 rounded-full outline-none transition-all disabled:cursor-not-allowed disabled:opacity-45 text-xs {followed ? 'bg-surface-primary text-accent-default' : 'bg-accent-default text-white'}"
							style={followed
								? 'border:1.5px solid var(--accent-default);box-shadow:var(--shadow-neu-md);'
								: 'box-shadow:0 4px 14px var(--glow-accent);border:1px solid transparent;'}
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
				<div class="mt-4 flex items-center gap-2 rounded-small ring-1 ring-status-danger/30 bg-danger-bg px-3 py-2 text-sm text-status-danger">
					<UserX size={14} />
					你已屏蔽此用户；关注和私信已停用。
				</div>
			{/if}
			<p class="mt-4 max-w-115 text-sm leading-6 text-text-secondary">{user.bio}</p>

			<!-- 位置 + 标签 -->
			<div class="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2">
				{#if user.location}
					<span class="flex items-center gap-1.5 text-text-secondary text-base">
						<MapPin class="size-3.25"  style="color:var(--accent-default)" strokeWidth={1.75} />
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

		<!-- Tabs（TabBar 统一下划线样式） -->
	<div class="mt-8">
		<TabBar
			items={[
				{ id: 'moments', icon: Grid3X3, label: '瞬间' },
				{ id: 'liked', icon: Heart, label: '喜欢' },
				{ id: 'saved', icon: Bookmark, label: '收藏' }
			]}
			value={activeTab}
			onChange={(id) => (activeTab = id as 'moments' | 'liked' | 'saved')}
		/>
	</div>

	<!-- Gallery -->
	<div class="mt-4 grid grid-cols-3 gap-1.5">
		{#each galleryItems as src, i}
			<div class="image-hover-frame tap-97 relative cursor-pointer rounded-small overflow-hidden" style="aspect-ratio:1;">
				<img src={src} alt={`photo ${i}`} class="image-hover-frame__image w-full h-full object-cover" />
				{#if activeTab === 'liked'}
					<div class="absolute inset-0 flex items-end justify-end p-1.5 bg-surface-overlay-soft">
						<div class="flex items-center justify-center rounded-full bg-accent-default size-5" >
							<Heart class="size-2.5"  style="color:var(--text-on-accent)" strokeWidth={2} fill="white" />
						</div>
					</div>
				{/if}
			</div>
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
