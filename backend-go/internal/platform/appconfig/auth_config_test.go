package appconfig

import "testing"

// TestAuthConfigEffective 防回归（审查遗留项）：存量配置无 auth section 时
// Effective 必须回退到迁移前的代码常量（验证码 10 分钟 / 锁定 5 次 / 15 分钟），
// 保证行为不变；显式配置优先。
func TestAuthConfigEffective(t *testing.T) {
	def := AuthConfig{}.Effective()
	if def.VerificationCodeTTLMinutes != DefaultVerificationCodeTTLMinutes ||
		def.LoginLockoutThreshold != DefaultLoginLockoutThreshold ||
		def.LoginLockoutMinutes != DefaultLoginLockoutMinutes {
		t.Fatalf("defaults: %+v", def)
	}
	if DefaultVerificationCodeTTLMinutes != 10 || DefaultLoginLockoutThreshold != 5 || DefaultLoginLockoutMinutes != 15 {
		t.Fatal("defaults must match the pre-migration hardcoded constants")
	}

	custom := AuthConfig{VerificationCodeTTLMinutes: 5, LoginLockoutThreshold: 3, LoginLockoutMinutes: 30}.Effective()
	if custom.VerificationCodeTTLMinutes != 5 || custom.LoginLockoutThreshold != 3 || custom.LoginLockoutMinutes != 30 {
		t.Fatalf("custom values must win: %+v", custom)
	}

	// 部分字段零值时只补零值字段（避免一整段配置被默认值覆盖）。
	mixed := AuthConfig{LoginLockoutThreshold: 2}.Effective()
	if mixed.LoginLockoutThreshold != 2 || mixed.VerificationCodeTTLMinutes != DefaultVerificationCodeTTLMinutes || mixed.LoginLockoutMinutes != DefaultLoginLockoutMinutes {
		t.Fatalf("partial: %+v", mixed)
	}
}
