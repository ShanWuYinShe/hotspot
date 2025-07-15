<template>
  <div class="emoji-picker">
    <NButton
      type="default"
      @click="togglePicker"
      class="emoji-trigger"
      v-tooltip="{ category: 'emoji', key: 'trigger' }"
    >
      <NIcon>
        <ChatbubbleOutline />
      </NIcon>
    </NButton>

    <ClientOnly>
      <NPopover
        v-model:show="visible"
        trigger="manual"
        placement="top-start"
        :show-arrow="false"
        class="emoji-popover"
      >
        <template #trigger>
          <div></div>
        </template>
        
        <div class="emoji-picker-content">
          <!-- 分类标签 -->
          <div class="emoji-categories">
            <div
              v-for="category in categories"
              :key="category.name"
              :class="['category-tab', { active: activeCategory === category.name }]"
              @click="activeCategory = category.name"
            >
              {{ category.icon }}
            </div>
          </div>

          <!-- 表情网格 -->
          <div ref="emojiGridRef" class="emoji-grid">
            <div
              v-for="emoji in currentEmojis"
              :key="emoji.char"
              class="emoji-item"
              @click="selectEmoji(emoji)"
              :title="emoji.name"
            >
              {{ emoji.char }}
            </div>
          </div>
        </div>
      </NPopover>
    </ClientOnly>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { NButton, NIcon, NPopover } from 'naive-ui'
import { ChatbubbleOutline } from '@vicons/ionicons5'

interface Emoji {
  char: string
  name: string
  category: string
}

interface EmojiCategory {
  name: string
  icon: string
  emojis: Emoji[]
}

interface Props {
  modelValue?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [visible: boolean]
  'select': [emoji: Emoji]
}>()

// 状态
const visible = ref(props.modelValue || false)
const activeCategory = ref('smileys')
const popoverRef = ref()
const emojiGridRef = ref()

// 切换选择器显示状态
const togglePicker = () => {
  visible.value = !visible.value
}

// 表情分类
const categories = ref<EmojiCategory[]>([
  {
    name: 'smileys',
    icon: '😀',
    emojis: [
      { char: '😀', name: '开心', category: 'smileys' },
      { char: '😃', name: '大笑', category: 'smileys' },
      { char: '😄', name: '笑', category: 'smileys' },
      { char: '😁', name: '咧嘴笑', category: 'smileys' },
      { char: '😆', name: '眯眼笑', category: 'smileys' },
      { char: '😅', name: '苦笑', category: 'smileys' },
      { char: '🤣', name: '笑哭', category: 'smileys' },
      { char: '😂', name: '哭笑', category: 'smileys' },
      { char: '🙂', name: '微笑', category: 'smileys' },
      { char: '🙃', name: '倒脸', category: 'smileys' },
      { char: '😉', name: '眨眼', category: 'smileys' },
      { char: '😊', name: '羞涩', category: 'smileys' },
      { char: '😇', name: '天使', category: 'smileys' },
      { char: '🥰', name: '爱心眼', category: 'smileys' },
      { char: '😍', name: '花痴', category: 'smileys' },
      { char: '🤩', name: '星眼', category: 'smileys' },
      { char: '😘', name: '飞吻', category: 'smileys' },
      { char: '😗', name: '亲吻', category: 'smileys' },
      { char: '😚', name: '害羞亲吻', category: 'smileys' },
      { char: '😋', name: '美味', category: 'smileys' },
      { char: '😛', name: '吐舌', category: 'smileys' },
      { char: '😜', name: '调皮', category: 'smileys' },
      { char: '🤪', name: '疯狂', category: 'smileys' },
      { char: '😝', name: '鬼脸', category: 'smileys' },
      { char: '🤑', name: '财迷', category: 'smileys' },
      { char: '🤗', name: '拥抱', category: 'smileys' },
      { char: '🤭', name: '捂嘴笑', category: 'smileys' },
      { char: '🤫', name: '嘘', category: 'smileys' },
      { char: '🤔', name: '思考', category: 'smileys' },
      { char: '🤐', name: '闭嘴', category: 'smileys' },
      { char: '🤨', name: '质疑', category: 'smileys' },
      { char: '😐', name: '面无表情', category: 'smileys' },
      { char: '😑', name: '无语', category: 'smileys' },
      { char: '😶', name: '沉默', category: 'smileys' },
      { char: '😏', name: '得意', category: 'smileys' },
      { char: '😒', name: '不爽', category: 'smileys' },
      { char: '🙄', name: '翻白眼', category: 'smileys' },
      { char: '😬', name: '尴尬', category: 'smileys' },
      { char: '🤥', name: '说谎', category: 'smileys' },
      { char: '😔', name: '沉思', category: 'smileys' },
      { char: '😪', name: '困', category: 'smileys' },
      { char: '🤤', name: '流口水', category: 'smileys' },
      { char: '😴', name: '睡觉', category: 'smileys' },
    ]
  },
  {
    name: 'gestures',
    icon: '👍',
    emojis: [
      { char: '👍', name: '点赞', category: 'gestures' },
      { char: '👎', name: '反对', category: 'gestures' },
      { char: '👌', name: 'OK', category: 'gestures' },
      { char: '✌️', name: '胜利', category: 'gestures' },
      { char: '🤞', name: '祈祷', category: 'gestures' },
      { char: '🤟', name: '爱你', category: 'gestures' },
      { char: '🤘', name: '摇滚', category: 'gestures' },
      { char: '🤙', name: '电话', category: 'gestures' },
      { char: '👈', name: '左指', category: 'gestures' },
      { char: '👉', name: '右指', category: 'gestures' },
      { char: '👆', name: '上指', category: 'gestures' },
      { char: '👇', name: '下指', category: 'gestures' },
      { char: '☝️', name: '食指', category: 'gestures' },
      { char: '✋', name: '举手', category: 'gestures' },
      { char: '🤚', name: '手背', category: 'gestures' },
      { char: '🖐️', name: '张开手', category: 'gestures' },
      { char: '🖖', name: '瓦肯礼', category: 'gestures' },
      { char: '👋', name: '挥手', category: 'gestures' },
      { char: '🤏', name: '捏', category: 'gestures' },
      { char: '👏', name: '鼓掌', category: 'gestures' },
      { char: '🙌', name: '举双手', category: 'gestures' },
      { char: '👐', name: '张开双手', category: 'gestures' },
      { char: '🤲', name: '祈求', category: 'gestures' },
      { char: '🙏', name: '合十', category: 'gestures' },
    ]
  },
  {
    name: 'hearts',
    icon: '❤️',
    emojis: [
      { char: '❤️', name: '红心', category: 'hearts' },
      { char: '🧡', name: '橙心', category: 'hearts' },
      { char: '💛', name: '黄心', category: 'hearts' },
      { char: '💚', name: '绿心', category: 'hearts' },
      { char: '💙', name: '蓝心', category: 'hearts' },
      { char: '💜', name: '紫心', category: 'hearts' },
      { char: '🖤', name: '黑心', category: 'hearts' },
      { char: '🤍', name: '白心', category: 'hearts' },
      { char: '🤎', name: '棕心', category: 'hearts' },
      { char: '💔', name: '破碎的心', category: 'hearts' },
      { char: '❣️', name: '心叹号', category: 'hearts' },
      { char: '💕', name: '两颗心', category: 'hearts' },
      { char: '💞', name: '旋转的心', category: 'hearts' },
      { char: '💓', name: '跳动的心', category: 'hearts' },
      { char: '💗', name: '成长的心', category: 'hearts' },
      { char: '💖', name: '闪闪的心', category: 'hearts' },
      { char: '💘', name: '丘比特之箭', category: 'hearts' },
      { char: '💝', name: '心形礼物', category: 'hearts' },
    ]
  },
  {
    name: 'objects',
    icon: '🎉',
    emojis: [
      { char: '🎉', name: '庆祝', category: 'objects' },
      { char: '🎊', name: '彩球', category: 'objects' },
      { char: '🎈', name: '气球', category: 'objects' },
      { char: '🎁', name: '礼物', category: 'objects' },
      { char: '🎂', name: '生日蛋糕', category: 'objects' },
      { char: '🍰', name: '蛋糕', category: 'objects' },
      { char: '🧁', name: '纸杯蛋糕', category: 'objects' },
      { char: '🍭', name: '棒棒糖', category: 'objects' },
      { char: '🍬', name: '糖果', category: 'objects' },
      { char: '🍫', name: '巧克力', category: 'objects' },
      { char: '🍿', name: '爆米花', category: 'objects' },
      { char: '🍩', name: '甜甜圈', category: 'objects' },
      { char: '🍪', name: '饼干', category: 'objects' },
      { char: '☕', name: '咖啡', category: 'objects' },
      { char: '🍵', name: '茶', category: 'objects' },
      { char: '🥤', name: '饮料', category: 'objects' },
      { char: '🍺', name: '啤酒', category: 'objects' },
      { char: '🍻', name: '干杯', category: 'objects' },
      { char: '🥂', name: '香槟', category: 'objects' },
      { char: '🍷', name: '红酒', category: 'objects' },
      { char: '🎵', name: '音符', category: 'objects' },
      { char: '🎶', name: '音乐', category: 'objects' },
      { char: '🎸', name: '吉他', category: 'objects' },
      { char: '📱', name: '手机', category: 'objects' },
      { char: '💻', name: '电脑', category: 'objects' },
      { char: '📷', name: '相机', category: 'objects' },
      { char: '📸', name: '闪光相机', category: 'objects' },
      { char: '🎮', name: '游戏', category: 'objects' },
    ]
  }
])

// 最近使用的表情
const recentEmojis = ref<Emoji[]>([])

// 计算属性
const currentEmojis = computed(() => {
  const category = categories.value.find(cat => cat.name === activeCategory.value)
  return category?.emojis || []
})

// 监听 modelValue 变化
watch(() => props.modelValue, (newValue) => {
  visible.value = newValue || false
})

// 监听 visible 变化
watch(visible, (newValue) => {
  emit('update:modelValue', newValue)
})

// 选择表情
const selectEmoji = (emoji: Emoji) => {
  emit('select', emoji)
  
  // 添加到最近使用
  addToRecent(emoji)
  
  // 隐藏选择器
  visible.value = false
}

// 添加到最近使用
const addToRecent = (emoji: Emoji) => {
  // 移除已存在的
  const index = recentEmojis.value.findIndex(e => e.char === emoji.char)
  if (index > -1) {
    recentEmojis.value.splice(index, 1)
  }
  
  // 添加到开头
  recentEmojis.value.unshift(emoji)
  
  // 限制数量
  if (recentEmojis.value.length > 18) {
    recentEmojis.value = recentEmojis.value.slice(0, 18)
  }
  
  // 保存到本地存储
  saveRecentEmojis()
}

// 保存最近使用到本地存储
const saveRecentEmojis = () => {
  try {
    localStorage.setItem('recent-emojis', JSON.stringify(recentEmojis.value))
  } catch (e) {
    console.warn('Failed to save recent emojis:', e)
  }
}

// 从本地存储加载最近使用
const loadRecentEmojis = () => {
  try {
    const saved = localStorage.getItem('recent-emojis')
    if (saved) {
      recentEmojis.value = JSON.parse(saved)
    }
  } catch (e) {
    console.warn('Failed to load recent emojis:', e)
  }
}

// 组件挂载时加载最近使用
onMounted(() => {
  loadRecentEmojis()
})

// 暴露方法
defineExpose({
  toggle: togglePicker,
  show: () => { visible.value = true },
  hide: () => { visible.value = false }
})
</script>

<style scoped lang="scss">
.emoji-picker {
  display: inline-block;
}

.emoji-picker-content {
  padding: 0.5rem;
  max-height: 300px;
  
  .emoji-categories {
    display: flex;
    gap: 0.25rem;
    margin-bottom: 0.75rem;
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding-bottom: 0.5rem;
    
    .category-btn {
      min-width: auto;
      padding: 0.25rem 0.5rem;
      border-radius: 6px;
    }
  }
  
  .emoji-grid {
    display: grid;
    grid-template-columns: repeat(8, 1fr);
    gap: 0.25rem;
    max-height: 180px;
    overflow-y: auto;
    margin-bottom: 0.75rem;
  }
  
  .emoji-item {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    border-radius: 4px;
    font-size: 1.25rem;
    transition: background-color 0.2s ease;
    
    &:hover {
      background: var(--el-fill-color-light);
    }
    
    &:active {
      transform: scale(0.95);
    }
  }
  
  .recent-section {
    border-top: 1px solid var(--el-border-color-lighter);
    padding-top: 0.5rem;
    
    .section-title {
      font-size: 0.75rem;
      color: var(--el-text-color-secondary);
      margin-bottom: 0.5rem;
    }
    
    .recent-emojis {
      display: grid;
      grid-template-columns: repeat(8, 1fr);
      gap: 0.25rem;
    }
  }
}

/* 滚动条样式 */
.emoji-grid::-webkit-scrollbar {
  width: 4px;
}

.emoji-grid::-webkit-scrollbar-track {
  background: var(--el-fill-color-lighter);
  border-radius: 2px;
}

.emoji-grid::-webkit-scrollbar-thumb {
  background: var(--el-fill-color);
  border-radius: 2px;
}

.emoji-grid::-webkit-scrollbar-thumb:hover {
  background: var(--el-fill-color-dark);
}
</style>

<style lang="scss">
.emoji-picker-popover {
  padding: 0 !important;
  
  .el-popper__arrow {
    display: none;
  }
}
</style>
