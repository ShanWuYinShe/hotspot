<template>
    <div class="p-6" :aria-labelledby="'tab-' + activeSection">
        <UAccordion 
            :items="accordionItems" 
            :default-value="activeSection"
            type="single"
            :collapsible="false"
            @update:model-value="handleSectionChange"
        >
            <template #account-body>
                <AccountSection
                    :user="user"
                    :local-settings="localSettings"
                    @change-avatar="emit('changeAvatar')"
                    @edit-nickname="emit('editNickname')"
                    @upgrade-account="emit('upgradeAccount')"
                    @change-password="emit('changePassword')"
                    @setup-2fa="emit('setup2FA')"
                />
            </template>
            
            <template #privacy-body>
                <PrivacySection
                    :local-settings="localSettings"
                />
            </template>
            
            <template #notifications-body>
                <NotificationsSection
                    :local-settings="localSettings"
                />
            </template>
            
            <template #appearance-body>
                <AppearanceSection
                    :local-settings="localSettings"
                />
            </template>
            
            <template #about-body>
                <AboutSection
                    @clear-cache="emit('clearCache')"
                    @show-help="emit('showHelp')"
                    @show-privacy-policy="emit('showPrivacyPolicy')"
                    @show-terms="emit('showTerms')"
                />
            </template>
            
            <template #danger-body>
                <DangerSection
                    @delete-account="emit('deleteAccount')"
                />
            </template>
        </UAccordion>
    </div>
</template>

<script setup lang="ts">
import type { UserSettings } from '~/types/api';
import type { AccordionItem } from '@nuxt/ui';

interface Props {
    activeSection: string;
    localSettings: UserSettings;
    user: any;
}

const props = defineProps<Props>();

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
    'sectionFocusRequest',
    'sectionChange'
]);

const accordionItems = computed<AccordionItem[]>(() => [
    {
        label: '账户设置',
        icon: 'i-heroicons-user-circle',
        value: 'account',
        slot: 'account'
    },
    {
        label: '隐私设置',
        icon: 'i-heroicons-shield-check',
        value: 'privacy',
        slot: 'privacy'
    },
    {
        label: '通知设置',
        icon: 'i-heroicons-bell',
        value: 'notifications',
        slot: 'notifications'
    },
    {
        label: '外观设置',
        icon: 'i-heroicons-paint-brush',
        value: 'appearance',
        slot: 'appearance'
    },
    {
        label: '关于应用',
        icon: 'i-heroicons-information-circle',
        value: 'about',
        slot: 'about'
    },
    {
        label: '危险操作',
        icon: 'i-heroicons-exclamation-triangle',
        value: 'danger',
        slot: 'danger'
    }
]);

const handleSectionChange = (value: string | string[] | undefined) => {
    if (typeof value === 'string') {
        emit('sectionChange', value);
    }
};
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