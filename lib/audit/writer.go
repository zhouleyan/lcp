package audit

import (
	"context"
	"sync"
	"time"

	"lcp.io/lcp/lib/logger"
)

// WriterConfig holds configuration for the async audit writer.
type WriterConfig struct {
	ChannelSize   int           // default 10000
	BatchSize     int           // default 100
	FlushInterval time.Duration // default 5s
}

// Writer is an async audit event writer that batches events before flushing to a Sink.
type Writer struct {
	sink   Sink
	cfg    WriterConfig
	ch     chan Event
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewWriter creates a new audit Writer.
func NewWriter(sink Sink, cfg WriterConfig) *Writer {
	if cfg.ChannelSize <= 0 {
		cfg.ChannelSize = 10000
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}
	return &Writer{
		sink: sink,
		cfg:  cfg,
		ch:   make(chan Event, cfg.ChannelSize),
	}
}

// Log enqueues an audit event. Non-blocking; drops with warning if channel is full.
func (w *Writer) Log(event Event) {
	select {
	case w.ch <- event:
	default:
		logger.Warnf("audit: channel full, dropping event: action=%s resource=%s", event.Action, event.ResourceType)
	}
}

// Start begins the background flush goroutine. Call Stop() to drain and stop.
func (w *Writer) Start(_ context.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.wg.Add(1)
	go w.run(ctx)
}

// Stop signals the writer to stop and waits for all buffered events to be flushed.
func (w *Writer) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
}

func (w *Writer) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.cfg.FlushInterval)
	defer ticker.Stop()

	batch := make([]Event, 0, w.cfg.BatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// Use background context for flush — shutdown must not cancel writes
		if err := w.sink.BatchCreate(context.Background(), batch); err != nil {
			logger.Errorf("audit: failed to flush %d events: %v", len(batch), err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case event := <-w.ch:
			batch = append(batch, event)
			if len(batch) >= w.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			// Drain remaining events from channel
			for {
				select {
				case event := <-w.ch:
					batch = append(batch, event)
					if len(batch) >= w.cfg.BatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}
