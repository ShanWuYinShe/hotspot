<script lang="ts">
	import { Hash, Lock, MapPin, Smile, Users, X } from '@lucide/svelte';
	import type { Component } from 'svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import ImagePicker from '@/components/forms/ImagePicker.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { loadDefaultLocation } from '@/lib/default-location';
	import { createMoment } from '@/api';
	import { uploadMediaFiles } from '@/lib/upload';
	import { showToast } from '@/stores/toast.svelte';

	// 三端共用的发布器（手机 Sheet / 平板 Sheet / 桌面对话框）。
	// 对齐 Figma Make：header 提供「创建群组」surface 切换。

	import CreateGroupSheet from '@/components/overlays/CreateGroupSheet.svelte';

	let { onClose }: { onClose: () => void } = $props();

	let surface = $state<'moment' | 'group'>('moment');

	const MOOD_TAGS = ['松弛', '漫游', '咖啡', '展览', '自然', '夜色'];

	const AUDIENCES: { id: 'nearby' | 'friends' | 'private'; label: string; hint: string; icon: Component }[] = [
		{ id: 'nearby', label: '附近可见', hint: '进入附近动态', icon: MapPin },
		{ id: 'friends', label: '好友可见', hint: '仅你的好友', icon: Users },
		{ id: 'private', label: '仅自己', hint: '保存在我的瞬间', icon: Lock }
	];

	let text = $state('');
	let selectedMoods = $state<Set<string>>(new Set());
	let imageFiles = $state<File[]>([]);
	let locationName = $state('');
	let audience = $state<'nearby' | 'friends' | 'private'>('nearby');
	let topicActive = $state(false);
	let submitting = $state(false);
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);

	$effect(() => startDefaultLocation((loc) => {
		referencePoint = loc?.point ?? null;
		locationName = loc?.location_name ?? '';
	}));

	const toggleMood = (tag: string) => {
		const next = new Set(selectedMoods);
		next.has(tag) ? next.delete(tag) : next.add(tag);
		selectedMoods = next;
	};

	const appendText = (snippet: string) => {
		if (text.includes(snippet)) return;
		const separator = text.trim().length ? '\n' : '';
		text = `${text}${separator}${snippet}`.slice(0, 280);
	};

	const toggleTopic = () => {
		const topic = locationName ? `#${locationName}` : '';
		if (!topic) return;
		if (topicActive) {
			text = text.replace(topic, '').replace(/\n\n+/g, '\n').trimEnd();
		} else {
			appendText(topic);
		}
		topicActive = !topicActive;
	};

	const canPublish = $derived(text.trim().length > 0 && !submitting);
	const activeAudience = $derived(AUDIENCES.find((item) => item.id === audience) ?? AUDIENCES[0]);

	const handleSubmit = async () => {
		if (!canPublish) return;
		submitting = true;
		try {
			// 多图上传（spec：batch_presign → 并发 PUT → batch_complete），拿 file_id[] 随动态一起创建
			const image_file_ids = imageFiles.length > 0
				? await uploadMediaFiles(imageFiles, 'moment_image')
				: undefined;
			const res = await createMoment({
				description: text.trim(),
				tags: Array.from(selectedMoods),
				// spec：location 必填（types.md 契约级）。优先用户参考点，缺失时回退
				// ClientConfig default_location；两者皆缺（配置不可用）才退化为零点。
				location: {
					point:
						referencePoint ??
						(await loadDefaultLocation())?.point ?? { latitude: 0, longitude: 0 },
					location_name: locationName || ''
				},
				// spec PrivacyLevel 仅有 'public' | 'private'：nearby→public，其余（含 UI 的 friends 占位）→private
				privacy_level: audience === 'nearby' ? 'public' : 'private',
				image_file_ids
			});
			if (res.code === 0) {
				showToast('发布成功', 'success');
				onClose();
			} else {
				// 业务失败（非抛错路径）：透出服务端信息
				showToast(res.message ?? '发布失败，请稍后重试', 'danger');
			}
		} catch {
			// 网络/传输失败（client 层非 200 会 throw）
			showToast('发布失败，请检查网络后重试', 'danger');
		} finally {
			submitting = false;
		}
	};
</script>

{#if surface === 'moment'}
<form
	class="flex h-full w-full flex-col overflow-hidden bg-surface-primary text-text-primary"
	onsubmit={(event) => {
		event.preventDefault();
		void handleSubmit();
	}}
>
	<header class="flex-none border-b border-border-default px-5 pb-3 pt-4">
		<div class="mb-3 flex justify-center" aria-hidden="true">
			<span class="h-1 w-9 rounded-full bg-border-strong"></span>
		</div>
		<div class="flex items-center justify-between gap-4">
			<div class="min-w-0">
				<p class="eyebrow">附近动态</p>
				<h1 id="create-composer-title" class="mt-1 text-xl font-medium leading-none tracking-title text-text-primary">
					记录此刻
				</h1>
			</div>
			<div class="flex items-center gap-2">
				<button
					type="button"
					onclick={() => (surface = 'group')}
					class="rounded-full border border-accent-border bg-accent-soft px-3 py-1.5 text-xs font-medium text-accent-default outline-none transition-colors hover:bg-accent-default hover:text-text-on-accent focus-visible:ring-2 focus-visible:ring-accent-default"
				>
					创建群组
				</button>
				<ActionIconButton icon={X} label="关闭发布器" onClick={onClose} size={36} />
			</div>
		</div>
	</header>

	<div class="no-scrollbar flex-1 overflow-y-auto px-5 py-4">
		<div class="mx-auto flex w-full max-w-140 flex-col gap-4">
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-2.5">
					<Avatar src={auth.user?.avatar_url} name={auth.user?.nickname} size={36} class="border border-border-default" />
					<div class="min-w-0">
						<p class="text-sm font-medium text-text-primary">{auth.user?.nickname || '用户'}</p>
						<span class="mt-0.5 flex items-center gap-1 text-xs text-text-secondary">
							<MapPin size={12} class="text-accent-default" strokeWidth={1.75} />
							<span class="truncate">{locationName || '当前位置'}</span>
						</span>
					</div>
				</div>
				<span class="rounded-full border border-accent-border bg-accent-soft px-2.5 py-1 text-xxs font-medium text-accent-default">
					正在发布
				</span>
			</div>

			<div class="input-surface rounded-lg p-4">
				<textarea
					bind:value={text}
					oninput={(e) => (text = e.currentTarget.value.slice(0, 280))}
					placeholder="写下此刻发生的事，或分享一个想被附近的人看见的瞬间…"
					class="min-h-30.5 w-full resize-none bg-transparent text-sm leading-7 text-text-primary placeholder:text-text-secondary"
					aria-label="动态内容"
				></textarea>
				<div class="mt-3 flex items-center justify-between border-t border-border-default pt-3">
					<div class="flex items-center gap-1">
						<button
							type="button"
							onclick={() => appendText('✨')}
							aria-label="添加心情"
							class="flex h-8 w-8 items-center justify-center rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default text-text-secondary hover:bg-interactive-hover"
					>
						<Smile size={16} strokeWidth={1.5} />
					</button>
					<button
						type="button"
						onclick={toggleTopic}
						aria-label="添加话题"
						aria-pressed={topicActive}
						class="flex h-8 w-8 items-center justify-center rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default {topicActive
							? 'bg-accent-soft text-accent-default hover:bg-interactive-active'
							: 'text-text-secondary hover:bg-interactive-hover'}"
					>
						<Hash size={16} strokeWidth={topicActive ? 2 : 1.5} />
					</button>
					</div>
					<span class="text-xs tabular-nums {text.length >= 260 ? 'text-status-danger' : 'text-text-secondary'}">
						{text.length}/280
					</span>
				</div>
			</div>

			<section aria-labelledby="image-label">
				<p id="image-label" class="mb-2 eyebrow">照片</p>
				<ImagePicker files={imageFiles} onFilesChange={(f) => (imageFiles = f)} max={9} />
			</section>

			<section aria-labelledby="mood-label">
				<div class="mb-2 flex items-center justify-between">
					<p id="mood-label" class="eyebrow">此刻氛围</p>
					{#if selectedMoods.size > 0}
						<span class="text-xxs text-text-secondary">已选 {selectedMoods.size} 项</span>
					{/if}
				</div>
				<div class="flex flex-wrap gap-2">
					{#each MOOD_TAGS as tag}
						{@const active = selectedMoods.has(tag)}
						<TagPill active={active} onClick={() => toggleMood(tag)}>
							{tag}
						</TagPill>
					{/each}
				</div>
			</section>

			<section
				aria-labelledby="audience-label"
				class="rounded-md ring-1 ring-border-default surface-tinted p-2"
			>
				<div class="flex items-center justify-between px-2 pb-2 pt-1">
					<p id="audience-label" class="eyebrow">谁可以看到</p>
					<span class="text-xxs text-text-secondary">{activeAudience.hint}</span>
				</div>
				<div class="grid grid-cols-3 gap-1">
					{#each AUDIENCES as { id, label, icon: AudienceIcon }}
						{@const active = audience === id}
						<button
							type="button"
							onclick={() => (audience = id)}
							aria-pressed={active}
							class="flex min-w-0 flex-col items-center gap-1 rounded-sm px-1 py-2 text-center outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default {active
								? 'bg-accent-soft text-accent-default hover:bg-interactive-active'
								: 'text-text-secondary hover:bg-interactive-hover'}"
						>
							<AudienceIcon size={14} strokeWidth={active ? 2 : 1.5} />
							<span class="truncate text-xxs font-medium">{label}</span>
						</button>
					{/each}
				</div>
			</section>
		</div>
	</div>

	<footer class="flex flex-none items-center gap-3 border-t border-border-default bg-surface-primary px-5 py-3">
		<div class="min-w-0 flex-1">
			<p class="truncate text-xs text-text-secondary">
				{selectedMoods.size ? `将以 ${Array.from(selectedMoods).join(' · ')} 的氛围出现` : '选择氛围，让附近的人更容易找到你'}
			</p>
		</div>
		<button
			type="submit"
			disabled={!canPublish}
			class="hover-btn shrink-0 rounded-full px-5 py-2.5 text-sm font-medium outline-none transition-all focus-visible:ring-2 focus-visible:ring-accent-default disabled:cursor-not-allowed disabled:border disabled:border-border-default disabled:bg-transparent disabled:text-text-secondary enabled:bg-accent-default enabled:text-text-on-accent enabled:shadow-glow-md enabled:hover:shadow-glow-lg"
		>
			发布动态
		</button>
	</footer>
</form>
{:else}
	<!-- 创建群组 surface（对齐 Figma Make CreateSheet 的 surface 切换；可返回发布动态） -->
	<CreateGroupSheet onClose={onClose} onBack={() => (surface = 'moment')} />
{/if}
