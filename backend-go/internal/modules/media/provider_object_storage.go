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
		cfg:    cfg,
		client: http.DefaultClient,
		now:    time.Now,
	}
}

func (p *ObjectStorageProvider) Presign(ctx context.Context, req PresignUploadRequest, ownerUserID string) (PresignedUpload, error) {
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
	objectURL, err := p.objectURL(storageKey)
	if err != nil {
		return ObjectMetadata{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, objectURL, nil)
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

func (p *ObjectStorageProvider) presignPutObjectURL(ctx context.Context, storageKey, mimeType, md5Value string) (string, map[string]string, time.Time, error) {
	_ = ctx
	endpoint, err := url.Parse(strings.TrimSpace(p.cfg.S3Endpoint))
	if err != nil {
		return "", nil, time.Time{}, err
	}
	if endpoint.Scheme == "" || endpoint.Host == "" || strings.TrimSpace(p.cfg.S3Bucket) == "" {
		return "", nil, time.Time{}, fmt.Errorf("media object storage not configured")
	}
	region := strings.TrimSpace(p.cfg.S3Region)
	if region == "" {
		region = "us-east-1"
	}
	now := p.now().UTC()
	expiresAt := now.Add(mediaPresignTTL)
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	credentialScope := dateStamp + "/" + region + "/s3/aws4_request"
	host := endpoint.Host
	canonicalURI := p.canonicalURI(endpoint, strings.TrimSpace(p.cfg.S3Bucket), storageKey)
	signedHeaders := "host"
	query := url.Values{}
	query.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	query.Set("X-Amz-Credential", p.cfg.S3AccessKey+"/"+credentialScope)
	query.Set("X-Amz-Date", amzDate)
	query.Set("X-Amz-Expires", strconv.Itoa(int(mediaPresignTTL.Seconds())))
	query.Set("X-Amz-SignedHeaders", signedHeaders)
	canonicalHeaders := "host:" + host + "\n"
	canonicalRequest := strings.Join([]string{
		http.MethodPut,
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
	presignedURL.Path = canonicalURI
	presignedURL.RawQuery = query.Encode()
	headers := map[string]string{"Content-Type": mimeType}
	if md5Value != "" {
		headers["Content-MD5"] = md5Value
	}
	return presignedURL.String(), headers, expiresAt, nil
}

func (p *ObjectStorageProvider) objectURL(storageKey string) (string, error) {
	endpoint, err := url.Parse(strings.TrimSpace(p.cfg.S3Endpoint))
	if err != nil {
		return "", err
	}
	if endpoint.Scheme == "" || endpoint.Host == "" || strings.TrimSpace(p.cfg.S3Bucket) == "" {
		return "", fmt.Errorf("media object storage not configured")
	}
	endpoint.Path = p.canonicalURI(endpoint, strings.TrimSpace(p.cfg.S3Bucket), storageKey)
	endpoint.RawQuery = ""
	return endpoint.String(), nil
}

func (p *ObjectStorageProvider) canonicalURI(endpoint *url.URL, bucket, storageKey string) string {
	cleanKey := path.Clean("/" + strings.TrimLeft(storageKey, "/"))
	if p.cfg.S3ForcePathStyle || strings.Contains(endpoint.Host, ":") || endpoint.Host == "localhost" {
		basePath := strings.TrimSuffix(endpoint.Path, "/")
		return basePath + "/" + bucket + cleanKey
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
