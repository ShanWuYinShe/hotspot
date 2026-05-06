<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { t } from '$lib/i18n';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { platformStore } from '$lib/stores/platform.svelte';
	import Image from 'lucide-svelte/icons/image';
	import UsersRound from 'lucide-svelte/icons/users-round';

	interface Props {}

	let {}: Props = $props();

	function handleCreateMoment() {
		uiStore.openCreateMoment();
	}

	function handleCreateGroup() {
		uiStore.openCreateGroup();
	}

	const isMobileOrTablet = $derived(platformStore.isMobile || platformStore.isTablet);
</script>

<Dialog.Root open={uiStore.showCreateSheet} onOpenChange={(open) => open ? uiStore.openCreateSheet() : uiStore.closeCreateSheet()}>
	<Dialog.Content
		class={isMobileOrTablet
			? 'left-1/2 top-auto bottom-0 w-[calc(100%-1rem)] max-w-sm translate-x-[-50%] translate-y-0 rounded-t-[28px] rounded-b-none px-5 pb-[calc(1.25rem+env(safe-area-inset-bottom,0px))] pt-5'
			: 'max-w-[480px] rounded-[24px] px-5 pt-5 pb-5'}
		style="background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: 0 -8px 40px rgba(0,0,0,0.18), 0 -2px 0 var(--neu-border);"
		hideClose
	>
		{#if isMobileOrTablet}
			<div class="mx-auto mb-4 h-1 w-9 rounded-full" style="background: var(--neu-border);"></div>
		{/if}
		<div class="mb-5 text-center">
			<h2 style="font-size: 17px; font-weight: 600; color: var(--neu-text);">{t('nav.createSheetTitle')}</h2>
			<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 4px;">{t('nav.createSheetDescription')}</p>
		</div>

		<div class="grid gap-3">
			<button
				class="flex h-16 items-center gap-3 rounded-[18px] px-4 text-left outline-none"
				style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
				onclick={handleCreateMoment}
			>
				<div
					class="flex items-center justify-center rounded-full"
					style="width: 38px; height: 38px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
				>
					<Image size={16} strokeWidth={1.5} color="var(--neu-accent-primary)" />
				</div>
				<div>
					<p style="font-size: 13px; font-weight: 600; color: var(--neu-text);">{t('nav.postMoment')}</p>
					<p style="font-size: 11px; color: var(--neu-text-muted);">{t('nav.createMomentHint')}</p>
				</div>
			</button>

			<button
				class="flex h-16 items-center gap-3 rounded-[18px] px-4 text-left outline-none"
				style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
				onclick={handleCreateGroup}
			>
				<div
					class="flex items-center justify-center rounded-full"
					style="width: 38px; height: 38px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
				>
					<UsersRound size={16} strokeWidth={1.5} color="var(--neu-fallback-text)" />
				</div>
				<div>
					<p style="font-size: 13px; font-weight: 600; color: var(--neu-text);">{t('nav.createGroup')}</p>
					<p style="font-size: 11px; color: var(--neu-text-muted);">{t('nav.createGroupHint')}</p>
				</div>
			</button>
		</div>
	</Dialog.Content>
</Dialog.Root>
