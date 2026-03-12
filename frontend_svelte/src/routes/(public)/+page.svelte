<script lang="ts">
    import { onMount, untrack } from "svelte";
    import { authStore } from "$lib/stores/auth.svelte";
    import { uiStore } from "$lib/stores/ui.svelte";
    import { mapAnchorStore } from "$lib/stores/map-anchor.svelte";
    import { feedService, momentsService } from "$lib/api/services";
    import { isSuccess } from "$lib/api/client";
    import { usePagination } from "$lib/hooks/use-pagination.svelte";
    import { Button } from "$lib/components/ui/button";
    import {
        Card,
        CardContent,
        CardDescription,
        CardHeader,
        CardTitle,
    } from "$lib/components/ui/card";
    import { Skeleton } from "$lib/components/ui/skeleton";
    import { Badge } from "$lib/components/ui/badge";
    import MomentCard from "$lib/components/moments/moment-card.svelte";
    import DashboardStats from "$lib/components/home/dashboard-stats.svelte";
    import TrendingTags from "$lib/components/home/trending-tags.svelte";
    import RecommendedUsers from "$lib/components/home/recommended-users.svelte";
    import RecommendedGroups from "$lib/components/home/recommended-groups.svelte";
    import ListSkeleton from "$lib/components/common/list-skeleton.svelte";
    import EmptyState from "$lib/components/common/empty-state.svelte";
    import LoadMoreButton from "$lib/components/common/load-more-button.svelte";
    import type { MomentItem, TagItem } from "$lib/types/api";
    import MapPin from "lucide-svelte/icons/map-pin";
    import Users from "lucide-svelte/icons/users";
    import Image from "lucide-svelte/icons/image";
    import MessageCircle from "lucide-svelte/icons/message-circle";
    import ArrowRight from "lucide-svelte/icons/arrow-right";
    import Sparkles from "lucide-svelte/icons/sparkles";
    import Compass from "lucide-svelte/icons/compass";
    import RefreshCw from "lucide-svelte/icons/refresh-cw";
    import Hash from "lucide-svelte/icons/hash";

    const features = [
        {
            icon: MapPin,
            title: "基于位置",
            description: "发现你附近的精彩内容和有趣的人",
            color: "from-blue-500/20 to-cyan-500/20",
            iconColor: "text-blue-500",
        },
        {
            icon: Image,
            title: "生活瞬间",
            description: "分享你的生活点滴，记录美好时刻",
            color: "from-pink-500/20 to-rose-500/20",
            iconColor: "text-pink-500",
        },
        {
            icon: Users,
            title: "群组互动",
            description: "加入兴趣群组，结识志同道合的朋友",
            color: "from-purple-500/20 to-violet-500/20",
            iconColor: "text-purple-500",
        },
        {
            icon: MessageCircle,
            title: "即时聊天",
            description: "与好友实时沟通，保持紧密联系",
            color: "from-green-500/20 to-emerald-500/20",
            iconColor: "text-green-500",
        },
    ];

    let refreshTrigger = $state(0);
    let showExplore = $state(false);

    // Data states for Explore mode
    let trendingTags = $state.raw<TagItem[]>([]);
    let loadingTags = $state(true);

    const feed = usePagination<MomentItem>({
        uniqueKey: "moment_id",
        initialLoading: true,
    });

    function getAnchor() {
        return {
            latitude: mapAnchorStore.latitude,
            longitude: mapAnchorStore.longitude,
        };
    }

    async function loadTrendingTags() {
        loadingTags = true;
        try {
            const response = await feedService.getTrending();
            if (isSuccess(response) && response.data) {
                trendingTags = response.data.trending_tags ?? [];
            }
        } catch {
            // ignore
        } finally {
            loadingTags = false;
        }
    }

    async function loadFeed(refresh = false) {
        // 如果未登录且不处于探索模式，不加载
        if (!authStore.isAuthenticated && !showExplore) return;

        await feed.load(async (cursor) => {
            if (authStore.isAuthenticated) {
                return await feedService.getMoments({
                    reference_point: getAnchor(),
                    source: "all",
                    radius: mapAnchorStore.radius * 1000,
                    pagination: { limit: 20, cursor },
                });
            } else {
                // 未登录：显示附近动态
                return await momentsService.search({
                    radius: mapAnchorStore.radius * 1000,
                    reference_point: getAnchor(),
                    pagination: { limit: 20, cursor },
                });
            }
        }, refresh);
    }

    function refreshAll() {
        refreshTrigger++;
        if (showExplore || authStore.isAuthenticated) {
            if (showExplore) loadTrendingTags();
            loadFeed(true);
        }
    }

    function handleGetStarted() {
        uiStore.openLoginModal(() => refreshAll());
    }

    function handleExplore() {
        showExplore = true;
        refreshAll();
    }

    onMount(() => {
        if (authStore.isAuthenticated) {
            refreshAll();
        }
    });

    // 监听位置变化自动刷新（添加防抖避免频繁请求）
    // 使用 untrack 防止追踪 refreshAll 内部的响应式访问（特别是 refreshTrigger 的自引用）
    let locationDebounceTimer: ReturnType<typeof setTimeout> | null = null;

    $effect(() => {
        if (authStore.isAuthenticated || showExplore) {
            mapAnchorStore.anchor.updatedAt;

            // 清除之前的定时器
            if (locationDebounceTimer) clearTimeout(locationDebounceTimer);

            // 500ms 防抖
            locationDebounceTimer = setTimeout(() => {
                untrack(() => refreshAll());
            }, 500);
        }

        // 清理函数
        return () => {
            if (locationDebounceTimer) {
                clearTimeout(locationDebounceTimer);
                locationDebounceTimer = null;
            }
        };
    });
</script>

<svelte:head>
    <title>HotSpot - 发现附近的精彩</title>
</svelte:head>

<div class="space-y-10">
    {#if !authStore.isAuthenticated}
        <!-- Hero section for guests -->
        <section
            class="relative overflow-hidden rounded-3xl bg-linear-to-br from-primary/10 via-primary/15 to-secondary/10 p-8 shadow-xl shadow-primary/10 ring-1 ring-primary/10 md:p-16"
        >
            <!-- Decorative elements -->
            <div
                class="absolute -right-20 -top-20 h-64 w-64 rounded-full bg-primary/10 blur-3xl"
            ></div>
            <div
                class="absolute -bottom-20 -left-20 h-64 w-64 rounded-full bg-secondary/10 blur-3xl"
            ></div>

            <div class="relative mx-auto max-w-3xl text-center">
                <div
                    class="mb-6 inline-flex items-center gap-2 rounded-2xl bg-primary/10 px-4 py-2 text-sm font-medium text-primary backdrop-blur-sm"
                >
                    <Sparkles class="h-4 w-4" />
                    基于位置的社交平台
                </div>
                <h1
                    class="mb-6 text-4xl font-bold tracking-tight md:text-5xl lg:text-6xl"
                >
                    <span
                        class="bg-linear-to-r from-primary via-primary/80 to-primary/60 bg-clip-text text-transparent"
                    >
                        发现附近的精彩
                    </span>
                </h1>
                <p
                    class="mx-auto mb-10 max-w-xl text-lg text-muted-foreground md:text-xl"
                >
                    HotSpot 帮助你发现身边有趣的人、精彩的瞬间和活跃的群组。
                    开始探索你的世界吧！
                </p>
                <div class="flex flex-col gap-4 sm:flex-row sm:justify-center">
                    <Button
                        size="lg"
                        class="gap-2 rounded-2xl px-8 shadow-lg shadow-primary/25 transition-all hover:shadow-xl hover:shadow-primary/30"
                        onclick={handleGetStarted}
                    >
                        立即开始
                        <ArrowRight class="h-4 w-4" />
                    </Button>
                    <Button
                        size="lg"
                        variant="outline"
                        class="rounded-2xl px-8"
                        onclick={handleExplore}
                    >
                        先逛逛看
                    </Button>
                </div>
            </div>
        </section>

        {#if showExplore}
            <!-- Explore Content (Trending + Feed) -->
            <div
                class="space-y-10 animate-in fade-in slide-in-from-bottom-4 duration-500"
            >
                <div
                    class="rounded-3xl border border-border/60 bg-card/80 p-4 shadow-sm backdrop-blur-md"
                >
                    <div class="flex items-center justify-between">
                        <div class="space-y-1">
                            <h2 class="text-2xl font-bold tracking-tight">
                                探索发现
                            </h2>
                            <p class="text-muted-foreground">
                                发现附近的精彩人与事
                            </p>
                        </div>
                        <Button
                            variant="ghost"
                            size="icon"
                            class="rounded-2xl"
                            onclick={refreshAll}
                            aria-label="刷新"
                        >
                            <RefreshCw
                                class="h-4 w-4 {feed.loading
                                    ? 'animate-spin'
                                    : ''}"
                            />
                        </Button>
                    </div>
                </div>

                <!-- Trending Topics -->
                {#if !loadingTags && trendingTags.length > 0}
                    <section
                        class="space-y-4 rounded-3xl border border-border/60 bg-card/80 p-4 shadow-sm backdrop-blur-md"
                    >
                        <div class="flex items-center gap-2 px-1">
                            <Hash class="h-5 w-5 text-primary" />
                            <h3 class="text-lg font-semibold tracking-tight">
                                热门话题
                            </h3>
                        </div>
                        <div class="flex flex-wrap gap-2">
                            {#each trendingTags as tag (tag.name)}
                                <a
                                    href="/search?tag={encodeURIComponent(
                                        tag.name,
                                    )}"
                                    class="group"
                                >
                                    <Badge
                                        variant="secondary"
                                        class="cursor-pointer rounded-2xl border border-border/50 bg-background/65 px-3 py-1.5 text-sm transition-all hover:border-primary/40 hover:bg-primary/10 hover:text-primary"
                                    >
                                        #{tag.name}
                                        <span
                                            class="ml-1.5 text-xs opacity-50 group-hover:opacity-100"
                                        >
                                            {tag.usage_count}
                                        </span>
                                    </Badge>
                                </a>
                            {/each}
                        </div>
                    </section>
                {/if}

                <!-- Moments Feed -->
                <section
                    class="space-y-6 rounded-3xl border border-border/60 bg-card/80 p-4 shadow-sm backdrop-blur-md"
                >
                    <div class="flex items-center justify-between px-1">
                        <div class="flex items-center gap-2">
                            <Sparkles class="h-5 w-5 text-primary" />
                            <h3 class="text-lg font-semibold tracking-tight">
                                推荐动态
                            </h3>
                        </div>
                    </div>

                    {#if feed.loading}
                        <div
                            class="columns-1 gap-3 sm:columns-2 lg:columns-3 xl:columns-4"
                        >
                            {#each Array(6) as _, i (i)}
                                <div class="mb-3 break-inside-avoid">
                                    <ListSkeleton
                                        type="moment"
                                        count={1}
                                        layout="list"
                                    />
                                </div>
                            {/each}
                        </div>
                    {:else if feed.error}
                        <EmptyState
                            icon={Compass}
                            title="加载动态失败"
                            description="请检查网络连接后重试"
                            actionText="重试"
                            onAction={() => loadFeed(true)}
                        />
                    {:else if feed.items.length === 0}
                        <EmptyState
                            icon={Compass}
                            title="暂无推荐"
                            description="去发布一些内容，或者探索更多区域吧"
                        />
                    {:else}
                        <div
                            class="columns-1 gap-3 sm:columns-2 lg:columns-3 xl:columns-4"
                        >
                            {#each feed.items as moment (moment.moment_id)}
                                <div class="mb-3 break-inside-avoid">
                                    <MomentCard
                                        {moment}
                                        class="border-border/60 bg-card/95 shadow-md shadow-black/5 dark:shadow-black/30"
                                    />
                                </div>
                            {/each}
                        </div>

                        <LoadMoreButton
                            loading={feed.loadingMore}
                            hasMore={feed.hasMore}
                            onLoadMore={() => loadFeed(false)}
                        />
                    {/if}
                </section>
            </div>
        {:else}
            <!-- Features -->
            <section>
                <div class="mb-8 text-center">
                    <h2 class="mb-2 text-2xl font-bold md:text-3xl">
                        核心功能
                    </h2>
                    <p class="text-muted-foreground">
                        探索 HotSpot 为你提供的精彩体验
                    </p>
                </div>
                <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
                    {#each features as feature (feature.title)}
                        <Card
                            class="group cursor-pointer border-0 bg-linear-to-br {feature.color} backdrop-blur-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-primary/10"
                        >
                            <CardHeader>
                                <div
                                    class="mb-3 inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-background/80 shadow-sm transition-transform group-hover:scale-110"
                                >
                                    <feature.icon
                                        class="h-6 w-6 {feature.iconColor}"
                                    />
                                </div>
                                <CardTitle class="text-lg">
                                    {feature.title}
                                </CardTitle>
                            </CardHeader>
                            <CardContent>
                                <CardDescription class="text-foreground/70">
                                    {feature.description}
                                </CardDescription>
                            </CardContent>
                        </Card>
                    {/each}
                </div>
            </section>
        {/if}

        <!-- Stats section -->
        <section
            class="rounded-3xl border border-border/60 bg-linear-to-r from-primary/5 to-secondary/5 p-8 shadow-sm md:p-12"
        >
            <div class="grid gap-8 text-center sm:grid-cols-3">
                <div>
                    <div
                        class="mb-2 text-3xl font-bold text-primary md:text-4xl"
                    >
                        10K+
                    </div>
                    <div class="text-sm text-muted-foreground">活跃用户</div>
                </div>
                <div>
                    <div
                        class="mb-2 text-3xl font-bold text-primary md:text-4xl"
                    >
                        50K+
                    </div>
                    <div class="text-sm text-muted-foreground">精彩瞬间</div>
                </div>
                <div>
                    <div
                        class="mb-2 text-3xl font-bold text-primary md:text-4xl"
                    >
                        1K+
                    </div>
                    <div class="text-sm text-muted-foreground">活跃群组</div>
                </div>
            </div>
        </section>
    {:else}
        <!-- Dashboard for authenticated users -->
        <div class="space-y-8">
            <!-- Header with stats -->
            <section>
                <div class="mb-6 flex items-center justify-between">
                    <div>
                        <h1 class="mb-2 text-2xl font-bold md:text-3xl">
                            欢迎回来，<span class="text-primary"
                                >{authStore.currentAccount?.nickname}</span
                            >！
                        </h1>
                        <p class="text-muted-foreground">发现附近的精彩内容</p>
                    </div>
                    <Button
                        variant="ghost"
                        size="icon"
                        class="rounded-2xl"
                        onclick={refreshAll}
                        disabled={feed.loading}
                        aria-label="刷新"
                    >
                        <RefreshCw
                            class="h-4 w-4 {feed.loading ? 'animate-spin' : ''}"
                        />
                    </Button>
                </div>

                <DashboardStats {refreshTrigger} />
            </section>

            <!-- Trending Topics -->
            <TrendingTags {refreshTrigger} />

            <!-- Recommended Users -->
            <RecommendedUsers {refreshTrigger} />

            <!-- Recommended Groups -->
            <RecommendedGroups {refreshTrigger} />

            <!-- Moments Feed -->
            <section class="space-y-6">
                <div
                    class="rounded-3xl border border-border/60 bg-card/80 p-4 shadow-sm backdrop-blur-md"
                >
                    <div class="flex items-center justify-between px-1">
                        <div class="flex items-center gap-2">
                            <Sparkles class="h-5 w-5 text-primary" />
                            <h2 class="text-lg font-semibold tracking-tight">
                                推荐动态
                            </h2>
                        </div>
                    </div>
                </div>

                {#if feed.loading}
                    <div
                        class="columns-1 gap-3 sm:columns-2 lg:columns-3 xl:columns-4"
                    >
                        {#each Array(6) as _, i (i)}
                            <div class="mb-3 break-inside-avoid">
                                <Card
                                    class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md"
                                >
                                    <CardContent class="space-y-3 p-4">
                                        <div class="flex items-center gap-3">
                                            <Skeleton
                                                class="h-10 w-10 rounded-full"
                                            />
                                            <div class="space-y-1.5">
                                                <Skeleton class="h-4 w-24" />
                                                <Skeleton class="h-3 w-16" />
                                            </div>
                                        </div>
                                        <Skeleton
                                            class="h-32 w-full rounded-lg"
                                        />
                                        <Skeleton class="h-4 w-full" />
                                        <Skeleton class="h-4 w-2/3" />
                                    </CardContent>
                                </Card>
                            </div>
                        {/each}
                    </div>
                {:else if feed.items.length === 0}
                    <div
                        class="flex flex-col items-center justify-center py-16 text-center"
                    >
                        <Compass
                            class="mb-4 h-12 w-12 text-muted-foreground/50"
                        />
                        <h3 class="text-lg font-medium">暂无推荐</h3>
                        <p class="text-muted-foreground">
                            去发布一些内容，或者探索更多区域吧
                        </p>
                    </div>
                {:else}
                    <div
                        class="columns-1 gap-3 sm:columns-2 lg:columns-3 xl:columns-4"
                    >
                        {#each feed.items as moment (moment.moment_id)}
                            <div class="mb-3 break-inside-avoid">
                                <MomentCard
                                    {moment}
                                    class="border-border/60 bg-card/95 shadow-md shadow-black/5 dark:shadow-black/30"
                                />
                            </div>
                        {/each}
                    </div>

                    {#if feed.hasMore}
                        <div class="flex justify-center pt-4">
                            <Button
                                variant="outline"
                                class="rounded-2xl px-8"
                                onclick={() => loadFeed(false)}
                                disabled={feed.loadingMore}
                            >
                                {#if feed.loadingMore}
                                    <RefreshCw
                                        class="mr-2 h-4 w-4 animate-spin"
                                    />
                                    加载中...
                                {:else}
                                    加载更多
                                {/if}
                            </Button>
                        </div>
                    {/if}
                {/if}
            </section>
        </div>
    {/if}
</div>
