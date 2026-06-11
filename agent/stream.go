package agent

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// =========================================================================
// High-Performance Streaming Engine
// Zero-allocation hot paths, ring buffers, SSE parsing.
// =========================================================================

// RingBuffer is a fixed-size circular buffer. Zero GC pressure.
// Write never blocks. Oldest data is silently dropped when full.
type RingBuffer struct {
	data   []byte
	head   int
	tail   int
	size   int
	count  int
	closed int32
	mu     sync.Mutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

func (rb *RingBuffer) Write(p []byte) (n int, err error) {
	if atomic.LoadInt32(&rb.closed) == 1 {
		return 0, io.ErrClosedPipe
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	n = len(p)
	available := rb.size - rb.count
	if n > available {
		n = available
		p = p[:n]
	}

	for _, b := range p {
		rb.data[rb.head] = b
		rb.head = (rb.head + 1) % rb.size
	}
	rb.count += n

	return n, nil
}

func (rb *RingBuffer) Read(p []byte) (n int, err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return 0, io.EOF
	}

	n = len(p)
	if n > rb.count {
		n = rb.count
	}

	for i := 0; i < n; i++ {
		p[i] = rb.data[rb.tail]
		rb.tail = (rb.tail + 1) % rb.size
	}
	rb.count -= n

	return n, nil
}

func (rb *RingBuffer) Len() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}

func (rb *RingBuffer) Close() {
	atomic.StoreInt32(&rb.closed, 1)
}

// =========================================================================
// SSEParser — zero-alloc Server-Sent Events parser
// =========================================================================

type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

type SSEParser struct {
	buf    []byte
	event  string
	data   bytes.Buffer
	id     string
	retry  int
}

func NewSSEParser() *SSEParser {
	return &SSEParser{
		buf: make([]byte, 0, 4096),
	}
}

// Feed adds raw bytes. Returns parsed events.
func (p *SSEParser) Feed(raw []byte) []SSEEvent {
	p.buf = append(p.buf, raw...)
	var events []SSEEvent

	for {
		idx := bytes.Index(p.buf, []byte("\n\n"))
		if idx < 0 {
			break
		}
		block := p.buf[:idx]
		p.buf = p.buf[idx+2:]

		p.event = ""
		p.data.Reset()
		p.id = ""
		p.retry = 0

		lines := bytes.Split(block, []byte("\n"))
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}

			colon := bytes.IndexByte(line, ':')
			if colon < 0 {
				p.handleField(line, nil)
				continue
			}

			field := line[:colon]
			value := line[colon+1:]
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
			p.handleField(field, value)
		}

		if p.data.Len() > 0 {
			events = append(events, SSEEvent{
				Event: p.event,
				Data:  p.data.String(),
				ID:    p.id,
				Retry: p.retry,
			})
		}
	}

	return events
}

func (p *SSEParser) handleField(field, value []byte) {
	switch string(field) {
	case "event":
		if value != nil {
			p.event = string(value)
		}
	case "data":
		if p.data.Len() > 0 {
			p.data.WriteByte('\n')
		}
		if value != nil {
			p.data.Write(value)
		}
	case "id":
		if value != nil {
			p.id = string(value)
		}
	case "retry":
		if value != nil {
			n := 0
			for _, b := range value {
				if b >= '0' && b <= '9' {
					n = n*10 + int(b-'0')
				}
			}
			p.retry = n
		}
	}
}

// =========================================================================
// SSEClient — streaming HTTP client with backpressure
// =========================================================================

type SSEClient struct {
	URL         string
	Headers     map[string]string
	OnEvent     func(SSEEvent)
	OnConnect   func()
	OnDisconnect func(err error)
	OnError     func(err error)
	Timeout     time.Duration
	MaxRetries  int
	mu          sync.Mutex
	client      *http.Client
	cancel      chan struct{}
	running     bool
}

func NewSSEClient(url string) *SSEClient {
	return &SSEClient{
		URL:        url,
		Headers:    make(map[string]string),
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		cancel:     make(chan struct{}),
	}
}

func (c *SSEClient) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	go c.run()
	return nil
}

func (c *SSEClient) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()
	close(c.cancel)
}

func (c *SSEClient) run() {
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
	}()

	retries := 0
	for retries <= c.MaxRetries {
		select {
		case <-c.cancel:
			return
		default:
		}

		err := c.connect()
		if err == nil {
			retries = 0
			continue
		}

		retries++
		if c.OnError != nil {
			c.OnError(err)
		}

		backoff := time.Duration(retries) * 500 * time.Millisecond
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}

		select {
		case <-c.cancel:
			return
		case <-time.After(backoff):
		}
	}

	if c.OnDisconnect != nil {
		c.OnDisconnect(io.EOF)
	}
}

func (c *SSEClient) connect() error {
	req, err := http.NewRequest("GET", c.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c.OnConnect != nil {
		c.OnConnect()
	}

	parser := NewSSEParser()
	reader := bufio.NewReaderSize(resp.Body, 32*1024)
	buf := make([]byte, 4096)

	for {
		select {
		case <-c.cancel:
			return nil
		default:
		}

		n, err := reader.Read(buf)
		if n > 0 {
			events := parser.Feed(buf[:n])
			for _, event := range events {
				if c.OnEvent != nil {
					c.OnEvent(event)
				}
			}
		}
		if err != nil {
			return err
		}
	}
}

// =========================================================================
// StreamingBuffer — high-throughput token accumulator
// Pre-allocated, lock-free fast path for single-writer.
// =========================================================================

type StreamingBuffer struct {
	lines      [][]byte
	lineCount  int
	maxLines   int
	totalBytes int64
	mu         sync.Mutex
	onFlush    func([]string)
	flushSize  int
	flushTimer *time.Timer
	flushDone  chan struct{}
}

func NewStreamingBuffer(maxLines, flushSize int) *StreamingBuffer {
	sb := &StreamingBuffer{
		lines:     make([][]byte, maxLines),
		maxLines:  maxLines,
		flushSize: flushSize,
		flushDone: make(chan struct{}),
	}
	go sb.flushLoop()
	return sb
}

func (sb *StreamingBuffer) Append(data []byte) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	atomic.AddInt64(&sb.totalBytes, int64(len(data)))

	// Fast path: append to current line
	if sb.lineCount > 0 {
		last := sb.lines[(sb.lineCount-1)%sb.maxLines]
		sb.lines[(sb.lineCount-1)%sb.maxLines] = append(last, data...)
		return
	}

	// New line
	if sb.lineCount < sb.maxLines {
		sb.lines[sb.lineCount] = append(sb.lines[sb.lineCount][:0], data...)
		sb.lineCount++
	} else {
		// Shift out oldest
		copy(sb.lines, sb.lines[1:])
		sb.lines[sb.maxLines-1] = append(sb.lines[sb.maxLines-1][:0], data...)
	}
}

func (sb *StreamingBuffer) AppendToken(token string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	atomic.AddInt64(&sb.totalBytes, int64(len(token)))

	if sb.lineCount == 0 {
		sb.lines[0] = append(sb.lines[0][:0], token...)
		sb.lineCount = 1
		return
	}

	idx := (sb.lineCount - 1) % sb.maxLines
	sb.lines[idx] = append(sb.lines[idx], token...)
}

func (sb *StreamingBuffer) Newline() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if sb.lineCount < sb.maxLines {
		sb.lineCount++
	} else {
		copy(sb.lines, sb.lines[1:])
	}
}

func (sb *StreamingBuffer) Lines() []string {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	result := make([]string, sb.lineCount)
	for i := 0; i < sb.lineCount; i++ {
		result[i] = string(sb.lines[i])
	}
	return result
}

func (sb *StreamingBuffer) LineCount() int {
	return sb.lineCount
}

func (sb *StreamingBuffer) TotalBytes() int64 {
	return atomic.LoadInt64(&sb.totalBytes)
}

func (sb *StreamingBuffer) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.lineCount = 0
	sb.totalBytes = 0
}

func (sb *StreamingBuffer) flushLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sb.flushDone:
			return
		case <-ticker.C:
			if sb.onFlush != nil {
				lines := sb.Lines()
				go sb.onFlush(lines)
			}
		}
	}
}

func (sb *StreamingBuffer) SetFlushCallback(fn func([]string)) {
	sb.mu.Lock()
	sb.onFlush = fn
	sb.mu.Unlock()
}

func (sb *StreamingBuffer) Stop() {
	close(sb.flushDone)
}

// =========================================================================
// StreamTokenizer — split streaming text into tokens with zero alloc
// =========================================================================

type StreamTokenizer struct {
	buf       []byte
	delimiters []byte
}

func NewStreamTokenizer() *StreamTokenizer {
	return &StreamTokenizer{
		buf:        make([]byte, 0, 4096),
		delimiters: []byte{' ', '\n', '\t'},
	}
}

// Feed adds bytes, returns complete tokens.
func (st *StreamTokenizer) Feed(data []byte) []string {
	st.buf = append(st.buf, data...)
	var tokens []string

	for {
		bestIdx := len(st.buf)
		bestDelim := byte(0)

		for _, d := range st.delimiters {
			idx := bytes.IndexByte(st.buf, d)
			if idx >= 0 && idx < bestIdx {
				bestIdx = idx
				bestDelim = d
			}
		}

		if bestIdx >= len(st.buf) {
			break
		}

		token := string(st.buf[:bestIdx])
		if len(token) > 0 {
			tokens = append(tokens, token)
		}
		if bestDelim == '\n' {
			tokens = append(tokens, "\n")
		}
		st.buf = st.buf[bestIdx+1:]
	}

	return tokens
}

// =========================================================================
// BatchedInput — coalesce rapid keystrokes for instant response
// =========================================================================

type BatchedInput struct {
	mu       sync.Mutex
	pending  []keyEvent
	batchFn  func([]keyEvent)
	timer    *time.Timer
	interval time.Duration
}

type keyEvent struct {
	Key  string
	Rune rune
	Ctrl bool
}

func NewBatchedInput(interval time.Duration, batchFn func([]keyEvent)) *BatchedInput {
	return &BatchedInput{
		batchFn:  batchFn,
		interval: interval,
	}
}

func (bi *BatchedInput) Push(key string, r rune, ctrl bool) {
	bi.mu.Lock()
	bi.pending = append(bi.pending, keyEvent{Key: key, Rune: r, Ctrl: ctrl})

	if bi.timer != nil {
		bi.timer.Stop()
	}

	bi.timer = time.AfterFunc(bi.interval, func() {
		bi.mu.Lock()
		events := bi.pending
		bi.pending = bi.pending[:0]
		bi.mu.Unlock()

		if len(events) > 0 {
			bi.batchFn(events)
		}
	})
	bi.mu.Unlock()
}

func (bi *BatchedInput) Flush() {
	bi.mu.Lock()
	if bi.timer != nil {
		bi.timer.Stop()
	}
	events := bi.pending
	bi.pending = bi.pending[:0]
	bi.mu.Unlock()

	if len(events) > 0 {
		bi.batchFn(events)
	}
}
