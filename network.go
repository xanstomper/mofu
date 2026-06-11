package mofu

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Networking (Anthology Ch.17)
// ---------------------------------------------------------------------------

// NetworkError wraps network failures with retryability.
type NetworkError struct {
	Err       error
	Retryable bool
}

func (e *NetworkError) Error() string { return e.Err.Error() }
func (e *NetworkError) Unwrap() error { return e.Err }

// RetryPolicy controls retries for HTTP operations.
type RetryPolicy struct {
	MaxRetries int
	Backoff    time.Duration
	Jitter     bool
}

// HTTPClient is a small wrapper around net/http with retry semantics.
type HTTPClient struct {
	client  *http.Client
	Policy  RetryPolicy
	BaseURL string
	mu      sync.Mutex
}

// NewHTTPClient returns an HTTPClient.
func NewHTTPClient(baseURL string, timeout time.Duration, policy RetryPolicy) *HTTPClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &HTTPClient{client: &http.Client{Timeout: timeout}, Policy: policy, BaseURL: baseURL}
}

// Do executes an HTTP request with retries.
func (c *HTTPClient) Do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Policy.MaxRetries; attempt++ {
		req, err := c.newRequest(ctx, method, path, body)
		if err != nil {
			return nil, &NetworkError{Err: err, Retryable: false}
		}
		resp, err := c.client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return resp, &NetworkError{Err: errors.New(resp.Status), Retryable: false}
		}
		if err != nil {
			lastErr = &NetworkError{Err: err, Retryable: true}
		} else {
			lastErr = &NetworkError{Err: errors.New(resp.Status), Retryable: true}
			resp.Body.Close()
		}
		if attempt < c.Policy.MaxRetries {
			wait := c.Policy.Backoff * time.Duration(attempt+1)
			if c.Policy.Jitter && wait > 0 {
				wait += time.Duration((time.Now().UnixNano() % 1000) / 1000 * int64(c.Policy.Backoff))
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
	}
	return nil, lastErr
}

func (c *HTTPClient) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	url := c.BaseURL + path
	return http.NewRequestWithContext(ctx, method, url, nil)
}

// WebSocketClient is a placeholder for a future WebSocket implementation.
type WebSocketClient struct {
	URL       string
	Headers   http.Header
	Reconnect bool
}

// SSEEvent is a server-sent event.
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry time.Duration
}

// SSEClient consumes server-sent events.
type SSEClient struct {
	client  *http.Client
	BaseURL string
}

// NewSSEClient returns an SSE client.
func NewSSEClient(baseURL string) *SSEClient {
	return &SSEClient{client: &http.Client{}, BaseURL: baseURL}
}

// Open starts an SSE stream and calls handler for each event.
func (s *SSEClient) Open(ctx context.Context, handler func(SSEEvent)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return readSSE(resp.Body, handler)
}

func readSSE(r sseReader, handler func(SSEEvent)) error {
	scanner := bufio.NewScanner(r)
	var ev SSEEvent
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if handler != nil {
				handler(ev)
			}
			ev = SSEEvent{}
			continue
		}
		switch {
		case strings.HasPrefix(line, "event:"):
			ev.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			if ev.Data != "" {
				ev.Data += "\n"
			}
			ev.Data += strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		case strings.HasPrefix(line, "id:"):
			ev.ID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		case strings.HasPrefix(line, "retry:"):
			if n, err := time.ParseDuration(strings.TrimSpace(strings.TrimPrefix(line, "retry:")) + "ms"); err == nil {
				ev.Retry = n
			}
		}
	}
	return scanner.Err()
}

type sseReader interface{ Read([]byte) (int, error) }
