package sopclient

import (
	"sync"
	"sync/atomic"
	"time"
)

// This file holds the refresh-merge rule for checkpoint progress (CTRL010).
//
// Refresh semantics: the controller re-reads SOP on every refresh and holds no
// persisted progress of its own. To keep a known checkpoint from regressing to
// unknown when a re-read yields no new evidence (for example a transient read of
// a run directory SOP is rewriting), the store keeps the last SOP-reported
// checkpoint per task in memory only. A re-read that reports no checkpoint keeps
// the previously known value; a re-read that reports a checkpoint replaces it.
// Nothing here computes or caches a percentage: the cached value is itself a
// verbatim SOP-reported record.
//
// The cache is deliberately bounded in both time and size so it cannot present
// stale progress forever:
//
//   - Entries expire after checkpointTTL. Once a remembered checkpoint is older
//     than the TTL, a re-read that yields no evidence reports the checkpoint as
//     absent again instead of replaying a possibly-retracted value. This keeps
//     "preserve on refresh" from becoming "sticky forever" when SOP legitimately
//     drops evidence (task reset, replan, a fresh run directory, or a removed
//     artifact).
//   - The cache is capped at maxCheckpointCache entries, evicting the least-
//     recently used task, so per-process memory cannot grow without bound across
//     unrelated stores/projects.
//
// A re-read that reports evidence always wins and refreshes the entry's age, so
// live SOP progress is never aged out while SOP keeps reporting it.
const (
	// checkpointTTL bounds how long a remembered checkpoint is replayed after
	// the last SOP-reported evidence for it. Past this window a refresh that
	// yields no evidence reports absence again rather than a stale value.
	checkpointTTL = 10 * time.Minute
	// maxCheckpointCache bounds the number of remembered tasks, so the cache is
	// not an unbounded growth path in a long-running controller process.
	maxCheckpointCache = 256
)

type checkpointEntry struct {
	progress CheckpointProgress
	seen     time.Time
}

type checkpointCache struct {
	mu    sync.Mutex
	known map[string]checkpointEntry
	now   func() time.Time // injectable clock for tests
}

func (c *checkpointCache) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// merge returns the checkpoint to display after a refresh. When the fresh read
// reported evidence (Present=true) it wins and is remembered (with a refreshed
// age). When the fresh read reported no evidence, the last remembered record is
// returned only if it was seen within checkpointTTL; otherwise the absent fresh
// record is returned, so a retracted/reset checkpoint is never replayed forever.
// The initial state (nothing ever reported) stays absent.
func (c *checkpointCache) merge(taskID string, fresh CheckpointProgress) CheckpointProgress {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.known == nil {
		c.known = map[string]checkpointEntry{}
	}
	now := c.clock()
	if fresh.Present {
		c.known[taskID] = checkpointEntry{progress: fresh, seen: now}
		c.evictLocked()
		return fresh
	}
	if last, ok := c.known[taskID]; ok {
		if now.Sub(last.seen) <= checkpointTTL {
			return last.progress
		}
		// Expired: drop it so a retracted checkpoint does not linger.
		delete(c.known, taskID)
	}
	return fresh
}

// evictLocked trims the cache to maxCheckpointCache entries, dropping the oldest
// (least-recently-seen) entries first. Caller holds c.mu.
func (c *checkpointCache) evictLocked() {
	for len(c.known) > maxCheckpointCache {
		var oldestID string
		var oldest time.Time
		for id, e := range c.known {
			if oldestID == "" || e.seen.Before(oldest) {
				oldestID, oldest = id, e.seen
			}
		}
		delete(c.known, oldestID)
	}
}

// checkpointStore holds the process-wide cache pointer in a form that is safe to
// lazily initialize from concurrent read paths. Two goroutines calling Task on
// the same *Store (including a Store value constructed outside OpenStore) must
// not race on the pointer assignment, so the pointer is stored atomically and
// initialized via CompareAndSwap rather than a plain field write.
type checkpointStore struct {
	ptr atomic.Pointer[checkpointCache]
}

// get returns the shared cache, creating it on first use. Concurrent callers
// converge on one cache instance; no caller observes a torn or nil pointer.
func (c *checkpointStore) get() *checkpointCache {
	if p := c.ptr.Load(); p != nil {
		return p
	}
	fresh := &checkpointCache{}
	if c.ptr.CompareAndSwap(nil, fresh) {
		return fresh
	}
	return c.ptr.Load()
}
