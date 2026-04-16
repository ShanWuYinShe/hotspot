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
}

type InMemoryRepository struct {
	mu              sync.RWMutex
	accountsByLogin map[string]accountRecord
	accountsByUser  map[string]accountRecord
	resetsByLogin   map[string][]passwordResetRecord
}

const passwordResetThrottleWindow = 10 * time.Minute

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		accountsByLogin: make(map[string]accountRecord),
		accountsByUser:  make(map[string]accountRecord),
		resetsByLogin:   make(map[string][]passwordResetRecord),
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
