package auth

import (
	"context"
	"sync"
	"time"
)

type accountRecord struct {
	Account
	PasswordHash      string
	RegisterIP        string
	RegisterLocation  string
	LastLoginIP       string
	LastLoginLocation string
}

type passwordResetRecord struct {
	RequestID        string
	LoginID          string
	VerificationCode string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	UsedAt           *time.Time
	FailedAttempts   int
}

// registerCodeRecord 注册验证码记录（与 passwordResetRecord 同构，独立存放：
// 注册码属于「未来账号」，不关联既有账号）。
type registerCodeRecord struct {
	RequestID        string
	LoginID          string
	VerificationCode string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	ConsumedAt       *time.Time
	FailedAttempts   int
}

type InMemoryRepository struct {
	mu              sync.RWMutex
	accountsByLogin map[string]accountRecord
	accountsByUser  map[string]accountRecord
	resetsByLogin   map[string][]passwordResetRecord
	registerCodes   map[string][]registerCodeRecord
}

const passwordResetThrottleWindow = 10 * time.Minute

// maxVerificationCodeFailures 验证码容错次数：同一 login_id 最新有效码累计
// 失败满此数即静默作废（防爆破；对齐登录侧 defaultMaxFailures = 5）。
// PG 与内存双实现共用（同包常量）。
const maxVerificationCodeFailures = 5

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		accountsByLogin: make(map[string]accountRecord),
		accountsByUser:  make(map[string]accountRecord),
		resetsByLogin:   make(map[string][]passwordResetRecord),
		registerCodes:   make(map[string][]registerCodeRecord),
	}
}

func (r *InMemoryRepository) Register(_ context.Context, params RegisterParams) (Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.accountsByLogin[params.LoginID]; exists {
		return Account{}, ErrLoginIDConflict
	}
	userID, err := generateUserID()
	if err != nil {
		return Account{}, err
	}
	hashed, err := hashPassword(params.Password)
	if err != nil {
		return Account{}, err
	}
	account := accountRecord{
		Account:           Account{UserID: userID, LoginID: params.LoginID, Nickname: params.Nickname, UserType: "normal"},
		PasswordHash:      hashed,
		RegisterIP:        params.RegisterIP,
		RegisterLocation:  params.RegisterLocation,
		LastLoginIP:       params.RegisterIP,
		LastLoginLocation: params.RegisterLocation,
	}
	r.accountsByLogin[params.LoginID] = account
	r.accountsByUser[userID] = account
	return account.Account, nil
}

func (r *InMemoryRepository) CreateGuest(_ context.Context, params GuestParams) (Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	userID, err := generateUserID()
	if err != nil {
		return Account{}, err
	}
	account := accountRecord{
		Account:           Account{UserID: userID, Nickname: params.Nickname, UserType: "temp"},
		RegisterIP:        params.RegisterIP,
		RegisterLocation:  params.RegisterLocation,
		LastLoginIP:       params.RegisterIP,
		LastLoginLocation: params.RegisterLocation,
	}
	r.accountsByUser[userID] = account
	return account.Account, nil
}

func (r *InMemoryRepository) Authenticate(_ context.Context, loginID, password string) (Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	account, ok := r.accountsByLogin[loginID]
	if !ok || !verifyPassword(account.PasswordHash, password) {
		return Account{}, ErrInvalidCredentials
	}
	return account.Account, nil
}

func (r *InMemoryRepository) UpdateLoginMeta(_ context.Context, userID, ip, location string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, ok := r.accountsByUser[userID]
	if !ok {
		return nil
	}
	account.LastLoginIP = ip
	account.LastLoginLocation = location
	r.accountsByUser[userID] = account
	if account.LoginID != "" {
		r.accountsByLogin[account.LoginID] = account
	}
	return nil
}

func (r *InMemoryRepository) CreatePasswordReset(_ context.Context, loginID string, expiresAt time.Time) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.accountsByLogin[loginID]; !ok {
		return "", nil
	}
	requestID, err := generateUserID()
	if err != nil {
		return "", err
	}
	code, err := generateVerificationCode()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	items := r.resetsByLogin[loginID]
	if n := len(items); n > 0 && now.Sub(items[n-1].CreatedAt) < passwordResetThrottleWindow {
		return "", ErrPasswordResetThrottled
	}
	r.resetsByLogin[loginID] = append(items, passwordResetRecord{
		RequestID:        requestID,
		LoginID:          loginID,
		VerificationCode: code,
		ExpiresAt:        expiresAt,
		CreatedAt:        now,
	})
	return code, nil
}

func (r *InMemoryRepository) ResetPassword(_ context.Context, loginID, verificationCode, newPassword string) (Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, ok := r.accountsByLogin[loginID]
	if !ok {
		return Account{}, ErrPasswordResetNotFound
	}
	now := time.Now().UTC()
	requestIndex := -1
	for i := len(r.resetsByLogin[loginID]) - 1; i >= 0; i-- {
		candidate := r.resetsByLogin[loginID][i]
		if candidate.UsedAt != nil || now.After(candidate.ExpiresAt) || candidate.VerificationCode != verificationCode {
			continue
		}
		requestIndex = i
		break
	}
	if requestIndex == -1 {
		// 验码失败即计数，满阈值静默作废最新有效码（对齐 PG 实现）。
		r.burnResetCodeOnFailure(loginID, now)
		return Account{}, ErrPasswordResetNotFound
	}
	hashed, err := hashPassword(newPassword)
	if err != nil {
		return Account{}, err
	}
	account.PasswordHash = hashed
	r.accountsByLogin[loginID] = account
	r.accountsByUser[account.UserID] = account
	usedAt := now
	request := r.resetsByLogin[loginID][requestIndex]
	request.UsedAt = &usedAt
	r.resetsByLogin[loginID][requestIndex] = request
	return account.Account, nil
}

func (r *InMemoryRepository) ChangePassword(_ context.Context, userID, oldPassword, newPassword string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, ok := r.accountsByUser[userID]
	if !ok || !verifyPassword(account.PasswordHash, oldPassword) {
		return ErrInvalidOldPassword
	}
	hashed, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	account.PasswordHash = hashed
	r.accountsByUser[userID] = account
	r.accountsByLogin[account.LoginID] = account
	return nil
}

func (r *InMemoryRepository) LastResetCode(loginID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.resetsByLogin[loginID]
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].UsedAt == nil {
			return items[i].VerificationCode
		}
	}
	return ""
}

// CreateRegisterCode 生成并落库注册验证码（不校验账号存在性，防枚举）。
func (r *InMemoryRepository) CreateRegisterCode(_ context.Context, loginID string, expiresAt time.Time) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	requestID, err := generateUserID()
	if err != nil {
		return "", err
	}
	code, err := generateVerificationCode()
	if err != nil {
		return "", err
	}
	r.registerCodes[loginID] = append(r.registerCodes[loginID], registerCodeRecord{
		RequestID:        requestID,
		LoginID:          loginID,
		VerificationCode: code,
		ExpiresAt:        expiresAt,
		CreatedAt:        time.Now().UTC(),
	})
	return code, nil
}

// VerifyRegisterCode 校验注册验证码有效（存在、未消费、未过期），不消费；
// 失败路径计数，满阈值静默作废最新有效码（对齐 PG 实现）。
func (r *InMemoryRepository) VerifyRegisterCode(_ context.Context, loginID, verificationCode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	items := r.registerCodes[loginID]
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.ConsumedAt == nil && !now.After(item.ExpiresAt) && item.VerificationCode == verificationCode {
			return nil
		}
	}
	r.burnRegisterCodeOnFailure(loginID, now)
	return ErrInvalidRegisterCode
}

// ConsumeRegisterCode 将验证码标记为已消费（一次性，幂等）。
func (r *InMemoryRepository) ConsumeRegisterCode(_ context.Context, loginID, verificationCode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	items := r.registerCodes[loginID]
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].ConsumedAt == nil && items[i].VerificationCode == verificationCode {
			record := items[i]
			consumedAt := now
			record.ConsumedAt = &consumedAt
			items[i] = record
			return nil
		}
	}
	return nil
}

// LastRegisterCode 测试辅助：返回 login_id 最新的未消费注册验证码。
func (r *InMemoryRepository) LastRegisterCode(loginID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.registerCodes[loginID]
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].ConsumedAt == nil {
			return items[i].VerificationCode
		}
	}
	return ""
}

// burnResetCodeOnFailure 给 login_id 最新有效重置码 +1 次失败，满
// maxVerificationCodeFailures 次即作废该码。调用方须持有 r.mu 写锁。
func (r *InMemoryRepository) burnResetCodeOnFailure(loginID string, now time.Time) {
	items := r.resetsByLogin[loginID]
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].UsedAt != nil || now.After(items[i].ExpiresAt) {
			continue
		}
		items[i].FailedAttempts++
		if items[i].FailedAttempts >= maxVerificationCodeFailures {
			usedAt := now
			items[i].UsedAt = &usedAt
		}
		return
	}
}

// burnRegisterCodeOnFailure 注册码同款失败计数 + 烧码。调用方须持有 r.mu 写锁。
func (r *InMemoryRepository) burnRegisterCodeOnFailure(loginID string, now time.Time) {
	items := r.registerCodes[loginID]
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].ConsumedAt != nil || now.After(items[i].ExpiresAt) {
			continue
		}
		items[i].FailedAttempts++
		if items[i].FailedAttempts >= maxVerificationCodeFailures {
			consumedAt := now
			items[i].ConsumedAt = &consumedAt
		}
		return
	}
}
