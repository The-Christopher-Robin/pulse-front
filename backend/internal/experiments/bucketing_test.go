package experiments

import (
	"math"
	"strconv"
	"testing"
)

func fiftyFifty() Experiment {
	return Experiment{
		Key:        "checkout_cta",
		Name:       "Checkout CTA copy",
		Status:     StatusRunning,
		Salt:       "v1",
		TrafficPct: 100,
		Variants: []Variant{
			{Key: "control", Name: "Buy now", Weight: 50},
			{Key: "treatment", Name: "Get it today", Weight: 50},
		},
	}
}

func TestAssign_IsDeterministic(t *testing.T) {
	e := fiftyFifty()
	users := []string{"u1", "u2", "u3", "u4", "u5"}
	for _, u := range users {
		first, _, err := Assign(e, u)
		if err != nil {
			t.Fatalf("assign %s: %v", u, err)
		}
		for i := 0; i < 20; i++ {
			v, _, _ := Assign(e, u)
			if v != first {
				t.Fatalf("non-sticky assignment for %s: first=%s got=%s", u, first, v)
			}
		}
	}
}

func TestAssign_RejectsEmptyUser(t *testing.T) {
	if _, _, err := Assign(fiftyFifty(), ""); err == nil {
		t.Fatal("expected error for empty user id")
	}
}

func TestAssign_InactiveExperimentReturnsHoldout(t *testing.T) {
	e := fiftyFifty()
	e.Status = StatusPaused
	v, exposed, err := Assign(e, "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != HoldoutVariant {
		t.Fatalf("expected holdout, got %s", v)
	}
	if exposed {
		t.Fatal("paused experiment should not count as exposed")
	}
}

func TestAssign_TrafficSplitRoughlyMatches(t *testing.T) {
	e := fiftyFifty()
	e.TrafficPct = 40

	const N = 20000
	in := 0
	for i := 0; i < N; i++ {
		_, exposed, _ := Assign(e, "user-"+strconv.Itoa(i))
		if exposed {
			in++
		}
	}
	got := float64(in) / float64(N)
	if math.Abs(got-0.4) > 0.03 {
		t.Fatalf("traffic slice drift: wanted ~0.40, got %.3f", got)
	}
}

func TestAssign_VariantDistributionHonorsWeights(t *testing.T) {
	e := Experiment{
		Key:        "feed_layout",
		Status:     StatusRunning,
		Salt:       "v2",
		TrafficPct: 100,
		Variants: []Variant{
			{Key: "control", Weight: 70},
			{Key: "treatment", Weight: 30},
		},
	}
	const N = 20000
	counts := map[string]int{}
	for i := 0; i < N; i++ {
		v, _, _ := Assign(e, "u-"+strconv.Itoa(i))
		counts[v]++
	}
	ctrl := float64(counts["control"]) / float64(N)
	if math.Abs(ctrl-0.7) > 0.03 {
		t.Fatalf("control share drift: wanted ~0.70, got %.3f", ctrl)
	}
}

func TestAssign_DifferentSaltsBreakCorrelation(t *testing.T) {
	a := fiftyFifty()
	b := fiftyFifty()
	a.Salt = "s-alpha"
	b.Salt = "s-beta"

	const N = 5000
	agree := 0
	for i := 0; i < N; i++ {
		u := "u-" + strconv.Itoa(i)
		va, _, _ := Assign(a, u)
		vb, _, _ := Assign(b, u)
		if va == vb {
			agree++
		}
	}
	rate := float64(agree) / float64(N)
	if math.Abs(rate-0.5) > 0.04 {
		t.Fatalf("different salts should give ~50%% agreement, got %.3f", rate)
	}
}
