<script lang="ts">
	import { goto } from '$app/navigation';
	import { Heart, MessageCircle } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import type { GalleryPost } from '@/lib/profile-data.svelte';
	import type { Tab } from '@/components/profile/ProfileTabs.svelte';
	import { ROUTES } from '@/lib/routes';

	// 画廊网格（移动/平板，迁移自旧版 ProfileSections 内 GalleryGrid）。

	let { posts, activeTab }: { posts: GalleryPost[]; activeTab: Tab } = $props();

	// 密度契约：移动/紧凑平板 2 列，宽平板/桌面 3 列（对齐 Figma Make）
	const numCols = $derived(platformStore.platform === 'desktop' || platformStore.isWideTablet ? 3 : 2);
	const gap = 10;
</script>

<div class="animate-fade-in" style="display:grid;grid-template-columns:repeat({numCols}, 1fr);gap:{gap}px;width:100%;">
	{#each posts as post (post.id)}
		<div
			class="neu-hover-btn relative rounded-[18px] cursor-pointer group neu-focus-ring overflow-hidden"
			style="aspect-ratio:1/1;box-shadow:0 0 0 1px var(--neu-border), 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light);"
			role="button"
			tabindex="0"
			onclick={() => void goto(`${ROUTES.moment(post.id)}?from=me`)}
			onkeydown={(e) => {
				if (e.key === 'Enter' || e.key === ' ') {
					e.preventDefault();
					void goto(`${ROUTES.moment(post.id)}?from=me`);
				}
			}}
		>
			<div class="relative h-full w-full" style="overflow:hidden;border-radius:18px;">
				<img src={post.img} alt="" class="w-full h-full object-cover" />
				<div class="absolute inset-0 flex items-end p-2" style="background:linear-gradient(to top, rgba(0,0,0,0.55) 0%, transparent 55%);">
					<div class="flex items-center gap-2">
						<div class="flex items-center gap-0.5">
							<Heart style="width:10px;height:10px;color:white;" fill="white" strokeWidth={0} />
							<span class="text-white text-[10px]">{post.likes}</span>
						</div>
						<div class="flex items-center gap-0.5">
							<MessageCircle style="width:10px;height:10px;color:white;" strokeWidth={1.5} />
							<span class="text-white text-[10px]">{post.comments}</span>
						</div>
					</div>
				</div>
			</div>
		</div>
	{/each}
</div>
