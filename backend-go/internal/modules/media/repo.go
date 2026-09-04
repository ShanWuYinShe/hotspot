package media

import "context"

type pendingUpload struct {
	UploadID                string
	UserID                  string
	StorageKey              string
	FileName                string
	DeclaredFileSize        int
	DeclaredMimeType        string
	Category                MediaCategory
	MD5                     string
	Completed               bool
	File                    UploadedMedia
	CompletedIdempotencyKey string
	CreatedAt               string
	CompletedAt             string
}

type Repository interface {
	Save(context.Context, pendingUpload) error
	Get(context.Context, string) (pendingUpload, bool, error)
	Complete(context.Context, string, UploadedMedia, string) (pendingUpload, bool, error)
	// ResolveByFileIDs returns the completed uploads owned by userID for the
	// given file_ids, filtered to the given category. Only rows with
	// status='completed' are returned. Unmatched file_ids are simply absent
	// from the result; callers treat absence as an authorization failure.
	ResolveByFileIDs(context.Context, []string, string, MediaCategory) (map[string]MediaAsset, error)
	// CompletedFileMetaByID returns display metadata for a completed upload by
	// its file_id, regardless of owner/category — report snapshots need the
	// file truth value, not ownership scoping. found=false when the file does
	// not exist or is not in status='completed'.
	CompletedFileMetaByID(context.Context, string) (FileMeta, bool, error)
}
