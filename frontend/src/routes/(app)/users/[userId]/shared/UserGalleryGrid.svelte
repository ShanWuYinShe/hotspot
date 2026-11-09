<script lang="ts">
	import { goto } from '$app/navigation';
	import { ImageIcon } from '@lucide/svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import { fadeImage, presetImageUrl, presetSrcSet } from '@/lib/images';
	import { ROUTES } from '@/lib/routes';
	import type { GalleryMomentItem } from './useUserProfileData.svelte';
	import type { BspLayoutCell } from '@/lib/semantic-bsp';

	let {
		userName,
		galleryMoments,
		galleryBspLayouts
	}: {
		userName: string;
		galleryMoments: GalleryMomentItem[];
		galleryBspLayouts: BspLayoutCell[];
	} = $props();
</script>

<section class="mt-6" aria-label="最近瞬间">
	<SectionHead title="最近瞬间" count={galleryMoments.length > 0 ? galleryMoments.length : undefined} icon={ImageIcon} />
	{#if galleryMoments.length > 0}
		<div
			class="grid grid-flow-dense grid-cols-2 gap-2 sm:grid-cols-3 sm:gap-3"
			style="grid-auto-rows: clamp(8.75rem, 16vw, 13.75rem);"
		>
			{#each galleryMoments as g, i (g.id)}
				{@const layout = galleryBspLayouts[i] ?? { tier: '1x1', cls: 'col-span-1 row-span-1' }}
				<button
					type="button"
					class="{layout.cls} media-placeholder image-hover-frame tap-97 relative h-full w-full cursor-pointer overflow-hidden rounded-md focus-ring"
					aria-label={`查看 ${userName} 的瞬间图片 ${i + 1}`}
					onclick={() => void goto(ROUTES.moment(g.id))}
				>
					<img
						src={presetImageUrl(g.url, 'card', 800)}
						srcset={presetSrcSet(g.url, 'card')}
						sizes="(min-width: 600px) 33vw, 50vw"
						alt={`${userName} 的瞬间图片 ${i + 1}`}
						loading={i < 2 ? 'eager' : 'lazy'}
						fetchpriority={i < 2 ? 'high' : undefined}
						decoding="async"
						class="image-hover-frame__image h-full w-full object-cover"
						use:fadeImage
					/>
					{#if g.time}
						<span
							class="pointer-events-none media-badge absolute bottom-2 left-2 z-[2]"
						>
							{g.time}
						</span>
					{/if}
				</button>
			{/each}
		</div>
	{:else}
		<EmptyState
			iconTone="accent"
			title="还没有带图瞬间"
			description="带图瞬间会展示在这里；文字瞬间暂不计入画廊。"
		>
			{#snippet icon()}
				<ImageIcon class="size-7" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{/if}
</section>
