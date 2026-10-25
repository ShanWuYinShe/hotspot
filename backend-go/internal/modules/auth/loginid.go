package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// loginIDKind 是 login_id 的格式三分（spec api_spec_base.md「通用字段约束」）：
// 手机号型=纯数字、邮箱型=含 '@' 且为有效 Email 格式、用户名型=不含 '@' 且非纯数字；
// 用户名型不得与手机号/邮箱型格式重叠。
type loginIDKind int

const (
	loginIDKindInvalid loginIDKind = iota
	loginIDKindPhone
	loginIDKindEmail
	loginIDKindUsername
)

// classifyLoginID 判定 login_id 类型；长度约束（3-64）由 isValidAuthLoginID 统一校验。
func classifyLoginID(loginID string) loginIDKind {
	switch {
	case isAllDigits(loginID):
		return loginIDKindPhone
	case strings.Contains(loginID, "@"):
		if isValidEmailLoginID(loginID) {
			return loginIDKindEmail
		}
		return loginIDKindInvalid
	default:
		return loginIDKindUsername
	}
}

// isValidEmailLoginID 与 users 模块 isValidLoginID 的邮箱分支同语义：
// 恰含一个 '@'、local/domain 非空、domain 含 '.' 且不以 '.' 开头/结尾/连续。
// 额外拒绝空白与控制字符（gosec G707：邮箱会进入 SMTP RCPT，虽然 net/smtp
// 的 validateLine 会二次拒绝 CR/LF，注册入口仍应先拒绝非法地址）。
func isValidEmailLoginID(loginID string) bool {
	for _, r := range loginID {
		if r <= ' ' || r == 0x7f {
			return false
		}
	}
	parts := strings.Split(loginID, "@")
	if len(parts) != 2 {
		return false
	}
	local, domain := parts[0], parts[1]
	if local == "" || domain == "" || !strings.Contains(domain, ".") {
		return false
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") || strings.Contains(domain, "..") {
		return false
	}
	return true
}

// isValidAuthLoginID 校验 register/signin 的 login_id 合法性：
// 长度 3-64；含 '@' 时必须为合法邮箱格式（手机号型=纯数字、用户名型=
// 不含 '@' 且非纯数字，二者在长度约束内即合法）。
func isValidAuthLoginID(loginID string) bool {
	if len(loginID) < 3 || len(loginID) > 64 {
		return false
	}
	return classifyLoginID(loginID) != loginIDKindInvalid
}

// isPhoneOrEmailLoginID 判定 login_id 是否手机号/邮箱型（注册需验证码验证所有权）。
func isPhoneOrEmailLoginID(loginID string) bool {
	kind := classifyLoginID(loginID)
	return kind == loginIDKindPhone || kind == loginIDKindEmail
}

// loginIDContact 回填注册用户的 email/phone（spec api_spec_user.md UserItem：
// email 仅在 login_id 为邮箱型时回填、phone 仅在手机号型时回填，其余类型保持
// NULL——此前实现把 login_id 无条件写入 email 列，用户名/手机号型注册会得到
// email="alice" / email="13800..." 的错误数据）。空串表示不回填（repo 层转 NULL）。
func loginIDContact(loginID string) (email, phone string) {
	switch classifyLoginID(loginID) {
	case loginIDKindEmail:
		return loginID, ""
	case loginIDKindPhone:
		return "", loginID
	default:
		return "", ""
	}
}

// guestNicknameSuffixChars 默认昵称随机后缀字符集（小写字母 + 数字，36^4 组合）。
const guestNicknameSuffixChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// guestNicknamePrefix 默认昵称前缀（spec 示例「游客·8k2f」）。
const guestNicknamePrefix = "游客·"

// generateGuestNickname 生成默认游客昵称「游客·」+ 4 位小写字母数字随机
// （crypto/rand 保证不可预测；昵称无唯一约束，碰撞概率 1/36^4 可接受）。
func generateGuestNickname() (string, error) {
	max := big.NewInt(int64(len(guestNicknameSuffixChars)))
	suffix := make([]byte, 4)
	for i := range suffix {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("auth.generateGuestNickname: %w", err)
		}
		suffix[i] = guestNicknameSuffixChars[n.Int64()]
	}
	return guestNicknamePrefix + string(suffix), nil
}
