package observability

// Redact 对敏感值做日志脱敏：保留末 4 位，其余替换为 ****。
// 用于 login_id（可能为手机号）、share_code（能力令牌）等 PII/凭据
// 进日志前处理——日志中不留可完整识别或可复用的明文。
func Redact(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}
