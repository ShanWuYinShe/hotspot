<template>
    <div
        class="min-h-screen antialiased font-sans overflow-x-hidden"
    >
        <UContainer class="min-h-screen flex flex-col">
            <div class="flex-1">
                <slot />
            </div>
            <USeparator />
            <footer class="py-4 text-center text-sm flex items-center justify-center gap-3">
                <UColorModeSwitch/>
                <span>© {{ new Date().getFullYear() }} HotSpot</span>
            </footer>
        </UContainer>
    </div>
</template>

<script setup>
</script>
