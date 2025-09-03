/**
 * 智能 Toast 通知系统 - 客户端版本
 * 提供防重复、智能分组、自动清理的 Toast 通知功能
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

// Toast历史记录接口
interface ToastHistory {
    count: number;
    lastShown: number;
    timeoutId?: ReturnType<typeof setTimeout>;
}

// 配置常量
const DUPLICATE_SUPPRESS_TIME = 3000; // 3秒内相同消息抑制
const CLEAR_HISTORY_TIME = 30000; // 30秒后清理历史记录
const MAX_REPEAT_SHOW = 3; // 最多连续显示3次相同消息

export function useSmartToast() {
    const toast = useToast();
    
    // 全局Toast历史记录
    const toastHistory = new Map<string, ToastHistory>();

    // 生成消息唯一键
    function generateMessageKey(message: ToastMessage): string {
        const key = `${message.title}:${message.description || ''}:${message.color || ''}`;
        return btoa(encodeURIComponent(key)); // Base64编码避免特殊字符
    }

    // 清理过期的历史记录
    function cleanupHistory() {
        const now = Date.now();
        for (const [key, history] of toastHistory.entries()) {
            if (now - history.lastShown > CLEAR_HISTORY_TIME) {
                if (history.timeoutId) {
                    clearTimeout(history.timeoutId);
                }
                toastHistory.delete(key);
            }
        }
    }

    // 智能显示Toast
    function add(message: ToastMessage | string, options: Record<string, any> = {}) {
        // 统一消息格式
        const normalizedMessage: ToastMessage = typeof message === 'string' 
            ? { title: message }
            : message;

        const messageKey = generateMessageKey(normalizedMessage);
        const now = Date.now();
        const history = toastHistory.get(messageKey);

        // 检查是否需要抑制重复消息
        if (history) {
            const timeSinceLastShown = now - history.lastShown;
            
            // 如果在抑制时间内且已达到最大重复次数，则不显示
            if (timeSinceLastShown < DUPLICATE_SUPPRESS_TIME && history.count >= MAX_REPEAT_SHOW) {
                console.log(`[SmartToast] 消息被抑制: "${normalizedMessage.title}" (已显示${history.count}次)`);
                return;
            }
            
            // 如果超过抑制时间，重置计数
            if (timeSinceLastShown >= DUPLICATE_SUPPRESS_TIME) {
                history.count = 0;
            }
        }

        // 更新或创建历史记录
        const newHistory: ToastHistory = {
            count: (history?.count || 0) + 1,
            lastShown: now
        };

        // 清理之前的超时
        if (history?.timeoutId) {
            clearTimeout(history.timeoutId);
        }

        // 设置新的清理超时
        newHistory.timeoutId = setTimeout(() => {
            toastHistory.delete(messageKey);
        }, CLEAR_HISTORY_TIME);

        toastHistory.set(messageKey, newHistory);

        // 合并选项并显示Toast
        const finalMessage = {
            ...normalizedMessage,
            ...options
        };

        // 如果是重复消息，在标题后添加计数提示
        if (newHistory.count > 1) {
            finalMessage.title = `${finalMessage.title} (${newHistory.count})`;
        }

        console.log(`[SmartToast] 显示消息: "${finalMessage.title}"`);
        toast.add(finalMessage);
    }

    // 成功消息的便捷方法
    function success(message: string | ToastMessage, options: Record<string, any> = {}) {
        const normalizedMessage = typeof message === 'string' 
            ? { title: message, color: 'success' as const }
            : { ...message, color: 'success' as const };
        
        add(normalizedMessage, options);
    }

    // 错误消息的便捷方法
    function error(message: string | ToastMessage, options: Record<string, any> = {}) {
        const normalizedMessage = typeof message === 'string' 
            ? { title: message, color: 'error' as const }
            : { ...message, color: 'error' as const };
        
        add(normalizedMessage, options);
    }

    // 警告消息的便捷方法
    function warning(message: string | ToastMessage, options: Record<string, any> = {}) {
        const normalizedMessage = typeof message === 'string' 
            ? { title: message, color: 'warning' as const }
            : { ...message, color: 'warning' as const };
        
        add(normalizedMessage, options);
    }

    // 信息消息的便捷方法
    function info(message: string | ToastMessage, options: Record<string, any> = {}) {
        const normalizedMessage = typeof message === 'string' 
            ? { title: message, color: 'info' as const }
            : { ...message, color: 'info' as const };
        
        add(normalizedMessage, options);
    }

    // 清除所有Toast
    function clear() {
        // 清理所有超时
        for (const history of toastHistory.values()) {
            if (history.timeoutId) {
                clearTimeout(history.timeoutId);
            }
        }
        toastHistory.clear();
        
        // 注意：Nuxt UI 的 useToast 可能没有 clear 方法
        // 这里只清理我们的历史记录
        console.log('[SmartToast] 已清理所有Toast历史记录');
    }

    // 获取当前活跃的Toast数量
    function getActiveCount(): number {
        return toastHistory.size;
    }

    // 定期清理过期记录
    setInterval(cleanupHistory, CLEAR_HISTORY_TIME);

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

