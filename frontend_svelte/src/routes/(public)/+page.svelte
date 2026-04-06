<script lang="ts">
    import { authStore } from "$lib/stores/auth.svelte";
    import { uiStore } from "$lib/stores/ui.svelte";
    import { mapAnchorStore } from "$lib/stores/map-anchor.svelte";
    import { t } from "$lib/i18n";
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

    type HomeMode = "guest" | "explore" | "authenticated";

    interface RefreshOptions {
        refreshFeed?: boolean;
        refreshWidgets?: boolean;
        refreshGuestTrending?: boolean;
    }

    const features = $derived([
        {
            icon: MapPin,
            title: t("home.locationBased"),
            description: t("home.discoverNearbyContent"),
            color: "from-blue-500/20 to-cyan-500/20",
            iconColor: "text-blue-500",
        },
        {
            icon: Image,
            title: t("home.lifeMoments"),
            description: t("home.shareLifeMoments"),
            color: "from-pink-500/20 to-rose-500/20",
            iconColor: "text-pink-500",
        },
        {
            icon: Users,
            title: t("home.groupInteraction"),
            description: t("home.joinInterestGroups"),
            color: "from-purple-500/20 to-violet-500/20",
            iconColor: "text-purple-500",
        },
        {
            icon: MessageCircle,
            title: t("home.instantChat"),
            description: t("home.chatWithFriends"),
            color: "from-green-500/20 to-emerald-500/20",
            iconColor: "text-green-500",
        },
    ]);

    let refreshTrigger = $state(0);
    let showExplore = $state(false);
    let trendingTags = $state.raw<TagItem[]>([]);
    let loadingTags = $state(false);
    let lastMode = $state<HomeMode>("guest");
    let lastAnchorUpdatedAt = $state<number | null>(null);
    let locationDebounceTimer: ReturnType<typeof setTimeout> | null = null;

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

    function getHomeMode(): HomeMode {
        if (authStore.isAuthenticated) return "authenticated";
        if (showExplore) return "explore";
        return "guest";
    }

    function isContentActive() {
        return getHomeMode() !== "guest";
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
        if (!isContentActive()) return;

        await feed.load(
            async (cursor) => {
                if (authStore.isAuthenticated) {
                    return await feedService.getMoments({
                        reference_point: getAnchor(),
                        source: "all",
                        radius: mapAnchorStore.radius * 1000,
                        pagination: { limit: 20, cursor },
                    });
                }

                return await momentsService.search({
                    radius: mapAnchorStore.radius * 1000,
                    reference_point: getAnchor(),
                    pagination: { limit: 20, cursor },
                });
            },
            refresh,
        );
    }

    async function refreshHome(options: RefreshOptions = {}) {
        const {
            refreshFeed = true,
            refreshWidgets = authStore.isAuthenticated,
            refreshGuestTrending = showExplore,
        } = options;

        if (refreshWidgets && authStore.isAuthenticated) {
            refreshTrigger++;
        }

        const tasks: Promise<void>[] = [];

        if (refreshGuestTrending && showExplore) {
            tasks.push(loadTrendingTags());
        }

        if (refreshFeed && isContentActive()) {
            tasks.push(loadFeed(true));
        }

        await Promise.all(tasks);
    }

    function refreshAll() {
        void refreshHome();
    }

    function handleGetStarted() {
        uiStore.openLoginModal(() => {
            showExplore = false;

            if (getHomeMode() === "authenticated") {
                void refreshHome({ refreshWidgets: true, refreshGuestTrending: false });
            }
        });
    }

    function handleExplore() {
        if (showExplore) {
            void refreshHome({ refreshWidgets: false, refreshGuestTrending: true });
            return;
        }

        showExplore = true;
    }

    $effect(() => {
        const mode = getHomeMode();

        if (mode === lastMode) {
            return;
        }

        lastMode = mode;
        lastAnchorUpdatedAt = null;

        if (mode === "guest") {
            trendingTags = [];
            loadingTags = false;
            feed.reset();
            return;
        }

        void refreshHome({
            refreshWidgets: mode === "authenticated",
            refreshGuestTrending: mode === "explore",
        });
    });

    $effect(() => {
        const mode = getHomeMode();

        if (mode === "guest") {
            if (locationDebounceTimer) {
                clearTimeout(locationDebounceTimer);
                locationDebounceTimer = null;
            }
            lastAnchorUpdatedAt = null;
            return;
        }

        const anchorUpdatedAt = mapAnchorStore.anchor.updatedAt;

        if (lastAnchorUpdatedAt === null) {
            lastAnchorUpdatedAt = anchorUpdatedAt;
            return;
        }

        if (anchorUpdatedAt === lastAnchorUpdatedAt) {
            return;
        }

        lastAnchorUpdatedAt = anchorUpdatedAt;

        if (locationDebounceTimer) {
            clearTimeout(locationDebounceTimer);
        }

        locationDebounceTimer = setTimeout(() => {
            void refreshHome({
                refreshWidgets: mode === "authenticated",
                refreshGuestTrending: mode === "explore",
            });
        }, 500);

        return () => {
            if (locationDebounceTimer) {
                clearTimeout(locationDebounceTimer);
                locationDebounceTimer = null;
            }
        };
    });
</script>

<svelte:head>
    <title>{t('home.title')}</title>
</svelte:head>

<div class="space-y-6 md:space-y-10">
    {#if !authStore.isAuthenticated}
        <!-- Hero section for guests -->
        <section
            class="relative overflow-hidden rounded-[2rem] border border-border/80 bg-card/95 p-5 shadow-[10px_10px_22px_var(--shadow-dark),-10px_-10px_22px_var(--shadow-light)] md:rounded-3xl md:bg-linear-to-br md:from-primary/10 md:via-primary/15 md:to-secondary/10 md:p-16 md:shadow-xl md:shadow-primary/10 md:ring-1 md:ring-primary/10"
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
                    class="mb-4 inline-flex items-center gap-2 rounded-[1.25rem] border border-border/70 bg-accent px-4 py-2 text-sm font-medium text-primary shadow-[6px_6px_14px_var(--shadow-dark),-6px_-6px_14px_var(--shadow-light)] md:mb-6 md:rounded-2xl md:bg-primary/10 md:shadow-none"
                >
                    <Sparkles class="h-4 w-4" />
                    {t('home.locationBasedSocial')}
                </div>
                <h1
                    class="mb-4 text-[2rem] font-bold tracking-tight md:mb-6 md:text-5xl lg:text-6xl"
                >
                    <span
                        class="bg-linear-to-r from-primary via-primary/80 to-primary/60 bg-clip-text text-transparent"
                    >
                        {t('home.discoverNearby')}
                    </span>
                </h1>
                <p
                    class="mx-auto mb-10 max-w-xl text-lg text-muted-foreground md:text-xl"
                >
                    {t('home.heroDescription')}
                </p>
                <div class="flex flex-col gap-3 sm:flex-row sm:justify-center">
                    <Button
                        size="lg"
                        class="gap-2 rounded-[1.5rem] px-8"
                        onclick={handleGetStarted}
                    >
                        {t('home.getStarted')}
                        <ArrowRight class="h-4 w-4" />
                    </Button>
                    <Button
                        size="lg"
                        variant="outline"
                        class="rounded-[1.5rem] px-8"
                        onclick={handleExplore}
                    >
                        {t('home.browseFirst')}
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
                    class="rounded-[1.75rem] border border-border/80 bg-card/92 p-4 shadow-[10px_10px_22px_var(--shadow-dark),-10px_-10px_22px_var(--shadow-light)] backdrop-blur-md"
                >
                    <div class="flex items-center justify-between">
                        <div class="space-y-1">
                            <h2 class="text-2xl font-bold tracking-tight">
                                {t('home.explore')}
                            </h2>
                            <p class="text-muted-foreground">
                                {t('home.discoverNearbyPeopleAndThings')}
                            </p>
                        </div>
                        <Button
                            variant="ghost"
                            size="icon"
                            class="rounded-2xl"
                            onclick={refreshAll}
                            aria-label={t('common.refresh')}
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
                        class="space-y-4 rounded-[1.75rem] border border-border/80 bg-card/92 p-4 shadow-[10px_10px_22px_var(--shadow-dark),-10px_-10px_22px_var(--shadow-light)] backdrop-blur-md"
                    >
                        <div class="flex items-center gap-2 px-1">
                            <Hash class="h-5 w-5 text-primary" />
                            <h3 class="text-lg font-semibold tracking-tight">
                                {t('home.trendingTopics')}
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
                    class="space-y-6 rounded-[1.75rem] border border-border/80 bg-card/92 p-4 shadow-[10px_10px_22px_var(--shadow-dark),-10px_-10px_22px_var(--shadow-light)] backdrop-blur-md"
                >
                    <div class="flex items-center justify-between px-1">
                        <div class="flex items-center gap-2">
                            <Sparkles class="h-5 w-5 text-primary" />
                            <h3 class="text-lg font-semibold tracking-tight">
                                {t('home.recommendedMoments')}
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
                            title={t('home.loadMomentsFailed')}
                            description={t('home.pleaseCheckNetwork')}
                            actionText={t('common.retry')}
                            onAction={() => loadFeed(true)}
                        />
                    {:else if feed.items.length === 0}
                        <EmptyState
                            icon={Compass}
                            title={t('home.noRecommendations')}
                            description={t('home.noRecommendationsDesc')}
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
                        {t('home.coreFeatures')}
                    </h2>
                    <p class="text-muted-foreground">
                        {t('home.exploreHotSpot')}
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
                    <div class="text-sm text-muted-foreground">{t('home.activeUsers')}</div>
                </div>
                <div>
                    <div
                        class="mb-2 text-3xl font-bold text-primary md:text-4xl"
                    >
                        50K+
                    </div>
                    <div class="text-sm text-muted-foreground">{t('home.greatMoments')}</div>
                </div>
                <div>
                    <div
                        class="mb-2 text-3xl font-bold text-primary md:text-4xl"
                    >
                        1K+
                    </div>
                    <div class="text-sm text-muted-foreground">{t('home.activeGroups')}</div>
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
                            {t('home.welcomeBack', { nickname: authStore.currentAccount?.nickname ?? '' })}
                        </h1>
                        <p class="text-muted-foreground">{t('home.discoverNearbyContent2')}</p>
                    </div>
                    <Button
                        variant="ghost"
                        size="icon"
                        class="rounded-2xl"
                        onclick={refreshAll}
                        disabled={feed.loading}
                        aria-label={t('common.refresh')}
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
                                {t('home.recommendedMoments')}
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
                        <h3 class="text-lg font-medium">{t('home.noRecommendations')}</h3>
                        <p class="text-muted-foreground">
                            {t('home.noRecommendationsDesc')}
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
                                    {t('common.loading')}
                                {:else}
                                    {t('common.loadMore')}
                                {/if}
                            </Button>
                        </div>
                    {/if}
                {/if}
            </section>
        </div>
    {/if}
</div>
