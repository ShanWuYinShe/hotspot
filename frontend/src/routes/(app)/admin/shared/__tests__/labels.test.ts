// admin 展示标签映射测试：中文翻译完整、未知键回退原键（诚实降级）。
import { describe, it, expect } from 'vitest';
import {
	AUDIT_ACTION_LABELS,
	AUDIT_TARGET_LABELS,
	REPORT_REASON_LABELS,
	REPORT_STATUS_LABELS,
	labelOf
} from '../labels';

describe('admin labels', () => {
	it('审计动作/目标中文覆盖常用枚举', () => {
		expect(AUDIT_ACTION_LABELS['user.ban']).toBe('封禁用户');
		expect(AUDIT_ACTION_LABELS['report.judge']).toBe('裁决举报');
		expect(AUDIT_TARGET_LABELS.user).toBe('用户');
		expect(AUDIT_TARGET_LABELS.config).toBe('配置');
	});

	it('举报原因/状态中文覆盖四类常见原因', () => {
		expect(REPORT_REASON_LABELS.spam).toBe('垃圾信息');
		expect(REPORT_REASON_LABELS.misinformation).toBe('不实信息');
		expect(REPORT_STATUS_LABELS.pending).toBe('待处理');
	});

	it('未知枚举键回退原键（新枚举上线不出现空白）', () => {
		expect(labelOf(REPORT_REASON_LABELS, 'brand_new_reason')).toBe('brand_new_reason');
	});
});
