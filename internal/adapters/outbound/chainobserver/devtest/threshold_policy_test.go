//go:build !integration

package devtest

import (
	"math/big"
	"testing"
)

func TestThresholdBPSAmountRoundsUp(t *testing.T) {
	amount := thresholdBPSAmount(big.NewInt(101), 8000)
	if amount.String() != "81" {
		t.Fatalf("expected ceil(101*0.8)=81, got %s", amount.String())
	}
}

func TestNewThresholdPolicyFallsBackToDefaults(t *testing.T) {
	policy := newThresholdPolicy(0, 0, 8000, 10000)
	if policy.detectedBPS != 8000 {
		t.Fatalf("expected detected fallback 8000, got %d", policy.detectedBPS)
	}
	if policy.confirmedBPS != 10000 {
		t.Fatalf("expected confirmed fallback 10000, got %d", policy.confirmedBPS)
	}
}

func TestNewThresholdPolicyKeepsDetectedNotHigherThanConfirmed(t *testing.T) {
	policy := newThresholdPolicy(9500, 9000, 8000, 10000)
	if policy.detectedBPS != 9000 {
		t.Fatalf("expected detected clipped to confirmed, got %d", policy.detectedBPS)
	}
}
