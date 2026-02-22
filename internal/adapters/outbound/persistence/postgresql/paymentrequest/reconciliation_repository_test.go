//go:build !integration

package paymentrequest

import "testing"

func TestSettlementStateEquals(t *testing.T) {
	heightA := int64(100)
	heightB := int64(101)
	hashA := "0xabc"
	hashB := "0xdef"

	base := settlementState{
		amountMinor:   "1000",
		confirmations: 2,
		blockHeight:   &heightA,
		blockHash:     &hashA,
		isCanonical:   true,
		metadataJSON:  `{"source":"utxo"}`,
	}

	if !base.equals(base) {
		t.Fatalf("expected identical state to be equal")
	}

	changedAmount := base
	changedAmount.amountMinor = "1001"
	if base.equals(changedAmount) {
		t.Fatalf("expected amount change to break equality")
	}

	changedHeight := base
	changedHeight.blockHeight = &heightB
	if base.equals(changedHeight) {
		t.Fatalf("expected block height change to break equality")
	}

	changedHash := base
	changedHash.blockHash = &hashB
	if base.equals(changedHash) {
		t.Fatalf("expected block hash change to break equality")
	}

	clearedHash := base
	clearedHash.blockHash = nil
	if base.equals(clearedHash) {
		t.Fatalf("expected nil block hash difference to break equality")
	}
}

func TestCanonicalizeJSON(t *testing.T) {
	canonical, err := canonicalizeJSON([]byte(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatalf("expected canonicalization success, got %v", err)
	}
	if canonical != `{"a":1,"b":2}` {
		t.Fatalf("expected sorted canonical json, got %s", canonical)
	}
}

func TestEncodeCanonicalMetadataEmpty(t *testing.T) {
	encoded, canonical, err := encodeCanonicalMetadata(nil)
	if err != nil {
		t.Fatalf("expected encode success, got %v", err)
	}
	if canonical != "{}" {
		t.Fatalf("expected canonical empty object, got %s", canonical)
	}
	if string(encoded) != "{}" {
		t.Fatalf("expected encoded empty object, got %s", string(encoded))
	}
}
