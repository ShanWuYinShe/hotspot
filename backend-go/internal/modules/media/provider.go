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
	// Delete 删除对象（幂等，404 视为已删），保留期 GC 清孤儿文件用。
	Delete(ctx context.Context, storageKey string) error
	// GetObject 读取完整对象（EXIF 清洗链路用）。
	GetObject(ctx context.Context, storageKey string) ([]byte, error)
	// PutObject 以指定 Content-Type 覆写对象（清洗后回写，键不变）。
	PutObject(ctx context.Context, storageKey, contentType string, data []byte) error
}
