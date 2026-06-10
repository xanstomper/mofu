package mofu

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type WorkerTask struct {
	ID       string
	Priority int
	Work     func(ctx context.Context, progress func(float64)) Msg
	OnDone   func(Msg)
}

type WorkerPool struct {
	mu            sync.Mutex
	tasks         []WorkerTask
	maxConcurrent int
	sem           chan struct{}
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	running       atomic.Bool
}

func NewWorkerPool(maxConcurrent int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		maxConcurrent: maxConcurrent,
		sem:           make(chan struct{}, maxConcurrent),
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (wp *WorkerPool) Start() {
	wp.running.Store(true)
}

func (wp *WorkerPool) Stop() {
	wp.running.Store(false)
	wp.cancel()
	wp.wg.Wait()
}

func (wp *WorkerPool) Submit(task WorkerTask) {
	wp.mu.Lock()
	wp.tasks = append(wp.tasks, task)
	wp.mu.Unlock()
}

func (wp *WorkerPool) processLoop() {
	for wp.running.Load() {
		wp.mu.Lock()
		if len(wp.tasks) == 0 {
			wp.mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}
		best := 0
		for i := 1; i < len(wp.tasks); i++ {
			if wp.tasks[i].Priority > wp.tasks[best].Priority {
				best = i
			}
		}
		task := wp.tasks[best]
		wp.tasks = append(wp.tasks[:best], wp.tasks[best+1:]...)
		wp.mu.Unlock()

		wp.wg.Add(1)
		go func(t WorkerTask) {
			defer wp.wg.Done()
			wp.sem <- struct{}{}
			defer func() { <-wp.sem }()

			progress := func(pct float64) {}
			msg := t.Work(wp.ctx, progress)
			if t.OnDone != nil {
				t.OnDone(msg)
			}
		}(task)
	}
}

type TaskGroup struct {
	pool    *WorkerPool
	tasks   []WorkerTask
	results []Msg
	mu      sync.Mutex
}

func (wp *WorkerPool) Group() *TaskGroup {
	return &TaskGroup{pool: wp}
}

func (g *TaskGroup) Add(id string, work func(ctx context.Context, progress func(float64)) Msg) {
	g.tasks = append(g.tasks, WorkerTask{ID: id, Work: work})
}

func (g *TaskGroup) RunAll(ctx context.Context) []Msg {
	var wg sync.WaitGroup
	for _, task := range g.tasks {
		wg.Add(1)
		go func(t WorkerTask) {
			defer wg.Done()
			msg := t.Work(ctx, func(pct float64) {})
			g.mu.Lock()
			g.results = append(g.results, msg)
			g.mu.Unlock()
		}(task)
	}
	wg.Wait()
	return g.results
}

type Progress struct {
	mu       sync.Mutex
	Value    float64
	Label    string
	OnChange func(float64)
}

func NewProgress() *Progress {
	return &Progress{}
}

func (p *Progress) Set(v float64) {
	p.mu.Lock()
	p.Value = v
	fn := p.OnChange
	p.mu.Unlock()
	if fn != nil {
		fn(v)
	}
}

func (p *Progress) Add(delta float64) {
	p.mu.Lock()
	p.Value += delta
	if p.Value > 1 {
		p.Value = 1
	}
	fn := p.OnChange
	p.mu.Unlock()
	if fn != nil {
		fn(p.Value)
	}
}
