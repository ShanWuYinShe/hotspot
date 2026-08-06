package resourceguard

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type ResourceKind string

const (
	ResourceUser   ResourceKind = "user"
	ResourceGroup  ResourceKind = "group"
	ResourceMoment ResourceKind = "moment"
)

type Checker interface {
	Ready(context.Context) bool
	MayExist(context.Context, ResourceKind, string) (bool, error)
	Record(context.Context, ResourceKind, string) error
}

type NoopChecker struct{}

func (NoopChecker) Ready(context.Context) bool                                   { return true }
func (NoopChecker) MayExist(context.Context, ResourceKind, string) (bool, error) { return true, nil }
func (NoopChecker) Record(context.Context, ResourceKind, string) error           { return nil }

type DBChecker struct {
	db      persistence.QueryableDB
	ttl     time.Duration
	mu      sync.RWMutex
	missing map[string]time.Time
}

func NewDBChecker(db persistence.QueryableDB) *DBChecker {
	return &DBChecker{db: db, ttl: 2 * time.Minute, missing: map[string]time.Time{}}
}

func (c *DBChecker) Ready(context.Context) bool {
	return c != nil && c.db != nil
}

func (c *DBChecker) MayExist(ctx context.Context, kind ResourceKind, id string) (bool, error) {
	if c == nil || c.db == nil || id == "" {
		return true, nil
	}
	if c.isMissing(kind, id) {
		return false, nil
	}
	var exists int
	err := c.db.QueryRow(ctx, queryForKind(kind), id).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.markMissing(kind, id)
		return false, nil
	}
	return true, err
}

func (c *DBChecker) Record(_ context.Context, kind ResourceKind, id string) error {
	if c == nil || id == "" {
		return nil
	}
	c.mu.Lock()
	delete(c.missing, cacheKey(kind, id))
	c.mu.Unlock()
	return nil
}

func (c *DBChecker) isMissing(kind ResourceKind, id string) bool {
	c.mu.RLock()
	expiresAt, ok := c.missing[cacheKey(kind, id)]
	c.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().UTC().Before(expiresAt) {
		return true
	}
	c.mu.Lock()
	delete(c.missing, cacheKey(kind, id))
	c.mu.Unlock()
	return false
}

func (c *DBChecker) markMissing(kind ResourceKind, id string) {
	c.mu.Lock()
	c.missing[cacheKey(kind, id)] = time.Now().UTC().Add(c.ttl)
	c.mu.Unlock()
}

func queryForKind(kind ResourceKind) string {
	switch kind {
	case ResourceUser:
		return "SELECT 1 FROM users WHERE user_id = $1"
	case ResourceGroup:
		return "SELECT 1 FROM groups WHERE group_id = $1"
	case ResourceMoment:
		return "SELECT 1 FROM moments WHERE moment_id = $1"
	default:
		return "SELECT 1"
	}
}

func cacheKey(kind ResourceKind, id string) string {
	return string(kind) + ":" + id
}

// defaultChecker is written once at boot and read on every request; guard it
// with an atomic pointer to avoid the data race a bare global would have.
var defaultChecker atomic.Pointer[Checker]

func init() {
	noop := Checker(NoopChecker{})
	defaultChecker.Store(&noop)
}

func SetDefaultChecker(checker Checker) {
	if checker == nil {
		checker = NoopChecker{}
	}
	defaultChecker.Store(&checker)
}

func DefaultChecker() Checker {
	if c := defaultChecker.Load(); c != nil {
		return *c
	}
	return NoopChecker{}
}

func RecordUser(ctx context.Context, userID string) {
	_ = DefaultChecker().Record(ctx, ResourceUser, userID)
}

func RecordGroup(ctx context.Context, groupID string) {
	_ = DefaultChecker().Record(ctx, ResourceGroup, groupID)
}

func RecordMoment(ctx context.Context, momentID string) {
	_ = DefaultChecker().Record(ctx, ResourceMoment, momentID)
}
