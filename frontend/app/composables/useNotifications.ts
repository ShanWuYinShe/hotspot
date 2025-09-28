/**
 * 统一的通知系统 Composable
 * 合并了原来的 useSmartToast 和 useErrorHandler 功能
 */

interface ToastMessage {
    title: string;
    description?: string;
    color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';
    icon?: string;
    timeout?: number;
}

export function useNotifications() {
    const toast = useToast();
    
    // 简单的重复检测
    const recentMessages = new Set<string>();
    
    // ============ 基础 Toast 功能 ============
    
    function add(message: ToastMessage | string) {
        const normalizedMessage: ToastMessage = typeof message === 'string' 
            ? { title: message }
            : message;
        
        const messageKey = normalizedMessage.title;
        
        // 防重复显示（3秒内）
        if (recentMessages.has(messageKey)) {
            return;
        }
        
        recentMessages.add(messageKey);
        setTimeout(() => recentMessages.delete(messageKey), 3000);
        
        toast.add(normalizedMessage);
    }

    function success(message: string | ToastMessage) {
        const normalizedMessage: ToastMessage = typeof message === 'string' 
            ? { title: message, color: 'success' }
            : { ...message, color: 'success' };
        add(normalizedMessage);
    }

    function error(message: string | ToastMessage) {
        const normalizedMessage: ToastMessage = typeof message === 'string' 
            ? { title: message, color: 'error' }
            : { ...message, color: 'error' };
        add(normalizedMessage);
    }

    function warning(message: string | ToastMessage) {
        const normalizedMessage: ToastMessage = typeof message === 'string' 
            ? { title: message, color: 'warning' }
            : { ...message, color: 'warning' };
        add(normalizedMessage);
    }

    function info(message: string | ToastMessage) {
        const normalizedMessage: ToastMessage = typeof message === 'string' 
            ? { title: message, color: 'info' }
            : { ...message, color: 'info' };
        add(normalizedMessage);
    }

    // ============ 错误处理功能 ============

    const handleError = (error: any, context?: string) => {
        console.error(`[ErrorHandler] ${context || 'Unknown'}:`, error)
        
        let title = '操作失败'
        let description = '请稍后重试'
        
        if (error?.message) {
            description = error.message
        } else if (typeof error === 'string') {
            description = error
        }
        
        if (context) {
            title = `${context}失败`
        }
        
        error({
            title,
            description
        })
    }

    const handleApiError = (error: any, operation?: string) => {
        if (error?.status === 401) {
            handleError('登录已过期，请重新登录', operation)
            return
        }
        
        if (error?.status === 403) {
            handleError('没有权限执行此操作', operation)
            return
        }
        
        if (error?.status === 404) {
            handleError('请求的资源不存在', operation)
            return
        }
        
        if (error?.status >= 500) {
            handleError('服务器错误，请稍后重试', operation)
            return
        }
        
        handleError(error, operation)
    }

    const handleNetworkError = (error: any, operation?: string) => {
        if (!navigator.onLine) {
            handleError('网络连接已断开，请检查网络设置', operation)
            return
        }
        
        handleError('网络请求失败，请检查网络连接', operation)
    }

    return {
        // Toast 功能
        add,
        success,
        error,
        warning,
        info,
        
        // 错误处理功能
        handleError,
        handleApiError,
        handleNetworkError
    };
}

// 为了保持向后兼容，导出别名
export const useSmartToast = useNotifications;
// 兼容性导出已移除，请使用新的错误处理系统
// export const useErrorHandler = useNotifications;

