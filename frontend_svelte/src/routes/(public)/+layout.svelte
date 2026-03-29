<script lang="ts">
    import Navbar from "$lib/components/common/navbar.svelte";
    import Sidebar from "$lib/components/common/sidebar.svelte";
    import { Button } from "$lib/components/ui/button";

    let { children } = $props();

    let sidebarOpen = $state(false);

    function closeSidebar() {
        sidebarOpen = false;
    }
</script>

<div class="min-h-screen bg-background">
    <Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

    <div class="app-shell-frame flex overflow-hidden bg-muted/20">
        <!-- Desktop sidebar -->
        <Sidebar class="hidden w-64 shrink-0 md:block" />
        <!-- Mobile sidebar overlay -->
        {#if sidebarOpen}
            <Button
                type="button"
                variant="ghost"
                size="icon"
                class="fixed inset-0 z-40 h-full w-full cursor-pointer bg-black/50 p-0 md:hidden border-none shadow-none"
                onclick={closeSidebar}
                onkeydown={(e) => e.key === "Escape" && closeSidebar()}
                aria-label="关闭侧边栏"
            />
            <div
                class="app-mobile-sidebar fixed left-0 z-50 w-64 md:hidden"
            >
                <Sidebar class="h-full bg-sidebar" />
            </div>
        {/if}

        <!-- Main content -->
        <main
            class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-3xl md:border md:border-border/60 md:bg-card/85 md:p-6 md:shadow-sm md:backdrop-blur-md no-scrollbar"
        >
            <div class="mobile-page-shell md:mx-auto md:max-w-7xl">
                {@render children()}
            </div>
        </main>
    </div>
</div>
