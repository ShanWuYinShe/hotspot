<template>
  <UCard
        variant="outline"
    class="h-full flex flex-col group"
    @click="$emit('click', user)"
  >
    <template #header>
      <div class="flex items-start justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <UAvatar
            :src="user.avatar_url"
            :alt="user.nickname"
            size="lg"
          />
          <div class="min-w-0 flex-1">
                        <h3
                            class="font-semibold text-neutral-900 dark:text-white truncate"
                        >
              {{ user.nickname }}
            </h3>
                        <p
                            v-if="user.bio"
                            class="text-sm text-neutral-500 dark:text-neutral-400 truncate mt-0.5"
                        >
              {{ user.bio }}
            </p>
                        <div
                            v-else
                            class="text-sm text-neutral-500 dark:text-neutral-400 italic mt-0.5"
                        >
              暂无简介
            </div>
          </div>
        </div>

        <div class="flex gap-1 shrink-0">
                    <UBadge
                        v-if="isFriend"
                        color="primary"
                        variant="subtle"
                        size="xs"
                        >好友</UBadge
                    >
                    <UBadge
                        v-else-if="isFollowing"
                        color="neutral"
                        variant="subtle"
                        size="xs"
                        >已关注</UBadge
                    >
                    <UBadge
                        v-if="userTypeBadge"
                        :color="userTypeBadge.color"
                        variant="subtle"
                        size="xs"
                    >
            {{ userTypeBadge.label }}
          </UBadge>
        </div>
      </div>
    </template>

    <div class="flex-1 flex flex-col justify-end gap-2">
      <div v-if="user.location" class="flex flex-wrap gap-2">
        <UBadge
          v-if="user.location.location_name"
          :label="user.location.location_name"
          icon="i-lucide-map-pin"
          color="neutral"
          variant="soft"
          size="xs"
          class="max-w-full truncate"
        />
        <UBadge
          v-if="user.location.distance !== undefined"
          :label="formatDistance(user.location.distance)"
          icon="i-lucide-navigation"
          color="neutral"
          variant="soft"
          size="xs"
        />
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-between gap-2" @click.stop>
        <UButton
          v-model="isFollowing"
          :variant="isFollowing ? 'ghost' : 'soft'"
          :color="isFollowing ? 'neutral' : 'primary'"
          size="xs"
          block
          :icon="isFollowing ? 'i-lucide-user-check' : 'i-lucide-user-plus'"
          :loading="followLoading"
          @click.stop="toggleFollow"
        >
                    {{ isFollowing ? "取消关注" : "关注" }}
        </UButton>

        <div class="flex gap-1">
          <UTooltip text="私信" :portal="false">
            <UButton
              variant="ghost"
              color="neutral"
              size="xs"
              icon="i-lucide-message-circle"
              @click="handleSendMessage"
            />
          </UTooltip>
          <UDropdownMenu :items="moreActions">
            <UButton
              variant="ghost"
              color="neutral"
              size="xs"
              icon="i-lucide-more-vertical"
            />
          </UDropdownMenu>
        </div>
      </div>
    </template>
  </UCard>
</template>

<script setup lang="ts">
import type { RelativeLocation, UserRelation, UserType as ApiUserType } from "~/types/api";
import { useUsersService } from "~/services/users.service";
import type { DropdownMenuItem } from "@nuxt/ui";
import { UserType } from "~/config/constants";

interface UserCardData {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  user_type: ApiUserType;
  bio?: string;
  tags?: string[];
  is_online: boolean | null;
  location?: RelativeLocation | null;
  relation?: UserRelation;
}

interface Props {
    user: UserCardData;
}
interface Emits {
    click: [user: UserCardData];
    message: [user: UserCardData];
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const { formatDistance } = useFormatters();
const { requireAuth } = useRequireAuth();
const userService = useUsersService();
const feedback = useFeedbackStore();
const modal = useModal();

const isOnline = computed(() => props.user.is_online ?? false);
const isFriend = computed(
    () => props.user.relation?.is_mutual_follow ?? false,
);
const isFollowing = ref(props.user.relation?.is_following ?? false);
watch(
    () => props.user.relation?.is_following,
    (val) => {
        isFollowing.value = val ?? false;
    },
);

const userTypeBadge = computed(() => {
    const t = props.user.user_type;
    if (!t || t === UserType.Normal) return null;
    if (isTempUser(t)) return { label: "临时", color: "warning" as const };
    return { label: "访客", color: "neutral" as const };
});

const followLoading = ref(false);
async function toggleFollow() {
    if (!requireAuth(isFollowing.value ? "取消关注" : "关注用户")) return;

    followLoading.value = true;
    try {
        if (isFollowing.value) {
            await userService.unfollow({ user_id: props.user.user_id });
            feedback.success("已取消关注");
            isFollowing.value = false;
        } else {
            await userService.follow({ user_id: props.user.user_id });
            feedback.success("关注成功");
            isFollowing.value = true;
        }
    } catch (e) {
        feedback.error("操作失败");
    } finally {
        followLoading.value = false;
    }
}

const handleSendMessage = () => {
    if (!requireAuth("发送消息")) return;
    emit("message", props.user);
};

const moreActions = computed((): DropdownMenuItem[][] => [
    [
  { 
            label: "举报用户",
            icon: "i-lucide-flag",
            color: "error",
    onSelect: async () => {
                if (!requireAuth("举报")) return;
                await modal.showReport("user", String(props.user.user_id));
                feedback.success("举报已提交");
            },
        },
    ],
]);
</script>
