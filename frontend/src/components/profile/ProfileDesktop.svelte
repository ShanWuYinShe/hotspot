<script lang="ts">
	import { goto } from '$app/navigation';
	import { Share2, MapPin, ChevronRight, Camera, Pencil, Grid3X3, Users, Heart, MessageCircle } from '@lucide/svelte';
	import ActionIconButton from '@/components/shared/ActionIconButton.svelte';
	import PageHeader from '@/components/shared/PageHeader.svelte';
	import SectionMoments from '@/components/profile/SectionMoments.svelte';
	import SectionGroups from '@/components/profile/SectionGroups.svelte';
	import RelationshipsWorkspace from '@/components/profile/RelationshipsWorkspace.svelte';
	import { useAccountData, useGalleryPosts, useMyGroups, useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES, type ProfileSection } from '@/lib/routes';

	// 桌面个人主页：总览（左资料面板 + 右最近瞬间）与分区全宽两种模式。
	// 迁移自 React ProfileFeature.tsx 内 ProfileDesktop。

	let { section }: { section: string | null } = $props();

	const { avatar, nickname, stats } = useAccountData();
	const { posts: myGalleryPosts } = useGalleryPosts('my');
	const { friends, following, followers } = useRelationships();
	const { groups: myGroups } = useMyGroups();

	const SECTION_META: Record<string, { title: string }> = {
		moments: { title: '我的瞬间' },
		groups: { title: '我的群组' },
		relationships: { title: '关系网络' }
	};
	const meta = $derived(section && SECTION_META[section] ? SECTION_META[section] : null);
</script>

{#if meta}
	<!-- 分区全宽模式 -->
	<div class="flex flex-col h-full">
		<PageHeader title={meta.title} sticky onBack={() => void goto(ROUTES.me)} />
		<div class="flex-1 min-h-0 overflow-hidden flex">
			{#if section === 'moments'}
				<SectionMoments />
			{:else if section === 'groups'}
				<SectionGroups />
			{:else}
				<RelationshipsWorkspace />
			{/if}
		</div>
	</div>
{:else}
	<!-- 个人主页总览 -->
	<div class="flex flex-col h-full">
		<!-- Sticky header -->
		<div class="flex-shrink-0 flex items-center justify-between px-6 py-4 bg-[var(--neu-bg)]" style="border-bottom:1px solid var(--neu-border);">
			<div>
				<p class="text-[var(--neu-text)] text-xl">鹿鸣</p>
				<p class="text-[var(--neu-text-muted)] text-xs mt-px">@elara.v</p>
			</div>
			<div class="flex gap-2">
				<ActionIconButton icon={Share2} label="分享个人主页" size={36} />
			</div>
		</div>

		<div class="flex flex-1 min-h-0">
			<!-- Left profile panel -->
			<div class="flex-shrink-0 overflow-y-auto no-scrollbar px-6 py-6 flex flex-col items-center" style="width:260px;border-right:1px solid var(--neu-border);">
				<!-- Avatar -->
				<div class="relative mb-4">
					<div class="absolute inset-0 rounded-full pointer-events-none bg-[var(--neu-glow)]" style="filter:blur(20px);transform:scale(1.2);opacity:0.5;"></div>
					<div class="relative rounded-full bg-[var(--neu-bg)]" style="width:96px;height:96px;padding:2px;box-shadow:8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light);border:1.5px solid var(--neu-border);">
						<img src={avatar} alt={nickname || '用户'} class="w-full h-full object-cover rounded-full" />
					</div>
					<button
						type="button"
						class="neu-hover-btn absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none bg-[var(--neu-accent-primary)]"
						style="width:26px;height:26px;border:2.5px solid var(--neu-bg);box-shadow:0 2px 8px var(--neu-glow);"
					>
						<Camera style="width:12px;height:12px;color:white;" strokeWidth={2} />
					</button>
				</div>

				<h2 class="text-[var(--neu-text)] text-lg text-center">鹿鸣</h2>
				<span class="text-[var(--neu-text-muted)] text-xs mt-0.5">@elara.v</span>
				<p class="text-[var(--neu-text-muted)] text-xs text-center mt-2 leading-relaxed px-2">
					触感思考者 · 在流动中寻觅静止
				</p>
				<div class="flex items-center gap-1 mt-2 mb-4">
					<MapPin style="width:11px;height:11px;color:var(--neu-accent-primary);" strokeWidth={1.5} />
					<span class="text-[var(--neu-text-muted)] text-[11px]">江苏·徐州</span>
				</div>

				<!-- Edit profile button -->
				<button
					type="button"
					class="neu-hover-btn w-full py-2.5 rounded-full outline-none mb-5 flex items-center justify-center gap-2 bg-[var(--neu-bg)] text-[var(--neu-text-muted)] text-[13px]"
					style="border:1px solid var(--neu-border);box-shadow:4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light);"
					onclick={() => void goto(ROUTES.meEdit)}
				>
					<Pencil style="width:13px;height:13px;" strokeWidth={1.5} />
					编辑资料
				</button>

				<!-- Stats -->
				<div class="w-full rounded-[18px] mb-4 bg-[var(--neu-bg)] overflow-hidden" style="box-shadow:0 0 0 1px var(--neu-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light);">
					<div style="overflow:hidden;border-radius:18px;">
						{#each [
							{ label: '瞬间', value: stats.moments_count ?? 0, section: 'moments' },
							{ label: '关注者', value: stats.followers_count ?? 0, section: 'followers' },
							{ label: '关注', value: stats.following_count ?? 0, section: 'following' }
						] as stat, i}
							{#if i > 0}
								<div class="bg-[var(--neu-border)]" style="height:1px;"></div>
							{/if}
							<button
								type="button"
								class="neu-hover-bg w-full flex items-center justify-between px-4 py-3 outline-none"
								style="cursor:pointer;"
								onclick={() => void goto(stat.section === 'followers' ? `${ROUTES.meSection('relationships')}?tab=followers` : ROUTES.meSection(stat.section as ProfileSection))}
							>
								<span class="text-[var(--neu-text-muted)] text-[13px]">{stat.label}</span>
								<div class="flex items-center gap-1.5">
									<span class="text-[var(--neu-text)] text-[17px]">{stat.value}</span>
									<ChevronRight style="width:12px;height:12px;color:var(--neu-text-muted);" strokeWidth={1.5} />
								</div>
							</button>
						{/each}
					</div>
				</div>

				<!-- Section quick links -->
				<div class="w-full rounded-[18px] bg-[var(--neu-bg)] overflow-hidden" style="box-shadow:0 0 0 1px var(--neu-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light);">
					<div style="overflow:hidden;border-radius:18px;">
						{#each [
							{ label: '我的瞬间', section: 'moments', icon: Grid3X3, count: myGalleryPosts.length },
							{ label: '我的群组', section: 'groups', icon: Users, count: myGroups.length },
							{ label: '关系网络', section: 'relationships', icon: Heart, count: friends.length + following.length + followers.length }
						] as item, i}
							{#if i > 0}
								<div class="bg-[var(--neu-border)]" style="height:1px;"></div>
							{/if}
							<button
								type="button"
								class="neu-hover-bg w-full flex items-center justify-between px-4 py-2.5 outline-none"
								onclick={() => void goto(ROUTES.meSection(item.section as ProfileSection))}
							>
								<div class="flex items-center gap-2.5">
									<item.icon style="width:14px;height:14px;color:var(--neu-text-muted);" strokeWidth={1.5} />
									<span class="text-[var(--neu-text-muted)] text-[13px]">{item.label}</span>
								</div>
								<div class="flex items-center gap-1">
									<span class="text-[var(--neu-text-muted)] text-xs">{item.count}</span>
									<ChevronRight style="width:12px;height:12px;color:var(--neu-text-muted);" strokeWidth={1.5} />
								</div>
							</button>
						{/each}
					</div>
				</div>
			</div>

			<!-- Right: recent moments preview -->
			<div class="flex-1 overflow-y-auto no-scrollbar px-6 py-5">
				<div class="flex items-center justify-between mb-4">
					<p class="text-[var(--neu-text)] text-[15px]">最近瞬间</p>
					<button
						type="button"
						class="neu-hover-btn flex items-center gap-1 outline-none text-[var(--neu-accent-primary)] text-xs"
						onclick={() => void goto(ROUTES.meSection('moments'))}
					>
						查看全部
						<ChevronRight style="width:13px;height:13px;" strokeWidth={2} />
					</button>
				</div>
				<div style="display:grid;grid-template-columns:repeat(3, 1fr);gap:10px;max-width:{3 * 240 + 2 * 10}px;">
					{#each myGalleryPosts.slice(0, 9) as post (post.id)}
						<div
							class="neu-hover-btn relative rounded-[16px] cursor-pointer overflow-hidden"
							style="aspect-ratio:1/1;box-shadow:0 0 0 1px var(--neu-border), 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light);"
							onclick={() => void goto(`${ROUTES.moment(post.id)}?from=me`)}
						>
							<div class="relative h-full w-full" style="overflow:hidden;border-radius:16px;">
								<img src={post.img} alt="" class="w-full h-full object-cover" />
								<div class="absolute inset-0 flex items-end p-2" style="background:linear-gradient(to top, rgba(0,0,0,0.6) 0%, transparent 55%);">
									<div class="flex items-center gap-2">
										<div class="flex items-center gap-0.5">
											<Heart style="width:10px;height:10px;color:white;" fill="white" strokeWidth={0} />
											<span class="text-white text-[10px]">{post.likes}</span>
										</div>
										<div class="flex items-center gap-0.5">
											<MessageCircle style="width:10px;height:10px;color:white;" strokeWidth={1.5} />
											<span class="text-white text-[10px]">{post.comments}</span>
										</div>
									</div>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</div>
		</div>
	</div>
{/if}
