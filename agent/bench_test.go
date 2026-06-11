package agent

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/xanstomper/mofu"
)

// =========================================================================
// Benchmarks — proving MOFU agent framework is faster
// =========================================================================

func BenchmarkRingBufferWrite1KB(b *testing.B) {
	rb := NewRingBuffer(1024 * 1024)
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Write(data)
	}
}

func BenchmarkRingBufferWrite4KB(b *testing.B) {
	rb := NewRingBuffer(4 * 1024 * 1024)
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Write(data)
	}
}

func BenchmarkRingBufferRead1KB(b *testing.B) {
	rb := NewRingBuffer(1024 * 1024)
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte('a')
	}
	for i := 0; i < 1000; i++ {
		rb.Write(data)
	}
	out := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Read(out)
	}
}

func BenchmarkSSEParserSingleEvent(b *testing.B) {
	parser := NewSSEParser()
	data := []byte("event: message\ndata: {\"text\": \"hello world\"}\n\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Feed(data)
	}
}

func BenchmarkSSEParserBatch(b *testing.B) {
	parser := NewSSEParser()
	var buf []byte
	for i := 0; i < 100; i++ {
		buf = append(buf, fmt.Sprintf("event: token\ndata: {\"token\": \"word_%d\"}\n\n", i)...)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Feed(buf)
	}
}

func BenchmarkStreamingBufferAppend(b *testing.B) {
	sb := NewStreamingBuffer(10000, 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.AppendToken("hello ")
	}
}

func BenchmarkStreamingBufferAppendNewline(b *testing.B) {
	sb := NewStreamingBuffer(10000, 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.AppendToken("line content")
		sb.Newline()
	}
}

func BenchmarkStreamTokenizer(b *testing.B) {
	st := NewStreamTokenizer()
	data := []byte("the quick brown fox jumps over the lazy dog\nand another line\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st.Feed(data)
	}
}

func BenchmarkVirtualScrollAppend(b *testing.B) {
	vs := NewVirtualScroll()
	text := strings.Repeat("x", 80)
	seg := []renderSegment{{text: text, fg: mofu.ColorBlack, bg: mofu.ColorBlack}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.AppendLine(seg)
	}
}

func BenchmarkVirtualScrollScrollDown(b *testing.B) {
	vs := NewVirtualScroll()
	text := strings.Repeat("x", 80)
	for i := 0; i < 100000; i++ {
		vs.AppendLine([]renderSegment{{text: text, fg: mofu.ColorBlack, bg: mofu.ColorBlack}})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.ScrollDown(1)
	}
}

func BenchmarkVirtualScrollPageDown(b *testing.B) {
	vs := NewVirtualScroll()
	text := strings.Repeat("x", 80)
	for i := 0; i < 100000; i++ {
		vs.AppendLine([]renderSegment{{text: text, fg: mofu.ColorBlack, bg: mofu.ColorBlack}})
	}
	vs.mu.Lock()
	vs.viewHeight = 40
	vs.mu.Unlock()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vs.PageDown()
	}
}

func BenchmarkMassiveLogViewerAddLine(b *testing.B) {
	mlv := NewMassiveLogViewer()
	line := LogLine{
		Timestamp: "15:04:05.000",
		Level:     "info",
		Source:    "api",
		Message:   "GET /api/users 200 OK 12ms",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mlv.AddLine(line)
	}
}

func BenchmarkMassiveLogViewerBulkAdd(b *testing.B) {
	lines := make([]LogLine, 1000)
	for i := range lines {
		lines[i] = LogLine{
			Timestamp: fmt.Sprintf("15:04:%02d.%03d", i%60, i%1000),
			Level:     "info",
			Source:    "api",
			Message:   "GET /api/users 200 OK",
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mlv := NewMassiveLogViewer()
		mlv.AddLines(lines)
	}
}

func BenchmarkAgentStreamDisplayWriteToken(b *testing.B) {
	d := NewAgentStreamDisplay(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.WriteToken("a")
	}
}

func BenchmarkAgentStreamDisplayWriteChunk(b *testing.B) {
	d := NewAgentStreamDisplay(10000)
	chunk := "Hello world this is a streaming token response from the API endpoint\n"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.WriteChunk(chunk)
	}
}

func BenchmarkBatchedInputPush(b *testing.B) {
	bi := NewBatchedInput(time.Millisecond, func(events []keyEvent) {})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bi.Push("a", 'a', false)
	}
}

// =========================================================================
// Comparison: MOFU vs string-append approach
// =========================================================================

func BenchmarkStringAppend(b *testing.B) {
	var result string
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result += "hello "
		_ = result
	}
}

func BenchmarkBytesBuffer(b *testing.B) {
	var buf []byte
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = append(buf, "hello "...)
		_ = buf
	}
}

func BenchmarkRingBufferVsString(b *testing.B) {
	rb := NewRingBuffer(1024 * 1024)
	data := []byte("hello ")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Write(data)
	}
}

// =========================================================================
// Throughput test: tokens per second
// =========================================================================

func TestStreamingThroughput(b *testing.T) {
	sb := NewStreamingBuffer(100000, 50)
	token := "The quick brown fox jumps over the lazy dog. "

	start := time.Now()
	count := 0
	for time.Since(start) < time.Second {
		sb.AppendToken(token)
		count++
	}
	elapsed := time.Since(start)
	tokensPerSec := float64(count) / elapsed.Seconds()
	bytesPerSec := float64(count*len(token)) / elapsed.Seconds()

	b.Logf("StreamingBuffer: %.0f tokens/sec, %.0f MB/sec", tokensPerSec, bytesPerSec/1024/1024)
}

func TestSSEParserThroughput(b *testing.T) {
	parser := NewSSEParser()
	var buf []byte
	for i := 0; i < 1000; i++ {
		buf = append(buf, fmt.Sprintf("event: token\ndata: {\"content\": \"word_%d token_text\"}\n\n", i)...)
	}

	start := time.Now()
	totalEvents := 0
	for time.Since(start) < time.Second {
		events := parser.Feed(buf)
		totalEvents += len(events)
	}
	elapsed := time.Since(start)
	eventsPerSec := float64(totalEvents) / elapsed.Seconds()

	b.Logf("SSEParser: %.0f events/sec", eventsPerSec)
}

func TestVirtualScrollLargeDataset(t *testing.T) {
	vs := NewVirtualScroll()
	text := strings.Repeat("Log line content with some data ", 3)

	start := time.Now()
	for i := 0; i < 1_000_000; i++ {
		vs.AppendLine([]renderSegment{{text: text, fg: mofu.ColorBlack, bg: mofu.ColorBlack}})
	}
	elapsed := time.Since(start)

	if vs.Len() != 1_000_000 {
		t.Errorf("expected 1M lines, got %d", vs.Len())
	}

	t.Logf("VirtualScroll: inserted 1M lines in %v (%.0f lines/sec)", elapsed, float64(1_000_000)/elapsed.Seconds())
}
