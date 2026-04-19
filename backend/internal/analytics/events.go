package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/The-Christopher-Robin/pulse-front/backend/internal/experiments"
)

type Event struct {
	UserID     string                 `json:"user_id"`
	EventType  string                 `json:"event_type"`
	TargetID   string                 `json:"target_id,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
}

// Writer batches exposures and events to Postgres. A single background goroutine
// drains a buffered channel on a timer or when the channel fills up, which keeps
// hot request paths free of per-event database round-trips.
type Writer struct {
	pool       *pgxpool.Pool
	exposures  chan experiments.Assignment
	events     chan Event
	flushEvery time.Duration
	batchSize  int

	wg      sync.WaitGroup
	stopCh  chan struct{}
	stopped bool
	mu      sync.Mutex
}

func NewWriter(pool *pgxpool.Pool, bufSize int, flushEvery time.Duration) *Writer {
	if bufSize <= 0 {
		bufSize = 256
	}
	if flushEvery <= 0 {
		flushEvery = 2 * time.Second
	}
	return &Writer{
		pool:       pool,
		exposures:  make(chan experiments.Assignment, bufSize),
		events:     make(chan Event, bufSize),
		flushEvery: flushEvery,
		batchSize:  bufSize,
		stopCh:     make(chan struct{}),
	}
}

func (w *Writer) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.runExposures(ctx)
	w.wg.Add(1)
	go w.runEvents(ctx)
}

func (w *Writer) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	close(w.stopCh)
	w.mu.Unlock()
	w.wg.Wait()
}

// Enqueue satisfies experiments.ExposureSink.
func (w *Writer) Enqueue(a experiments.Assignment) {
	select {
	case w.exposures <- a:
	default:
		// Buffer full, drop rather than block the request path. A production
		// build would surface a metric here.
	}
}

func (w *Writer) TrackEvent(e Event) error {
	if e.EventType == "" {
		return errors.New("event_type required")
	}
	if e.UserID == "" {
		return errors.New("user_id required")
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	select {
	case w.events <- e:
		return nil
	default:
		return errors.New("event buffer full")
	}
}

func (w *Writer) runExposures(ctx context.Context) {
	defer w.wg.Done()
	buf := make([]experiments.Assignment, 0, w.batchSize)
	t := time.NewTicker(w.flushEvery)
	defer t.Stop()

	flush := func() {
		if len(buf) == 0 {
			return
		}
		if err := w.insertExposures(ctx, buf); err != nil {
			log.Printf("exposures flush failed: %v", err)
		}
		buf = buf[:0]
	}

	for {
		select {
		case <-w.stopCh:
			flush()
			return
		case <-ctx.Done():
			flush()
			return
		case <-t.C:
			flush()
		case a := <-w.exposures:
			buf = append(buf, a)
			if len(buf) >= w.batchSize {
				flush()
			}
		}
	}
}

func (w *Writer) runEvents(ctx context.Context) {
	defer w.wg.Done()
	buf := make([]Event, 0, w.batchSize)
	t := time.NewTicker(w.flushEvery)
	defer t.Stop()

	flush := func() {
		if len(buf) == 0 {
			return
		}
		if err := w.insertEvents(ctx, buf); err != nil {
			log.Printf("events flush failed: %v", err)
		}
		buf = buf[:0]
	}

	for {
		select {
		case <-w.stopCh:
			flush()
			return
		case <-ctx.Done():
			flush()
			return
		case <-t.C:
			flush()
		case e := <-w.events:
			buf = append(buf, e)
			if len(buf) >= w.batchSize {
				flush()
			}
		}
	}
}

func (w *Writer) insertExposures(ctx context.Context, batch []experiments.Assignment) error {
	rows := make([][]any, 0, len(batch))
	for _, a := range batch {
		rows = append(rows, []any{a.ExperimentKey, a.VariantKey, a.UserID, a.OccurredAt})
	}
	_, err := w.pool.CopyFrom(ctx,
		pgx.Identifier{"exposures"},
		[]string{"experiment_key", "variant_key", "user_id", "occurred_at"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func (w *Writer) insertEvents(ctx context.Context, batch []Event) error {
	rows := make([][]any, 0, len(batch))
	for _, e := range batch {
		props, err := json.Marshal(e.Properties)
		if err != nil {
			props = []byte("{}")
		}
		rows = append(rows, []any{e.UserID, e.EventType, nullable(e.TargetID), props, e.OccurredAt})
	}
	_, err := w.pool.CopyFrom(ctx,
		pgx.Identifier{"events"},
		[]string{"user_id", "event_type", "target_id", "properties", "occurred_at"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ConversionByVariant joins exposures against a target conversion event and
// returns exposure, conversion, and conversion-rate numbers per variant for a
// given experiment and time window.
func (w *Writer) ConversionByVariant(ctx context.Context, experimentKey, convEvent string, since time.Time) ([]VariantConversion, error) {
	const q = `
WITH window AS (
    SELECT $1::text AS exp_key, $2::text AS ev, $3::timestamptz AS since
),
exp_users AS (
    SELECT e.variant_key, e.user_id, MIN(e.occurred_at) AS first_seen
    FROM exposures e, window
    WHERE e.experiment_key = window.exp_key AND e.occurred_at >= window.since
    GROUP BY e.variant_key, e.user_id
),
conv_users AS (
    SELECT DISTINCT ev.user_id
    FROM events ev, window
    WHERE ev.event_type = window.ev AND ev.occurred_at >= window.since
)
SELECT
    eu.variant_key,
    COUNT(*)::int AS exposed_users,
    COUNT(*) FILTER (WHERE cu.user_id IS NOT NULL)::int AS converted_users
FROM exp_users eu
LEFT JOIN conv_users cu ON cu.user_id = eu.user_id
GROUP BY eu.variant_key
ORDER BY eu.variant_key;
`
	rows, err := w.pool.Query(ctx, q, experimentKey, convEvent, since)
	if err != nil {
		return nil, fmt.Errorf("query conversion: %w", err)
	}
	defer rows.Close()
	out := []VariantConversion{}
	for rows.Next() {
		var v VariantConversion
		if err := rows.Scan(&v.VariantKey, &v.Exposed, &v.Converted); err != nil {
			return nil, fmt.Errorf("scan conversion row: %w", err)
		}
		if v.Exposed > 0 {
			v.Rate = float64(v.Converted) / float64(v.Exposed)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

type VariantConversion struct {
	VariantKey string  `json:"variant_key"`
	Exposed    int     `json:"exposed"`
	Converted  int     `json:"converted"`
	Rate       float64 `json:"rate"`
}
