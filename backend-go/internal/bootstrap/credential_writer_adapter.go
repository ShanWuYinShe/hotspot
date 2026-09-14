package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"hotspot/backend-go/internal/modules/auth"
	"hotspot/backend-go/internal/modules/users"
)

// upgradeCredentialWriter adapts auth.Service to the users.AccountCredentialWriter
// interface (spec api_spec_user.md → upgrade 契约级语义：升级须把 login_id+password
// 落登录凭证，否则升级用户无法重新登录)。适配器把 auth 的 login_id 占用错误转译为
// users.ErrLoginIDTaken（10400），避免 users 模块反向依赖 auth 的错误类型。
type upgradeCredentialWriter struct{ svc *auth.Service }

func (w upgradeCredentialWriter) AttachCredentials(ctx context.Context, userID, loginID, password string) error {
	if err := w.svc.AttachCredentials(ctx, userID, loginID, password); err != nil {
		if errors.Is(err, auth.ErrLoginIDConflict) {
			return fmt.Errorf("upgrade credential: %w", users.ErrLoginIDTaken)
		}
		return err
	}
	return nil
}
