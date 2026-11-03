<script lang="ts">
	import { Image as ImageIcon, UserRound, Users } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import type { Pin } from './location-data';
	import { avatarImageWidths, buildImageSrcSet, fadeImage, optimizedImageUrl } from '@/lib/images';

	// 图钉媒体展示（地图三处共用：桌面面板列表行 / 移动 Sheet 卡 / 桌面选中详情）。
	// 与全站列表媒体同一语言：用户/群组直接用 shared/Avatar（circle / rounded，含首字母
	// 与图片失败兜底），瞬间用圆角缩略图，无图/聚合桶给中性无底色图标。
	// 不再使用「类型色砖 + marker 光晕」旧视觉：色块与阴影已从全站列表媒体移除，
	// 类型语义由行内类型文字（PIN_COLORS）承载。

	let { pin, size = 44, selected = false }: {
		pin: Pin;
		/** 媒体边长（px）：列表行 44 / 详情卡 48，对齐全站列表头像刻度（md/lg 档）。 */
		size?: number;
		/** 所在列表行处于选中态（accent 背景）：无图/聚合图标换 on-accent 档。 */
		selected?: boolean;
	} = $props();

	const name = $derived(pin.name || pin.title || '');
	const iconColor = $derived(selected ? 'var(--cst-text-on-accent)' : 'var(--cst-text-secondary)');
</script>

{#if pin.isAggregate}
	<!-- 聚合桶没有真实实体（无头像/封面）：中性无底色类型图标，计数由行尾 Badge 承载 -->
	<span class="flex shrink-0 items-center justify-center" style="width:{size / 16}rem;height:{size / 16}rem" aria-hidden="true">
		{#if pin.type === 'group'}
			<Users class="size-5" style="color:{iconColor}" strokeWidth={1.75} />
		{:else if pin.type === 'moment'}
			<ImageIcon class="size-5" style="color:{iconColor}" strokeWidth={1.75} />
		{:else}
			<UserRound class="size-5" style="color:{iconColor}" strokeWidth={1.75} />
		{/if}
	</span>
{:else if pin.type === 'moment'}
	{#if pin.image}
		<span
			class="media-placeholder flex shrink-0 items-center justify-center overflow-hidden rounded-sm"
			style="width:{size / 16}rem;height:{size / 16}rem"
		>
			<img
				src={optimizedImageUrl(pin.image, { width: size * 2 })}
				srcset={buildImageSrcSet(pin.image, avatarImageWidths(size))}
				sizes="{size / 16}rem"
				alt={name}
				loading="lazy"
				decoding="async"
				class="object-cover"
				style="width:{size / 16}rem;height:{size / 16}rem;"
				use:fadeImage
			/>
		</span>
	{:else}
		<span class="flex shrink-0 items-center justify-center" style="width:{size / 16}rem;height:{size / 16}rem" aria-hidden="true">
			<ImageIcon class="size-5" style="color:{iconColor}" strokeWidth={1.75} />
		</span>
	{/if}
{:else}
	<!-- 用户/群组：与全站列表一致直接用 Avatar（circle / rounded），在线角标走 badge snippet -->
	<Avatar src={pin.avatar} {name} {size} shape={pin.type === 'group' ? 'rounded' : 'circle'}>
		{#snippet badge()}
			{#if pin.type === 'user' && pin.online}
				<OnlineDot size="sm" class="bottom-0 right-0" />
			{/if}
		{/snippet}
	</Avatar>
{/if}
