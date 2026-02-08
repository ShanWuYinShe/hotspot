<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { RippleButton } from '$lib/components/ui/interactive';
	import ThemeToggle from './theme-toggle.svelte';
	import SearchModal from './search-modal.svelte';
	import NotificationDropdown from './notification-dropdown.svelte';
	import Search from 'lucide-svelte/icons/search';
	import User from 'lucide-svelte/icons/user';
	import Settings from 'lucide-svelte/icons/settings';
	import LogOut from 'lucide-svelte/icons/log-out';
	import Menu from 'lucide-svelte/icons/menu';
	import Plus from 'lucide-svelte/icons/plus';
	import Image from 'lucide-svelte/icons/image';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import CreateMomentModal from './create-moment-modal.svelte';
	import CreateGroupModal from './create-group-modal.svelte';

	interface Props {
		onMenuClick?: () => void;
		showMenu?: boolean;
	}

	let { onMenuClick, showMenu = true }: Props = $props();
	let showSearchModal = $state(false);
	let showCreateMoment = $state(false);
	let showCreateGroup = $state(false);

	// 监听快捷键（Cmd/Ctrl + K 打开搜索）
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

	function openCreateMoment() {
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal();
			return;
		}
		showCreateMoment = true;
	}

	function openCreateGroup() {
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal();
			return;
		}
		showCreateGroup = true;
	}
</script>

<header
	class="sticky top-0 z-50 w-full border-b border-border/50 bg-background/80 shadow-md shadow-black/10 backdrop-blur supports-backdrop-filter:bg-background/60"
	style="padding-top: env(safe-area-inset-top, 0px);"
>
	<div class="mx-auto flex h-14 w-full items-center gap-1.5 px-3 sm:h-16 sm:gap-3 sm:px-4 md:px-6">
		{#if showMenu}
			<RippleButton 
				variant="icon"
				size="icon"
				class="shrink-0 md:hidden"
				onclick={onMenuClick}
				circular
			>
				<Menu class="h-5 w-5" />
			</RippleButton>
		{/if}

		<!-- Logo -->
		<a href="/" class="flex shrink-0 items-center gap-2 rounded-xl px-1.5 py-1 transition-all hover:bg-accent/60 sm:px-2">
			<div class="flex h-8 w-8 items-center justify-center rounded-xl bg-linear-to-br from-primary/20 to-primary/5 ring-1 ring-primary/20 sm:h-9 sm:w-9">
				<img src="/logo_32.png" alt="HotSpot" class="h-5 w-5 rounded-md sm:h-6 sm:w-6" />
			</div>
			<span class="hidden text-lg font-bold sm:inline-block bg-linear-to-r from-primary to-primary/60 bg-clip-text text-transparent">HotSpot</span>
		</a>

		<!-- Right side -->
		<div class="ml-auto flex shrink-0 items-center gap-1 sm:gap-2">
			<RippleButton
				variant="icon"
				size="icon"
				class="h-9 w-9 sm:h-10 sm:w-10"
				aria-label="打开搜索"
				onclick={() => (showSearchModal = true)}
				circular
			>
				<Search class="h-4 w-4 sm:h-5 sm:w-5" />
			</RippleButton>

			<!-- Theme toggle - 在小屏幕上隐藏 -->
			<div class="hidden sm:block">
				<ThemeToggle />
			</div>

			{#if authStore.isAuthenticated}
				<!-- Create Menu - 仅登录后显示 -->
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<RippleButton
								variant="icon"
								size="icon"
								class="h-9 w-9 sm:h-10 sm:w-10"
								{...props}
								circular
							>
								<Plus class="h-4 w-4 sm:h-5 sm:w-5" />
							</RippleButton>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="end">
						<DropdownMenu.Item onclick={openCreateMoment}>
							<Image class="mr-2 h-4 w-4" />
							发布动态
						</DropdownMenu.Item>
						<DropdownMenu.Item onclick={openCreateGroup}>
							<UsersRound class="mr-2 h-4 w-4" />
							创建群组
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>

				<!-- Notifications Dropdown -->
				<NotificationDropdown />

				<!-- User menu -->
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<button {...props} class="inline-flex items-center justify-center rounded-full p-0.5 hover:bg-accent transition-colors sm:p-1">
								<Avatar class="h-8 w-8 ring-2 ring-primary/20">
									<AvatarImage src={authStore.currentAccount?.avatar_url} />
									<AvatarFallback class="bg-primary/10 text-primary text-xs">
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
				<!-- 未登录：移动端显示图标，桌面端显示文字 -->
				<RippleButton
					variant="icon"
					size="icon"
					class="h-9 w-9 sm:hidden"
					aria-label="登录或注册"
					onclick={handleAuthEntry}
					circular
				>
					<User class="h-4 w-4" />
				</RippleButton>
				<Button size="sm" class="hidden rounded-full sm:inline-flex" onclick={handleAuthEntry}>
					登录 / 注册
				</Button>
			{/if}
		</div>
	</div>
</header>

<SearchModal bind:open={showSearchModal} />
<CreateMomentModal bind:open={showCreateMoment} />
<CreateGroupModal bind:open={showCreateGroup} />
