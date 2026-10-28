<script lang="ts">
	import { MoreHorizontal, PencilLine, Trash2 } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import MomentCard from '@/components/moments/MomentCard.svelte';
	import MomentEditDialog from '@/components/overlays/MomentEditDialog.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { deleteMoment } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import { platform } from '@/stores/platform.svelte';
	import type { FeedItem } from '@/components/moments/moment-feed.svelte';

	// 我的瞬间卡 + 管理菜单（R13 评审 P1-7 批准）：
	// 桌面 hover 显现、触屏常显（desktop-reveal 契约）；菜单项复用详情页的「编辑/删除」，
	// 编辑走 MomentEditDialog（与详情页同一实现），删除走 ConfirmDialog + deleteMoment。
	// 删除成功后由父级从本地列表移除（onDeleted），不依赖重新拉取。

	let {
		item,
		liked,
		saved,
		onLike,
		onSave,
		onClick,
		onCommentClick,
		priority = false,
		onDeleted,
		onEdited
	}: {
		item: FeedItem;
		liked: boolean;
		saved: boolean;
		onLike: () => void;
		onSave: () => void;
		onClick?: () => void;
		onCommentClick?: () => void;
		priority?: boolean;
		onDeleted: () => void;
		onEdited: (updated: { description: string; tags: string[]; privacy_level: 'public' | 'private' | null }) => void;
	} = $props();

	const isMobile = $derived(platform.platform === 'mobile');
	let moreOpen = $state(false);
	let menuAnchor = $state<HTMLElement | null>(null);
	let editOpen = $state(false);
	let deleteOpen = $state(false);
	let deleting = $state(false);

	// 编辑初值：内容/标签来自列表数据；可见性未知（列表不返回 privacy_level）时传 null，
	// MomentEditDialog 会省略该字段（patch 语义保持原值），不会把私密瞬间误存为公开。
	const editInitial = $derived({
		description: item.content ?? '',
		tags: item.tags ?? (item.tag ? [item.tag] : []),
		privacy_level: null as 'public' | 'private' | null
	});

	async function handleDelete() {
		if (deleting) return;
		deleting = true;
		try {
			const res = await deleteMoment({ moment_id: item.id });
			if (res.code === 0) {
				showToast('瞬间已删除', 'success');
				deleteOpen = false;
				onDeleted();
			} else {
				showToast(res.message || '删除失败', 'danger');
			}
		} catch {
			showToast('删除失败，请重试', 'danger');
		} finally {
			deleting = false;
		}
	}
</script>

<div class="group relative isolate">
	<MomentCard
		class="aspect-[3/4] h-auto rounded-lg"
		{item}
		{liked}
		{saved}
		{priority}
		compact={isMobile}
		onLike={onLike}
		onSave={onSave}
		onClick={onClick ?? (() => {})}
		onCommentClick={onCommentClick}
	/>

	<!-- 管理入口：压在卡片右上角（标签在左上，不打架）；菜单展开期间保持可见 -->
	<div
		class="desktop-reveal absolute right-2 top-2 z-20"
		style={moreOpen ? 'opacity:1' : undefined}
	>
		<DropdownMenu.Root bind:open={moreOpen}>
			<div bind:this={menuAnchor}>
				<ActionIconButton
					icon={MoreHorizontal}
					label="管理这条瞬间"
					variant="overlay"
					tone="inverse"
					size={32}
					onClick={() => (moreOpen = !moreOpen)}
				/>
			</div>
			<DropdownMenu.Content customAnchor={menuAnchor}>
				<DropdownMenu.Item
					onSelect={() => {
						moreOpen = false;
						editOpen = true;
					}}
				>
					<PencilLine class="size-3.5 text-accent-default" strokeWidth={1.75} />
					编辑瞬间
				</DropdownMenu.Item>
				<DropdownMenu.Item
					class="text-status-danger hover:text-text-status-danger-on-soft data-[highlighted]:text-text-status-danger-on-soft hover:bg-danger-bg data-[highlighted]:bg-danger-bg"
					onSelect={() => {
						moreOpen = false;
						deleteOpen = true;
					}}
				>
					<Trash2 class="size-3.5" strokeWidth={1.75} />
					删除瞬间
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</div>

	<MomentEditDialog
		open={editOpen}
		momentId={item.id}
		initial={editInitial}
		onClose={() => (editOpen = false)}
		onUpdated={(updated) => onEdited(updated)}
	/>

	<ConfirmDialog
		open={deleteOpen}
		tone="danger"
		icon={Trash2}
		title="删除这条瞬间？"
		description="删除后不可恢复，附近的人将不再看到它。"
		confirmLabel="删除"
		layout={isMobile ? 'stacked' : 'row'}
		loading={deleting}
		onCancel={() => (deleteOpen = false)}
		onConfirm={() => void handleDelete()}
	/>
</div>