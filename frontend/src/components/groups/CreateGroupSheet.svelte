<script lang="ts">
	import { X, Camera, Globe, Lock, Check, ChevronRight, MapPin } from '@lucide/svelte';
	import { createGroup } from '@/api';
	import { loadDefaultLocation } from '@/lib/default-location';

	// CreateGroupSheet — 创建群组的底部 Sheet / 模态。
	// 使用处：NearbyGroups(FAB)、Profile > 我的群组（页头按钮）、
	//        以及 NewConversationSheet 的「创建群聊」入口。
	// 创建走 groups/create（返回 GroupItem，group_id 经 onCreated 回调交给调用方）。

	let { onClose, onCreated }: { onClose: () => void; onCreated?: (groupId: string) => void } = $props();

	const CATEGORIES = ['摄影', '音乐', '户外', '阅读', '美食', '健身', '游戏', '其他'];

	let name = $state('');
	let desc = $state('');
	let category = $state('摄影');
	let isPrivate = $state(false);
	let step = $state<'form' | 'done'>('form');
	let creating = $state(false);
	let createError = $state('');

	const canCreate = $derived(name.trim().length >= 2);

	/** 群组位置：默认位置不可得时回退徐州（与 ClientConfig.defaults 一致）。 */
	async function resolveLocation() {
		const loc = await loadDefaultLocation().catch(() => null);
		return loc ?? { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: '徐州' };
	}

	const handleCreate = async () => {
		if (!canCreate || creating) return;
		creating = true;
		createError = '';
		try {
			const location = await resolveLocation();
			const res = await createGroup({
				name: name.trim(),
				description: desc.trim() || undefined,
				tags: [category],
				join_method: isPrivate ? 'invite_only' : 'open',
				location
			});
			if (res.code !== 0) {
				createError = res.message ?? '创建失败，请稍后重试';
				return;
			}
			// spec: groups/create 直接返回 GroupItem { group: GroupInfo }
			const json = res as unknown as { data?: any };
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

<div class="flex flex-col h-full bg-[var(--neu-bg)]">
	<!-- Handle -->
	<div class="flex justify-center pt-3 pb-1 flex-shrink-0">
		<div class="rounded-full bg-[var(--neu-border)]" style="width:36px;height:4px;"></div>
	</div>

	<!-- Header -->
	<div
		class="flex items-center justify-between px-5 py-3 flex-shrink-0"
		style="border-bottom:1px solid var(--neu-border);"
	>
		<h2 class="text-[17px] text-[var(--neu-text)]">创建群组</h2>
		<button
			type="button"
			onclick={onClose}
			class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none bg-[var(--neu-bg)]"
			style="width:32px;height:32px;box-shadow:3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light);border:1px solid var(--neu-border);"
		>
			<X style="width:14px;height:14px;color:var(--neu-text-muted);" strokeWidth={2} />
		</button>
	</div>

	{#if step === 'done'}
		<div class="flex-1 flex flex-col items-center justify-center gap-4 px-8 text-center">
			<div
				class="flex items-center justify-center rounded-full bg-[var(--neu-accent-primary)]"
				style="width:72px;height:72px;box-shadow:0 8px 24px var(--neu-glow);"
			>
				<Check style="width:32px;height:32px;color:white;" strokeWidth={2.5} />
			</div>
			<div>
				<p class="text-[20px] font-semibold text-[var(--neu-text)]">群组已创建！</p>
				<p class="text-[13px] mt-1.5 text-[var(--neu-text-muted)]">「{name}」已成功创建，现在可以邀请好友加入了</p>
			</div>
			<button
				type="button"
				onclick={onClose}
				class="neu-hover-btn px-8 py-3 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white text-[15px] font-medium"
				style="box-shadow:0 6px 20px var(--neu-glow);"
			>
				完成
			</button>
		</div>
	{:else}
		<div class="flex-1 overflow-y-auto no-scrollbar px-5 pt-4 pb-8 flex flex-col gap-5">
			<!-- Cover photo selector -->
			<div class="flex justify-center">
				<button
					type="button"
					class="neu-hover-btn flex flex-col items-center justify-center gap-2 rounded-[22px] outline-none bg-[var(--neu-bg)]"
					style="width:100%;height:110px;box-shadow:inset 4px 4px 10px var(--neu-shadow-dark), inset -4px -4px 10px var(--neu-shadow-light);border:2px dashed var(--neu-border);"
				>
					<Camera style="width:26px;height:26px;color:var(--neu-text-muted);" strokeWidth={1.25} />
					<span class="text-[12px] text-[var(--neu-text-muted)]">添加封面图片</span>
				</button>
			</div>

			<!-- Group name -->
			<div>
				<label class="text-[11px] tracking-[0.1em] uppercase text-[var(--neu-text-muted)]" for="group-name">群组名称 *</label>
				<div class="flex items-center gap-2.5 mt-1.5 px-4 rounded-[14px] neu-input-surface" style="height:46px;">
					<input
						id="group-name"
						class="flex-1 bg-transparent outline-none text-[var(--neu-text)] text-[14px]"
						placeholder="给你的群组起个名字…"
						bind:value={name}
						maxlength={30}
					/>
					<span class="text-[11px] shrink-0 text-[var(--neu-text-muted)]">{name.length}/30</span>
				</div>
			</div>

			<!-- Description -->
			<div>
				<label class="text-[11px] tracking-[0.1em] uppercase text-[var(--neu-text-muted)]" for="group-desc">简介</label>
				<div class="mt-1.5 px-4 py-3 rounded-[14px] neu-input-surface">
					<textarea
						id="group-desc"
						class="w-full bg-transparent outline-none resize-none text-[var(--neu-text)] text-[13px] min-h-[72px]"
						placeholder="描述一下这个群组的主题…"
						bind:value={desc}
						maxlength={150}
					></textarea>
					<div class="flex justify-end">
						<span class="text-[10px] text-[var(--neu-text-muted)]">{desc.length}/150</span>
					</div>
				</div>
			</div>

			<!-- Category chips -->
			<fieldset class="flex flex-col border-0 p-0 m-0 min-w-0">
				<legend class="text-[11px] tracking-[0.1em] uppercase text-[var(--neu-text-muted)]">分类</legend>
				<div class="flex flex-wrap gap-2 mt-2">
					{#each CATEGORIES as cat}
						<button
							type="button"
							onclick={() => (category = cat)}
							class="neu-hover-btn px-3.5 py-1.5 rounded-full outline-none text-[12px] {category === cat ? 'bg-[var(--neu-accent-primary)] text-white font-medium' : 'bg-[var(--neu-bg)] text-[var(--neu-text-muted)]'}"
							style={category === cat
								? 'box-shadow:0 3px 8px var(--neu-glow);'
								: 'box-shadow:3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light);border:1px solid var(--neu-border);'}
						>
							{cat}
						</button>
					{/each}
				</div>
			</fieldset>

			<!-- Privacy toggle -->
			<div
				class="flex items-center justify-between px-4 py-3.5 rounded-[16px] bg-[var(--neu-bg)]"
				style="box-shadow:5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light);border:1px solid var(--neu-border);"
			>
				<div class="flex items-center gap-3">
					{#if isPrivate}
						<Lock style="width:18px;height:18px;color:var(--neu-accent-primary);" strokeWidth={1.5} />
					{:else}
						<Globe style="width:18px;height:18px;color:var(--neu-accent-primary);" strokeWidth={1.5} />
					{/if}
					<div>
						<p class="text-[14px] text-[var(--neu-text)]">{isPrivate ? '私密群组' : '公开群组'}</p>
						<p class="text-[11px] text-[var(--neu-text-muted)]">
							{isPrivate ? '仅受邀请的人可加入' : '所有人都可以找到并加入'}
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => (isPrivate = !isPrivate)}
					class="neu-hover-btn flex-shrink-0 rounded-full outline-none"
					style="width:48px;height:28px;background:{isPrivate ? 'var(--neu-accent-primary)' : 'rgba(120,120,128,0.25)'};position:relative;transition:background 0.22s ease;box-shadow:{isPrivate ? 'inset 0 1px 2px rgba(0,0,0,0.12)' : 'inset 0 1px 2px rgba(0,0,0,0.08)'};"
					aria-label={isPrivate ? '切换为公开' : '切换为私密'}
				>
					<span
						class="absolute bg-white rounded-full"
						style="top:3px;width:22px;height:22px;border-radius:11px;box-shadow:0 1px 4px rgba(0,0,0,0.26);left:{isPrivate ? 22 : 2}px;transition:left 0.2s cubic-bezier(0.34,1.56,0.64,1);"
					></span>
				</button>
			</div>

			<!-- Location -->
			<div
				class="flex items-center gap-2.5 px-4 py-3 rounded-[16px] bg-[var(--neu-bg)]"
				style="box-shadow:5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light);border:1px solid var(--neu-border);"
			>
				<MapPin style="width:16px;height:16px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={1.5} />
				<div class="flex-1 min-w-0">
					<p class="text-[13px] text-[var(--neu-text)]">徐州市 · 云龙区</p>
					<p class="text-[11px] text-[var(--neu-text-muted)]">自动定位 — 可修改</p>
				</div>
				<ChevronRight style="width:13px;height:13px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</div>

			{#if createError}
				<p class="text-[12px] text-[var(--semantic-warning)]">{createError}</p>
			{/if}

			<!-- Create button -->
			<button
				type="button"
				onclick={() => void handleCreate()}
				disabled={!canCreate || creating}
				class="neu-hover-btn flex items-center justify-center gap-2 w-full py-4 rounded-[18px] outline-none text-[15px] font-medium {canCreate && !creating ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-[var(--neu-bg)] text-[var(--neu-text-muted)]'}"
				style="box-shadow:{canCreate && !creating ? '0 6px 22px var(--neu-glow)' : 'inset 4px 4px 10px var(--neu-shadow-dark), inset -4px -4px 10px var(--neu-shadow-light)'};border:{canCreate && !creating ? 'none' : '1px solid var(--neu-border)'};transition:all 0.22s ease;cursor:{canCreate && !creating ? 'pointer' : 'not-allowed'};"
			>
				{creating ? '创建中…' : '创建群组'}
			</button>
		</div>
	{/if}
</div>
