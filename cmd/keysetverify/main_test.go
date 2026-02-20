package main

import "testing"

const (
	testTPub = "tpubDC2pzLGKv5DoHtRoYjJsbgESSzFqc3mtPzahMMqhH89bqqHot28MFUHkUECJrBGFb2KPQZUrApq4Ti6Y69S2K3snrsT8E5Zjt1GqTMj7xn5"
	testXPub = "xpub6BfCU6SeCoGM26Ex6YKnPku57sABcfGprMzPzonYwDPi6Yd6ooHG72cvEC7XKgK1o7nUnyxydj11mXbvhHanRcRVoGhpYYuWJ3gRhPCmQKj"
)

func TestVerifyIndexZeroBitcoinMatch(t *testing.T) {
	expected, keyErr := deriveAddress("bitcoin", "testnet", "bip84_p2wpkh", testTPub)
	if keyErr != nil {
		t.Fatalf("expected test fixture derivation to succeed, got %+v", keyErr)
	}

	result, exitCode := verifyIndexZero(verifyInput{
		Chain:             "bitcoin",
		Network:           "testnet",
		AddressScheme:     "bip84_p2wpkh",
		KeysetID:          "ks_btc_testnet",
		ExtendedPublicKey: testTPub,
		ExpectedAddress:   expected,
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d result=%+v", exitCode, result)
	}
	if !result.Match {
		t.Fatalf("expected match=true, got result=%+v", result)
	}
}

func TestVerifyIndexZeroEthereumMatch(t *testing.T) {
	expected, keyErr := deriveAddress("ethereum", "sepolia", "evm_bip44", testXPub)
	if keyErr != nil {
		t.Fatalf("expected test fixture derivation to succeed, got %+v", keyErr)
	}

	result, exitCode := verifyIndexZero(verifyInput{
		Chain:             "ethereum",
		Network:           "sepolia",
		AddressScheme:     "evm_bip44",
		KeysetID:          "ks_eth_sepolia",
		ExtendedPublicKey: testXPub,
		ExpectedAddress:   expected,
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d result=%+v", exitCode, result)
	}
	if !result.Match {
		t.Fatalf("expected match=true, got result=%+v", result)
	}
}

func TestVerifyIndexZeroMismatch(t *testing.T) {
	result, exitCode := verifyIndexZero(verifyInput{
		Chain:             "ethereum",
		Network:           "sepolia",
		AddressScheme:     "evm_bip44",
		KeysetID:          "ks_eth_sepolia",
		ExtendedPublicKey: testXPub,
		ExpectedAddress:   "0x0000000000000000000000000000000000000000",
	})

	if exitCode != 3 {
		t.Fatalf("expected exit code 3 for mismatch, got %d result=%+v", exitCode, result)
	}
	if result.ErrorCode != "address_mismatch" {
		t.Fatalf("expected address_mismatch, got %s", result.ErrorCode)
	}
}

func TestVerifyIndexZeroInvalidKey(t *testing.T) {
	result, exitCode := verifyIndexZero(verifyInput{
		Chain:             "bitcoin",
		Network:           "testnet",
		AddressScheme:     "bip84_p2wpkh",
		KeysetID:          "ks_btc_testnet",
		ExtendedPublicKey: "not-a-valid-key",
		ExpectedAddress:   "tb1qinvalid",
	})

	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for invalid key, got %d result=%+v", exitCode, result)
	}
	if result.ErrorCode != "invalid_key_material_format" {
		t.Fatalf("expected invalid_key_material_format, got %s", result.ErrorCode)
	}
}
