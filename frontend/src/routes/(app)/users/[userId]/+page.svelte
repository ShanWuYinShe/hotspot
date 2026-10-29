<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		ArrowLeft,
		Check,
		ChevronRight,
		Flag,
		ImageIcon,
		MapPin,
		MessageCircle,
		MoreHorizontal,
		PencilLine,
		Share2,
		Tag,
		UserCheck,
		UserPlus,
		Users,
		UserX,
		X
	} from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import IdentityHero from '@/components/profile/IdentityHero.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import Button from '@/components/ui/button.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import ReportDialog from '@/components/overlays/ReportDialog.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { formatCardTime, formatCount, formatDistance } from '@/lib/utils';
	import { openTheDoor, follow, block, setRemark, netError } from '@/api';
	import type { UserPresenceStatus } from '@/api/types';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { ROUTES } from '@/lib/routes';
	import { smartBack } from '@/lib/smart-back';
	import { fadeImage, presetImageUrl, presetSrcSet } from '@/lib/images';
	import { copyShareLink } from '@/lib/share';

	// 用户详情：身份封面卡（IdentityHero other 模式）+ 瞬间网格 + 桌面 rail。
	// R13：头像/昵称/简介/位置/标签/指标/关系动作收敛进一张身份卡（原先散落在
	// 头像行、简介行、胶囊指标与右浮动作四段），封面承担页面视觉焦点。

	let { params }: { params: { userId: string } } = $props();

	interface UserData {
		name: string;
		bio: string;
		location: string;
		avatar: string;
		followers: number;
		following: number;
		moments: number;
		galleryMoments: Array<{ id: string; url: string; time: string }>;
		tags: string[];
	}

	let user = $state<UserData>({
		name: '',
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
	// 在线状态（spec: UserItem.presence_status，null = 对方隐藏了在线状态）：
	// 此前头像角标恒亮「在线」是假信息，现按接口真值渲染。
	let presence = $state<UserPresenceStatus | null>(null);
	let remark = $state('');
	let remarkDraft = $state('');
	let savingRemark = $state(false);
	let shareCopied = $state(false);

	const uid = $derived(params.userId ?? '');
	const device = $derived(platform.platform);
	const isMobile = $derived(device === 'mobile');

	// 自己看自己（mock 登录用户即 user_lumos）：此前仍显示「私信/关注/更多操作」，
	// 甚至能把「关注」点成功、把「备注」写到自己头上（全站复审 s2-19）。
	// 自己的资料页不发关系动作——改资料走 /me/edit（入口在个人中心）。
	const isSelf = $derived(!!uid && auth.user?.user_id === uid);

	// 身份卡副行：有备注时展示备注（只对自己可见）+ 账号标识
	const handleLine = $derived(remark ? `${remark} · @${uid}` : `@${uid}`);

	const metrics = $derived([
		{ label: '瞬间', value: formatCount(user.moments) },
		{ label: '粉丝', value: formatCount(user.followers) },
		{ label: '关注', value: formatCount(user.following) }
	]);

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchProfile() {
			try {
				loading = true;
				loadError = null;
				const res = await openTheDoor(uid, true).catch(() => netError(null));
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
						bio: u.bio ?? '',
						// 位置旁拼实时距离(后端 RelativeLocation.distance 已下发未映射;Gemini 审查 r20)
						location: [u.location?.location_name, formatDistance(u.location?.distance)].filter(Boolean).join(' · '),
						avatar: u.user.avatar_url ?? '',
						followers: s?.followers_count ?? 0,
						following: s?.following_count ?? 0,
						moments: s?.moments_count ?? 0,
						galleryMoments:
	// 只展示带图瞬间（无图动态渲染 <img src=""> 会破图）；保留 moment_id 供跳转详情
	data.recent_moments?.items
		?.filter((m) => m.images?.[0]?.url)
		.map((m) => ({
			id: m.moment_id,
			url: m.images![0]!.url,
			// 缩略图上的极简 meta：发布时间（避免纯图网格被读成装饰图）
			time: formatCardTime(m.created_at ?? '')
		})) ?? [],
						tags: u.tags ?? []
					};
					presence = u.presence_status ?? null;
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

	// in-flight 去重（防连点双翻转）；业务失败（code≠0）显式判 code 回滚
	let followPending = $state(false);
	let blockPending = false;

	const toggleFollow = async () => {
		if (blocked || followPending) return;
		if (!requireAuth({ action: '关注', targetId: uid, run: () => void toggleFollow() })) return;
		const next = !followed;
		const prevFollowed = followed;
		const prevFollowers = user.followers;
		followed = next;
		user.followers = Math.max(0, user.followers + (next ? 1 : -1));
		followPending = true;
		try {
			// spec: follow/unfollow uses user_id + active boolean.
			const res = await follow({ user_id: uid, active: next });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			followed = prevFollowed;
			user.followers = prevFollowers;
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			followPending = false;
		}
	};

	let confirmBlockOpen = $state(false);
	let confirmUnblockOpen = $state(false);

	const toggleBlock = async () => {
		if (blockPending) return;
		if (!requireAuth({ action: '屏蔽', targetId: uid, run: () => void toggleBlock() })) return;
		const next = !blocked;
		const prevBlocked = blocked;
		const prevFollowed = followed;
		const prevFollowers = user.followers;
		blocked = next;
		if (next && followed) {
			user.followers = Math.max(0, user.followers - 1);
		}
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
			user.followers = prevFollowers;
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			blockPending = false;
		}
	};

	// 举报弹层开合（理由选择/提交/提示均由 ReportDialog 内部承担，本页不自写提交逻辑）
	let reportOpen = $state(false);

	// 分享：share/create 生成分享码并复制 /share/[code] 落地链接；
	// 创建失败（匿名/网络异常）降级为复制页面原始链接（lib/share 统一降级链）。
	const copyProfileLink = async () => {
		const result = await copyShareLink({ type: 'user', id: uid });
		if (result === 'failed') {
			showToast('复制失败，请手动复制地址栏链接', 'danger');
			return;
		}
		shareCopied = true;
		showToast(result === 'shared' ? '分享链接已复制' : '链接已复制', 'success');
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
				class="mt-2 w-full input-surface rounded-md px-3.5 py-2.5 text-md text-text-primary"
			/>
		</label>
		<Button
			type="button"
			variant="secondary"
			disabled={!followed || savingRemark}
			class="mt-2 w-full"
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
		</Button>
		{#if !followed}
			<p class="mt-1 text-xs text-text-secondary">关注对方后可设备注</p>
		{/if}
		<Button
			type="button"
			variant="ghost"
			class="mt-2 w-full text-status-danger hover:text-text-status-danger-on-soft hover:bg-danger-bg"
			onclick={() => {
				moreOpen = false;
				if (blocked) confirmUnblockOpen = true;
				else confirmBlockOpen = true;
			}}
		>
			<UserX size={14} />
			{blocked ? '解除屏蔽' : '屏蔽用户'}
		</Button>
		<!-- 举报：走全站统一 ReportDialog（理由选择 + 提交 + 提示都在原语内），
		     此前本页手写了一套「举报理由手风琴 + 提交钮」嵌在更多操作弹层里，
		     与 ChatView/动态详情/群详情用的 ReportDialog 形态不一致（全站复审 s2-14）。 -->
		<Button
			type="button"
			variant="ghost"
			class="mt-1 w-full text-text-secondary"
			onclick={() => {
				moreOpen = false;
				reportOpen = true;
			}}
		>
			<Flag size={14} />
			举报用户
		</Button>
	</Dialog.Content>
</Dialog.Root>

<ReportDialog
	open={reportOpen}
	targetType="user"
	targetId={uid}
	targetLabel={user.name}
	onClose={() => (reportOpen = false)}
/>

<!-- 屏蔽/解除屏蔽二次确认（destructive 操作防误触），走全站唯一确认原语 ConfirmDialog -->
<ConfirmDialog
	open={confirmBlockOpen}
	tone="danger"
	icon={UserX}
	title="屏蔽该用户？"
	description="屏蔽后将无法收到对方消息与动态，且自动取消关注。"
	confirmLabel="屏蔽"
	layout={isMobile ? 'stacked' : 'row'}
	onCancel={() => (confirmBlockOpen = false)}
	onConfirm={() => {
		confirmBlockOpen = false;
		void toggleBlock();
	}}
/>
<ConfirmDialog
	open={confirmUnblockOpen}
	tone="neutral"
	icon={Check}
	title="解除屏蔽？"
	description="解除后可以重新关注对方并接收其动态。"
	confirmLabel="解除屏蔽"
	layout={isMobile ? 'stacked' : 'row'}
	onCancel={() => (confirmUnblockOpen = false)}
	onConfirm={() => {
		confirmUnblockOpen = false;
		void toggleBlock();
	}}
/>

<SplitPageLayout
	contentMaxWidth="58rem"
	contentClassName="animate-fade-in pt-4 pb-6"
>
	{#if loading}
		<!-- 单实体资料页：结构轻、通常 <1s，用 InlineLoading 而非骨架（避免骨架形状与最终封面差异造成二次位移） -->
		<div>
			<div class="mb-4 flex items-center">
				<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => smartBack()} size={42} />
			</div>
			<InlineLoading label="加载资料…" />
		</div>
	{:else if loadError}
		<div>
			<div class="mb-4 flex items-center">
				<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => smartBack()} size={42} />
			</div>
			<LoadError message={loadError} onRetry={() => reloadKey++} />
		</div>
	{:else}
		<!-- 身份封面卡（other 模式）：返回/分享/更多浮在封面上，关系动作收敛到卡片动作区 -->
		<IdentityHero
			mode="other"
			titleTag="h1"
			name={user.name}
			handle={handleLine}
			bio={user.bio || '这个人还没有填写简介。'}
			avatar={user.avatar}
			location={user.location || '未设置位置'}
			tags={user.tags}
			online={presence === null ? undefined : presence === 'online'}
			metrics={metrics}
		>
			{#snippet coverActions()}
				<ActionIconButton
					icon={ArrowLeft}
					label="返回"
					variant="overlay"
					tone="inverse"
					size={42}
					onClick={() => smartBack()}
				/>
				<div class="flex gap-2">
					<ActionIconButton
						icon={Share2}
						label={shareCopied ? '链接已复制' : '分享资料'}
						variant="overlay"
						tone="inverse"
						onClick={copyProfileLink}
					/>
					{#if !isSelf}
						<ActionIconButton
							icon={MoreHorizontal}
							label="更多操作"
							variant="overlay"
							tone="inverse"
							onClick={() => {
								remarkDraft = remark;
								moreOpen = true;
							}}
						/>
					{/if}
				</div>
			{/snippet}

			{#snippet actions()}
				<!-- R13 收尾：动作区同族同宽——未关注时 主=关注(primary)/次=私信(soft)；
				     已关注时 主=私信(primary)/次=已关注(soft)。不再出现第三种「accent 描边胶囊」。 -->
				{#if !isSelf}
					{#if followed}
						<Button
							type="button"
							variant="primary"
							size="md"
							class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
							disabled={blocked}
							onclick={() => !blocked && void goto(ROUTES.message(uid))}
						>
							<MessageCircle class="size-4" strokeWidth={1.75} />
							私信
						</Button>
						<Button
							type="button"
							variant="soft"
							size="md"
							class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
							aria-pressed={followed}
							disabled={followPending || blocked}
							onclick={() => void toggleFollow()}
						>
							<UserCheck class="size-4" strokeWidth={1.75} />
							{followed ? '已关注' : '关注'}
						</Button>
					{:else}
						<Button
							type="button"
							variant="primary"
							size="md"
							class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
							aria-pressed={followed}
							disabled={followPending || blocked}
							onclick={() => void toggleFollow()}
						>
							<UserPlus class="size-4" strokeWidth={1.75} />
							关注
						</Button>
						<Button
							type="button"
							variant="soft"
							size="md"
							class="hit-slop-44 flex-1 gap-1.5 sm:flex-none"
							disabled={blocked}
							onclick={() => !blocked && void goto(ROUTES.message(uid))}
						>
							<MessageCircle class="size-4" strokeWidth={1.75} />
							私信
						</Button>
					{/if}
				{/if}
			{/snippet}

			{#snippet children()}
				{#if blocked}
					<p
						class="flex items-center gap-2 rounded-md bg-danger-bg/60 px-3 py-2 text-sm text-text-status-danger-on-soft"
					>
						<UserX size={14} />
						你已屏蔽此用户；关注和私信已停用。
					</p>
				{/if}
			{/snippet}
		</IdentityHero>

		<!-- 瞬间网格：整格跳转动态详情（大图查看在详情页内）；网格对齐全站图片卡语言 -->
		<section class="mt-6" aria-label="最近瞬间">
			<SectionHead title="最近瞬间" count={user.galleryMoments.length > 0 ? user.galleryMoments.length : undefined} icon={ImageIcon} />
			{#if user.galleryMoments.length > 0}
				<div class="grid grid-cols-2 gap-2 sm:grid-cols-3 sm:gap-3">
					{#each user.galleryMoments as g, i (g.id)}
						<button
							type="button"
							class="media-placeholder image-hover-frame tap-97 relative aspect-square cursor-pointer overflow-hidden rounded-md focus-ring"
							aria-label={`查看 ${user.name} 的瞬间图片 ${i + 1}`}
							onclick={() => void goto(ROUTES.moment(g.id))}
						>
							<img
								src={presetImageUrl(g.url, 'card', 800)}
								srcset={presetSrcSet(g.url, 'card')}
								sizes="(min-width: 600px) 33vw, 50vw"
								alt={`${user.name} 的瞬间图片 ${i + 1}`}
								loading={i < 2 ? 'eager' : 'lazy'}
								fetchpriority={i < 2 ? 'high' : undefined}
								decoding="async"
								class="image-hover-frame__image h-full w-full object-cover"
								use:fadeImage
							/>
							{#if g.time}
								<span
									class="pointer-events-none media-badge absolute bottom-2 left-2 z-[2]"
								>
									{g.time}
								</span>
							{/if}
						</button>
					{/each}
				</div>
			{:else}
				<EmptyState
					iconTone="accent"
					class="py-12"
					title="还没有照片"
					description="该用户发布带图瞬间后会显示在这里。"
				>
					{#snippet icon()}
						<ImageIcon class="size-7" strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			{/if}
		</section>
	{/if}

	{#snippet rail()}
		<!-- 空数据不占卡：无位置/无标签/无共同好友时整卡隐藏（此前失败态与无标签用户都留空框） -->
		<div class="flex flex-col gap-4 py-2">
			{#if user.location}
				<section class="rounded-lg bg-surface-island p-4 shadow-xs" aria-label="所在位置">
					<SectionHead variant="rail" title="所在位置" icon={MapPin} />
					<button
						type="button"
						class="hover-bg -mx-2 flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left focus-ring"
						onclick={() => void goto(ROUTES.map)}
					>
						<span class="min-w-0 flex-1 truncate text-xs text-text-primary">{user.location}</span>
						<ChevronRight class="size-3.5 shrink-0 text-text-secondary" strokeWidth={1.5} />
					</button>
				</section>
			{/if}

			{#if user.tags.length > 0}
				<section class="rounded-lg bg-surface-island p-4 shadow-xs" aria-label="兴趣标签">
					<SectionHead variant="rail" title="兴趣标签" icon={Tag} />
					<div class="flex flex-wrap gap-2">
						{#each user.tags as t (t)}
							<span
								class="inline-flex h-6 items-center justify-center rounded-full bg-accent-soft px-2.5 text-xs font-medium leading-none text-text-accent-on-soft"
							>
								#{t}
							</span>
						{/each}
					</div>
				</section>
			{/if}

			{#if mutualFriends.length > 0}
				<section class="rounded-lg bg-surface-island p-4 shadow-xs" aria-label="共同好友">
					<SectionHead variant="rail" title="共同好友" icon={Users} />
					<div class="flex flex-col gap-2.5">
						{#each mutualFriends as f (f.id)}
							<RailUserRow avatar={f.avatar} name={f.name} sub={f.sub} onClick={() => void goto(ROUTES.user(f.id))} />
						{/each}
					</div>
				</section>
			{/if}
		</div>
	{/snippet}

</SplitPageLayout>