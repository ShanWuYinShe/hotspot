import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF30;
const TAG: string = 'ApiError';

/**
 * 业务错误类型（HTTP 层抛出的非成功 ApiResponse）。
 *
 * 此前错误信息靠运行时动态挂属性（`err as Record<string, Object>` 塞
 * code/status/data/forbidden_reason），调用方只能 untyped 读取、拼错字段名
 * 无任何编译期提示；类型化后调用方经 instanceof / 静态方法安全取值。
 */
export class ApiError extends Error {
  readonly code?: number;
  readonly status?: number;
  /** 响应 data 原始对象（如 10200 时为 { forbidden_reason }），可能缺省 */
  readonly data?: Object;
  /** 10200 权限失败的 data.forbidden_reason 便捷字段 */
  readonly forbidden_reason?: string;

  constructor(message: string, code?: number, status?: number, data?: Object, forbiddenReason?: string) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
    this.data = data;
    this.forbidden_reason = forbiddenReason;
  }

  /** 从任意抛出物中安全提取 forbidden_reason（非 ApiError 或缺省时为空串）。 */
  static forbiddenReasonOf(err: Object): string {
    if (err instanceof ApiError) {
      const reason = err.forbidden_reason;
      return typeof reason === 'string' ? reason : '';
    }
    hilog.debug(DOMAIN_NUMBER, TAG, 'forbiddenReasonOf called on non-ApiError');
    return '';
  }
}
