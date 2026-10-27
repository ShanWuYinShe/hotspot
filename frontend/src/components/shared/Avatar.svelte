<script lang="ts" module>
	/**
	 * 头像命名刻度（审查 M2：全站 20 种尺寸、1-2px 近邻档并存 → 收敛 5 档，
	 * 对齐 theme-registry「近邻档位一律归并」纪律）。
	 * xs=20 行内徽标 / sm=24 预留 / md=36 列表与卡片 / lg=44 页头与触控行 / xl=80 主页 hero。
	 */
	export const AVATAR_SIZES = {
		xs: 20,
		sm: 24,
		md: 36,
		lg: 44,
		xl: 80
	} as const;

	export type AvatarSize = keyof typeof AVATAR_SIZES;
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { UserRound } from '@lucide/svelte';
	import { avatarImageWidths, buildImageSrcSet, fadeImage, optimizedImageUrl } from '@/lib/images';

	// Avatar — 用户/群组头像统一展示组件。
	// 包装元素持有 border-radius + outline + overflow:hidden，徽标作为子元素
	// 与描边同一元素，避免「分离边框的 broken-border」问题。

	let {
		src,
		name,
		size = 'lg',
		shape = 'circle',
		ring = false,
		decorative = false,
		badge,
		class: className = '',
		style = ''
	}: {
		/** Image URL. When empty/undefined/null (spec 恒定 nullable) or the image fails to load, the name initial is shown instead. */
		src?: string | null;
		/** Display name — supplies the alt text and the fallback initial. */
		name?: string;
		/** Named scale (preferred) or explicit px for one-off exceptions. */
		size?: AvatarSize | number;
		/** Circle (people) or rounded rectangle (groups). */
		shape?: 'circle' | 'rounded';
		/** Draw a thin themed ring around the avatar. */
		ring?: boolean;
		/** 装饰性头像（相邻已有可见姓名文本）：图片 alt 置空并 aria-hidden，避免屏幕阅读器重复念名。 */
		decorative?: boolean;
		/** Optional badge (e.g. online dot) rendered absolutely positioned inside the wrapper. */
		badge?: Snippet;
		class?: string;
		style?: string;
	} = $props();

	let hasError = $state(false);

	// 头像 src 变化（如上传新头像）后重置错误态，否则永远停留首字母兜底
	$effect(() => {
		void src;
		hasError = false;
	});

	const px = $derived(typeof size === 'number' ? size : AVATAR_SIZES[size]);
	const showImage = $derived(Boolean(src) && !hasError);
	// 无名字时不再显示「?」占位——由人形图标兜底(语义更明,访客评论头像等场景;Gemini 审查 r19)
	const initial = $derived((name ?? '').trim().slice(0, 1));
	const borderRadius = $derived(shape === 'circle' ? '9999px' : `${Math.round(px * 0.28)}px`);
</script>

<div
	aria-hidden={decorative || undefined}
	class="relative shrink-0 overflow-hidden bg-surface-subtle {className}"
	style="width:{px}px;height:{px}px;border-radius:{borderRadius};{ring ? 'border:var(--cst-border-width-hairline) solid var(--cst-accent-border-soft);' : ''}{style}"
>
	{#if showImage}
		<img
			src={optimizedImageUrl(src, { width: px * 2 })}
			srcset={buildImageSrcSet(src, avatarImageWidths(px))}
			sizes={`${px}px`}
			loading="lazy"
			decoding="async"
			alt={decorative ? '' : (name ?? '')}
			class="h-full w-full object-cover"
			use:fadeImage
			onerror={() => (hasError = true)}
		/>
	{:else if initial}
		<span
			class="flex h-full w-full items-center justify-center text-text-secondary bg-accent-bg"
			style="font-size:{Math.round(px * 0.42)}px;"
		>
			{initial}
		</span>
	{:else}
		<span class="flex h-full w-full items-center justify-center text-text-secondary bg-accent-bg">
			<UserRound class="size-1/2" strokeWidth={1.5} aria-hidden="true" />
		</span>
	{/if}
	{#if badge}
		{@render badge()}
	{/if}
</div>
