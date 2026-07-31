<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { MapPin, Compass, Plus, X, MessageCircle, CircleUser } from '@lucide/svelte';
	import { composer, closeComposer, toggleComposer } from '@/stores/composer.svelte';
	import { attachDismissibleLayer } from '@/lib/dismissible-layer';
	import CreateSheet from '@/components/layout/CreateSheet.svelte';
	import RailIcon from '@/components/layout/RailIcon.svelte';
	import { ROUTES, isPrimaryRouteActive, isSubPagePath } from '@/lib/routes';

	// 移动横屏：紧凑左栏（无状态栏），全高内容 + 右侧滑入发布面板。
	// 迁移自 React MainLayout.tsx 内 MobileLandscape。

	let { children }: { children: Snippet } = $props();

	const NAV_ITEMS: { path: string; icon: Component; label: string }[] = [
		{ path: ROUTES.home, icon: MapPin, label: '附近' },
		{ path: ROUTES.nearbyMoments, icon: Compass, label: '发现' },
		{ path: ROUTES.messages, icon: MessageCircle, label: '消息' },
		{ path: ROUTES.me, icon: CircleUser, label: '我的' }
	];

	const pathname = $derived(page.url.pathname);
	const isSubPage = $derived(isSubPagePath(pathname));
	const showCreate = $derived(composer.composerOpen);

	$effect(() => {
		return attachDismissibleLayer({
			open: composer.composerOpen,
			onDismiss: closeComposer
		});
	});
</script>

<div
	class="flex h-[100dvh] w-full overflow-hidden font-sans transition-colors duration-500"
	style="background-color:var(--page-bg);color:var(--neu-text);"
>
	{#if !isSubPage}
		<aside
			class="flex-shrink-0 flex flex-col items-center"
			style="width:64px;padding-top:max(env(safe-area-inset-top, 8px), 8px);padding-bottom:max(env(safe-area-inset-bottom, 8px), 8px);padding-left:max(env(safe-area-inset-left, 0px), 0px);gap:8px;border-right:1px solid var(--neu-border);background-color:var(--neu-bg);"
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
				class="neu-hover-btn flex items-center justify-center rounded-full outline-none my-1"
				style="width:44px;height:44px;background:{composer.composerOpen ? 'var(--neu-text-muted)' : 'var(--neu-accent-primary)'};box-shadow:{composer.composerOpen ? 'none' : '0 4px 14px var(--neu-glow)'};transition:background 0.22s;"
				aria-label={composer.composerOpen ? '关闭' : '发布'}
				onclick={toggleComposer}
			>
				{#if composer.composerOpen}
					<X style="width:18px;height:18px;color:white;" strokeWidth={2.2} />
				{:else}
					<Plus style="width:18px;height:18px;color:white;" strokeWidth={2.2} />
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

	<main class="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar" style="background-color:var(--neu-bg);">
		{@render children()}
	</main>

	{#if showCreate}
		<div
			class="fixed inset-0 z-40"
			style="background:rgba(0,0,0,0.4);backdrop-filter:blur(4px);"
			onclick={closeComposer}
		/>
		<div
			class="fixed top-0 bottom-0 right-0 z-50 overflow-hidden"
			role="dialog"
			aria-modal="true"
			aria-labelledby="create-composer-title"
			style="width:min(560px, 90%);border-radius:28px 0 0 28px;background-color:var(--neu-bg);box-shadow:-8px 0 40px rgba(0,0,0,0.22);"
		>
			<CreateSheet onClose={closeComposer} />
		</div>
	{/if}
</div>
