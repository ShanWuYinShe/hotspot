package privatemessages

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubMediaURLSigner struct {
	calls []string
	err   error
}

func (s *stubMediaURLSigner) PresignGet(_ context.Context, storageKey string, _ time.Duration) (string, error) {
	s.calls = append(s.calls, storageKey)
	if s.err != nil {
		return "", s.err
	}
	return "https://cdn.example.com/" + storageKey, nil
}

func TestSignMediaItemsRewritesURLAndThumbnail(t *testing.T) {
	signer := &stubMediaURLSigner{}
	svc := &Service{mediaURLSigner: signer}
	thumb := "uploads/u1/thumb.png"
	items := []PrivateMessageItem{{Media: &MessageMedia{URL: "uploads/u1/a.png", ThumbnailURL: &thumb}}}
	svc.signMediaItems(context.Background(), items)
	if got := items[0].Media.URL; got != "https://cdn.example.com/uploads/u1/a.png" {
		t.Fatalf("url = %q", got)
	}
	if items[0].Media.ThumbnailURL == nil || *items[0].Media.ThumbnailURL != "https://cdn.example.com/uploads/u1/thumb.png" {
		t.Fatalf("thumbnail = %#v", items[0].Media.ThumbnailURL)
	}
	if len(signer.calls) != 2 {
		t.Fatalf("calls = %#v", signer.calls)
	}
}

func TestSignMediaMapDoesNotMutateAbsoluteURL(t *testing.T) {
	signer := &stubMediaURLSigner{}
	svc := &Service{mediaURLSigner: signer}
	media := map[string]any{"url": "https://existing.example.com/a.png"}
	svc.signMediaMap(context.Background(), media)
	if media["url"] != "https://existing.example.com/a.png" {
		t.Fatalf("url = %#v", media["url"])
	}
	if len(signer.calls) != 0 {
		t.Fatalf("absolute url must not be re-signed, calls = %#v", signer.calls)
	}
}

func TestSignMediaKeepsStorageKeyWithoutSignerOrOnError(t *testing.T) {
	item := []PrivateMessageItem{{Media: &MessageMedia{URL: "uploads/u1/a.png"}}}
	(&Service{}).signMediaItems(context.Background(), item)
	if item[0].Media.URL != "uploads/u1/a.png" {
		t.Fatalf("nil signer must keep key, got %q", item[0].Media.URL)
	}
	svc := &Service{mediaURLSigner: &stubMediaURLSigner{err: errors.New("sign failed")}}
	svc.signMediaItems(context.Background(), item)
	if item[0].Media.URL != "uploads/u1/a.png" {
		t.Fatalf("sign error must keep key, got %q", item[0].Media.URL)
	}
}

func TestServiceHistorySignsMediaURL(t *testing.T) {
	repo := &stubMsgRepo{histResult: PaginatedMessages{Items: []PrivateMessageItem{{
		Media: &MessageMedia{URL: "uploads/u1/a.png"},
	}}}}
	svc := NewServiceWithDeps(repo, &stubUserReader{})
	svc.SetMediaURLSigner(&stubMediaURLSigner{})
	result, err := svc.History(context.Background(), ListPrivateMessagesRequest{})
	if err != nil {
		t.Fatalf("history error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Media.URL != "https://cdn.example.com/uploads/u1/a.png" {
		t.Fatalf("signed media = %#v", result.Items)
	}
}

func TestServiceLastSignsMediaURL(t *testing.T) {
	repo := &stubMsgRepo{lastResult: &PrivateMessageItem{Media: &MessageMedia{URL: "uploads/u1/b.png"}}}
	svc := NewServiceWithDeps(repo, &stubUserReader{})
	svc.SetMediaURLSigner(&stubMediaURLSigner{})
	item, err := svc.Last(context.Background(), "u2", nil)
	if err != nil || item == nil {
		t.Fatalf("last error: %v", err)
	}
	if item.Media.URL != "https://cdn.example.com/uploads/u1/b.png" {
		t.Fatalf("signed media = %q", item.Media.URL)
	}
}
