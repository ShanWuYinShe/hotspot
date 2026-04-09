<script lang="ts">
    import { onMount } from "svelte";
    import Navbar from "$lib/components/common/navbar.svelte";
    import Sidebar from "$lib/components/common/sidebar.svelte";
    import MobileBottomNav from "$lib/components/common/mobile-bottom-nav.svelte";
    import { Button } from "$lib/components/ui/button";
    import { t } from "$lib/i18n";

    let { children } = $props();

    let sidebarOpen = $state(false);
    let isMobile = $state(false);

    function closeSidebar() {
        sidebarOpen = false;
    }

    onMount(() => {
        const mediaQuery = window.matchMedia('(max-width: 767px)');
        const update = () => {
            isMobile = mediaQuery.matches;
        };
        update();
        mediaQuery.addEventListener('change', update);
        return () => mediaQuery.removeEventListener('change', update);
    });
</script>

<div class="min-h-screen bg-background">
    {#if isMobile}
        <div class="mobile-shell-frame">
            <Navbar showMenu={false} />
            <main class="mobile-app-main mobile-shell-content no-scrollbar">
                <div class="mobile-page-shell">
                    {@render children()}
                </div>
            </main>
            <MobileBottomNav />
        </div>
    {:else}
        <div>
            <Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

            <div class="app-shell-frame flex overflow-hidden bg-muted/20">
                <Sidebar class="hidden h-full w-64 shrink-0 md:block" />
                {#if sidebarOpen}
                    <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        class="fixed inset-0 z-40 h-full w-full cursor-pointer bg-black/50 p-0 md:hidden border-none shadow-none"
                        onclick={closeSidebar}
                        onkeydown={(e) => e.key === "Escape" && closeSidebar()}
                        aria-label={t('nav.closeSidebar')}
                    />
                    <div class="app-mobile-sidebar fixed left-0 z-50 w-64 md:hidden">
                        <Sidebar class="h-full bg-sidebar" />
                    </div>
                {/if}

                <main class="mobile-app-main no-scrollbar pb-safe-compact md:mx-4 md:rounded-3xl md:border md:border-border/60 md:bg-card/85 md:p-6 md:shadow-sm md:backdrop-blur-md">
                    <div class="mobile-page-shell md:mx-auto md:max-w-7xl">
                        {@render children()}
                    </div>
                </main>
            </div>
        </div>
    {/if}
</div>

