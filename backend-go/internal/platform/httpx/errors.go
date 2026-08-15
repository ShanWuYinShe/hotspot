package httpx

import "fmt"

// Code 是 API 业务错误码的类型化枚举（权威定义：spec/api_spec_errors.md）。
// 所有错误码必须通过本组常量引用，禁止在业务代码中内联魔法数字——
// 统一出口保证改一处、全局生效，且错误码无法与任意 int 混用。
type Code int

const (
	CodeOK Code = 0

	// HTTP API 业务错误码（10000-19999，spec/api_spec_errors.md）
	CodeInvalidRequest       Code = 10000 // 参数错误/请求体非法
	CodeUnauthorized         Code = 10100 // 认证失败（token 无效/过期、密码错误）
	CodeForbidden            Code = 10200 // 权限不足/访问受限
	CodeAccountBanned        Code = 10201 // 账号被封禁（强制登出 + 封禁页）
	CodeNotFound             Code = 10300 // 资源不存在
	CodeConflict             Code = 10400 // 业务冲突（已存在/满员/已撤回）
	CodeInvalidOldCredential Code = 10401 // 旧凭证验证失败（旧密码错误/验证码错误；不触发 token 刷新流程）
	CodeRateLimited          Code = 10500 // 请求过于频繁（按 Retry-After 重试）
	CodeInternalUnavailable  Code = 10600 // 服务器错误/系统不可用

	// 幂等性错误码（22000-22099，spec/api_spec_errors.md）
	CodeIdempotencyConflict Code = 22000 // 幂等键冲突（换键重试）
	CodeIdempotencyInFlight Code = 22002 // 幂等操作进行中（稍后重试）
)

// ForbiddenReason 是 10200 失败的结构化原因（权威定义：spec/api_spec_errors.md「ForbiddenReason」）。
// 所有 10200 响应必须携带 data.forbidden_reason，客户端按枚举分支渲染提示/引导。
type ForbiddenReason string

const (
	ReasonNotAdmin                    ForbiddenReason = "not_admin"
	ReasonInsufficientAdminPermission ForbiddenReason = "insufficient_admin_permission"
	ReasonAdminProtectedTarget        ForbiddenReason = "admin_protected_target"
	ReasonNotGroupMember              ForbiddenReason = "not_group_member"
	ReasonNotGroupAdmin               ForbiddenReason = "not_group_admin"
	ReasonNotGroupOwner               ForbiddenReason = "not_group_owner"
	ReasonNotResourceOwner            ForbiddenReason = "not_resource_owner"
	ReasonPrivacySetting              ForbiddenReason = "privacy_setting"
	ReasonBlockedByUser               ForbiddenReason = "blocked_by_user"
	ReasonStrangerQuotaExceeded       ForbiddenReason = "stranger_quota_exceeded"
	ReasonBannedInGroup               ForbiddenReason = "banned_in_group"
)

// ForbiddenErrorData 是 10200 失败响应的 data 结构（契约级）。
type ForbiddenErrorData struct {
	ForbiddenReason ForbiddenReason `json:"forbidden_reason"`
}

type APIError struct {
	Code    Code
	Message string
	Data    any // 可选：错误响应的 data（仅端点显式声明时设置，如 ForbiddenErrorData）
}

// Forbidden 构造带 forbidden_reason 的 10200 权限错误（所有 10200 的统一出口）。
func Forbidden(reason ForbiddenReason, message string) *APIError {
	return &APIError{Code: CodeForbidden, Message: message, Data: ForbiddenErrorData{ForbiddenReason: reason}}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d: %s", e.Code, e.Message)
}

func InvalidRequest(message string) *APIError {
	return &APIError{Code: CodeInvalidRequest, Message: message}
}
