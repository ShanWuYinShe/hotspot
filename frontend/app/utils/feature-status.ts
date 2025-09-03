/**
 * 功能状态管理工具
 * 统一处理未实现功能的用户提示
 */

export interface FeatureInfo {
  name: string;
  description?: string;
  status: 'coming_soon' | 'in_development' | 'beta' | 'available';
  estimatedDate?: string;
}

// 功能状态配置
export const FEATURES: Record<string, FeatureInfo> = {
  // 聊天相关
  voice_call: {
    name: '语音通话',
    description: '支持一对一语音通话',
    status: 'coming_soon',
  },
  video_call: {
    name: '视频通话', 
    description: '支持一对一视频通话',
    status: 'coming_soon',
  },
  file_upload: {
    name: '文件上传',
    description: '支持发送图片、文档等文件',
    status: 'in_development',
  },
  location_share: {
    name: '位置分享',
    description: '分享当前位置信息',
    status: 'in_development',
  },
  message_search: {
    name: '消息搜索',
    description: '搜索聊天记录',
    status: 'coming_soon',
  },
  chat_info: {
    name: '聊天信息',
    description: '查看聊天详情和设置',
    status: 'coming_soon',
  },
  clear_history: {
    name: '清空聊天记录',
    description: '删除所有聊天记录',
    status: 'coming_soon',
  },
  
  // 群组相关
  group_settings: {
    name: '群组设置',
    description: '管理群组信息和权限',
    status: 'coming_soon',
  },
  invite_members: {
    name: '邀请成员',
    description: '邀请新成员加入群组',
    status: 'coming_soon',
  },
  
  // 用户相关
  add_friend: {
    name: '添加好友',
    description: '发送好友申请',
    status: 'coming_soon',
  },
  block_user: {
    name: '屏蔽用户',
    description: '屏蔽不想看到的用户',
    status: 'coming_soon',
  },
  scan_qr: {
    name: '扫码加好友',
    description: '通过二维码添加好友',
    status: 'coming_soon',
  },
  message_request: {
    name: '消息请求',
    description: '处理陌生人消息请求',
    status: 'coming_soon',
  },
  
  // 个人资料
  edit_profile: {
    name: '编辑资料',
    description: '修改个人信息',
    status: 'coming_soon',
  },
  edit_nickname: {
    name: '编辑昵称',
    description: '修改用户昵称',
    status: 'coming_soon',
  },
  upload_avatar: {
    name: '头像上传',
    description: '上传个人头像',
    status: 'coming_soon',
  },
  bind_account: {
    name: '绑定账号',
    description: '绑定第三方账号',
    status: 'coming_soon',
  },
  
  // 动态相关
  comment_moment: {
    name: '评论动态',
    description: '对动态进行评论',
    status: 'coming_soon',
  },
  share_moment: {
    name: '分享动态',
    description: '分享动态到其他平台',
    status: 'coming_soon',
  },
  report_moment: {
    name: '举报动态',
    description: '举报不当内容',
    status: 'coming_soon',
  },
};

/**
 * 获取功能状态提示消息
 */
export function getFeatureStatusMessage(featureKey: string): {
  title: string;
  description: string;
  color: 'info' | 'warning' | 'success';
} {
  const feature = FEATURES[featureKey];
  
  if (!feature) {
    return {
      title: '功能暂未开放',
      description: '该功能正在开发中，敬请期待',
      color: 'info',
    };
  }
  
  switch (feature.status) {
    case 'coming_soon':
      return {
        title: '功能即将上线',
        description: `${feature.name}功能即将上线，敬请期待`,
        color: 'info',
      };
    case 'in_development':
      return {
        title: '功能开发中',
        description: `${feature.name}功能正在开发中，即将上线`,
        color: 'warning',
      };
    case 'beta':
      return {
        title: '测试版功能',
        description: `${feature.name}功能处于测试阶段`,
        color: 'warning',
      };
    case 'available':
      return {
        title: '功能可用',
        description: `${feature.name}功能已可用`,
        color: 'success',
      };
    default:
      return {
        title: '功能暂未开放',
        description: feature.description || '该功能正在开发中，敬请期待',
        color: 'info',
      };
  }
}

/**
 * 显示功能状态提示的Composable
 */
export function useFeatureStatus() {
  const toast = useToast();
  
  const showFeatureStatus = (featureKey: string) => {
    const message = getFeatureStatusMessage(featureKey);
    toast.add({
      title: message.title,
      description: message.description,
      color: message.color,
    });
  };
  
  const isFeatureAvailable = (featureKey: string): boolean => {
    const feature = FEATURES[featureKey];
    return feature?.status === 'available' || feature?.status === 'beta';
  };
  
  return {
    showFeatureStatus,
    isFeatureAvailable,
    getFeatureStatusMessage,
  };
}
