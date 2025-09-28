<template>
  <div class="space-y-6" role="group" aria-labelledby="account-heading">
    <!-- 账户信息卡片 -->
    <UCard :ui="cardHeaderUi">
      <template #header>
        <div class="flex items-center gap-4 md:gap-5">
          <div class="p-4 md:p-5 rounded-xl bg-primary-100 dark:bg-primary-900/40 flex-shrink-0">
            <UIcon name="i-heroicons-user-circle" class="w-7 h-7 md:w-9 md:h-9 text-primary-600 dark:text-primary-400" />
          </div>
          <div class="min-w-0 flex-1">
            <h2 
              id="account-heading" 
              tabindex="-1" 
              class="text-lg md:text-xl lg:text-2xl font-bold text-neutral-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500 rounded-sm"
            >
              账户信息
            </h2>
            <p class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400">
              管理您的个人资料和账户设置
            </p>
          </div>
        </div>
      </template>

      <!-- 用户头像区域 -->
      <div class="flex flex-col sm:flex-row items-center gap-4 md:gap-6 p-4 md:p-6 bg-white/70 dark:bg-neutral-800/50 rounded-lg backdrop-blur-sm">
        <div class="relative group flex-shrink-0">
          <UAvatar 
            :src="user?.avatar_url" 
            :alt="user?.nickname || '用户头像'" 
            :size="avatarSize" 
            class="ring-2 md:ring-4 ring-white dark:ring-neutral-700 shadow-lg transition-opacity duration-200" 
          />
          <UButton 
            @click="handleChangeAvatar" 
            icon="i-heroicons-camera" 
            size="xs" 
            color="primary" 
            class="absolute -bottom-1 -right-1 md:-bottom-2 md:-right-2 rounded-full shadow-lg transition-colors duration-200" 
            square 
            :aria-label="$t?.('settings.changeAvatar') || '更换头像'"
          />
        </div>
        
        <div class="text-center sm:text-left space-y-2 md:space-y-3 flex-1 min-w-0">
          <div>
            <!-- 加载状态 -->
            <template v-if="!user">
              <USkeleton class="h-6 md:h-7 w-32 mx-auto sm:mx-0 mb-2" />
              <USkeleton class="h-4 w-24 mx-auto sm:mx-0" />
            </template>
            <!-- 正常状态 -->
            <template v-else>
              <h3 class="text-lg md:text-xl lg:text-2xl font-bold text-neutral-900 dark:text-white truncate">
                {{ displayNickname }}
              </h3>
              <p class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400 flex items-center gap-2 justify-center sm:justify-start mt-1">
                <UIcon name="i-heroicons-identification" class="w-3 h-3 md:w-4 md:h-4 flex-shrink-0" />
                <span class="truncate">{{ displayLoginId }}</span>
              </p>
            </template>
          </div>
          
          <div class="flex flex-col sm:flex-row gap-2 md:gap-3">
            <!-- 加载状态 -->
            <template v-if="!user">
              <USkeleton class="h-8 w-24" />
              <USkeleton class="h-8 w-24" />
            </template>
            <!-- 正常状态 -->
            <template v-else>
              <UButton 
                variant="soft" 
                size="sm" 
                @click="handleChangeAvatar" 
                :loading="isChangingAvatar"
                :disabled="isChangingAvatar"
                class="transition-colors duration-200"
              >
                <template #leading>
                  <UIcon name="i-heroicons-photo" class="w-3 h-3 md:w-4 md:h-4" />
                </template>
                更换头像
              </UButton>
              <UButton 
                variant="outline" 
                size="sm" 
                @click="handleEditNickname" 
                :loading="isEditingNickname"
                :disabled="isEditingNickname"
                class="transition-colors duration-200"
              >
                <template #leading>
                  <UIcon name="i-heroicons-pencil" class="w-3 h-3 md:w-4 md:h-4" />
                </template>
                编辑昵称
              </UButton>
            </template>
          </div>
        </div>
      </div>
    </UCard>

    <!-- 基本信息 -->
    <UCard :ui="cardDefaultUi">
      <template #header>
        <div class="flex items-center gap-4 md:gap-5">
          <div class="p-3 md:p-4 rounded-lg bg-neutral-100 dark:bg-neutral-800 flex-shrink-0">
            <UIcon name="i-heroicons-document-text" class="w-5 h-5 md:w-6 md:h-6 text-neutral-600 dark:text-neutral-400" />
          </div>
          <h3 class="text-base md:text-lg font-semibold text-neutral-900 dark:text-white">
            基本信息
          </h3>
        </div>
      </template>
      
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-6 p-1 pb-4 md:pb-6">
        <UFormField label="昵称" class="space-y-2">
          <div class="flex gap-3">
            <UInput 
              :value="displayNickname" 
              readonly 
              class="flex-1" 
              placeholder="未设置昵称" 
            />
            <UButton 
              variant="soft" 
              size="sm" 
              @click="handleEditNickname" 
              class="transition-colors duration-200"
            >
              编辑
            </UButton>
          </div>
        </UFormField>
        
        <UFormField label="账户类型" class="space-y-2">
          <div class="flex items-center gap-2 md:gap-3">
            <UBadge 
              :label="accountTypeLabel" 
              :color="accountTypeColor" 
              variant="soft" 
              size="sm" 
              class="flex items-center gap-1"
            >
              <template #leading>
                <UIcon :name="accountTypeIcon" class="w-3 h-3 md:w-4 md:h-4" />
              </template>
            </UBadge>
            <UButton 
              v-if="isTemporaryUser" 
              variant="soft" 
              color="primary" 
              size="xs" 
              @click="handleUpgradeAccount" 
              class="transition-colors duration-200 text-xs"
            >
              升级账户
            </UButton>
          </div>
        </UFormField>
      </div>
    </UCard>

    <!-- 账户安全 -->
    <UCard v-if="!isTemporaryUser" :ui="cardDefaultUi">
      <template #header>
        <div class="flex items-center gap-4 md:gap-5">
          <div class="p-3 md:p-4 rounded-lg bg-green-100 dark:bg-green-900/30 flex-shrink-0">
            <UIcon name="i-heroicons-shield-check" class="w-5 h-5 md:w-6 md:h-6 text-green-600 dark:text-green-400" />
          </div>
          <h3 class="text-base md:text-lg font-semibold text-neutral-900 dark:text-white">
            账户安全
          </h3>
        </div>
      </template>
      
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3 md:gap-4 p-1 pb-4 md:pb-6">
        <!-- 登录密码 -->
        <UCard :ui="securityItemUi">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-3 md:gap-4 min-w-0 flex-1">
              <div class="p-3 md:p-4 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex-shrink-0">
                <UIcon name="i-heroicons-key" class="w-5 h-5 md:w-6 md:h-6 text-blue-600 dark:text-blue-400" />
              </div>
              <div class="min-w-0">
                <div class="font-medium text-sm md:text-base text-neutral-900 dark:text-white">
                  登录密码
                </div>
                <div class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400">
                  上次修改：30天前
                </div>
              </div>
            </div>
            <UButton 
              variant="soft" 
              size="xs" 
              @click="handleChangePassword" 
              class="transition-colors duration-200 flex-shrink-0"
            >
              修改
            </UButton>
          </div>
        </UCard>
        
        <!-- 双重认证 -->
        <UCard :ui="securityItemUi">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-3 md:gap-4 min-w-0 flex-1">
              <div class="p-3 md:p-4 rounded-lg bg-orange-100 dark:bg-orange-900/30 flex-shrink-0">
                <UIcon name="i-heroicons-device-phone-mobile" class="w-5 h-5 md:w-6 md:h-6 text-orange-600 dark:text-orange-400" />
              </div>
              <div class="min-w-0">
                <div class="font-medium text-sm md:text-base text-neutral-900 dark:text-white">
                  双重认证
                </div>
                <div class="text-xs md:text-sm text-neutral-600 dark:text-neutral-400">
                  未启用
                </div>
              </div>
            </div>
            <UButton 
              variant="soft" 
              size="xs" 
              @click="handleSetup2FA" 
              class="transition-colors duration-200 flex-shrink-0"
            >
              设置
            </UButton>
          </div>
        </UCard>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import type { UserSettings } from '~/types/api';
import { isTempUser } from '~/utils/userTypes';

interface Props {
  user: any;
  localSettings: UserSettings;
  isChangingAvatar?: boolean;
  isEditingNickname?: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  changeAvatar: [];
  editNickname: [];
  upgradeAccount: [];
  changePassword: [];
  setup2FA: [];
}>();

// 计算属性
const isTemporaryUser = computed(() => props.user?.user_id ? isTempUser(props.user.user_id) : false);

const displayNickname = computed(() => props.user?.nickname || '未设置昵称');

const displayLoginId = computed(() => props.user?.login_id || '临时用户');

// 使用CSS响应式设计而非JavaScript检测
const avatarSize = 'xl'; // 在移动端通过CSS类控制大小

const accountTypeLabel = computed(() => 
  isTemporaryUser.value ? '临时用户' : '正式用户'
);

const accountTypeColor = computed(() => 
  isTemporaryUser.value ? 'warning' : 'success'
);

const accountTypeIcon = computed(() => 
  isTemporaryUser.value ? 'i-heroicons-clock' : 'i-heroicons-check-badge'
);

// 事件处理
const handleChangeAvatar = () => emit('changeAvatar');
const handleEditNickname = () => emit('editNickname');
const handleUpgradeAccount = () => emit('upgradeAccount');
const handleChangePassword = () => emit('changePassword');
const handleSetup2FA = () => emit('setup2FA');

// UI 配置
const cardHeaderUi = {
  root: 'p-4 md:p-6 rounded-xl border border-primary-200/70 dark:border-primary-800/50 bg-gradient-to-br from-primary-50 to-primary-100 dark:from-primary-900/20 dark:to-primary-800/20',
  body: 'p-0'
};

const cardDefaultUi = {
  root: 'p-4 md:p-6 rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-800',
  body: 'p-0'
};

const securityItemUi = {
  root: 'p-3 md:p-4 rounded-lg bg-neutral-50 dark:bg-neutral-900 border border-neutral-200 dark:border-neutral-700',
  body: 'p-0'
};
</script>