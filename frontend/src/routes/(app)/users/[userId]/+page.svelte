<script lang="ts">
	import { goto } from '$app/navigation';
	import { MapPin, MessageCircle, UserPlus, UserCheck, Share2, MoreHorizontal, Flag, PencilLine, UserX, X, Check, ArrowLeft } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import SplitPageLayout from '@/components/layout/page/SplitPageLayout.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import RailSection from '@/components/rail/RailSection.svelte';
	import RailUserRow from '@/components/rail/RailUserRow.svelte';
	import RailLocationMini from '@/components/rail/RailLocationMini.svelte';
	import MetricRow from '@/components/data-display/MetricRow.svelte';
	import { openTheDoor, follow, block, createReport, setRemark , netError } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { requireAuth } from '@/stores/auth.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { smartBack } from '@/lib/smart-back';

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

	let confirmBlockOpen = $state(false);
	let confirmUnblockOpen = $state(false);

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

	// 举报理由（api types ReportReason 枚举取常用四档；harassment 骚扰 / impersonation 冒充他人）
	const REPORT_REASONS = [
		{ id: 'spam', label: '垃圾信息' },
		{ id: 'harassment', label: '骚扰攻击' },
		{ id: 'impersonation', label: '冒充他人' },
		{ id: 'other', label: '其他' }
	] as const;
	let reportReason = $state<(typeof REPORT_REASONS)[number]['id']>('spam');
	let reportPending = $state(false);
	let reportOpen = $state(false);

	const submitReport = async () => {
		if (reportPending) return;
		if (!requireAuth({ action: '举报', targetId: uid, run: () => void submitReport() })) return;
		reportPending = true;
		try {
			const res = await createReport({ target_type: 'user', target_id: uid, reason: reportReason });
			if (res.code === 0) {
				reportSent = true;
				showToast('举报已记录，感谢你的反馈', 'success');
				moreOpen = false;
			} else {
				showToast(res.message || '举报失败', 'danger');
			}
		} catch {
			showToast('举报失败，请重试', 'danger');
		} finally {
			reportPending = false;
		}
	};

	const copyProfileLink = async () => {
		try {
			await navigator.clipboard?.writeText(`${window.location.origin}${ROUTES.user(uid)}`);
			shareCopied = true;
			showToast('链接已复制', 'success');
			window.setTimeout(() => (shareCopied = false), 1600);
		} catch {
			showToast('复制失败，请手动复制地址栏链接', 'danger');
		}
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
				class="hover-btn mt-2 flex w-full items-center justify-center gap-2 rounded-sm border border-border-default py-2.5 text-sm text-text-primary hover:bg-accent-soft focus-ring disabled:opacity-muted"
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
					class="hover-btn mt-2 flex w-full items-center justify-center gap-2 rounded-sm py-2.5 text-sm text-status-danger outline-none hover:bg-danger-bg focus-visible:ring-2 focus-visible:ring-status-danger"
					onclick={() => (blocked ? (confirmUnblockOpen = true) : (confirmBlockOpen = true))}
				>
					<UserX size={14} />
					{blocked ? '解除屏蔽' : '屏蔽用户'}
				</button>
				<button
					type="button"
					class="hover-btn mt-1 flex w-full items-center justify-center gap-2 rounded-sm py-2.5 text-sm text-text-secondary focus-ring"
					onclick={() => (reportOpen = !reportOpen)}
					aria-expanded={reportOpen}
				>
					{#if reportSent}
						<Check size={14} />
						举报已记录
					{:else}
						<Flag size={14} />
						{reportOpen ? '收起理由' : '举报用户'}
					{/if}
				</button>
				{#if reportOpen && !reportSent}
					<div class="mt-2 flex flex-col gap-1.5" aria-label="举报理由">
						{#each REPORT_REASONS as r (r.id)}
							{@const selected = reportReason === r.id}
							<!-- 选中/未选中 hover 差异化（四态契约）：未选中 hover 淡底预览、选中 hover 加深 -->
							<button
								type="button"
								aria-pressed={selected}
								onclick={() => (reportReason = r.id)}
								class="hover-btn flex items-center justify-between rounded-sm border px-3 py-2 text-left text-sm transition-colors duration-rapid ease-standard focus-ring
									{selected
									? 'border-accent-border bg-accent-soft text-accent-default hover:bg-interactive-active'
									: 'border-border-default text-text-secondary hover:bg-interactive-hover'}"
							>
								{r.label}
								{#if selected}<Check class="size-3.5" strokeWidth={2.5} />{/if}
							</button>
						{/each}
						<button
							type="button"
							disabled={reportPending}
							onclick={() => void submitReport()}
							class="hover-btn mt-1 flex w-full items-center justify-center gap-2 rounded-full border border-border-default py-2.5 text-sm font-medium text-status-danger focus-ring hover:bg-danger-bg disabled:pointer-events-none disabled:opacity-faint"
						>
							{reportPending ? '提交中…' : '提交举报'}
						</button>
					</div>
				{/if}
		</Dialog.Content>
	</Dialog.Root>

	<!-- 屏蔽/解除屏蔽二次确认（destructive 操作防误触） -->
	<Dialog.Root bind:open={confirmBlockOpen}>
		<Dialog.Content class="max-w-90 p-0">
			<div class="px-5 pt-6 pb-4">
				<div class="mb-3.5 flex size-10 items-center justify-center rounded-full bg-danger-bg text-status-danger ring-1 ring-status-danger-border">
					<UserX class="size-5" />
				</div>
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">屏蔽该用户？</Dialog.Title>
				<Dialog.Description class="mt-1 text-sm text-text-secondary leading-relaxed">屏蔽后将无法收到对方消息与动态，且自动取消关注。</Dialog.Description>
			</div>
			<div class="flex flex-col gap-2 px-5 pb-5">
				<Button type="button" variant="danger" size="lg" class="w-full" onclick={() => { confirmBlockOpen = false; void toggleBlock(); }}>屏蔽</Button>
				<Button type="button" variant="secondary" size="lg" class="w-full" onclick={() => (confirmBlockOpen = false)}>取消</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>
	<Dialog.Root bind:open={confirmUnblockOpen}>
		<Dialog.Content class="max-w-90 p-0">
			<div class="px-5 pt-6 pb-4">
				<div class="mb-3.5 flex size-10 items-center justify-center rounded-full bg-accent-soft text-accent-default ring-1 ring-accent-border">
					<Check class="size-5" />
				</div>
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">解除屏蔽？</Dialog.Title>
				<Dialog.Description class="mt-1 text-sm text-text-secondary leading-relaxed">解除后可以重新关注对方并接收其动态。</Dialog.Description>
			</div>
			<div class="flex flex-col gap-2 px-5 pb-5">
				<Button type="button" variant="primary" size="lg" class="w-full" onclick={() => { confirmUnblockOpen = false; void toggleBlock(); }}>解除屏蔽</Button>
				<Button type="button" variant="secondary" size="lg" class="w-full" onclick={() => (confirmUnblockOpen = false)}>取消</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>

<SplitPageLayout contentClassName="animate-fade-in pt-4 {platform.platform === 'mobile' ? 'pb-28' : 'pb-6'}">
	<!-- 错误态收敛进 layout children（与正文同 gutter，不再偏移） -->
	{#if loadError}
		<LoadError message={loadError} onRetry={() => reloadKey++} />
	{:else}
	<!-- 资料头（无封面数据：紧凑头部，返回/分享/更多 浮在首行，对齐动态详情作者行层级） -->
	<div class="flex items-center justify-between">
		<ActionIconButton icon={ArrowLeft} label="返回" onClick={() => smartBack()} />
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
							<Avatar src={user.avatar} name={user.name} size="xl" class="border-2 border-surface-primary shadow-raised-lg" />
							<!-- 角标统一 OnlineDot（审查 M3：手写 14px 异档收编为 lg 档 + 内边距覆写） -->
							<OnlineDot size="lg" class="bottom-1 right-1 border-2" />
						</div>
						<div class="min-w-0 pt-1">
							<h1 class="truncate text-2xl font-medium tracking-title text-text-primary">{user.name}</h1>
							<p class="mt-1 truncate text-text-secondary">{user.handle}{remark ? ` · ${remark}` : ''}</p>
						</div>
					</div>

					<!-- Action buttons（桌面/平板在头像旁；移动端下沉为底部操作条，见 footer snippet） -->
					{#if platform.platform !== 'mobile'}
					<div class="flex shrink-0 gap-2 pb-1">
						<button
							type="button"
							class="hover-btn tap-92 hit-slop flex items-center gap-1.5 rounded-full border border-border-default bg-surface-primary px-3.5 py-2 text-xs text-text-secondary shadow-raised-sm hover:bg-accent-soft focus-ring disabled:cursor-not-allowed disabled:opacity-muted"
							disabled={blocked}
							onclick={() => !blocked && void goto(ROUTES.message(uid))}
						>
							<MessageCircle class="size-3.5"  strokeWidth={1.5} />
							私信
						</button>
						<!-- 关注按钮：边框/投影走 token 类（描边 1.5px→hairline 档收敛，border 基类保尺寸稳定） -->
						<button
							type="button"
							class="hover-btn tap-92 hit-slop flex items-center gap-1.5 px-4 py-2 rounded-full border focus-ring transition-all disabled:cursor-not-allowed disabled:opacity-muted text-xs {followed ? 'border-accent-default bg-surface-primary text-accent-default shadow-raised-sm hover:bg-accent-soft' : 'border-transparent bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md'}"
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
					{/if}
				</div>

			{#if blocked}
				<div class="mt-5 flex items-center gap-2 rounded-sm ring-1 ring-status-danger-border bg-danger-bg px-3 py-2 text-sm text-status-danger">
					<UserX size={14} />
					你已屏蔽此用户；关注和私信已停用。
				</div>
			{/if}
			<p class="mt-5 max-w-115 text-sm leading-6 text-text-secondary">{user.bio || '这个人还没有填写简介。'}</p>

			<!-- 位置 + 标签 -->
			<div class="mt-5 flex flex-wrap items-center gap-x-4 gap-y-2">
				{#if user.location}
					<span class="flex items-center gap-1.5 text-text-secondary text-sm">
						<MapPin class="size-3.5 text-accent-default"   strokeWidth={1.75} />
						{user.location}
					</span>
				{/if}
				{#each user.tags as t (t)}
					<span class="inline-flex h-6 items-center justify-center rounded-full border border-accent-border bg-accent-soft px-2.5 text-xs font-medium leading-none text-accent-default">
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
	{#if user.galleryMoments.length > 0}
		<div class="mt-5 grid grid-cols-3 gap-1.5">
			{#each user.galleryMoments as g, i (g.id)}
				<button
					type="button"
					class="image-hover-frame tap-97 relative cursor-pointer rounded-sm overflow-hidden focus-ring"
					style="aspect-ratio:1;"
					aria-label={`查看 ${user.name} 的瞬间图片 ${i + 1}`}
					onclick={() => void goto(ROUTES.moment(g.id))}
				>
					<img src={g.url} alt={`${user.name} 的瞬间图片 ${i + 1}`} loading="lazy" class="image-hover-frame__image w-full h-full object-cover" />
				</button>
			{/each}
		</div>
	{:else}
		<EmptyState bare title="还没有照片" description="该用户发布带图瞬间后会显示在这里。" />
	{/if}
</div>
	{/if}

	{#snippet rail()}
	<RailSection title="所在位置">
		<RailLocationMini name={user.location} />
	</RailSection>

	<RailSection title="兴趣标签">
		<div class="flex gap-2 flex-wrap">
			{#each user.tags as t (t)}
				<span class="inline-flex h-6 items-center justify-center rounded-full border border-accent-border bg-accent-soft px-2.5 text-xs font-medium leading-none text-accent-default">
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

	<!-- 移动端底部操作条：主操作（私信/关注）下沉拇指热区，桌面/平板使用头部按钮 -->
	{#snippet footer()}
		{#if platform.platform === 'mobile' && !blocked}
			<div
				class="flex gap-3 border-t border-border-default px-4 pt-2.5"
				style="background:var(--cst-surface-nav-glass);backdrop-filter:blur(var(--cst-blur-md));-webkit-backdrop-filter:blur(var(--cst-blur-md));padding-bottom:max(10px, env(safe-area-inset-bottom, 10px));"
			>
				<button
					type="button"
					class="hover-btn tap-92 flex h-11 flex-1 items-center justify-center gap-1.5 rounded-xl border border-border-default bg-surface-primary text-sm text-text-secondary shadow-raised-sm hover:bg-accent-soft focus-ring disabled:opacity-muted"
					disabled={blocked}
					onclick={() => !blocked && void goto(ROUTES.message(uid))}
				>
					<MessageCircle class="size-4"  strokeWidth={1.75} />
					私信
				</button>
				<button
					type="button"
					class="hover-btn tap-92 flex h-11 flex-1 items-center justify-center gap-1.5 rounded-xl border focus-ring transition-all disabled:opacity-muted text-sm {followed ? 'border-accent-default bg-surface-primary text-accent-default shadow-raised-sm hover:bg-accent-soft' : 'border-transparent bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md'}"
					disabled={blocked}
					onclick={toggleFollow}
				>
					{#if followed}
						<UserCheck class="size-4"  strokeWidth={1.75} />
						已关注
					{:else}
						<UserPlus class="size-4"  strokeWidth={1.75} />
						关注
					{/if}
				</button>
			</div>
		{/if}
	{/snippet}
</SplitPageLayout>

