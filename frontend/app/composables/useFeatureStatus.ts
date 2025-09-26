/**
 * 功能状态管理 - 统一管理功能的开发状态
 * 简洁而不简陋：提供完整的功能状态管理，但保持代码简洁
 */

export interface FeatureConfig {
  enabled: boolean
  message?: string
  action?: () => void | Promise<void>
  fallback?: () => void | Promise<void>
}

export interface FeatureStatus {
  // 用户相关功能
  friends: FeatureConfig
  userFollow: FeatureConfig
  userBlock: FeatureConfig
  
  // 群组相关功能
  groupCreate: FeatureConfig
  groupManage: FeatureConfig
  
  // 动态相关功能
  momentCreate: FeatureConfig
  momentEdit: FeatureConfig
  momentDelete: FeatureConfig
  momentSearch: FeatureConfig
  
  // 消息相关功能
  privateChat: FeatureConfig
  groupChat: FeatureConfig
  messageMedia: FeatureConfig
  
  // 系统功能
  notifications: FeatureConfig
  reports: FeatureConfig
  settings: FeatureConfig
}

/**
 * 功能状态配置
 * 通过配置控制功能的启用状态，而不是硬编码"开发中"提示
 */
const featureStatus: FeatureStatus = {
  // 用户相关功能 - 基础实现完成，可以启用
  friends: {
    enabled: true,
    message: '好友功能'
  },
  
  userFollow: {
    enabled: true,
    message: '关注功能'
  },
  
  userBlock: {
    enabled: true,
    message: '拉黑功能'
  },
  
  // 群组相关功能 - 已有基础实现
  groupCreate: {
    enabled: true,
    message: '创建群组'
  },
  
  groupManage: {
    enabled: true,
    message: '群组管理'
  },
  
  // 动态相关功能 - 恢复完整功能
  momentCreate: {
    enabled: true,
    message: '发布动态'
  },
  
  momentEdit: {
    enabled: true,
    message: '编辑动态'
  },
  
  momentDelete: {
    enabled: true,
    message: '删除动态'
  },
  
  momentSearch: {
    enabled: true,
    message: '搜索动态'
  },
  
  // 消息相关功能 - 基于统一架构实现
  privateChat: {
    enabled: true,
    message: '私聊功能'
  },
  
  groupChat: {
    enabled: true,
    message: '群聊功能'
  },
  
  messageMedia: {
    enabled: true,
    message: '媒体消息'
  },
  
  // 系统功能
  notifications: {
    enabled: true,
    message: '通知功能'
  },
  
  reports: {
    enabled: true,
    message: '举报功能'
  },
  
  settings: {
    enabled: true,
    message: '设置功能'
  }
}

/**
 * 功能状态管理 Composable
 */
export const useFeatureStatus = () => {
  const toast = useToast()
  
  /**
   * 检查功能是否启用
   */
  const isFeatureEnabled = (feature: keyof FeatureStatus): boolean => {
    return featureStatus[feature].enabled
  }
  
  /**
   * 执行功能或显示提示
   */
  const executeFeature = async (
    feature: keyof FeatureStatus,
    action: () => void | Promise<void>,
    options?: {
      skipCheck?: boolean
      customMessage?: string
    }
  ) => {
    const config = featureStatus[feature]
    
    if (options?.skipCheck || config.enabled) {
      try {
        await action()
      } catch (error) {
        console.error(`${config.message}执行失败:`, error)
        toast.add({
          title: `${config.message}失败`,
          description: error instanceof Error ? error.message : '请稍后重试',
          color: 'error'
        })
      }
    } else {
      // 功能未启用时的处理
      if (config.fallback) {
        await config.fallback()
      } else {
        toast.add({
          title: '功能开发中',
          description: options?.customMessage || `${config.message}即将上线`,
          color: 'info'
        })
      }
    }
  }
  
  /**
   * 批量检查功能状态
   */
  const checkFeatures = (features: (keyof FeatureStatus)[]): Record<string, boolean> => {
    return features.reduce((acc, feature) => {
      acc[feature] = isFeatureEnabled(feature)
      return acc
    }, {} as Record<string, boolean>)
  }
  
  /**
   * 获取功能配置
   */
  const getFeatureConfig = (feature: keyof FeatureStatus): FeatureConfig => {
    return featureStatus[feature]
  }
  
  return {
    isFeatureEnabled,
    executeFeature,
    checkFeatures,
    getFeatureConfig,
    featureStatus: readonly(featureStatus)
  }
}

/**
 * 功能状态装饰器 - 简化功能调用
 */
export const withFeatureCheck = <T extends any[]>(
  feature: keyof FeatureStatus,
  fn: (...args: T) => void | Promise<void>
) => {
  return async (...args: T) => {
    const { executeFeature } = useFeatureStatus()
    await executeFeature(feature, () => fn(...args))
  }
}
