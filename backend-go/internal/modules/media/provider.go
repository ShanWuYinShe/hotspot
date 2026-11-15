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
	// PutObject 以指定 Content-Type 覆写对象（发布清洗后的图片字节用）。
	PutObject(ctx context.Context, storageKey, contentType string, data []byte) error
	// CopyObject 在存储端把 srcKey 复制到 dstKey（copy-on-complete 发布用），
	// 不把大对象读进应用内存。失败返回错误，调用方不得提交完成态。
	CopyObject(ctx context.Context, srcKey, dstKey string) error
	// ListObjects 按前缀列举对象（S3 ListObjectsV2 单页语义）：prefix 为对象
	// 键前缀；continuationToken 为空表示首页，非空时原样回传上一页的
	// NextContinuationToken；maxKeys 为单页上限（越界由实现夹紧）。保留期
	// sweeper 的 uploads/ 孤儿清理（retention.go）用它枚举对象。
	ListObjects(ctx context.Context, prefix, continuationToken string, maxKeys int) (ObjectListPage, error)
}
