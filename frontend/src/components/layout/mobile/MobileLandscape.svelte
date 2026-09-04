<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Home, MapPin, Plus, X, MessageCircle, CircleUser } from '@lucide/svelte';
	import { composer, openComposer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import CreateSheet from '@/components/overlays/CreateSheet.svelte';
	import RailIcon from '@/components/layout/mobile/RailIcon.svelte';
	import RoutedContentViewport from '@/components/layout/page/RoutedContentViewport.svelte';
	import { ROUTES, isPrimaryRouteActive, isSubPagePath } from '@/lib/routes';

	// 移动横屏：紧凑左栏（无状态栏），全高内容 + 右侧滑入发布面板。

	let { children }: { children: Snippet } = $props();

	const NAV_ITEMS: { path: string; icon: Component; label: string }[] = [
		{ path: ROUTES.home, icon: Home, label: '首页' },
		{ path: ROUTES.nearby, icon: MapPin, label: '附近' },
		{ path: ROUTES.messages, icon: MessageCircle, label: '消息' },
		{ path: ROUTES.me, icon: CircleUser, label: '我的' }
	];

	const pathname = $derived(page.url.pathname);
	const isSubPage = $derived(isSubPagePath(pathname));
	const showCreate = $derived(composer.composerOpen);

</script>

<div
	class="flex h-dvh w-full overflow-hidden font-sans transition-colors duration-gentle"
	style="background-color:var(--cst-surface-page);color:var(--cst-text-primary);"
>
	{#if !isSubPage}
		<aside
			class="shrink-0 flex flex-col items-center"
			style="width:var(--cst-mobile-rail-width);padding-top:max(env(safe-area-inset-top, 8px), 8px);padding-bottom:max(env(safe-area-inset-bottom, 8px), 8px);padding-left:max(env(safe-area-inset-left, 0px), 0px);gap:var(--cst-bottom-nav-pad-top);border-right:var(--cst-border-width-hairline) solid var(--cst-border-default);background-color:var(--cst-surface-primary);"
		>
			{#each NAV_ITEMS.slice(0, 2) as item}
				<RailIcon
					Icon={item.icon}
					label={item.label}
					isActive={isPrimaryRouteActive(pathname, item.path)}
					onClick={() => goto(item.path)}
				/>
			{/each}

			<button
				type="button"
				class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring my-1 size-11"
				style="background:{composer.composerOpen ? 'var(--cst-text-secondary)' : 'var(--cst-accent-default)'};box-shadow:{composer.composerOpen ? 'none' : 'var(--cst-elevation-fab)'};transition:background var(--cst-motion-standard) var(--cst-ease-standard), transform var(--cst-motion-snappy) var(--cst-ease-standard), filter var(--cst-motion-snappy) var(--cst-ease-standard);"
				aria-label={composer.composerOpen ? '关闭' : '发布'}
				onclick={toggleComposer}
			>
				{#if composer.composerOpen}
					<X class="size-4.5 text-text-on-accent"   strokeWidth={2} />
				{:else}
					<Plus class="size-4.5 text-text-on-accent"   strokeWidth={2} />
				{/if}
			</button>

			{#each NAV_ITEMS.slice(2, 4) as item}
				<RailIcon
					Icon={item.icon}
					label={item.label}
					isActive={isPrimaryRouteActive(pathname, item.path)}
					onClick={() => goto(item.path)}
				/>
			{/each}
		</aside>
	{/if}

	<RoutedContentViewport class="shrink-0" contentClassName="bg-surface-primary">
		{@render children()}
	</RoutedContentViewport>

	<Dialog.Root open={showCreate} onOpenChange={(next) => (next ? openComposer() : closeComposer())}>
		<Dialog.Content
			side="right"
			class="overflow-hidden rounded-l-sheet border-l-0"
			style="width:min(560px,90%);box-shadow:var(--cst-elevation-side-sheet);"
		>
			<CreateSheet onClose={closeComposer} />
		</Dialog.Content>
	</Dialog.Root>
</div>
