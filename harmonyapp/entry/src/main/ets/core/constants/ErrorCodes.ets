/**
 * API 错误码常量定义
 *
 * 遵循 spec/api_spec_errors.md：
 * - 10000-10600: 通用业务错误
 * - 22000-22002: 幂等性错误
 */
export class ErrorCode {
  static readonly PARAM_ERROR: number = 10000;
  static readonly AUTH_FAILED: number = 10100;
  static readonly PERMISSION_DENIED: number = 10200;
  static readonly ACCOUNT_BANNED: number = 10201;
  static readonly RESOURCE_NOT_FOUND: number = 10300;
  static readonly BUSINESS_CONFLICT: number = 10400;
  static readonly RATE_LIMITED: number = 10500;
  static readonly SERVICE_UNAVAILABLE: number = 10600;
  static readonly IDEMPOTENCY_KEY_CONFLICT: number = 22000;
  static readonly IDEMPOTENCY_CACHE_EXPIRED: number = 22001;
  static readonly IDEMPOTENCY_IN_FLIGHT: number = 22002;
}
