// Package render provides the tiered memory system for MOFU.
//
// The memory system manages data across four tiers:
//
//	L1 Hot:     Current UI state, active widgets, focused elements
//	L2 Cached:  Computed state, layout cache, style cache
//	L3 Streamed: External data sources, network feeds, file watches
//	L4 Persisted: Disk cache, session snapshots, undo history
//
// This tiering ensures that hot data is always in cache while cold data
// is evicted to slower storage, minimizing GC pressure and memory usage.
package render

import (
	"container/list"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Tier definitions
// ---------------------------------------------------------------------------

// Tier represents a memory tier level.
type Tier int

const (
	TierHot      Tier = 0 // L1: current UI state, active widgets
	TierCached   Tier = 1 // L2: computed state, layout, styles
	TierStreamed Tier = 2 // L3: external data, network, files
	TierPersist  Tier = 3 // L4: disk cache, snapshots, undo
)

// TierConfig holds configuration for a single tier.
type TierConfig struct {
	MaxEntries int
	TTL        time.Duration // 0 = no expiry
}

// DefaultTierConfigs returns sensible defaults for each tier.
func DefaultTierConfigs() [4]TierConfig {
	return [4]TierConfig{
		{MaxEntries: 256, TTL: 0},                 // L1 hot: no expiry, small
		{MaxEntries: 1024, TTL: 5 * time.Minute},  // L2 cached: 5min TTL
		{MaxEntries: 4096, TTL: 30 * time.Minute}, // L3 streamed: 30min TTL
		{MaxEntries: 16384, TTL: 0},               // L4 persisted: no expiry, large
	}
}

// ---------------------------------------------------------------------------
// Cache entry
// ---------------------------------------------------------------------------

// entry is a single cached item with metadata.
type entry struct {
	key      string
	value    any
	tier     Tier
	created  time.Time
	lastUsed time.Time
	hitCount int
}

// ---------------------------------------------------------------------------
// TieredCache — LRU cache with tiered eviction
// ---------------------------------------------------------------------------

// TieredCache is a multi-tier LRU cache with TTL-based expiry.
// Data flows: L1 (hot) → L2 (cached) → L3 (streamed) → L4 (persisted)
type TieredCache struct {
	mu      sync.RWMutex
	tiers   [4]*list.List // LRU lists per tier
	index   map[string]*list.Element
	configs [4]TierConfig
	hits    uint64
	misses  uint64
}

// NewTieredCache creates a TieredCache with default configurations.
func NewTieredCache() *TieredCache {
	tc := &TieredCache{
		index:   make(map[string]*list.Element),
		configs: DefaultTierConfigs(),
	}
	for i := 0; i < 4; i++ {
		tc.tiers[i] = list.New()
	}
	return tc
}

// NewTieredCacheWithConfigs creates a TieredCache with custom tier configs.
func NewTieredCacheWithConfigs(cfgs [4]TierConfig) *TieredCache {
	tc := &TieredCache{
		index:   make(map[string]*list.Element),
		configs: cfgs,
	}
	for i := 0; i < 4; i++ {
		tc.tiers[i] = list.New()
	}
	return tc
}

// Get retrieves a value by key, promoting it to L1 on access.
// Returns (value, true) if found, (nil, false) otherwise.
func (tc *TieredCache) Get(key string) (any, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	elem, ok := tc.index[key]
	if !ok {
		tc.misses++
		return nil, false
	}

	e := elem.Value.(*entry)

	// Check TTL expiry
	if tc.configs[e.tier].TTL > 0 && time.Since(e.created) > tc.configs[e.tier].TTL {
		tc.removeLocked(e, elem)
		tc.misses++
		return nil, false
	}

	// Promote to L1
	e.lastUsed = time.Now()
	e.hitCount++
	tc.tiers[e.tier].Remove(elem)
	e.tier = TierHot
	tc.tiers[TierHot].PushFront(elem)
	tc.hits++

	return e.value, true
}

// Set stores a value at the specified tier.
func (tc *TieredCache) Set(key string, value any, tier Tier) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Update existing entry
	if elem, ok := tc.index[key]; ok {
		e := elem.Value.(*entry)
		e.value = value
		e.tier = tier
		e.lastUsed = time.Now()
		e.created = time.Now()
		tc.tiers[e.tier].Remove(elem)
		tc.tiers[tier].PushFront(elem)
		return
	}

	// Create new entry
	e := &entry{
		key:      key,
		value:    value,
		tier:     tier,
		created:  time.Now(),
		lastUsed: time.Now(),
	}
	elem := tc.tiers[tier].PushFront(e)
	tc.index[key] = elem

	// Evict if over capacity
	tc.evict(tier)
}

// Delete removes a key from all tiers.
func (tc *TieredCache) Delete(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	elem, ok := tc.index[key]
	if !ok {
		return
	}
	e := elem.Value.(*entry)
	tc.removeLocked(e, elem)
}

// Has reports whether a key exists (without promoting).
func (tc *TieredCache) Has(key string) bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	_, ok := tc.index[key]
	return ok
}

// Clear removes all entries from all tiers.
func (tc *TieredCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for i := 0; i < 4; i++ {
		tc.tiers[i].Init()
	}
	tc.index = make(map[string]*list.Element)
}

// Stats returns cache hit/miss statistics.
func (tc *TieredCache) Stats() CacheStats {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	stats := CacheStats{
		Hits:   tc.hits,
		Misses: tc.misses,
	}
	for i := 0; i < 4; i++ {
		stats.Counts[i] = tc.tiers[i].Len()
	}
	if tc.hits+tc.misses > 0 {
		stats.HitRate = float64(tc.hits) / float64(tc.hits+tc.misses)
	}
	return stats
}

// TierCounts returns the number of entries in each tier.
func (tc *TieredCache) TierCounts() [4]int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	var counts [4]int
	for i := 0; i < 4; i++ {
		counts[i] = tc.tiers[i].Len()
	}
	return counts
}

// evict removes the least-recently-used entry from the tier if over capacity.
func (tc *TieredCache) evict(tier Tier) {
	for tc.tiers[tier].Len() > tc.configs[tier].MaxEntries {
		back := tc.tiers[tier].Back()
		if back == nil {
			break
		}
		e := back.Value.(*entry)
		tc.removeLocked(e, back)
	}
}

func (tc *TieredCache) removeLocked(e *entry, elem *list.Element) {
	tc.tiers[e.tier].Remove(elem)
	delete(tc.index, e.key)
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

// CacheStats holds cache performance statistics.
type CacheStats struct {
	Hits    uint64
	Misses  uint64
	HitRate float64
	Counts  [4]int // Entry count per tier
}

// EvictByTier demotes all entries from one tier to the next lower tier.
// Useful for manual memory pressure management.
func (tc *TieredCache) EvictByTier(from Tier) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if from >= TierPersist {
		return
	}

	to := from + 1
	var next *list.Element
	for e := tc.tiers[from].Front(); e != nil; e = next {
		next = e.Next()
		ent := e.Value.(*entry)
		tc.tiers[from].Remove(e)
		ent.tier = to
		tc.tiers[to].PushFront(e)
		tc.evict(to)
	}
}
