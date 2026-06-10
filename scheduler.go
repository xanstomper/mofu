package mofu

import (
	"sync"
	"time"
)

type FrameTick struct {
	Delta   time.Duration
	Elapsed time.Duration
	Frame   int64
	FPS     float64
}

type Scheduler struct {
	fps        float64
	frameCh    chan FrameTick
	done       chan struct{}
	running    bool
	frameCount int64
	startTime  time.Time
	lastTick   time.Time
	mu         sync.Mutex
}

func NewScheduler(fps float64) *Scheduler {
	return &Scheduler{
		fps:     fps,
		frameCh: make(chan FrameTick, 8),
		done:    make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.startTime = time.Now()
	s.lastTick = s.startTime
	s.frameCount = 0
	s.mu.Unlock()
	go s.run()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.done)
}

func (s *Scheduler) FrameCh() <-chan FrameTick {
	return s.frameCh
}

func (s *Scheduler) FPS() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fps
}

func (s *Scheduler) SetFPS(fps float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fps = fps
}

func (s *Scheduler) run() {
	interval := time.Duration(float64(time.Second) / s.fps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case now := <-ticker.C:
			s.mu.Lock()
			if !s.running {
				s.mu.Unlock()
				return
			}
			s.frameCount++
			delta := now.Sub(s.lastTick)
			elapsed := now.Sub(s.startTime)
			tick := FrameTick{
				Delta:   delta,
				Elapsed: elapsed,
				Frame:   s.frameCount,
				FPS:     s.fps,
			}
			s.lastTick = now
			s.mu.Unlock()

			select {
			case s.frameCh <- tick:
			default:
			}
		}
	}
}
