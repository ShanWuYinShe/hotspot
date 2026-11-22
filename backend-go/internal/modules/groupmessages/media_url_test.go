package groupmessages

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

func TestSignGroupMediaMapRewritesURLAndThumbnail(t *testing.T) {
	signer := &stubMediaURLSigner{}
	svc := &Service{mediaURLSigner: signer}
	media := map[string]any{
		"url":           "published/u1/a.png",
		"thumbnail_url": "published/u1/thumb.png",
	}
	svc.signMediaMap(context.Background(), media)
	if media["url"] != "https://cdn.example.com/published/u1/a.png" {
		t.Fatalf("url = %#v", media["url"])
	}
	if media["thumbnail_url"] != "https://cdn.example.com/published/u1/thumb.png" {
		t.Fatalf("thumbnail = %#v", media["thumbnail_url"])
	}
	if len(signer.calls) != 2 {
		t.Fatalf("calls = %#v", signer.calls)
	}
}

func TestSignGroupMediaKeepsKeyWithoutSignerOrOnError(t *testing.T) {
	media := map[string]any{"url": "published/u1/a.png"}
	(&Service{}).signMediaMap(context.Background(), media)
	if media["url"] != "published/u1/a.png" {
		t.Fatalf("nil signer must keep key, got %#v", media["url"])
	}
	svc := &Service{mediaURLSigner: &stubMediaURLSigner{err: errors.New("sign failed")}}
	svc.signMediaMap(context.Background(), media)
	if media["url"] != "published/u1/a.png" {
		t.Fatalf("sign error must keep key, got %#v", media["url"])
	}
}
