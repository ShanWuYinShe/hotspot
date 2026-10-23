package cursor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"hotspot/backend-go/internal/platform/config"
)

var ErrInvalidCursor = errors.New("cursor: invalid cursor")

// Codec wraps an AES-GCM cipher derived from a single AuthTokenSecret and a
// namespace, amortising the (relatively expensive) aes.NewCipher + GCM setup
// across every cursor encode/decode call instead of rebuilding it per
// request. The codec assumes AuthTokenSecret is load-once at process start
// (which it is — env vars are not re-read after boot); if the secret ever
// changes at runtime, callers must construct a fresh Codec.
type Codec struct {
	gcm        cipher.AEAD
	nonceSize  int
	renderOnce sync.Once
	renderErr  error
	namespace  string
}

// NewCodec returns a Codec bound to a specific namespace. The cipher is
// built lazily on first use so that test code that never touches a cursor
// does not pay the setup cost (and so any config-load error surfaces at the
// first real call rather than at constructor time).
func NewCodec(namespace string) *Codec {
	return &Codec{namespace: namespace}
}

func (c *Codec) ensure() error {
	c.renderOnce.Do(func() {
		secret := config.Load().AuthTokenSecret
		if secret == "" {
			secret = "dev-secret-change-me"
		}
		key := sha256.Sum256([]byte(c.namespace + ":" + secret))
		block, err := aes.NewCipher(key[:])
		if err != nil {
			c.renderErr = err
			return
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			c.renderErr = err
			return
		}
		c.gcm = gcm
		c.nonceSize = gcm.NonceSize()
	})
	return c.renderErr
}

// Encode JSON-marshals payload, encrypts it with AES-GCM and returns the
// base64-rawurl-encoded token. Errors are returned rather than swallowed.
func (c *Codec) Encode(payload any) (string, error) {
	if err := c.ensure(); err != nil {
		return "", err
	}
	blob, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, c.nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.gcm.Seal(nonce, nonce, blob, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decode reverses Encode, writing the decrypted JSON into out. An empty
// input returns nil (no cursor) without touching out.
func (c *Codec) Decode(value string, out any) error {
	payload, err := c.decrypt(value)
	if err != nil {
		return err
	}
	if payload == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return ErrInvalidCursor
	}
	return nil
}

// DecodeString reverses EncodeString.
func (c *Codec) DecodeString(value string) (string, error) {
	payload, err := c.decrypt(value)
	if err != nil {
		return "", err
	}
	if payload == nil {
		return "", nil
	}
	return string(payload), nil
}

// decrypt 是 Decode/DecodeString 共用的解密通道：base64 解码 + AES-GCM
// Open。nil,nil 表示空输入（无游标，调用方按各自空值语义处理）；其余错误
// 统一归为 ErrInvalidCursor（此前两函数各复制一份同构实现，改校验要同步两处）。
func (c *Codec) decrypt(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	if err := c.ensure(); err != nil {
		return nil, ErrInvalidCursor
	}
	blob, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	if len(blob) < c.nonceSize {
		return nil, ErrInvalidCursor
	}
	nonce, ciphertext := blob[:c.nonceSize], blob[c.nonceSize:]
	payload, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	return payload, nil
}

// codecCache memoises Codec instances by namespace so that the package-level
// Encode/Decode helpers do not pay aes.NewCipher + cipher.NewGCM on every
// call. Codecs build their cipher lazily on first use, so the cache itself
// is cheap; the cost amortisation happens inside each Codec's sync.Once.
var (
	codecCacheMu sync.Mutex
	codecCache   = map[string]*Codec{}
)

func codecFor(namespace string) *Codec {
	codecCacheMu.Lock()
	defer codecCacheMu.Unlock()
	if c, ok := codecCache[namespace]; ok {
		return c
	}
	c := NewCodec(namespace)
	codecCache[namespace] = c
	return c
}

func Encode(namespace string, payload any) (string, error) {
	return codecFor(namespace).Encode(payload)
}

func Decode(namespace string, value string, out any) error {
	return codecFor(namespace).Decode(value, out)
}
