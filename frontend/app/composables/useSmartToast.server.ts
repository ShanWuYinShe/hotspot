/**
 * 智能 Toast 通知系统 - 服务端版本
 * 提供服务端安全的占位实现
 */

// Toast消息接口
interface ToastMessage {
    title: string;
    description?: string;
    color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';
    icon?: string;
    timeout?: number;
    actions?: Array<{
        label: string;
        color?: string;
        variant?: string;
        click?: () => void;
    }>;
}

export function useSmartToast() {
    // 服务端占位实现
    function add(message: ToastMessage | string, options: Record<string, any> = {}) {
        console.warn('[SmartToast] Toast notifications are not available on server side')
    }

    function success(message: string | ToastMessage, options: Record<string, any> = {}) {
        console.warn('[SmartToast] Toast notifications are not available on server side')
    }

    function error(message: string | ToastMessage, options: Record<string, any> = {}) {
        console.warn('[SmartToast] Toast notifications are not available on server side')
    }

    function warning(message: string | ToastMessage, options: Record<string, any> = {}) {
        console.warn('[SmartToast] Toast notifications are not available on server side')
    }

    function info(message: string | ToastMessage, options: Record<string, any> = {}) {
        console.warn('[SmartToast] Toast notifications are not available on server side')
    }

    function clear() {
        console.warn('[SmartToast] Toast notifications are not available on server side')
    }

    function getActiveCount(): number {
        return 0
    }

    return {
        add,
        success,
        error,
        warning,
        info,
        clear,
        getActiveCount
    };
}

