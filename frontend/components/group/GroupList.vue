<template>
  <div class="group-list">
    <NSkeleton v-if="loading" :rows="3" animated>
      <template #template>
        <div class="group-card-skeleton">
          <el-skeleton-item variant="circle" style="width:48px;height:48px;margin-right:16px;" />
          <div class="group-card-skeleton-info">
            <el-skeleton-item variant="text" style="width:120px;height:18px;margin-bottom:8px;" />
            <el-skeleton-item variant="text" style="width:180px;height:14px;" />
          </div>
        </div>
      </template>
    </NSkeleton>
    <template v-else>
      <div v-if="groups.length === 0" class="group-list-empty">
        <NEmpty description="暂无群组">
          <template #icon>
            <NIcon :size="48" class="empty-icon"><UserFilled /></NIcon>
          </template>
          <template #extra>
            <NButton type="primary" @click="$emit('create-group')">创建群组</NButton>
          </template>
        </NEmpty>
      </div>
      <div v-else>
        <GroupCard
          v-for="group in groups"
          :key="group.group_id"
          :group="group"
          @click="$emit('group-click', group)"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
// Naive UI 组件导入
import { NSkeleton, NEmpty, NIcon, NButton } from 'naive-ui'

import GroupCard from './GroupCard.vue';
import { Person as UserFilled } from '@vicons/ionicons5';
import type { GroupInfo } from '~/types/models/group';

const props = defineProps<{
  groups: GroupInfo[];
  loading?: boolean;
}>();
const emit = defineEmits(['group-click', 'create-group']);
</script>

<style scoped>
.group-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.group-list-empty {
  padding: 32px 0;
}
.group-card-skeleton {
  display: flex;
  align-items: center;
  padding: 16px 0;
}
.group-card-skeleton-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.empty-icon {
  color: #b2becd;
}
</style>
