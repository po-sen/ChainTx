//go:build !integration

package devtest

import "testing"

func TestNewConfirmationPolicyFallsBackToDefaults(t *testing.T) {
	policy := newConfirmationPolicy(0, 0, 1, 1)
	if policy.btcMin != 1 {
		t.Fatalf("expected btc default 1, got %d", policy.btcMin)
	}
	if policy.evmMin != 1 {
		t.Fatalf("expected evm default 1, got %d", policy.evmMin)
	}
}

func TestNewConfirmationPolicyUsesCustomValues(t *testing.T) {
	policy := newConfirmationPolicy(3, 12, 1, 1)
	if policy.btcMin != 3 {
		t.Fatalf("expected btc 3, got %d", policy.btcMin)
	}
	if policy.evmMin != 12 {
		t.Fatalf("expected evm 12, got %d", policy.evmMin)
	}
}
