package groupmessages

import "errors"

var (
	ErrMessageNotFound     = errors.New("group message not found")
	ErrPermissionDenied    = errors.New("权限不足")
	ErrMessageRecalled     = errors.New("message already recalled")
	ErrMessageNotEditable  = errors.New("message not editable")
	ErrMessageRecallWindow = errors.New("recall window expired")
	ErrInvalidCursor       = errors.New("groupmessages: invalid cursor")
)
