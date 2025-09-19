/**
 * 智能 Toast 通知系统 - 简洁版本
 * 保持防重复功能但简化实现
 */

interface ToastMessage {
    title: string;
    description?: string;
    color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral';
    icon?: string;
    timeout?: number;
}

export function useSmartToast() {
    const toast = useToast();
    
    // 简单的重复检测
    const recentMessages = new Set<string>();
    
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

    return {
        add,
        success,
        error,
        warning,
        info
    };
}
