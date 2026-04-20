package experiments

import (
	"sync"
	"testing"
)

type fakeSink struct {
	mu       sync.Mutex
	received []Assignment
}

func (f *fakeSink) Enqueue(a Assignment) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.received = append(f.received, a)
}

func (f *fakeSink) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.received)
}

// TestHoldoutDoesNotCountAsExposure directly exercises Assign to make sure the
// service contract holds: users outside the traffic slice return the holdout
// variant with Exposed=false, so the service layer will know not to log them.
func TestHoldoutDoesNotCountAsExposure(t *testing.T) {
	e := Experiment{
		Key:        "cart_upsell",
		Status:     StatusRunning,
		Salt:       "s-holdout",
		TrafficPct: 0,
		Variants: []Variant{
			{Key: "control", Weight: 50},
			{Key: "treatment", Weight: 50},
		},
	}
	v, exposed, err := Assign(e, "u-42")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if v != HoldoutVariant {
		t.Fatalf("expected holdout, got %s", v)
	}
	if exposed {
		t.Fatal("holdout should not be marked as exposed")
	}
}
