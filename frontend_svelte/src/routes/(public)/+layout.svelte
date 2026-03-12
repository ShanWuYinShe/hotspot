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

    <div
        class="flex h-[calc(100dvh-4rem)] min-h-[calc(100dvh-4rem)] overflow-hidden bg-muted/30"
    >
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
                class="fixed left-0 top-14 z-50 w-64 md:hidden"
                style="height: calc(100dvh - 3.5rem - env(safe-area-inset-bottom, 0px));"
            >
                <Sidebar class="h-full bg-sidebar" />
            </div>
        {/if}

        <!-- Main content -->
        <main
            class="mx-3 flex-1 overflow-y-auto rounded-3xl border border-border/60 bg-card/85 p-4 shadow-sm backdrop-blur-md md:mx-4 md:p-6 no-scrollbar"
            style="padding-bottom: calc(1rem + env(safe-area-inset-bottom, 0px));"
        >
            <div class="mx-auto w-full max-w-7xl">
                {@render children()}
            </div>
        </main>
    </div>
</div>
