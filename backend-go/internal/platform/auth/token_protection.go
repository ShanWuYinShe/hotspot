package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// HashRefreshToken 返回 refresh token 的 SHA-256 hex。存储层只落哈希：
// DB/备份泄漏不再等价于全体会话可劫持。明文与哈希同为 64 位 hex，
// 哈希化的一次性数据迁移由 migration 053 的 data_migrations 旗标表保证。
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// TokenEscrow 宽限期托管：并发刷新败者需要在 30s 宽限期内拿回获胜链的新
// token 明文，而明文哈希后不可逆——因此轮换时以 AES-256-GCM 加密托管
// （密钥取自 AUTH_TOKEN_SECRET 派生，不入 DB），行随既有 sweeper 在宽限期
// 过后清除。DB 泄漏 + 无 secret 仍不可还原，哈希化的防劫持价值不受损。
type TokenEscrow struct {
	aead cipher.AEAD
}

func NewTokenEscrow(secret string) (*TokenEscrow, error) {
	if secret == "" {
		return nil, errors.New("token escrow: empty secret")
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("token escrow: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("token escrow: %w", err)
	}
	return &TokenEscrow{aead: aead}, nil
}

// Seal 加密明文（输出 = nonce || ciphertext），空明文返回 nil（无可托管）。
func (e *TokenEscrow) Seal(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, nil
	}
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("token escrow: %w", err)
	}
	return e.aead.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Open 解密托管密文；空输入返回空串（该行无托管，如迁移过渡期轮换行）。
func (e *TokenEscrow) Open(sealed []byte) (string, error) {
	if len(sealed) == 0 {
		return "", nil
	}
	nonceSize := e.aead.NonceSize()
	if len(sealed) < nonceSize {
		return "", errors.New("token escrow: ciphertext shorter than nonce")
	}
	plaintext, err := e.aead.Open(nil, sealed[:nonceSize], sealed[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("token escrow: %w", err)
	}
	return string(plaintext), nil
}
