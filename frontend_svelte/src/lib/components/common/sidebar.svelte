<script lang="ts">
    import { page } from "$app/state";
    import { goto } from "$app/navigation";
    import { cn } from "$lib/utils";
    import { authStore } from "$lib/stores/auth.svelte";
    import { adminRbacService } from "$lib/api/services";
    import { isSuccess } from "$lib/api/client";
    import { mapAnchorStore } from "$lib/stores/map-anchor.svelte";
    import { uiStore } from "$lib/stores/ui.svelte";
    import { notificationStore } from "$lib/stores/notification.svelte";
    import { messageStore } from "$lib/stores/message.svelte";
    import { toggleMode, mode } from "mode-watcher";
	import { Badge } from "$lib/components/ui/badge";
	import { Button } from "$lib/components/ui/button";
    import { RippleNavItem } from "$lib/components/ui/interactive";
    import { rippleFeedback, circularFeedback } from "$lib/attachments";
    import LocationPickerModal from "./location-picker-modal.svelte";
    import Home from "lucide-svelte/icons/home";
    import Image from "lucide-svelte/icons/image";
    import UsersRound from "lucide-svelte/icons/users-round";
    import MessageCircle from "lucide-svelte/icons/message-circle";
    import Bell from "lucide-svelte/icons/bell";
    import UserPlus from "lucide-svelte/icons/user-plus";
    import Heart from "lucide-svelte/icons/heart";
    import UserCheck from "lucide-svelte/icons/user-check";
    import Navigation from "lucide-svelte/icons/navigation";
    import Settings from "lucide-svelte/icons/settings";
    import Shield from "lucide-svelte/icons/shield";
    import Sun from "lucide-svelte/icons/sun";
    import Moon from "lucide-svelte/icons/moon";
    import ChevronRight from "lucide-svelte/icons/chevron-right";

    interface NavItem {
        href: string;
        label: string;
        icon: typeof Home;
        requiresAuth?: boolean;
        badge?: number;
    }

    const mainNavItems: NavItem[] = [
        { href: "/", label: "首页", icon: Home },
        { href: "/users/nearby", label: "附近的人", icon: UsersRound },
        { href: "/groups/nearby", label: "附近群组", icon: UsersRound },
        { href: "/moments/nearby", label: "附近动态", icon: Image },
    ];

    const personalNavItems: NavItem[] = [
        {
            href: "/chat",
            label: "消息",
            icon: MessageCircle,
            requiresAuth: true,
        },
        {
            href: "/notifications",
            label: "通知",
            icon: Bell,
            requiresAuth: true,
        },
        {
            href: "/moments",
            label: "我的瞬间",
            icon: Image,
            requiresAuth: true,
        },
        {
            href: "/groups",
            label: "我的群组",
            icon: UsersRound,
            requiresAuth: true,
        },
        { href: "/friends", label: "好友", icon: Heart, requiresAuth: true },
        {
            href: "/follows",
            label: "关注",
            icon: UserCheck,
            requiresAuth: true,
        },
        {
            href: "/followers",
            label: "粉丝",
            icon: UserPlus,
            requiresAuth: true,
        },
    ];

    // 优化：使用 $derived 缓存当前路径和参数，避免在模板中重复计算
    const currentPathname = $derived(page.url.pathname);
    const currentParams = $derived(page.url.searchParams);

    // 预计算所有导航项的激活状态
    const activeStates = $derived.by(() => {
        const states = new Map<string, boolean>();

        const normalizePath = (path: string) => {
            if (path === "/") return "/";
            return path.replace(/\/+$/, "") || "/";
        };

        const checkActive = (href: string): boolean => {
            const url = new URL(href, page.url.origin);
            const targetPath = normalizePath(url.pathname);
            const currentPath = normalizePath(currentPathname);
            const hasQuery = url.searchParams.toString().length > 0;

            if (hasQuery) {
                if (currentPath !== targetPath) return false;
                for (const [key, value] of url.searchParams) {
                    if (currentParams.get(key) !== value) return false;
                }
                return true;
            }

            if (currentPath === targetPath) {
                return true;
            }

            if (!currentPath.startsWith(targetPath + "/")) {
                return false;
            }

            const nextSegment =
                currentPath.slice(targetPath.length + 1).split("/")[0] || "";
            if (nextSegment === "nearby" && !targetPath.endsWith("/nearby")) {
                return false;
            }

            return true;
        };

        // 预计算所有导航项
        [...mainNavItems, ...personalNavItems].forEach((item) => {
            states.set(item.href, checkActive(item.href));
        });

        return states;
    });

    // 辅助函数：从 Map 中获取激活状态
    function isActive(href: string): boolean {
        return activeStates.get(href) ?? false;
    }

    function handleNavClick(item: NavItem, e: MouseEvent) {
        if (item.requiresAuth && !authStore.isAuthenticated) {
            e.preventDefault();
            uiStore.saveRedirectUrl(item.href);
            uiStore.openLoginModal();
        }
    }

    interface Props {
        class?: string;
    }

    let { class: className }: Props = $props();

    // 位置选择弹窗状态
    let showLocationPicker = $state(false);

    const currentMode = $derived(mode.current);
    let isAdminUser = $state(false);
    let adminCheckedUserId = "";
    let checkingAdmin = false;

    $effect(() => {
        const isAuthenticated = authStore.isAuthenticated;
        const currentUserId = authStore.currentUserId ?? "";
        if (!isAuthenticated || !currentUserId) {
            isAdminUser = false;
            adminCheckedUserId = "";
            return;
        }
        if (checkingAdmin || adminCheckedUserId === currentUserId) return;
        checkingAdmin = true;
        void (async () => {
            const requestUserId = currentUserId;
            try {
                const res = await adminRbacService.getMyPermissions();
                if (authStore.currentUserId !== requestUserId) return;
                isAdminUser = isSuccess(res) && !!res.data;
            } catch {
                if (authStore.currentUserId !== requestUserId) return;
                isAdminUser = false;
            } finally {
                if (authStore.currentUserId === requestUserId) {
                    adminCheckedUserId = requestUserId;
                }
                checkingAdmin = false;
            }
        })();
    });
</script>

<aside
    class={cn(
        "relative flex h-full flex-col border-r border-sidebar-border/60 bg-sidebar/95 text-sidebar-foreground shadow-sm shadow-black/5 dark:shadow-black/20 backdrop-blur",
        className,
    )}
>
	<!-- 主导航区域 - 可滚动，底部通过额外 padding 为左下角入口腾出空间 -->
	<!-- 移动端预留: 浏览锚点(56px + p-3*2) + 主题切换(44px + mt-3) + safe-area = 约140px -->
	<!-- 桌面端预留: 浏览锚点(56px + p-3*2) + safe-area = 约80px -->
	<nav class="flex-1 min-h-0 overflow-y-auto px-3 pt-4 pb-28 custom-scrollbar">
        <!-- 主导航 -->
        <div class="space-y-1">
            {#each mainNavItems as item (item.href)}
                <RippleNavItem href={item.href} active={isActive(item.href)}>
                    <item.icon
                        class="h-5 w-5 transition-transform group-hover:scale-110"
                    />
                    <span>{item.label}</span>
                </RippleNavItem>
            {/each}
        </div>
        <!-- 个人中心 -->
        <div class="mt-6">
            <h3
                class="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground/70"
            >
                个人中心
            </h3>
            <div class="space-y-1">
                {#each personalNavItems as item (item.href)}
                    <RippleNavItem
                        href={item.href}
                        active={isActive(item.href)}
                        onclick={(e) => handleNavClick(item, e)}
                    >
                        <div class="relative">
                            <item.icon
                                class="h-5 w-5 transition-transform group-hover:scale-110"
                            />
                            <!-- 通知角标 -->
                            {#if authStore.isAuthenticated && item.href === "/notifications" && notificationStore.unreadCount > 0}
                                <Badge
                                    variant="destructive"
                                    class="absolute -right-2 -top-2 h-5 min-w-5 px-1 text-[10px] font-bold flex items-center justify-center rounded-2xl"
                                >
                                    {notificationStore.unreadCount > 99
                                        ? "99+"
                                        : notificationStore.unreadCount}
                                </Badge>
                            {/if}
                        </div>
                        <span class="flex-1">{item.label}</span>
                        <!-- 消息未读角标（在文字右侧） -->
                        {#if authStore.isAuthenticated && item.href === "/chat" && messageStore.totalUnreadCount > 0}
                            <Badge
                                variant="destructive"
                                class="h-5 min-w-5 px-1.5 text-[10px] font-bold flex items-center justify-center rounded-2xl"
                            >
                                {messageStore.totalUnreadCount > 99
                                    ? "99+"
                                    : messageStore.totalUnreadCount}
                            </Badge>
                        {/if}
                        {#if !authStore.isAuthenticated}
                            <span
                                class="rounded-2xl bg-muted px-2 py-0.5 text-xs text-muted-foreground"
                            >
                                登录
                            </span>
                        {/if}
                    </RippleNavItem>
                {/each}
            </div>
        </div>

        {#if authStore.isAuthenticated && isAdminUser}
            <div class="mt-6">
                <h3
                    class="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground/70"
                >
                    管理员
                </h3>
                <RippleNavItem href="/admin" active={isActive("/admin")}>
                    <Shield
                        class="h-5 w-5 transition-transform group-hover:scale-110"
                    />
                    <span>管理后台</span>
                </RippleNavItem>
            </div>
        {/if}
    </nav>

	<!-- 底部区域：固定在侧边栏左下角，通过 absolute 锚定 -->
	<div class="pointer-events-none absolute inset-x-0 bottom-0">
		<div
			class="pointer-events-auto border-t border-sidebar-border/30 bg-sidebar/98 backdrop-blur-sm p-3"
			style="padding-bottom: max(0.75rem, env(safe-area-inset-bottom, 0rem));"
		>
			<!-- 浏览锚点按钮 -->
			<Button
				variant="ghost"
				class="group relative flex w-full items-center gap-3 overflow-hidden rounded-xl bg-linear-to-r from-primary/5 to-primary/10 p-3 text-left transition-all duration-200 hover:from-primary/10 hover:to-primary/20 min-h-14"
				onclick={() => (showLocationPicker = true)}
				{@attach rippleFeedback}
			>
				<div
					class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary transition-transform group-hover:scale-105"
				>
					<Navigation class="h-5 w-5" />
				</div>
				<div class="min-w-0 flex-1">
					<div class="text-xs font-medium text-muted-foreground">
						浏览锚点
					</div>
					<div class="truncate text-sm font-semibold text-foreground">
						{mapAnchorStore.displayName}
					</div>
				</div>
				<ChevronRight
					class="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5"
				/>
			</Button>

			<!-- 主题切换 + 设置按钮（仅移动端显示） -->
			<div class="mt-3 flex items-center gap-2 md:hidden">
			<!-- 主题切换按钮 -->
			<Button
				variant="ghost"
				size="icon"
				class="group relative flex flex-1 items-center justify-center gap-2 overflow-hidden rounded-lg bg-muted/40 p-3 transition-all duration-200 hover:bg-muted/60 min-h-11"
				onclick={toggleMode}
				{@attach circularFeedback}
				aria-label="切换主题"
			>
                <div
                    class="flex h-5 w-5 items-center justify-center text-muted-foreground transition-transform group-hover:scale-110"
                >
                    {#if currentMode === "dark"}
                        <Sun class="h-5 w-5" />
                    {:else}
                        <Moon class="h-5 w-5" />
                    {/if}
                </div>
				<span class="text-sm font-medium text-foreground">
					{currentMode === "dark" ? "浅色" : "深色"}
				</span>
			</Button>

			<!-- 设置按钮 -->
			<Button
				variant="ghost"
				size="icon"
				class="group relative flex items-center justify-center overflow-hidden rounded-lg bg-muted/40 p-3 transition-all duration-200 hover:bg-muted/60 min-h-11 min-w-11"
				onclick={() => goto("/settings")}
				{@attach circularFeedback}
				aria-label="设置"
			>
				<Settings
					class="h-5 w-5 text-muted-foreground transition-transform group-hover:scale-110"
				/>
			</Button>
			</div>
		</div>
	</div>
</aside>
<LocationPickerModal bind:open={showLocationPicker} />
