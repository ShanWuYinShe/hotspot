/**
 * 手势识别 Composable
 * 支持触摸、鼠标手势、拖拽、缩放、旋转
 */

interface Point {
  x: number
  y: number
}

interface GestureState {
  startPoint: Point
  currentPoint: Point
  deltaX: number
  deltaY: number
  distance: number
  angle: number
  velocity: Point
  isActive: boolean
}

interface GestureOptions {
  /** 最小移动距离 */
  threshold?: number
  /** 启用的手势类型 */
  gestures?: ('pan' | 'pinch' | 'rotate' | 'swipe' | 'tap' | 'longpress')[]
  /** 长按时间 */
  longPressDelay?: number
  /** 双击间隔 */
  doubleTapDelay?: number
  /** 滑动速度阈值 */
  swipeVelocity?: number
}

export function useGestures(
  target: Ref<HTMLElement | undefined>,
  options: GestureOptions = {}
) {
  const {
    threshold = 10,
    gestures = ['pan', 'tap'],
    longPressDelay = 500,
    doubleTapDelay = 300,
    swipeVelocity = 0.5
  } = options

  const state = reactive<GestureState>({
    startPoint: { x: 0, y: 0 },
    currentPoint: { x: 0, y: 0 },
    deltaX: 0,
    deltaY: 0,
    distance: 0,
    angle: 0,
    velocity: { x: 0, y: 0 },
    isActive: false
  })

  // 触摸状态
  const touches = ref<Touch[]>([])
  const lastTapTime = ref(0)
  const tapCount = ref(0)
  const longPressTimer = ref<NodeJS.Timeout>()

  // 事件回调函数
  const eventCallbacks = {
    onPanStart: (callback: (state: GestureState) => void) => {
      eventCallbacks._panStartCallback = callback
    },
    onPanMove: (callback: (state: GestureState) => void) => {
      eventCallbacks._panMoveCallback = callback
    },
    onPanEnd: (callback: (state: GestureState) => void) => {
      eventCallbacks._panEndCallback = callback
    },
    onPinchStart: (callback: (state: GestureState) => void) => {
      eventCallbacks._pinchStartCallback = callback
    },
    onPinchMove: (callback: (state: GestureState, scale: number) => void) => {
      eventCallbacks._pinchMoveCallback = callback
    },
    onPinchEnd: (callback: (state: GestureState) => void) => {
      eventCallbacks._pinchEndCallback = callback
    },
    onRotateStart: (callback: (state: GestureState) => void) => {
      eventCallbacks._rotateStartCallback = callback
    },
    onRotateMove: (callback: (state: GestureState, rotation: number) => void) => {
      eventCallbacks._rotateMoveCallback = callback
    },
    onRotateEnd: (callback: (state: GestureState) => void) => {
      eventCallbacks._rotateEndCallback = callback
    },
    onSwipe: (callback: (direction: 'left' | 'right' | 'up' | 'down', state: GestureState) => void) => {
      eventCallbacks._swipeCallback = callback
    },
    onTap: (callback: (state: GestureState) => void) => {
      eventCallbacks._tapCallback = callback
    },
    onDoubleTap: (callback: (state: GestureState) => void) => {
      eventCallbacks._doubleTapCallback = callback
    },
    onLongPress: (callback: (state: GestureState) => void) => {
      eventCallbacks._longPressCallback = callback
    },
    _panStartCallback: null as ((state: GestureState) => void) | null,
    _panMoveCallback: null as ((state: GestureState) => void) | null,
    _panEndCallback: null as ((state: GestureState) => void) | null,
    _pinchStartCallback: null as ((state: GestureState) => void) | null,
    _pinchMoveCallback: null as ((state: GestureState, scale: number) => void) | null,
    _pinchEndCallback: null as ((state: GestureState) => void) | null,
    _rotateStartCallback: null as ((state: GestureState) => void) | null,
    _rotateMoveCallback: null as ((state: GestureState, rotation: number) => void) | null,
    _rotateEndCallback: null as ((state: GestureState) => void) | null,
    _swipeCallback: null as ((direction: 'left' | 'right' | 'up' | 'down', state: GestureState) => void) | null,
    _tapCallback: null as ((state: GestureState) => void) | null,
    _doubleTapCallback: null as ((state: GestureState) => void) | null,
    _longPressCallback: null as ((state: GestureState) => void) | null
  }

  // 触发事件的辅助函数
  const emitPanStart = (state: GestureState) => eventCallbacks._panStartCallback?.(state)
  const emitPanMove = (state: GestureState) => eventCallbacks._panMoveCallback?.(state)
  const emitPanEnd = (state: GestureState) => eventCallbacks._panEndCallback?.(state)
  const emitPinchStart = (state: GestureState) => eventCallbacks._pinchStartCallback?.(state)
  const emitPinchMove = (state: GestureState, scale: number) => eventCallbacks._pinchMoveCallback?.(state, scale)
  const emitPinchEnd = (state: GestureState) => eventCallbacks._pinchEndCallback?.(state)
  const emitRotateStart = (state: GestureState) => eventCallbacks._rotateStartCallback?.(state)
  const emitRotateMove = (state: GestureState, rotation: number) => eventCallbacks._rotateMoveCallback?.(state, rotation)
  const emitRotateEnd = (state: GestureState) => eventCallbacks._rotateEndCallback?.(state)
  const emitSwipe = (direction: 'left' | 'right' | 'up' | 'down', state: GestureState) => eventCallbacks._swipeCallback?.(direction, state)
  const emitTap = (state: GestureState) => eventCallbacks._tapCallback?.(state)
  const emitDoubleTap = (state: GestureState) => eventCallbacks._doubleTapCallback?.(state)
  const emitLongPress = (state: GestureState) => eventCallbacks._longPressCallback?.(state)

  // 工具函数
  const getDistance = (p1: Point, p2: Point): number => {
    return Math.sqrt(Math.pow(p2.x - p1.x, 2) + Math.pow(p2.y - p1.y, 2))
  }

  const getAngle = (p1: Point, p2: Point): number => {
    return Math.atan2(p2.y - p1.y, p2.x - p1.x) * 180 / Math.PI
  }

  const getCenter = (touches: Touch[]): Point => {
    if (touches.length === 1) {
      return { x: touches[0].clientX, y: touches[0].clientY }
    }
    
    const x = touches.reduce((sum, touch) => sum + touch.clientX, 0) / touches.length
    const y = touches.reduce((sum, touch) => sum + touch.clientY, 0) / touches.length
    return { x, y }
  }

  const updateState = (point: Point) => {
    state.currentPoint = point
    state.deltaX = point.x - state.startPoint.x
    state.deltaY = point.y - state.startPoint.y
    state.distance = getDistance(state.startPoint, point)
    state.angle = getAngle(state.startPoint, point)
  }

  // 触摸事件处理
  const handleTouchStart = (event: TouchEvent) => {
    touches.value = Array.from(event.touches)
    const center = getCenter(touches.value)
    
    state.startPoint = center
    state.currentPoint = center
    state.isActive = true
    
    // 长按检测
    if (gestures.includes('longpress')) {
      longPressTimer.value = setTimeout(() => {
        emitLongPress({ ...state })
      }, longPressDelay)
    }
    
    // 多点触控
    if (touches.value.length === 2) {
      if (gestures.includes('pinch')) {
        emitPinchStart({ ...state })
      }
      if (gestures.includes('rotate')) {
        emitRotateStart({ ...state })
      }
    } else if (touches.value.length === 1) {
      if (gestures.includes('pan')) {
        emitPanStart({ ...state })
      }
    }
  }

  const handleTouchMove = (event: TouchEvent) => {
    if (!state.isActive) return
    
    event.preventDefault()
    touches.value = Array.from(event.touches)
    const center = getCenter(touches.value)
    
    updateState(center)
    
    // 清除长按定时器
    if (longPressTimer.value) {
      clearTimeout(longPressTimer.value)
      longPressTimer.value = undefined
    }
    
    if (touches.value.length === 2) {
      const touch1 = touches.value[0]
      const touch2 = touches.value[1]
      
      // 缩放手势
      if (gestures.includes('pinch')) {
        const currentDistance = getDistance(
          { x: touch1.clientX, y: touch1.clientY },
          { x: touch2.clientX, y: touch2.clientY }
        )
        const initialDistance = getDistance(state.startPoint, state.startPoint) // 需要保存初始距离
        const scale = currentDistance / (initialDistance || 1)
        emitPinchMove({ ...state }, scale)
      }
      
      // 旋转手势
      if (gestures.includes('rotate')) {
        const currentAngle = getAngle(
          { x: touch1.clientX, y: touch1.clientY },
          { x: touch2.clientX, y: touch2.clientY }
        )
        emitRotateMove({ ...state }, currentAngle)
      }
    } else if (touches.value.length === 1) {
      // 拖拽手势
      if (gestures.includes('pan') && state.distance > threshold) {
        emitPanMove({ ...state })
      }
    }
  }

  const handleTouchEnd = (event: TouchEvent) => {
    if (!state.isActive) return
    
    // 清除长按定时器
    if (longPressTimer.value) {
      clearTimeout(longPressTimer.value)
      longPressTimer.value = undefined
    }
    
    const wasMultiTouch = touches.value.length > 1
    touches.value = Array.from(event.touches)
    
    if (touches.value.length === 0) {
      // 所有触摸结束
      
      // 滑动检测
      if (gestures.includes('swipe') && state.distance > threshold) {
        const absX = Math.abs(state.deltaX)
        const absY = Math.abs(state.deltaY)
        
        if (absX > absY) {
          // 水平滑动
          const direction = state.deltaX > 0 ? 'right' : 'left'
          emitSwipe(direction, { ...state })
        } else {
          // 垂直滑动
          const direction = state.deltaY > 0 ? 'down' : 'up'
          emitSwipe(direction, { ...state })
        }
      }
      
      // 点击检测
      if (gestures.includes('tap') && state.distance < threshold) {
        const now = Date.now()
        
        if (now - lastTapTime.value < doubleTapDelay) {
          tapCount.value++
        } else {
          tapCount.value = 1
        }
        
        lastTapTime.value = now
        
        if (tapCount.value === 1) {
          setTimeout(() => {
            if (tapCount.value === 1) {
              emitTap({ ...state })
            } else if (tapCount.value === 2) {
              emitDoubleTap({ ...state })
            }
            tapCount.value = 0
          }, doubleTapDelay)
        }
      }
      
      // 结束事件
      if (wasMultiTouch) {
        if (gestures.includes('pinch')) {
          emitPinchEnd({ ...state })
        }
        if (gestures.includes('rotate')) {
          emitRotateEnd({ ...state })
        }
      } else {
        if (gestures.includes('pan')) {
          emitPanEnd({ ...state })
        }
      }
      
      state.isActive = false
    }
  }

  // 鼠标事件处理（用于桌面端）
  const handleMouseDown = (event: MouseEvent) => {
    const point = { x: event.clientX, y: event.clientY }
    state.startPoint = point
    state.currentPoint = point
    state.isActive = true
    
    if (gestures.includes('pan')) {
      emitPanStart({ ...state })
    }
  }

  const handleMouseMove = (event: MouseEvent) => {
    if (!state.isActive) return
    
    const point = { x: event.clientX, y: event.clientY }
    updateState(point)
    
    if (gestures.includes('pan') && state.distance > threshold) {
      emitPanMove({ ...state })
    }
  }

  const handleMouseUp = (event: MouseEvent) => {
    if (!state.isActive) return
    
    if (gestures.includes('tap') && state.distance < threshold) {
      emitTap({ ...state })
    }
    
    if (gestures.includes('pan')) {
      emitPanEnd({ ...state })
    }
    
    state.isActive = false
  }

  // 初始化函数 - 需要在组件中手动调用
  const initialize = () => {
    const element = target.value
    if (!element) return
    
    // 触摸事件
    element.addEventListener('touchstart', handleTouchStart, { passive: false })
    element.addEventListener('touchmove', handleTouchMove, { passive: false })
    element.addEventListener('touchend', handleTouchEnd, { passive: false })
    
    // 鼠标事件
    element.addEventListener('mousedown', handleMouseDown)
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    
    // 返回清理函数
    return () => {
      // 清理事件监听器
      element.removeEventListener('touchstart', handleTouchStart)
      element.removeEventListener('touchmove', handleTouchMove)
      element.removeEventListener('touchend', handleTouchEnd)
      element.removeEventListener('mousedown', handleMouseDown)
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }

  return {
    state: readonly(state),
    isActive: readonly(toRef(state, 'isActive')),
    initialize,
    ...eventCallbacks
  }
}

/**
 * 拖拽功能 Composable
 */
export function useDraggable(
  target: Ref<HTMLElement | undefined>,
  options: {
    axis?: 'x' | 'y' | 'both'
    containment?: Ref<HTMLElement | undefined>
    grid?: [number, number]
    disabled?: Ref<boolean>
  } = {}
) {
  const { axis = 'both', containment, grid, disabled } = options
  
  const isDragging = ref(false)
  const position = reactive({ x: 0, y: 0 })
  const startPosition = reactive({ x: 0, y: 0 })
  
  const { state } = useGestures(target, {
    gestures: ['pan']
  })
  
  // 监听拖拽状态
  watch(() => state.isActive, (active) => {
    if (disabled?.value) return
    
    if (active) {
      isDragging.value = true
      startPosition.x = position.x
      startPosition.y = position.y
    } else {
      isDragging.value = false
    }
  })
  
  // 更新位置
  watch([() => state.deltaX, () => state.deltaY], ([deltaX, deltaY]) => {
    if (!isDragging.value || disabled?.value) return
    
    let newX = startPosition.x + deltaX
    let newY = startPosition.y + deltaY
    
    // 轴限制
    if (axis === 'x') newY = position.y
    if (axis === 'y') newX = position.x
    
    // 网格对齐
    if (grid) {
      newX = Math.round(newX / grid[0]) * grid[0]
      newY = Math.round(newY / grid[1]) * grid[1]
    }
    
    // 边界限制
    if (containment?.value) {
      const container = containment.value.getBoundingClientRect()
      const element = target.value?.getBoundingClientRect()
      
      if (element) {
        newX = Math.max(0, Math.min(newX, container.width - element.width))
        newY = Math.max(0, Math.min(newY, container.height - element.height))
      }
    }
    
    position.x = newX
    position.y = newY
    
    // 应用变换
    if (target.value) {
      target.value.style.transform = `translate(${position.x}px, ${position.y}px)`
    }
  })
  
  return {
    isDragging: readonly(isDragging),
    position: readonly(position),
    setPosition: (x: number, y: number) => {
      position.x = x
      position.y = y
      if (target.value) {
        target.value.style.transform = `translate(${x}px, ${y}px)`
      }
    }
  }
}
