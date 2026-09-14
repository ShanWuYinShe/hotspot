package media

import (
	"context"
	"time"
)

type Provider interface {
	Presign(context.Context, PresignFileInput, string) (PresignedUpload, error)
	PresignGet(context.Context, string, time.Duration) (string, error)
	Head(context.Context, string) (ObjectMetadata, error)
	// ReadPrefix 读取对象前 n 字节（Range GET），供 complete 阶段魔数嗅探
	// （spec api_spec_media.md：Content-Type 由客户端声明可伪造，须以内容为准）。
	ReadPrefix(ctx context.Context, storageKey string, n int64) ([]byte, error)
}
