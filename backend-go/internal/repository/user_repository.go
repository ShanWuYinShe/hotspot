package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"backend-go/internal/domain"
)

// UserRepository 封装用户相关的持久化操作。
type UserRepository interface {
	CreateUser(ctx context.Context, u *domain.User) error
	FindByLoginID(ctx context.Context, loginID string) (*domain.User, error)
	FindByID(ctx context.Context, userID string) (*domain.User, error)
	UpdatePasswordAndType(ctx context.Context, userID string, passwordHash *string, userType string) error
}

type userRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) CreateUser(ctx context.Context, u *domain.User) error {
	const q = `
INSERT INTO users (user_id, login_id, nickname, password_hash, user_type, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`
	_, err := r.pool.Exec(ctx, q, u.UserID, u.LoginID, u.Nickname, u.Password, u.UserType, u.CreatedAt, u.UpdatedAt)
	return err
}

func (r *userRepository) FindByLoginID(ctx context.Context, loginID string) (*domain.User, error) {
	const q = `
SELECT user_id, login_id, nickname, password_hash, user_type, created_at, updated_at
FROM users
WHERE login_id = $1
`
	row := r.pool.QueryRow(ctx, q, loginID)
	var u domain.User
	if err := row.Scan(&u.UserID, &u.LoginID, &u.Nickname, &u.Password, &u.UserType, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, userID string) (*domain.User, error) {
	const q = `
SELECT user_id, login_id, nickname, password_hash, user_type, created_at, updated_at
FROM users
WHERE user_id = $1
`
	row := r.pool.QueryRow(ctx, q, userID)
	var u domain.User
	if err := row.Scan(&u.UserID, &u.LoginID, &u.Nickname, &u.Password, &u.UserType, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) UpdatePasswordAndType(ctx context.Context, userID string, passwordHash *string, userType string) error {
	const q = `
UPDATE users
SET password_hash = $2, user_type = $3, updated_at = NOW()
WHERE user_id = $1
`
	_, err := r.pool.Exec(ctx, q, userID, passwordHash, userType)
	return err
}
