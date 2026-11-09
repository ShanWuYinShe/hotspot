<script lang="ts">
	import { Camera, Check, Trash2, UserRound } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Button from '@/components/ui/button.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import Input from '@/components/forms/Input.svelte';
	import Textarea from '@/components/forms/Textarea.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import { smartBack } from '@/lib/smart-back';

	import {
		useEditProfileData,
		MAX_BIO,
		BIO_WARN,
		TAG_OPTIONS,
		GENDER_OPTIONS
	} from './useEditProfileData.svelte';

	let avatarInput = $state<HTMLInputElement | null>(null);
	const page = useEditProfileData();
	const device = $derived(platform.platform);
</script>

<svelte:head>
	<title>编辑资料 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentMaxWidth="var(--cst-content-form-narrow)" contentClassName="pt-3 lg:pt-2.5 pb-10">
	{#snippet header()}
		<PageHeader
			variant="integrated"
			title="编辑资料"
			islandMeta={page.islandMeta}
			maxWidth="var(--cst-content-form-narrow)"
			onBack={() => smartBack()}
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
		>
			{#snippet islandIcon()}
				<UserRound class="size-3.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet trailing()}
				{#if !page.needsAuth && device !== 'mobile'}
					<Button
						type="button"
						variant={page.saved ? 'secondary' : 'primary'}
						size="md"
						class="hit-slop-44 px-4 py-2"
						loading={page.saving}
						disabled={page.saved || page.saving || !page.userId}
						onclick={() => void page.handleSave()}
					>
						{#if page.saved}
							<Check class="size-3.5 text-accent-default" strokeWidth={2.5} />
						{/if}
						{page.saved ? '已保存' : '保存'}
					</Button>
				{/if}
			{/snippet}
			{#snippet islandTrailing()}
				{#if !page.needsAuth}
					<Button
						type="button"
						variant={page.saved ? 'secondary' : 'primary'}
						size="sm"
						shape="pill"
						class="hit-slop gap-1 px-2.5"
						loading={page.saving}
						disabled={page.saved || page.saving || !page.userId}
						aria-label={page.saving ? '保存中' : page.saved ? '已保存' : '保存资料'}
						onclick={() => void page.handleSave()}
					>
						{#if page.saved}
							<Check class="size-3.5 text-accent-default" strokeWidth={2.5} />
						{/if}
						{page.saved ? '已保存' : '保存'}
					</Button>
				{/if}
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if page.loading}
		<InlineLoading label="加载资料…" />
	{:else if page.needsAuth}
		<NeedsAuthNotice
			spotlight
			title="登录后编辑"
			description="登录后即可编辑头像、昵称与简介，让附近的人更容易记住你。"
		/>
	{:else if page.loadError}
		<LoadError message={page.loadError} onRetry={() => page.refresh()} />
	{:else}
		<div class="flex w-full flex-col gap-4">
			<!-- 头像与账号 -->
			<section class="flex items-center gap-4 rounded-lg bg-surface-island p-4 shadow-xs" aria-label="头像与账号">
				<div class="relative shrink-0">
					<Avatar
						src={page.avatar}
						name={page.nickname}
						size="xl"
						shape="rounded"
						class="border-heavy border-surface-primary"
						style="box-shadow:var(--cst-elevation-profile-avatar);"
					/>
					<button
						type="button"
						class="hover-btn hit-slop-44 absolute -bottom-1 -right-1 flex items-center justify-center rounded-full focus-ring bg-accent-default size-7 border-strong border-surface-primary shadow-raised-sm hover:shadow-raised-md"
						aria-label="更换头像"
						disabled={page.uploadingAvatar}
						onclick={() => avatarInput?.click()}
					>
						<Camera class="size-3 text-text-on-accent" strokeWidth={2} />
					</button>
				</div>
				<input
					bind:this={avatarInput}
					type="file"
					accept="image/jpeg,image/png,image/webp"
					class="hidden"
					onchange={(e) => {
						const input = e.currentTarget as HTMLInputElement;
						const file = input.files?.[0];
						input.value = '';
						if (file) void page.handleAvatarUpload(file);
					}}
				/>
				<div class="min-w-0 flex-1">
					<p class="truncate text-md font-medium text-text-primary">{page.nickname || '未设置昵称'}</p>
					{#if page.userId}
						<p class="mt-0.5 truncate text-xs text-text-secondary">@{page.userId}</p>
					{/if}
					<p class="mt-1 text-xs text-text-secondary">
						{page.uploadingAvatar ? '头像上传中…' : '点头像可更换照片'}
					</p>
				</div>
			</section>

			<!-- 基本资料 -->
			<section aria-label="基本资料">
				<SectionHead title="基本资料" subtitle="公开显示在你的主页上，附近的人会看到这些信息。" />
				<div class="flex flex-col gap-4 rounded-lg bg-surface-island p-4 shadow-xs">
					<Input
						id="edit-nickname"
						value={page.nickname}
						onChange={(v) => {
							page.nickname = v.slice(0, 30);
							if (page.nicknameError) page.nicknameError = null;
							if (page.formError) page.formError = null;
						}}
						placeholder="你的名字"
						hasError={!!page.nicknameError}
						errorText={page.nicknameError ?? undefined}
						inputClass="w-full"
					>
						{#snippet labelSnippet()}
							<label class="text-text-secondary text-xs uppercase tracking-field" for="edit-nickname">昵称</label>
						{/snippet}
					</Input>

					<Textarea
						id="edit-bio"
						label="个人简介"
						value={page.bio}
						onChange={(v) => (page.bio = v.slice(0, MAX_BIO))}
						placeholder="介绍一下自己…"
						rows={3}
						maxlength={MAX_BIO}
					>
						{#snippet footer()}
							<span class={page.bio.length > BIO_WARN ? 'text-status-warning' : 'text-text-secondary'}>{page.bio.length}/{MAX_BIO}</span>
						{/snippet}
					</Textarea>
				</div>
			</section>

			<!-- 更多信息：性别与兴趣标签 -->
			<section aria-label="更多信息">
				<SectionHead title="更多信息" subtitle="帮助附近的人更快了解你，随时可以修改或清空。" />
				<div class="flex flex-col gap-4 rounded-lg bg-surface-island p-4 shadow-xs">
					<fieldset class="flex flex-col gap-2.5 border-0 p-0 m-0 min-w-0">
						<legend class="flex items-center gap-1.5 text-text-secondary text-xs uppercase tracking-field">
							性别
						</legend>
						<div class="flex flex-wrap gap-2">
							{#each GENDER_OPTIONS as g (g.id)}
								<TagPill active={page.gender === g.id} onClick={() => (page.gender = page.gender === g.id ? null : g.id)}>
									{g.label}
								</TagPill>
							{/each}
						</div>
					</fieldset>

					<fieldset class="flex flex-col gap-2.5 border-0 p-0 m-0 min-w-0">
						<legend class="flex items-center gap-1.5 text-text-secondary text-xs uppercase tracking-field">
							兴趣标签
							<span class="opacity-dim">({page.tags.size}/5)</span>
						</legend>
						<div class="flex flex-wrap gap-2">
							{#each TAG_OPTIONS as t (t)}
								{@const active = page.tags.has(t)}
								{@const disabled = !active && page.tags.size >= 5}
								<TagPill active={active} onClick={() => page.toggleTag(t)} {disabled}>
									#{t}
								</TagPill>
							{/each}
							{#each page.extraTags as t (t)}
								<TagPill active onClick={() => page.toggleTag(t)}>
									#{t}
								</TagPill>
							{/each}
						</div>
					</fieldset>
				</div>
			</section>

			{#if page.formError}
				<p class="text-xs text-status-danger" role="alert">{page.formError}</p>
			{/if}

			<!-- 危险区域：注销账号 -->
			<div>
				<p class="eyebrow mb-3">账号</p>
				{#if page.deleteConfirmOpen}
					<div class="flex flex-col gap-2.5 rounded-lg bg-danger-bg/40 p-4">
						<p class="text-sm text-text-primary">删除不可恢复，请输入 <code class="font-mono text-xs text-status-danger">DELETE_MY_ACCOUNT</code> 确认：</p>
						<Input
							compact
							type="text"
							value={page.deleteConfirmText}
							onChange={(v) => (page.deleteConfirmText = v)}
							placeholder="DELETE_MY_ACCOUNT"
							autocomplete="off"
							ariaLabel="注销确认码"
							density="compact"
							inputClass="w-full"
						/>
						<div class="flex justify-end gap-2">
							<Button type="button" variant="secondary" size="sm" onclick={() => (page.deleteConfirmOpen = false)}>
								取消
							</Button>
							<Button
								type="button"
								variant="danger"
								size="sm"
								disabled={page.deleteConfirmText !== 'DELETE_MY_ACCOUNT' || page.deleting}
								loading={page.deleting}
								onclick={() => void page.confirmDelete()}
							>
								确认注销
							</Button>
						</div>
					</div>
				{:else}
					<Button
						type="button"
						variant="danger-soft"
						size="md"
						class="hit-slop-44 w-full justify-start gap-2"
						onclick={() => (page.deleteConfirmOpen = true)}
					>
						<Trash2 class="size-4" strokeWidth={1.75} />
						注销账号
					</Button>
				{/if}
			</div>
		</div>

		<!-- 移动端底栏保存按钮 -->
		{#if device === 'mobile' && !page.needsAuth}
			<div
				class="nav-glass nav-glass--bottom sticky bottom-0 z-10 -mx-[var(--cst-page-gutter)] mt-1 px-4 pt-3"
				style="padding-bottom:max(10px, env(safe-area-inset-bottom, 10px));"
			>
				<Button
					type="button"
					variant={page.saved ? 'secondary' : 'primary'}
					size="lg"
					class="w-full gap-2"
					loading={page.saving}
					disabled={page.saved || page.saving || !page.userId}
					onclick={() => void page.handleSave()}
				>
					{#if page.saved}
						<Check class="size-4 text-accent-default" strokeWidth={2.5} />
					{/if}
					{page.saved ? '已保存' : page.saving ? '保存中…' : '保存资料'}
				</Button>
			</div>
		{/if}
	{/if}
</StandardPageLayout>
