<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { RippleButton } from '$lib/components/ui/interactive';
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
		class="left-1/2 top-auto bottom-0 w-[calc(100%-1rem)] max-w-sm translate-x-[-50%] translate-y-0 rounded-t-[2rem] rounded-b-none border border-border/80 bg-card/96 px-5 pb-[calc(1.25rem+env(safe-area-inset-bottom,0px))] pt-5 shadow-[0_-18px_40px_rgba(74,64,54,0.18)] ring-0 dark:shadow-[0_-18px_44px_rgba(0,0,0,0.45)]"
		hideClose
	>
		<div class="mx-auto mb-4 h-1.5 w-12 rounded-full bg-muted-foreground/20"></div>
		<div class="mb-5 space-y-1 text-center">
			<h2 class="text-lg font-semibold text-foreground">{t('nav.createSheetTitle')}</h2>
			<p class="text-sm text-muted-foreground">{t('nav.createSheetDescription')}</p>
		</div>

		<div class="grid gap-3">
			<RippleButton
				variant="default"
				class="h-16 justify-start rounded-[1.5rem] border border-border/80 bg-card px-4 shadow-[6px_6px_16px_var(--shadow-dark),-6px_-6px_16px_var(--shadow-light)]"
				onclick={handleCreateMoment}
			>
				<div class="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/12 text-primary">
					<Image class="h-5 w-5" />
				</div>
				<div class="flex flex-col items-start">
					<span class="text-sm font-semibold text-foreground">{t('nav.postMoment')}</span>
					<span class="text-xs text-muted-foreground">{t('nav.createMomentHint')}</span>
				</div>
			</RippleButton>

			<RippleButton
				variant="default"
				class="h-16 justify-start rounded-[1.5rem] border border-border/80 bg-card px-4 shadow-[6px_6px_16px_var(--shadow-dark),-6px_-6px_16px_var(--shadow-light)]"
				onclick={handleCreateGroup}
			>
				<div class="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/12 text-primary">
					<UsersRound class="h-5 w-5" />
				</div>
				<div class="flex flex-col items-start">
					<span class="text-sm font-semibold text-foreground">{t('nav.createGroup')}</span>
					<span class="text-xs text-muted-foreground">{t('nav.createGroupHint')}</span>
				</div>
			</RippleButton>
		</div>
	</Dialog.Content>
</Dialog.Root>
