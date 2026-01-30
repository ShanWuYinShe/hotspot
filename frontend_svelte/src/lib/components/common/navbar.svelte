<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import ThemeToggle from './theme-toggle.svelte';
	import SearchModal from './search-modal.svelte';
	import Bell from 'lucide-svelte/icons/bell';
	import Search from 'lucide-svelte/icons/search';
	import User from 'lucide-svelte/icons/user';
	import Settings from 'lucide-svelte/icons/settings';
	import LogOut from 'lucide-svelte/icons/log-out';
	import Menu from 'lucide-svelte/icons/menu';

	interface Props {
		onMenuClick?: () => void;
		showMenu?: boolean;
	}

	let { onMenuClick, showMenu = true }: Props = $props();
	let showSearchModal = $state(false);

	onMount(() => {
		const handleKeydown = (event: KeyboardEvent) => {
			if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
				event.preventDefault();
				showSearchModal = true;
			}
		};

		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	function handleLogout() {
		authStore.logout();
		goto('/');
	}

	function handleAuthEntry() {
		uiStore.openLoginModal();
	}
</script>

<header
	class="sticky top-0 z-50 w-full border-b border-border/50 bg-background/80 shadow-md shadow-black/10 backdrop-blur supports-backdrop-filter:bg-background/60"
>
	<div class="mx-auto flex h-16 w-full items-center gap-3 px-4 md:px-6">
		{#if showMenu}
			<Button variant="ghost" size="icon" class="mr-2 md:hidden" onclick={onMenuClick}>
				<Menu class="h-5 w-5" />
			</Button>
		{/if}

		<!-- Logo -->
		<a href="/" class="flex items-center gap-2 rounded-xl px-2 py-1 transition-all hover:bg-accent/60">
			<div class="flex h-9 w-9 items-center justify-center rounded-xl bg-linear-to-br from-primary/20 to-primary/5 ring-1 ring-primary/20">
				<img src="/logo_32.png" alt="HotSpot" class="h-6 w-6 rounded-md" />
			</div>
			<span class="hidden text-lg font-bold sm:inline-block bg-linear-to-r from-primary to-primary/60 bg-clip-text text-transparent">HotSpot</span>
		</a>

		<!-- Right side -->
		<div class="ml-auto flex items-center gap-2">
			<Button
				variant="ghost"
				size="icon"
				class="rounded-full"
				aria-label="打开搜索"
				onclick={() => (showSearchModal = true)}
			>
				<Search class="h-5 w-5" />
			</Button>

			<!-- Theme toggle -->
			<ThemeToggle />

			{#if authStore.isAuthenticated}
				<!-- Notifications -->
				<Button variant="ghost" size="icon" onclick={() => goto('/notifications')} class="relative">
					<Bell class="h-5 w-5" />
				</Button>

				<!-- User menu -->
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<button {...props} class="inline-flex items-center justify-center rounded-full p-1 hover:bg-accent transition-colors">
								<Avatar class="h-8 w-8 ring-2 ring-primary/20">
									<AvatarImage src={authStore.currentAccount?.avatar_url} />
									<AvatarFallback class="bg-primary/10 text-primary">
										{authStore.currentAccount?.nickname?.slice(0, 2) ?? 'U'}
									</AvatarFallback>
								</Avatar>
							</button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content class="w-56" align="end">
						<DropdownMenu.Label>
							<div class="flex flex-col space-y-1">
								<p class="text-sm font-medium">{authStore.currentAccount?.nickname}</p>
								<p class="text-xs text-muted-foreground">ID: {authStore.currentUserId}</p>
							</div>
						</DropdownMenu.Label>
						<DropdownMenu.Separator />
						<DropdownMenu.Item>
							{#snippet child({ props })}
								<button {...props} class="flex w-full items-center" onclick={() => goto('/profile')}>
									<User class="mr-2 h-4 w-4" />
									个人资料
								</button>
							{/snippet}
						</DropdownMenu.Item>
						<DropdownMenu.Item>
							{#snippet child({ props })}
								<button {...props} class="flex w-full items-center" onclick={() => goto('/settings')}>
									<Settings class="mr-2 h-4 w-4" />
									设置
								</button>
							{/snippet}
						</DropdownMenu.Item>
						<DropdownMenu.Separator />
						<DropdownMenu.Item>
							{#snippet child({ props })}
								<button {...props} class="flex w-full items-center text-destructive" onclick={handleLogout}>
									<LogOut class="mr-2 h-4 w-4" />
									退出登录
								</button>
							{/snippet}
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			{:else}
				<Button size="sm" class="rounded-full" onclick={handleAuthEntry}>登录 / 注册</Button>
			{/if}
		</div>
	</div>
</header>

<SearchModal bind:open={showSearchModal} />
