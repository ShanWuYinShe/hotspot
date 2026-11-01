<script lang="ts">
	import { ArrowLeft, Camera, Globe, Lock, Check, MapPin } from '@lucide/svelte';
	import { onDestroy } from 'svelte';
	import { createGroup } from '@/api';
	import { uploadMediaFile } from '@/lib/upload';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { loadDefaultLocation } from '@/lib/default-location';
	import Toggle from '@/components/forms/Toggle.svelte';
	import Input from '@/components/forms/Input.svelte';
	import Textarea from '@/components/forms/Textarea.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import SheetShell from '@/components/overlays/SheetShell.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Button from '@/components/ui/button.svelte';

	// CreateGroupSheet — 创建群组的底部 Sheet / 模态。
	// 使用处：NearbyGroups(FAB)、Profile > 我的群组（页头按钮）、
	//        以及 NewConversationSheet 的「创建群聊」入口。
	// 创建走 groups/create（返回 GroupItem，group_id 经 onCreated 回调交给调用方）。

	let {
		onClose,
		onBack,
		onCreated
	}: { onClose: () => void; /** 从发布器切来时的返回入口（CreateSheet surface 切换） */ onBack?: () => void; onCreated?: (groupId: string) => void } = $props();

	const CATEGORIES = ['摄影', '音乐', '户外', '阅读', '美食', '健身', '游戏', '其他'];

	let name = $state('');
	let desc = $state('');
	let category = $state('摄影');
	let isPrivate = $state(false);
	let requiresApproval = $state(false);
	let step = $state<'form' | 'done'>('form');
	let creating = $state(false);
	let createError = $state('');
	let locationName = $state('');
	let coverInput = $state<HTMLInputElement | null>(null);
	let newAvatarFile = $state<File | null>(null);
	let previewUrl = $state('');
	let uploadingCover = $state(false);

	const canCreate = $derived(name.trim().length >= 2);

	onDestroy(() => {
		if (previewUrl) URL.revokeObjectURL(previewUrl);
	});

	function handleCoverChange(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file || !file.type.startsWith('image/')) return;
		newAvatarFile = file;
		if (previewUrl) URL.revokeObjectURL(previewUrl);
		previewUrl = URL.createObjectURL(file);
	}

	$effect(() => startDefaultLocation((loc) => {
		locationName = loc?.location_name ?? '';
	}));

	/** 群组位置：来自 ClientConfig default_location，取不到就不传。 */
	async function resolveLocation() {
		return loadDefaultLocation().catch(() => null);
	}

	const handleCreate = async () => {
		if (!canCreate || creating) return;
		creating = true;
		createError = '';
		try {
			const location = await resolveLocation();
			// spec: groups/create 的 location 为必填字段；拿不到位置时明确报错，不发缺字段请求
			if (!location) {
				createError = '无法获取当前位置，请稍后重试';
				return;
			}
			let avatarFileId: string | undefined;
			if (newAvatarFile) {
				uploadingCover = true;
				try {
					avatarFileId = await uploadMediaFile(newAvatarFile, 'group_avatar');
				} catch {
					createError = '封面上传失败，请重试或先不传封面';
					return;
				} finally {
					uploadingCover = false;
				}
			}
			const res = await createGroup({
				name: name.trim(),
				description: desc.trim() || undefined,
				avatar_file_id: avatarFileId,
				tags: [category],
				join_method: isPrivate ? 'invite_only' : 'direct',
				requires_approval: requiresApproval,
				location
			});
			if (res.code !== 0) {
				createError = res.message ?? '创建失败，请稍后重试';
				return;
			}
			// spec: groups/create 直接返回 GroupItem { group: GroupInfo }
			const json = res as unknown as { data?: { group?: { group_id?: string }; group_id?: string } };
			const groupId = json.data?.group?.group_id ?? json.data?.group_id ?? '';
			step = 'done';
			onCreated?.(groupId);
		} catch {
			createError = '创建失败，请检查网络后重试';
		} finally {
			creating = false;
		}
	};
</script>

{#snippet headerExtra()}
	{#if onBack}
		<button
			type="button"
			class="hover-action hit-slop flex shrink-0 items-center gap-1 rounded-full px-2 py-1 text-xs text-text-secondary focus-ring"
			aria-label="返回发布瞬间"
			onclick={onBack}
		>
			<ArrowLeft class="size-3.5" strokeWidth={1.75} />
			返回
		</button>
	{/if}
{/snippet}

<SheetShell title="创建群组" {onClose} {headerExtra}>
	{#if step === 'done'}
		<div class="flex-1 flex flex-col items-center justify-center gap-4 px-8 text-center">
			<div
				class="flex items-center justify-center rounded-full bg-accent-default size-18 shadow-raised-md"
			>
				<Check class="size-8 text-text-on-accent"   strokeWidth={2.5} />
			</div>
			<div>
				<!-- 成功态大标题：仍是「标题」语义，字重跟 h1-h4 基线（font-medium） -->
				<p class="text-xl font-medium text-text-primary">群组已创建！</p>
				<p class="text-sm mt-1.5 text-text-secondary">「{name}」已成功创建，现在可以邀请好友加入了</p>
			</div>
			<Button type="button" size="lg" onclick={onClose}>完成</Button>
		</div>
	{:else}
		<div class="flex-1 overflow-y-auto no-scrollbar px-5 pt-4 pb-8 flex flex-col gap-5">
			<!-- Cover photo selector（与 GroupSettingsSheet 同一形态；创建后也可修改） -->
			<div class="flex items-center gap-4">
				<div class="relative shrink-0">
					{#if previewUrl}
						<Avatar src={previewUrl} name={name || '群'} size={64} shape="rounded" class="ring-1 ring-border-default" />
					{:else}
						<div class="flex size-16 items-center justify-center rounded-xl bg-accent-soft ring-1 ring-border-default" aria-hidden="true">
							<Camera class="size-5 text-text-secondary" strokeWidth={1.5} />
						</div>
					{/if}
					<button
						type="button"
						class="hover-btn hit-slop absolute -bottom-1 -right-1 flex size-7 items-center justify-center rounded-full border-strong border-surface-primary bg-accent-default shadow-raised-sm focus-ring hover:shadow-raised-md"
						onclick={() => coverInput?.click()}
						aria-label="选择群组封面"
					>
						<Camera class="size-3 text-text-on-accent" strokeWidth={2} />
					</button>
				</div>
				<div class="min-w-0">
					<p class="text-sm text-text-primary">群组封面</p>
					<p class="mt-0.5 text-xs text-text-secondary">可选，创建后也可在群设置中修改</p>
				</div>
				<input
					bind:this={coverInput}
					type="file"
					accept="image/*"
					class="hidden"
					onchange={handleCoverChange}
				/>
			</div>

			<!-- Group name（字符计数走原语 suffix，仍在壳内右端） -->
			<Input
				id="group-name"
				label="群组名称 *"
				value={name}
				onChange={(v) => (name = v)}
				placeholder="给你的群组起个名字…"
				maxlength={30}
			>
				{#snippet suffix()}
					<span class="text-xs shrink-0 text-text-secondary">{name.length}/30</span>
				{/snippet}
			</Input>

			<!-- Description（字符计数走原语 footer，壳内右下） -->
			<Textarea
				id="group-desc"
				label="简介"
				value={desc}
				onChange={(v) => (desc = v)}
				placeholder="描述一下这个群组的主题…"
				rows={2}
				maxlength={150}
				inputClass="min-h-18"
			>
				{#snippet footer()}
					<span class="text-xxs text-text-secondary">{desc.length}/150</span>
				{/snippet}
			</Textarea>

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
			<div
				class="flex items-center justify-between rounded-lg bg-surface-island px-4 py-3.5 shadow-xs"
			>
				<div class="flex items-center gap-3">
					{#if isPrivate}
						<Lock class="size-4.5 text-accent-default"   strokeWidth={1.5} />
					{:else}
						<Globe class="size-4.5 text-accent-default"   strokeWidth={1.5} />
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

			<!-- Approval toggle（requires_approval：审核通过后才算真正加入） -->
			<div
				class="flex items-center justify-between rounded-lg bg-surface-island px-4 py-3.5 shadow-xs"
			>
				<div class="flex items-center gap-3">
					<Check class="size-4.5 {requiresApproval ? 'text-accent-default' : 'text-text-secondary'}" strokeWidth={1.5} />
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

			<!-- Location：只读使用浏览位置，不渲染虚假的“可修改”箭头 -->
			<div
				class="flex items-center gap-2.5 rounded-lg bg-surface-island px-4 py-3 shadow-xs"
			>
				<MapPin class="size-4 shrink-0 text-accent-default" strokeWidth={1.5} />
				<div class="flex-1 min-w-0">
					<p class="text-sm text-text-primary">群组位置</p>
					<p class="text-xs text-text-secondary">
						{locationName || '获取位置中…'} · 使用当前浏览位置，暂不支持手动选择
					</p>
				</div>
			</div>

		</div>

		<!-- 粘底创建条：与发布器/群设置一致，主操作常驻可见，不被滚动区吞掉 -->
		<footer class="flex-none border-t border-border-default bg-surface-primary px-5 py-3">
			{#if createError}
				<p class="mb-2 text-sm text-status-danger">{createError}</p>
			{/if}
			<Button
				type="button"
				onclick={() => void handleCreate()}
				disabled={!canCreate || creating}
				loading={creating}
				size="lg"
				class="w-full"
			>
				{uploadingCover ? '上传封面…' : creating ? '创建中…' : '创建群组'}
			</Button>
		</footer>
	{/if}
</SheetShell>
