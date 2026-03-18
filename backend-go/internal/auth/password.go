package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword 使用 bcrypt 生成密码哈希。
func HashPassword(password string) (string, error) {
	// bcrypt 默认成本 10，在大多数环境下性能与安全性平衡较好。
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 对比明文密码与 bcrypt 哈希。
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
