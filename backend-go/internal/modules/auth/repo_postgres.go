package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Register(ctx context.Context, params RegisterParams) (Account, error) {
	if _, ok := r.queryable(ctx); !ok {
		return Account{}, persistence.ErrNotImplemented
	}
	userID, err := generateUserID()
	if err != nil {
		return Account{}, err
	}
	hashed, err := hashPassword(params.Password)
	if err != nil {
		return Account{}, err
	}
	now := params.Defaults.CreatedAt.UTC()
	s2CellID := geo.LeafCellID(geo.Point{Latitude: params.Defaults.Latitude, Longitude: params.Defaults.Longitude})
	// Wrap all three inserts (user profile, auth account, settings) in a single
	// transaction. Previously these were separate Execs with manual compensating
	// deletes on failure, which is racy under concurrent registration and leaks
	// partial rows if the process dies mid-write. Tx rollback atomically undoes
	// all three. RunInTransaction is a no-op for in-memory factories (tests).
	var conflict bool
	var execErr error
	txErr := persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := queryable.Exec(txCtx, queryInsertUserProfile, userID, params.Nickname, params.Defaults.AvatarURL, "normal", params.LoginID, params.Defaults.LocationName, params.Defaults.Latitude, params.Defaults.Longitude, s2CellID, now, now, params.RegisterIP, params.RegisterLocation, params.RegisterIP, params.RegisterLocation); err != nil {
			execErr = err
			return err
		}
		if err := queryable.Exec(txCtx, queryInsertAuthAccount, params.LoginID, userID, hashed, now, now); err != nil {
			if persistence.IsUniqueViolation(err) {
				conflict = true
			}
			execErr = err
			return err
		}
		if err := queryable.Exec(txCtx, queryUpsertUserSettings, userID); err != nil {
			execErr = err
			return err
		}
		return nil
	})
	if txErr != nil {
		if conflict || persistence.IsUniqueViolation(txErr) {
			return Account{}, ErrLoginIDConflict
		}
		if execErr != nil {
			return Account{}, fmt.Errorf("auth.Register: %w", execErr)
		}
		return Account{}, fmt.Errorf("auth.Register: %w", txErr)
	}
	return Account{UserID: userID, LoginID: params.LoginID, Nickname: params.Nickname, UserType: "normal"}, nil
}

// queryableForTx returns the queryable bound to ctx (an in-flight transaction)
// when one is present, otherwise falls back to the shared pool. This lets the
// same repo method run against either a tx (when RunInTransaction is active) or
// the pool, without the caller caring which.

// AttachCredential 为既有用户追加登录凭证（users/upgrade 复用注册的凭证写入：
// 同一张 auth_accounts 表、同一 bcrypt 口径）。INSERT 撞 login_id 唯一约束时
// 返回 ErrLoginIDConflict（users 侧映射 10400）；ctx 携带事务时自动加入该事务，
// 使凭证与 user_type 同事务落库。
func (r *PostgresRepository) AttachCredential(ctx context.Context, userID, loginID, password string) error {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return persistence.ErrNotImplemented
	}
	hashed, err := hashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryInsertAuthAccount, loginID, userID, hashed, now, now); err != nil {
		if persistence.IsUniqueViolation(err) {
			return ErrLoginIDConflict
		}
		return fmt.Errorf("auth.AttachCredential: insert: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateGuest(ctx context.Context, params GuestParams) (Account, error) {
	userID, err := generateUserID()
	if err != nil {
		return Account{}, err
	}
	now := params.Defaults.CreatedAt.UTC()
	s2CellID := geo.LeafCellID(geo.Point{Latitude: params.Defaults.Latitude, Longitude: params.Defaults.Longitude})
	// profile + settings 两步包进同一事务（替代原补偿式删除——进程中途
	// 崩溃时补偿不会执行，会泄漏半创建用户）。
	if err := persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryInsertUserProfile, userID, params.Nickname, params.Defaults.AvatarURL, "temp", "", params.Defaults.LocationName, params.Defaults.Latitude, params.Defaults.Longitude, s2CellID, now, now, params.RegisterIP, params.RegisterLocation, params.RegisterIP, params.RegisterLocation); err != nil {
			return fmt.Errorf("auth.CreateGuest: profile insert: %w", err)
		}
		if err := txQueryable.Exec(txCtx, queryUpsertUserSettings, userID); err != nil {
			return fmt.Errorf("auth.CreateGuest: settings upsert: %w", err)
		}
		return nil
	}); err != nil {
		return Account{}, err
	}
	return Account{UserID: userID, Nickname: params.Nickname, UserType: "temp"}, nil
}

func (r *PostgresRepository) Authenticate(ctx context.Context, loginID, password string) (Account, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return Account{}, persistence.ErrNotImplemented
	}
	record, found, err := r.accountByLoginID(ctx, queryable, loginID)
	if err != nil {
		return Account{}, fmt.Errorf("auth.Authenticate: %w", err)
	}
	if !found || !verifyPassword(record.PasswordHash, password) {
		return Account{}, ErrInvalidCredentials
	}
	return record.Account, nil
}

func (r *PostgresRepository) UpdateLoginMeta(ctx context.Context, userID, ip, location string) error {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, queryUpdateLoginMeta, userID, ip, location); err != nil {
		return fmt.Errorf("auth.UpdateLoginMeta: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreatePasswordReset(ctx context.Context, loginID string, expiresAt time.Time) (string, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return "", persistence.ErrNotImplemented
	}
	if _, found, err := r.accountByLoginID(ctx, queryable, loginID); err != nil {
		return "", fmt.Errorf("auth.CreatePasswordReset: account lookup: %w", err)
	} else if !found {
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
	if latest, found, err := r.latestPasswordResetByLoginID(ctx, queryable, loginID); err != nil {
		return "", fmt.Errorf("auth.CreatePasswordReset: throttle check: %w", err)
	} else if found && now.Sub(latest.CreatedAt) < passwordResetThrottleWindow {
		return "", ErrPasswordResetThrottled
	}
	if err := queryable.Exec(ctx, queryInsertPasswordReset, requestID, loginID, code, expiresAt.UTC(), now); err != nil {
		return "", fmt.Errorf("auth.CreatePasswordReset: insert: %w", err)
	}
	return code, nil
}

func (r *PostgresRepository) ResetPassword(ctx context.Context, loginID, verificationCode, newPassword string) (Account, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return Account{}, persistence.ErrNotImplemented
	}
	account, found, err := r.accountByLoginID(ctx, queryable, loginID)
	if err != nil {
		return Account{}, fmt.Errorf("auth.ResetPassword: account lookup: %w", err)
	}
	if !found {
		return Account{}, ErrPasswordResetNotFound
	}
	request, ok, err := r.passwordResetByCode(ctx, queryable, loginID, verificationCode)
	if err != nil {
		return Account{}, fmt.Errorf("auth.ResetPassword: reset code lookup: %w", err)
	}
	now := time.Now().UTC()
	if !ok || request.UsedAt != nil || now.After(request.ExpiresAt) {
		// 验码失败即计数（防爆破：满 5 次静默作废最新有效码，对外仍返回
		// 原错误，不新增可探测口径；见 maxVerificationCodeFailures）。
		if err := r.recordResetCodeFailure(ctx, queryable, loginID, now); err != nil {
			return Account{}, fmt.Errorf("auth.ResetPassword: record failure: %w", err)
		}
		return Account{}, ErrPasswordResetNotFound
	}
	hashed, err := hashPassword(newPassword)
	if err != nil {
		return Account{}, err
	}
	// 改密码 + 标记 reset 已用两步包进同一事务：避免「密码已改但 reset
	// 记录未消耗」导致旧验证码可重放。
	if err := persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryUpdateAccountPassword, account.UserID, hashed, now); err != nil {
			return fmt.Errorf("auth.ResetPassword: password update: %w", err)
		}
		if err := txQueryable.Exec(txCtx, queryUsePasswordReset, request.RequestID, now); err != nil {
			return fmt.Errorf("auth.ResetPassword: mark reset used: %w", err)
		}
		return nil
	}); err != nil {
		return Account{}, err
	}
	return account.Account, nil
}

func (r *PostgresRepository) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return persistence.ErrNotImplemented
	}
	account, found, err := r.accountByUserID(ctx, queryable, userID)
	if err != nil {
		return fmt.Errorf("auth.ChangePassword: account lookup: %w", err)
	}
	if !found || !verifyPassword(account.PasswordHash, oldPassword) {
		return ErrInvalidOldPassword
	}
	hashed, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := queryable.Exec(ctx, queryUpdateAccountPassword, userID, hashed, time.Now().UTC()); err != nil {
		return fmt.Errorf("auth.ChangePassword: password update: %w", err)
	}
	return nil
}

// CreateRegisterCode 生成并落库注册验证码。与密码重置不同：注册码不校验账号
// 是否存在（login_id 可能尚未注册，不 FK 账号表），防枚举由「无论是否存在均
// 落库 + 发送」实现；频率控制由 IP 维度 ratelimit 桶 + per-login_id 节流窗口
// （registerCodeThrottleWindow，与密码重置的 repo 级节流同模式）双重兜底。
func (r *PostgresRepository) CreateRegisterCode(ctx context.Context, loginID string, expiresAt time.Time) (string, error) {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return "", persistence.ErrNotImplemented
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
	// per-login_id 发送节流：IP 桶可被代理池绕过，同一手机号/邮箱的
	// 验证码轰炸（骚扰 + 短信成本）须在 repo 层按 login_id 兜底。
	if latest, err := r.latestRegisterCodeCreatedAt(ctx, queryable, loginID); err != nil {
		return "", fmt.Errorf("auth.CreateRegisterCode: throttle check: %w", err)
	} else if latest != nil && now.Sub(*latest) < registerCodeThrottleWindow {
		return "", ErrRegisterCodeThrottled
	}
	if err := queryable.Exec(ctx, queryInsertRegisterCode, requestID, loginID, code, expiresAt.UTC(), now); err != nil {
		return "", fmt.Errorf("auth.CreateRegisterCode: insert: %w", err)
	}
	return code, nil
}

// latestRegisterCodeCreatedAt 返回该 login_id 最近一条注册验证码的落库时间；
// 无记录时返回 (nil, nil)。
func (r *PostgresRepository) latestRegisterCodeCreatedAt(ctx context.Context, queryable persistence.Queryable, loginID string) (*time.Time, error) {
	var createdAt time.Time
	err := queryable.QueryRow(ctx, queryLatestRegisterCodeCreatedAt, loginID).Scan(&createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &createdAt, nil
}

// VerifyRegisterCode 校验注册验证码有效（存在、未消费、未过期），不消费——
// 消费发生在注册成功后（ConsumeRegisterCode）；失败路径只计数，满阈值才烧码。
func (r *PostgresRepository) VerifyRegisterCode(ctx context.Context, loginID, verificationCode string) error {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var requestID string
	if err := queryable.QueryRow(ctx, queryValidRegisterCode, loginID, verificationCode, time.Now().UTC()).Scan(&requestID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if ferr := r.recordRegisterCodeFailure(ctx, queryable, loginID, time.Now().UTC()); ferr != nil {
				return fmt.Errorf("auth.VerifyRegisterCode: record failure: %w", ferr)
			}
			return ErrInvalidRegisterCode
		}
		return fmt.Errorf("auth.VerifyRegisterCode: lookup: %w", err)
	}
	return nil
}

// recordResetCodeFailure 给 login_id 最新有效重置码 +1 次失败，满
// maxVerificationCodeFailures 次即作废该码（烧码后正确码同样失效，调用方
// 必须重新发送）。无有效码时为 no-op。
func (r *PostgresRepository) recordResetCodeFailure(ctx context.Context, queryable persistence.Queryable, loginID string, now time.Time) error {
	var requestID string
	var failures int
	if err := queryable.QueryRow(ctx, queryBumpResetCodeFailure, loginID, now).Scan(&requestID, &failures); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("auth.recordResetCodeFailure: bump: %w", err)
	}
	if failures >= maxVerificationCodeFailures {
		if err := queryable.Exec(ctx, queryBurnResetCode, requestID, now); err != nil {
			return fmt.Errorf("auth.recordResetCodeFailure: burn: %w", err)
		}
	}
	return nil
}

// recordRegisterCodeFailure 注册码同款失败计数 + 烧码。
func (r *PostgresRepository) recordRegisterCodeFailure(ctx context.Context, queryable persistence.Queryable, loginID string, now time.Time) error {
	var requestID string
	var failures int
	if err := queryable.QueryRow(ctx, queryBumpRegisterCodeFailure, loginID, now).Scan(&requestID, &failures); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("auth.recordRegisterCodeFailure: bump: %w", err)
	}
	if failures >= maxVerificationCodeFailures {
		if err := queryable.Exec(ctx, queryBurnRegisterCode, requestID, now); err != nil {
			return fmt.Errorf("auth.recordRegisterCodeFailure: burn: %w", err)
		}
	}
	return nil
}

// ConsumeRegisterCode 将验证码标记为已消费（一次性）。UPDATE 天然幂等：
// 已消费的码再消费为 no-op。
func (r *PostgresRepository) ConsumeRegisterCode(ctx context.Context, loginID, verificationCode string) error {
	queryable, ok := r.queryable(ctx)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, queryConsumeRegisterCode, loginID, verificationCode, time.Now().UTC()); err != nil {
		return fmt.Errorf("auth.ConsumeRegisterCode: update: %w", err)
	}
	return nil
}

func (r *PostgresRepository) accountByLoginID(ctx context.Context, queryable persistence.Queryable, loginID string) (accountRecord, bool, error) {
	var record accountRecord
	if err := queryable.QueryRow(ctx, queryGetAccountByLoginID, loginID).Scan(&record.LoginID, &record.UserID, &record.PasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accountRecord{}, false, nil
		}
		return accountRecord{}, false, fmt.Errorf("auth.accountByLoginID scan: %w", err)
	}
	return record.withDerivedFields(), true, nil
}

func (r *PostgresRepository) accountByUserID(ctx context.Context, queryable persistence.Queryable, userID string) (accountRecord, bool, error) {
	var record accountRecord
	if err := queryable.QueryRow(ctx, queryGetAccountByUserID, userID).Scan(&record.LoginID, &record.UserID, &record.PasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accountRecord{}, false, nil
		}
		return accountRecord{}, false, fmt.Errorf("auth.accountByUserID scan: %w", err)
	}
	return record.withDerivedFields(), true, nil
}

func (r *PostgresRepository) passwordResetByCode(ctx context.Context, queryable persistence.Queryable, loginID, verificationCode string) (passwordResetRecord, bool, error) {
	var record passwordResetRecord
	if err := queryable.QueryRow(ctx, queryGetPasswordReset, loginID, verificationCode).Scan(&record.RequestID, &record.LoginID, &record.VerificationCode, &record.ExpiresAt, &record.CreatedAt, &record.UsedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return passwordResetRecord{}, false, nil
		}
		return passwordResetRecord{}, false, fmt.Errorf("auth.passwordResetByCode scan: %w", err)
	}
	return record, true, nil
}

func (r *PostgresRepository) latestPasswordResetByLoginID(ctx context.Context, queryable persistence.Queryable, loginID string) (passwordResetRecord, bool, error) {
	var record passwordResetRecord
	if err := queryable.QueryRow(ctx, queryLatestPasswordResetByLoginID, loginID).Scan(&record.RequestID, &record.LoginID, &record.VerificationCode, &record.ExpiresAt, &record.CreatedAt, &record.UsedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return passwordResetRecord{}, false, nil
		}
		return passwordResetRecord{}, false, fmt.Errorf("auth.latestPasswordResetByLoginID scan: %w", err)
	}
	return record, true, nil
}

func (r *PostgresRepository) queryable(ctx context.Context) (persistence.Queryable, bool) {
	return persistence.RepoQueryable(ctx, r.factory)
}

func (r accountRecord) withDerivedFields() accountRecord {
	r.Account = Account{UserID: r.UserID, LoginID: r.LoginID, Nickname: r.LoginID, UserType: "normal"}
	return r
}
