/**
 * 🎯 最优雅的API错误处理示例
 * 展示统一的错误处理流程和最佳实践
 */

// ✅ 这就是我们需要的全部！超级简单优雅
export const apiUsageExamples = {
  
  // 🎯 基础用法 - 自动处理所有错误
  async basicUsage() {
    const api = useApi();
    
    try {
      // 后端返回：{ code: 0, msg: "success", data: {...} }
      const user = await api.post('/user/profile', { user_id: 123 });
      // ✅ 成功时，user 直接是 data 部分，无需额外处理
      if (import.meta.dev) console.log('用户信息:', user);
      
    } catch (error: unknown) {
      // ❌ 失败时，error 已经被统一处理和增强
      // - HTTP错误：网络问题、服务器错误等
      // - 业务错误：code !== 0 的情况
      const apiError = error as { message?: string; isBusinessError?: boolean; businessCode?: number };
      console.error('请求失败:', apiError.message);
      
      // 🏷️ 可以根据错误类型做特殊处理
      if (apiError.isBusinessError) {
        if (import.meta.dev) console.log('业务错误码:', apiError.businessCode);
        // 1000: 验证失败, 1002: 认证失败, 1003: 权限不足, 1004: 资源不存在, 1005: 冲突
      }
    }
  },

  // 🎯 分页数据用法
  async paginatedUsage() {
    const api = useApi();
    
    try {
      // 后端返回：{ code: 0, msg: "success", data: { items: [...], pagination: {...} } }
      const result = await api.post('/user/nearby', { 
        latitude: 39.9042, 
        longitude: 116.4074,
        cursor: 0,
        limit: 20 
      });
      
      // ✅ result 直接是分页数据，无需额外处理
      if (import.meta.dev) {
        if (import.meta.dev) console.log('附近用户:', result.items);
        if (import.meta.dev) console.log('是否有更多:', result.pagination.has_more);
      }
      
    } catch (error: unknown) {
      // ❌ 统一错误处理
      const apiError = error as { message?: string };
      console.error('获取附近用户失败:', apiError.message);
    }
  },

  // 🎯 在组件中的用法
  componentUsage: `
    <script setup lang="ts">
    const api = useApi();
    const { $toast } = useNuxtApp();
    
    const loading = ref(false);
    const users = ref([]);
    
    const fetchUsers = async () => {
      loading.value = true;
      try {
        // 🎯 就这么简单！
        const result = await api.post('/user/nearby', params);
        users.value = result.items;
        
      } catch (error: unknown) {
        // ✅ 错误已经被统一处理，只需要显示给用户
        const apiError = error as { message?: string };
        $toast.error(apiError.message || '请求失败');
        
      } finally {
        loading.value = false;
      }
    };
    </script>
  `,

  // 🎯 Service层的用法
  serviceUsage: `
    // services/user.service.ts
    export const useUserService = () => {
      const api = useApi();
      
      const getNearbyUsers = async (params: NearbyUsersParams) => {
        // 🎯 Service层专注业务逻辑，错误处理交给调用方
        return await api.post('/user/nearby', params);
      };
      
      return { getNearbyUsers };
    };
  `
};

/**
 * 🔍 后端响应格式说明
 */
export const backendResponseFormats = {
  
  // ✅ 成功响应
  success: {
    code: 0,
    msg: "success", // 或者 "message": "success"
    data: {
      // 实际数据
      user_id: 123,
      nickname: "张三",
      avatar_url: "https://..."
    }
  },

  // ❌ 业务错误响应
  businessError: {
    code: 1004, // 业务错误码
    msg: "用户不存在", // 或者 "message": "用户不存在"
    data: null
  },

  // 📄 分页响应
  paginatedSuccess: {
    code: 0,
    msg: "success",
    data: {
      items: [
        { user_id: 1, nickname: "用户1" },
        { user_id: 2, nickname: "用户2" }
      ],
      pagination: {
        has_more: true,
        next_cursor: 123
      }
    }
  }
};

/**
 * 🎯 错误码对照表
 */
export const errorCodes = {
  1000: "验证失败",
  1002: "认证失败", 
  1003: "权限不足",
  1004: "资源不存在",
  1005: "冲突错误",
  5000: "服务器内部错误"
};
