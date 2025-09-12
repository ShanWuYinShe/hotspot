/**
 * 交互设计指南
 * 
 * 明确定义哪些元素需要交互特效，哪些不需要
 */

// ✅ 需要交互特效的元素类型
export const INTERACTIVE_ELEMENTS = {
  // 按钮 - 用户可以点击执行操作
  buttons: {
    effects: ['hover:brightness-110', 'active:brightness-90', 'focus:ring-2', 'transition-colors'],
    description: '按钮需要明确的视觉反馈，告知用户可以点击'
  },
  
  // 链接 - 用户可以点击跳转
  links: {
    effects: ['hover:text-primary-600', 'hover:underline', 'transition-colors'],
    description: '链接需要悬停状态，表明可点击性'
  },
  
  // 表单控件 - 输入框、选择器等
  formControls: {
    effects: ['focus:ring-2', 'focus:border-primary-500', 'transition-colors'],
    description: '表单控件需要焦点状态，帮助用户理解当前输入位置'
  },
  
  // 可点击的卡片 - 导航到详情页的卡片
  interactiveCards: {
    effects: ['hover:bg-neutral-50', 'hover:border-primary-200', 'cursor-pointer', 'transition-colors'],
    description: '可点击卡片需要悬停反馈，但避免位移效果'
  },
  
  // 菜单项 - 下拉菜单、导航菜单项
  menuItems: {
    effects: ['hover:bg-primary-100', 'focus:bg-primary-100', 'transition-colors'],
    description: '菜单项需要悬停和焦点状态，便于键盘导航'
  },
  
  // 标签页 - 可切换的标签
  tabs: {
    effects: ['hover:bg-neutral-100', 'data-[state=active]:bg-white', 'data-[state=active]:text-primary-600', 'transition-colors'],
    description: '标签页需要激活状态和悬停反馈'
  }
} as const;

// ❌ 不需要交互特效的元素类型
export const NON_INTERACTIVE_ELEMENTS = {
  // 纯展示内容
  displayContent: {
    elements: ['文本', '图片', '标题', '描述'],
    reason: '纯展示内容不需要交互反馈，避免误导用户'
  },
  
  // 容器元素
  containers: {
    elements: ['布局容器', '分组容器', '网格容器'],
    reason: '容器元素是结构性的，不应该有交互暗示'
  },
  
  // 装饰性元素
  decorative: {
    elements: ['分隔线', '背景图案', '装饰图标'],
    reason: '装饰性元素不应该分散用户对真正交互元素的注意力'
  },
  
  // 状态指示器
  statusIndicators: {
    elements: ['加载状态', '错误提示', '成功消息', '进度条'],
    reason: '状态指示器是信息性的，不需要交互特效'
  },
  
  // 头像和徽章
  avatarsAndBadges: {
    elements: ['用户头像', '状态徽章', '标签'],
    reason: '除非可点击（如用户资料），否则不需要交互效果'
  }
} as const;

// 交互特效的层次结构
export const INTERACTION_HIERARCHY = {
  // 主要操作 - 最强的视觉反馈
  primary: {
    effects: ['hover:brightness-110', 'hover:shadow-md', 'active:brightness-90'],
    usage: '主要按钮、重要链接'
  },
  
  // 次要操作 - 中等视觉反馈
  secondary: {
    effects: ['hover:bg-neutral-50', 'hover:text-primary-600'],
    usage: '次要按钮、导航链接'
  },
  
  // 微交互 - 轻微视觉反馈
  subtle: {
    effects: ['hover:bg-neutral-25', 'transition-colors'],
    usage: '菜单项、标签页'
  }
} as const;

// 禁用的特效类型
export const FORBIDDEN_EFFECTS = {
  // 位移特效
  transform: {
    effects: ['hover:scale-*', 'hover:translate-*', 'hover:rotate-*'],
    reason: '位移特效会让用户感到不稳定，影响可用性'
  },
  
  // 过度动画
  animations: {
    effects: ['animate-bounce', 'animate-pulse', 'animate-spin'],
    reason: '动画应该谨慎使用，避免分散注意力'
  },
  
  // 复杂过渡
  complexTransitions: {
    effects: ['transition-all', 'duration-500+'],
    reason: '过渡应该快速且专注，避免影响响应性'
  }
} as const;

// 使用指南
export const USAGE_GUIDELINES = {
  // 何时添加交互特效
  whenToAdd: [
    '元素可以被点击或触摸',
    '元素可以接收键盘焦点',
    '元素的状态会发生变化',
    '需要向用户提供操作反馈'
  ],
  
  // 何时不添加交互特效
  whenNotToAdd: [
    '纯展示性内容',
    '装饰性元素',
    '不可交互的容器',
    '已经有明确交互暗示的元素'
  ],
  
  // 最佳实践
  bestPractices: [
    '使用颜色变化而非位移',
    '保持过渡时间在150-200ms',
    '确保交互反馈是即时的',
    '避免过度设计',
    '考虑无障碍访问需求'
  ]
} as const;
