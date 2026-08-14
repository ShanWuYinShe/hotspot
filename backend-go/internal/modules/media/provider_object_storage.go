package media

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

const mediaPresignTTL = 15 * time.Minute

type ObjectStorageProvider struct {
	cfg    config.Config
	client *http.Client
	now    func() time.Time
}

func NewObjectStorageProvider(cfg config.Config) *ObjectStorageProvider {
	return &ObjectStorageProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		now: time.Now,
	}
}

func (p *ObjectStorageProvider) Presign(ctx context.Context, req PresignFileInput, ownerUserID string) (PresignedUpload, error) {
	uploadID := fmt.Sprintf("upload_%d", p.now().UTC().UnixNano())
	storageKey := storageKeyFor(ownerUserID, uploadID, req.FileName)
	uploadURL, headers, expiresAt, err := p.presignPutObjectURL(ctx, storageKey, req.MimeType, req.MD5)
	if err != nil {
		return PresignedUpload{}, err
	}
	return PresignedUpload{
		UploadID:   uploadID,
		UploadURL:  uploadURL,
		Headers:    headers,
		ExpiresAt:  expiresAt.Format(time.RFC3339),
		StorageKey: storageKey,
	}, nil
}

func (p *ObjectStorageProvider) Head(ctx context.Context, storageKey string) (ObjectMetadata, error) {
	signedURL, err := p.presignInternalURL(ctx, http.MethodHead, storageKey, 5*time.Minute)
	if err != nil {
		return ObjectMetadata{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, signedURL, nil)
	if err != nil {
		return ObjectMetadata{}, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return ObjectMetadata{}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return ObjectMetadata{}, ErrUploadNotReady
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ObjectMetadata{}, fmt.Errorf("media object storage head failed: status %d", resp.StatusCode)
	}
	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil || size <= 0 {
		return ObjectMetadata{}, fmt.Errorf("media object storage head missing content length")
	}
	mimeType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if mimeType == "" {
		return ObjectMetadata{}, fmt.Errorf("media object storage head missing content type")
	}
	return ObjectMetadata{FileSize: size, MimeType: mimeType}, nil
}

// PresignGet returns a client-facing absolute URL that grants short-term
// read access to the object at storageKey. The URL is signed against the
// public (client-facing) endpoint so it is valid through the reverse proxy.
func (p *ObjectStorageProvider) PresignGet(ctx context.Context, storageKey string, ttl time.Duration) (string, error) {
	return p.presignURL(ctx, http.MethodGet, storageKey, ttl)
}

func (p *ObjectStorageProvider) presignURL(ctx context.Context, method, storageKey string, ttl time.Duration) (string, error) {
	return p.presignURLWithEndpoint(ctx, method, storageKey, ttl, p.clientFacingEndpoint)
}

func (p *ObjectStorageProvider) presignInternalURL(ctx context.Context, method, storageKey string, ttl time.Duration) (string, error) {
	return p.presignURLWithEndpoint(ctx, method, storageKey, ttl, p.internalEndpoint)
}

func (p *ObjectStorageProvider) presignURLWithEndpoint(ctx context.Context, method, storageKey string, ttl time.Duration, endpointFn func() (*url.URL, bool, error)) (string, error) {
	_ = ctx
	endpoint, forcePathStyle, err := endpointFn()
	if err != nil {
		return "", err
	}
	if endpoint.Scheme == "" || endpoint.Host == "" || strings.TrimSpace(p.cfg.S3Bucket) == "" {
		return "", fmt.Errorf("media object storage not configured")
	}
	region := strings.TrimSpace(p.cfg.S3Region)
	if region == "" {
		region = "us-east-1"
	}
	now := p.now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	credentialScope := dateStamp + "/" + region + "/s3/aws4_request"
	host := endpoint.Host
	// The canonical URI signed here is the path the S3 server actually
	// receives. When a reverse proxy (e.g. nginx) mounts the S3 endpoint
	// under a subpath such as /s3 and strips it before forwarding, the
	// server sees only /<bucket>/<key>, so the proxy mount must NOT be
	// part of the signed canonical URI. The client-facing URL below still
	// keeps endpoint.Path so the request is routed to the proxy correctly.
	canonicalURI := canonicalURIForEndpoint(endpoint, strings.TrimSpace(p.cfg.S3Bucket), storageKey, forcePathStyle)
	signedHeaders := "host"
	query := url.Values{}
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", p.cfg.S3AccessKey+"/"+credentialScope)
	query.Set("X-Amz-Date", amzDate)
	query.Set("X-Amz-Expires", strconv.Itoa(int(ttl.Seconds())))
	query.Set("X-Amz-SignedHeaders", signedHeaders)
	canonicalHeaders := "host:" + host + "\n"
	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI,
		query.Encode(),
		canonicalHeaders,
		signedHeaders,
		"UNSIGNED-PAYLOAD",
	}, "\n")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hex.EncodeToString(hashSHA256([]byte(canonicalRequest))),
	}, "\n")
	signature := hex.EncodeToString(hmacSHA256(p.signingKey(dateStamp, region), stringToSign))
	query.Set("X-Amz-Signature", signature)
	presignedURL := *endpoint
	presignedURL.Path = strings.TrimSuffix(endpoint.Path, "/") + canonicalURI
	presignedURL.RawQuery = query.Encode()
	return presignedURL.String(), nil
}

func (p *ObjectStorageProvider) presignPutObjectURL(ctx context.Context, storageKey, mimeType, md5Value string) (string, map[string]string, time.Time, error) {
	uploadURL, err := p.presignURL(ctx, http.MethodPut, storageKey, mediaPresignTTL)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	expiresAt := p.now().UTC().Add(mediaPresignTTL)
	headers := map[string]string{"Content-Type": mimeType}
	if md5Value != "" {
		headers["Content-MD5"] = md5Value
	}
	return uploadURL, headers, expiresAt, nil
}

func (p *ObjectStorageProvider) internalEndpoint() (*url.URL, bool, error) {
	endpoint, err := url.Parse(strings.TrimSpace(p.cfg.S3Endpoint))
	if err != nil {
		return nil, false, err
	}
	return endpoint, p.cfg.S3ForcePathStyle, nil
}

func (p *ObjectStorageProvider) clientFacingEndpoint() (*url.URL, bool, error) {
	rawEndpoint := strings.TrimSpace(p.cfg.S3PublicEndpoint)
	forcePathStyle := p.cfg.S3PublicForcePathStyle
	if rawEndpoint == "" {
		return p.internalEndpoint()
	}
	endpoint, err := url.Parse(rawEndpoint)
	if err != nil {
		return nil, false, err
	}
	return endpoint, forcePathStyle, nil
}

func canonicalURIForEndpoint(endpoint *url.URL, bucket, storageKey string, forcePathStyle bool) string {
	cleanKey := path.Clean("/" + strings.TrimLeft(storageKey, "/"))
	if forcePathStyle || strings.Contains(endpoint.Host, ":") || endpoint.Host == "localhost" {
		// Server-relative path only (/<bucket>/<key>). The proxy mount
		// (endpoint.Path) is intentionally excluded because the S3 server
		// receives the path after the proxy strips its mount prefix.
		return "/" + bucket + cleanKey
	}
	return cleanKey
}

func (p *ObjectStorageProvider) signingKey(dateStamp, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+p.cfg.S3SecretKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, "s3")
	return hmacSHA256(kService, "aws4_request")
}

func hashSHA256(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(data))
	return h.Sum(nil)
}
