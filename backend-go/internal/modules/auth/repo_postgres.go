package auth

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory, _ Repository) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Register(ctx context.Context, params RegisterParams) (Account, error) {
	queryable, ok := r.queryable()
	if !ok {
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
	if err := queryable.Exec(ctx, queryInsertUserProfile, userID, params.Nickname, params.Defaults.AvatarURL, "normal", params.LoginID, params.Defaults.LocationName, params.Defaults.Latitude, params.Defaults.Longitude, now, now, params.RegisterIP, params.RegisterLocation, params.RegisterIP, params.RegisterLocation); err != nil {
		return Account{}, err
	}
	if err := queryable.Exec(ctx, queryInsertAuthAccount, params.LoginID, userID, hashed, now, now); err != nil {
		if isUniqueViolation(err) {
			_ = queryable.Exec(ctx, queryDeleteUserProfileByUserID, userID)
			return Account{}, ErrLoginIDConflict
		}
		_ = queryable.Exec(ctx, queryDeleteUserProfileByUserID, userID)
		return Account{}, err
	}
	if err := queryable.Exec(ctx, queryUpsertUserSettings, userID); err != nil {
		_ = queryable.Exec(ctx, queryDeleteAuthAccountByLoginID, params.LoginID)
		_ = queryable.Exec(ctx, queryDeleteUserProfileByUserID, userID)
		return Account{}, err
	}
	return Account{UserID: userID, LoginID: params.LoginID, Nickname: params.Nickname, UserType: "normal"}, nil
}

func (r *PostgresRepository) CreateGuest(ctx context.Context, params GuestParams) (Account, error) {
	queryable, ok := r.queryable()
	if !ok {
		return Account{}, persistence.ErrNotImplemented
	}
	userID, err := generateUserID()
	if err != nil {
		return Account{}, err
	}
	now := params.Defaults.CreatedAt.UTC()
	if err := queryable.Exec(ctx, queryInsertUserProfile, userID, params.Nickname, params.Defaults.AvatarURL, "temp", "", params.Defaults.LocationName, params.Defaults.Latitude, params.Defaults.Longitude, now, now, params.RegisterIP, params.RegisterLocation, params.RegisterIP, params.RegisterLocation); err != nil {
		return Account{}, err
	}
	if err := queryable.Exec(ctx, queryUpsertUserSettings, userID); err != nil {
		_ = queryable.Exec(ctx, queryDeleteUserProfileByUserID, userID)
		return Account{}, err
	}
	return Account{UserID: userID, Nickname: params.Nickname, UserType: "temp"}, nil
}

func (r *PostgresRepository) Authenticate(ctx context.Context, loginID, password string) (Account, error) {
	queryable, ok := r.queryable()
	if !ok {
		return Account{}, persistence.ErrNotImplemented
	}
	record, found, err := r.accountByLoginID(ctx, queryable, loginID)
	if err != nil {
		return Account{}, err
	}
	if !found || !verifyPassword(record.PasswordHash, password) {
		return Account{}, ErrInvalidCredentials
	}
	return record.Account, nil
}

func (r *PostgresRepository) UpdateLoginMeta(ctx context.Context, userID, ip, location string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateLoginMeta, userID, ip, location)
}

func (r *PostgresRepository) CreatePasswordReset(ctx context.Context, loginID string, expiresAt time.Time) (string, error) {
	queryable, ok := r.queryable()
	if !ok {
		return "", persistence.ErrNotImplemented
	}
	if _, found, err := r.accountByLoginID(ctx, queryable, loginID); err != nil {
		return "", err
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
		return "", err
	} else if found && now.Sub(latest.CreatedAt) < passwordResetThrottleWindow {
		return "", ErrPasswordResetThrottled
	}
	if err := queryable.Exec(ctx, queryInsertPasswordReset, requestID, loginID, code, expiresAt.UTC(), now); err != nil {
		return "", err
	}
	return code, nil
}

func (r *PostgresRepository) ResetPassword(ctx context.Context, loginID, verificationCode, newPassword string) (Account, error) {
	queryable, ok := r.queryable()
	if !ok {
		return Account{}, persistence.ErrNotImplemented
	}
	account, found, err := r.accountByLoginID(ctx, queryable, loginID)
	if err != nil {
		return Account{}, err
	}
	if !found {
		return Account{}, ErrPasswordResetNotFound
	}
	request, ok, err := r.passwordResetByCode(ctx, queryable, loginID, verificationCode)
	if err != nil {
		return Account{}, err
	}
	now := time.Now().UTC()
	if !ok || request.UsedAt != nil || now.After(request.ExpiresAt) {
		return Account{}, ErrPasswordResetNotFound
	}
	hashed, err := hashPassword(newPassword)
	if err != nil {
		return Account{}, err
	}
	if err := queryable.Exec(ctx, queryUpdateAccountPassword, account.UserID, hashed, now); err != nil {
		return Account{}, err
	}
	if err := queryable.Exec(ctx, queryUsePasswordReset, request.RequestID, now); err != nil {
		return Account{}, err
	}
	return account.Account, nil
}

func (r *PostgresRepository) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	queryable, ok := r.queryable()
	if !ok {
		return persistence.ErrNotImplemented
	}
	account, found, err := r.accountByUserID(ctx, queryable, userID)
	if err != nil {
		return err
	}
	if !found || !verifyPassword(account.PasswordHash, oldPassword) {
		return ErrInvalidOldPassword
	}
	hashed, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	return queryable.Exec(ctx, queryUpdateAccountPassword, userID, hashed, time.Now().UTC())
}

func (r *PostgresRepository) accountByLoginID(ctx context.Context, queryable persistence.QueryableDB, loginID string) (accountRecord, bool, error) {
	var record accountRecord
	if err := queryable.QueryRow(ctx, queryGetAccountByLoginID, loginID).Scan(&record.LoginID, &record.UserID, &record.PasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accountRecord{}, false, nil
		}
		return accountRecord{}, false, err
	}
	return record.withDerivedFields(), true, nil
}

func (r *PostgresRepository) accountByUserID(ctx context.Context, queryable persistence.QueryableDB, userID string) (accountRecord, bool, error) {
	var record accountRecord
	if err := queryable.QueryRow(ctx, queryGetAccountByUserID, userID).Scan(&record.LoginID, &record.UserID, &record.PasswordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accountRecord{}, false, nil
		}
		return accountRecord{}, false, err
	}
	return record.withDerivedFields(), true, nil
}

func (r *PostgresRepository) passwordResetByCode(ctx context.Context, queryable persistence.QueryableDB, loginID, verificationCode string) (passwordResetRecord, bool, error) {
	var record passwordResetRecord
	if err := queryable.QueryRow(ctx, queryGetPasswordReset, loginID, verificationCode).Scan(&record.RequestID, &record.LoginID, &record.VerificationCode, &record.ExpiresAt, &record.CreatedAt, &record.UsedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return passwordResetRecord{}, false, nil
		}
		return passwordResetRecord{}, false, err
	}
	return record, true, nil
}

func (r *PostgresRepository) latestPasswordResetByLoginID(ctx context.Context, queryable persistence.QueryableDB, loginID string) (passwordResetRecord, bool, error) {
	var record passwordResetRecord
	if err := queryable.QueryRow(ctx, queryLatestPasswordResetByLoginID, loginID).Scan(&record.RequestID, &record.LoginID, &record.VerificationCode, &record.ExpiresAt, &record.CreatedAt, &record.UsedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return passwordResetRecord{}, false, nil
		}
		return passwordResetRecord{}, false, err
	}
	return record, true, nil
}

func (r *PostgresRepository) queryable() (persistence.QueryableDB, bool) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, false
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	return queryable, ok
}

func (r accountRecord) withDerivedFields() accountRecord {
	r.Account = Account{UserID: r.UserID, LoginID: r.LoginID, Nickname: r.LoginID, UserType: "normal"}
	return r
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
