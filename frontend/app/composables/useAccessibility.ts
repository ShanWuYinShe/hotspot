/**
 * 无障碍访问 Composable
 * 提供键盘导航、焦点管理、ARIA 支持、屏幕阅读器优化
 */

interface AccessibilityOptions {
  /** 是否启用键盘导航 */
  keyboardNavigation?: boolean
  /** 是否启用焦点陷阱 */
  focusTrap?: boolean
  /** 是否启用屏幕阅读器支持 */
  screenReader?: boolean
  /** 自动焦点目标 */
  autoFocus?: boolean
  /** 焦点恢复 */
  restoreFocus?: boolean
}

interface FocusableElement extends HTMLElement {
  tabIndex: number
}

// 可聚焦元素选择器
const FOCUSABLE_SELECTORS = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
  '[contenteditable="true"]'
].join(',')

export function useAccessibility(
  containerRef: Ref<HTMLElement | undefined>,
  options: AccessibilityOptions = {}
) {
  const {
    keyboardNavigation = true,
    focusTrap = false,
    screenReader = true,
    autoFocus = false,
    restoreFocus = true
  } = options

  const focusedElement = ref<HTMLElement | null>(null)
  const previousFocus = ref<HTMLElement | null>(null)
  const announcements = ref<string[]>([])

  // 获取所有可聚焦元素
  const getFocusableElements = (): FocusableElement[] => {
    if (!containerRef.value) return []
    
    const elements = Array.from(
      containerRef.value.querySelectorAll(FOCUSABLE_SELECTORS)
    ) as FocusableElement[]
    
    return elements.filter(el => {
      const style = window.getComputedStyle(el)
      return style.display !== 'none' && 
             style.visibility !== 'hidden' && 
             !el.hasAttribute('disabled')
    })
  }

  // 焦点管理
  const focusFirst = () => {
    const elements = getFocusableElements()
    if (elements.length > 0) {
      elements[0].focus()
      focusedElement.value = elements[0]
    }
  }

  const focusLast = () => {
    const elements = getFocusableElements()
    if (elements.length > 0) {
      const lastElement = elements[elements.length - 1]
      lastElement.focus()
      focusedElement.value = lastElement
    }
  }

  const focusNext = () => {
    const elements = getFocusableElements()
    const currentIndex = elements.indexOf(focusedElement.value as FocusableElement)
    
    if (currentIndex < elements.length - 1) {
      elements[currentIndex + 1].focus()
      focusedElement.value = elements[currentIndex + 1]
    } else if (focusTrap) {
      focusFirst()
    }
  }

  const focusPrevious = () => {
    const elements = getFocusableElements()
    const currentIndex = elements.indexOf(focusedElement.value as FocusableElement)
    
    if (currentIndex > 0) {
      elements[currentIndex - 1].focus()
      focusedElement.value = elements[currentIndex - 1]
    } else if (focusTrap) {
      focusLast()
    }
  }

  // 键盘事件处理
  const handleKeydown = (event: KeyboardEvent) => {
    if (!keyboardNavigation) return

    switch (event.key) {
      case 'Tab':
        if (focusTrap) {
          event.preventDefault()
          if (event.shiftKey) {
            focusPrevious()
          } else {
            focusNext()
          }
        }
        break
        
      case 'ArrowDown':
      case 'ArrowRight':
        event.preventDefault()
        focusNext()
        break
        
      case 'ArrowUp':
      case 'ArrowLeft':
        event.preventDefault()
        focusPrevious()
        break
        
      case 'Home':
        event.preventDefault()
        focusFirst()
        break
        
      case 'End':
        event.preventDefault()
        focusLast()
        break
        
      case 'Escape':
        if (restoreFocus && previousFocus.value) {
          previousFocus.value.focus()
        }
        break
    }
  }

  // 焦点事件处理
  const handleFocus = (event: FocusEvent) => {
    const target = event.target as HTMLElement
    if (getFocusableElements().includes(target as FocusableElement)) {
      focusedElement.value = target
    }
  }

  // 屏幕阅读器公告
  const announce = (message: string, priority: 'polite' | 'assertive' = 'polite') => {
    if (!screenReader) return
    
    announcements.value.push(message)
    
    // 创建临时的 aria-live 区域
    const liveRegion = document.createElement('div')
    liveRegion.setAttribute('aria-live', priority)
    liveRegion.setAttribute('aria-atomic', 'true')
    liveRegion.className = 'sr-only'
    liveRegion.textContent = message
    
    document.body.appendChild(liveRegion)
    
    // 清理
    setTimeout(() => {
      document.body.removeChild(liveRegion)
    }, 1000)
  }

  // ARIA 属性管理
  const setAriaLabel = (element: HTMLElement, label: string) => {
    element.setAttribute('aria-label', label)
  }

  const setAriaDescribedBy = (element: HTMLElement, id: string) => {
    element.setAttribute('aria-describedby', id)
  }

  const setAriaExpanded = (element: HTMLElement, expanded: boolean) => {
    element.setAttribute('aria-expanded', expanded.toString())
  }

  const setAriaSelected = (element: HTMLElement, selected: boolean) => {
    element.setAttribute('aria-selected', selected.toString())
  }

  const setAriaPressed = (element: HTMLElement, pressed: boolean) => {
    element.setAttribute('aria-pressed', pressed.toString())
  }

  // 跳过链接生成
  const createSkipLink = (targetId: string, text: string = '跳到主内容') => {
    const skipLink = document.createElement('a')
    skipLink.href = `#${targetId}`
    skipLink.textContent = text
    skipLink.className = 'sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:px-4 focus:py-2 focus:bg-primary-600 focus:text-white focus:rounded'
    
    skipLink.addEventListener('click', (e) => {
      e.preventDefault()
      const target = document.getElementById(targetId)
      if (target) {
        target.focus()
        target.scrollIntoView({ behavior: 'smooth' })
      }
    })
    
    return skipLink
  }

  // 高对比度模式检测
  const isHighContrast = ref(false)
  const detectHighContrast = () => {
    if (typeof window !== 'undefined') {
      isHighContrast.value = window.matchMedia('(prefers-contrast: high)').matches
    }
  }

  // 减少动画偏好检测
  const prefersReducedMotion = ref(false)
  const detectReducedMotion = () => {
    if (typeof window !== 'undefined') {
      prefersReducedMotion.value = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    }
  }

  // 初始化
  onMounted(() => {
    if (!containerRef.value) return
    
    // 保存之前的焦点
    previousFocus.value = document.activeElement as HTMLElement
    
    // 自动聚焦
    if (autoFocus) {
      nextTick(() => {
        focusFirst()
      })
    }
    
    // 添加事件监听器
    containerRef.value.addEventListener('keydown', handleKeydown)
    containerRef.value.addEventListener('focus', handleFocus, true)
    
    // 检测用户偏好
    detectHighContrast()
    detectReducedMotion()
    
    // 监听偏好变化
    if (typeof window !== 'undefined') {
      const contrastQuery = window.matchMedia('(prefers-contrast: high)')
      const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
      
      contrastQuery.addEventListener('change', detectHighContrast)
      motionQuery.addEventListener('change', detectReducedMotion)
      
      // 清理函数
      onUnmounted(() => {
        contrastQuery.removeEventListener('change', detectHighContrast)
        motionQuery.removeEventListener('change', detectReducedMotion)
      })
    }
  })

  onUnmounted(() => {
    if (containerRef.value) {
      containerRef.value.removeEventListener('keydown', handleKeydown)
      containerRef.value.removeEventListener('focus', handleFocus, true)
    }
    
    // 恢复焦点
    if (restoreFocus && previousFocus.value) {
      previousFocus.value.focus()
    }
  })

  return {
    // 焦点管理
    focusedElement: readonly(focusedElement),
    focusFirst,
    focusLast,
    focusNext,
    focusPrevious,
    
    // 屏幕阅读器
    announce,
    announcements: readonly(announcements),
    
    // ARIA 属性
    setAriaLabel,
    setAriaDescribedBy,
    setAriaExpanded,
    setAriaSelected,
    setAriaPressed,
    
    // 工具函数
    createSkipLink,
    getFocusableElements,
    
    // 用户偏好
    isHighContrast: readonly(isHighContrast),
    prefersReducedMotion: readonly(prefersReducedMotion)
  }
}

/**
 * 无障碍访问指令
 */
export const vAccessible = {
  mounted(el: HTMLElement, binding: any) {
    const { value } = binding
    
    if (value?.label) {
      el.setAttribute('aria-label', value.label)
    }
    
    if (value?.describedBy) {
      el.setAttribute('aria-describedby', value.describedBy)
    }
    
    if (value?.role) {
      el.setAttribute('role', value.role)
    }
    
    if (value?.live) {
      el.setAttribute('aria-live', value.live)
    }
  },
  
  updated(el: HTMLElement, binding: any) {
    const { value } = binding
    
    if (value?.expanded !== undefined) {
      el.setAttribute('aria-expanded', value.expanded.toString())
    }
    
    if (value?.selected !== undefined) {
      el.setAttribute('aria-selected', value.selected.toString())
    }
    
    if (value?.pressed !== undefined) {
      el.setAttribute('aria-pressed', value.pressed.toString())
    }
  }
}

/**
 * 屏幕阅读器专用组件
 */
export const ScreenReaderOnly = defineComponent({
  name: 'ScreenReaderOnly',
  setup(_, { slots }) {
    return () => h('span', {
      class: 'sr-only'
    }, slots.default?.())
  }
})

/**
 * 实时公告组件
 */
export const LiveAnnouncer = defineComponent({
  name: 'LiveAnnouncer',
  props: {
    message: String,
    priority: {
      type: String as PropType<'polite' | 'assertive'>,
      default: 'polite'
    }
  },
  setup(props) {
    return () => h('div', {
      'aria-live': props.priority,
      'aria-atomic': 'true',
      class: 'sr-only'
    }, props.message)
  }
})
