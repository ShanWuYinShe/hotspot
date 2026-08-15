package moments

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubImageFetcher struct {
	rows map[string][]momentImageRow
	err  error
}

func (f stubImageFetcher) FetchMomentImageRows(context.Context, []string) (map[string][]momentImageRow, error) {
	return f.rows, f.err
}

type stubImageResolver struct {
	urls map[string]string
	err  error
}

func (r stubImageResolver) ResolveMomentImages(context.Context, []string, string) (map[string]ImageAsset, error) {
	return nil, nil
}

func (r stubImageResolver) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	if u, ok := r.urls[key]; ok {
		return u, nil
	}
	return "https://cdn.example.com/" + key, nil
}

func TestHydrateMomentImages(t *testing.T) {
	ctx := context.Background()
	fetcher := stubImageFetcher{rows: map[string][]momentImageRow{
		"m1": {
			{StorageKey: "key1", ThumbnailURL: "thumb1", Width: 100, Height: 200, MimeType: "image/jpeg"},
			{StorageKey: "key2", ThumbnailURL: "", Width: 0, Height: 0, MimeType: ""},
		},
	}}
	resolver := stubImageResolver{urls: map[string]string{"thumb1": "https://cdn.example.com/thumb1"}}

	// 空列表直接返回
	HydrateMomentImages(ctx, fetcher, resolver, nil)

	items := []MomentItem{{MomentID: "m1"}, {MomentID: "m2"}}
	HydrateMomentImages(ctx, fetcher, resolver, items)
	if len(items[0].Images) != 2 {
		t.Fatalf("expected 2 images, got %#v", items[0].Images)
	}
	if items[0].Images[0].URL != "https://cdn.example.com/key1" || *items[0].Images[0].ThumbnailURL != "https://cdn.example.com/thumb1" {
		t.Fatalf("unexpected image[0]: %#v", items[0].Images[0])
	}
	if *items[0].Images[1].ThumbnailURL != "https://cdn.example.com/key2" {
		t.Fatalf("missing thumbnail should fall back to url: %#v", items[0].Images[1])
	}
	if len(items[1].Images) != 0 {
		t.Fatalf("m2 has no images: %#v", items[1].Images)
	}

	// fetch 错误 → 静默跳过
	HydrateMomentImages(ctx, stubImageFetcher{err: errors.New("boom")}, resolver, items)

	// nil resolver → 跳过
	HydrateMomentImages(ctx, fetcher, nil, items)

	// presign 错误 → 该图片被跳过（使用干净 items，避免上次结果残留）
	failingResolver := stubImageResolver{err: errors.New("presign failed")}
	fresh := []MomentItem{{MomentID: "m1"}}
	HydrateMomentImages(ctx, fetcher, failingResolver, fresh)
	if len(fresh[0].Images) != 0 {
		t.Fatalf("presign failures must drop images, got %#v", fresh[0].Images)
	}
}

func TestHydrateSingleMoment(t *testing.T) {
	ctx := context.Background()
	fetcher := stubImageFetcher{rows: map[string][]momentImageRow{
		"m1": {{StorageKey: "key1"}},
	}}
	item := &MomentItem{MomentID: "m1"}
	HydrateSingleMoment(ctx, fetcher, stubImageResolver{}, item)
	if len(item.Images) != 1 {
		t.Fatalf("expected 1 image, got %#v", item.Images)
	}
	// nil item 直接返回
	HydrateSingleMoment(ctx, fetcher, stubImageResolver{}, nil)
}
