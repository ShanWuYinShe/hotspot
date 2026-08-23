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
// local@domain 两段均非空且 domain 含 '.'。
func isValidEmailLoginID(loginID string) bool {
	parts := strings.SplitN(loginID, "@", 2)
	return len(parts[0]) > 0 && len(parts[1]) > 0 && strings.Contains(parts[1], ".")
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
