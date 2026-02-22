//go:build !integration

package devtest

import "testing"

func TestNewConfirmationPolicyFallsBackToDefaults(t *testing.T) {
	policy := newConfirmationPolicy(0, 0, 0, 0, 1, 1)
	if policy.btcBusinessMin != 1 {
		t.Fatalf("expected btc business default 1, got %d", policy.btcBusinessMin)
	}
	if policy.btcFinalityMin != 1 {
		t.Fatalf("expected btc finality fallback 1, got %d", policy.btcFinalityMin)
	}
	if policy.evmBusinessMin != 1 {
		t.Fatalf("expected evm business default 1, got %d", policy.evmBusinessMin)
	}
	if policy.evmFinalityMin != 1 {
		t.Fatalf("expected evm finality fallback 1, got %d", policy.evmFinalityMin)
	}
}

func TestNewConfirmationPolicyUsesCustomValues(t *testing.T) {
	policy := newConfirmationPolicy(3, 6, 12, 24, 1, 1)
	if policy.btcBusinessMin != 3 {
		t.Fatalf("expected btc business 3, got %d", policy.btcBusinessMin)
	}
	if policy.btcFinalityMin != 6 {
		t.Fatalf("expected btc finality 6, got %d", policy.btcFinalityMin)
	}
	if policy.evmBusinessMin != 12 {
		t.Fatalf("expected evm business 12, got %d", policy.evmBusinessMin)
	}
	if policy.evmFinalityMin != 24 {
		t.Fatalf("expected evm finality 24, got %d", policy.evmFinalityMin)
	}
}

func TestNewConfirmationPolicyClipsFinalityToBusiness(t *testing.T) {
	policy := newConfirmationPolicy(5, 3, 7, 2, 1, 1)
	if policy.btcFinalityMin != 5 {
		t.Fatalf("expected btc finality clipped to 5, got %d", policy.btcFinalityMin)
	}
	if policy.evmFinalityMin != 7 {
		t.Fatalf("expected evm finality clipped to 7, got %d", policy.evmFinalityMin)
	}
}
