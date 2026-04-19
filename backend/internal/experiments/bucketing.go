package experiments

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

// bucketOf returns a deterministic float in [0,1) for a user within an experiment,
// using SHA-256 over (salt || experimentKey || userID). A separate domain (e.g.
// "traffic" vs "variant") keeps traffic and variant decisions independent.
func bucketOf(salt, experimentKey, userID, domain string) float64 {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte{0})
	h.Write([]byte(experimentKey))
	h.Write([]byte{0})
	h.Write([]byte(userID))
	h.Write([]byte{0})
	h.Write([]byte(domain))
	sum := h.Sum(nil)
	// First 8 bytes as uint64, scaled to [0,1).
	n := binary.BigEndian.Uint64(sum[:8])
	return float64(n) / float64(^uint64(0))
}

// Assign returns the variant an experiment hands to a given userID, and a flag
// indicating whether the user is in the experiment's traffic slice at all.
// Users outside the traffic slice land on HoldoutVariant and should not be
// counted as exposed to any treatment.
func Assign(e Experiment, userID string) (string, bool, error) {
	if userID == "" {
		return HoldoutVariant, false, errors.New("user id required")
	}
	if !e.IsActive() {
		return HoldoutVariant, false, nil
	}
	if e.TrafficPct <= 0 {
		return HoldoutVariant, false, nil
	}
	if e.TrafficPct >= 100 {
		// fall through: everyone is in traffic
	} else {
		trafficBucket := bucketOf(e.Salt, e.Key, userID, "traffic")
		if trafficBucket >= float64(e.TrafficPct)/100.0 {
			return HoldoutVariant, false, nil
		}
	}

	total := e.TotalWeight()
	if total <= 0 || len(e.Variants) == 0 {
		return HoldoutVariant, false, errors.New("experiment has no variant weight")
	}

	variantBucket := bucketOf(e.Salt, e.Key, userID, "variant")
	target := variantBucket * float64(total)
	cumulative := 0.0
	for _, v := range e.Variants {
		cumulative += float64(v.Weight)
		if target < cumulative {
			return v.Key, true, nil
		}
	}
	return e.Variants[len(e.Variants)-1].Key, true, nil
}
