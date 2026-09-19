package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// StatusCache memoizes health checks so a catalogue read does not re-hash every
// installed package's files.
//
// Status hashes every managed file, and the GUI reads a status for every
// package on every catalogue load, so refreshing the list after one operation
// paid to re-hash every other installed package. Entries are keyed on the digest
// of the recorded installed state, so a change to that record -- a version
// bump, a new file set, a different destination -- changes the key and the next
// read recomputes even if a caller forgets to invalidate. The manager also
// invalidates the operated package explicitly on every mutation, whatever the
// outcome, so a mutation costs one recomputation instead of a whole-catalogue
// sweep.
//
// What this changes is when the check runs, not how. A package whose state
// changed, or whose entry was invalidated, is re-hashed in full. A managed file
// edited outside the installer is still detected -- the size and modification
// time are deliberately not trusted, which is what
// TestHealthCheckHashesContentWhenSizeAndTimeMatch pins down -- but only once
// the entry is invalidated, not on every catalogue read. Invalidate forces that
// re-verification for one package and InvalidateAll for every package.
type StatusCache struct {
	mu      sync.Mutex
	entries map[string]statusCacheEntry
}

type statusCacheEntry struct {
	identity string
	status   Status
}

// NewStatusCache returns an empty cache. A nil *StatusCache is also usable and
// caches nothing, so a manager built without one keeps verifying every file.
func NewStatusCache() *StatusCache {
	return &StatusCache{entries: make(map[string]statusCacheEntry)}
}

// Invalidate drops one package's cached health check, so the next read re-hashes
// its files.
func (cache *StatusCache) Invalidate(id string) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	delete(cache.entries, id)
}

// InvalidateAll drops every cached health check, so the next catalogue read
// verifies every installed package from its files again.
func (cache *StatusCache) InvalidateAll() {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	clear(cache.entries)
}

func (cache *StatusCache) get(id, identity string) (Status, bool) {
	if cache == nil {
		return Status{}, false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	entry, found := cache.entries[id]
	if !found || entry.identity != identity {
		return Status{}, false
	}
	return entry.status.clone(), true
}

func (cache *StatusCache) put(id, identity string, status Status) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.entries[id] = statusCacheEntry{identity: identity, status: status.clone()}
}

// stateIdentity digests what the recorded installed state claims, so a cached
// health check is reused only while the record it was derived from is
// unchanged. encodeState is the same canonical encoding the state file is
// written with, so the digest covers the manifest, every recorded file path,
// hash and mode, the original-file backups, and menu ownership.
func stateIdentity(state *Installed) (string, error) {
	data, err := encodeState(*state)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

// clone copies the issue list, so a caller that reads a cached status cannot
// reach into the copy the cache holds.
func (status Status) clone() Status {
	status.Issues = append([]HealthIssue(nil), status.Issues...)
	return status
}
