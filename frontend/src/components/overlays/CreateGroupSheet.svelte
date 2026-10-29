<script lang="ts">
	import { Camera, Globe, Lock, Check, ChevronRight, MapPin } from '@lucide/svelte';
	import { createGroup } from '@/api';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { loadDefaultLocation } from '@/lib/default-location';
	import Toggle from '@/components/forms/Toggle.svelte';
	import Input from '@/components/forms/Input.svelte';
	import Textarea from '@/components/forms/Textarea.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import SheetShell from '@/components/overlays/SheetShell.svelte';
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

	const canCreate = $derived(name.trim().length >= 2);

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
			const res = await createGroup({
				name: name.trim(),
				description: desc.trim() || undefined,
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

<SheetShell title="创建群组" {onClose}>
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
			<!-- Cover photo selector -->
			<div class="flex justify-center">
				<!-- disabled 必须可见可解释：视觉降级 + 行内「即将开放」说明（title 在触屏不可见） -->
				<button
					type="button"
					class="flex flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-border-default bg-surface-primary shadow-inset-sm hover:border-accent-border hover:bg-accent-soft focus-ring disabled:cursor-not-allowed disabled:opacity-faint w-full h-28"
					disabled
					aria-disabled="true"
					title="暂未开放"
				>
					<Camera class="size-6.5 text-text-secondary" strokeWidth={1.5} />
					<span class="text-sm text-text-secondary">添加封面图片</span>
					<span class="text-xxs text-text-secondary">即将开放</span>
				</button>
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
				class="flex items-center justify-between rounded-md border border-border-default bg-surface-primary px-4 py-3.5 shadow-raised-md"
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
				class="flex items-center justify-between rounded-md border border-border-default bg-surface-primary px-4 py-3.5 shadow-raised-md"
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

			<!-- Location -->
			<div
				class="flex items-center gap-2.5 rounded-md border border-border-default bg-surface-primary px-4 py-3 shadow-raised-md"
			>
				<MapPin class="size-4 shrink-0 text-accent-default"   strokeWidth={1.5} />
				<div class="flex-1 min-w-0">
					<p class="text-sm text-text-primary">{locationName || '自动定位中…'}</p>
					<p class="text-xs text-text-secondary">自动定位 — 可修改</p>
				</div>
				<ChevronRight class="size-3.5 text-text-secondary"   strokeWidth={1.5} />
			</div>

			{#if createError}
				<p class="text-sm text-status-danger">{createError}</p>
			{/if}

			<!-- Create button -->
			<Button
				type="button"
				onclick={() => void handleCreate()}
				disabled={!canCreate || creating}
				size="lg"
				class="w-full"
			>
				{creating ? '创建中…' : '创建群组'}
			</Button>
	</div>
{/if}
</SheetShell>
