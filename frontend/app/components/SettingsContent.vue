<template>
    <div class="p-6 space-y-6" :aria-labelledby="'tab-' + activeSection">
        <Transition name="fade-scale" mode="out-in" @after-enter="emit('sectionFocusRequest')">
            <div :key="activeSection">
                <AccountSection
                    v-if="activeSection==='account'"
                    :user="user"
                    :local-settings="localSettings"
                    @change-avatar="emit('changeAvatar')"
                    @edit-nickname="emit('editNickname')"
                    @upgrade-account="emit('upgradeAccount')"
                    @change-password="emit('changePassword')"
                    @setup-2fa="emit('setup2FA')"
                />
                <PrivacySection
                    v-else-if="activeSection==='privacy'"
                    :local-settings="localSettings"
                />
                <NotificationsSection
                    v-else-if="activeSection==='notifications'"
                    :local-settings="localSettings"
                />
                <AppearanceSection
                    v-else-if="activeSection==='appearance'"
                    :local-settings="localSettings"
                />
                <AboutSection
                    v-else-if="activeSection==='about'"
                    @clear-cache="emit('clearCache')"
                    @show-help="emit('showHelp')"
                    @show-privacy-policy="emit('showPrivacyPolicy')"
                    @show-terms="emit('showTerms')"
                />
                <DangerSection
                    v-else-if="activeSection==='danger'"
                    @delete-account="emit('deleteAccount')"
                />
            </div>
        </Transition>
    </div>
</template>

<script setup lang="ts">
import type { UserSettings } from '~/types/api';

interface Props {
    activeSection: string;
    localSettings: UserSettings;
    user: any;
}

defineProps<Props>();

const emit = defineEmits([
    'saveNickname',
    'avatarUploaded',
    'upgradeAccount',
    'changePassword',
    'setup2FA',
    'clearCache',
    'showHelp',
    'showPrivacyPolicy',
    'showTerms',
    'deleteAccount',
    'editNickname',
    'changeAvatar',
    'sectionFocusRequest'
]);
</script>

<style scoped>
.fade-scale-enter-active, .fade-scale-leave-active {
    transition: opacity .18s ease, transform .18s ease;
}
.fade-scale-enter-from, .fade-scale-leave-to {
    opacity: 0;
    transform: translateY(6px) scale(.98);
}
@media (prefers-reduced-motion: reduce) {
    .fade-scale-enter-active, .fade-scale-leave-active {
        transition: none;
    }
    .fade-scale-enter-from, .fade-scale-leave-to {
        transform: none;
    }
}
</style>