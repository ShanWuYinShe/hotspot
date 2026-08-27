<script lang="ts">
	import { Camera, User, FileText, Check } from '@lucide/svelte';
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
	import { getAccount, updateAccount, deleteAccount } from '@/api';
	import { uploadMediaFile } from '@/lib/upload';
	import { ROUTES } from '@/lib/routes';

	// 编辑资料：昵称/简介/兴趣标签（5 个上限）+ 注销按钮。

	const TAG_OPTIONS = ['摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术', '科技', '游戏', '户外', '时尚', '电影'];

	let nickname = $state('');
	let bio = $state('');
	let tags = $state<Set<string>>(new Set());
	let saved = $state(false);
	let saving = $state(false);
	let uploadingAvatar = $state(false);
	let deleteConfirmOpen = $state(false);
	let deleteConfirmText = $state('');
	let deleting = $state(false);
	let avatar = $state('');
	let userId = $state('');
	let avatarInput = $state<HTMLInputElement | null>(null);

	const device = $derived(platform.platform);

	$effect(() => {
		let cancelled = false;
		async function fetchAccount() {
			try {
				const res = await getAccount().catch(() => ({ code: -1, message: '网络异常', data: null }));
				if (cancelled) return;
				const data = res.code === 0 ? res.data : null;
				if (data) {
					// spec: AccountData.nickname, .bio, .tags, .avatar_url, .user_id
					nickname = data.nickname;
					bio = data.bio ?? '';
					avatar = data.avatar_url ?? '';
					userId = data.user_id;
					if (data.tags) tags = new Set(data.tags);
				}
			} finally {
				// ignore fetch error
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
		saving = true;
		try {
			// spec: updateAccount { nickname?, avatar_file_id?, bio?, tags?, gender? }
			const res = await updateAccount({
				nickname: nickname || undefined,
				bio: bio || undefined,
				tags: tags.size > 0 ? Array.from(tags) : undefined
			});
			if (res.code !== 0) throw new Error(res.message || '保存失败');
			saved = true;
			setTimeout(() => {
				saved = false;
				history.back();
			}, 800);
		} catch {
			showToast('保存失败，请重试', 'danger');
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
		<PageHeader title="编辑资料" onBack={() => history.back()} titleSize={device === 'desktop' ? 20 : device === 'tablet' ? 24 : 22}>
	{#snippet trailing()}
		<Button
			type="button"
			variant={saved ? 'secondary' : 'primary'}
			size="md"
			class="rounded-full px-4 py-2"
			loading={saving}
			disabled={saved || saving}
			onclick={() => void handleSave()}
		>
			{#if saved}
				<Check class="size-3.5 text-text-on-status" strokeWidth={2.5} />
			{/if}
			{saved ? '已保存' : '保存'}
		</Button>
	{/snippet}
	</PageHeader>
{/snippet}

	<div class="animate-fade-in flex flex-col gap-5">
	<!-- Avatar section -->
	<div class="relative rounded-xl overflow-hidden" style="height:var(--cst-cover-height-sm);">
		<!-- 无封面数据时的占位底：accent 淡底渐变，避免整块空白显得未完成 -->
		<div class="absolute inset-0" style="background:linear-gradient(135deg, var(--cst-accent-soft-bg), var(--cst-surface-primary) 70%);"></div>
		<button
			type="button"
			class="hover-btn absolute top-3 right-3 flex items-center gap-1.5 rounded-full focus-ring bg-surface-overlay-strong text-content-inverse text-sm px-3 py-1.5 disabled:opacity-muted"
			style="backdrop-filter:blur(var(--cst-blur-md));border:var(--cst-border-width-hairline) solid var(--cst-border-inverse);"
			disabled
			title="暂未开放"
		>
			<Camera class="size-3.5"  strokeWidth={1.75} />
			更换横幅
		</button>
	</div>

	<!-- Avatar edit -->
	<div class="flex items-center gap-4 -mt-8">
		<div class="relative shrink-0">
			<Avatar src={avatar} name={nickname} size={72} shape="rounded" class="border-3 border-surface-primary" style="box-shadow:var(--cst-elevation-profile-avatar);" />
			<button
				type="button"
				class="hover-btn hit-slop absolute -bottom-1 -right-1 flex items-center justify-center rounded-full focus-ring bg-accent-default size-6"
				style="border:var(--cst-border-width-strong) solid var(--cst-surface-primary);box-shadow:var(--shadow-glow-sm);"
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
	<div class="flex flex-col gap-1.5">
		<div class="flex items-center gap-1.5">
			<User class="size-3.5 text-accent-default"   strokeWidth={1.75} />
			<label class="text-text-secondary text-xs uppercase tracking-field" for="edit-nickname">昵称</label>
		</div>
		<div class="px-4 py-3 rounded-md input-surface">
			<input
				id="edit-nickname"
				bind:value={nickname}
				oninput={(e) => (nickname = e.currentTarget.value.slice(0, 30))}
				placeholder="你的名字"
				class="w-full bg-transparent text-text-primary text-sm"
			/>
		</div>
	</div>

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
					oninput={(e) => (bio = e.currentTarget.value.slice(0, 150))}
					placeholder="介绍一下自己…"
					rows={3}
					class="w-full bg-transparent resize-none text-text-primary text-sm leading-copy"
				></textarea>
				<div class="flex justify-end mt-1">
					<span class="{bio.length > 127 ? 'text-accent-default' : 'text-text-secondary'} text-xs">
						{bio.length}/150
					</span>
				</div>
			</div>
		</div>
	</div>

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
				class="w-full justify-start rounded-md px-4 py-3.5 text-status-danger"
				onclick={() => (deleteConfirmOpen = true)}
			>
				注销账号
			</Button>
		{/if}
	</div>
	</div>
</StandardPageLayout>
