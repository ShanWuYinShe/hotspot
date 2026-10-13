<script lang="ts">
	// 简介长度契约：上限 150，超过 85%（128）进入告警色
	const MAX_BIO = 150;
	const BIO_WARN = Math.floor(MAX_BIO * 0.85);
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';	import { Camera, User, FileText, Check, Trash2 } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import { logout } from '@/stores/auth.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Button from '@/components/ui/button.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import Input from '@/components/forms/Input.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import { getAccount, updateAccount, deleteAccount , netError } from '@/api';
	import type { Gender } from '@/api/types';
	import { uploadMediaFile } from '@/lib/upload';
	import { ROUTES } from '@/lib/routes';
	import { smartBack } from '@/lib/smart-back';

	// 编辑资料：昵称/简介/兴趣标签（5 个上限）+ 性别 + 注销按钮。

	const TAG_OPTIONS = ['摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术', '科技', '游戏', '户外', '时尚', '电影'];
	// 性别（spec: api_spec_user.md → Gender；'other' 覆盖非二元/不愿透露；null = 未设置）
	const GENDER_OPTIONS: { id: Gender; label: string }[] = [
		{ id: 'male', label: '男' },
		{ id: 'female', label: '女' },
		{ id: 'other', label: '其他' }
	];

	let nickname = $state('');
	let bio = $state('');
	let tags = $state<Set<string>>(new Set());
	// gender 为 null 时保存不携带该字段（updateAccount 为 patch 语义，undefined = 保持不变）
	let gender = $state<Gender | null>(null);
	let saved = $state(false);
	let savedTimer: number | undefined;
	$effect(() => () => window.clearTimeout(savedTimer));
	let saving = $state(false);
	let uploadingAvatar = $state(false);
	let deleteConfirmOpen = $state(false);
	let deleteConfirmText = $state('');
	let deleting = $state(false);
	let avatar = $state('');
	let userId = $state('');
	let avatarInput = $state<HTMLInputElement | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	// inline 错误（对齐 AuthForm：aria-invalid + 描述关联 + role="alert" 就近错误行）：
	// nicknameError 为字段级，formError 为表单级（服务端保存失败）
	let nicknameError = $state<string | null>(null);
	let formError = $state<string | null>(null);
	// 10100（未登录/会话失效）是正常业务态：给登录引导而非裸技术报错（对齐 notify/me）。
	let needsAuth = $state(false);
	let reloadKey = $state(0);

	const device = $derived(platform.platform);

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchAccount() {
			loading = true;
			loadError = null;
			needsAuth = false;
			try {
				const res = await getAccount().catch(() => netError(null));
				if (cancelled) return;
				if (res.code === 10100) {
					// 未登录：给登录引导而非裸技术报错
					needsAuth = true;
					return;
				}
				const data = res.code === 0 ? res.data : null;
				if (data) {
					// spec: AccountData.nickname, .bio, .tags, .avatar_url, .user_id, .gender?
					nickname = data.nickname;
					bio = data.bio ?? '';
					avatar = data.avatar_url ?? '';
					userId = data.user_id;
					if (data.tags) tags = new Set(data.tags);
					gender = data.gender ?? null;
				} else {
					// 加载失败必须显式暴露：否则空表单 + 可保存 = 用户一键清空自己的资料
					loadError = res.message ?? '加载资料失败';
				}
			} catch (err) {
				if (!cancelled) loadError = err instanceof Error ? err.message : '加载资料失败';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchAccount();
		return () => {
			cancelled = true;
		};
	});

	const toggleTag = (t: string) => {
		const n = new Set(tags);
		n.has(t) ? n.delete(t) : n.add(t);
		tags = n;
	};

	const handleSave = async () => {
		if (saving) return;
		const trimmedNickname = nickname.trim();
		if (!trimmedNickname) {
			// 字段级错误就近内联展示并聚焦，替代只靠 toast 的瞬时反馈
			nicknameError = '昵称不能为空';
			document.getElementById('edit-nickname')?.focus();
			return;
		}
		nicknameError = null;
		formError = null;
		saving = true;
		try {
			// spec: updateAccount { nickname?, avatar_file_id?, bio?, tags?, gender? }
			// Patch 清空语义（api_spec_user.md）：tags: [] = 清空标签、bio 传空串 = 清空简介；
			// 传 undefined 才表示"保持不变"——删光标签后必须能清空而不是保留旧值。
			const res = await updateAccount({
				nickname: trimmedNickname,
				bio: bio.trim(),
				tags: Array.from(tags),
				...(gender ? { gender } : {})
			});
			if (res.code !== 0) throw new Error(res.message || '保存失败');
			saved = true;
			showToast('已保存', 'success');
			savedTimer = window.setTimeout(() => {
				saved = false;
				// 保存后走路由去 /me 而非 history.back()：后者在整页直达编辑页时经
				// bfcache 恢复旧文档，个人页显示保存前的旧资料；路由导航必重挂重拉。
				void goto(ROUTES.me);
			}, 800);
		} catch (err) {
			// 服务端保存失败：表单级错误行内联展示（保留可重试的持久反馈）
			formError = err instanceof Error && err.message ? err.message : '保存失败，请重试';
		} finally {
			saving = false;
		}
	};

	const handleAvatarChange = async (e: Event) => {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file || uploadingAvatar) return;
		uploadingAvatar = true;
		try {
			const fileId = await uploadMediaFile(file, 'avatar');
			await updateAccount({ avatar_file_id: fileId });
			// 重新拉取账户，拿服务端最终 avatar_url
			const res = await getAccount().catch(() => null);
			if (res?.code === 0 && res.data?.avatar_url) avatar = res.data.avatar_url;
			showToast('头像已更新', 'success');
		} catch {
			showToast('头像上传失败，请重试', 'danger');
		} finally {
			uploadingAvatar = false;
		}
	};

	const confirmDelete = async () => {
		if (deleteConfirmText !== 'DELETE_MY_ACCOUNT' || deleting) return;
		deleting = true;
		try {
			// spec: users/delete_account 需精确输入 DELETE_MY_ACCOUNT，删除不可逆
			await deleteAccount({ confirmation: deleteConfirmText });
			await logout();
			void goto(ROUTES.home, { replaceState: true });
		} catch {
			deleting = false;
			showToast('注销失败，请重试', 'danger');
		}
	};
</script>
<svelte:head>
	<title>编辑资料 · HotSpot</title>
</svelte:head>


<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-10">
	{#snippet header()}
		<PageHeader title="编辑资料" onBack={() => smartBack()} titleSize={device === 'desktop' ? 'xl' : '2xl'}>
	{#snippet trailing()}
		<Button
			type="button"
			variant={saved ? 'secondary' : 'primary'}
			size="md"
			class="rounded-full px-4 py-2"
			loading={saving}
			disabled={saved || saving || !userId}
			onclick={() => void handleSave()}
		>
			{#if saved}
				<Check class="size-3.5 text-accent-default" strokeWidth={2.5} />
			{/if}
			{saved ? '已保存' : '保存'}
		</Button>
	{/snippet}
	</PageHeader>
{/snippet}

	{#if loading}
	<InlineLoading label="加载资料…" />
	{:else if needsAuth}
	<NeedsAuthNotice title="登录后查看" description="登录后即可编辑你的资料" />
	{:else if loadError}
	<LoadError message={loadError} onRetry={() => reloadKey++} />
	{:else}
	<div class="animate-fade-in mx-auto flex w-full max-w-3xl flex-col gap-5">
	<!-- Avatar section -->
	<div class="relative rounded-xl overflow-hidden h-cover-height-sm">
		<!-- 无封面数据时的占位底：主题 token 承载（vibrant 渐变 / forest 纸面纯色），页面只引用；
		     上传能力开放前「更换横幅」收敛为图标按钮（禁用态） -->
		<div class="absolute inset-0" style="background:var(--cst-surface-cover-placeholder);"></div>
		<button
			type="button"
			class="hover-btn absolute top-3 right-3 flex items-center justify-center rounded-full focus-ring bg-surface-overlay-strong text-content-inverse hover:opacity-hover size-8 disabled:opacity-muted"
			style="backdrop-filter:blur(var(--cst-blur-md));"
			disabled
			title="更换横幅（暂未开放）"
			aria-label="更换横幅（暂未开放）"
		>
			<Camera class="size-4"  strokeWidth={1.75} />
		</button>
	</div>

	<!-- Avatar edit -->
	<div class="flex items-center gap-4 -mt-8">
		<div class="relative shrink-0">
			<Avatar src={avatar} name={nickname} size="xl" shape="rounded" class="border-heavy border-surface-primary" style="box-shadow:var(--cst-elevation-profile-avatar);" />
			<button
				type="button"
				class="hover-btn absolute -bottom-1 -right-1 flex items-center justify-center rounded-full focus-ring bg-accent-default size-7 border-strong border-surface-primary shadow-raised-sm hover:shadow-raised-md"
				aria-label="更换头像"
				disabled={uploadingAvatar}
				onclick={() => avatarInput?.click()}
			>
				<Camera class="size-3 text-text-on-accent"   strokeWidth={2} />
			</button>
		</div>
		<input
			bind:this={avatarInput}
			type="file"
			accept="image/jpeg,image/png,image/webp"
			class="hidden"
			onchange={(e) => void handleAvatarChange(e)}
		/>
		<div>
			<p class="text-text-primary text-md">{nickname || '未设置昵称'}</p>
			{#if userId}
				<p class="text-text-secondary text-sm mt-0.5">@{userId}</p>
			{/if}
		</div>
	</div>

	<!-- Form fields — spec: nickname, bio, tags are the only user-editable profile fields. -->
	<!-- 昵称：迁移至 forms/Input 原语（ui-components.md backlog #2 首个消费方）。 -->
	<Input
		id="edit-nickname"
		value={nickname}
		onChange={(v) => {
			nickname = v.slice(0, 30);
			// 输入即清除 inline 错误（对齐 AuthForm clearFieldError）
			if (nicknameError) nicknameError = null;
			if (formError) formError = null;
		}}
		placeholder="你的名字"
		hasError={!!nicknameError}
		errorText={nicknameError ?? undefined}
		inputClass="w-full"
	>
		{#snippet labelSnippet()}
			<div class="flex items-center gap-1.5">
				<User class="size-3.5 text-accent-default" strokeWidth={1.75} />
				<label class="text-text-secondary text-xs uppercase tracking-field" for="edit-nickname">昵称</label>
			</div>
		{/snippet}
	</Input>

	<div class="flex flex-col gap-1.5">
		<div class="flex items-center gap-1.5">
			<FileText class="size-3.5 text-accent-default"   strokeWidth={1.75} />
			<label class="text-text-secondary text-xs uppercase tracking-field" for="edit-bio">个人简介</label>
		</div>
		<div class="px-4 py-3 rounded-md input-surface">
			<div>
				<textarea
					id="edit-bio"
					bind:value={bio}
					oninput={(e) => (bio = e.currentTarget.value.slice(0, MAX_BIO))}
					placeholder="介绍一下自己…"
					rows={3}
					class="w-full bg-transparent resize-none text-text-primary text-sm leading-copy"
				></textarea>
				<div class="flex justify-end mt-1">
					<span class="{bio.length > BIO_WARN ? 'text-status-warning' : 'text-text-secondary'} text-xs">
						{bio.length}/{MAX_BIO}
					</span>
				</div>
			</div>
		</div>
	</div>

	<!-- Gender（spec: updateAccount.gender，单选；不选 = 保持服务端当前值） -->
	<fieldset class="flex flex-col gap-2.5 border-0 p-0 m-0 min-w-0">
		<legend class="flex items-center gap-1.5 text-text-secondary text-xs uppercase tracking-field">
			性别
		</legend>
		<div class="flex flex-wrap gap-2">
			{#each GENDER_OPTIONS as g (g.id)}
				<TagPill active={gender === g.id} onClick={() => (gender = gender === g.id ? null : g.id)}>
					{g.label}
				</TagPill>
			{/each}
		</div>
	</fieldset>

	{#if formError}
		<p class="text-xs text-status-danger" role="alert">{formError}</p>
	{/if}

	<!-- Interest tags -->
	<fieldset class="flex flex-col gap-2.5 border-0 p-0 m-0 min-w-0">
		<legend class="flex items-center gap-1.5 text-text-secondary text-xs uppercase tracking-field">
			兴趣标签
			<span class="opacity-dim">({tags.size}/5)</span>
		</legend>
		<div class="flex flex-wrap gap-2">
			{#each TAG_OPTIONS as t (t)}
				{@const active = tags.has(t)}
				{@const disabled = !active && tags.size >= 5}
				<TagPill active={active} onClick={() => toggleTag(t)} disabled={disabled}>
					#{t}
				</TagPill>
			{/each}
		</div>
	</fieldset>

	<!-- Danger zone -->
	<div class="mt-3">
		<p class="eyebrow mb-3">账号</p>
		{#if deleteConfirmOpen}
			<div class="flex flex-col gap-2.5 rounded-md border border-border-default bg-surface-primary p-4">
				<p class="text-sm text-text-primary">删除不可恢复，请输入 <code class="font-mono text-xs text-status-danger">DELETE_MY_ACCOUNT</code> 确认：</p>
				<input
					type="text"
					bind:value={deleteConfirmText}
					placeholder="DELETE_MY_ACCOUNT"
					autocomplete="off"
					aria-label="注销确认码"
					class="w-full input-surface rounded-md px-3 py-2 text-sm text-text-primary"
				/>
				<div class="flex justify-end gap-2">
					<Button type="button" variant="secondary" size="sm" onclick={() => (deleteConfirmOpen = false)}>
						取消
					</Button>
					<Button
						type="button"
						variant="danger"
						size="sm"
						disabled={deleteConfirmText !== 'DELETE_MY_ACCOUNT' || deleting}
						loading={deleting}
						onclick={() => void confirmDelete()}
					>
						确认注销
					</Button>
				</div>
			</div>
		{:else}
			<Button
				type="button"
				variant="outline"
				size="md"
				class="w-full justify-start gap-2 rounded-md border-status-danger-border px-4 py-3.5 text-status-danger hover:bg-danger-bg"
				onclick={() => (deleteConfirmOpen = true)}
			>
				<Trash2 class="size-4" strokeWidth={1.75} />
				注销账号
			</Button>
		{/if}
	</div>
	</div>
	{/if}
</StandardPageLayout>
