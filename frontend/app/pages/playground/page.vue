<script setup lang="ts">

</script>

<template>
    <UPage>
        <template #left>
            <UPageAside>
                <UChangelogVersion
                    title="Introducing Nuxt UI v3"
                    description="Nuxt UI v3 is out! After 1500+ commits, this major redesign brings improved accessibility, Tailwind CSS support, and full Vue compatibility."
                />

            </UPageAside>
        </template>

        <template #right>
            <UPageAside>
                <UChangelogVersion
                    title="AAAAAA"
                    description="BBBBB"
                />
            </UPageAside>
        </template>

        <slot />
    </UPage>
</template>