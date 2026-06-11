package mofu

import (
	"sync"
	"sync/atomic"
)

// ---------------------------------------------------------------------------
// Arena Allocator — frame-local bulk allocation
// ---------------------------------------------------------------------------

// Arena is a bump allocator that allocates memory in bulk and frees it all at once.
// It is ideal for frame-local data that is discarded each frame.
type Arena struct {
	mu       sync.Mutex
	chunks   [][]byte
	current  []byte
	offset   int
	total    int64
	allocs   int64
	frees    int64
}

// NewArena creates a new arena with the given chunk size.
func NewArena(chunkSize int) *Arena {
	if chunkSize <= 0 {
		chunkSize = 4096
	}
	chunk := make([]byte, chunkSize)
	return &Arena{
		chunks:  [][]byte{chunk},
		current: chunk,
	}
}

// Alloc allocates n bytes from the arena. Returns nil if n <= 0.
// The returned slice is valid until Reset is called.
func (a *Arena) Alloc(n int) []byte {
	if n <= 0 {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Align to 8 bytes
	n = (n + 7) &^ 7

	// Check if current chunk has space
	if a.offset+n > len(a.current) {
		// Allocate new chunk
		chunkSize := len(a.current)
		if n > chunkSize {
			chunkSize = n
		}
		a.current = make([]byte, chunkSize)
		a.chunks = append(a.chunks, a.current)
		a.offset = 0
	}

	result := a.current[a.offset : a.offset+n]
	a.offset += n
	a.total += int64(n)
	a.allocs++

	return result
}

// Reset frees all allocated memory. The arena can be reused after Reset.
func (a *Arena) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Keep the first chunk, discard the rest
	if len(a.chunks) > 1 {
		a.chunks = a.chunks[:1]
	}
	a.current = a.chunks[0]
	a.offset = 0
	a.frees++
}

// Stats returns arena allocation statistics.
func (a *Arena) Stats() ArenaStats {
	a.mu.Lock()
	defer a.mu.Unlock()
	return ArenaStats{
		Chunks:  len(a.chunks),
		Total:   a.total,
		Allocs:  a.allocs,
		Frees:   a.frees,
		Current: int64(a.offset),
	}
}

// ArenaStats tracks arena allocation statistics.
type ArenaStats struct {
	Chunks  int
	Total   int64
	Allocs  int64
	Frees   int64
	Current int64
}

// ---------------------------------------------------------------------------
// String Interning — deduplicate repeated strings
// ---------------------------------------------------------------------------

// StringInterner deduplicates strings to reduce memory usage.
type StringInterner struct {
	mu      sync.RWMutex
	strings map[string]string
	hits    int64
	misses  int64
}

// NewStringInterner creates a new string interner.
func NewStringInterner() *StringInterner {
	return &StringInterner{
		strings: make(map[string]string),
	}
}

// Intern returns a canonicalized version of the string.
// If the string was already interned, returns the existing copy.
func (si *StringInterner) Intern(s string) string {
	si.mu.RLock()
	if interned, ok := si.strings[s]; ok {
		si.mu.RUnlock()
		atomic.AddInt64(&si.hits, 1)
		return interned
	}
	si.mu.RUnlock()

	si.mu.Lock()
	// Double-check after acquiring write lock
	if interned, ok := si.strings[s]; ok {
		si.mu.Unlock()
		atomic.AddInt64(&si.hits, 1)
		return interned
	}
	si.strings[s] = s
	si.mu.Unlock()
	atomic.AddInt64(&si.misses, 1)
	return s
}

// Stats returns interner statistics.
func (si *StringInterner) Stats() InternerStats {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return InternerStats{
		Count: len(si.strings),
		Hits:  atomic.LoadInt64(&si.hits),
		Misses: atomic.LoadInt64(&si.misses),
	}
}

// InternerStats tracks string interning statistics.
type InternerStats struct {
	Count  int
	Hits   int64
	Misses int64
}

// ---------------------------------------------------------------------------
// Structural Sharing — copy-on-write for shared data
// ---------------------------------------------------------------------------

// SharedSlice is a copy-on-write slice that shares underlying storage
// until mutation is needed.
type SharedSlice[T any] struct {
	data []T
	refs *int32
}

// NewSharedSlice creates a shared slice from existing data.
func NewSharedSlice[T any](data []T) SharedSlice[T] {
	refs := int32(1)
	return SharedSlice[T]{data: data, refs: &refs}
}

// Clone returns a copy that shares the underlying data.
// The copy and original share storage until one is mutated.
func (s SharedSlice[T]) Clone() SharedSlice[T] {
	if s.refs != nil {
		atomic.AddInt32(s.refs, 1)
	}
	return SharedSlice[T]{data: s.data, refs: s.refs}
}

// Get returns the element at index i.
func (s SharedSlice[T]) Get(i int) T {
	return s.data[i]
}

// Len returns the length.
func (s SharedSlice[T]) Len() int {
	return len(s.data)
}

// Set sets the element at index i. If the slice is shared, it makes a private copy first.
func (s *SharedSlice[T]) Set(i int, v T) {
	s.ensureOwned()
	s.data[i] = v
}

// Append appends elements. If the slice is shared, it makes a private copy first.
func (s *SharedSlice[T]) Append(items ...T) {
	s.ensureOwned()
	s.data = append(s.data, items...)
}

// Data returns the underlying slice (read-only access).
func (s SharedSlice[T]) Data() []T {
	return s.data
}

// Release decrements the reference count. If it reaches zero, the data is eligible for GC.
func (s *SharedSlice[T]) Release() {
	if s.refs != nil && atomic.AddInt32(s.refs, -1) == 0 {
		s.data = nil
		s.refs = nil
	}
}

func (s *SharedSlice[T]) ensureOwned() {
	if s.refs == nil {
		return
	}
	if atomic.LoadInt32(s.refs) > 1 {
		// Copy on write
		newData := make([]T, len(s.data))
		copy(newData, s.data)
		atomic.AddInt32(s.refs, -1)
		s.data = newData
		refs := int32(1)
		s.refs = &refs
	}
}

// ---------------------------------------------------------------------------
// Frame-local pool — reuse objects across frames
// ---------------------------------------------------------------------------

// FramePool is a pool that resets all objects at the end of each frame.
type FramePool[T any] struct {
	mu       sync.Mutex
	objects  []*T
	factory  func() *T
	maxSize  int
}

// NewFramePool creates a frame pool with a factory function.
func NewFramePool[T any](factory func() *T, maxSize int) *FramePool[T] {
	if maxSize <= 0 {
		maxSize = 256
	}
	return &FramePool[T]{
		objects: make([]*T, 0, maxSize),
		factory: factory,
		maxSize: maxSize,
	}
}

// Get returns an object from the pool, or creates a new one.
func (p *FramePool[T]) Get() *T {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.objects) > 0 {
		obj := p.objects[len(p.objects)-1]
		p.objects = p.objects[:len(p.objects)-1]
		return obj
	}

	return p.factory()
}

// Put returns an object to the pool.
func (p *FramePool[T]) Put(obj *T) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.objects) < p.maxSize {
		p.objects = append(p.objects, obj)
	}
}

// Reset clears the pool, releasing all objects for GC.
func (p *FramePool[T]) Reset() {
	p.mu.Lock()
	p.objects = p.objects[:0]
	p.mu.Unlock()
}

// Len returns the number of available objects.
func (p *FramePool[T]) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.objects)
}

// ---------------------------------------------------------------------------
// Cell Buffer Pool — reuse render buffers
// ---------------------------------------------------------------------------

// CellBufferPool reuses SceneBuffer-sized cell arrays.
type CellBufferPool struct {
	mu      sync.Mutex
	buffers [][]byte
	size    int
}

// NewCellBufferPool creates a pool for cell buffers of the given size.
func NewCellBufferPool(cellSize int) *CellBufferPool {
	return &CellBufferPool{
		buffers: make([][]byte, 0, 16),
		size:    cellSize,
	}
}

// Get returns a buffer, or allocates a new one.
func (p *CellBufferPool) Get() []byte {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.buffers) > 0 {
		buf := p.buffers[len(p.buffers)-1]
		p.buffers = p.buffers[:len(p.buffers)-1]
		return buf
	}

	return make([]byte, p.size)
}

// Put returns a buffer to the pool.
func (p *CellBufferPool) Put(buf []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(buf) == p.size && len(p.buffers) < 32 {
		p.buffers = append(p.buffers, buf)
	}
}

// Reset clears the pool.
func (p *CellBufferPool) Reset() {
	p.mu.Lock()
	p.buffers = p.buffers[:0]
	p.mu.Unlock()
}
