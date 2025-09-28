<template>
    <UApp 
        :toaster="{
            position: 'bottom-right',
            expand: true,
            progress: true,
            duration: 5000
        }"
    >
        <NuxtLayout>
            <NuxtPage />
        </NuxtLayout>
    </UApp>
</template>

<script setup lang="ts">
// 设置应用头部信息
useSeoMeta({
    title: "HotSpot - 基于位置的社交平台",
    description: "发现身边有趣的人和事",
});
</script>

<style>
html { scroll-behavior: smooth; }
</style>
