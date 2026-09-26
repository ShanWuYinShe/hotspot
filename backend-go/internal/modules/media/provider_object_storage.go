package media

import (
	"bytes"
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
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idgen"
)

const mediaPresignTTL = 15 * time.Minute

// presign GET 读 URL 的进程内缓存（审计 perf P3-4）：同一 key 在窗口内返回
// 同一签名 URL——(a) 免去每次 feed 渲染对每图重复的 SigV4 签名；(b) URL 跨请求
// 稳定，客户端（浏览器磁盘缓存）命中率显著提升。签名仅是授权，不冻结对象内容，
// 对象被覆盖后旧 URL 即取新内容，无陈旧性问题。两代轮换限容：条目存活 1–2 个
// 窗口（远短于 GET TTL），总内存有界。PUT/HEAD 不缓存（上传与内部探测要求新鲜）。
const (
	presignCacheWindow     = time.Hour
	presignCacheMaxEntries = 8192
)

type presignCacheEntry struct {
	url string
}

type ObjectStorageProvider struct {
	cfg    config.Config
	client *http.Client
	now    func() time.Time

	presignMu      sync.Mutex
	presignCur     map[string]presignCacheEntry
	presignPrev    map[string]presignCacheEntry
	presignRotated time.Time // 零值 = 尚未初始化，首次访问即起算窗口
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
	// upload_id 用 crypto/rand 随机 ID（P3-2）：原先 UnixNano 顺序可枚举，
	// 攻击者可按时间窗遍历猜测他人 upload_id；idgen 提供 96 位熵防遍历，
	// 同时消除并发同纳秒碰撞（与 share/moment 等 ID 生成口径一致）。
	uploadID, err := idgen.WithPrefix("upload_")
	if err != nil {
		return PresignedUpload{}, err
	}
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

// Delete 删除存储对象（保留期 GC 用）：对象不存在视为已删（幂等），
// 其余非 2xx 返回错误由调用方决定重试策略。
func (p *ObjectStorageProvider) Delete(ctx context.Context, storageKey string) error {
	signedURL, err := p.presignInternalURL(ctx, http.MethodDelete, storageKey, 5*time.Minute)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, signedURL, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("media object storage delete failed: status %d", resp.StatusCode)
	}
	return nil
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

// ReadPrefix fetches at most n leading bytes of the object via a range GET
// (S3 semantics: 206 Partial Content; a server ignoring Range answers 200
// with the full body — the LimitReader caps the read either way).
func (p *ObjectStorageProvider) ReadPrefix(ctx context.Context, storageKey string, n int64) ([]byte, error) {
	signedURL, err := p.presignInternalURL(ctx, http.MethodGet, storageKey, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, signedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", n-1))
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUploadNotReady
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("media object storage read prefix failed: status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, n))
}

// GetObject 读取完整对象（EXIF 清洗链路）。
func (p *ObjectStorageProvider) GetObject(ctx context.Context, storageKey string) ([]byte, error) {
	signedURL, err := p.presignInternalURL(ctx, http.MethodGet, storageKey, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, signedURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUploadNotReady
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("media object storage get failed: status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// PutObject 覆写对象（清洗后原键回写；Content-Type 必须保持，否则存储端
// MIME 变化会使后续 Head 核对与客户端展示错位）。
func (p *ObjectStorageProvider) PutObject(ctx context.Context, storageKey, contentType string, data []byte) error {
	signedURL, err := p.presignInternalURL(ctx, http.MethodPut, storageKey, 5*time.Minute)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, signedURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("media object storage put failed: status %d", resp.StatusCode)
	}
	return nil
}

// PresignGet returns a client-facing absolute URL that grants short-term
// read access to the object at storageKey. The URL is signed against the
// public (client-facing) endpoint so it is valid through the reverse proxy.
// 读 URL 走两代缓存（审计 perf P3-4）：窗口内同 key 返回同一签名 URL，签名
// 结果 1–2 个窗口后自然过期重签（远短于读 TTL，安全性不受影响）。
func (p *ObjectStorageProvider) PresignGet(ctx context.Context, storageKey string, ttl time.Duration) (string, error) {
	cacheKey := "get:" + storageKey + ":" + ttl.String()
	if url, ok := p.cachedPresignedURL(cacheKey); ok {
		return url, nil
	}
	url, err := p.presignURL(ctx, http.MethodGet, storageKey, ttl)
	if err != nil {
		return "", err
	}
	p.storePresignedURL(cacheKey, url)
	return url, nil
}

// cachedPresignedURL 查缓存并按窗口懒轮换（cur → prev，prev 丢弃）。
func (p *ObjectStorageProvider) cachedPresignedURL(cacheKey string) (string, bool) {
	p.presignMu.Lock()
	defer p.presignMu.Unlock()
	now := p.now()
	if p.presignRotated.IsZero() {
		p.presignRotated = now
		p.presignCur = map[string]presignCacheEntry{}
	}
	if now.Sub(p.presignRotated) >= presignCacheWindow {
		p.presignPrev = p.presignCur
		p.presignCur = map[string]presignCacheEntry{}
		p.presignRotated = now
	}
	if entry, ok := p.presignCur[cacheKey]; ok {
		return entry.url, true
	}
	if entry, ok := p.presignPrev[cacheKey]; ok {
		return entry.url, true
	}
	return "", false
}

// storePresignedURL 写入当前代；容量满时放弃缓存（签名路径仍正确返回）。
func (p *ObjectStorageProvider) storePresignedURL(cacheKey, url string) {
	p.presignMu.Lock()
	defer p.presignMu.Unlock()
	if p.presignCur == nil {
		p.presignCur = map[string]presignCacheEntry{}
	}
	if len(p.presignCur) >= presignCacheMaxEntries {
		return
	}
	p.presignCur[cacheKey] = presignCacheEntry{url: url}
}

func (p *ObjectStorageProvider) presignURL(ctx context.Context, method, storageKey string, ttl time.Duration) (string, error) {
	return p.presignURLWithEndpoint(ctx, method, storageKey, ttl, p.clientFacingEndpoint)
}

func (p *ObjectStorageProvider) presignInternalURL(ctx context.Context, method, storageKey string, ttl time.Duration) (string, error) {
	return p.presignURLWithEndpoint(ctx, method, storageKey, ttl, p.internalEndpoint)
}

func (p *ObjectStorageProvider) presignURLWithEndpoint(ctx context.Context, method, storageKey string, ttl time.Duration, endpointFn func(context.Context) (*url.URL, bool, error)) (string, error) {
	endpoint, forcePathStyle, err := endpointFn(ctx)
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

func (p *ObjectStorageProvider) internalEndpoint(context.Context) (*url.URL, bool, error) {
	endpoint, err := url.Parse(strings.TrimSpace(p.cfg.S3Endpoint))
	if err != nil {
		return nil, false, err
	}
	return endpoint, p.cfg.S3ForcePathStyle, nil
}

func (p *ObjectStorageProvider) clientFacingEndpoint(ctx context.Context) (*url.URL, bool, error) {
	staticEndpoint, forcePathStyle, err := p.configClientFacingEndpoint()
	if err != nil {
		return nil, false, err
	}
	if ep, ok := clientRequestEndpointFromContext(ctx); ok {
		dynamic := *staticEndpoint
		dynamic.Scheme = ep.Scheme
		dynamic.Host = ep.Host
		return &dynamic, true, nil
	}
	return staticEndpoint, forcePathStyle, nil
}

func (p *ObjectStorageProvider) configClientFacingEndpoint() (*url.URL, bool, error) {
	rawEndpoint := strings.TrimSpace(p.cfg.S3PublicEndpoint)
	forcePathStyle := p.cfg.S3PublicForcePathStyle
	if rawEndpoint == "" {
		return p.internalEndpoint(context.Background())
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
