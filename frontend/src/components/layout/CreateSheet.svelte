<script lang="ts">
	import { Hash, ImagePlus, Lock, MapPin, Smile, Users, X } from '@lucide/svelte';
	import type { Component } from 'svelte';
	import ActionIconButton from '@/components/shared/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { loadDefaultLocation, type GeoPoint } from '@/lib/default-location';
	import { createMoment } from '@/api';

	// 三端共用的发布器（手机 Sheet / 平板 Sheet / 桌面对话框）。
	// 迁移自 React CreateSheet.tsx。

	let { onClose }: { onClose: () => void } = $props();

	const MOOD_TAGS = ['松弛', '漫游', '咖啡', '展览', '自然', '夜色'];
	const LOCATIONS = ['徐州 · 云龙湖附近', '徐州 · 回龙窝', '徐州 · 彭城广场'];

	const AUDIENCES: { id: 'nearby' | 'friends' | 'private'; label: string; hint: string; icon: Component }[] = [
		{ id: 'nearby', label: '附近可见', hint: '进入附近动态', icon: MapPin },
		{ id: 'friends', label: '好友可见', hint: '仅你的好友', icon: Users },
		{ id: 'private', label: '仅自己', hint: '保存在我的瞬间', icon: Lock }
	];

	let text = $state('');
	let selectedMoods = $state<Set<string>>(new Set());
	let hasImage = $state(false);
	let locationIndex = $state(0);
	let audience = $state<'nearby' | 'friends' | 'private'>('nearby');
	let topicActive = $state(false);
	let submitting = $state(false);
	let referencePoint = $state<GeoPoint | null>(null);

	$effect(() => {
		let cancelled = false;
		void loadDefaultLocation().then((loc) => {
			if (!cancelled) referencePoint = loc?.point ?? null;
		});
		return () => {
			cancelled = true;
		};
	});

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
		const topic = '#云龙湖漫游';
		if (topicActive) {
			text = text.replace(topic, '').replace(/\n\n+/g, '\n').trimEnd();
		} else {
			appendText(topic);
		}
		topicActive = !topicActive;
	};

	const canPost = $derived(text.trim().length > 0 && !submitting);
	const activeAudience = $derived(AUDIENCES.find((item) => item.id === audience) ?? AUDIENCES[0]);

	const handleSubmit = async () => {
		if (!canPost) return;
		submitting = true;
		try {
			const res = await createMoment({
				description: text.trim(),
				tags: Array.from(selectedMoods),
				// spec: location requires point + location_name. Point from ClientConfig default_location.
				location: referencePoint
					? { point: referencePoint, location_name: LOCATIONS[locationIndex] }
					: { point: { latitude: 0, longitude: 0 }, location_name: LOCATIONS[locationIndex] },
				// spec PrivacyLevel: nearby→public, friends→friends, private→private
				privacy_level: audience === 'nearby' ? 'public' : audience
			});
			if (res.code === 0) {
				onClose();
			}
		} finally {
			submitting = false;
		}
	};
</script>

<form
	class="flex h-full w-full flex-col overflow-hidden bg-[var(--neu-bg)] text-[var(--neu-text)]"
	onsubmit={(event) => {
		event.preventDefault();
		void handleSubmit();
	}}
>
	<header class="flex-none border-b border-[var(--neu-border)] px-5 pb-3 pt-4">
		<div class="mb-3 flex justify-center" aria-hidden="true">
			<span class="h-1 w-9 rounded-full bg-[var(--neu-border-pressed)]"></span>
		</div>
		<div class="flex items-center justify-between gap-4">
			<div class="min-w-0">
				<p class="neu-eyebrow">附近动态</p>
				<h1 id="create-composer-title" class="mt-1 text-[20px] font-medium leading-none tracking-[-0.01em] text-[var(--neu-text)]">
					记录此刻
				</h1>
			</div>
			<ActionIconButton icon={X} label="关闭发布器" onClick={onClose} size={36} />
		</div>
	</header>

	<div class="no-scrollbar flex-1 overflow-y-auto px-5 py-4">
		<div class="mx-auto flex w-full max-w-[560px] flex-col gap-4">
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-2.5">
					<Avatar src={auth.user?.avatar_url} name={auth.user?.nickname} size={36} className="border border-[var(--neu-border)]" />
					<div class="min-w-0">
						<p class="text-[13px] font-medium text-[var(--neu-text)]">{auth.user?.nickname || '用户'}</p>
						<button
							type="button"
							onclick={() => (locationIndex = (locationIndex + 1) % LOCATIONS.length)}
							class="mt-0.5 flex items-center gap-1 rounded-md text-left text-[11px] text-[var(--neu-text-muted)] outline-none transition-colors hover:text-[var(--neu-accent-primary)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
							aria-label="切换发布位置"
						>
							<MapPin size={12} class="text-[var(--neu-accent-primary)]" strokeWidth={1.75} />
							<span class="truncate">{LOCATIONS[locationIndex]}</span>
						</button>
					</div>
				</div>
				<span class="rounded-full border border-[var(--neu-accent-soft-border)] bg-[var(--neu-accent-soft)] px-2.5 py-1 text-[10px] font-medium text-[var(--neu-accent-primary)]">
					正在发布
				</span>
			</div>

			<div class="neu-input-surface rounded-[20px] p-4">
				<textarea
					bind:value={text}
					oninput={(e) => (text = e.currentTarget.value.slice(0, 280))}
					placeholder="写下此刻发生的事，或分享一个想被附近的人看见的瞬间…"
					class="min-h-[122px] w-full resize-none bg-transparent text-[15px] leading-7 text-[var(--neu-text)] outline-none placeholder:text-[var(--neu-text-muted)]"
					aria-label="动态内容"
				></textarea>
				<div class="mt-3 flex items-center justify-between border-t border-[var(--neu-border)] pt-3">
					<div class="flex items-center gap-1">
						<button
							type="button"
							onclick={() => (hasImage = !hasImage)}
							aria-label={hasImage ? '已添加照片' : '添加照片'}
							aria-pressed={hasImage}
							class="flex h-8 w-8 items-center justify-center rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] {hasImage
								? 'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] hover:bg-[var(--neu-active-hover)]'
								: 'text-[var(--neu-text-muted)] hover:bg-[var(--neu-accent-soft)] hover:text-[var(--neu-accent-primary)]'}"
						>
							<ImagePlus size={16} strokeWidth={hasImage ? 2 : 1.6} />
						</button>
						<button
							type="button"
							onclick={() => appendText('✨')}
							aria-label="添加心情"
							class="flex h-8 w-8 items-center justify-center rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] text-[var(--neu-text-muted)] hover:bg-[var(--neu-accent-soft)] hover:text-[var(--neu-accent-primary)]"
						>
							<Smile size={16} strokeWidth={1.6} />
						</button>
						<button
							type="button"
							onclick={toggleTopic}
							aria-label="添加话题"
							aria-pressed={topicActive}
							class="flex h-8 w-8 items-center justify-center rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] {topicActive
								? 'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] hover:bg-[var(--neu-active-hover)]'
								: 'text-[var(--neu-text-muted)] hover:bg-[var(--neu-accent-soft)] hover:text-[var(--neu-accent-primary)]'}"
						>
							<Hash size={16} strokeWidth={topicActive ? 2 : 1.6} />
						</button>
					</div>
					<span class="text-[11px] tabular-nums {text.length >= 260 ? 'text-[var(--destructive)]' : 'text-[var(--neu-text-muted)]'}">
						{text.length}/280
					</span>
				</div>
			</div>

			{#if hasImage}
				<div class="relative overflow-hidden rounded-[18px] neu-ring-border bg-[var(--page-bg)]">
					<div class="overflow-hidden rounded-[18px]">
						<img
							src="https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=600"
							alt="待发布的动态配图：云龙湖边的傍晚"
							class="h-36 w-full object-cover"
						/>
						<span class="absolute bottom-2.5 left-2.5 rounded-full bg-black/55 px-2 py-1 text-[10px] text-white backdrop-blur-sm">1 张照片</span>
						<button
							type="button"
							onclick={() => (hasImage = false)}
							class="absolute right-2.5 top-2.5 flex h-7 w-7 items-center justify-center rounded-full bg-black/55 text-white outline-none backdrop-blur-sm transition-transform hover:scale-105 focus-visible:ring-2 focus-visible:ring-white"
							aria-label="移除照片"
						>
							<X size={14} strokeWidth={2} />
						</button>
					</div>
				</div>
			{/if}

			<section aria-labelledby="mood-label">
				<div class="mb-2 flex items-center justify-between">
					<p id="mood-label" class="neu-eyebrow">此刻氛围</p>
					{#if selectedMoods.size > 0}
						<span class="text-[10px] text-[var(--neu-text-muted)]">已选 {selectedMoods.size} 项</span>
					{/if}
				</div>
				<div class="flex flex-wrap gap-2">
					{#each MOOD_TAGS as tag}
						{@const active = selectedMoods.has(tag)}
						<button
							type="button"
							onclick={() => toggleMood(tag)}
							aria-pressed={active}
							class="rounded-full border px-3 py-1.5 text-[12px] outline-none transition-all focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] {active
								? 'border-[var(--neu-accent-soft-border)] bg-[var(--neu-accent-soft)] font-medium text-[var(--neu-accent-primary)] hover:border-[var(--neu-accent-primary)] hover:bg-[var(--neu-active-hover)]'
								: 'border-[var(--neu-border)] bg-transparent text-[var(--neu-text-muted)] hover:border-[var(--neu-border-pressed)] hover:text-[var(--neu-text)]'}"
						>
							{tag}
						</button>
					{/each}
				</div>
			</section>

			<section
				aria-labelledby="audience-label"
				class="rounded-[18px] ring-1 ring-[var(--neu-border)] bg-[color-mix(in_srgb,var(--neu-bg)_94%,var(--page-bg)_6%)] p-2"
			>
				<div class="flex items-center justify-between px-2 pb-2 pt-1">
					<p id="audience-label" class="neu-eyebrow">谁可以看到</p>
					<span class="text-[10px] text-[var(--neu-text-muted)]">{activeAudience.hint}</span>
				</div>
				<div class="grid grid-cols-3 gap-1">
					{#each AUDIENCES as { id, label, icon: AudienceIcon }}
						{@const active = audience === id}
						<button
							type="button"
							onclick={() => (audience = id)}
							aria-pressed={active}
							class="flex min-w-0 flex-col items-center gap-1 rounded-[12px] px-1 py-2 text-center outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] {active
								? 'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] hover:bg-[var(--neu-active-hover)]'
								: 'text-[var(--neu-text-muted)] hover:bg-[var(--neu-accent-soft)]/50'}"
						>
							<AudienceIcon size={14} strokeWidth={active ? 2 : 1.5} />
							<span class="truncate text-[10px] font-medium">{label}</span>
						</button>
					{/each}
				</div>
			</section>
		</div>
	</div>

	<footer class="flex flex-none items-center gap-3 border-t border-[var(--neu-border)] bg-[var(--neu-bg)] px-5 py-3">
		<div class="min-w-0 flex-1">
			<p class="truncate text-[11px] text-[var(--neu-text-muted)]">
				{selectedMoods.size ? `将以 ${Array.from(selectedMoods).join(' · ')} 的氛围出现` : '选择氛围，让附近的人更容易找到你'}
			</p>
		</div>
		<button
			type="submit"
			disabled={!canPost}
			class="neu-hover-btn shrink-0 rounded-full px-5 py-2.5 text-[13px] font-medium outline-none transition-all focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] disabled:cursor-not-allowed disabled:border disabled:border-[var(--neu-border)] disabled:bg-transparent disabled:text-[var(--neu-text-muted)] enabled:bg-[var(--neu-accent-primary)] enabled:text-white enabled:shadow-[0_6px_18px_var(--neu-glow)]"
		>
			发布动态
		</button>
	</footer>
</form>

