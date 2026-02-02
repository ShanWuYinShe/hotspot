/**
 * Svelte Action: 移动端触摸反馈效果
 * 
 * 为按钮添加触摸高亮效果，确保反馈区域是圆形而不是长条
 * 符合移动端交互最佳实践 (最小44x44px触摸目标)
 */

interface TouchFeedbackOptions {
	/** 反馈持续时间(ms) */
	duration?: number;
	/** 反馈颜色 */
	color?: string;
	/** 是否使用圆形反馈 */
	circular?: boolean;
}

const defaultOptions: Required<TouchFeedbackOptions> = {
	duration: 400,
	color: 'currentColor',
	circular: true
};

export function touchFeedback(node: HTMLElement, options: TouchFeedbackOptions = {}) {
	const opts = { ...defaultOptions, ...options };
	
	// 确保元素有相对定位
	if (getComputedStyle(node).position === 'static') {
		node.style.position = 'relative';
	}
	
	// 添加overflow hidden以限制ripple范围
	node.style.overflow = 'hidden';
	
	function createRipple(event: PointerEvent | TouchEvent) {
		// 移除旧的ripple
		const existingRipples = node.querySelectorAll('.touch-ripple');
		existingRipples.forEach(ripple => ripple.remove());
		
		// 创建新的ripple元素
		const ripple = document.createElement('span');
		ripple.classList.add('touch-ripple');
		
		// 获取点击位置
		let clientX: number, clientY: number;
		if ('touches' in event) {
			clientX = event.touches[0].clientX;
			clientY = event.touches[0].clientY;
		} else {
			clientX = event.clientX;
			clientY = event.clientY;
		}
		
		const rect = node.getBoundingClientRect();
		
		// 计算ripple大小和位置
		let size: number;
		let x: number;
		let y: number;
		
		if (opts.circular) {
			// 圆形反馈：使用元素最小边作为直径
			size = Math.min(rect.width, rect.height);
			// 居中显示
			x = (rect.width - size) / 2;
			y = (rect.height - size) / 2;
		} else {
			// 点击位置ripple
			size = Math.max(rect.width, rect.height) * 2;
			x = clientX - rect.left - size / 2;
			y = clientY - rect.top - size / 2;
		}
		
		// 设置样式
		Object.assign(ripple.style, {
			position: 'absolute',
			left: `${x}px`,
			top: `${y}px`,
			width: `${size}px`,
			height: `${size}px`,
			borderRadius: '50%',
			backgroundColor: opts.color === 'currentColor' 
				? 'currentColor' 
				: opts.color,
			opacity: '0.15',
			pointerEvents: 'none',
			transform: 'scale(0)',
			transition: `transform ${opts.duration}ms cubic-bezier(0.4, 0, 0.2, 1), opacity ${opts.duration}ms cubic-bezier(0.4, 0, 0.2, 1)`
		});
		
		node.appendChild(ripple);
		
		// 触发动画
		requestAnimationFrame(() => {
			ripple.style.transform = 'scale(1)';
			ripple.style.opacity = '0';
		});
		
		// 动画结束后移除
		setTimeout(() => {
			ripple.remove();
		}, opts.duration);
	}
	
	// 监听触摸和点击事件
	node.addEventListener('pointerdown', createRipple);
	node.addEventListener('touchstart', createRipple, { passive: true });
	
	return {
		update(newOptions: TouchFeedbackOptions) {
			Object.assign(opts, newOptions);
		},
		destroy() {
			node.removeEventListener('pointerdown', createRipple);
			node.removeEventListener('touchstart', createRipple);
			
			// 清理残留的ripple
			const ripples = node.querySelectorAll('.touch-ripple');
			ripples.forEach(ripple => ripple.remove());
		}
	};
}

/**
 * 预设：圆形触摸反馈 (用于icon按钮)
 */
export function circularFeedback(node: HTMLElement) {
	return touchFeedback(node, { circular: true });
}

/**
 * 预设：点击位置ripple (用于普通按钮)
 */
export function rippleFeedback(node: HTMLElement) {
	return touchFeedback(node, { circular: false });
}
