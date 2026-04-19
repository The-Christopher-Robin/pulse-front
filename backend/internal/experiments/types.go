package experiments

import "time"

type Status string

const (
	StatusDraft   Status = "draft"
	StatusRunning Status = "running"
	StatusPaused  Status = "paused"
	StatusEnded   Status = "ended"
)

type Variant struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

type Experiment struct {
	Key        string    `json:"key"`
	Name       string    `json:"name"`
	Status     Status    `json:"status"`
	Salt       string    `json:"salt"`
	TrafficPct int       `json:"traffic_pct"`
	Variants   []Variant `json:"variants"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Assignment struct {
	ExperimentKey string    `json:"experiment_key"`
	VariantKey    string    `json:"variant_key"`
	UserID        string    `json:"user_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Exposed       bool      `json:"exposed"`
}

const HoldoutVariant = "holdout"

func (e Experiment) IsActive() bool {
	return e.Status == StatusRunning
}

func (e Experiment) TotalWeight() int {
	total := 0
	for _, v := range e.Variants {
		total += v.Weight
	}
	return total
}
