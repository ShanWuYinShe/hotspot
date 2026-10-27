<script lang="ts">
	import { Image as ImageIcon, UserRound, Users } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import type { Pin } from './location-data';
	import { PIN_COLORS, PIN_TONES } from './location-shared';
	import { avatarImageWidths, buildImageSrcSet, fadeImage, optimizedImageUrl } from '@/lib/images';

	// 图钉媒体展示（地图三处共用：桌面面板列表行 / 移动 Sheet 卡 / 桌面选中详情）。
	// R13 统一为「类型图标砖」：按 --cst-marker-user/group/moment 的 bg/glow 档位铺一块
	// 40px 类型色砖，内部承载头像 / 瞬间缩略图 / 类型图标——同一行高、同一外部轮廓，
	// 列表扫读时先按颜色认类型、再看内容；选中态（accent 实底行）换 surface-primary 底，
	// 避免类型色与 accent 同色隐形。

	let { pin, size = 42, selected = false }: {
		pin: Pin;
		size?: number;
		/** 所在列表行处于选中态（accent 背景）时传入：砖底换中性面，避免与类型色撞色。 */
		selected?: boolean;
	} = $props();

	const Icon = $derived(pin.type === 'group' ? Users : pin.type === 'moment' ? ImageIcon : UserRound);
	const inner = $derived(size - 10);
</script>

<div
	class="relative flex shrink-0 items-center justify-center rounded-sm"
	style="width:{size}px;height:{size}px;background:{selected ? 'var(--cst-surface-primary)' : PIN_TONES[pin.type].bg};box-shadow:{selected ? 'none' : `var(--cst-marker-halo) ${PIN_TONES[pin.type].glow}`};"
>
	{#if pin.type === 'user' && pin.avatar}
		<div class="relative">
			<Avatar src={pin.avatar} name={pin.name || pin.title || ''} size={inner} />
			{#if pin.online}
				<!-- 角标统一 OnlineDot（审查 M3：手写 10px inline style 复制品收编） -->
				<OnlineDot size="sm" class="bottom-0 right-0" />
			{/if}
		</div>
	{:else if pin.type === 'moment' && pin.image}
		<img
			src={optimizedImageUrl(pin.image, { width: inner * 2 })}
			srcset={buildImageSrcSet(pin.image, avatarImageWidths(inner))}
			sizes="{inner}px"
			alt={pin.title || pin.name || ''}
			loading="lazy"
			decoding="async"
			class="rounded-full object-cover"
			style="width:{inner}px;height:{inner}px;"
			use:fadeImage
		/>
	{:else}
		<Icon
			class="size-4.5"
			style="color:{selected ? 'var(--cst-accent-default)' : PIN_COLORS[pin.type]};"
			strokeWidth={1.75}
			aria-hidden="true"
		/>
	{/if}
</div>
