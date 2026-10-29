<script lang="ts">
	import { Hash, Lock, MapPin, Smile, X } from '@lucide/svelte';
	import type { Component } from 'svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import ImagePicker from '@/components/forms/ImagePicker.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import Textarea from '@/components/forms/Textarea.svelte';
	import { auth } from '@/stores/auth.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { loadDefaultLocation } from '@/lib/default-location';
	import { clearMomentsProgress } from '@/lib/moments-feed-progress.svelte';
	import { createMoment } from '@/api';
	import { uploadMediaFiles } from '@/lib/upload';
	import { showToast } from '@/stores/toast.svelte';
	import { draft, clearDraft } from '@/stores/composer.svelte';
	import Button from '@/components/ui/button.svelte';

	// 三端共用的发布器（手机 Sheet / 平板 Sheet / 桌面对话框）。
	// 对齐 Figma Make：header 提供「创建群组」surface 切换。

	import CreateGroupSheet from '@/components/overlays/CreateGroupSheet.svelte';

	let { onClose }: { onClose: () => void } = $props();

	let surface = $state<'moment' | 'group'>('moment');

	const MOOD_TAGS = ['松弛', '漫游', '咖啡', '展览', '自然', '夜色'];

	// spec PrivacyLevel 仅有 'public' | 'private'（api_spec_types.md）：曾展示的
	// 「好友可见」选项后端并无对应档位（会静默存成仅自己，好友看不到），在服务端
	// 补齐 friends 档之前必须摘掉，不能假装提供该能力。
	const AUDIENCES: { id: 'nearby' | 'private'; label: string; hint: string; icon: Component }[] = [
		{ id: 'nearby', label: '附近可见', hint: '进入附近瞬间', icon: MapPin },
		{ id: 'private', label: '仅自己', hint: '保存在我的瞬间', icon: Lock }
	];

	// 草稿字段走 composer store（跨弹层挂载存活，关闭重开不丢稿）；
	// 位置每次重开重新定位（不存草稿，避免陈旧定位）。
	let locationName = $state('');
	let topicActive = $state(false);
	let submitting = $state(false);
	// 发布阶段提示（审计 P2-8：大图慢网上传期间给到第 n 张 / 百分比反馈）
	let submitStatus = $state('');
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);

	$effect(() => startDefaultLocation((loc) => {
		referencePoint = loc?.point ?? null;
		locationName = loc?.location_name ?? '';
	}));

	const toggleMood = (tag: string) => {
		const next = draft.moods.includes(tag) ? draft.moods.filter((t) => t !== tag) : [...draft.moods, tag];
		draft.moods = next;
	};

	const appendText = (snippet: string) => {
		if (draft.text.includes(snippet)) return;
		const separator = draft.text.trim().length ? '\n' : '';
		draft.text = `${draft.text}${separator}${snippet}`.slice(0, 280);
	};

	const toggleTopic = () => {
		const topic = locationName ? `#${locationName}` : '';
		if (!topic) return;
		if (topicActive) {
			draft.text = draft.text.replace(topic, '').replace(/\n\n+/g, '\n').trimEnd();
		} else {
			appendText(topic);
		}
		topicActive = !topicActive;
	};

	const canPublish = $derived(draft.text.trim().length > 0 && !submitting);
	const activeAudience = $derived(AUDIENCES.find((item) => item.id === draft.audience) ?? AUDIENCES[0]);

	const handleSubmit = async () => {
		if (!canPublish) return;
		submitting = true;
		submitStatus = '';
		try {
			// 多图上传（spec：batch_presign → 并发 PUT → batch_complete），拿 file_id[] 随动态一起创建
			const image_file_ids = draft.images.length > 0
				? await uploadMediaFiles(draft.images, 'moment_image', (fileIndex, fraction) => {
						submitStatus = `正在上传第 ${fileIndex + 1}/${draft.images.length} 张 · ${Math.round(fraction * 100)}%`;
					})
				: undefined;
			submitStatus = '正在发布…';
			const res = await createMoment({
				description: draft.text.trim(),
				tags: [...draft.moods],
				// spec：location 必填（types.md 契约级）。优先用户参考点，缺失时回退
				// ClientConfig default_location；两者皆缺（配置不可用）才退化为零点。
				location: {
					point:
						referencePoint ??
						(await loadDefaultLocation())?.point ?? { latitude: 0, longitude: 0 },
					location_name: locationName || ''
				},
				privacy_level: draft.audience === 'nearby' ? 'public' : 'private',
				image_file_ids
			});
			if (res.code === 0) {
				showToast('发布成功', 'success');
				// 附近瞬间 feed 的会话进度缓存立即失效：否则发布后 5 分钟内
				// （TTL）列表仍采用发布前的旧集合，用户看不到自己刚发的内容。
				clearMomentsProgress();
				clearDraft();
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
			submitStatus = '';
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
				<p class="eyebrow">附近瞬间</p>
				<h2 id="create-composer-title" class="mt-1 text-xl font-medium leading-none tracking-title text-text-primary">
					记录此刻
				</h2>
			</div>
			<div class="flex items-center gap-2">
				<Button type="button" variant="surface-accent" size="sm" onclick={() => (surface = 'group')}>
					创建群组
				</Button>
				<ActionIconButton icon={X} label="关闭发布器" onClick={onClose} size={36} />
			</div>
		</div>
	</header>

	<div class="no-scrollbar flex-1 overflow-y-auto px-5 py-4">
		<div class="mx-auto flex w-full max-w-140 flex-col gap-4">
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-2.5">
					<Avatar src={auth.user?.avatar_url} name={auth.user?.nickname} size="md" class="border border-border-default" />
					<div class="min-w-0">
						<p class="text-sm font-medium text-text-primary">{auth.user?.nickname || '用户'}</p>
						<span class="mt-0.5 flex items-center gap-1 text-xs text-text-secondary">
							<MapPin size={12} class="text-accent-default" strokeWidth={1.75} />
							<span class="truncate">{locationName || '当前位置'}</span>
						</span>
					</div>
				</div>
				{#if submitting}
					<Badge tone="accent">{submitStatus || '正在发布…'}</Badge>
				{/if}
			</div>

			<!-- 正文 + 壳内工具条：正文统一走 forms/Textarea（壳 px-3.5 py-3），工具条与计数走 footer -->
			<Textarea
				value={draft.text}
				onChange={(v) => (draft.text = v.slice(0, 280))}
				placeholder="写下此刻发生的事，或分享一个想被附近的人看见的瞬间…"
				ariaLabel="瞬间内容"
				inputClass="min-h-30.5"
			>
				{#snippet footer()}
					<div class="flex w-full items-center justify-between border-t border-border-default pt-3">
						<div class="flex items-center gap-1">
							<button
								type="button"
								onclick={() => appendText('✨')}
								aria-label="添加心情"
								class="hit-slop flex h-8 w-8 items-center justify-center rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default text-text-secondary hover:bg-interactive-hover"
							>
								<Smile size={16} strokeWidth={1.5} />
							</button>
							<button
								type="button"
								onclick={toggleTopic}
								aria-label="添加话题"
								aria-pressed={topicActive}
								class="hit-slop flex h-8 w-8 items-center justify-center rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default {topicActive
									? 'bg-accent-soft text-text-accent-on-soft hover:bg-interactive-active'
									: 'text-text-secondary hover:bg-interactive-hover'}"
							>
								<Hash size={16} strokeWidth={topicActive ? 2 : 1.5} />
							</button>
						</div>
						<span class="text-xs tabular-nums {draft.text.length >= 260 ? 'text-status-danger' : 'text-text-secondary'}">
							{draft.text.length}/280
						</span>
					</div>
				{/snippet}
			</Textarea>

			<section aria-labelledby="image-label">
				<p id="image-label" class="mb-2 eyebrow">照片</p>
				<ImagePicker files={draft.images} onFilesChange={(f) => (draft.images = f)} max={9} />
			</section>

			<section aria-labelledby="mood-label">
				<div class="mb-2 flex items-center justify-between">
					<p id="mood-label" class="eyebrow">此刻氛围</p>
					{#if draft.moods.length > 0}
						<span class="text-xxs text-text-secondary">已选 {draft.moods.length} 项</span>
					{/if}
				</div>
				<div class="flex flex-wrap gap-2">
					{#each MOOD_TAGS as tag}
						{@const active = draft.moods.includes(tag)}
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
				<!-- 可见范围统一走注册原语 TabBar segmented（与瞬间编辑弹层同形）：
				     旧实现是两列自绘按钮（扁条 + text-xxs + 移动端热区不足），属同类型组件双实现。 -->
				<TabBar
					variant="segmented"
					items={AUDIENCES.map(({ id, label, icon }) => ({ id, label, icon }))}
					value={draft.audience}
					onChange={(id) => (draft.audience = id as 'nearby' | 'private')}
				/>
			</section>
		</div>
	</div>

	<footer class="flex flex-none items-center gap-3 border-t border-border-default bg-surface-primary px-5 py-3">
		<div class="min-w-0 flex-1">
			<p class="truncate text-xs text-text-secondary">
				{draft.moods.length ? `将以 ${draft.moods.join(' · ')} 的氛围出现` : '选择氛围，让附近的人更容易找到你'}
			</p>
		</div>
		<Button type="submit" size="md" disabled={!canPublish} class="px-5">
			{submitStatus || '发布瞬间'}
		</Button>
	</footer>
</form>
{:else}
	<!-- 创建群组 surface（对齐 Figma Make CreateSheet 的 surface 切换；可返回发布瞬间） -->
	<CreateGroupSheet onClose={onClose} onBack={() => (surface = 'moment')} />
{/if}
