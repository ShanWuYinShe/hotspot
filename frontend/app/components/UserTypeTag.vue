<template>
  <UBadge
    v-if="shouldShowTag"
    :color="tagStyle.color"
    size="xs"
    class="ml-1"
  >
    {{ tagStyle.text }}
  </UBadge>
</template>

<script setup lang="ts">
import { getUserTypeTagStyle, isTempUser, isSystemUser } from '~/utils/userTypes'

interface Props {
  userId: string
  showForAllTypes?: boolean // 是否为所有类型显示标签，默认只为临时用户和系统用户显示
}

const props = withDefaults(defineProps<Props>(), {
  showForAllTypes: false
})

// 计算标签样式
const tagStyle = computed(() => {
  const style = getUserTypeTagStyle(props.userId)
  
  // 根据用户类型返回对应的 UBadge color
  if (isTempUser(props.userId)) {
    return {
      text: style.text,
      color: 'warning' as const
    }
  } else if (isSystemUser(props.userId)) {
    return {
      text: style.text,
      color: 'primary' as const
    }
  } else {
    return {
      text: '普通',
      color: 'neutral' as const
    }
  }
})

// 是否应该显示标签
const shouldShowTag = computed(() => {
  if (props.showForAllTypes) {
    return true
  }
  
  // 默认只为临时用户和系统用户显示标签
  return isTempUser(props.userId) || isSystemUser(props.userId)
})
</script>
