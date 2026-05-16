// Package usagelog records per-key API usage events and ships them in batches
// to the Laravel log-ingestion endpoint.
//
// Pipeline:
//
//	handler.Lookup() → Logger.Record() → buffered channel
//	                                   → background goroutine
//	                                   → HTTP POST /api/internal/logs/ingest
//	                                   → Laravel IngestLogBatchJob
//	                                   → api_call_logs table
//	                                   → AggregateUsageCountersJob (every 1 min)
//	                                   → usage_counters table
package usagelog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Event is one authenticated API call that should be counted against a key.
type Event struct {
	KeyID     string // ULID from valid_keys.json
	BIN       string
	Country   string
	Result    string // ENABLE | DISABLE | …
	ElapsedNs int64
}

// formatLine produces the exact log line the Laravel GoEngineLogParser expects:
//
//	[RFC3339Nano] kid=<key_id> BIN=<bin> Country=<country> Result=<result> Elapsed=<ns>ns
func formatLine(ev Event, ts time.Time) string {
	return fmt.Sprintf("[%s] kid=%s BIN=%s Country=%s Result=%s Elapsed=%dns",
		ts.UTC().Format(time.RFC3339Nano),
		ev.KeyID, ev.BIN, ev.Country, ev.Result, ev.ElapsedNs)
}

type taggedEvent struct {
	ev Event
	ts time.Time
}

// Logger batches usage events and ships them to the Laravel ingest endpoint.
//
// Design goals:
//   - Record() must never block — lookup latency is sacred.
//   - Events are buffered in a channel (default 50 000 slots).
//   - A single background goroutine drains the channel and ships batches.
//   - If the channel is full, the event is silently dropped (one missed log
//     is always better than increased p99 latency).
//   - On graceful shutdown (close done) the goroutine drains the remaining
//     events and ships them before returning.
type Logger struct {
	ch       chan taggedEvent
	endpoint string
	secret   string
}

// NewLogger creates a Logger.
//
//	bufSize  — channel depth; tune to your burst rate × flush interval
//	endpoint — full URL of /api/internal/logs/ingest (empty string = no-op)
//	secret   — value for the X-Internal-Secret header
func NewLogger(bufSize int, endpoint, secret string) *Logger {
	return &Logger{
		ch:       make(chan taggedEvent, bufSize),
		endpoint: endpoint,
		secret:   secret,
	}
}

// Record enqueues ev for async shipping. Never blocks.
func (l *Logger) Record(ev Event) {
	select {
	case l.ch <- taggedEvent{ev: ev, ts: time.Now().UTC()}:
	default:
		// Buffer full — silently drop to protect lookup latency.
	}
}

// Start spawns the background worker. Call once after NewLogger.
//
//	batchSize  — max lines per HTTP POST (matches log_ingestion_batch_size on Laravel)
//	flushEvery — timer that flushes even if batchSize is not reached
//	done       — closed on graceful shutdown; worker drains then exits
func (l *Logger) Start(batchSize int, flushEvery time.Duration, done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(flushEvery)
		defer ticker.Stop()

		batch := make([]taggedEvent, 0, batchSize)

		flush := func() {
			if len(batch) == 0 {
				return
			}
			lines := make([]string, len(batch))
			for i, e := range batch {
				lines[i] = formatLine(e.ev, e.ts)
			}
			batch = batch[:0]
			if err := l.ship(lines); err != nil {
				log.Printf("⚠️  usagelog: ship failed: %v", err)
			}
		}

		for {
			select {
			case e := <-l.ch:
				batch = append(batch, e)
				if len(batch) >= batchSize {
					flush()
				}

			case <-ticker.C:
				flush()

			case <-done:
				// Drain remaining events before returning.
			drain:
				for {
					select {
					case e := <-l.ch:
						batch = append(batch, e)
					default:
						break drain
					}
				}
				flush()
				return
			}
		}
	}()
}

// ship POSTs lines as JSON to the Laravel ingest endpoint.
// No-op if endpoint is empty (useful in dev/test without Laravel running).
func (l *Logger) ship(lines []string) error {
	if l.endpoint == "" {
		return nil
	}

	body, err := json.Marshal(map[string][]string{"lines": lines})
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, l.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", l.secret)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
