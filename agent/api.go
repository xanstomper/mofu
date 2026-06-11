package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// =========================================================================
// 1. API Streaming — connect agents to real streaming APIs
// OpenAI, Anthropic, Ollama compatible.
// =========================================================================

// StreamChunk is a single token/chunk from an API stream.
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
	Usage   *TokenUsage
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamRequest is the request body for streaming API calls.
type StreamRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamResponse is the parsed SSE data from streaming APIs.
type StreamResponse struct {
	ID      string         `json:"id"`
	Choices []StreamChoice `json:"choices"`
	Usage   *TokenUsage    `json:"usage"`
}

type StreamChoice struct {
	Delta struct {
		Content string `json:"content"`
		Role    string `json:"role"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

// APIStream connects to a streaming API and yields chunks.
// Works with OpenAI-compatible endpoints (OpenAI, Anthropic, Ollama, etc).
type APIStream struct {
	URL         string
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	HTTPClient  *http.Client
	mu          sync.Mutex
}

func NewAPIStream(url, apiKey, model string) *APIStream {
	return &APIStream{
		URL:         url,
		APIKey:      apiKey,
		Model:       model,
		MaxTokens:   4096,
		Temperature: 0.7,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Stream sends messages and returns a channel of chunks.
func (a *APIStream) Stream(messages []Message) <-chan StreamChunk {
	ch := make(chan StreamChunk, 64)

	go func() {
		defer close(ch)

		reqBody := StreamRequest{
			Model:       a.Model,
			Messages:    messages,
			MaxTokens:   a.MaxTokens,
			Temperature: a.Temperature,
			Stream:      true,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}

		req, err := http.NewRequest("POST", a.URL, bytes.NewReader(body))
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		if a.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+a.APIKey)
		}

		resp, err := a.HTTPClient.Do(req)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			ch <- StreamChunk{Error: fmt.Errorf("API error %d: %s", resp.StatusCode, string(errBody))}
			return
		}

		parser := NewSSEParser()
		buf := make([]byte, 4096)

		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				events := parser.Feed(buf[:n])
				for _, event := range events {
					chunk := parseStreamEvent(event.Data)
					if chunk != nil {
						ch <- *chunk
						if chunk.Done || chunk.Error != nil {
							return
						}
					}
				}
			}
			if err != nil {
				ch <- StreamChunk{Done: true}
				return
			}
		}
	}()

	return ch
}

func parseStreamEvent(data string) *StreamChunk {
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" {
		return &StreamChunk{Done: true}
	}

	var resp StreamResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil
	}

	chunk := &StreamChunk{}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		chunk.Content = choice.Delta.Content
		if choice.FinishReason != "" {
			chunk.Done = true
		}
	}

	if resp.Usage != nil {
		chunk.Usage = &TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return chunk
}

// =========================================================================
// 2. Live Data Feed — generic streaming data with backpressure
// =========================================================================

// LiveDataFeed is a generic channel-based data feed with buffering.
type LiveDataFeed[T any] struct {
	ch       chan T
	bufSize  int
	dropped  int64
	mu       sync.Mutex
	closed   bool
}

func NewLiveDataFeed[T any](bufSize int) *LiveDataFeed[T] {
	return &LiveDataFeed[T]{
		ch:      make(chan T, bufSize),
		bufSize: bufSize,
	}
}

func (f *LiveDataFeed[T]) Send(data T) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return false
	}

	select {
	case f.ch <- data:
		return true
	default:
		f.dropped++
		// Drop oldest, push new
		select {
		case <-f.ch:
		default:
		}
		f.ch <- data
		return true
	}
}

func (f *LiveDataFeed[T]) Recv() (T, bool) {
	data, ok := <-f.ch
	return data, ok
}

func (f *LiveDataFeed[T]) RecvTimeout(d time.Duration) (T, bool) {
	select {
	case data := <-f.ch:
		return data, true
	case <-time.After(d):
		var zero T
		return zero, false
	}
}

func (f *LiveDataFeed[T]) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		close(f.ch)
	}
}

func (f *LiveDataFeed[T]) Dropped() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dropped
}

func (f *LiveDataFeed[T]) Len() int {
	return len(f.ch)
}

// =========================================================================
// 3. Instant Render Pipeline — zero-latency display updates
// =========================================================================

// RenderPipeline processes data → render at maximum speed.
type RenderPipeline struct {
	inputCh    chan RenderCommand
	outputCh   chan []byte
	mu         sync.RWMutex
	fps        int
	paused     bool
	lastFrame  time.Time
	frameCount int64
	dropCount  int64
}

type RenderCommand struct {
	Type   string
	X, Y   int
	Text   string
	Fg, Bg string
	Attrs  string
}

func NewRenderPipeline(fps int) *RenderPipeline {
	rp := &RenderPipeline{
		inputCh:  make(chan RenderCommand, 1024),
		outputCh: make(chan []byte, 64),
		fps:      fps,
	}
	return rp
}

func (rp *RenderPipeline) Submit(cmd RenderCommand) {
	select {
	case rp.inputCh <- cmd:
	default:
		rp.mu.Lock()
		rp.dropCount++
		rp.mu.Unlock()
	}
}

func (rp *RenderPipeline) SubmitBatch(cmds []RenderCommand) {
	for _, cmd := range cmds {
		rp.Submit(cmd)
	}
}

func (rp *RenderPipeline) Stats() (frameCount, dropCount int64) {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.frameCount, rp.dropCount
}

// =========================================================================
// 4. Instant Agent — agent with live API streaming built in
// =========================================================================

// InstantAgent is a production-ready agent that streams from real APIs.
type InstantAgent struct {
	mu           sync.RWMutex
	Name         string
	State        AgentState
	api          *APIStream
	stream       <-chan StreamChunk
	buffer       *StreamingBuffer
	history      []Message
	systemPrompt string
	output       string
	totalTokens  int
	totalCost    float64
	startTime    time.Time
	onToken      func(token string)
	onDone       func(output string)
	onError      func(err error)
	onToolCall   func(name, input string)
	toolHandlers map[string]func(input string) (string, error)
}

func NewInstantAgent(name, apiURL, apiKey, model string) *InstantAgent {
	return &InstantAgent{
		Name:         name,
		api:          NewAPIStream(apiURL, apiKey, model),
		buffer:       NewStreamingBuffer(10000, 50),
		startTime:    time.Now(),
		toolHandlers: make(map[string]func(string) (string, error)),
	}
}

func (ia *InstantAgent) SetSystemPrompt(prompt string) {
	ia.mu.Lock()
	ia.systemPrompt = prompt
	ia.mu.Unlock()
}

func (ia *InstantAgent) OnToken(fn func(string))     { ia.mu.Lock(); ia.onToken = fn; ia.mu.Unlock() }
func (ia *InstantAgent) OnDone(fn func(string))       { ia.mu.Lock(); ia.onDone = fn; ia.mu.Unlock() }
func (ia *InstantAgent) OnError(fn func(error))       { ia.mu.Lock(); ia.onError = fn; ia.mu.Unlock() }
func (ia *InstantAgent) OnToolCall(fn func(string, string)) { ia.mu.Lock(); ia.onToolCall = fn; ia.mu.Unlock() }

func (ia *InstantAgent) RegisterTool(name string, handler func(input string) (string, error)) {
	ia.mu.Lock()
	ia.toolHandlers[name] = handler
	ia.mu.Unlock()
}

func (ia *InstantAgent) Send(userMessage string) {
	ia.mu.Lock()
	ia.State = StateStreaming
	ia.output = ""

	messages := make([]Message, 0)
	if ia.systemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: ia.systemPrompt})
	}
	messages = append(messages, ia.history...)
	messages = append(messages, Message{Role: "user", Content: userMessage})
	ia.history = append(ia.history, Message{Role: "user", Content: userMessage})
	ia.mu.Unlock()

	ch := ia.api.Stream(messages)
	ia.stream = ch

	go ia.processStream(ch)
}

func (ia *InstantAgent) processStream(ch <-chan StreamChunk) {
	for chunk := range ch {
		if chunk.Error != nil {
			ia.mu.Lock()
			ia.State = StateError
			ia.mu.Unlock()
			if ia.onError != nil {
				ia.onError(chunk.Error)
			}
			return
		}

		if chunk.Content != "" {
			ia.mu.Lock()
			ia.output += chunk.Content
			ia.mu.Unlock()

			ia.buffer.AppendToken(chunk.Content)

			if ia.onToken != nil {
				ia.onToken(chunk.Content)
			}
		}

		if chunk.Usage != nil {
			ia.mu.Lock()
			ia.totalTokens += chunk.Usage.TotalTokens
			ia.mu.Unlock()
		}

		if chunk.Done {
			ia.mu.Lock()
			ia.State = StateDone
			ia.history = append(ia.history, Message{Role: "assistant", Content: ia.output})
			finalOutput := ia.output
			ia.mu.Unlock()

			if ia.onDone != nil {
				ia.onDone(finalOutput)
			}
			return
		}
	}
}

func (ia *InstantAgent) GetOutput() string {
	ia.mu.RLock()
	defer ia.mu.RUnlock()
	return ia.output
}

func (ia *InstantAgent) GetState() AgentState {
	ia.mu.RLock()
	defer ia.mu.RUnlock()
	return ia.State
}

func (ia *InstantAgent) GetBuffer() *StreamingBuffer {
	return ia.buffer
}

func (ia *InstantAgent) GetHistory() []Message {
	ia.mu.RLock()
	defer ia.mu.RUnlock()
	cp := make([]Message, len(ia.history))
	copy(cp, ia.history)
	return cp
}

func (ia *InstantAgent) Reset() {
	ia.mu.Lock()
	ia.history = nil
	ia.output = ""
	ia.totalTokens = 0
	ia.totalCost = 0
	ia.State = StateIdle
	ia.startTime = time.Now()
	ia.mu.Unlock()
	ia.buffer.Reset()
}
