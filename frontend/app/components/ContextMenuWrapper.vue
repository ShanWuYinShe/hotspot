<template>
  <!-- 右键菜单包装器 - 使用 UContextMenu -->
  <UContextMenu :items="contextMenuItems">
    <slot />
  </UContextMenu>
</template>

<script setup lang="ts">
import type { ContextMenuItem } from '@nuxt/ui'

// Props
interface Props {
  type: 'user' | 'group' | 'moment' | 'message' | 'general'
  data?: any
  customItems?: ContextMenuItem[][]
}

const props = withDefaults(defineProps<Props>(), {
  type: 'general'
})

// Emits
const emit = defineEmits<{
  action: [action: string, data?: any]
}>()

const { t } = useI18n()
const toast = useSmartToast()
const { copyToClipboard } = useUXEnhancements()

// 根据类型生成菜单项
const contextMenuItems = computed((): ContextMenuItem[][] => {
  if (props.customItems) {
    return props.customItems
  }

  switch (props.type) {
    case 'user':
      return getUserMenuItems()
    case 'group':
      return getGroupMenuItems()
    case 'moment':
      return getMomentMenuItems()
    case 'message':
      return getMessageMenuItems()
    default:
      return getGeneralMenuItems()
  }
})

// 用户相关菜单
const getUserMenuItems = (): ContextMenuItem[][] => [
  [
    {
      label: '查看资料',
      icon: 'i-heroicons-user',
      click: () => handleAction('viewProfile')
    },
    {
      label: '发送消息',
      icon: 'i-heroicons-chat-bubble-left-right',
      click: () => handleAction('sendMessage')
    },
    {
      label: '添加好友',
      icon: 'i-heroicons-user-plus',
      click: () => handleAction('addFriend')
    }
  ],
  [
    {
      label: '复制用户ID',
      icon: 'i-heroicons-clipboard',
      click: () => handleCopyUserId()
    },
    {
      label: '分享用户',
      icon: 'i-heroicons-share',
      click: () => handleAction('shareUser')
    }
  ],
  [
    {
      label: '举报用户',
      icon: 'i-heroicons-flag',
      click: () => handleAction('reportUser'),
      color: 'error'
    }
  ]
]

// 群组相关菜单
const getGroupMenuItems = (): ContextMenuItem[][] => [
  [
    {
      label: '查看群组',
      icon: 'i-heroicons-eye',
      click: () => handleAction('viewGroup')
    },
    {
      label: '加入群组',
      icon: 'i-heroicons-plus',
      click: () => handleAction('joinGroup')
    }
  ],
  [
    {
      label: '复制群组链接',
      icon: 'i-heroicons-link',
      click: () => handleCopyGroupLink()
    },
    {
      label: '分享群组',
      icon: 'i-heroicons-share',
      click: () => handleAction('shareGroup')
    }
  ],
  [
    {
      label: '举报群组',
      icon: 'i-heroicons-flag',
      click: () => handleAction('reportGroup'),
      color: 'error'
    }
  ]
]

// 动态相关菜单
const getMomentMenuItems = (): ContextMenuItem[][] => {
  const isOwner = props.data?.user_id === useAuthStore().user?.user_id
  
  const baseItems: ContextMenuItem[][] = [
    [
      {
        label: '查看详情',
        icon: 'i-heroicons-eye',
        click: () => handleAction('viewMoment')
      },
      {
        label: '点赞',
        icon: 'i-heroicons-heart',
        click: () => handleAction('likeMoment')
      },
      {
        label: '评论',
        icon: 'i-heroicons-chat-bubble-left',
        click: () => handleAction('commentMoment')
      }
    ],
    [
      {
        label: '复制链接',
        icon: 'i-heroicons-link',
        click: () => handleCopyMomentLink()
      },
      {
        label: '分享动态',
        icon: 'i-heroicons-share',
        click: () => handleAction('shareMoment')
      }
    ]
  ]

  if (isOwner) {
    baseItems.push([
      {
        label: '编辑动态',
        icon: 'i-heroicons-pencil',
        click: () => handleAction('editMoment')
      },
      {
        label: '删除动态',
        icon: 'i-heroicons-trash',
        click: () => handleAction('deleteMoment'),
        color: 'error'
      }
    ])
  } else {
    baseItems.push([
      {
        label: '举报动态',
        icon: 'i-heroicons-flag',
        click: () => handleAction('reportMoment'),
        color: 'error'
      }
    ])
  }

  return baseItems
}

// 消息相关菜单
const getMessageMenuItems = (): ContextMenuItem[][] => {
  const isOwner = props.data?.sender_id === useAuthStore().user?.user_id
  
  const baseItems: ContextMenuItem[][] = [
    [
      {
        label: '回复消息',
        icon: 'i-heroicons-arrow-uturn-left',
        click: () => handleAction('replyMessage')
      },
      {
        label: '复制消息',
        icon: 'i-heroicons-clipboard',
        click: () => handleCopyMessage()
      }
    ],
    [
      {
        label: '转发消息',
        icon: 'i-heroicons-arrow-right',
        click: () => handleAction('forwardMessage')
      }
    ]
  ]

  if (isOwner) {
    baseItems.push([
      {
        label: '编辑消息',
        icon: 'i-heroicons-pencil',
        click: () => handleAction('editMessage')
      },
      {
        label: '撤回消息',
        icon: 'i-heroicons-arrow-uturn-down',
        click: () => handleAction('recallMessage')
      },
      {
        label: '删除消息',
        icon: 'i-heroicons-trash',
        click: () => handleAction('deleteMessage'),
        color: 'error'
      }
    ])
  } else {
    baseItems.push([
      {
        label: '举报消息',
        icon: 'i-heroicons-flag',
        click: () => handleAction('reportMessage'),
        color: 'error'
      }
    ])
  }

  return baseItems
}

// 通用菜单
const getGeneralMenuItems = (): ContextMenuItem[][] => [
  [
    {
      label: '刷新页面',
      icon: 'i-heroicons-arrow-path',
      click: () => window.location.reload()
    },
    {
      label: '返回上一页',
      icon: 'i-heroicons-arrow-left',
      click: () => window.history.back()
    }
  ],
  [
    {
      label: '复制页面链接',
      icon: 'i-heroicons-link',
      click: () => handleCopyPageLink()
    },
    {
      label: '分享页面',
      icon: 'i-heroicons-share',
      click: () => handleAction('sharePage')
    }
  ]
]

// 处理菜单动作
const handleAction = (action: string) => {
  emit('action', action, props.data)
}

// 复制用户ID
const handleCopyUserId = async () => {
  if (props.data?.user_id) {
    const success = await copyToClipboard(props.data.user_id.toString())
    if (success) {
      toast.success({
        title: '已复制',
        description: '用户ID已复制到剪贴板'
      })
    }
  }
}

// 复制群组链接
const handleCopyGroupLink = async () => {
  if (props.data?.group_id) {
    const link = `${window.location.origin}/groups/${props.data.group_id}`
    const success = await copyToClipboard(link)
    if (success) {
      toast.success({
        title: '已复制',
        description: '群组链接已复制到剪贴板'
      })
    }
  }
}

// 复制动态链接
const handleCopyMomentLink = async () => {
  if (props.data?.moment_id) {
    const link = `${window.location.origin}/moments/${props.data.moment_id}`
    const success = await copyToClipboard(link)
    if (success) {
      toast.success({
        title: '已复制',
        description: '动态链接已复制到剪贴板'
      })
    }
  }
}

// 复制消息内容
const handleCopyMessage = async () => {
  if (props.data?.content) {
    const success = await copyToClipboard(props.data.content)
    if (success) {
      toast.success({
        title: '已复制',
        description: '消息内容已复制到剪贴板'
      })
    }
  }
}

// 复制页面链接
const handleCopyPageLink = async () => {
  const success = await copyToClipboard(window.location.href)
  if (success) {
    toast.success({
      title: '已复制',
      description: '页面链接已复制到剪贴板'
    })
  }
}
</script>
