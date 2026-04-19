package experiments

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Registry holds the set of experiments loaded from Postgres, refreshed
// periodically by a background goroutine so running services do not need a
// restart when marketing flips a traffic percentage.
type Registry struct {
	pool *pgxpool.Pool

	mu       sync.RWMutex
	byKey    map[string]Experiment
	loadedAt atomic.Int64
}

func NewRegistry(pool *pgxpool.Pool) *Registry {
	return &Registry{
		pool:  pool,
		byKey: make(map[string]Experiment),
	}
}

func (r *Registry) Load(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `
		SELECT key, name, status, salt, traffic_pct, variants, updated_at
		FROM experiments
	`)
	if err != nil {
		return fmt.Errorf("query experiments: %w", err)
	}
	defer rows.Close()

	next := make(map[string]Experiment, 32)
	for rows.Next() {
		var (
			e            Experiment
			variantBlob  []byte
			rawStatus    string
		)
		if err := rows.Scan(&e.Key, &e.Name, &rawStatus, &e.Salt, &e.TrafficPct, &variantBlob, &e.UpdatedAt); err != nil {
			return fmt.Errorf("scan experiment: %w", err)
		}
		e.Status = Status(rawStatus)
		if err := json.Unmarshal(variantBlob, &e.Variants); err != nil {
			return fmt.Errorf("unmarshal variants for %s: %w", e.Key, err)
		}
		next[e.Key] = e
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate experiments: %w", err)
	}

	r.mu.Lock()
	r.byKey = next
	r.mu.Unlock()
	r.loadedAt.Store(time.Now().Unix())
	return nil
}

func (r *Registry) Get(key string) (Experiment, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.byKey[key]
	return e, ok
}

func (r *Registry) Active() []Experiment {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Experiment, 0, len(r.byKey))
	for _, e := range r.byKey {
		if e.IsActive() {
			out = append(out, e)
		}
	}
	return out
}

func (r *Registry) All() []Experiment {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Experiment, 0, len(r.byKey))
	for _, e := range r.byKey {
		out = append(out, e)
	}
	return out
}

// Watch refreshes the registry on a ticker. Cancel the context to stop it.
func (r *Registry) Watch(ctx context.Context, every time.Duration, onError func(error)) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.Load(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}
