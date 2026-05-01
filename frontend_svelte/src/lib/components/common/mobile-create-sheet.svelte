<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { t } from '$lib/i18n';
	import Image from 'lucide-svelte/icons/image';
	import UsersRound from 'lucide-svelte/icons/users-round';

	import { uiStore } from '$lib/stores/ui.svelte';

	interface Props {}

	let {}: Props = $props();

	function handleCreateMoment() {
		uiStore.openCreateMoment();
	}

	function handleCreateGroup() {
		uiStore.openCreateGroup();
	}
</script>

<Dialog.Root open={uiStore.showCreateSheet} onOpenChange={(open) => open ? uiStore.openCreateSheet() : uiStore.closeCreateSheet()}>
	<Dialog.Content
		class="left-1/2 top-auto bottom-0 w-[calc(100%-1rem)] max-w-sm translate-x-[-50%] translate-y-0 rounded-t-[22px] rounded-b-none px-5 pb-[calc(1.25rem+env(safe-area-inset-bottom,0px))] pt-5"
		style="background: var(--neu-bg); border: 1px solid var(--neu-border); border-bottom: none; box-shadow: 0 -10px 30px var(--neu-shadow-dark);"
		hideClose
	>
		<div class="mx-auto mb-4 h-1.5 w-12 rounded-full" style="background: var(--neu-border);"></div>
		<div class="mb-5 text-center">
			<h2 style="font-size: 17px; font-weight: 600; color: var(--neu-text);">{t('nav.createSheetTitle')}</h2>
			<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 4px;">{t('nav.createSheetDescription')}</p>
		</div>

		<div class="grid gap-3">
			<button
				class="flex h-16 items-center gap-3 rounded-[18px] px-4 text-left outline-none"
				style="background: var(--neu-bg); box-shadow: 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
				onclick={handleCreateMoment}
			>
				<div
					class="flex items-center justify-center rounded-full"
					style="width: 38px; height: 38px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
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
				style="background: var(--neu-bg); box-shadow: 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
				onclick={handleCreateGroup}
			>
				<div
					class="flex items-center justify-center rounded-full"
					style="width: 38px; height: 38px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
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
