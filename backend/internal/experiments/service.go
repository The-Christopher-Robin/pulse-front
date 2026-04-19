package experiments

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/The-Christopher-Robin/pulse-front/backend/internal/cache"
)

type Service struct {
	registry *Registry
	cache    *cache.Client
	exposure ExposureSink
	now      func() time.Time
}

type ExposureSink interface {
	Enqueue(Assignment)
}

func NewService(reg *Registry, c *cache.Client, sink ExposureSink) *Service {
	return &Service{
		registry: reg,
		cache:    c,
		exposure: sink,
		now:      time.Now,
	}
}

// AssignAll evaluates every active experiment for a user in one pass and emits
// one exposure event per treatment assignment (not for holdouts). The caller
// can rely on sticky bucketing: re-calling with the same userID yields the
// same variant keys as long as the experiment definition has not changed.
func (s *Service) AssignAll(ctx context.Context, userID string) (map[string]Assignment, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id required")
	}
	cacheKey := "assign:" + userID

	if cached, ok, _ := s.cache.GetString(ctx, cacheKey); ok {
		decoded := map[string]Assignment{}
		if err := json.Unmarshal([]byte(cached), &decoded); err == nil && s.stillFresh(decoded) {
			return decoded, nil
		}
	}

	active := s.registry.Active()
	out := make(map[string]Assignment, len(active))
	for _, e := range active {
		variant, exposed, err := Assign(e, userID)
		if err != nil {
			continue
		}
		a := Assignment{
			ExperimentKey: e.Key,
			VariantKey:    variant,
			UserID:        userID,
			OccurredAt:    s.now().UTC(),
			Exposed:       exposed,
		}
		out[e.Key] = a
		if exposed {
			s.exposure.Enqueue(a)
		}
	}

	if raw, err := json.Marshal(out); err == nil {
		_ = s.cache.SetString(ctx, cacheKey, string(raw), 5*time.Minute)
	}
	return out, nil
}

func (s *Service) stillFresh(m map[string]Assignment) bool {
	active := s.registry.Active()
	if len(active) != len(m) {
		return false
	}
	for _, e := range active {
		if _, ok := m[e.Key]; !ok {
			return false
		}
	}
	return true
}

func (s *Service) Registry() *Registry {
	return s.registry
}
