<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { t } from '$lib/i18n';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import { messageStore } from '$lib/stores/message.svelte';
	import { RippleButton } from '$lib/components/ui/interactive';
	import Home from 'lucide-svelte/icons/house';
	import Compass from 'lucide-svelte/icons/compass';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import User from 'lucide-svelte/icons/user';
	import Plus from 'lucide-svelte/icons/plus';

	interface Props {}

	let {}: Props = $props();

	const navItems = $derived([
		{ href: '/', label: t('nav.nearby'), icon: Home },
		{ href: '/moments/nearby', label: t('nav.discover'), icon: Compass },
		{ href: '/chat', label: t('nav.messagesTab'), icon: MessageCircle, requiresAuth: true },
		{ href: '/profile', label: t('nav.me'), icon: User, requiresAuth: true },
	]);

	const currentPath = $derived(page.url.pathname);

	function isActive(href: string) {
		if (href === '/') {
			return currentPath === '/';
		}
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

<nav class="mobile-bottom-nav md:hidden" aria-label="Mobile navigation">
	<div class="mx-auto flex max-w-md items-end justify-between gap-1 px-3 pt-2">
		{#each navItems.slice(0, 2) as item (item.href)}
			<RippleButton
				variant="ghost"
				size="icon"
				class="mobile-bottom-nav__item"
				onclick={() => navigate(item)}
			>
				<div class="mobile-bottom-nav__inner">
					<item.icon class="h-5 w-5 {isActive(item.href) ? 'text-primary' : 'text-muted-foreground'}" />
					<span class="mobile-bottom-nav__label {isActive(item.href) ? 'text-primary' : 'text-muted-foreground'}">{item.label}</span>
				</div>
			</RippleButton>
		{/each}

		<RippleButton
			variant="icon"
			size="icon"
			class="mobile-bottom-nav__create"
			onclick={() => uiStore.openCreateSheet()}
			aria-label={t('nav.publish')}
		>
			<Plus class="h-5 w-5" />
		</RippleButton>

		{#each navItems.slice(2) as item (item.href)}
			<RippleButton
				variant="ghost"
				size="icon"
				class="mobile-bottom-nav__item"
				onclick={() => navigate(item)}
			>
				<div class="mobile-bottom-nav__inner">
					<div class="relative">
						<item.icon class="h-5 w-5 {isActive(item.href) ? 'text-primary' : 'text-muted-foreground'}" />
						{#if item.href === '/chat' && messageBadge > 0}
							<span class="mobile-bottom-nav__badge">{messageBadge > 99 ? '99+' : messageBadge}</span>
						{:else if item.href === '/profile' && profileBadge > 0}
							<span class="mobile-bottom-nav__badge">{profileBadge > 99 ? '99+' : profileBadge}</span>
						{/if}
					</div>
					<span class="mobile-bottom-nav__label {isActive(item.href) ? 'text-primary' : 'text-muted-foreground'}">{item.label}</span>
				</div>
			</RippleButton>
		{/each}
	</div>
</nav>
