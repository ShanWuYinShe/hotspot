<script lang="ts">
	// 群设置编辑（owner/admin）：复用 CreateGroupSheet 字段结构 + 封面上传。
	// 提交走 onSubmit 回调（页面注入 gd.updateGroupSettings），本组件保持纯 UI；
	// 封面在上传时先 uploadMediaFile('group_avatar') 换 file_id 再随 patch 提交。
	import { Camera, Check, Globe, Lock, ShieldAlert, UserMinus, Users } from '@lucide/svelte';
	import { uploadMediaFile } from '@/lib/upload';
	import Toggle from '@/components/forms/Toggle.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import SheetShell from '@/components/overlays/SheetShell.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import type { GroupItem, UpdateGroupRequest } from '@/api/types';

	let {
		open,
		initial,
		onClose,
		onSubmit,
		isOwner = false,
		members = [],
		onTransferOwnership,
		onDismiss
	}: {
		open: boolean;
		initial: GroupItem | null;
		onClose: () => void;
		/** 返回 true 表示提交成功（父组件负责 toast 与刷新）。 */
		onSubmit: (patch: Omit<UpdateGroupRequest, 'group_id'>) => Promise<boolean>;
		/** 危险区（仅群主可见）：转让/解散 */
		isOwner?: boolean;
		/** 群成员（转让目标候选，页面注入 gd.members） */
		members?: { userId: string; name: string; role: string }[];
		onTransferOwnership?: (userId: string) => Promise<{ code: number; message?: string }>;
		onDismiss?: (reason?: string) => Promise<{ code: number; message?: string }>;
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

	// ── 危险区（owner）：转让群主 / 解散群组 ──
	let transferOpen = $state(false);
	let transferTarget = $state<{ userId: string; name: string } | null>(null);
	let transferring = $state(false);
	let dismissConfirmText = $state('');
	let dismissing = $state(false);
	let dangerError = $state('');
	// 可转让成员：非群主、非自己
	const transferableMembers = $derived(members.filter((m) => m.role !== 'owner' && m.userId));
	// 解散精确确认：输入群名（参照 me/edit 注销范式）
	const dismissReady = $derived(dismissConfirmText === initial?.group.name);

	// 打开时重置危险区状态
	$effect(() => {
		if (!open) return;
		transferOpen = false;
		transferTarget = null;
		dismissConfirmText = '';
		dangerError = '';
	});

	const handleConfirmTransfer = async () => {
		if (!transferTarget || !onTransferOwnership || transferring) return;
		transferring = true;
		dangerError = '';
		try {
			const res = await onTransferOwnership(transferTarget.userId);
			if (res.code === 0) {
				transferTarget = null;
				transferOpen = false;
				onClose();
			} else {
				dangerError = res.message ?? '转让失败';
			}
		} catch {
			dangerError = '转让失败，请重试';
		} finally {
			transferring = false;
		}
	};

	const handleDismiss = async () => {
		if (!dismissReady || !onDismiss || dismissing) return;
		dismissing = true;
		dangerError = '';
		try {
			const res = await onDismiss();
			if (res.code !== 0) dangerError = res.message ?? '解散失败';
		} catch {
			dangerError = '解散失败，请重试';
		} finally {
			dismissing = false;
		}
	};
</script>

<!-- Dialog 外壳：open 控制显隐（对齐 InviteMembersSheet 范式）。此前缺这层导致
	SheetShell 常驻文档流；bottom 档 sm 起居中限宽由 Dialog.Content 统一承担 -->
<Dialog.Root open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
	<Dialog.Content side="bottom" class="overflow-hidden max-h-[88vh]" aria-label="群设置">
		<SheetShell title="群设置" {onClose}>
	<div class="flex-1 overflow-y-auto no-scrollbar px-5 pt-4 pb-8 flex flex-col gap-5">
		<!-- Cover photo：空封面渲染占位图标（img src="" 会让浏览器把 alt 文字
			渲染进 64px 圆圈并溢出，与右侧文案重叠） -->
		<div class="flex items-center gap-4">
			<div class="relative shrink-0">
				{#if previewUrl || avatar}
					<img src={previewUrl || avatar} alt="群组封面" class="size-16 rounded-xl object-cover ring-1 ring-border-default" />
				{:else}
					<div class="flex items-center justify-center size-16 rounded-xl ring-1 ring-border-default bg-accent-soft" aria-hidden="true">
						<Camera class="size-5 text-text-secondary" strokeWidth={1.5} />
					</div>
				{/if}
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
					class="flex-1 bg-transparent text-text-primary text-sm"
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
					class="w-full bg-transparent resize-none text-text-primary text-sm min-h-18"
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
			style="box-shadow:{canSave ? 'var(--shadow-glow-lg)' : 'var(--shadow-inset-sm)'};border:{canSave ? 'none' : 'var(--cst-border-width-hairline) solid var(--cst-border-default)'};transition:all var(--cst-motion-standard) var(--cst-ease-standard);cursor:{canSave ? 'pointer' : 'not-allowed'};"
		>
			{saving ? '保存中…' : '保存设置'}
		</button>

		{#if isOwner}
			<!-- 危险区（仅群主；转让后原群主降级，解散不可逆） -->
			<section aria-labelledby="danger-zone-label" class="mt-2 rounded-md border border-status-danger-border bg-danger-bg p-4">
				<p id="danger-zone-label" class="flex items-center gap-1.5 text-sm font-medium text-status-danger">
					<ShieldAlert class="size-4" strokeWidth={1.75} />
					危险区
				</p>

				<!-- 转让群主 -->
				<div class="mt-3">
					<button
						type="button"
						onclick={() => (transferOpen = !transferOpen)}
						class="hover-btn flex w-full items-center justify-between rounded-md px-1 py-1 text-left outline-none focus-visible:ring-2 focus-visible:ring-accent-default"
					>
						<span class="flex items-center gap-2 text-sm text-text-primary">
							<Users class="size-4 text-text-secondary" strokeWidth={1.75} />
							转让群主
						</span>
						<span class="text-xs text-text-secondary">{transferOpen ? '收起' : '选择成员'}</span>
					</button>
					{#if transferOpen}
						<div class="mt-2 flex flex-col gap-1">
							{#each transferableMembers as m (m.userId)}
								<div class="flex items-center gap-2.5 rounded-md px-2 py-2">
									<span class="min-w-0 flex-1 truncate text-sm text-text-primary">{m.name}</span>
									<span class="text-xs text-text-secondary">{m.role === 'admin' ? '管理员' : '成员'}</span>
									<button
										type="button"
										onclick={() => (transferTarget = { userId: m.userId, name: m.name })}
										class="hover-btn rounded-full border border-border-default px-3 py-1.5 text-xs text-text-primary focus-ring"
									>转让</button>
								</div>
							{:else}
								<p class="px-2 py-1 text-xs text-text-secondary">群内暂无其他成员可转让</p>
							{/each}
						</div>
					{/if}
				</div>

				<!-- 解散群组（输入群名精确确认，参照 me/edit 注销范式） -->
				<div class="mt-3 border-t border-status-danger-border pt-3">
					<p class="flex items-center gap-2 text-sm text-text-primary">
						<UserMinus class="size-4 text-text-secondary" strokeWidth={1.75} />
						解散群组
					</p>
					<p class="mt-1 text-xs leading-5 text-text-secondary">解散后不可恢复，所有成员将被移出。输入群名 <code class="font-mono text-xs text-status-danger">{initial?.group.name}</code> 确认：</p>
					<input
						type="text"
						bind:value={dismissConfirmText}
						placeholder={initial?.group.name}
						autocomplete="off"
						aria-label="解散确认群名"
						class="mt-2 w-full input-surface rounded-md px-3 py-2 text-sm text-text-primary outline-none focus-visible:ring-2 focus-visible:ring-status-danger"
					/>
					<div class="mt-2 flex justify-end">
						<button
							type="button"
							onclick={() => void handleDismiss()}
							disabled={!dismissReady || dismissing}
							class="hover-btn rounded-full px-4 py-2 text-sm font-medium outline-none focus-visible:ring-2 focus-visible:ring-status-danger disabled:cursor-not-allowed disabled:opacity-muted enabled:bg-status-danger enabled:text-text-on-accent"
						>
							{dismissing ? '解散中…' : '解散群组'}
						</button>
					</div>
				</div>

				{#if dangerError}
					<p class="mt-2 text-sm text-status-warning">{dangerError}</p>
				{/if}
			</section>
		{/if}
	</div>
	</SheetShell>
	</Dialog.Content>
</Dialog.Root>

<!-- 转让确认（危险操作，参照删除公告的 Dialog 范式） -->
<Dialog.Root open={transferTarget !== null} onOpenChange={(next) => (next ? undefined : (transferTarget = null))}>
	<Dialog.Content side="center" class="max-w-sm p-5" aria-label="确认转让群主">
		<p class="text-md text-text-primary">确定将群主转让给 {transferTarget?.name}？</p>
		<p class="mt-1 text-sm text-text-secondary">转让后你将变为普通成员，无法再执行群管理操作。</p>
		<div class="mt-4 flex items-center justify-end gap-3">
			<button type="button" onclick={() => (transferTarget = null)} class="hover-btn rounded-full px-4 py-2 text-sm text-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-accent-default">取消</button>
			<button type="button" onclick={() => void handleConfirmTransfer()} disabled={transferring} class="hover-btn rounded-full px-5 py-2 text-sm font-medium text-text-on-accent outline-none focus-visible:ring-2 focus-visible:ring-accent-default disabled:cursor-not-allowed disabled:opacity-muted enabled:bg-accent-default">确认转让</button>
		</div>
	</Dialog.Content>
</Dialog.Root>
