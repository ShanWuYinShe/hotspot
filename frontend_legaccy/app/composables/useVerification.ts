/**
 * 手机验证码 Composable
 * 
 * 提供：
 * - 发送验证码
 * - 验证验证码
 * - 倒计时管理
 */

import type { 
  SendVerificationCodeRequest, 
  SendVerificationCodeData,
  VerifyCodeRequest,
  VerifyCodeData
} from '~/types/api'

export function useVerification() {
  const { withErrorHandling } = useErrorHandler()
  
  // 倒计时状态（秒）
  const countdown = ref(0)
  const isCountingDown = computed(() => countdown.value > 0)
  
  // 倒计时定时器
  let countdownTimer: ReturnType<typeof setInterval> | null = null
  
  /**
   * 开始倒计时
   * @param seconds 倒计时秒数，默认 60 秒
   */
  const startCountdown = (seconds: number = 60) => {
    countdown.value = seconds
    
    if (countdownTimer) {
      clearInterval(countdownTimer)
    }
    
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        stopCountdown()
      }
    }, 1000)
  }
  
  /**
   * 停止倒计时
   */
  const stopCountdown = () => {
    if (countdownTimer) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
    countdown.value = 0
  }
  
  /**
   * 发送验证码
   * @param request 发送验证码请求
   * @returns 验证码会话数据
   */
  const sendVerificationCode = async (
    request: SendVerificationCodeRequest,
    options?: {
      title?: string
      loading?: Ref<boolean>
      autoCountdown?: boolean // 是否自动开始倒计时，默认 true
    }
  ): Promise<SendVerificationCodeData | null> => {
    return await withErrorHandling<SendVerificationCodeData>(
      async () => {
        const { data } = await $fetch<{ code: number; data: SendVerificationCodeData }>('/api/v1/public/auth/send_verification_code', {
          method: 'POST',
          body: request
        })
        
        if (data) {
          // 自动开始倒计时（根据 retry_after 或默认 60 秒）
          if (options?.autoCountdown !== false) {
            const seconds = data.retry_after ?? 60
            startCountdown(seconds)
          }
        }
        
        return data
      },
      {
        title: options?.title ?? '发送验证码失败',
        fallbackMessage: '请稍后重试',
        loading: options?.loading
      }
    )
  }
  
  /**
   * 验证验证码
   * @param request 验证码验证请求
   * @returns 验证通过后的 token
   */
  const verifyCode = async (
    request: VerifyCodeRequest,
    options?: {
      title?: string
      loading?: Ref<boolean>
    }
  ): Promise<VerifyCodeData | null> => {
    return await withErrorHandling<VerifyCodeData>(
      async () => {
        const { data } = await $fetch<{ code: number; data: VerifyCodeData }>('/api/v1/public/auth/verify_code', {
          method: 'POST',
          body: request
        })
        
        return data
      },
      {
        title: options?.title ?? '验证失败',
        fallbackMessage: '验证码错误或已过期，请重试',
        loading: options?.loading
      }
    )
  }
  
  // 组件卸载时清理定时器
  onUnmounted(() => {
    stopCountdown()
  })
  
  return {
    // 状态
    countdown,
    isCountingDown,
    
    // 方法
    sendVerificationCode,
    verifyCode,
    startCountdown,
    stopCountdown
  }
}
