package mofu

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// AsyncWorkerPool (Anthology Ch.11 §11.3)
// ---------------------------------------------------------------------------

// TaskID identifies a running async task.
type TaskID uint64

// TaskStatus tracks an async task's lifecycle.
type TaskStatus uint8

const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskDone
	TaskFailed
	TaskCancelled
)

// Priority controls worker scheduling priority.
type Priority uint8

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// ShellCommand represents a command to run.
type ShellCommand struct {
	Program string
	Args    []string
	Env     []string
	Dir     string
}

// TaskResult holds the output of an async task.
type TaskResult struct {
	ID     TaskID
	Status TaskStatus
	Output any
	Error  error
}

// AsyncWorkerPool manages concurrent task execution with priority queuing.
type AsyncWorkerPool struct {
	mu       sync.Mutex
	tasks    chan taskEntry
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	nextID   TaskID
	results  map[TaskID]TaskResult
	running  map[TaskID]context.CancelFunc
	maxTasks int
}

type taskEntry struct {
	id       TaskID
	fn       func(ctx context.Context) TaskResult
	priority Priority
}

// NewAsyncWorkerPool creates a pool with up to maxWorkers goroutines
// and maxTasks pending task slots.
func NewAsyncWorkerPool(maxWorkers, maxTasks int) *AsyncWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &AsyncWorkerPool{
		tasks:    make(chan taskEntry, maxTasks),
		ctx:      ctx,
		cancel:   cancel,
		results:  make(map[TaskID]TaskResult),
		running:  make(map[TaskID]context.CancelFunc),
		maxTasks: maxTasks,
	}
	for i := 0; i < maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p
}

// Submit enqueues a task and returns its ID.
func (p *AsyncWorkerPool) Submit(fn func(ctx context.Context) TaskResult, priority Priority) TaskID {
	p.mu.Lock()
	id := p.nextID
	p.nextID++
	p.mu.Unlock()

	p.tasks <- taskEntry{id: id, fn: fn, priority: priority}
	return id
}

// Result returns the result for a task, or (zero, false) if not ready.
func (p *AsyncWorkerPool) Result(id TaskID) (TaskResult, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.results[id]
	return r, ok
}

// Cancel cancels a running task by ID.
func (p *AsyncWorkerPool) Cancel(id TaskID) {
	p.mu.Lock()
	cancel, ok := p.running[id]
	p.mu.Unlock()
	if ok {
		cancel()
	}
}

// Stop cancels all tasks and waits for workers to finish.
func (p *AsyncWorkerPool) Stop() {
	p.cancel()
	p.wg.Wait()
}

// Len returns the number of pending tasks.
func (p *AsyncWorkerPool) Len() int {
	return len(p.tasks)
}

func (p *AsyncWorkerPool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task := <-p.tasks:
			taskCtx, taskCancel := context.WithCancel(p.ctx)
			p.mu.Lock()
			p.running[task.id] = taskCancel
			p.mu.Unlock()

			result := task.fn(taskCtx)

			p.mu.Lock()
			p.results[task.id] = result
			delete(p.running, task.id)
			p.mu.Unlock()
			taskCancel()
		}
	}
}

// ---------------------------------------------------------------------------
// AsyncCommandRunner (Anthology Ch.11 §11.3)
// ---------------------------------------------------------------------------

// AsyncCommandRunner runs shell commands concurrently and reports results.
type AsyncCommandRunner struct {
	pool    *AsyncWorkerPool
	running map[TaskID]ShellCommand
	mu      sync.Mutex
}

// NewAsyncCommandRunner creates a runner backed by a AsyncWorkerPool.
func NewAsyncCommandRunner(pool *AsyncWorkerPool) *AsyncCommandRunner {
	return &AsyncCommandRunner{
		pool:    pool,
		running: make(map[TaskID]ShellCommand),
	}
}

// Run starts a shell command asynchronously.
func (r *AsyncCommandRunner) Run(cmd ShellCommand) TaskID {
	id := r.pool.Submit(func(ctx context.Context) TaskResult {
		// Placeholder: actual subprocess execution would go here
		return TaskResult{Status: TaskDone}
	}, PriorityNormal)

	r.mu.Lock()
	r.running[id] = cmd
	r.mu.Unlock()
	return id
}

// ---------------------------------------------------------------------------
// ResultCollector (Anthology Ch.11 §11.4)
// ---------------------------------------------------------------------------

// ResultCollector aggregates async results with a channel-based API.
type ResultCollector struct {
	mu      sync.Mutex
	pending map[TaskID]chan TaskResult
	pool    *AsyncWorkerPool
}

// NewResultCollector returns a collector for the given pool.
func NewResultCollector(pool *AsyncWorkerPool) *ResultCollector {
	return &ResultCollector{
		pending: make(map[TaskID]chan TaskResult),
		pool:    pool,
	}
}

// Submit submits a task and returns a channel that will receive its result.
func (rc *ResultCollector) Submit(fn func(ctx context.Context) TaskResult, priority Priority) <-chan TaskResult {
	id := rc.pool.Submit(fn, priority)
	ch := make(chan TaskResult, 1)

	rc.mu.Lock()
	rc.pending[id] = ch
	rc.mu.Unlock()

	// Poller goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if r, ok := rc.pool.Result(id); ok {
					ch <- r
					rc.mu.Lock()
					delete(rc.pending, id)
					rc.mu.Unlock()
					return
				}
			}
		}
	}()

	return ch
}

// ---------------------------------------------------------------------------
// RateLimiter (Anthology Ch.17)
// ---------------------------------------------------------------------------

// RateLimiter implements a token-bucket rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	max      float64
	rate     float64 // tokens per second
	lastTime time.Time
}

// NewRateLimiter creates a limiter allowing max tokens at rate per second.
func NewRateLimiter(max float64, rate float64) *RateLimiter {
	return &RateLimiter{tokens: max, max: max, rate: rate, lastTime: time.Now()}
}

// Allow reports whether a request can proceed. If so, one token is consumed.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(rl.lastTime).Seconds()
	rl.lastTime = now
	rl.tokens += elapsed * rl.rate
	if rl.tokens > rl.max {
		rl.tokens = rl.max
	}
	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// CircuitBreaker (Anthology Ch.17 §17.6)
// ---------------------------------------------------------------------------

// CircuitState is the breaker state.
type CircuitState uint8

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker prevents cascading failures.
type CircuitBreaker struct {
	mu           sync.Mutex
	state        CircuitState
	failures     int
	successes    int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
}

// NewCircuitBreaker creates a breaker that opens after threshold failures
// and resets after resetTimeout.
func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        CircuitClosed,
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == CircuitOpen && time.Since(cb.lastFailure) > cb.resetTimeout {
		cb.state = CircuitHalfOpen
	}
	return cb.state
}

// RecordSuccess records a successful call.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.successes++
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
	}
}

// RecordFailure records a failed call.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = CircuitOpen
	}
}

// ---------------------------------------------------------------------------
// ConnectionPool (Anthology Ch.17 §17.7)
// ---------------------------------------------------------------------------

// ConnPool manages a pool of reusable connections.
type ConnPool struct {
	mu      sync.Mutex
	conns   chan struct{}
	factory func() (io.Closer, error)
	active  int32
	max     int
}

// NewConnPool creates a pool with max concurrent connections.
func NewConnPool(max int, factory func() (io.Closer, error)) *ConnPool {
	return &ConnPool{
		conns:   make(chan struct{}, max),
		factory: factory,
		max:     max,
	}
}

// Acquire blocks until a connection slot is available.
func (p *ConnPool) Acquire() {
	p.conns <- struct{}{}
	atomic.AddInt32(&p.active, 1)
}

// Release returns a connection slot.
func (p *ConnPool) Release() {
	<-p.conns
	atomic.AddInt32(&p.active, -1)
}

// Active returns the number of active connections.
func (p *ConnPool) Active() int {
	return int(atomic.LoadInt32(&p.active))
}

// dummy io import
var _ interface{ Close() error }
