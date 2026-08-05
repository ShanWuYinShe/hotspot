<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, MessageCircle, UserPlus, UserCheck, Share2, Heart, Grid3X3, Bookmark, MoreHorizontal, Flag, PencilLine, UserX, X, Check, ArrowLeft } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import RailSection from '@/components/layout/desktop/RailSection.svelte';
	import RailUserRow from '@/components/layout/desktop/RailUserRow.svelte';
	import RailLocationMini from '@/components/layout/desktop/RailLocationMini.svelte';
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
		banner: string;
		followers: number;
		following: number;
		moments: number;
		gallery: string[];
		tags: string[];
	}

	let user = $state<UserData>({
		name: '',
		handle: '',
		bio: '',
		location: '',
		avatar: '',
		banner: '',
		followers: 0,
		following: 0,
		moments: 0,
		gallery: [],
		tags: []
	});
	let loading = $state(true);
	let mutualFriends = $state<{ id: string; name: string; sub: string; avatar: string }[]>([]);
	let similarUsers = $state<{ id: number; name: string; sub: string; avatar: string }[]>([]);
	let followed = $state(false);
	let activeTab = $state<'moments' | 'liked' | 'saved'>('moments');
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
						banner: '',
						followers: s?.followers_count ?? 0,
						following: s?.following_count ?? 0,
						moments: s?.moments_count ?? 0,
						gallery: data.recent_moments?.items?.map((m) => m.images?.[0]?.url ?? '') ?? [],
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
			class="inset-x-4 bottom-4 mx-auto max-w-md rounded-[24px] p-5"
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
				<label class="mt-5 block text-[12px] text-text-secondary">
					备注名称
					<input
						bind:value={remarkDraft}
						oninput={(e) => (remarkDraft = e.currentTarget.value.slice(0, 24))}
						placeholder="只对你可见"
						class="mt-2 w-full rounded-[14px] ring-1 ring-border-default bg-transparent px-3 py-2.5 text-[13px] text-text-primary focus-ring"
					/>
				</label>
				<button
					type="button"
					class="hover-btn mt-2 flex w-full items-center justify-center gap-2 rounded-[14px] ring-1 ring-border-default py-2.5 text-[13px] text-text-primary focus-ring"
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
					class="hover-btn mt-2 flex w-full items-center justify-center gap-2 rounded-[14px] py-2.5 text-[13px] text-status-danger outline-none focus-visible:ring-2 focus-visible:ring-status-danger"
					onclick={toggleBlock}
				>
					<UserX size={14} />
					{blocked ? '解除屏蔽' : '屏蔽用户'}
				</button>
				<button
					type="button"
					class="hover-btn mt-1 flex w-full items-center justify-center gap-2 rounded-[14px] py-2.5 text-[13px] text-text-secondary focus-ring"
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

<SplitPageLayout contentClassName="pb-6">
	<!-- Banner（全幅封面：负 margin 抵消 SplitPageLayout 的 gutter） -->
	<div class="relative overflow-hidden -mx-[var(--page-gutter)]" style="height:clamp(150px, 25vw, 210px);flex-shrink:0;">
			<img src={user.banner} alt="banner" class="absolute inset-0 w-full h-full object-cover" />
			<div class="absolute inset-0" style="background:var(--surface-profile-cover-scrim);"></div>

			<!-- Top buttons -->
			<div class="absolute top-0 left-0 right-0 flex items-center justify-between px-4 pt-4">
				<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => history.back()} variant="overlay" tone="inverse" />
				<div class="flex gap-2">
					<ActionIconButton icon={Share2} label={shareCopied ? '链接已复制' : '分享资料'} onClick={copyProfileLink} variant="overlay" tone="inverse" />
					<ActionIconButton
						icon={MoreHorizontal}
						label="更多操作"
						onClick={() => {
							remarkDraft = remark;
							moreOpen = true;
						}}
						variant="overlay"
						tone="inverse"
					/>
				</div>
			</div>
		</div>

	<!-- Avatar + info（gutter 由 SplitPageLayout 提供） -->
	<div class="-mt-10 pb-4 flex flex-col">
				<div class="flex items-end justify-between mb-3">
					<div class="relative">
						<div class="rounded-[22px] overflow-hidden" style="width:76px;height:76px;border:3px solid var(--surface-primary);box-shadow:0 6px 20px rgba(0,0,0,0.22);">
							<img src={user.avatar} alt={user.name} class="w-full h-full object-cover" />
						</div>
						<span class="absolute bottom-1 right-1 rounded-full bg-status-online" style="width:12px;height:12px;border:2px solid var(--surface-primary);"></span>
					</div>

					<!-- Action buttons -->
					<div class="flex gap-2 pb-1">
						<button
							type="button"
							class="hover-btn tap-93 flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none disabled:cursor-not-allowed disabled:opacity-45 bg-surface-primary text-text-secondary text-xs"
							style="border:1px solid var(--border-default);box-shadow:4px 4px 10px var(--elevation-shadow-dark), -4px -4px 10px var(--elevation-shadow-light);"
							disabled={blocked}
							onclick={() => !blocked && void goto(ROUTES.message(1))}
						>
							<MessageCircle style="width:14px;height:14px;" strokeWidth={1.5} />
							私信
						</button>
						<button
							type="button"
							class="hover-btn tap-93 flex items-center gap-1.5 px-4 py-2 rounded-full outline-none transition-all disabled:cursor-not-allowed disabled:opacity-45 text-xs {followed ? 'bg-surface-primary text-accent-default' : 'bg-accent-default text-white'}"
							style={followed
								? 'border:1.5px solid var(--accent-default);box-shadow:4px 4px 10px var(--elevation-shadow-dark), -4px -4px 10px var(--elevation-shadow-light);'
								: 'box-shadow:0 4px 14px var(--glow-accent);border:1px solid transparent;'}
							disabled={blocked}
							onclick={toggleFollow}
						>
							{#if followed}
								<UserCheck style="width:14px;height:14px;" strokeWidth={1.5} />
								已关注
							{:else}
								<UserPlus style="width:14px;height:14px;" strokeWidth={1.5} />
								关注
							{/if}
						</button>
					</div>
				</div>

				<h1 class="text-text-primary text-xl">{user.name}</h1>
				<p class="text-text-secondary text-[13px]" style="margin-top:1px;">{user.handle}{remark ? ` · ${remark}` : ''}</p>
				{#if blocked}
					<div class="mt-3 flex items-center gap-2 rounded-[12px] ring-1 ring-status-danger/30 bg-[color-mix(in_srgb,var(--status-danger)_8%,transparent)] px-3 py-2 text-[12px] text-status-danger">
						<UserX size={14} />
						你已屏蔽此用户；关注和私信已停用。
					</div>
				{/if}
				<p class="text-text-secondary text-[13px]" style="line-height:1.65;margin-top:8px;">{user.bio}</p>

				<div class="flex items-center gap-1.5 mt-2">
					<MapPin style="width:11px;height:11px;color:var(--accent-default);" strokeWidth={1.5} />
					<span class="text-text-secondary text-[11px]">{user.location}</span>
				</div>

				<!-- Tags -->
				<div class="flex gap-2 mt-3 flex-wrap">
					{#each user.tags as t (t)}
						<span class="px-2.5 py-0.5 rounded-full bg-accent-soft text-accent-default text-[11px]" style="border:1px solid var(--accent-border);">
							#{t}
						</span>
					{/each}
				</div>

				<!-- Stats -->
				<div class="flex gap-0 mt-5 overflow-hidden rounded-[20px] bg-surface-primary" style="box-shadow:0 0 0 1px var(--border-default), 6px 6px 14px var(--elevation-shadow-dark), -6px -6px 14px var(--elevation-shadow-light);">
					<div style="overflow:hidden;border-radius:20px;">
						{#each [{ label: '瞬间', v: fmtNum(user.moments) }, { label: '粉丝', v: fmtNum(user.followers) }, { label: '关注', v: fmtNum(user.following) }] as s, i}
							<div class="flex-1 flex flex-col items-center py-4 relative">
								{#if i > 0}
									<div class="absolute left-0 top-1/2 -translate-y-1/2 bg-border-default" style="width:1px;height:28px;"></div>
								{/if}
								<span class="text-text-primary text-[22px]">{s.v}</span>
								<span class="uppercase mt-1 text-text-secondary text-[9px]" style="letter-spacing:0.1em;">{s.label}</span>
								</div>
							{/each}
						</div>
					</div>

				<!-- Tabs -->
			<div class="flex" style="border-bottom:1px solid var(--border-default);padding-left:16px;padding-right:16px;flex-shrink:0;">
				{#each [{ id: 'moments', icon: Grid3X3, label: '瞬间' }, { id: 'liked', icon: Heart, label: '喜欢' }, { id: 'saved', icon: Bookmark, label: '收藏' }] as t}
					{@const active = activeTab === t.id}
					<button
						type="button"
						class="hover-btn relative flex items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none text-[13px] {active ? 'text-accent-default' : 'text-text-secondary'}"
						style="flex:1;font-weight:{active ? 500 : 400};"
						onclick={() => (activeTab = t.id as 'moments' | 'liked' | 'saved')}
					>
						<t.icon style="width:13px;height:13px;" strokeWidth={active ? 2 : 1.5} />
						{t.label}
						{#if active}
							<span class="bg-accent-default" style="position:absolute;bottom:-1px;left:18%;right:18%;height:2px;border-radius:1px;"></span>
						{/if}
					</button>
				{/each}
			</div>

			<!-- Gallery -->
			<div style="display:grid;grid-template-columns:repeat(3, 1fr);gap:3px;padding:3px;">
				{#each user.gallery as src, i}
					<div class="image-hover-frame tap-97 relative cursor-pointer" style="aspect-ratio:1;overflow:hidden;">
						<img src={src} alt={`photo ${i}`} class="image-hover-frame__image w-full h-full object-cover" />
						{#if activeTab === 'liked'}
							<div class="absolute inset-0 flex items-end justify-end p-1.5 bg-surface-overlay-soft">
								<div class="flex items-center justify-center rounded-full bg-accent-default" style="width:20px;height:20px;">
									<Heart style="width:10px;height:10px;color:var(--text-on-accent);" strokeWidth={2} fill="white" />
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
				<span class="px-2.5 py-0.5 rounded-full bg-accent-soft text-accent-default text-[11px]" style="border:1px solid var(--accent-border);">
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


	<RailSection title="你可能也关注" action="更多" onAction={() => void goto(ROUTES.nearbyPeople)}>
		<div class="flex flex-col gap-3">
			{#each similarUsers as u (u.id)}
				<RailUserRow avatar={u.avatar} name={u.name} sub={u.sub} onClick={() => void goto(ROUTES.user(u.id))} />
			{/each}
		</div>
	</RailSection>
	{/snippet}
</SplitPageLayout>
