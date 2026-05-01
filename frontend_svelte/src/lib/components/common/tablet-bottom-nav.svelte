<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { t } from '$lib/i18n';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import { messageStore } from '$lib/stores/message.svelte';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Compass from 'lucide-svelte/icons/compass';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import User from 'lucide-svelte/icons/user';
	import Plus from 'lucide-svelte/icons/plus';

	const navItems = [
		{ href: '/', label: t('nav.nearby'), icon: MapPin },
		{ href: '/moments/nearby', label: t('nav.discover'), icon: Compass },
		{ href: '/chat', label: t('nav.messagesTab'), icon: MessageCircle, requiresAuth: true },
		{ href: '/profile', label: t('nav.me'), icon: User, requiresAuth: true },
	];

	const currentPath = $derived(page.url.pathname);

	function isActive(href: string) {
		if (href === '/') return currentPath === '/';
		if (href === '/profile') return currentPath === '/profile' || currentPath === '/settings';
		return currentPath === href || currentPath.startsWith(`${href}/`);
	}

	function navigate(item: { href: string; requiresAuth?: boolean }) {
		if (item.requiresAuth && !authStore.isAuthenticated) {
			uiStore.openLoginModalWithRedirect({ url: item.href });
			return;
		}
		void goto(item.href);
	}

	const messageBadge = $derived(authStore.isAuthenticated ? messageStore.totalUnreadCount : 0);
	const profileBadge = $derived(authStore.isAuthenticated ? notificationStore.unreadCount : 0);


</script>

<div
	class="flex w-full shrink-0 justify-center"
	style="padding: 8px 24px; padding-bottom: max(env(safe-area-inset-bottom, 16px), 16px); background: transparent;"
>
	<div
		class="flex items-center"
		style="width: 100%; max-width: 480px; border-radius: 36px; background: var(--neu-glass-bg); backdrop-filter: blur(28px) saturate(1.8); -webkit-backdrop-filter: blur(28px) saturate(1.8); box-shadow: var(--neu-glass-shadow); border: 1px solid var(--neu-glass-border); padding: 6px 12px; gap: 4px;"
	>
		{#each navItems.slice(0, 2) as item (item.href)}
			<button
				class="relative flex flex-col items-center justify-center outline-none select-none"
				style="flex: 1; min-width: 0; padding: 7px 0; gap: 3px;"
				onclick={() => navigate(item)}
				aria-label={item.label}
				aria-current={isActive(item.href) ? 'page' : undefined}
			>
				<item.icon
					size={22}
					strokeWidth={isActive(item.href) ? 2.0 : 1.5}
					color={isActive(item.href) ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'}
					style="transition: color 0.22s ease;"
				/>
				<span
					style="font-size: 10px; line-height: 1; color: {isActive(item.href) ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'}; font-weight: {isActive(item.href) ? 600 : 400}; transition: color 0.22s ease; letter-spacing: 0.01em;"
				>
					{item.label}
				</span>
				{#if isActive(item.href)}
					<span
						style="position: absolute; bottom: 0; width: 24px; height: 2.5px; border-radius: 2px; background: var(--neu-accent-primary); transform-origin: center;"
					></span>
				{/if}
			</button>
		{/each}

		<!-- Center FAB -->
		<div class="flex shrink-0 items-center justify-center" style="width: 64px;">
			<button
				class="flex items-center justify-center rounded-full outline-none select-none"
				style="width: 48px; height: 48px; background: var(--neu-accent-primary); box-shadow: 0 6px 20px var(--neu-glow); transition: background 0.22s, box-shadow 0.22s;"
				onclick={() => uiStore.openCreateSheet()}
				aria-label={t('nav.publish')}
			>
				<Plus size={19} strokeWidth={2.2} color="var(--neu-active-text)" />
			</button>
		</div>

		{#each navItems.slice(2) as item (item.href)}
			<button
				class="relative flex flex-col items-center justify-center outline-none select-none"
				style="flex: 1; min-width: 0; padding: 7px 0; gap: 3px;"
				onclick={() => navigate(item)}
				aria-label={item.label}
				aria-current={isActive(item.href) ? 'page' : undefined}
			>
				<span style="position: relative;">
					<item.icon
						size={22}
						strokeWidth={isActive(item.href) ? 2.0 : 1.5}
						color={isActive(item.href) ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'}
						style="transition: color 0.22s ease;"
					/>
					{#if item.href === '/chat' && messageBadge > 0}
						<span
							class="absolute flex items-center justify-center rounded-full"
							style="top: -6px; right: -8px; height: 18px; min-width: 18px; padding: 0 5px; background: var(--neu-error); color: var(--neu-active-text); font-size: 10px; font-weight: 700; line-height: 18px;"
						>
							{messageBadge > 99 ? '99+' : messageBadge}
						</span>
					{:else if item.href === '/profile' && profileBadge > 0}
						<span
							class="absolute flex items-center justify-center rounded-full"
							style="top: -6px; right: -8px; height: 18px; min-width: 18px; padding: 0 5px; background: var(--neu-error); color: var(--neu-active-text); font-size: 10px; font-weight: 700; line-height: 18px;"
						>
							{profileBadge > 99 ? '99+' : profileBadge}
						</span>
					{/if}
				</span>
				<span
					style="font-size: 10px; line-height: 1; color: {isActive(item.href) ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'}; font-weight: {isActive(item.href) ? 600 : 400}; transition: color 0.22s ease; letter-spacing: 0.01em;"
				>
					{item.label}
				</span>
				{#if isActive(item.href)}
					<span
						style="position: absolute; bottom: 0; width: 24px; height: 2.5px; border-radius: 2px; background: var(--neu-accent-primary); transform-origin: center;"
					></span>
				{/if}
			</button>
		{/each}
	</div>
</div>
