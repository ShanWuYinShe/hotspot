<script lang="ts">
	// 群设置编辑（owner/admin）：复用 CreateGroupSheet 字段结构 + 封面上传。
	// 提交走 onSubmit 回调（页面注入 gd.updateGroupSettings），本组件保持纯 UI；
	// 封面在上传时先 uploadMediaFile('group_avatar') 换 file_id 再随 patch 提交。
	import { Camera, Check, Globe, Lock } from '@lucide/svelte';
	import { uploadMediaFile } from '@/lib/upload';
	import Toggle from '@/components/forms/Toggle.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import SheetShell from '@/components/overlays/SheetShell.svelte';
	import type { GroupItem, UpdateGroupRequest } from '@/api/types';

	let {
		open,
		initial,
		onClose,
		onSubmit
	}: {
		open: boolean;
		initial: GroupItem | null;
		onClose: () => void;
		/** 返回 true 表示提交成功（父组件负责 toast 与刷新）。 */
		onSubmit: (patch: Omit<UpdateGroupRequest, 'group_id'>) => Promise<boolean>;
	} = $props();

	const CATEGORIES = ['摄影', '音乐', '户外', '阅读', '美食', '健身', '游戏', '其他'];

	let name = $state('');
	let desc = $state('');
	let category = $state('摄影');
	let isPrivate = $state(false);
	let requiresApproval = $state(false);
	let avatar = $state('');
	let newAvatarFile = $state<File | null>(null);
	let saving = $state(false);
	let saveError = $state('');
	let coverInput = $state<HTMLInputElement | null>(null);
	// 待上传封面预览（objectURL；保存成功后由 loadDetail 拉取服务端最终 URL）
	let previewUrl = $state('');

	const canSave = $derived(name.trim().length >= 2 && !saving);

	// 打开时按 initial 回填（与 MomentEditDialog 的打开重置模式一致）
	$effect(() => {
		if (!open || !initial) return;
		name = initial.group.name;
		desc = initial.description ?? '';
		category = initial.tags?.[0] ?? '摄影';
		// 密码群暂不支持改门槛（与 CreateGroupSheet 一致只做 direct/invite_only 二元），
		// password 群编辑时保持公开形态，密码字段不动则不传。
		isPrivate = initial.join_method === 'invite_only';
		requiresApproval = initial.requires_approval;
		avatar = initial.group.avatar_url ?? '';
		newAvatarFile = null;
		previewUrl = '';
		saveError = '';
	});

	const handleCoverChange = (e: Event) => {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file || !file.type.startsWith('image/')) return;
		newAvatarFile = file;
		// 释放旧预览 URL，避免累积
		if (previewUrl) URL.revokeObjectURL(previewUrl);
		previewUrl = URL.createObjectURL(file);
	};

	const handleSave = async () => {
		if (!canSave) return;
		saving = true;
		saveError = '';
		try {
			const patch: Omit<UpdateGroupRequest, 'group_id'> = {
				name: name.trim(),
				description: desc.trim() || undefined,
				tags: [category],
				join_method: isPrivate ? 'invite_only' : 'direct',
				requires_approval: requiresApproval
			};
			if (newAvatarFile) {
				const fileId = await uploadMediaFile(newAvatarFile, 'group_avatar');
				patch.avatar_file_id = fileId;
			}
			const ok = await onSubmit(patch);
			if (!ok) return;
			onClose();
		} catch {
			saveError = '保存失败，请检查网络后重试';
		} finally {
			saving = false;
		}
	};
</script>

<SheetShell title="群设置" {onClose}>
	<div class="flex-1 overflow-y-auto no-scrollbar px-5 pt-4 pb-8 flex flex-col gap-5">
		<!-- Cover photo -->
		<div class="flex items-center gap-4">
			<div class="relative shrink-0">
				<img src={previewUrl || avatar} alt="群组封面" class="size-16 rounded-xl object-cover ring-1 ring-border-default bg-surface-primary" />
				<button
					type="button"
					class="hover-btn hit-slop absolute -bottom-1 -right-1 flex items-center justify-center rounded-full focus-ring bg-accent-default size-6.5"
					style="border:var(--cst-border-width-strong) solid var(--cst-surface-primary);box-shadow:var(--shadow-glow-sm);"
					onclick={() => coverInput?.click()}
					aria-label="更换封面"
				>
					<Camera class="size-3 text-text-on-accent" strokeWidth={2} />
				</button>
			</div>
			<div class="min-w-0">
				<p class="text-sm text-text-primary">群组封面</p>
				<p class="mt-0.5 text-xs text-text-secondary">支持 jpg/png，建议正方形图片</p>
			</div>
		</div>
		<input
			bind:this={coverInput}
			type="file"
			accept="image/*"
			class="hidden"
			onchange={(e) => handleCoverChange(e)}
		/>

		<!-- Group name -->
		<div>
			<label class="text-xs tracking-field uppercase text-text-secondary" for="group-settings-name">群组名称 *</label>
			<div class="flex items-center gap-2.5 mt-1.5 px-4 rounded-md input-surface h-11.5">
				<input
					id="group-settings-name"
					class="flex-1 bg-transparent text-text-primary text-sm outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
					placeholder="给你的群组起个名字…"
					bind:value={name}
					maxlength={30}
				/>
				<span class="text-xs shrink-0 text-text-secondary">{name.length}/30</span>
			</div>
		</div>

		<!-- Description -->
		<div>
			<label class="text-xs tracking-field uppercase text-text-secondary" for="group-settings-desc">简介</label>
			<div class="mt-1.5 px-4 py-3 rounded-md input-surface">
				<textarea
					id="group-settings-desc"
					class="w-full bg-transparent resize-none text-text-primary text-sm min-h-18 outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
					placeholder="描述一下这个群组的主题…"
					bind:value={desc}
					maxlength={150}
				></textarea>
				<div class="flex justify-end">
					<span class="text-xxs text-text-secondary">{desc.length}/150</span>
				</div>
			</div>
		</div>

		<!-- Category chips -->
		<fieldset class="flex flex-col border-0 p-0 m-0 min-w-0">
			<legend class="text-xs tracking-field uppercase text-text-secondary">分类</legend>
			<div class="flex flex-wrap gap-2 mt-2">
				{#each CATEGORIES as cat}
					<TagPill active={category === cat} onClick={() => (category = cat)}>
						{cat}
					</TagPill>
				{/each}
			</div>
		</fieldset>

		<!-- Privacy toggle -->
		<div class="flex items-center justify-between rounded-md border border-border-default bg-surface-primary px-4 py-3.5 shadow-raised-md">
			<div class="flex items-center gap-3">
				{#if isPrivate}
					<Lock class="size-4.5 text-accent-default" strokeWidth={1.5} />
				{:else}
					<Globe class="size-4.5 text-accent-default" strokeWidth={1.5} />
				{/if}
				<div>
					<p class="text-sm text-text-primary">{isPrivate ? '私密群组' : '公开群组'}</p>
					<p class="text-xs text-text-secondary">
						{isPrivate ? '仅受邀请的人可加入' : '所有人都可以找到并加入'}
					</p>
				</div>
			</div>
			<Toggle
				compact
				on={isPrivate}
				onToggle={() => (isPrivate = !isPrivate)}
				label={isPrivate ? '切换为公开' : '切换为私密'}
			/>
		</div>

		<!-- Approval toggle -->
		<div class="flex items-center justify-between rounded-md border border-border-default bg-surface-primary px-4 py-3.5 shadow-raised-md">
			<div class="flex items-center gap-3">
				<Check class="size-4.5" style="color:{requiresApproval ? 'var(--cst-accent-default)' : 'var(--cst-text-secondary)'};" strokeWidth={1.5} />
				<div>
					<p class="text-sm text-text-primary">新人需审核</p>
					<p class="text-xs text-text-secondary">
						{requiresApproval ? '申请需群主/管理员审核通过后加入' : '申请后直接加入'}
					</p>
				</div>
			</div>
			<Toggle
				compact
				on={requiresApproval}
				onToggle={() => (requiresApproval = !requiresApproval)}
				label={requiresApproval ? '关闭新人审核' : '开启新人审核'}
			/>
		</div>

		{#if saveError}
			<p class="text-sm text-status-warning">{saveError}</p>
		{/if}

		<!-- Save button -->
		<button
			type="button"
			onclick={() => void handleSave()}
			disabled={!canSave}
			class="hover-btn flex items-center justify-center gap-2 w-full py-4 rounded-lg focus-ring text-sm font-medium {canSave ? 'bg-accent-default text-text-on-accent' : 'bg-surface-primary text-text-secondary'}"
			style="box-shadow:{canSave ? 'var(--shadow-glow-lg)' : 'var(--shadow-inset-sm)'};border:{canSave ? 'none' : '1px solid var(--cst-border-default)'};transition:all var(--cst-motion-standard) var(--cst-ease-standard);cursor:{canSave ? 'pointer' : 'not-allowed'};"
		>
			{saving ? '保存中…' : '保存设置'}
		</button>
	</div>
</SheetShell>
