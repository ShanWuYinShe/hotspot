<template>
  <div class="min-h-screen grid grid-cols-1 lg:grid-cols-5 bg-neutral-50 dark:bg-neutral-950">
    <!-- Left Sidebar: Branding (2/5 on desktop, hidden on mobile) -->
    <aside
      class="hidden lg:flex lg:col-span-2 flex-col justify-between items-center p-8 border-r border-neutral-200/60 dark:border-neutral-800/60 bg-linear-to-br from-primary-50/50 via-white to-neutral-50 dark:from-primary-950/30 dark:via-neutral-950 dark:to-neutral-950"
    >
      <!-- Logo Area -->
      <div class="flex flex-col items-center text-center gap-4">
        <div class="relative flex items-center justify-center">
          <div class="absolute inset-0 bg-primary-500/20 rounded-full blur-2xl animate-pulse" />
          <UAvatar
            icon="i-lucide-flame"
            size="xl"
            color="primary"
            variant="solid"
            :src="'/logo_256.png'"
          />
        </div>
        <UPageHeader
          title="HotSpot"
          description="发现身边的精彩瞬间"
        />
      </div>

      <!-- Footer -->
      <UFooter>
        <template #left>
          <span class="text-xs text-neutral-500 dark:text-neutral-400">
            © {{ new Date().getFullYear() }} HotSpot
          </span>
        </template>
        <template #right>
          <UColorModeButton
            variant="ghost"
            color="neutral"
            size="xs"
          />
        </template>
      </UFooter>
    </aside>

    <!-- Main Content: Auth Card (3/5 on desktop, full on mobile) -->
    <main class="lg:col-span-3 flex flex-col items-center justify-center p-6">
      <div class="w-full max-w-lg animate-fade-in-up">
        <!-- Mobile Logo (only on mobile) -->
        <div class="lg:hidden mb-4">
          <div class="flex flex-col items-center justify-center text-center gap-3">
            <div class="relative flex items-center justify-center">
              <div class="absolute inset-0 bg-primary-500/20 rounded-full blur-xl animate-pulse" />
              <UAvatar
                icon="i-lucide-flame"
                size="xl"
                color="primary"
                variant="solid"
              />
            </div>
            <UPageHeader
              title="HotSpot"
              description="发现身边的精彩瞬间"
            />
          </div>
        </div>

        <!-- Auth Card -->
        <slot />

        <!-- Mobile Footer (only on mobile) -->
        <UFooter class="lg:hidden mt-6">
          <template #left>
            <span class="text-sm text-neutral-500 dark:text-neutral-400">
              © {{ new Date().getFullYear() }} HotSpot
            </span>
          </template>
          <template #right>
            <UColorModeButton
              variant="ghost"
              color="neutral"
              size="xs"
            />
          </template>
        </UFooter>
      </div>
    </main>
  </div>
</template>
